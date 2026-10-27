package core

import (
	"fmt"
	"math"
	"slices"

	"github.com/jokruger/kavun/internal/signature"
)

// Signature is what a callable accepts: its arity (fixed parameter count), whether it also accepts more
// (variadic), and — when built from a template — the exact count set the template names.
//
// A Signature is built once, at init, by ParseSignature / MustSignature (from a signature template such as
// "(width[, fill])") or by SignatureOf (from the bare pair). The zero value is SignatureOf(0, false): no
// template, arity 0, fixed, accepting exactly 0 arguments. A Signature is small and safe to copy by value.
type Signature struct {
	template string // the template verbatim; "" when built by SignatureOf
	// counts: the finite accepted counts besides arity, sorted, distinct, each below restFrom when hasRest.
	// Built once by ParseSignature and never written after, so copies share it read-only — which is also why a
	// Signature is not comparable with ==.
	counts   []int
	arity    int  // the fewest arguments accepted
	restFrom int  // with hasRest: every count >= restFrom is accepted
	hasRest  bool // whether the accepted set is open-ended
	variadic bool // whether any count other than arity is accepted
	// floorSpan makes the dispatcher's floor check one unsigned compare (admitsFloor): 0 when fixed (n must equal
	// arity), MaxInt when variadic (any n >= arity: n < arity wraps above MaxInt). The zero value is the fixed reading.
	floorSpan uint
}

// ParseSignature parses a signature template — the parameter list a script author would write for the same
// body — and derives what it accepts. The grammar is strict and admits only its canonical spelling:
//
//	template := reading { " | " reading }
//	reading  := "(" [ list ] ")"
//	list     := group | rest | name { ", " name } [ ", " rest | group ]
//	rest     := "..." name
//	group    := "[" ( ", " if a name precedes it ) name { ", " name } [ group ] "]"
//	name     := [a-z_][a-z0-9_]*
//
// An optional group continues the list it closes, so it opens with ", " inside the bracket ("(width[, fill])"),
// except at the start of an empty list ("([sep])"). A group is the last thing in its list; a rest
// ("...name") is the last item of a reading and never sits inside a group. A name appears once per reading.
// Every reading must accept at least one argument count that no other reading of the template accepts: a
// reading whose counts the others already cover (an identical reading included) is refused as redundant.
//
// Arity is the fewest arguments any reading accepts; the signature is variadic when it accepts any other count.
// A malformed template answers a Go error naming the template, the problem and its byte offset.
func ParseSignature(template string) (Signature, error) {
	sh, err := signature.Parse(template)
	if err != nil {
		return Signature{}, err
	}
	return Signature{template: template, counts: sh.Counts, arity: sh.Arity, restFrom: sh.RestFrom, hasRest: sh.HasRest,
		variadic: sh.Variadic, floorSpan: floorSpanOf(sh.Variadic)}, nil
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
	return Signature{arity: arity, restFrom: arity, hasRest: variadic, variadic: variadic, floorSpan: floorSpanOf(variadic)}
}

// Arity answers the count of fixed parameters: the fewest arguments the callable accepts.
func (s Signature) Arity() int { return s.arity }

// IsVariadic reports whether the callable accepts any argument count other than its arity.
func (s Signature) IsVariadic() bool { return s.variadic }

// Template answers the template the Signature was parsed from, verbatim; "" when built by SignatureOf.
func (s Signature) Template() string { return s.template }

// Accepts reports whether the callable accepts exactly k arguments. A negative k is never accepted.
func (s Signature) Accepts(k int) bool {
	if k < 0 {
		return false
	}
	return k == s.arity || (s.hasRest && k >= s.restFrom) || slices.Contains(s.counts, k)
}

// Max answers the largest argument count the signature names and whether the accepted set is open above it. For a
// closed set n is its largest member; for an open set (a "...rest", or SignatureOf with variadic) n is the count
// the open run starts at — every k >= n is accepted. Without a template n is the arity.
func (s Signature) Max() (n int, open bool) {
	if s.hasRest {
		return s.restFrom, true
	}
	if len(s.counts) > 0 {
		return s.counts[len(s.counts)-1], false
	}
	return s.arity, false
}

// admitsFloor reports whether n passes the dispatcher's floor check: exactly the arity for a fixed signature, at
// least the arity for a variadic one. The full count set (Accepts) is the body's to bound, not the dispatcher's.
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
