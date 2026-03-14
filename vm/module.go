package vm

import (
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

type Module struct {
	Attrs map[string]core.Object
}

// Import returns an immutable map for the module.
func (m *Module) Import(moduleName string) (interface{}, error) {
	return m.AsImmutableMap(moduleName), nil
}

// AsImmutableMap converts builtin module into an immutable map.
func (m *Module) AsImmutableMap(moduleName string) *value.ImmutableMap {
	attrs := make(map[string]core.Object, len(m.Attrs))
	for k, v := range m.Attrs {
		attrs[k] = v.Copy()
	}
	attrs["__module_name__"] = &value.String{Value: moduleName}
	return &value.ImmutableMap{Value: attrs}
}
