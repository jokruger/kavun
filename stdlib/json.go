package stdlib

import (
	"bytes"
	gojson "encoding/json"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/module"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/json"
)

func init() {
	// 4..127 reserved
	InitModule("json", module.Json, nil, nil, map[uint64]*core.BuiltinFunction{
		0: core.NewBuiltinFunction("decode", jsonDecode, 1, false),
		1: core.NewBuiltinFunction("encode", jsonEncode, 1, false),
		2: core.NewBuiltinFunction("indent", jsonIndent, 3, false),
		3: core.NewBuiltinFunction("html_escape", jsonHTMLEscape, 1, false),
	})
}

func jsonDecode(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("json.decode", "1", len(args))
	}

	b, ok := args[0].AsBytes(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.decode", "first", "bytes/string", args[0].TypeName())
	}

	v, err := json.Decode(a, b)
	if err != nil {
		nv, err := a.NewStringValue(err.Error())
		if err != nil {
			return core.Undefined, err
		}
		return a.NewErrorValue(nv, core.KindUser, false)
	}

	return v, nil
}

func jsonEncode(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("json.encode", "1", len(args))
	}

	b, err := json.Encode(a, args[0])
	if err != nil {
		nv, err := a.NewStringValue(err.Error())
		if err != nil {
			return core.Undefined, err
		}
		return a.NewErrorValue(nv, core.KindUser, false)
	}

	return a.NewBytesValue(b, false)
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

	b, ok := args[0].AsBytes(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.indent", "first", "bytes/string", args[0].TypeName())
	}

	var dst bytes.Buffer
	err := gojson.Indent(&dst, b, prefix, indent)
	if err != nil {
		nv, err := a.NewStringValue(err.Error())
		if err != nil {
			return core.Undefined, err
		}
		return a.NewErrorValue(nv, core.KindUser, false)
	}

	return a.NewBytesValue(dst.Bytes(), false)
}

func jsonHTMLEscape(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("json.html_escape", "1", len(args))
	}

	b, ok := args[0].AsBytes(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.html_escape", "first", "bytes/string", args[0].TypeName())
	}

	var dst bytes.Buffer
	gojson.HTMLEscape(&dst, b)
	return a.NewBytesValue(dst.Bytes(), false)
}
