package core

func UndefinedValue() Value {
	return Value{}
}

func undefinedTypeName(v Value) string {
	return "undefined"
}

func undefinedTypeEncodeJSON(v Value) ([]byte, error) {
	return []byte("null"), nil
}

func undefinedTypeEncodeBinary(v Value) ([]byte, error) {
	return []byte{}, nil
}

func undefinedTypeDecodeBinary(v *Value, data []byte) error {
	return nil
}

func undefinedTypeString(v Value) string {
	return "undefined"
}

func undefinedTypeInterface(v Value) any {
	return nil
}

func undefinedTypeIsIterable(v Value) bool {
	return true
}

func undefinedTypeEqual(v Value, r Value) bool {
	return r.Type == VT_UNDEFINED
}

func undefinedTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	return UndefinedValue(), nil
}

func undefinedTypeAsBool(v Value) (bool, bool) {
	return false, true
}