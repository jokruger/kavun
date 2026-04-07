package stdlib

import (
	"bytes"
	gojson "encoding/json"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/stdlib/json"
	"github.com/jokruger/gs/value"
)

var jsonModule = map[string]core.Value{
	"decode":      core.NewStaticBuiltinFunction("decode", jsonDecode, 1, false),
	"encode":      core.NewStaticBuiltinFunction("encode", jsonEncode, 1, false),
	"indent":      core.NewStaticBuiltinFunction("indent", jsonIndent, 3, false),
	"html_escape": core.NewStaticBuiltinFunction("html_escape", jsonHTMLEscape, 1, false),
}

func jsonDecode(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("json.decode", "1", len(args))
	}

	if !args[0].IsObject() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("json.decode", "first", "bytes/string", args[0].TypeName())
	}

	alloc := vm.Allocator()
	switch o := args[0].Object().(type) {
	case *value.Bytes:
		v, err := json.Decode(alloc, o.Value())
		if err != nil {
			return vm.Allocator().NewErrorValue(alloc.NewStringValue(err.Error())), nil
		}
		return v, nil

	case *value.String:
		v, err := json.Decode(alloc, []byte(o.Value()))
		if err != nil {
			return vm.Allocator().NewErrorValue(alloc.NewStringValue(err.Error())), nil
		}
		return v, nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("json.decode", "first", "bytes/string", args[0].TypeName())
	}
}

func jsonEncode(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("json.encode", "1", len(args))
	}

	alloc := vm.Allocator()

	b, err := json.Encode(args[0])
	if err != nil {
		return vm.Allocator().NewErrorValue(alloc.NewStringValue(err.Error())), nil
	}

	return alloc.NewBytesValue(b), nil
}

func jsonIndent(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("json.indent", "3", len(args))
	}

	prefix, ok := args[1].AsString()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("json.indent", "prefix", "string(compatible)", args[1].TypeName())
	}

	indent, ok := args[2].AsString()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("json.indent", "indent", "string(compatible)", args[2].TypeName())
	}

	if !args[0].IsObject() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("json.indent", "first", "bytes/string", args[0].TypeName())
	}

	alloc := vm.Allocator()
	switch o := args[0].Object().(type) {
	case *value.Bytes:
		var dst bytes.Buffer
		err := gojson.Indent(&dst, o.Value(), prefix, indent)
		if err != nil {
			return vm.Allocator().NewErrorValue(alloc.NewStringValue(err.Error())), nil
		}
		return alloc.NewBytesValue(dst.Bytes()), nil

	case *value.String:
		var dst bytes.Buffer
		err := gojson.Indent(&dst, []byte(o.Value()), prefix, indent)
		if err != nil {
			return vm.Allocator().NewErrorValue(alloc.NewStringValue(err.Error())), nil
		}
		return alloc.NewBytesValue(dst.Bytes()), nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("json.indent", "first", "bytes/string", args[0].TypeName())
	}
}

func jsonHTMLEscape(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("json.html_escape", "1", len(args))
	}

	if !args[0].IsObject() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("json.html_escape", "first", "bytes/string", args[0].TypeName())
	}

	switch o := args[0].Object().(type) {
	case *value.Bytes:
		var dst bytes.Buffer
		gojson.HTMLEscape(&dst, o.Value())
		return vm.Allocator().NewBytesValue(dst.Bytes()), nil

	case *value.String:
		var dst bytes.Buffer
		gojson.HTMLEscape(&dst, []byte(o.Value()))
		return vm.Allocator().NewBytesValue(dst.Bytes()), nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("json.html_escape", "first", "bytes/string", args[0].TypeName())
	}
}
