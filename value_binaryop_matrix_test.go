package kavun_test

// This file is a literal, checkable transcription of the type/operator redesigns' full
// (lhs, op, rhs) -> result reference table — see docs/types.md's "Operators across types" for the
// narrative version. One test function per matrix section, one t.Run subtest per row (or
// representative row, for the universal undefined/error wildcards), so a failure here points
// straight at the specific behavior to re-check. Every row transcribed here was cross-checked
// against the real core/*.go behavior as it was written, across both the arithmetic/rank redesign
// and the later comparison (==/!=/< > <= >=) redesign.
//
// Deliberate departure from this repo's usual flat-sequential test style (see kavun_test.go's
// CLAUDE.md convention): this test's entire job is being a checkable transcription of the matrix,
// and t.Run subtest names are what make "which row broke" legible in `go test -v` output.

import (
	"testing"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/internal/require"
)

func matrixOK(t *testing.T, name string, lhs core.Value, op token.Token, rhs core.Value, want core.Value) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		got, err := lhs.BinaryOp(op, rhs)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}

func matrixErr(t *testing.T, name string, lhs core.Value, op token.Token, rhs core.Value) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		_, err := lhs.BinaryOp(op, rhs)
		require.Error(t, err)
	})
}

// matrixBoth asserts a cross-type row in both directions — "lhs op rhs" and "rhs op lhs" — as two
// separate subtests. Which side's hook actually answers is deliberately never asserted anywhere in
// this file: for the mirrored numeric pairings it is unobservable by design (see
// docs/extending-types.md's "Mirrored ownership"), and everywhere else it is exactly the detail
// rule 3's delegate step exists to hide. The reversed expectation is passed explicitly, because
// `-`, `/` and `%` are where a mirror written with inverted operands — or a reflected branch that
// forgot to swap its operands — actually shows up.
func matrixBoth(t *testing.T, lname string, lhs core.Value, op token.Token, rname string, rhs core.Value, want core.Value, wantReversed core.Value) {
	t.Helper()
	matrixOK(t, lname+" "+op.String()+" "+rname+" -> "+want.TypeName(), lhs, op, rhs, want)
	matrixOK(t, rname+" "+op.String()+" "+lname+" -> "+wantReversed.TypeName(), rhs, op, lhs, wantReversed)
}

// matrixBothErr is matrixBoth for a pairing that is a vm error whichever side is written first.
func matrixBothErr(t *testing.T, lname string, lhs core.Value, op token.Token, rname string, rhs core.Value) {
	t.Helper()
	matrixErr(t, lname+" "+op.String()+" "+rname+" -> vm error", lhs, op, rhs)
	matrixErr(t, rname+" "+op.String()+" "+lname+" -> vm error", rhs, op, lhs)
}

// matrixOrder asserts all four ordering operators in both directions for a pair where lo < hi holds
// semantically. Same rationale as matrixBoth: the result is the contract, the answering hook is not
// — and an operator mapping that got inverted (Less answered as Greater) only surfaces when all
// four are checked from both sides.
func matrixOrder(t *testing.T, loName string, lo core.Value, hiName string, hi core.Value) {
	t.Helper()
	for _, c := range []struct {
		op         token.Token
		loHi, hiLo core.Value
	}{
		{token.Less, core.True, core.False},
		{token.Greater, core.False, core.True},
		{token.LessEq, core.True, core.False},
		{token.GreaterEq, core.False, core.True},
	} {
		matrixOK(t, loName+" "+c.op.String()+" "+hiName, lo, c.op, hi, c.loHi)
		matrixOK(t, hiName+" "+c.op.String()+" "+loName, hi, c.op, lo, c.hiLo)
	}
}

func matrixUnaryOK(t *testing.T, name string, op token.Token, rhs core.Value, want core.Value) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		got, err := rhs.UnaryOp(op)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}

func matrixUnaryErr(t *testing.T, name string, op token.Token, rhs core.Value) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		_, err := rhs.UnaryOp(op)
		require.Error(t, err)
	})
}

