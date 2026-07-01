package generator

import (
	"fmt"
	"regexp"
	"strings"
)

// schema is the subset of a .capnp file that capwrap-gen understands.
type schema struct {
	Package    string
	Interfaces []iface
}

type iface struct {
	Name    string
	Methods []method
}

type method struct {
	Name    string // original schema name, e.g. "sayHello"
	Params  []field
	Results []field
	// Unsupported is set (with a reason) when a param/result uses a type the
	// MVP generator cannot map. Such methods still get a stub server adapter so
	// the generated code satisfies the capnp interface, but no typed Go API.
	Unsupported string
}

type field struct {
	Name string // original schema name, e.g. "message"
	Type capnpType
}

var (
	pkgRe    = regexp.MustCompile(`\$Go\.package\("([^"]+)"\)`)
	ifaceRe  = regexp.MustCompile(`(?s)interface\s+(\w+)\s*\{`)
	methodRe = regexp.MustCompile(`^\s*(\w+)\s*@\d+\s*$`)
)

// parse reads a .capnp source into the subset schema capwrap-gen supports.
func parse(src string) (*schema, error) {
	src = stripComments(src)

	s := &schema{}
	if m := pkgRe.FindStringSubmatch(src); m != nil {
		s.Package = m[1]
	}

	for _, loc := range ifaceRe.FindAllStringSubmatchIndex(src, -1) {
		name := src[loc[2]:loc[3]]
		body, err := balancedBody(src, loc[1]-1, '{', '}')
		if err != nil {
			return nil, fmt.Errorf("interface %s: %w", name, err)
		}
		methods, err := parseMethods(body)
		if err != nil {
			return nil, fmt.Errorf("interface %s: %w", name, err)
		}
		s.Interfaces = append(s.Interfaces, iface{Name: name, Methods: methods})
	}

	if len(s.Interfaces) == 0 {
		return nil, fmt.Errorf("no interface found in schema")
	}
	return s, nil
}

// parseMethods walks an interface body, splitting on top-level semicolons and
// reading each "name @n (params) -> (results)" declaration.
func parseMethods(body string) ([]method, error) {
	var methods []method
	for _, stmt := range splitTopLevel(body, ';') {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		open := strings.IndexByte(stmt, '(')
		if open < 0 {
			return nil, fmt.Errorf("malformed method %q", stmt)
		}
		head := methodRe.FindStringSubmatch(stmt[:open])
		if head == nil {
			return nil, fmt.Errorf("malformed method header %q", stmt[:open])
		}
		m := method{Name: head[1]}

		params, rest, err := balancedGroup(stmt[open:])
		if err != nil {
			return nil, fmt.Errorf("method %s params: %w", m.Name, err)
		}
		arrow := strings.Index(rest, "->")
		if arrow < 0 {
			return nil, fmt.Errorf("method %s: missing '->'", m.Name)
		}
		results, _, err := balancedGroup(rest[arrow+2:])
		if err != nil {
			return nil, fmt.Errorf("method %s results: %w", m.Name, err)
		}

		if m.Params, err = parseFields(params); err != nil {
			return nil, fmt.Errorf("method %s params: %w", m.Name, err)
		}
		if m.Results, err = parseFields(results); err != nil {
			return nil, fmt.Errorf("method %s results: %w", m.Name, err)
		}
		m.Unsupported = unsupportedReason(m)
		methods = append(methods, m)
	}
	return methods, nil
}

// parseFields parses "name :Type, name :Type" inside a params/results group.
func parseFields(group string) ([]field, error) {
	var fields []field
	for _, part := range splitTopLevel(group, ',') {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		colon := strings.IndexByte(part, ':')
		if colon < 0 {
			return nil, fmt.Errorf("malformed field %q", part)
		}
		name := strings.TrimSpace(part[:colon])
		typ := strings.TrimSpace(part[colon+1:])
		fields = append(fields, field{Name: name, Type: capnpType(typ)})
	}
	return fields, nil
}

func unsupportedReason(m method) string {
	for _, f := range append(append([]field{}, m.Params...), m.Results...) {
		if !f.Type.supported() {
			return fmt.Sprintf("field %q has unsupported type %q", f.Name, f.Type)
		}
	}
	return ""
}

// balancedBody returns the text between the matching open/close runes, given
// the index of the opening rune.
func balancedBody(src string, openIdx int, open, close byte) (string, error) {
	depth := 0
	for i := openIdx; i < len(src); i++ {
		switch src[i] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return src[openIdx+1 : i], nil
			}
		}
	}
	return "", fmt.Errorf("unbalanced %q", string(open))
}

// balancedGroup reads a leading "( ... )" from s, returning the inner text and
// the remainder after the closing paren.
func balancedGroup(s string) (inner, rest string, err error) {
	s = strings.TrimLeft(s, " \t\r\n")
	if len(s) == 0 || s[0] != '(' {
		return "", "", fmt.Errorf("expected '('")
	}
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[1:i], s[i+1:], nil
			}
		}
	}
	return "", "", fmt.Errorf("unbalanced '('")
}

// splitTopLevel splits s on sep, ignoring separators nested inside parens.
func splitTopLevel(s string, sep byte) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// stripComments removes '#'-to-end-of-line Cap'n Proto comments.
func stripComments(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
