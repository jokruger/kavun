package stdlib

import (
	"bytes"
	gojson "encoding/json"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/json"
)

func init() {
	// 4..127 reserved
	InitModule("json", core.BI_MOD_JSON, nil, nil, map[uint64]*core.BuiltinFunction{
		0: core.NewBuiltinFunction("decode", jsonDecode, 1, false),
		1: core.NewBuiltinFunction("encode", jsonEncode, 1, false),
		2: core.NewBuiltinFunction("indent", jsonIndent, 3, false),
		3: core.NewBuiltinFunction("html_escape", jsonHTMLEscape, 1, false),
	})
}

func jsonDecode(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("json.decode", "1", len(args))
	}

	b, ok := args[0].AsBytes(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.decode", "first", "bytes/string", args[0].TypeName(a))
	}

	v, err := json.Decode(a, b)
	if err != nil {
		return a.NewErrorValue(a.NewStringValue(err.Error()), core.KindUser, false), nil
	}

	return v, nil
}

func jsonEncode(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("json.encode", "1", len(args))
	}

	b, err := json.Encode(a, args[0])
	if err != nil {
		return a.NewErrorValue(a.NewStringValue(err.Error()), core.KindUser, false), nil
	}

	return a.NewBytesValue(b, false), nil
}

func jsonIndent(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("json.indent", "3", len(args))
	}

	prefix, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.indent", "prefix", "string(compatible)", args[1].TypeName(a))
	}

	indent, ok := args[2].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.indent", "indent", "string(compatible)", args[2].TypeName(a))
	}

	b, ok := args[0].AsBytes(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.indent", "first", "bytes/string", args[0].TypeName(a))
	}

	var dst bytes.Buffer
	err := gojson.Indent(&dst, b, prefix, indent)
	if err != nil {
		return a.NewErrorValue(a.NewStringValue(err.Error()), core.KindUser, false), nil
	}

	return a.NewBytesValue(dst.Bytes(), false), nil
}

func jsonHTMLEscape(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("json.html_escape", "1", len(args))
	}

	b, ok := args[0].AsBytes(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.html_escape", "first", "bytes/string", args[0].TypeName(a))
	}

	var dst bytes.Buffer
	gojson.HTMLEscape(&dst, b)
	return a.NewBytesValue(dst.Bytes(), false), nil
}