// ## Universal — undefined / error
// Wildcard rows ("any, either order") are tested against a representative sample of types
// (int, string, bool, array, time) rather than all 16 — the matrix's own convention for these
// rows, since the behavior is identical for every other type by construction (undefined/error's
// hooks never inspect the other operand's type at all).
//
// ==/!= rows are tested via Value.Equal() directly, not BinaryOp() — bc.Equal/bc.NotEqual never
// dispatch to BinaryOp for any type, undefined/error included (confirmed by reading vm/vm.go).
func TestMatrix_Universal(t *testing.T) {
	i := core.IntValue(1)
	s := core.NewStringValue("x")
	b := core.True
	arr := core.NewArrayValue([]core.Value{core.IntValue(1)}, false)

	t.Run("undefined == undefined -> true", func(t *testing.T) {
		require.True(t, core.Undefined.Equal(core.Undefined))
	})
	for name, v := range map[string]core.Value{"int": i, "string": s, "bool": b, "array": arr} {
		t.Run("undefined == "+name+" -> false", func(t *testing.T) {
			require.False(t, core.Undefined.Equal(v))
		})
		t.Run(name+" == undefined -> false", func(t *testing.T) {
			require.False(t, v.Equal(core.Undefined))
		})
		matrixOK(t, "undefined + "+name+" -> undefined", core.Undefined, token.Add, v, core.Undefined)
		matrixOK(t, name+" + undefined -> undefined", v, token.Add, core.Undefined, core.Undefined)
		matrixOK(t, "undefined < "+name+" -> undefined", core.Undefined, token.Less, v, core.Undefined)
	}

	e := core.NewErrorValue(core.Undefined, core.KindUser, false)
	e2 := core.NewErrorValue(core.NewStringValue("other"), core.KindUser, false)
	t.Run("error == error (same payload) -> true", func(t *testing.T) {
		require.True(t, e.Equal(core.NewErrorValue(core.Undefined, core.KindUser, false)))
	})
	t.Run("error == error (different payload) -> false", func(t *testing.T) {
		require.False(t, e.Equal(e2))
	})
	for name, v := range map[string]core.Value{"int": i, "string": s, "bool": b, "array": arr} {
		t.Run("error == "+name+" -> false", func(t *testing.T) {
			require.False(t, e.Equal(v))
		})
		t.Run(name+" == error -> false", func(t *testing.T) {
			require.False(t, v.Equal(e))
		})
		matrixErr(t, "error + "+name+" -> vm error", e, token.Add, v)
		matrixErr(t, name+" + error -> vm error", v, token.Add, e)
	}

	// undefined wins over error, per the implementor contract.
	matrixOK(t, "error + undefined -> undefined", e, token.Add, core.Undefined, core.Undefined)
	matrixOK(t, "undefined + error -> undefined", core.Undefined, token.Add, e, core.Undefined)
}

// ## Unary
//
// `!` is deliberately NOT tested via UnaryOp() here — it never reaches that hook at all (vm.go's
// UnaryNot opcode calls IsTrue() directly for every type, confirmed by reading vm/vm.go), so its
// rows are tested against IsTrue() instead, matching how the VM actually computes it.
func TestMatrix_Unary(t *testing.T) {
	t.Run("! undefined -> true (IsTrue is false, negated)", func(t *testing.T) {
		require.False(t, core.Undefined.IsTrue())
	})
	t.Run("! error -> false (IsTrue is unconditionally true, negated)", func(t *testing.T) {
		require.True(t, core.NewErrorValue(core.Undefined, core.KindUser, false).IsTrue())
	})

	matrixUnaryOK(t, "- int -> int", token.Sub, core.IntValue(5), core.IntValue(-5))
	matrixUnaryOK(t, "- float -> float", token.Sub, core.FloatValue(5), core.FloatValue(-5))
	matrixUnaryOK(t, "- byte -> byte (ring negation)", token.Sub, core.ByteValue(1), core.ByteValue(255))
	matrixUnaryOK(t, "- undefined -> undefined", token.Sub, core.Undefined, core.Undefined)
	matrixUnaryOK(t, "^ undefined -> undefined", token.Xor, core.Undefined, core.Undefined)
	matrixUnaryOK(t, "^ int -> int", token.Xor, core.IntValue(0), core.IntValue(^int64(0)))
	matrixUnaryOK(t, "^ byte -> byte", token.Xor, core.ByteValue(0), core.ByteValue(0xFF))
	matrixUnaryOK(t, "^ bool -> bool", token.Xor, core.True, core.False)

	// deliberately not defined
	matrixUnaryErr(t, "- rune -> vm error (position type, not a ring)", token.Sub, core.RuneValue('a'))
	matrixUnaryErr(t, "- bool -> vm error (no arithmetic at all)", token.Sub, core.True)
	matrixUnaryErr(t, "- string -> vm error (no implicit number parsing)", token.Sub, core.NewStringValue("5"))
	matrixUnaryErr(t, "^ rune -> vm error (rune excluded from bitwise entirely)", token.Xor, core.RuneValue('a'))
	matrixUnaryErr(t, "- error -> vm error", token.Sub, core.NewErrorValue(core.Undefined, core.KindUser, false))
	matrixUnaryErr(t, "^ error -> vm error", token.Xor, core.NewErrorValue(core.Undefined, core.KindUser, false))
}

