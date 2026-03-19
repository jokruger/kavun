package stdlib

import (
	"encoding/hex"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

var hexModule = map[string]core.Object{
	"encode": value.NewBuiltinFunction("encode", hexEncodeToString, 1, false),
	"decode": value.NewBuiltinFunction("decode", hexDecodeString, 1, false),
}

func hexDecodeString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("hex.decode", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("hex.decode", "first", "string(compatible)", args[0])
	}
	res, err := hex.DecodeString(s1)
	if err != nil {
		return wrapError(err), nil
	}
	if len(res) > core.MaxBytesLen {
		return nil, core.NewBytesLimitError("hex.decode")
	}
	return value.NewBytes(res), nil
}

func hexEncodeToString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("hex.encode", "1", len(args))
	}
	y1, ok := args[0].AsByteSlice()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("hex.encode", "first", "bytes(compatible)", args[0])
	}
	res := hex.EncodeToString(y1)
	return value.NewString(res), nil
}
