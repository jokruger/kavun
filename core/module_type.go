package core

import (
	"fmt"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// A builtin module, as `import` answers it.
//
// ## Overview
//
// `import("math")` answers a value of type `module`: the module's functions and constants are its **members**, it
// is read-only, and it has no content of its own to convert. Every builtin module — the standard library ones and
// any an embedder registers — is its own type with this one Kavun type name, so scripts see one shape:
//
// ```go
// math := import("math")
// type_name(math)        // => "module"
// math.cbrt(27)          // => 3.0 — a member call
// math.pi                // => 3.141592653589793 — a constant member
// [8, 27].map(math.cbrt) // => [2.0, 3.0] — a function member read as a value
// format(math)           // `import("math")` — it renders as the expression that answers it
// ```
//
// A source module (`import("./helpers")`, or one added with `Script.AddCustomModule`) is not a `module` value: it
// answers whatever it `export`s, frozen — a record keeps record rules. See
// [language.md § Modules](../../language.md#modules).
//
// Which builtin modules a script may import is the host's decision. By default only the **pure** ones (they only
// compute) are importable; a world-touching module (`clock`, `exec`, `fs`, `io`, `os`) needs a grant, checked at
// compile time and again by the VM before the bytecode runs (`module_not_allowed`). See [embedding.md § The module
// grant](../../embedding.md#the-module-grant).
//
// ## Operators
//
// | operator | reading | example |
// | --- | --- | --- |
// | `==` / `!=` | the same module (for a builtin module: the same import) | `import("math") == import("math")` → `true` |
// | `in` | a member name | `"pi" in math` → `true`; `1 in math` → `false` |
// | `!` | truthiness — a module is always truthy | `!math` → `false` |
// | everything else | raises `invalid_binary_operator` / `invalid_unary_operator` | `math + 1`, `math < math`, `-math` |
//
// A module is not indexable (`math["pi"]` raises `not_accessible` — members are dot access only), not sliceable and
// not callable.
//
// ## Members
//
// A module answers its own functions and constants, plus the four universal members. It has **no** `string()`:
// a module has no text content, so `string(math)` raises `conversion` and `math.string()` raises like any missing
// member.
//
// | member | answers |
// | --- | --- |
// | its functions | `m.f(args)` calls; `m.f` is the `builtin_function` value — the same value every time (`math.cbrt == math.cbrt`) |
// | its constants | the constant's value |
// | `is_true()` | `true` |
// | `format([spec])` | the render, `import("math")`, with the generic spec (`math.format(">20")`); `{:T}` → `module` |
// | `copy()` / `freeze()` | the module itself |
//
// A name the module does not have raises, naming the module — as a read and as a call:
//
// ```go
// math := import("math")
// math.pii       // Error: invalid_selector: module 'math' has no member 'pii'
// math.pii()     // Error: invalid_method: module 'math' has no member 'pii'
// math.pi = 1    // Error: not_assignable: module 'math' is read-only: cannot assign member 'pi'
// ```
//
// ## Size and iteration
//
// `len(m)` counts its functions and constants; `for k, v in m` visits them sorted by name, with functions as
// `builtin_function` values:
//
// ```go
// h := import("hex")
// len(h)                                  // => 2
// names := []
// for k, _ in h { names.push_in_place(k) }
// names                                   // => ["decode", "encode"]
// ```
//
//kavun:type module

// moduleDescr is the descriptor every module type starts from; ModuleBuilder adds its Methods and Properties. The
// hooks reach the module through moduleDefs[v.Type], never through Ptr, so an embedder factory may use Ptr.
func moduleDescr() ValueTypeDescr {
	d := DefaultValueType
	d.Name = ConstHook("module")                                    // PURE by contract
	d.String = moduleTypeString                                     // PURE by contract
	d.Format = moduleTypeFormat                                     // PURE by contract
	d.EncodeBinary = codeEncodeBinary                               // PURE by contract (refuses: code, not data)
	d.DecodeBinary = codeDecodeBinary                               // IMPURE by contract (refuses: code, not data)
	d.IsTrue = Const2Hook[bool, error](true, nil)                   // PURE by contract
	d.Equal = moduleTypeEqual                                       // PURE by contract
	d.Copy = func(v Value, _ bool) (Value, error) { return v, nil } // PURE by contract
	d.IsIterable = ConstHook(true)                                  // PURE by contract
	d.Iterator = moduleTypeIterator                                 // PURE by contract (constructs fresh iterator)
	d.Len = moduleTypeLen                                           // PURE by contract
	d.Contains = moduleTypeContains                                 // PURE by contract
	d.AccessNamedProperty = moduleTypeAccessNamedProperty           // PURE by contract
	d.AssignNamedProperty = moduleTypeAssignNamedProperty           // IMPURE by contract (refuses: a module is read-only)
	d.AssignIndex = moduleTypeAssignIndex                           // IMPURE by contract (refuses: a module is read-only)
	d.CallNamedMethod = moduleTypeCallNamedMethod                   // METHOD-DEPENDENT by contract
	d.IsNamedMethodPure = func(string) bool { return false }        // every member is in the table
	return d
}

func moduleName(v Value) string {
	if def := moduleDefs[v.Type]; def != nil {
		return def.Name
	}
	return fmt.Sprintf("<module:%d>", v.Type)
}

// A module renders as the expression that answers it: import("math").
func moduleTypeString(v Value) string {
	return fmt.Sprintf("import(%q)", moduleName(v))
}

func moduleTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics("module", sp, fspec.AlignLeft), nil
	}
	return fspec.ApplyGenerics(ValueTypes[v.Type].String(v), sp, fspec.AlignLeft), nil // an embedder may amend String
}

