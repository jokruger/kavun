package kavun

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/module"
	"github.com/jokruger/kavun/stdlib"
	_ "github.com/jokruger/kavun/vm"
)

const (
	// SourceFileExtDefault is the default extension for source files.
	SourceFileExtDefault = ".kvn"
	UsedDefinedModule    = module.UserDefined
)

var (
	NewBuiltinFunction = core.NewBuiltinFunction
	InitModule         = stdlib.InitModule
	AllModuleNames     = stdlib.AllModuleNames
)