// ## Numeric arithmetic — int, float, decimal
//
// int/float and int/decimal are implemented on both sides for the hot path (see
// docs/extending-types.md's "Mirrored ownership"), so every cross-type row here is asserted in both
// directions via matrixBoth — the reversed rows are the only thing that would catch one half of a
// mirror drifting from the other, or being written with its operands inverted.
func TestMatrix_NumericArithmetic(t *testing.T) {
	dec := func(i int64) core.Value { return core.NewDecimalValue(dec128.FromInt64(i)) }
	decs := func(s string) core.Value { return core.NewDecimalValue(dec128.FromString(s)) }

	// same-type
	matrixOK(t, "int + int -> int", core.IntValue(10), token.Add, core.IntValue(4), core.IntValue(14))
	matrixOK(t, "int - int -> int", core.IntValue(10), token.Sub, core.IntValue(4), core.IntValue(6))
	matrixOK(t, "int * int -> int", core.IntValue(10), token.Mul, core.IntValue(4), core.IntValue(40))
	matrixOK(t, "int / int -> int (truncating)", core.IntValue(10), token.Quo, core.IntValue(4), core.IntValue(2))
	matrixOK(t, "int % int -> int", core.IntValue(10), token.Rem, core.IntValue(4), core.IntValue(2))
	matrixOK(t, "float + float -> float", core.FloatValue(10), token.Add, core.FloatValue(4), core.FloatValue(14))
	matrixOK(t, "float - float -> float", core.FloatValue(10), token.Sub, core.FloatValue(4), core.FloatValue(6))
	matrixOK(t, "float * float -> float", core.FloatValue(10), token.Mul, core.FloatValue(4), core.FloatValue(40))
	matrixOK(t, "float / float -> float", core.FloatValue(10), token.Quo, core.FloatValue(4), core.FloatValue(2.5))
	matrixOK(t, "float % float -> float", core.FloatValue(10), token.Rem, core.FloatValue(4), core.FloatValue(2))
	matrixOK(t, "decimal + decimal -> decimal", dec(10), token.Add, dec(4), dec(14))
	matrixOK(t, "decimal - decimal -> decimal", dec(10), token.Sub, dec(4), dec(6))
	matrixOK(t, "decimal * decimal -> decimal", dec(10), token.Mul, dec(4), dec(40))
	matrixOK(t, "decimal / decimal -> decimal", dec(10), token.Quo, dec(4), decs("2.5"))
	matrixOK(t, "decimal % decimal -> decimal", dec(10), token.Rem, dec(4), dec(2))

	// int op float, both directions. % was found missing for float/decimal during Phase 6's final
	// sweep — core/float.go and core/decimal.go had no token.Rem case at all, contradicting this
	// matrix's own stated design; fixed alongside this test.
	matrixBoth(t, "int", core.IntValue(10), token.Add, "float", core.FloatValue(4), core.FloatValue(14), core.FloatValue(14))
	matrixBoth(t, "int", core.IntValue(10), token.Sub, "float", core.FloatValue(4), core.FloatValue(6), core.FloatValue(-6))
	matrixBoth(t, "int", core.IntValue(10), token.Mul, "float", core.FloatValue(4), core.FloatValue(40), core.FloatValue(40))
	matrixBoth(t, "int", core.IntValue(10), token.Quo, "float", core.FloatValue(4), core.FloatValue(2.5), core.FloatValue(0.4))
	matrixBoth(t, "int", core.IntValue(10), token.Rem, "float", core.FloatValue(4), core.FloatValue(2), core.FloatValue(4))

	// int op decimal, both directions
	matrixBoth(t, "int", core.IntValue(10), token.Add, "decimal", dec(4), dec(14), dec(14))
	matrixBoth(t, "int", core.IntValue(10), token.Sub, "decimal", dec(4), dec(6), dec(-6))
	matrixBoth(t, "int", core.IntValue(10), token.Mul, "decimal", dec(4), dec(40), dec(40))
	matrixBoth(t, "int", core.IntValue(10), token.Quo, "decimal", dec(4), decs("2.5"), decs("0.4"))
	matrixBoth(t, "int", core.IntValue(10), token.Rem, "decimal", dec(4), dec(2), dec(4))

	// float and decimal deliberately do not accept each other for arithmetic, in either direction —
	// no silent widening, no lossy bridge. Ordering between them IS defined and exact, see
	// TestMatrix_Comparisons.
	for _, op := range []token.Token{token.Add, token.Sub, token.Mul, token.Quo, token.Rem} {
		matrixBothErr(t, "float", core.FloatValue(10), op, "decimal", dec(4))
	}
}

