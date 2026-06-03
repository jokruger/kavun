package stdlib

import (
	"encoding/hex"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func init() {
	// 2..127 reserved
	InitModule("hex", core.BI_MOD_HEX, nil, nil, map[uint64]*core.BuiltinFunction{
		0: core.NewBuiltinFunction("encode", hexEncodeToString, 1, false),
		1: core.NewBuiltinFunction("decode", hexDecodeString, 1, false),
	})
}

func hexDecodeString(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("hex.decode", "1", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("hex.decode", "first", "string(compatible)", args[0].TypeName(a))
	}
	res, err := hex.DecodeString(s1)
	if err != nil {
		return wrapError(err)
	}
	return a.NewBytesValue(res, false), nil
}

func hexEncodeToString(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("hex.encode", "1", len(args))
	}
	y1, ok := args[0].AsBytes(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("hex.encode", "first", "bytes(compatible)", args[0].TypeName(a))
	}
	res := hex.EncodeToString(y1)
	return a.NewStringValue(res), nil
}
