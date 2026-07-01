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
	Suffix    string // appended to accessor names on capnpc-go name collisions
}

// scalarTypes maps every capnp type the MVP generator can wrap.
var scalarTypes = map[capnpType]typeInfo{
	"Text":    {GoType: "string", GetterErr: true, SetterErr: true, Suffix: "Text"},
	"Data":    {GoType: "[]byte", GetterErr: true, SetterErr: true, Suffix: "Data"},
	"Bool":    {GoType: "bool", Suffix: "Bool"},
	"Int8":    {GoType: "int8", Suffix: "Int8"},
	"Int16":   {GoType: "int16", Suffix: "Int16"},
	"Int32":   {GoType: "int32", Suffix: "Int32"},
	"Int64":   {GoType: "int64", Suffix: "Int64"},
	"UInt8":   {GoType: "uint8", Suffix: "Uint8"},
	"UInt16":  {GoType: "uint16", Suffix: "Uint16"},
	"UInt32":  {GoType: "uint32", Suffix: "Uint32"},
	"UInt64":  {GoType: "uint64", Suffix: "Uint64"},
	"Float32": {GoType: "float32", Suffix: "Float32"},
	"Float64": {GoType: "float64", Suffix: "Float64"},
}

func (t capnpType) info() (typeInfo, bool) {
	info, ok := scalarTypes[t]
	return info, ok
}

func (t capnpType) supported() bool {
	_, ok := t.info()
	return ok
}

// reservedAccessors are methods present on every capnpc-go struct; a field
// whose PascalCase name collides with one gets its type name appended.
var reservedAccessors = map[string]bool{
	"Message": true,
	"Segment": true,
}

// pascal converts a schema field/method name to exported Go form.
func pascal(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// accessor returns the getter name capnpc-go generates for a field, applying
// the collision rule (e.g. "message" :Text becomes "MessageText").
func accessor(f field) string {
	name := pascal(f.Name)
	if reservedAccessors[name] {
		if info, ok := f.Type.info(); ok {
			name += info.Suffix
		}
	}
	return name
}