// ## Bitwise — same-type only, byte/int only; shift accepts int as the count
func TestMatrix_Bitwise(t *testing.T) {
	matrixOK(t, "byte & byte -> byte", core.ByteValue(0xF0), token.And, core.ByteValue(0x0F), core.ByteValue(0))
	matrixOK(t, "int & int -> int", core.IntValue(0xF0), token.And, core.IntValue(0x0F), core.IntValue(0))
	matrixOK(t, "byte << byte -> byte", core.ByteValue(1), token.Shl, core.ByteValue(4), core.ByteValue(16))
	matrixOK(t, "byte << int -> byte (shift count exception)", core.ByteValue(1), token.Shl, core.IntValue(4), core.ByteValue(16))
	matrixOK(t, "int << int -> int", core.IntValue(1), token.Shl, core.IntValue(4), core.IntValue(16))

	matrixErr(t, "rune & rune -> vm error (rune has no bitwise at all)", core.RuneValue('a'), token.And, core.RuneValue('a'))
	matrixErr(t, "byte & int -> vm error (bitwise is same-type only, count exception excluded)", core.ByteValue(1), token.And, core.IntValue(1))
	matrixErr(t, "int << byte -> vm error (count exception is one-directional)", core.IntValue(1), token.Shl, core.ByteValue(1))
}

// ## Ordinal — byte, rune arithmetic against int and same-type
func TestMatrix_Ordinal(t *testing.T) {
	matrixOK(t, "byte + int -> byte, wraps", core.ByteValue(255), token.Add, core.IntValue(1), core.ByteValue(0))
	matrixOK(t, "int + byte -> byte, wraps", core.IntValue(1), token.Add, core.ByteValue(255), core.ByteValue(0))
	matrixOK(t, "byte - int -> byte, wraps", core.ByteValue(0), token.Sub, core.IntValue(1), core.ByteValue(255))
	matrixOK(t, "int - byte -> byte, wraps (byte is a ring in both directions)", core.IntValue(0), token.Sub, core.ByteValue(1), core.ByteValue(255))
	matrixOK(t, "byte - byte -> byte (ring subtraction, not a distance)", core.ByteValue(0), token.Sub, core.ByteValue(1), core.ByteValue(255))
	matrixOK(t, "byte + byte -> byte, wraps", core.ByteValue(255), token.Add, core.ByteValue(1), core.ByteValue(0))

	matrixOK(t, "rune + int -> rune", core.RuneValue('a'), token.Add, core.IntValue(1), core.RuneValue('b'))
	matrixOK(t, "int + rune -> rune", core.IntValue(1), token.Add, core.RuneValue('a'), core.RuneValue('b'))
	matrixOK(t, "rune - int -> rune", core.RuneValue('b'), token.Sub, core.IntValue(1), core.RuneValue('a'))
	matrixOK(t, "rune - rune -> int (genuine distance, unlike byte - byte)", core.RuneValue('b'), token.Sub, core.RuneValue('a'), core.IntValue(1))

	matrixOK(t, "byte - rune -> int (byte widens to rune via rule 2)", core.ByteValue('B'), token.Sub, core.RuneValue('A'), core.IntValue(1))
	matrixOK(t, "rune - byte -> int (reversed order)", core.RuneValue('B'), token.Sub, core.ByteValue('A'), core.IntValue(1))

	matrixErr(t, "byte + rune -> vm error (widens to rune, inherits rune+rune's rejection)", core.ByteValue('A'), token.Add, core.RuneValue('B'))
	matrixErr(t, "rune + byte -> vm error (reversed order)", core.RuneValue('B'), token.Add, core.ByteValue('A'))
}

