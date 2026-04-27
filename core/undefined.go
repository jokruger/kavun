package core

// UndefinedValue creates new boxed undefined value.
func UndefinedValue() Value {
	return Value{}
}

/* Undefined type methods */

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

func undefinedTypeAccess(v Value, a *Arena, index Value, mode Opcode) (Value, error) {
	return UndefinedValue(), nil
}

func undefinedTypeAsBool(v Value) (bool, bool) {
	return false, true
}
