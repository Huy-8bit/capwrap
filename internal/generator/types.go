package generator

import "strings"

// capnpType is a raw type token from a .capnp schema, e.g. "Text" or "Int64".
type capnpType string

// typeInfo describes how a supported scalar/text type maps onto Go and onto the
// accessors capnpc-go emits.
type typeInfo struct {
	GoType    string // Go type in the generated request/response struct
	GetterErr bool   // capnp getter returns (T, error)
	SetterErr bool   // capnp setter returns error
}

// scalarTypes maps every capnp type the MVP generator can wrap.
var scalarTypes = map[capnpType]typeInfo{
	"Text":    {GoType: "string", GetterErr: true, SetterErr: true},
	"Data":    {GoType: "[]byte", GetterErr: true, SetterErr: true},
	"Bool":    {GoType: "bool"},
	"Int8":    {GoType: "int8"},
	"Int16":   {GoType: "int16"},
	"Int32":   {GoType: "int32"},
	"Int64":   {GoType: "int64"},
	"UInt8":   {GoType: "uint8"},
	"UInt16":  {GoType: "uint16"},
	"UInt32":  {GoType: "uint32"},
	"UInt64":  {GoType: "uint64"},
	"Float32": {GoType: "float32"},
	"Float64": {GoType: "float64"},
}

func (t capnpType) info() (typeInfo, bool) {
	info, ok := scalarTypes[t]
	return info, ok
}

func (t capnpType) supported() bool {
	_, ok := t.info()
	return ok
}

// reservedAccessors are methods present on every capnpc-go struct. A field
// whose PascalCase name collides with one gets a trailing underscore, matching
// capnpc-go (e.g. field "message" becomes accessor "Message_").
var reservedAccessors = map[string]bool{
	"Message":       true,
	"Segment":       true,
	"String":        true,
	"ToPtr":         true,
	"IsValid":       true,
	"EncodeAsPtr":   true,
	"DecodeFromPtr": true,
}

// pascal converts a schema field/method name to exported Go form.
func pascal(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// accessor returns the getter name capnpc-go generates for a field, applying
// the collision rule (e.g. "message" becomes "Message_").
func accessor(f field) string {
	name := pascal(f.Name)
	if reservedAccessors[name] {
		name += "_"
	}
	return name
}