// ## Sequence/text — concatenation, fixed rank bytes > runes > string
func TestMatrix_SequenceConcat(t *testing.T) {
	str := func(s string) core.Value { return core.NewStringValue(s) }
	byt := func(s string) core.Value { return core.NewBytesValue([]byte(s), false) }
	run := func(s string) core.Value { return core.NewRunesValue([]rune(s), false) }

	matrixOK(t, "string + string -> string", str("a"), token.Add, str("b"), str("ab"))
	matrixOK(t, "bytes + bytes -> bytes", byt("a"), token.Add, byt("b"), byt("ab"))
	matrixOK(t, "runes + runes -> runes", run("a"), token.Add, run("b"), run("ab"))

	matrixOK(t, "bytes + runes -> bytes (rank bytes > runes > string)", byt("a"), token.Add, run("b"), byt("ab"))
	matrixOK(t, "runes + bytes -> bytes", run("a"), token.Add, byt("b"), byt("ab"))
	matrixOK(t, "bytes + string -> bytes", byt("a"), token.Add, str("b"), byt("ab"))
	matrixOK(t, "string + bytes -> bytes", str("a"), token.Add, byt("b"), byt("ab"))
	matrixOK(t, "runes + string -> runes", run("a"), token.Add, str("b"), run("ab"))
	matrixOK(t, "string + runes -> runes", str("a"), token.Add, run("b"), run("ab"))

	matrixOK(t, "byte + bytes -> bytes", core.ByteValue('a'), token.Add, byt("b"), byt("ab"))
	matrixOK(t, "bytes + byte -> bytes (reversed)", byt("a"), token.Add, core.ByteValue('b'), byt("ab"))
	matrixOK(t, "rune + runes -> runes", core.RuneValue('a'), token.Add, run("b"), run("ab"))
	matrixOK(t, "runes + rune -> runes (reversed)", run("a"), token.Add, core.RuneValue('b'), run("ab"))
	matrixOK(t, "rune + string -> string", core.RuneValue('a'), token.Add, str("b"), str("ab"))
	matrixOK(t, "string + rune -> string (reversed)", str("a"), token.Add, core.RuneValue('b'), str("ab"))
	matrixOK(t, "rune + bytes -> bytes", core.RuneValue('a'), token.Add, byt("b"), byt("ab"))
	matrixOK(t, "bytes + rune -> bytes (reversed)", byt("a"), token.Add, core.RuneValue('b'), byt("ab"))

	// content order still respects which operand was written first, independent of which type wins
	matrixOK(t, "runes + bytes content order preserved", run("cd"), token.Add, byt("ab"), byt("cdab"))
	matrixOK(t, "bytes + runes content order preserved", byt("ab"), token.Add, run("cd"), byt("abcd"))
}

