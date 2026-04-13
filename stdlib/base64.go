package stdlib

import (
	"encoding/base64"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
)

var base64Module = map[string]core.Value{
	"encode":         core.NewBuiltinFunctionValue("encode", b64EncodeToString, 1, false),
	"decode":         core.NewBuiltinFunctionValue("decode", b64DecodeString, 1, false),
	"raw_encode":     core.NewBuiltinFunctionValue("raw_encode", b64RawEncodeToString, 1, false),
	"raw_decode":     core.NewBuiltinFunctionValue("raw_decode", b64RawDecodeString, 1, false),
	"url_encode":     core.NewBuiltinFunctionValue("url_encode", b64URLEncodeToString, 1, false),
	"url_decode":     core.NewBuiltinFunctionValue("url_decode", b64URLDecodeString, 1, false),
	"raw_url_encode": core.NewBuiltinFunctionValue("raw_url_encode", b64RawURLEncodeToString, 1, false),
	"raw_url_decode": core.NewBuiltinFunctionValue("raw_url_decode", b64RawURLDecodeString, 1, false),
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
		return wrapError(vm, err), nil
	}
	if len(res) > core.MaxBytesLen {
		return core.Undefined, errs.NewBytesLimitError("base64.raw_url_decode")
	}
	return vm.Allocator().NewBytesValue(res), nil
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
		return wrapError(vm, err), nil
	}
	if len(res) > core.MaxBytesLen {
		return core.Undefined, errs.NewBytesLimitError("base64.url_decode")
	}
	return vm.Allocator().NewBytesValue(res), nil
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
		return wrapError(vm, err), nil
	}
	if len(res) > core.MaxBytesLen {
		return core.Undefined, errs.NewBytesLimitError("base64.raw_decode")
	}
	return vm.Allocator().NewBytesValue(res), nil
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
		return wrapError(vm, err), nil
	}
	if len(res) > core.MaxBytesLen {
		return core.Undefined, errs.NewBytesLimitError("base64.decode")
	}
	return vm.Allocator().NewBytesValue(res), nil
}

func b64RawURLEncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("base64.raw_url_encode", "1", len(args))
	}
	y1, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.raw_url_encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := base64.RawURLEncoding.EncodeToString(y1)
	return vm.Allocator().NewStringValue(res), nil
}

func b64URLEncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("base64.url_encode", "1", len(args))
	}
	y1, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.url_encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := base64.URLEncoding.EncodeToString(y1)
	return vm.Allocator().NewStringValue(res), nil
}

func b64RawEncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("base64.raw_encode", "1", len(args))
	}
	y1, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.raw_encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := base64.RawStdEncoding.EncodeToString(y1)
	return vm.Allocator().NewStringValue(res), nil
}

func b64EncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("base64.encode", "1", len(args))
	}
	y1, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := base64.StdEncoding.EncodeToString(y1)
	return vm.Allocator().NewStringValue(res), nil
}
