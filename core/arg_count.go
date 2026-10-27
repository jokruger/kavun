package core

import (
	"errors"
	"strconv"

	"github.com/jokruger/kavun/errs"
)

// ErrArgCount is the sentinel a table entry's body (a Methods entry, a builtin or module function, a builtin
// closure) returns when it refuses its argument count: an upper bound or a gap on its rest, which the dispatcher's
// floor check does not see. The dispatcher translates it into the wrong_num_arguments error for the entry's name
// and signature, so a body never spells its own template. It is valid only as the direct answer of such a body;
// returned from anywhere else it is a defect (TestMemberArity reaches every entry).
var ErrArgCount = errors.New("argument count refused by the callee")

// ArgCountError builds the one wrong_num_arguments error of a native callable named name that refused n arguments.
// With a template the text quotes it — "(join) expected ([sep]), got 2 arguments"; without one (SignatureOf) it
// states the pair — "(f) expected 2 arguments, got 1 argument", "(f) expected 2 or more arguments, got 0
// arguments". A host hook with no declaration (a CallNamedMethod name path, a host type's Call) reports its count
// through it with SignatureOf.
func ArgCountError(sig Signature, name string, n int) error {
	if t := sig.Template(); t != "" {
		return errs.NewWrongNumArgumentsError(name, t, countNoun(n))
	}
	expected := countNoun(sig.Arity())
	if sig.IsVariadic() {
		expected = strconv.Itoa(sig.Arity()) + " or more arguments"
	}
	return errs.NewWrongNumArgumentsError(name, expected, countNoun(n))
}

// countResult is a dispatcher's translation of a body's error (out of line, called only when err != nil):
// ErrArgCount becomes the entry's wrong_num_arguments error, anything else passes through.
func countResult(sig Signature, name string, n int, res Value, err error) (Value, error) {
	if errors.Is(err, ErrArgCount) {
		return Undefined, ArgCountError(sig, name, n)
	}
	return res, err
}

// countNoun spells a count with its noun: "1 argument", "0 arguments", "2 arguments".
func countNoun(n int) string {
	if n == 1 {
		return "1 argument"
	}
	return strconv.Itoa(n) + " arguments"
}
