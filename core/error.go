package core

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/binary"
)

const errorTypeName = "error"
const KindUser = "user"

type Error struct {
	Payload  Value
	Kind     string
	Category errs.Category
	Fatal    bool
}

func (e *Error) Set(payload Value, kind string, category errs.Category, fatal bool) {
	e.Payload = payload
	e.Kind = kind
	e.Category = category
	e.Fatal = fatal
}

func NewErrorValue(payload Value, kind string, category errs.Category, fatal bool) Value {
	return Value{
		Type:      value.Error,
		Immutable: true,
		Ptr:       unsafe.Pointer(&Error{Payload: payload, Kind: kind, Category: category, Fatal: fatal}),
	}
}

func NewRuntimeErrorValue(kind string, category errs.Category, fatal bool, message string) Value {
	return Value{
		Type:      value.Error,
		Immutable: true,
		Ptr:       unsafe.Pointer(&Error{Payload: NewStringValue(message), Kind: kind, Category: category, Fatal: fatal}),
	}
}

// A failure as a first-class value.
//
// ## Overview
//
// An `error` is a value that carries a payload describing a failure. Errors reach a script two ways:
//
// - **constructed** — `error(payload)` wraps any value; the script raises it itself with `raise(...)`;
// - **raised by the runtime** — division by zero, a bad index, a failed conversion, … — and caught with
// `recover()` inside a deferred function.
//
// Every error is **truthy** — an error without a payload is still an error, and there is no zero error
// (`error()` with no arguments is itself a runtime error: `(error) expected (payload[, fatal]), got 0 arguments`).
// Each error carries a machine-readable `kind()` tag, a `value()` payload, and a **category** saying where it came
// from, read with `is_runtime()` / `is_user()` / `is_requirement()`.
//
// ### Categories
//
// Every kind belongs to exactly one category — the error's origin, an axis independent of what went wrong:
//
// | category | predicate | raised by | reaches a script? |
// | --- | --- | --- | --- |
// | runtime | `is_runtime()` | a builtin, a member function or a stdlib module | yes |
// | user | `is_user()` | the script's own `error(...)` / `raise(...)` | yes |
// | requirement | `is_requirement()` | the script's own `require(cond, payload)` | yes |
// | system | — | the VM itself: stack overflow, an internal invariant, a host defect | **never** |
//
// A system error is always fatal, so it bypasses every `recover()` and only ever reaches the host — which is why
// there is no `is_system()` to call. The host reads the category from `RuntimeError.Category` (see
// [embedding](../../embedding.md#handling-errors)).
//
// ## Construction
//
// ```go
// error("boom")               // payload "boom", kind "user"
// error(42)                   // any payload type
// error({code: 404})          // including containers
// error(undefined)            // legal: the payload-less error — still truthy
// error(error("inner"))       // => error("inner") — an error argument is never re-wrapped
// error("boom", true)         // second argument: fatal flag — see below
// ```
//
// `error(x)` **wraps** its argument — it is a constructor, not a conversion, so there is no `.error()` member
// on other types and no conversion *from* `error` back into a payload type.
//
// **An argument that is already an error is answered as-is**, never wrapped a second time: `error(err)` is
// `err`. A bare re-wrap only ever added an unlabelled layer, and it relabelled the payload's own diagnosis —
// a caught runtime error came back out as kind `"user"` in the user category. The severity form keeps the
// payload and kind and changes only the flag, which is exactly what `raise(err, fatal)` already did:
//
// ```go
// e = error("boom")
// error(e) == e               // => true — the same error, not a nested one
// error(e).value()            // => "boom" — a string, not an error
// error(e, true)              // fatal; kind() and value() are still "user"/"boom"
//
// // a runtime error keeps its own diagnosis through the constructor
// c = func() r { defer func() { r = recover() }(); (1).nope() }()   // a caught invalid_method error
// error(c).kind()             // => "invalid_method" — not relabelled to "user"
// error(c).is_runtime()       // => true — the category travels with the error too
// ```
//
// Deliberate nesting is spelled with a payload that **names** the cause, which is the annotated chain a bare
// re-wrap could never express:
//
// ```go
// e = error("config file not found")
// error({msg: "loading config failed", cause: e})   // the chain, labelled
// ```
//
// The optional second argument is a `bool` fatal flag. A fatal error is **not catchable**: `raise(error("x",
// true))` unwinds past every `recover()` and terminates the script. Use it for states no handler should paper
// over.
//
// ## Raising and catching
//
// `raise(x)` raises. A non-error argument is wrapped first — `raise("text")` and `raise(42)` raise a kind
// `"user"` error with that payload. `recover()`, called inside a deferred function, answers the in-flight error
// and stops the unwinding; with nothing in flight it answers `undefined`.
//
// ```go
//
//	io := import("io")
//
//	safe_div := func(a, b) res {
//	    defer func() {
//	        e := recover()
//	        if e != undefined {
//	            io.print("caught: ", e.kind())   // "division_by_zero"
//	            res = 0
//	        }
//	    }()
//	    res = a / b
//	}
//
//	safe_div(10, 2)   // => 5
//	safe_div(10, 0)   // prints "caught: division_by_zero", answers 0
//
// ```
//
// Note the **named result** (`res`): a plain local assigned inside the deferred function would be lost — the
// named result is how a handler substitutes a value. In a script's own body, which has no named result, a
// top-level `defer` assigns a variable instead (see [`defer` at the top
// level](../../language.md#defer-at-the-top-level-of-a-script)).
//
// ### `require(cond, payload)`
//
// The input check a script opens with. When `cond` is true it answers `undefined` and the script carries on;
// otherwise it raises a recoverable error of kind `requirement` carrying `payload` **untouched** — so a
// structured rejection reaches the host as structure, not as a sentence.
//
// ```go
// input := {amount: 5d}                 // the host's binding
// require(is_decimal(input.amount), {field: "amount", reason: "not a number"})
// require(input.amount > 0d,        {field: "amount", reason: "must be positive"})
// // the happy path follows
// ```
//
// Truthiness is the same `is_true()` uses. Exactly two arguments; anything else is `wrong_num_arguments`.
//
// ### Selective re-raise
//
// Handle the kinds you expect; re-raise everything else unchanged (`raise(e)` keeps the original kind and
// payload):
//
// ```go
//
//	defer func() {
//	    e := recover()
//	    if e == undefined {
//	        return                          // nothing in flight
//	    }
//	    if e.kind() == "division_by_zero" {
//	        // handle this one
//	    } else {
//	        raise(e)                        // not ours — let it keep unwinding
//	    }
//	}()
//
// ```
//
// A `division_by_zero` handled this way answers the substitute; an `index_out_of_bounds` raised in the same
// position passes through to the outer handler with its kind intact.
//
// ## Operators
//
// Errors **propagate by raising**: an error is not a number, a string, or a flag, so combining it with anything
// raises `invalid_binary_operator` — whichever side it stands on. Only equality is defined, and it compares
// **payloads** (the kind and flags do not participate):
//
// ```go
// error("boom") == error("boom")   // => true — equal payloads
// error("boom") == error("bam")    // => false
// error(1) == error(1)             // => true
// error(1) == 1                    // => false — an error never equals a non-error
// error("a") != error("b")         // => true
//
// error("x") + 1                   // Error: invalid_binary_operator: error + int
// 1 + error("x")                   // Error: invalid_binary_operator: int + error
// error("x") + error("y")          // Error: invalid_binary_operator: error + error
// error(1) < error(2)              // Error: invalid_binary_operator — no ordering
// !error("x")                      // => false — every error is truthy
// ```
//
// ## Every error kind
//
// Each kind has one severity and one [category](#categories), always. **Recoverable** kinds can reach a script's
// `recover()`; **system** kinds are fatal and only ever reach the host.
//
// ### Script and script-data kinds — recoverable, category *runtime*
//
// | kind | raised by | example trigger |
// | --- | --- | --- |
// | `division_by_zero` | `/` or `%` by zero, on `int` and `decimal` | `1 / 0`, `1d / 0d` |
// | `invalid_argument_type` | an argument outside a member's accepted readings | `[1, 2].join(1.5)` |
// | `invalid_value` | a value of the right type outside the valid domain — overflow, a declared limit, an empty sequence | `range(1, 4, 0)`; `"ab".repeat(1e12)`; `[].first()` |
// | `wrong_num_arguments` | wrong argument count | `"a".upper(1)`, `[1, 2].chunk()` |
// | `index_out_of_bounds` | index outside the sequence | `[1, 2][5]` |
// | `invalid_index_type` | index of the wrong type | `[1, 2]["a"]` |
// | `invalid_selector` | a selector a type does not answer | `{a: 1}.b.c` |
// | `invalid_unpack_type` | destructuring a value that cannot be unpacked | `a, b := 5` |
// | `not_mutable` | any mutating member on a frozen receiver | `freeze([1, 2]).push_in_place(3)` |
// | `not_assignable` | an assignment statement into a frozen container | `a := freeze([1, 2]); a[0] = 9` |
// | `not_accessible` / `not_appendable` / `not_deletable` / `not_iterable` / `not_sliceable` | an operation the type does not support | `range(1, 10)[1:3]` |
// | `invalid_method` | calling a member the type does not have | `(5).len()` |
// | `invalid_binary_operator` / `invalid_unary_operator` | an operator pair with no meaning | `error("x") + 1`; `-[1]` |
// | `not_callable` | calling a non-callable value | `x := 5; x()` |
// | `not_implemented` | a reading deferred to a type that does not exist yet | `range(1, 5).contains(1..3)` |
// | `conversion` | a failed conversion or parse | `"abc".int()`, `decimal("abc")`, `(1e39).decimal()` |
// | `unsupported_format_spec` | a malformed or unsupported format spec | `(5).format("{:,}")`, `format("{0:", [1])` |
// | `formatting` | a value that cannot be rendered at all | — |
// | `undefined_variable` | reading a name the runtime cannot resolve | — |
// | `module_not_found` | `import` of a name that is not registered | `import("nope")` |
// | `module_not_allowed` | bytecode importing a builtin module the host did not grant (checked by the VM before the first instruction; never reaches `recover()`) | `import("os")` without a grant |
// | `json_encoding` | a value with no JSON representation | `json.encode(func(){})` |
// | `json_decoding` | malformed JSON input | `json.decode("{")` |
// | `binary_encoding` | a value that cannot be binary-encoded, or corrupt input | — |
// | `io` | the world refused: a file, a directory, an environment variable, a process | `fs.remove("/nope")` |
//
// ### Script-authored kinds — recoverable
//
// | kind | category | raised by | example trigger |
// | --- | --- | --- | --- |
// | `user` | user | `error(...)` construction, `raise(x)` | `raise("text")`, `raise({code: 42})` |
// | `requirement` | requirement | `require(cond, payload)` with a falsy condition | `require(false, "bad input")` |
//
// ### System kinds — **fatal**, category *system*
//
// These bypass every `recover()`, skip every remaining `defer`, and reach the host only. A script can never hold
// one, which is why no `is_system()` predicate exists.
//
// | kind | means |
// | --- | --- |
// | `stack_overflow` | the call-frame or operand stack ran out — usually unbounded recursion |
// | `internal` | a VM invariant was violated, or a Go panic was contained at the run boundary |
// | `host` | a host setup mistake: `SetValueType` on a builtin slot, an unregistered module ID |
//
// `not_mutable` is uniform: **every** mutating member on an immutable receiver raises it, whatever the verb —
// and before it looks at its arguments, so `[1].freeze().sort_in_place(1)` raises `not_mutable`, not
// `wrong_num_arguments`.
// `not_assignable` is the assignment statement's own kind.
//
// ## Telling apart the kinds of wrong
//
// A kind says what failed, not **whose fault it was** — and it cannot: `int(x)` does not know where `x` came
// from. Three kinds carry that signal:
//
// | signal | read it as |
// | --- | --- |
// | `requirement` | the script's own input check rejected the caller's data. The payload says which field and why. |
// | `conversion`, `json_decoding`, `io` | the data or the world did not cooperate — a bad parse, malformed input, a missing file |
// | anything else | most likely a defect in the script itself |
//
// A host that wants to distinguish "the caller sent bad data" from "the rules have a bug" should branch on that,
// not on the message text. See [embedding](../../embedding.md#handling-errors).
//
//kavun:type error