// ## Sequence/text — removal, lhs-only, no reflected direction
func TestMatrix_SequenceRemoval(t *testing.T) {
	str := func(s string) core.Value { return core.NewStringValue(s) }
	byt := func(s string) core.Value { return core.NewBytesValue([]byte(s), false) }
	run := func(s string) core.Value { return core.NewRunesValue([]rune(s), false) }

	matrixOK(t, "string - string -> string", str("foobar"), token.Sub, str("bar"), str("foo"))
	matrixOK(t, "bytes - byte -> bytes", byt("banana"), token.Sub, core.ByteValue('a'), byt("bnn"))
	matrixOK(t, "bytes - bytes -> bytes", byt("banana"), token.Sub, byt("an"), byt("ba"))
	matrixOK(t, "runes - rune -> runes", run("banana"), token.Sub, core.RuneValue('a'), run("bnn"))
	matrixOK(t, "runes - runes -> runes", run("banana"), token.Sub, run("an"), run("ba"))
	matrixOK(t, "string - rune -> string", str("banana"), token.Sub, core.RuneValue('a'), str("bnn"))
	matrixOK(t, "string - runes -> string", str("banana"), token.Sub, run("an"), str("ba"))

	// bytes owns every pairing it's in for removal too, same as + and ordering — the rhs just
	// needs a byte encoding, which rune/string/runes all already have (via the same conversions +
	// and ordering already use). This was a real gap found and fixed: bytes - string/runes used to
	// be a vm error even though the encoding machinery was already sitting right there, unused.
	matrixOK(t, "bytes - rune -> bytes", byt("banana"), token.Sub, core.RuneValue('a'), byt("bnn"))
	matrixOK(t, "bytes - string -> bytes", byt("banana"), token.Sub, str("an"), byt("ba"))
	matrixOK(t, "bytes - runes -> bytes", byt("banana"), token.Sub, run("an"), byt("ba"))

	// deliberately not defined
	matrixErr(t, "string - byte -> vm error", str("foo"), token.Sub, core.ByteValue('f'))
	matrixErr(t, "string - bytes -> vm error", str("foo"), token.Sub, byt("f"))
	matrixErr(t, "runes - bytes -> vm error", run("foo"), token.Sub, byt("f"))
	matrixErr(t, "runes - string -> vm error", run("foo"), token.Sub, str("f"))
	matrixErr(t, "rune - string -> vm error (no reflected direction for removal)", core.RuneValue('f'), token.Sub, str("foo"))
	matrixErr(t, "byte - bytes -> vm error (no reflected direction for removal)", core.ByteValue('a'), token.Sub, byt("banana"))
}

// ## Collections — array, dict, record
func TestMatrix_Collections(t *testing.T) {
	arr := func(vs ...int64) core.Value {
		vals := make([]core.Value, len(vs))
		for i, v := range vs {
			vals[i] = core.IntValue(v)
		}
		return core.NewArrayValue(vals, false)
	}

	matrixOK(t, "array + array -> array", arr(1, 2), token.Add, arr(3, 4), arr(1, 2, 3, 4))
	matrixErr(t, "array + int -> vm error (no scalar append)", arr(1), token.Add, core.IntValue(1))
	matrixErr(t, "int + array -> vm error (no scalar prepend)", core.IntValue(1), token.Add, arr(1))
	matrixErr(t, "array - int -> vm error", arr(1), token.Sub, core.IntValue(1))
	matrixErr(t, "array - array -> vm error (ambiguous)", arr(1), token.Sub, arr(1))

	d := func(m map[string]core.Value) core.Value { return core.NewDictValue(m, false) }
	r := func(m map[string]core.Value) core.Value { return core.NewRecordValue(m, false) }

	dictResult, err := d(map[string]core.Value{"a": core.IntValue(1)}).BinaryOp(token.Add, d(map[string]core.Value{"b": core.IntValue(2)}))
	require.NoError(t, err)
	require.Equal(t, d(map[string]core.Value{"a": core.IntValue(1), "b": core.IntValue(2)}), dictResult)

	recResult, err := r(map[string]core.Value{"a": core.IntValue(1)}).BinaryOp(token.Add, r(map[string]core.Value{"b": core.IntValue(2)}))
	require.NoError(t, err)
	require.Equal(t, r(map[string]core.Value{"a": core.IntValue(1), "b": core.IntValue(2)}), recResult)

	// record + dict / dict + record -> dict, rhs wins collisions, regardless of which side triggered reflection
	mixed1, err := r(map[string]core.Value{"a": core.IntValue(1)}).BinaryOp(token.Add, d(map[string]core.Value{"a": core.IntValue(2)}))
	require.NoError(t, err)
	require.Equal(t, d(map[string]core.Value{"a": core.IntValue(2)}), mixed1)

	mixed2, err := d(map[string]core.Value{"a": core.IntValue(1)}).BinaryOp(token.Add, r(map[string]core.Value{"a": core.IntValue(2)}))
	require.NoError(t, err)
	require.Equal(t, d(map[string]core.Value{"a": core.IntValue(2)}), mixed2)

	delResult, err := d(map[string]core.Value{"a": core.IntValue(1), "b": core.IntValue(2)}).BinaryOp(token.Sub, core.NewStringValue("a"))
	require.NoError(t, err)
	require.Equal(t, d(map[string]core.Value{"b": core.IntValue(2)}), delResult)

	matrixErr(t, "record - string -> vm error (record has no member functions/operator removal)", r(map[string]core.Value{"a": core.IntValue(1)}), token.Sub, core.NewStringValue("a"))
}

