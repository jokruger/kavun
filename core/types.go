package core

import (
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jokruger/dec128"
	"github.com/jokruger/fin128/civil"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

type VM interface {
	Abort()                             // aborts execution of the current script
	IsStackEmpty() bool                 // returns true if there are no frames on the call stack
	Call(Value, []Value) (Value, error) // calls a compiled function
	Run() error                         // runs the VM until completion
	Recover() Value                     // returns the in-flight error if in "deferred-for" frame
}

type NativeFunc = func(VM, []Value) (Value, error)
type Pos int

func (p Pos) IsValid() bool {
	return p != NoPos
}

const (
	// Pos constants
	NoPos Pos = 0

	// Builtin module/function slot constants
	ModuleSlotSize = 128
	MaxModules     = 32
)

// Builtin module/function registry
var BuiltinFunctions [MaxModules * ModuleSlotSize]*BuiltinFunction

// Primitive value (used in static storage)
type Primitive struct {
	Type uint8
	Data uint64
}

func (p Primitive) Value() Value {
	return Value{Type: p.Type, Immutable: true, Data: p.Data}
}

// Static variables
type Static struct {
	Primitives        []Primitive
	Decimals          []dec128.Dec128
	Strings           []string
	StringLens        []int64 // rune count per static string — always len(Strings) entries, see BuildStringLens
	Runes             []Runes
	Bytes             []Bytes
	Times             []time.Time
	FormatSpecs       []FormatSpec
	CompiledFunctions []CompiledFunction
	NameLists         [][]string
	Ranges            []IntRange
}

// BuildStringLens populates StringLens with the rune count of each static string. It must run at every point a
// Static is finalized (compiler build, bytecode decode) so the VM's LoadStaticString never recounts on the hot
// path. Idempotent.
func (s *Static) BuildStringLens() {
	if len(s.StringLens) == len(s.Strings) {
		return
	}
	lens := make([]int64, len(s.Strings))
	for i := range s.Strings {
		lens[i] = int64(utf8.RuneCountInString(s.Strings[i]))
	}
	s.StringLens = lens
}

// ValueTypeDescr is a Kavun data type descriptor structure.
// See docs/purity.md for purity contract.
type ValueTypeDescr struct {
	Name         func(v Value) string                                                      // PURE by contract
	String       func(v Value) string                                                      // PURE by contract
	Format       func(v Value, sp fspec.FormatSpec) (string, error)                        // PURE by contract
	Interface    func(v Value) any                                                         // PURE by contract
	EncodeJSON   func(v Value) ([]byte, error)                                             // PURE by contract
	EncodeBinary func(v Value) ([]byte, error)                                             // PURE by contract
	DecodeBinary func(v *Value, data []byte) error                                         // IMPURE by contract (mutates target)
	IsTrue       func(v Value) (bool, error)                                               // PURE by contract
	Copy         func(v Value, deep bool) (Value, error)                                   // PURE by contract: deep=true recursively copies nested Values (copy()); deep=false copies only the top-level container/wrapper, sharing nested structure (copy_shallow())
	Equal        func(v Value, other Value, final bool) bool                               // PURE by contract
	UnaryOp      func(v Value, op token.Token) (Value, error)                              // PURE by contract
	BinaryOp     func(v Value, other Value, op token.Token, reflected bool) (Value, error) // PURE by contract

	IsIterable func(v Value) bool                                         // PURE by contract
	Iterator   func(v Value) (Value, error)                               // PURE by contract (constructs fresh iterator)
	Len        func(v Value) int64                                        // PURE by contract
	Contains   func(v Value, e Value) (bool, error)                       // PURE by contract — the `in` operator: contains' VALUE readings (element | run | family), raising on an unacceptable operand; a callable raises, an operator operand is always a value
	Append     func(v Value, args []Value, mutate bool) (Value, error)    // MUTATE-DEPENDENT by contract: mutate=true mutates the receiver in place (append_in_place()); mutate=false returns an independent value with the items appended (append())
	Delete     func(v Value, key Value, mutate bool) (Value, error)       // MUTATE-DEPENDENT by contract: mutate=true mutates the receiver in place (delete_in_place()); mutate=false returns an independent container without the key (delete())
	Slice      func(v Value, s Value, e Value) (Value, error)             // PURE by contract
	SliceStep  func(v Value, s Value, e Value, step Value) (Value, error) // PURE by contract

	IsCallable func(v Value) bool                                // PURE by contract
	IsVariadic func(v Value) bool                                // PURE by contract
	Arity      func(v Value) int                                 // PURE by contract
	Call       func(vm VM, v Value, args []Value) (Value, error) // CALLABLE-DEPENDENT by contract

	AccessIndex         func(v Value, k Value) (Value, error)            // PURE by contract — x[k]
	AccessNamedProperty func(vm VM, v Value, name string) (Value, error) // PURE by contract — x.name

	AssignIndex         func(v Value, k Value, r Value) error            // IMPURE by contract (mutates target) — x[k] = r
	AssignNamedProperty func(vm VM, v Value, name string, r Value) error // IMPURE by contract (mutates target) — x.name = r

	CallNamedMethod   func(vm VM, v Value, name string, args []Value) (Value, error) // METHOD-DEPENDENT by contract: purity varies per member name, reported by IsNamedMethodPure (see docs/purity.md)
	IsNamedMethodPure func(name string) bool                                         // optimizer information only: true only for a name the type answers on the name path and whose call is pure

	Next  func(v Value) bool           // LOCALISED-STATE by contract (advances iterator cursor)
	Key   func(v Value) (Value, error) // LOCALISED-STATE by contract (reads iterator cursor)
	Value func(v Value) (Value, error) // LOCALISED-STATE by contract (reads iterator cursor)
	Elem  func(v Value) (Value, error) // LOCALISED-STATE by contract: the single-variable for-in binding — the container's ELEMENT. Defaults to the Value hook; a map iterator answers the KEY, because a map's element is its key

	AsBool     func(v Value) (bool, bool)             // PURE by contract
	AsByte     func(v Value) (byte, bool)             // PURE by contract
	AsRune     func(v Value) (rune, bool)             // PURE by contract
	AsInt      func(v Value) (int64, bool)            // PURE by contract
	AsFloat    func(v Value) (float64, bool)          // PURE by contract
	AsDecimal  func(v Value) (dec128.Dec128, bool)    // PURE by contract
	AsDate     func(v Value) (civil.Date, bool)       // PURE by contract
	AsTime     func(v Value) (time.Time, bool)        // PURE by contract
	AsBytes    func(v Value) ([]byte, bool)           // PURE by contract
	AsString   func(v Value) (string, bool)           // PURE by contract
	AsRunes    func(v Value) ([]rune, bool)           // PURE by contract
	AsArray    func(v Value) ([]Value, bool)          // PURE by contract
	AsDict     func(v Value) (map[string]Value, bool) // PURE by contract
	AsIntRange func(v Value) (IntRange, bool)         // PURE by contract

	// Member functions and properties by member ID (core/member). Each is a keyed slice literal — the literal is
	// the table — and a zero entry (nil Fn/Get/Set) is the miss that falls through to the named hook.
	Methods    []MethodEntry   // x.name(args): answered by Methods[id].Fn, else CallNamedMethod
	Properties []PropertyEntry // x.name and x.name = r: answered by Properties[id].Get/Set, else the named hooks
}

// MethodEntry is one slot of a type's Methods table.
type MethodEntry struct {
	Fn   func(vm VM, v Value, id member.ID, args []Value) (Value, error) // validates its own argument count
	Pure bool                                                            // optimizer information only (docs/purity.md); never read on the call path
}

// PropertyEntry is one slot of a type's Properties table.
type PropertyEntry struct {
	Get  func(vm VM, v Value, id member.ID) (Value, error)
	Set  func(vm VM, v Value, id member.ID, r Value) error // nil: assignment falls through to AssignNamedProperty
	Pure bool                                              // purity of Get; Set is impure by definition
}

// DefaultValueType provides default implementations for all ValueType hooks.
var DefaultValueType = ValueTypeDescr{
	Name:         func(v Value) string { return fmt.Sprintf("<unknown:%d>", v.Type) },                     // PURE by contract
	String:       func(v Value) string { return v.TypeName() },                                            // PURE by contract
	Format:       defaultFormat,                                                                           // PURE by contract
	Interface:    func(_ Value) any { return nil },                                                        // PURE by contract
	EncodeJSON:   func(v Value) ([]byte, error) { return nil, errs.NewNoJSONEncodingError(v.TypeName()) }, // PURE by contract
	EncodeBinary: func(v Value) ([]byte, error) { return nil, errs.NewBinaryEncodingError(v.TypeName()) }, // PURE by contract
	DecodeBinary: func(v *Value, _ []byte) error { return errs.NewBinaryEncodingError(v.TypeName()) },     // IMPURE by contract (mutates target)
	IsTrue:       Const2Hook[bool, error](false, nil),                                                     // PURE by contract
	Copy:         func(v Value, _ bool) (Value, error) { return v, nil },                                  // PURE by contract
	Equal:        defaultEqual,                                                                            // PURE by contract
	UnaryOp:      defaultUnaryOp,                                                                          // PURE by contract
	BinaryOp:     defaultBinaryOp,                                                                         // PURE by contract

	IsIterable: ConstHook(false),          // PURE by contract
	Iterator:   ValueHook(Undefined, nil), // PURE by contract (constructs fresh iterator)
	Len:        ConstHook(int64(0)),       // PURE by contract

	Contains: func(v Value, e Value) (bool, error) { // PURE by contract — a type with no membership raises, never a silent false
		return false, errs.NewInvalidBinaryOperatorError("in", e.TypeName(), v.TypeName())
	},

	Append:    defaultAppend,    // MUTATE-DEPENDENT by contract (see ValueTypeDescr.Append)
	Delete:    defaultDelete,    // IMPURE by contract
	Slice:     defaultSlice,     // PURE by contract
	SliceStep: defaultSliceStep, // PURE by contract

	IsCallable: ConstHook(false), // PURE by contract
	IsVariadic: ConstHook(false), // PURE by contract

	Arity: ConstHook(0), // PURE by contract
	Call:  defaultCall,  // CALLABLE-DEPENDENT by contract

	AccessIndex:         defaultAccessIndex,         // PURE by contract
	AccessNamedProperty: defaultAccessNamedProperty, // PURE by contract

	AssignIndex:         func(v Value, _, _ Value) error { return errs.NewNotAssignableError(v.TypeName()) }, // IMPURE by contract
	AssignNamedProperty: defaultAssignNamedProperty,                                                          // IMPURE by contract

	CallNamedMethod:   CallMemberByLookup,                 // METHOD-DEPENDENT by contract: purity varies per member name, reported by IsNamedMethodPure (see docs/purity.md)
	IsNamedMethodPure: func(string) bool { return false }, // conservative: a type opts in per name before the optimizer will fold a call to it

	Next:  ConstHook(false),                                                    // LOCALISED-STATE by contract (advances iterator cursor)
	Key:   ValueHook(Undefined, nil),                                           // LOCALISED-STATE by contract (reads iterator cursor)
	Value: ValueHook(Undefined, nil),                                           // LOCALISED-STATE by contract (reads iterator cursor)
	Elem:  func(v Value) (Value, error) { return ValueTypes[v.Type].Value(v) }, // LOCALISED-STATE by contract: defaults to the type's own Value hook

	AsBool:     Const2Hook(false, false),                                   // PURE by contract
	AsByte:     Const2Hook(byte(0), false),                                 // PURE by contract
	AsRune:     Const2Hook(rune(0), false),                                 // PURE by contract
	AsInt:      Const2Hook(int64(0), false),                                // PURE by contract
	AsFloat:    Const2Hook(float64(0), false),                              // PURE by contract
	AsDecimal:  Const2Hook(dec128.Decimal0, false),                         // PURE by contract
	AsDate:     Const2Hook(civil.Date{}, false),                            // PURE by contract
	AsTime:     Const2Hook(time.Time{}, false),                             // PURE by contract
	AsBytes:    Const2Hook[[]byte](nil, false),                             // PURE by contract
	AsString:   Const2Hook("", false),                                      // PURE by contract
	AsRunes:    defaultAsRunes,                                             // PURE by contract
	AsArray:    func(Value) ([]Value, bool) { return nil, false },          // PURE by contract
	AsDict:     func(Value) (map[string]Value, bool) { return nil, false }, // PURE by contract
	AsIntRange: Const2Hook(IntRange{}, false),                              // PURE by contract
}

// ValueTypes is the global registry of value type descriptors, indexed by type ID.
var ValueTypes [256]ValueTypeDescr

// SetValueType registers a user-defined value type descriptor for the given type ID.
func SetValueType(t uint8, f ValueTypeDescr) error {
	if t < value.FirstUserDefinedType {
		// A host setup mistake, not a script fault: fatal, kind "host".
		return errs.NewHostError(fmt.Sprintf("(SetValueType) cannot set value type for built-in type %d", t))
	}
	if err := validateTables(&f); err != nil {
		return errs.NewHostError(fmt.Sprintf("(SetValueType) type %d: %v", t, err))
	}
	setValueType(t, f)
	return nil
}

// setValueType registers a builtin type: a table violation is a defect in core, found at init.
func setValueType(t uint8, f ValueTypeDescr) {
	if err := validateTables(&f); err != nil {
		panic(fmt.Sprintf("(setValueType) type %d: %v", t, err))
	}
	fv := reflect.ValueOf(&f).Elem()
	dv := reflect.ValueOf(DefaultValueType)

	for i := 0; i < fv.NumField(); i++ {
		field := fv.Field(i)
		if field.IsNil() {
			field.Set(dv.Field(i))
		}
	}

	ValueTypes[t] = f
}

// validateTables checks a descriptor's Methods and Properties: within member.Max, every filled slot at a bound
// member ID, and no `_in_place` member declared pure (docs/conventions.md: `_in_place` mutates the receiver).
func validateTables(f *ValueTypeDescr) error {
	if len(f.Methods) > int(member.Max) {
		return fmt.Errorf("Methods table too long: %d (max: %d)", len(f.Methods), member.Max)
	}
	if len(f.Properties) > int(member.Max) {
		return fmt.Errorf("Properties table too long: %d (max: %d)", len(f.Properties), member.Max)
	}
	for i, e := range f.Methods {
		if e.Fn == nil {
			continue
		}
		name := member.ID(i).String()
		if name == "" {
			return fmt.Errorf("Methods[%d] is set but member id %d is not bound", i, i)
		}
		if e.Pure && strings.HasSuffix(name, "_in_place") {
			return fmt.Errorf("Methods[%d] (%s) is declared pure but an _in_place member mutates its receiver", i, name)
		}
	}
	for i, e := range f.Properties {
		if e.Get == nil && e.Set == nil {
			continue
		}
		if member.ID(i).String() == "" {
			return fmt.Errorf("Properties[%d] is set but member id %d is not bound", i, i)
		}
	}
	return nil
}

// MemberIsPure answers whether a call to member name on a value of type t is pure (docs/purity.md): the bound
// Methods slot's Pure flag when the type tables the name, else the type's IsNamedMethodPure. Optimizer only.
func MemberIsPure(t uint8, name string) bool {
	d := &ValueTypes[t]
	if id := member.Lookup(name); int(id) < len(d.Methods) && d.Methods[id].Fn != nil {
		return d.Methods[id].Pure
	}
	return d.IsNamedMethodPure(name)
}

// HasMember answers whether type t tables member function name. A name answered only through CallNamedMethod is
// not reported: the name path is open-ended, so its roster cannot be enumerated.
func HasMember(t uint8, name string) bool {
	d := &ValueTypes[t]
	id := member.Lookup(name)
	return id != member.Unknown && int(id) < len(d.Methods) && d.Methods[id].Fn != nil
}
