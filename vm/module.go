package vm

import (
	"github.com/jokruger/kavun/core"
)

type Module struct {
	Attrs map[string]core.Value
}

// Import returns an immutable record for the module.
func (m *Module) Import(alloc *core.Arena, moduleName string) (any, error) {
	t, err := m.AsImmutableRecord(alloc, moduleName)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// AsImmutableRecord converts builtin module into an immutable record.
func (m *Module) AsImmutableRecord(alloc *core.Arena, moduleName string) (core.Value, error) {
	attrs := make(map[string]core.Value, len(m.Attrs))
	for k, v := range m.Attrs {
		t, err := v.Copy(alloc)
		if err != nil {
			return core.Undefined, err
		}
		attrs[k] = t
	}
	t := alloc.NewStringValue(moduleName)
	attrs["__module_name__"] = t
	return alloc.NewRecordValue(attrs, true), nil
}
