package core

import (
	"fmt"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

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
		return d.Methods[id].Fn(vm, v, id, args)
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
