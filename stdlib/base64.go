package stdlib

import (
	"encoding/base64"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/module"
	"github.com/jokruger/kavun/errs"
)

func init() {
	// 8..127 reserved
	InitModule("base64", module.Base64, nil, nil, map[uint64]*core.BuiltinFunction{
		0: core.NewBuiltinFunction("encode", b64EncodeToString, 1, false),
		1: core.NewBuiltinFunction("decode", b64DecodeString, 1, false),
		2: core.NewBuiltinFunction("raw_encode", b64RawEncodeToString, 1, false),
		3: core.NewBuiltinFunction("raw_decode", b64RawDecodeString, 1, false),
		4: core.NewBuiltinFunction("url_encode", b64URLEncodeToString, 1, false),
		5: core.NewBuiltinFunction("url_decode", b64URLDecodeString, 1, false),
		6: core.NewBuiltinFunction("raw_url_encode", b64RawURLEncodeToString, 1, false),
		7: core.NewBuiltinFunction("raw_url_decode", b64RawURLDecodeString, 1, false),
	})
}

func b64RawURLDecodeString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("base64.raw_url_decode", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.raw_url_decode", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := base64.RawURLEncoding.DecodeString(s1)
	if err != nil {
		return wrapError(a, err)
	}
	return a.NewBytesValue(res, false)
}

func b64URLDecodeString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("base64.url_decode", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.url_decode", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := base64.URLEncoding.DecodeString(s1)
	if err != nil {
		return wrapError(a, err)
	}
	return a.NewBytesValue(res, false)
}

func b64RawDecodeString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("base64.raw_decode", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.raw_decode", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := base64.RawStdEncoding.DecodeString(s1)
	if err != nil {
		return wrapError(a, err)
	}
	return a.NewBytesValue(res, false)
}

func b64DecodeString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("base64.decode", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.decode", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := base64.StdEncoding.DecodeString(s1)
	if err != nil {
		return wrapError(a, err)
	}
	return a.NewBytesValue(res, false)
}

func b64RawURLEncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("base64.raw_url_encode", "1", len(args))
	}
	y1, ok := args[0].AsBytes(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.raw_url_encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := base64.RawURLEncoding.EncodeToString(y1)
	return a.NewStringValue(res)
}

func b64URLEncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("base64.url_encode", "1", len(args))
	}
	y1, ok := args[0].AsBytes(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.url_encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := base64.URLEncoding.EncodeToString(y1)
	return a.NewStringValue(res)
}

func b64RawEncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("base64.raw_encode", "1", len(args))
	}
	y1, ok := args[0].AsBytes(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.raw_encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := base64.RawStdEncoding.EncodeToString(y1)
	return a.NewStringValue(res)
}

func b64EncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("base64.encode", "1", len(args))
	}
	y1, ok := args[0].AsBytes(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := base64.StdEncoding.EncodeToString(y1)
	return a.NewStringValue(res)
}
