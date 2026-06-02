package stdlib

import (
	"fmt"
	"sort"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/vm"
)

const (
	BuiltinModuleMath   = uint8(1)
	BuiltinModuleOS     = uint8(2)
	BuiltinModuleText   = uint8(3)
	BuiltinModuleTimes  = uint8(4)
	BuiltinModuleRand   = uint8(5)
	BuiltinModuleFmt    = uint8(6)
	BuiltinModuleJSON   = uint8(7)
	BuiltinModuleBase64 = uint8(8)
	BuiltinModuleHex    = uint8(9)
)

var BuiltinModuleIDs = map[string]uint8{
	"math":   BuiltinModuleMath,
	"os":     BuiltinModuleOS,
	"text":   BuiltinModuleText,
	"times":  BuiltinModuleTimes,
	"rand":   BuiltinModuleRand,
	"fmt":    BuiltinModuleFmt,
	"json":   BuiltinModuleJSON,
	"base64": BuiltinModuleBase64,
	"hex":    BuiltinModuleHex,
}

var BuiltinModuleNames = map[uint8]string{
	BuiltinModuleMath:   "math",
	BuiltinModuleOS:     "os",
	BuiltinModuleText:   "text",
	BuiltinModuleTimes:  "times",
	BuiltinModuleRand:   "rand",
	BuiltinModuleFmt:    "fmt",
	BuiltinModuleJSON:   "json",
	BuiltinModuleBase64: "base64",
	BuiltinModuleHex:    "hex",
}

// BuiltinModules are builtin type standard library modules.
var BuiltinModules = map[string]map[string]core.Value{
	"math":   mathModule,
	"os":     osModule,
	"text":   textModule,
	"times":  timesModule,
	"rand":   randModule,
	"fmt":    fmtModule,
	"json":   jsonModule,
	"base64": base64Module,
	"hex":    hexModule,
}

// BuiltinModuleFunctionIDs contains per-module reverse index: function name -> static builtin function ID.
var BuiltinModuleFunctionIDs map[string]map[string]uint64

func makeModuleBuiltinFunctionID(moduleID uint8, slot int) uint64 {
	if slot < 0 || slot > 255 {
		panic(fmt.Sprintf("builtin module %d has too many static functions: %d", moduleID, slot+1))
	}
	return core.BuiltinFunctionID(moduleID, uint8(slot))
}

func init() {
	vm.SetBuiltinModuleLoader(loadBuiltinModuleByID)

	for name, id := range BuiltinModuleIDs {
		core.RegisterBuiltinModule(id, name)
	}

	BuiltinModuleFunctionIDs = make(map[string]map[string]uint64, len(BuiltinModules))
	for modName, attrs := range BuiltinModules {
		modID, ok := BuiltinModuleIDs[modName]
		if !ok {
			continue
		}

		var names []string
		for fnName, v := range attrs {
			if v.Type == core.VT_BUILTIN_FUNCTION {
				names = append(names, fnName)
			}
		}
		sort.Strings(names)

		idx := make(map[string]uint64)
		for slot, fnName := range names {
			v := attrs[fnName]
			bf, ok := core.ResolveBuiltinFunction(v)
			if !ok {
				continue
			}
			id := makeModuleBuiltinFunctionID(modID, slot)
			core.RegisterBuiltinFunctionAt(id, bf.Name, bf.Func, bf.Arity, bf.Variadic)
			attrs[fnName] = core.BuiltinFunctionValue(id)
			idx[fnName] = id
		}
		BuiltinModuleFunctionIDs[modName] = idx
	}
}

func loadBuiltinModuleByID(a *core.Arena, id uint8) (core.Value, error) {
	name, ok := BuiltinModuleNames[id]
	if !ok {
		return core.Undefined, fmt.Errorf("unknown builtin module id %d", id)
	}
	attrs := BuiltinModules[name]
	mod := &vm.Module{Attrs: attrs}
	return mod.AsImmutableRecord(a, name)
}
