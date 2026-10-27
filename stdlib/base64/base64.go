package base64

import (
	"encoding/base64"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
)

// Base64 encoding and decoding: the standard and URL-safe alphabets, with or without padding.
//
// Example:
//
// ```go
// base64 = import("base64")
// base64.encode(bytes("hello"))
// ```
//
// **Failures.** A malformed input raises kind `conversion`, with the function name in the message: `(base64.decode)
// illegal base64 data at input byte 0`.
//
//kavun:module base64 pure

// Decode URL-safe Base64 text written without padding.
//
// `s` is a `string`.
//
//kavun:fn raw_url_decode slot=7 pure (s) -> bytes
func b64RawURLDecodeString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.raw_url_decode", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := base64.RawURLEncoding.DecodeString(s1)
	if err != nil {
		return modkit.RaiseGo(errs.KindConversion, "base64.raw_url_decode", err)
	}
	return core.NewBytesValue(res, false), nil
}

// Decode URL-safe Base64 text.
//
// `s` is a `string`.
//
//kavun:fn url_decode slot=5 pure (s) -> bytes
func b64URLDecodeString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.url_decode", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := base64.URLEncoding.DecodeString(s1)
	if err != nil {
		return modkit.RaiseGo(errs.KindConversion, "base64.url_decode", err)
	}
	return core.NewBytesValue(res, false), nil
}

// Decode standard Base64 text written without padding.
//
// `s` is a `string`.
//
//kavun:fn raw_decode slot=3 pure (s) -> bytes
func b64RawDecodeString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.raw_decode", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := base64.RawStdEncoding.DecodeString(s1)
	if err != nil {
		return modkit.RaiseGo(errs.KindConversion, "base64.raw_decode", err)
	}
	return core.NewBytesValue(res, false), nil
}

// Decode standard Base64 text.
//
// `s` is a `string`.
//
//kavun:fn decode slot=1 pure (s) -> bytes
func b64DecodeString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.decode", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := base64.StdEncoding.DecodeString(s1)
	if err != nil {
		return modkit.RaiseGo(errs.KindConversion, "base64.decode", err)
	}
	return core.NewBytesValue(res, false), nil
}

// Encode `data` as URL-safe Base64 without padding.
//
// `data` is `bytes`.
//
//kavun:fn raw_url_encode slot=6 pure (data) -> string
func b64RawURLEncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	y1, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.raw_url_encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := base64.RawURLEncoding.EncodeToString(y1)
	return core.NewStringValue(res), nil
}

// Encode `data` as URL-safe Base64.
//
// `data` is `bytes`.
//
//kavun:fn url_encode slot=4 pure (data) -> string
func b64URLEncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	y1, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.url_encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := base64.URLEncoding.EncodeToString(y1)
	return core.NewStringValue(res), nil
}

// Encode `data` as standard Base64 without padding.
//
// `data` is `bytes`.
//
//kavun:fn raw_encode slot=2 pure (data) -> string
func b64RawEncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	y1, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.raw_encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := base64.RawStdEncoding.EncodeToString(y1)
	return core.NewStringValue(res), nil
}

// Encode `data` as standard Base64.
//
// `data` is `bytes`.
//
//kavun:fn encode slot=0 pure (data) -> string
func b64EncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	y1, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("base64.encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := base64.StdEncoding.EncodeToString(y1)
	return core.NewStringValue(res), nil
}
