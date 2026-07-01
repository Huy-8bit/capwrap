// Package protoconv translates a small subset of protobuf (.proto) schemas into
// Cap'n Proto (.capnp) source, so capwrap users can author services in familiar
// proto syntax. Each unary `rpc M(Req) returns (Resp)` is flattened: the fields
// of Req/Resp become the Cap'n Proto method's params/results.
//
// Supported: proto3, a service with unary rpc methods, and messages made of
// scalar/string/bytes fields. Streaming, repeated, map, oneof, enums and nested
// message fields are not supported and cause the affected method to be skipped.
package protoconv

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
)

// ToCapnp parses proto source and returns equivalent Cap'n Proto source. Skipped
// lists methods that could not be translated (with the reason).
func ToCapnp(src string) (capnp string, skipped []string, err error) {
	p, err := parseProto(src)
	if err != nil {
		return "", nil, err
	}
	if p.GoPackage == "" || p.GoImport == "" {
		return "", nil, fmt.Errorf(`missing go_package option; add: option go_package = "your/import/path;pkgname";`)
	}
	if len(p.Services) == 0 {
		return "", nil, fmt.Errorf("no service found in .proto")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "@0x%016x;\n\n", fileID(src))
	b.WriteString("using Go = import \"/go.capnp\";\n\n")
	fmt.Fprintf(&b, "$Go.package(%q);\n", p.GoPackage)
	fmt.Fprintf(&b, "$Go.import(%q);\n\n", p.GoImport)
	b.WriteString("# Code generated from a .proto by capwrap-gen. Edit the .proto instead.\n")

	for _, svc := range p.Services {
		fmt.Fprintf(&b, "\ninterface %s {\n", svc.Name)
		ord := 0
		for _, m := range svc.Methods {
			params, perr := p.flatten(m.ReqType)
			results, rerr := p.flatten(m.RespType)
			switch {
			case m.Streaming:
				skipped = append(skipped, fmt.Sprintf("%s.%s: streaming not supported", svc.Name, m.Name))
				continue
			case perr != nil:
				skipped = append(skipped, fmt.Sprintf("%s.%s: %v", svc.Name, m.Name, perr))
				continue
			case rerr != nil:
				skipped = append(skipped, fmt.Sprintf("%s.%s: %v", svc.Name, m.Name, rerr))
				continue
			}
			fmt.Fprintf(&b, "  %s @%d (%s) -> (%s);\n", lowerFirst(m.Name), ord, params, results)
			ord++
		}
		b.WriteString("}\n")
	}
	return b.String(), skipped, nil
}

// flatten renders a message's fields as a Cap'n Proto param/result list.
func (p *protoFile) flatten(msgName string) (string, error) {
	msg, ok := p.Messages[trimPkg(msgName)]
	if !ok {
		return "", fmt.Errorf("message %q not found", msgName)
	}
	var parts []string
	for _, f := range msg.Fields {
		if f.Repeated {
			return "", fmt.Errorf("field %q is repeated (unsupported)", f.Name)
		}
		ct, ok := scalarToCapnp[f.Type]
		if !ok {
			return "", fmt.Errorf("field %q has unsupported type %q", f.Name, f.Type)
		}
		parts = append(parts, fmt.Sprintf("%s :%s", camel(f.Name), ct))
	}
	return strings.Join(parts, ", "), nil
}

// --- proto model ---------------------------------------------------------

type protoFile struct {
	GoPackage string
	GoImport  string
	Messages  map[string]*protoMessage
	Services  []protoService
}

type protoMessage struct {
	Name   string
	Fields []protoField
}

type protoField struct {
	Type     string
	Name     string
	Repeated bool
}

type protoService struct {
	Name    string
	Methods []protoMethod
}

type protoMethod struct {
	Name      string
	ReqType   string
	RespType  string
	Streaming bool
}

