package stdlib

import (
	"fmt"
	"slices"

	"github.com/jokruger/kavun/core"
)

type Module struct {
	ID   uint8
	Name string
	Slot uint64
	Body core.Value
}

var (
	id2Module   = make([]*Module, core.MaxModules)
	name2Module = make(map[string]*Module)
)

func InitModule(name string, id uint8, cs map[string]core.Value, fns map[uint64]*core.BuiltinFunction) {
	m := &Module{
		ID:   id,
		Name: name,
		Slot: uint64(id) * core.ModuleSlotSize,
	}

	attrs := make(map[string]core.Value, len(cs)+len(fns))
	for k, v := range cs {
		attrs[k] = v
	}
	for i, fn := range fns {
		fn.Module = name
		id := m.Slot + i
		core.BuiltinFunctions[id] = fn
		attrs[fn.Name] = core.BuiltinFunctionValue(id)
	}
	m.Body = core.NewRecordValue(attrs, true)

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

func GetModule(id uint8) (core.Value, error) {
	if id >= core.MaxModules {
		return core.Undefined, fmt.Errorf("invalid builtin module ID: %d", id)
	}
	m := id2Module[id]
	if m == nil {
		return core.Undefined, fmt.Errorf("builtin module not found for ID: %d", id)
	}
	return m.Body, nil
}

func AllModuleNames() []string {
	names := make([]string, 0, len(name2Module))
	for name := range name2Module {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
