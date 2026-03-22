package stdlib

import (
	"bytes"
	gojson "encoding/json"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/stdlib/json"
	"github.com/jokruger/gs/value"
)

var jsonModule = map[string]core.Object{
	"decode":      value.NewStaticBuiltinFunction("decode", jsonDecode, 1, false),
	"encode":      value.NewStaticBuiltinFunction("encode", jsonEncode, 1, false),
	"indent":      value.NewStaticBuiltinFunction("indent", jsonIndent, 3, false),
	"html_escape": value.NewStaticBuiltinFunction("html_escape", jsonHTMLEscape, 1, false),
}

func jsonDecode(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("json.decode", "1", len(args))
	}

	alloc := vm.Allocator()
	switch o := args[0].(type) {
	case *value.Bytes:
		v, err := json.Decode(alloc, o.Value())
		if err != nil {
			return vm.Allocator().NewError(alloc.NewString(err.Error())), nil
		}
		return v, nil
	case *value.String:
		v, err := json.Decode(alloc, []byte(o.Value()))
		if err != nil {
			return vm.Allocator().NewError(alloc.NewString(err.Error())), nil
		}
		return v, nil
	default:
		return nil, core.NewInvalidArgumentTypeError("json.decode", "first", "bytes/string", args[0])
	}
}

func jsonEncode(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("json.encode", "1", len(args))
	}

	alloc := vm.Allocator()

	b, err := json.Encode(args[0])
	if err != nil {
		return alloc.NewError(alloc.NewString(err.Error())), nil
	}

	return alloc.NewBytes(b), nil
}

func jsonIndent(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 3 {
		return nil, core.NewWrongNumArgumentsError("json.indent", "3", len(args))
	}

	prefix, ok := args[1].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("json.indent", "prefix", "string(compatible)", args[1])
	}

	indent, ok := args[2].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("json.indent", "indent", "string(compatible)", args[2])
	}

	alloc := vm.Allocator()
	switch o := args[0].(type) {
	case *value.Bytes:
		var dst bytes.Buffer
		err := gojson.Indent(&dst, o.Value(), prefix, indent)
		if err != nil {
			return vm.Allocator().NewError(alloc.NewString(err.Error())), nil
		}
		return vm.Allocator().NewBytes(dst.Bytes()), nil
	case *value.String:
		var dst bytes.Buffer
		err := gojson.Indent(&dst, []byte(o.Value()), prefix, indent)
		if err != nil {
			return vm.Allocator().NewError(alloc.NewString(err.Error())), nil
		}
		return vm.Allocator().NewBytes(dst.Bytes()), nil
	default:
		return nil, core.NewInvalidArgumentTypeError("json.indent", "first", "bytes/string", args[0])
	}
}

func jsonHTMLEscape(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("json.html_escape", "1", len(args))
	}

	switch o := args[0].(type) {
	case *value.Bytes:
		var dst bytes.Buffer
		gojson.HTMLEscape(&dst, o.Value())
		return vm.Allocator().NewBytes(dst.Bytes()), nil
	case *value.String:
		var dst bytes.Buffer
		gojson.HTMLEscape(&dst, []byte(o.Value()))
		return vm.Allocator().NewBytes(dst.Bytes()), nil
	default:
		return nil, core.NewInvalidArgumentTypeError("json.html_escape", "first", "bytes/string", args[0])
	}
}
