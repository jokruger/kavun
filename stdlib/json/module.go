package json

import (
	"bytes"
	gojson "encoding/json"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

// JSON encoding and decoding of Kavun values.
//
// Example:
//
// ```go
// json = import("json")
// json.encode({"a": 1, "b": true})
// ```
//
// **Failures.** `decode` and `indent` raise kind `json_decoding` on malformed input. `decode` reads a number without a
// fraction or exponent as `int` and any other as `float`, and raises `json_decoding` when it does not fit — an integer
// outside int64 (`(json.decode) number 9223372036854775808 is out of int range`) or a float that would be ±Inf
// (`1e999`); it never wraps or clamps. A host passing JSON it decoded itself (`json.Number` through `kavun.ValueOf`)
// gets the same rule — [embedding.md § Converting Go values](../../embedding.md#converting-go-values). `encode` raises
// kind `json_encoding` on a value with no JSON representation, naming the path to it once — `.items[0].price: value
// type <compiled-function/0> does not support JSON encoding` — and on text holding octets that are not symbols (JSON is
// UTF-8 by definition; encode such text as `bytes`, which goes as base64).
//
//kavun:module json pure

// Decode JSON text (`bytes` or `string`) into a Kavun value.
//
// `data` is `bytes` or a `string`.
//
//kavun:fn decode slot=0 pure (data) -> value
func jsonDecode(vm core.VM, args []core.Value) (core.Value, error) {

	b, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.decode", "first", "bytes/string", args[0].TypeName())
	}

	v, err := Decode(b)
	if err != nil {
		// Single translation point for the decoder's Go errors; an internal one (a contained decoder panic) is
		// already an *errs.Error and passes through with its own fatal severity.
		if e := errs.AsError(err); e != nil {
			return core.Undefined, err
		}
		return core.Undefined, errs.NewJSONDecodingError("(json.decode) " + err.Error())
	}

	return v, nil
}

// Encode a Kavun value as JSON bytes.
//
//kavun:fn encode slot=1 pure (value) -> bytes
func jsonEncode(vm core.VM, args []core.Value) (core.Value, error) {

	b, err := Encode(args[0])
	if err != nil {
		// Value.EncodeJSON already translated this into a json_encoding error; raise it as it stands.
		return core.Undefined, err
	}

	return core.NewBytesValue(b, false), nil
}

// Pretty-format JSON text with a line `prefix` and an `indent` per level.
//
// `data` is `bytes` or a `string` and `prefix` and `indent` are `string`s.
//
//kavun:fn indent slot=2 pure (data, prefix, indent) -> bytes
func jsonIndent(vm core.VM, args []core.Value) (core.Value, error) {

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

	var dst bytes.Buffer
	err := gojson.Indent(&dst, b, prefix, indent)
	if err != nil {
		return core.Undefined, errs.NewJSONDecodingError("(json.indent) " + err.Error())
	}

	return core.NewBytesValue(dst.Bytes(), false), nil
}

// Escape JSON text for safe embedding in HTML.
//
// `data` is `bytes` or a `string`.
//
//kavun:fn html_escape slot=3 pure (data) -> bytes
func jsonHTMLEscape(vm core.VM, args []core.Value) (core.Value, error) {

	b, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("json.html_escape", "first", "bytes/string", args[0].TypeName())
	}

	var dst bytes.Buffer
	gojson.HTMLEscape(&dst, b)
	return core.NewBytesValue(dst.Bytes(), false), nil
}
