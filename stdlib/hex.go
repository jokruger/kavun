package stdlib

import (
	"encoding/hex"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

func init() {
	m := core.NewModule(value.ModuleHex, "hex", true)
	// 2..127 reserved
	m.PureFn(members.Encode, 0, hexEncodeToString, 1, core.Fixed)
	m.PureFn(members.Decode, 1, hexDecodeString, 1, core.Fixed)
	if err := m.Register(); err != nil {
		panic(err)
	}
}

func hexDecodeString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("hex.decode", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("hex.decode", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := hex.DecodeString(s1)
	if err != nil {
		return raiseGo(errs.KindConversion, "hex.decode", err)
	}
	return core.NewBytesValue(res, false), nil
}

func hexEncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("hex.encode", "1", len(args))
	}
	y1, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("hex.encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := hex.EncodeToString(y1)
	return core.NewStringValue(res), nil
}