func moduleTypeEqual(v Value, other Value, final bool) bool {
	if other.Type == v.Type {
		return other.Ptr == v.Ptr
	}
	if final {
		return false
	}
	return ValueTypes[other.Type].Equal(other, v, true)
}

func moduleRoster(v Value) Value {
	return moduleDefs[v.Type].roster
}

// for k, v in math: the members sorted by name, as the roster record iterates.
func moduleTypeIterator(v Value) (Value, error) {
	return recordTypeIterator(moduleRoster(v))
}

func moduleTypeLen(v Value) int64 {
	return recordTypeLen(moduleRoster(v))
}

// "pi" in math: a member name.
func moduleTypeContains(v Value, e Value) (bool, error) {
	return recordTypeContains(moduleRoster(v), e)
}

func moduleTypeAccessNamedProperty(vm VM, v Value, name string) (Value, error) {
	if r, ok, err := PropertyByLookup(vm, v, name); ok {
		return r, err
	}
	return Undefined, newModuleNoMemberError(errs.KindInvalidSelector, moduleName(v), name)
}

func moduleTypeCallNamedMethod(vm VM, v Value, name string, args []Value) (Value, error) {
	d := &ValueTypes[v.Type]
	if id := member.Lookup(name); id != member.Unknown && int(id) < len(d.Methods) && d.Methods[id].Fn != nil {
		return d.Methods[id].Call(vm, v, id, args)
	}
	return Undefined, newModuleNoMemberError(errs.KindInvalidMethod, moduleName(v), name)
}

func moduleTypeAssignNamedProperty(_ VM, v Value, name string, _ Value) error {
	return newModuleReadOnlyError(moduleName(v), name)
}

func moduleTypeAssignIndex(v Value, k Value, _ Value) error {
	name, ok := k.AsString()
	if !ok {
		name = k.String()
	}
	return newModuleReadOnlyError(moduleName(v), name)
}

// newModuleNoMemberError is a module's miss, as a selector (kind invalid_selector) or a call (invalid_method).
func newModuleNoMemberError(kind string, module string, name string) *errs.Error {
	return errs.NewRecoverableError(kind, fmt.Sprintf("module '%s' has no member '%s'", module, name))
}

func newModuleReadOnlyError(module string, name string) *errs.Error {
	return errs.NewRecoverableError(errs.KindNotAssignable, fmt.Sprintf("module '%s' is read-only: cannot assign member '%s'", module, name))
}
