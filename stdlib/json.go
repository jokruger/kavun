package stdlib

import (
	"bytes"
	gojson "encoding/json"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/stdlib/json"
	"github.com/jokruger/gs/value"
)

var jsonModule = map[string]core.Object{
	"decode":      value.NewBuiltinFunction("decode", jsonDecode, 1, false),
	"encode":      value.NewBuiltinFunction("encode", jsonEncode, 1, false),
	"indent":      value.NewBuiltinFunction("indent", jsonIndent, 3, false),
	"html_escape": value.NewBuiltinFunction("html_escape", jsonHTMLEscape, 1, false),
}

func jsonDecode(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("json.decode", "1", len(args))
	}

	switch o := args[0].(type) {
	case *value.Bytes:
		v, err := json.Decode(o.Value())
		if err != nil {
			return value.NewError(value.NewString(err.Error())), nil
		}
		return v, nil
	case *value.String:
		v, err := json.Decode([]byte(o.Value()))
		if err != nil {
			return value.NewError(value.NewString(err.Error())), nil
		}
		return v, nil
	default:
		return nil, core.InvalidArgumentType("json.decode", "first", "bytes/string", args[0])
	}
}

func jsonEncode(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("json.encode", "1", len(args))
	}

	b, err := json.Encode(args[0])
	if err != nil {
		return value.NewError(value.NewString(err.Error())), nil
	}

	return value.NewBytes(b), nil
}

func jsonIndent(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 3 {
		return nil, core.WrongNumArguments("json.indent", "3", len(args))
	}

	prefix, ok := args[1].AsString()
	if !ok {
		return nil, core.InvalidArgumentType("json.indent", "prefix", "string(compatible)", args[1])
	}

	indent, ok := args[2].AsString()
	if !ok {
		return nil, core.InvalidArgumentType("json.indent", "indent", "string(compatible)", args[2])
	}

	switch o := args[0].(type) {
	case *value.Bytes:
		var dst bytes.Buffer
		err := gojson.Indent(&dst, o.Value(), prefix, indent)
		if err != nil {
			return value.NewError(value.NewString(err.Error())), nil
		}
		return value.NewBytes(dst.Bytes()), nil
	case *value.String:
		var dst bytes.Buffer
		err := gojson.Indent(&dst, []byte(o.Value()), prefix, indent)
		if err != nil {
			return value.NewError(value.NewString(err.Error())), nil
		}
		return value.NewBytes(dst.Bytes()), nil
	default:
		return nil, core.InvalidArgumentType("json.indent", "first", "bytes/string", args[0])
	}
}

func jsonHTMLEscape(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("json.html_escape", "1", len(args))
	}

	switch o := args[0].(type) {
	case *value.Bytes:
		var dst bytes.Buffer
		gojson.HTMLEscape(&dst, o.Value())
		return value.NewBytes(dst.Bytes()), nil
	case *value.String:
		var dst bytes.Buffer
		gojson.HTMLEscape(&dst, []byte(o.Value()))
		return value.NewBytes(dst.Bytes()), nil
	default:
		return nil, core.InvalidArgumentType("json.html_escape", "first", "bytes/string", args[0])
	}
}
