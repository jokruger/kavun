package stdlib

import (
	"bytes"
	gojson "encoding/json"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/json"
)

var jsonModule = map[string]core.Value{
	"decode":      core.NewBuiltinFunctionValue("decode", jsonDecode, 1, false),
	"encode":      core.NewBuiltinFunctionValue("encode", jsonEncode, 1, false),
	"indent":      core.NewBuiltinFunctionValue("indent", jsonIndent, 3, false),
	"html_escape": core.NewBuiltinFunctionValue("html_escape", jsonHTMLEscape, 1, false),
}

func jsonDecode(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("json.decode", "1", len(args))
	}

	b, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.decode", "first", "bytes/string", args[0].TypeName())
	}

	alloc := vm.Allocator()
	v, err := json.Decode(alloc, b)
	if err != nil {
		t := alloc.NewStringValue(err.Error())
		return vm.Allocator().NewErrorValue(t), nil
	}

	return v, nil
}

func jsonEncode(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("json.encode", "1", len(args))
	}

	alloc := vm.Allocator()
	b, err := json.Encode(args[0])
	if err != nil {
		t := alloc.NewStringValue(err.Error())
		return vm.Allocator().NewErrorValue(t), nil
	}

	return alloc.NewBytesValue(b), nil
}

func jsonIndent(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("json.indent", "3", len(args))
	}

	prefix, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.indent", "prefix", "string(compatible)", args[1].TypeName())
	}

	indent, ok := args[2].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.indent", "indent", "string(compatible)", args[2].TypeName())
	}

	b, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.indent", "first", "bytes/string", args[0].TypeName())
	}

	alloc := vm.Allocator()
	var dst bytes.Buffer
	err := gojson.Indent(&dst, b, prefix, indent)
	if err != nil {
		t := alloc.NewStringValue(err.Error())
		return vm.Allocator().NewErrorValue(t), nil
	}

	return alloc.NewBytesValue(dst.Bytes()), nil
}

func jsonHTMLEscape(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("json.html_escape", "1", len(args))
	}

	b, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.html_escape", "first", "bytes/string", args[0].TypeName())
	}

	var dst bytes.Buffer
	gojson.HTMLEscape(&dst, b)
	return vm.Allocator().NewBytesValue(dst.Bytes()), nil
}
