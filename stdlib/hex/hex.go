package hex

import (
	"encoding/hex"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
)

// Hexadecimal encoding and decoding of bytes.
//
// Example:
//
// ```go
// hex = import("hex")
// hex.encode(bytes("ok"))
// ```
//
// **Failures.** A malformed input raises kind `conversion`: `(hex.decode) encoding/hex: invalid byte …`.
//
//kavun:module hex pure

// Decode hexadecimal text.
//
// `s` is a `string`.
//
//kavun:fn decode slot=1 pure (s) -> bytes
func hexDecodeString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("hex.decode", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := hex.DecodeString(s1)
	if err != nil {
		return modkit.RaiseGo(errs.KindConversion, "hex.decode", err)
	}
	return core.NewBytesValue(res, false), nil
}

// Encode `data` as hexadecimal text.
//
// `data` is `bytes`.
//
//kavun:fn encode slot=0 pure (data) -> string
func hexEncodeToString(vm core.VM, args []core.Value) (ret core.Value, err error) {
	y1, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("hex.encode", "first", "bytes(compatible)", args[0].TypeName())
	}
	res := hex.EncodeToString(y1)
	return core.NewStringValue(res), nil
}