var scalarToCapnp = map[string]string{
	"double":   "Float64",
	"float":    "Float32",
	"int32":    "Int32",
	"sint32":   "Int32",
	"sfixed32": "Int32",
	"int64":    "Int64",
	"sint64":   "Int64",
	"sfixed64": "Int64",
	"uint32":   "UInt32",
	"fixed32":  "UInt32",
	"uint64":   "UInt64",
	"fixed64":  "UInt64",
	"bool":     "Bool",
	"string":   "Text",
	"bytes":    "Data",
}

var (
	goPkgRe = regexp.MustCompile(`option\s+go_package\s*=\s*"([^"]+)"`)
	blockRe = regexp.MustCompile(`(service|message)\s+(\w+)\s*\{`)
	rpcRe   = regexp.MustCompile(`rpc\s+(\w+)\s*\(\s*(stream\s+)?([.\w]+)\s*\)\s+returns\s*\(\s*(stream\s+)?([.\w]+)\s*\)`)
	fieldRe = regexp.MustCompile(`^\s*(repeated\s+)?([.\w]+)\s+(\w+)\s*=\s*\d+`)
)

func parseProto(src string) (*protoFile, error) {
	src = stripComments(src)
	p := &protoFile{Messages: map[string]*protoMessage{}}

	if m := goPkgRe.FindStringSubmatch(src); m != nil {
		spec := m[1] // "import/path;pkgname" or "import/path"
		if i := strings.LastIndex(spec, ";"); i >= 0 {
			p.GoImport, p.GoPackage = spec[:i], spec[i+1:]
		} else {
			p.GoImport = spec
			p.GoPackage = spec[strings.LastIndex(spec, "/")+1:]
		}
	}

	for _, loc := range blockRe.FindAllStringSubmatchIndex(src, -1) {
		kind := src[loc[2]:loc[3]]
		name := src[loc[4]:loc[5]]
		body, err := balancedBody(src, loc[1]-1)
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", kind, name, err)
		}
		switch kind {
		case "message":
			p.Messages[name] = &protoMessage{Name: name, Fields: parseFields(body)}
		case "service":
			p.Services = append(p.Services, protoService{Name: name, Methods: parseMethods(body)})
		}
	}
	return p, nil
}

func parseFields(body string) []protoField {
	var fields []protoField
	for _, line := range strings.Split(body, ";") {
		if strings.Contains(line, "oneof") || strings.Contains(line, "map<") {
			continue // unsupported constructs are ignored (may leave method unsupported)
		}
		m := fieldRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		fields = append(fields, protoField{
			Repeated: m[1] != "",
			Type:     m[2],
			Name:     m[3],
		})
	}
	return fields
}

func parseMethods(body string) []protoMethod {
	var methods []protoMethod
	for _, m := range rpcRe.FindAllStringSubmatch(body, -1) {
		methods = append(methods, protoMethod{
			Name:      m[1],
			ReqType:   m[3],
			RespType:  m[5],
			Streaming: m[2] != "" || m[4] != "",
		})
	}
	return methods
}

// --- helpers -------------------------------------------------------------

func balancedBody(src string, openIdx int) (string, error) {
	depth := 0
	for i := openIdx; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[openIdx+1 : i], nil
			}
		}
	}
	return "", fmt.Errorf("unbalanced '{'")
}

func stripComments(src string) string {
	// block comments
	for {
		i := strings.Index(src, "/*")
		if i < 0 {
			break
		}
		j := strings.Index(src[i:], "*/")
		if j < 0 {
			src = src[:i]
			break
		}
		src = src[:i] + " " + src[i+j+2:]
	}
	// line comments
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func trimPkg(name string) string {
	return name[strings.LastIndex(name, ".")+1:]
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// camel converts a snake_case proto field name to lowerCamelCase for Cap'n Proto.
func camel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// fileID derives a stable Cap'n Proto file ID from the source, with the high bit
// set as Cap'n Proto requires.
func fileID(src string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(src))
	return h.Sum64() | (1 << 63)
}
