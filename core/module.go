package core

import (
	"fmt"
	"slices"
	"unsafe"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// ModuleDef is a registered builtin module: a TYPE (module id == type id) whose Methods entries are its functions and
// whose Properties entries answer its function values and constants. Scripts reach its value through import.
type ModuleDef struct {
	Name string
	Type uint8        // type id == module id
	Pure bool         // the default grant is every module with Pure (compiler and VM)
	New  func() Value // the value import answers; builtin modules answer one singleton

	roster Value // frozen record name → value, functions and constants: len, in, for-in
}

var (
	moduleDefs   [256]*ModuleDef
	moduleByName = map[string]*ModuleDef{}
)

// LookupModule answers the module registered under name.
func LookupModule(name string) (*ModuleDef, bool) {
	def, ok := moduleByName[name]
	return def, ok
}

// ModuleByType answers the module registered at type id t, or nil.
func ModuleByType(t uint8) *ModuleDef {
	return moduleDefs[t]
}

// ModuleNames answers every registered module name, sorted.
func ModuleNames() []string {
	names := make([]string, 0, len(moduleByName))
	for name := range moduleByName {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// PureModuleNames answers the names of the modules granted by default, sorted.
func PureModuleNames() []string {
	names := make([]string, 0, len(moduleByName))
	for name, def := range moduleByName {
		if def.Pure {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// ModuleBuilder declares a module's functions and constants, then registers it as a type:
//
//	core.NewModule(value.ModuleMath, "math", true).
//		Const(members.Pi, core.FloatValue(math.Pi)).
//		PureFn(members.Sqrt, 40, mathSqrt, 1, core.Fixed).
//		Register()
//
// Builtin modules take a type id in 112–143, embedder modules one in 192–255 (shared with embedder types).
type ModuleBuilder struct {
	def    *ModuleDef
	descr  ValueTypeDescr
	roster map[string]Value
	slots  map[uint64]bool
	err    error
}

// NewModule starts a module of type typeID. pure is the module's grant class: a pure module neither reads nor
// changes the world, and is importable without an explicit grant.
func NewModule(typeID uint8, name string, pure bool) *ModuleBuilder {
	def := &ModuleDef{Name: name, Type: typeID, Pure: pure}
	def.New = func() Value { return Value{Type: typeID, Immutable: true, Ptr: unsafe.Pointer(def)} }
	b := &ModuleBuilder{def: def, descr: moduleDescr(), roster: map[string]Value{}, slots: map[uint64]bool{}}
	b.method(members.IsTrue, NewPureMethod(memberIsTrue, 0, Fixed))
	b.method(members.Format, NewPureMethod(memberFormat, 0, Variadic))
	b.method(members.Copy, NewPureMethod(memberSelf, 0, Fixed))
	b.method(members.Freeze, NewPureMethod(memberSelf, 0, Fixed))
	return b
}

// Fn declares a module function the optimizer never folds, at slot (0–127) of the module's builtin-function slots.
func (b *ModuleBuilder) Fn(id member.ID, slot uint64, fn NativeFunc, arity int, args Arguments) *ModuleBuilder {
	return b.fn(id, slot, fn, arity, args, false)
}

// PureFn declares a module function the optimizer may fold: an explicit purity claim (docs/purity.md).
func (b *ModuleBuilder) PureFn(id member.ID, slot uint64, fn NativeFunc, arity int, args Arguments) *ModuleBuilder {
	return b.fn(id, slot, fn, arity, args, true)
}

func (b *ModuleBuilder) fn(id member.ID, slot uint64, fn NativeFunc, arity int, args Arguments, pure bool) *ModuleBuilder {
	name, ok := b.name(id)
	if !ok {
		return b
	}
	if slot >= ModuleSlotSize {
		return b.fail(fmt.Errorf("function %q: slot %d out of range (max: %d)", name, slot, ModuleSlotSize-1))
	}
	abs := uint64(b.def.Type)*ModuleSlotSize + slot
	if b.slots[slot] || BuiltinFunctions[abs] != nil {
		return b.fail(fmt.Errorf("function %q: slot %d already filled", name, slot))
	}
	b.slots[slot] = true
	BuiltinFunctions[abs] = &BuiltinFunction{Func: fn, Module: b.def.Name, Name: name, Arity: arity, Variadic: bool(args), Pure: pure}
	fnValue := BuiltinFunctionValue(abs)
	call := func(vm VM, _ Value, _ member.ID, args []Value) (Value, error) { return fn(vm, args) }
	if pure {
		b.method(id, NewPureMethod(call, arity, args))
	} else {
		b.method(id, NewMethod(call, arity, args))
	}
	b.property(id, NewPureProperty(func(VM, Value, member.ID) (Value, error) { return fnValue, nil }, nil))
	b.roster[name] = fnValue
	return b
}

// Const declares a module constant.
func (b *ModuleBuilder) Const(id member.ID, val Value) *ModuleBuilder {
	name, ok := b.name(id)
	if !ok {
		return b
	}
	b.property(id, NewPureProperty(func(VM, Value, member.ID) (Value, error) { return val, nil }, nil))
	b.roster[name] = val
	return b
}

// Descr answers the module type's descriptor so an embedder can amend hooks before Register.
func (b *ModuleBuilder) Descr() *ValueTypeDescr {
	return &b.descr
}

// New sets the value import answers (default: one immutable singleton). The descriptor's hooks reach the module
// through its type id, never through Ptr, so a per-import factory may keep its own state there.
func (b *ModuleBuilder) New(f func() Value) *ModuleBuilder {
	b.def.New = f
	return b
}

// Register installs the module and its type. Builtin registration panics on the error at init.
func (b *ModuleBuilder) Register() error {
	t, name := b.def.Type, b.def.Name
	if b.err != nil {
		return errs.NewHostError(fmt.Sprintf("(module %s) %v", name, b.err))
	}
	if t < value.ModuleBase64 || (t > value.Reserved191 && t < value.FirstUserDefinedType) {
		return errs.NewHostError(fmt.Sprintf("(module %s) type id %d is not a module type id", name, t))
	}
	if moduleDefs[t] != nil {
		return errs.NewHostError(fmt.Sprintf("(module %s) type id %d is already module '%s'", name, t, moduleDefs[t].Name))
	}
	if _, ok := moduleByName[name]; ok {
		return errs.NewHostError(fmt.Sprintf("(module %s) name already registered", name))
	}
	b.def.roster = NewRecordValue(b.roster, true)
	if t >= value.FirstUserDefinedType {
		if err := SetValueType(t, b.descr); err != nil {
			return err
		}
	} else {
		if err := validateTables(&b.descr); err != nil {
			return errs.NewHostError(fmt.Sprintf("(module %s) %v", name, err))
		}
		setValueType(t, b.descr)
	}
	moduleDefs[t] = b.def
	moduleByName[name] = b.def
	return nil
}

// UnregisterModule removes an embedder module (type id 192–255): its registration, its builtin-function slots and
// its type descriptor. For host teardown and test harnesses.
func UnregisterModule(typeID uint8) error {
	if typeID < value.FirstUserDefinedType {
		return errs.NewHostError(fmt.Sprintf("(UnregisterModule) cannot unregister builtin module type %d", typeID))
	}
	def := moduleDefs[typeID]
	if def == nil {
		return errs.NewHostError(fmt.Sprintf("(UnregisterModule) no module at type %d", typeID))
	}
	base := uint64(typeID) * ModuleSlotSize
	clear(BuiltinFunctions[base : base+ModuleSlotSize])
	ValueTypes[typeID] = DefaultValueType
	moduleDefs[typeID] = nil
	delete(moduleByName, def.Name)
	return nil
}

// name answers id's member name, recording the builder's first refusal: an unbound id, a name this module already
// declares, or one of the four universal members every module answers itself.
func (b *ModuleBuilder) name(id member.ID) (string, bool) {
	if b.err != nil {
		return "", false
	}
	name := id.String()
	switch {
	case id == member.Unknown || name == "":
		b.fail(fmt.Errorf("member id %d is not bound", id))
	case id == members.IsTrue || id == members.Format || id == members.Copy || id == members.Freeze:
		b.fail(fmt.Errorf("%q is a universal member and cannot be declared", name))
	case b.has(name):
		b.fail(fmt.Errorf("%q is declared twice", name))
	default:
		return name, true
	}
	return "", false
}

func (b *ModuleBuilder) has(name string) bool {
	_, ok := b.roster[name]
	return ok
}

func (b *ModuleBuilder) fail(err error) *ModuleBuilder {
	if b.err == nil {
		b.err = err
	}
	return b
}

func (b *ModuleBuilder) method(id member.ID, m Method) {
	if int(id) >= len(b.descr.Methods) {
		b.descr.Methods = slices.Grow(b.descr.Methods, int(id)+1-len(b.descr.Methods))[:id+1]
	}
	b.descr.Methods[id] = m
}

func (b *ModuleBuilder) property(id member.ID, p Property) {
	if int(id) >= len(b.descr.Properties) {
		b.descr.Properties = slices.Grow(b.descr.Properties, int(id)+1-len(b.descr.Properties))[:id+1]
	}
	b.descr.Properties[id] = p
}