// ## Domain-specific — time
//
// `time + int`/`time - int` treat the int as raw nanoseconds (time.Duration(r), and Go's
// time.Duration is nanoseconds) — not seconds, despite how that might read at a glance. See
// docs/types.md's "time" section and docs/types/time.md.
func TestMatrix_Time(t *testing.T) {
	baseTime := time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC)
	epoch := core.NewTimeValue(baseTime)
	later := core.NewTimeValue(baseTime.Add(5 * time.Nanosecond))

	matrixOK(t, "time + int -> time (add nanoseconds)", epoch, token.Add, core.IntValue(5), later)
	matrixOK(t, "time - int -> time (subtract nanoseconds)", later, token.Sub, core.IntValue(5), epoch)
	matrixOK(t, "time - time -> int (duration in nanoseconds)", later, token.Sub, epoch, core.IntValue(5))
	matrixOK(t, "time < time -> bool", epoch, token.Less, later, core.True)
}

// ## Comparisons — ordering
//
// Every cross-type pairing goes through matrixOrder — all four ordering operators, both directions.
// The precision rows below stay explicit, since their whole point is an asymmetric result that a
// symmetric helper would hide.
func TestMatrix_Comparisons(t *testing.T) {
	dec := func(i int64) core.Value { return core.NewDecimalValue(dec128.FromInt64(i)) }
	str := func(s string) core.Value { return core.NewStringValue(s) }
	byt := func(s string) core.Value { return core.NewBytesValue([]byte(s), false) }
	run := func(s string) core.Value { return core.NewRunesValue([]rune(s), false) }

	// same-type
	matrixOK(t, "int < int", core.IntValue(1), token.Less, core.IntValue(2), core.True)
	matrixOK(t, "float < float", core.FloatValue(1), token.Less, core.FloatValue(2), core.True)
	matrixOK(t, "decimal < decimal", dec(1), token.Less, dec(2), core.True)
	matrixOK(t, "byte < byte", core.ByteValue(1), token.Less, core.ByteValue(2), core.True)
	matrixOK(t, "rune < rune", core.RuneValue('a'), token.Less, core.RuneValue('b'), core.True)
	matrixOK(t, "string < string", str("a"), token.Less, str("b"), core.True)
	matrixOK(t, "bytes < bytes", byt("a"), token.Less, byt("b"), core.True)
	matrixOK(t, "runes < runes", run("a"), token.Less, run("b"), core.True)
	matrixOK(t, "bool ordering: false < true", core.False, token.Less, core.True, core.True)

	// numeric family, cross-type. float/decimal ordering was a vm error before docs/types.md's exact
	// big.Rat resolution ("Resolved: question 1") — the round-trip check it replaced would have
	// accepted the pairing incorrectly anyway (see the doc's explanation of why it was dropped).
	// Ordering is exact now; arithmetic between the two stays a vm error, unchanged.
	matrixOrder(t, "int", core.IntValue(1), "float", core.FloatValue(2))
	matrixOrder(t, "int", core.IntValue(1), "decimal", dec(2))
	matrixOrder(t, "float", core.FloatValue(1), "decimal", dec(2))
	matrixOrder(t, "byte", core.ByteValue(1), "int", core.IntValue(2))
	matrixOrder(t, "rune", core.RuneValue('a'), "int", core.IntValue(int64('b')))
	matrixOrder(t, "byte", core.ByteValue('a'), "rune", core.RuneValue('b'))
	matrixOrder(t, "byte", core.ByteValue(1), "float", core.FloatValue(1.5))
	matrixOrder(t, "rune", core.RuneValue('a'), "float", core.FloatValue(float64('a')+0.5))
	matrixOrder(t, "byte", core.ByteValue(1), "decimal", dec(2))
	matrixOrder(t, "rune", core.RuneValue('a'), "decimal", dec(int64('a')+1))

	// bool against the rest of the numeric family was a real gap, caught empirically: bool had no
	// ordering relationship with anything but itself before this pass (its arithmetic is deferred
	// entirely, and ordering was never added alongside it) -- true < 5 was a vm error until this fix.
	matrixOrder(t, "bool", core.False, "int", core.IntValue(5))
	matrixOrder(t, "bool", core.False, "byte", core.ByteValue(5))
	matrixOrder(t, "bool", core.False, "rune", core.RuneValue('a'))
	matrixOrder(t, "bool", core.False, "float", core.FloatValue(0.5))
	matrixOrder(t, "bool", core.False, "decimal", dec(1))

	// sequence/text — ordering spans the whole rank chain bytes > runes > string
	matrixOrder(t, "bytes", byt("a"), "runes", run("b"))
	matrixOrder(t, "string", str("a"), "bytes", byt("b"))
	matrixOrder(t, "string", str("a"), "runes", run("b"))

	// The whole reason the round-trip check was dropped: exact comparison must not silently collapse
	// distinct values that happen to round-trip through float64 the same way. float64's literal 0.1
	// is actually very slightly larger than the exact decimal 0.1 (~0.1000000000000000055...), so an
	// exact comparison must say decimal("0.1") < float(0.1) -- not "equal" and not "close enough".
	matrixOK(t, "decimal(\"0.1\") < float(0.1) -- exact, not a round-trip-check false positive",
		core.NewDecimalValue(dec128.FromString("0.1")), token.Less, core.FloatValue(0.1), core.True)
	matrixOK(t, "float(0.1) > decimal(\"0.1\") -- same row, reversed",
		core.FloatValue(0.1), token.Greater, core.NewDecimalValue(dec128.FromString("0.1")), core.True)
	big := int64(1) << 53
	matrixOK(t, "int(2^53+1) <= float(2^53) is false -- no silent precision collapse",
		core.IntValue(big+1), token.LessEq, core.FloatValue(float64(big)), core.False)
	matrixOK(t, "float(2^53) < int(2^53+1) is true -- same row, reversed",
		core.FloatValue(float64(big)), token.Less, core.IntValue(big+1), core.True)

	arr := core.NewArrayValue([]core.Value{core.IntValue(1)}, false)
	matrixErr(t, "array < array -> vm error (no ordering pairing defined)", arr, token.Less, arr)
}

