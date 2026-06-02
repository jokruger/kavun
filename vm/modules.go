package vm

import (
	"github.com/jokruger/kavun/core"
)

// Importable interface represents importable module instance.
type Importable interface {
	// Import should return either an Object or module source code ([]byte).
	Import(alloc *core.Arena, moduleName string) (any, error)
}

// ModuleGetter enables implementing dynamic module loading.
type ModuleGetter interface {
	Get(name string) Importable
}

// ModuleMap represents a set of named modules. Use NewModuleMap to create a
// new module map.
type ModuleMap struct {
	m          map[string]Importable
	builtinIDs map[string]uint8
}

// BuiltinModuleLoader resolves builtin module values by static module ID.
type BuiltinModuleLoader func(alloc *core.Arena, id uint8) (core.Value, error)

var builtinModuleLoader BuiltinModuleLoader

// SetBuiltinModuleLoader sets global resolver used by OpImportBuiltinModule.
func SetBuiltinModuleLoader(loader BuiltinModuleLoader) {
	builtinModuleLoader = loader
}

// LoadBuiltinModuleByID resolves a builtin module by static ID.
func LoadBuiltinModuleByID(alloc *core.Arena, id uint8) (core.Value, error) {
	if builtinModuleLoader == nil {
		return core.Undefined, nil
	}
	return builtinModuleLoader(alloc, id)
}

// NewModuleMap creates a new module map.
func NewModuleMap() *ModuleMap {
	return &ModuleMap{
		m:          make(map[string]Importable),
		builtinIDs: make(map[string]uint8),
	}
}

// Add adds an import module.
func (m *ModuleMap) Add(name string, module Importable) {
	m.m[name] = module
}

// AddBuiltinModule adds a builtin module.
func (m *ModuleMap) AddBuiltinModule(name string, attrs map[string]core.Value) {
	m.m[name] = &Module{Attrs: attrs}
}

// AddBuiltinModuleWithID adds a builtin module with a static module ID.
func (m *ModuleMap) AddBuiltinModuleWithID(id uint8, name string, attrs map[string]core.Value) {
	m.m[name] = &Module{Attrs: attrs}
	m.builtinIDs[name] = id
}

// AddSourceModule adds a source module.
func (m *ModuleMap) AddSourceModule(name string, src []byte) {
	m.m[name] = &SourceModule{Src: src}
}

// Remove removes a named module.
func (m *ModuleMap) Remove(name string) {
	delete(m.m, name)
}

// Get returns an import module identified by name. It returns if the name is
// not found.
func (m *ModuleMap) Get(name string) Importable {
	return m.m[name]
}

// GetBuiltinModule returns a builtin module identified by name. It returns
// if the name is not found or the module is not a builtin module.
func (m *ModuleMap) GetBuiltinModule(name string) *Module {
	mod, _ := m.m[name].(*Module)
	return mod
}

// GetBuiltinModuleID returns static module ID for builtin module name.
func (m *ModuleMap) GetBuiltinModuleID(name string) (uint8, bool) {
	id, ok := m.builtinIDs[name]
	return id, ok
}

// GetSourceModule returns a source module identified by name. It returns if
// the name is not found or the module is not a source module.
func (m *ModuleMap) GetSourceModule(name string) *SourceModule {
	mod, _ := m.m[name].(*SourceModule)
	return mod
}

// Copy creates a copy of the module map.
func (m *ModuleMap) Copy() *ModuleMap {
	c := &ModuleMap{
		m:          make(map[string]Importable),
		builtinIDs: make(map[string]uint8),
	}
	for name, mod := range m.m {
		c.m[name] = mod
	}
	for name, id := range m.builtinIDs {
		c.builtinIDs[name] = id
	}
	return c
}

// Len returns the number of named modules.
func (m *ModuleMap) Len() int {
	return len(m.m)
}

// AddMap adds named modules from another module map.
func (m *ModuleMap) AddMap(o *ModuleMap) {
	for name, mod := range o.m {
		m.m[name] = mod
	}
}

// SourceModule is an importable module that's written in Kavun.
type SourceModule struct {
	Src []byte
}

// Import returns a module source code.
func (m *SourceModule) Import(*core.Arena, string) (any, error) {
	return m.Src, nil
}
