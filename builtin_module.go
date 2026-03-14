package gs

import (
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

type BuiltinModule struct {
	Attrs map[string]core.Object
}

// Import returns an immutable map for the module.
func (m *BuiltinModule) Import(moduleName string) (any, error) {
	return m.AsImmutableMap(moduleName), nil
}

// AsImmutableMap converts builtin module into an immutable map.
func (m *BuiltinModule) AsImmutableMap(moduleName string) *value.ImmutableMap {
	attrs := make(map[string]core.Object, len(m.Attrs))
	for k, v := range m.Attrs {
		attrs[k] = v.Copy()
	}
	attrs["__module_name__"] = &value.String{Value: moduleName}
	return &value.ImmutableMap{Value: attrs}
}