var TypeError = ValueTypeDescr{
	Name:         ConstHook(errorTypeName),                            // PURE by contract
	String:       errorTypeString,                                     // PURE by contract
	Format:       errorTypeFormat,                                     // PURE by contract
	Interface:    func(v Value) any { return errors.New(v.String()) }, // PURE by contract
	EncodeJSON:   errorTypeEncodeJSON,                                 // PURE by contract
	EncodeBinary: errorTypeEncodeBinary,                               // PURE by contract
	DecodeBinary: errorTypeDecodeBinary,                               // IMPURE by contract (mutates target)
	IsTrue:       Const2Hook[bool, error](true, nil),                  // PURE by contract
	Copy:         errorTypeCopy,                                       // PURE by contract
	Equal:        errorTypeEqual,                                      // PURE by contract
	UnaryOp:      errorTypeUnaryOp,                                    // PURE by contract
	BinaryOp:     errorTypeBinaryOp,                                   // PURE by contract
	AsString:     errorTypeAsString,                                   // PURE by contract
	AsBool:       Const2Hook(true, true),                              // PURE by contract

	Methods: errorMethods,
}

func errorTypeEncodeJSON(v Value) ([]byte, error) {
	// the payload's render (total: a non-string payload renders as its string form), encoded exactly as a string
	// is — JSON escaping, and the same refusal of text holding octets that are not symbols
	s, _ := errorTypeAsString(v)
	b, err := stringTypeEncodeJSON(NewStringValue(s))
	if err != nil {
		return nil, err
	}
	return append(append([]byte(`{"error":`), b...), '}'), nil
}

func errorTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Error)(v.Ptr)
	pb, err := o.Payload.EncodeBinary()
	if err != nil {
		return nil, fmt.Errorf("error (payload): %w", err)
	}

	b := binary.AppendBytes(nil, []byte(o.Kind))
	b = append(b, byte(o.Category))
	if o.Fatal {
		b = append(b, byte(1))
	} else {
		b = append(b, byte(0))
	}
	b = binary.AppendBytes(b, pb)
	return b, nil
}

func errorTypeDecodeBinary(v *Value, data []byte) error {
	offset := 0
	kb, err := binary.ReadBytes(data, &offset, "error (kind)")
	if err != nil {
		return err
	}
	if len(data)-offset < 2 {
		return fmt.Errorf("error (category, fatal): expected 2 bytes, got %d", len(data)-offset)
	}
	category := errs.Category(data[offset])
	offset++
	fatal := data[offset] != 0
	offset++

	pb, err := binary.ReadBytes(data, &offset, "error (payload)")
	if err != nil {
		return err
	}
	var payload Value
	if err := payload.DecodeBinary(pb); err != nil {
		return fmt.Errorf("error (payload): %w", err)
	}
	if offset != len(data) {
		return fmt.Errorf("error: trailing %d bytes", len(data)-offset)
	}

	*v = NewErrorValue(payload, string(kb), category, fatal)
	return nil
}