// ## Deliberate non-definitions — explicitly decided vm errors, not merely absent
func TestMatrix_DeliberateNonDefinitions(t *testing.T) {
	arr := core.NewArrayValue([]core.Value{core.IntValue(1)}, false)

	matrixErr(t, "int - rune -> vm error (position type, asymmetric)", core.IntValue(1), token.Sub, core.RuneValue('a'))
	matrixErr(t, "rune + rune -> vm error", core.RuneValue('a'), token.Add, core.RuneValue('b'))
	matrixErr(t, "scalar - array -> vm error", core.IntValue(1), token.Sub, arr)
	matrixErr(t, "array - array -> vm error (ambiguous)", arr, token.Sub, arr)
	matrixErr(t, "float + decimal -> vm error", core.FloatValue(1), token.Add, core.NewDecimalValue(dec128.FromInt64(1)))
	matrixErr(t, "bool + int -> vm error (arithmetic deferred entirely)", core.True, token.Add, core.IntValue(1))
	matrixErr(t, "array < array -> vm error", arr, token.Less, arr)
	matrixBothErr(t, "byte", core.ByteValue(1), token.Add, "float", core.FloatValue(1)) // no rule 1/2 pairing either way
	matrixErr(t, "byte & int -> vm error (bitwise same-type only)", core.ByteValue(1), token.And, core.IntValue(1))
	matrixErr(t, "int << byte -> vm error (count exception one-directional)", core.IntValue(1), token.Shl, core.ByteValue(1))
	matrixErr(t, "rune & rune -> vm error (rune has no bitwise)", core.RuneValue('a'), token.And, core.RuneValue('a'))
	matrixBothErr(t, "byte", core.ByteValue(1), token.And, "rune", core.RuneValue('a')) // rule 2 widening doesn't extend to bitwise
	matrixUnaryErr(t, "unary - rune -> vm error", token.Sub, core.RuneValue('a'))
	matrixUnaryErr(t, "unary - bool -> vm error", token.Sub, core.True)
}
