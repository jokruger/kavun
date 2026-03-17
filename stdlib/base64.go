package stdlib

import (
	"encoding/base64"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

var base64Module = map[string]core.Object{
	"encode":         value.NewBuiltinFunction("encode", b64EncodeToString, 1, false),
	"decode":         value.NewBuiltinFunction("decode", b64DecodeString, 1, false),
	"raw_encode":     value.NewBuiltinFunction("raw_encode", b64RawEncodeToString, 1, false),
	"raw_decode":     value.NewBuiltinFunction("raw_decode", b64RawDecodeString, 1, false),
	"url_encode":     value.NewBuiltinFunction("url_encode", b64URLEncodeToString, 1, false),
	"url_decode":     value.NewBuiltinFunction("url_decode", b64URLDecodeString, 1, false),
	"raw_url_encode": value.NewBuiltinFunction("raw_url_encode", b64RawURLEncodeToString, 1, false),
	"raw_url_decode": value.NewBuiltinFunction("raw_url_decode", b64RawURLDecodeString, 1, false),
}

func b64RawURLDecodeString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	res, err := base64.RawURLEncoding.DecodeString(s1)
	if err != nil {
		return wrapError(err), nil
	}
	if len(res) > core.MaxBytesLen {
		return nil, gse.ErrBytesLimit
	}
	return value.NewBytes(res), nil
}

func b64URLDecodeString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	res, err := base64.URLEncoding.DecodeString(s1)
	if err != nil {
		return wrapError(err), nil
	}
	if len(res) > core.MaxBytesLen {
		return nil, gse.ErrBytesLimit
	}
	return value.NewBytes(res), nil
}

func b64RawDecodeString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	res, err := base64.RawStdEncoding.DecodeString(s1)
	if err != nil {
		return wrapError(err), nil
	}
	if len(res) > core.MaxBytesLen {
		return nil, gse.ErrBytesLimit
	}
	return value.NewBytes(res), nil
}

func b64DecodeString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	res, err := base64.StdEncoding.DecodeString(s1)
	if err != nil {
		return wrapError(err), nil
	}
	if len(res) > core.MaxBytesLen {
		return nil, gse.ErrBytesLimit
	}
	return value.NewBytes(res), nil
}

func b64RawURLEncodeToString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	y1, ok := args[0].AsByteSlice()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "bytes(compatible)", Found: args[0].TypeName()}
	}
	res := base64.RawURLEncoding.EncodeToString(y1)
	return value.NewString(res), nil
}

func b64URLEncodeToString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	y1, ok := args[0].AsByteSlice()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "bytes(compatible)", Found: args[0].TypeName()}
	}
	res := base64.URLEncoding.EncodeToString(y1)
	return value.NewString(res), nil
}

func b64RawEncodeToString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	y1, ok := args[0].AsByteSlice()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "bytes(compatible)", Found: args[0].TypeName()}
	}
	res := base64.RawStdEncoding.EncodeToString(y1)
	return value.NewString(res), nil
}

func b64EncodeToString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	y1, ok := args[0].AsByteSlice()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "bytes(compatible)", Found: args[0].TypeName()}
	}
	res := base64.StdEncoding.EncodeToString(y1)
	return value.NewString(res), nil
}
