package core

import (
	"fmt"
	"math"

	"github.com/jokruger/kavun/signature"
)

// Signature is what the dispatcher needs of a callable: its template (for the wrong-count error), its arity
// (fixed parameter count) and whether it also accepts more (variadic). Nothing parsed from the template is kept:
// the exact count set and the readings belong to package signature — a caller that wants them parses the
// template (signature.Parse(sig.Template())).
//
// A Signature is built once, at init, by ParseSignature / MustSignature (from a signature template such as
// "(width[, fill])") or by SignatureOf (from the bare pair). The zero value is SignatureOf(0, false): no
// template, arity 0, fixed. A Signature is small, safe to copy by value and comparable with ==.
type Signature struct {
	template string // the template verbatim; "" when built by SignatureOf
	arity    int    // the fewest arguments accepted
	// floorSpan makes the dispatcher's floor check one unsigned compare (admitsFloor): 0 when fixed (n must equal
	// arity), MaxInt when variadic (any n >= arity: n < arity wraps above MaxInt). The zero value is the fixed reading.
	floorSpan uint
}

// ParseSignature parses a signature template — the parameter list a script author would write for the same
// body — and keeps what dispatch needs. The grammar and the redundancy rule are documented on package signature
// (signature.Parse). Arity is the fewest arguments any reading accepts; the signature is variadic when it accepts
// any other count. A malformed template answers a Go error naming the template, the problem and its byte offset.
func ParseSignature(template string) (Signature, error) {
	sh, err := signature.Parse(template)
	if err != nil {
		return Signature{}, err
	}
	return Signature{template: template, arity: sh.Arity, floorSpan: floorSpanOf(sh.Variadic)}, nil
}

// MustSignature is ParseSignature for a template known at init: it panics on a malformed template, like
// regexp.MustCompile. A panic here is a programming error in the declaration, never a runtime condition.
func MustSignature(template string) Signature {
	s, err := ParseSignature(template)
	if err != nil {
		panic(err)
	}
	return s
}

// SignatureOf builds a Signature from the bare pair, with no template: it accepts k == arity, or any
// k > arity when variadic. A negative arity is a programming error and panics.
func SignatureOf(arity int, variadic bool) Signature {
	if arity < 0 {
		panic(fmt.Sprintf("core.SignatureOf: negative arity %d", arity))
	}
	return Signature{arity: arity, floorSpan: floorSpanOf(variadic)}
}

// Arity answers the count of fixed parameters: the fewest arguments the callable accepts.
func (s Signature) Arity() int { return s.arity }

// IsVariadic reports whether the callable accepts any argument count other than its arity.
func (s Signature) IsVariadic() bool { return s.floorSpan != 0 }

// Template answers the template the Signature was parsed from, verbatim; "" when built by SignatureOf.
func (s Signature) Template() string { return s.template }

// admitsFloor reports whether n passes the dispatcher's floor check: exactly the arity for a fixed signature, at
// least the arity for a variadic one. The full count set (signature.Shape.Accepts) is the body's to bound, not the
// dispatcher's.
func (s Signature) admitsFloor(n int) bool {
	// n - arity wraps above MaxInt when n < arity, so one unsigned compare covers both bounds
	return uint(n-s.arity) <= s.floorSpan
}

// floorSpanOf answers the floorSpan of a fixed (0) or variadic (every n >= arity) signature.
func floorSpanOf(variadic bool) uint {
	if variadic {
		return math.MaxInt
	}
	return 0
}
