package kavun

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/module"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/vm"
)

const (
	// SourceFileExtDefault is the default extension for source files.
	SourceFileExtDefault = ".kvn"
	UsedDefinedModule    = module.UserDefined
	UserDefinedType      = value.FirstUserDefinedType
)

// RuntimeError is the error Compiled.Run, Compiled.RunContext and Eval answer when a script fails at run time.
// See vm.RuntimeError for the fields and the errors.Is / errors.As contract.
type RuntimeError = vm.RuntimeError

var (
	NewBuiltinFunction = core.NewBuiltinFunction
	InitModule         = stdlib.InitModule
	AllModuleNames     = stdlib.AllModuleNames
)
