package stdlib

import (
	"fmt"
	"maps"
	"slices"

	"github.com/jokruger/kavun/core"
)

type BuiltinModuleInitializer func(a *core.Arena, m map[string]core.Value) error

type Module struct {
	ID    uint8
	Name  string
	Slot  uint64
	Attrs map[string]core.Value
	Init  BuiltinModuleInitializer
}

var (
	id2Module   = make([]*Module, core.BI_MAX_MODULES)
	name2Module = make(map[string]*Module)
)

func InitModule(name string, id uint8, bmi BuiltinModuleInitializer, cs map[string]core.Value, fns map[uint64]*core.BuiltinFunction) {
	m := &Module{
		ID:    id,
		Name:  name,
		Slot:  uint64(id) * core.BI_SLOT_SIZE,
		Attrs: make(map[string]core.Value, len(cs)+len(fns)),
		Init:  bmi,
	}

	for k, v := range cs {
		m.Attrs[k] = v
	}

	for i, fn := range fns {
		fn.Module = name
		id := m.Slot + i
		core.BuiltinFunctions[id] = fn
		m.Attrs[fn.Name] = core.BuiltinFunctionValue(id)
	}

	id2Module[id] = m
	name2Module[name] = m
}

func RemoveModule(name string) {
	if m, ok := name2Module[name]; ok {
		id2Module[m.ID] = nil
	}
	delete(name2Module, name)
}

func GetModuleID(name string) (uint8, bool) {
	m, ok := name2Module[name]
	if !ok {
		return 0, false
	}
	return m.ID, true
}

func GetModuleName(id uint8) (string, bool) {
	m := id2Module[id]
	if m == nil {
		return "", false
	}
	return m.Name, true
}

func GetModuleDefinition(name string) (*Module, bool) {
	m, ok := name2Module[name]
	return m, ok
}

func GetModule(a *core.Arena, id uint8) (core.Value, error) {
	// find module
	if id >= core.BI_MAX_MODULES {
		return core.Undefined, fmt.Errorf("invalid builtin module ID: %d", id)
	}
	m := id2Module[id]
	if m == nil {
		return core.Undefined, fmt.Errorf("builtin module not found for ID: %d", id)
	}
	attrs := m.Attrs

	// initialize module if needed
	if m.Init != nil {
		attrs = maps.Clone(attrs)
		if err := m.Init(a, attrs); err != nil {
			return core.Undefined, fmt.Errorf("failed to initialize builtin module %s: %w", m.Name, err)
		}
	}

	// return module as immutable record value
	return a.NewRecordValue(attrs, true)
}

func AllModuleNames() []string {
	names := make([]string, 0, len(name2Module))
	for name := range name2Module {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