func errorTypeString(v Value) string {
	o := (*Error)(v.Ptr)
	if o.Payload.Type == value.Undefined {
		return "error()"
	}
	return fmt.Sprintf("error(%s)", o.Payload.String())
}

func errorTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	switch sp.Verb {
	case 0:
		s, _ := errorTypeAsString(v) // total: a non-string payload renders as its string form
		return fspec.ApplyGenerics(s, sp, fspec.AlignLeft), nil

	case 'v':
		return errorTypeString(v), nil

	case 'T':
		return fspec.ApplyGenerics(errorTypeName, sp, fspec.AlignLeft), nil

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
}

// deep=true recursively copies the payload (today's copy() semantics); deep=false only allocates a new Error
// wrapper, leaving the payload sharing the source (copy_shallow()).
func errorTypeCopy(v Value, deep bool) (Value, error) {
	o := (*Error)(v.Ptr)
	if !deep {
		return NewErrorValue(o.Payload, o.Kind, o.Category, o.Fatal), nil
	}
	pl, err := o.Payload.Copy(true)
	if err != nil {
		return Undefined, err
	}
	return NewErrorValue(pl, o.Kind, o.Category, o.Fatal), nil
}

func errorTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Error:
		o := (*Error)(v.Ptr)
		x := (*Error)(other.Ptr)
		return o.Payload.Equal(x.Payload)
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func errorTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}
	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract.
// error has no unary operations.
func errorTypeUnaryOp(v Value, op token.Token) (Value, error) {
	return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
}

func errorTypeAsString(v Value) (string, bool) {
	o := (*Error)(v.Ptr)
	if s, ok := o.Payload.AsString(); ok {
		return s, true
	}
	return o.Payload.String(), true
}
