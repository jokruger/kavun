package kavun

import (
	"fmt"
	"path/filepath"

	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/vm"
)

// Script represents a script with its source code, variables, and compilation settings. It simplifies the process of
// compiling and executing embedded scripts by managing the necessary components and configurations.
type Script struct {
	oc               *compiler.OptimizationConfig
	allowedModules   []string
	customModules    map[string][]byte
	bindings         []string
	source           []byte
	importDir        string
	enableFileImport bool
	assignmentMode   compiler.AssignmentMode
}

// NewScript creates a Script instance with the given source code and bound variable names (optional). The script is
// initialized with default settings, including smart assignment mode, file import disabled and all builtin modules
// allowed.
func NewScript(source []byte, bindings ...string) *Script {
	return &Script{
		oc:             compiler.O3(),
		source:         source,
		bindings:       bindings,
		assignmentMode: compiler.AssignmentModeSmart,
	}
}

// SetOptimizationConfig sets the optimization configuration for the script.
func (s *Script) SetOptimizationConfig(oc *compiler.OptimizationConfig) {
	s.oc = oc
}

// SetSource sets the source code for the script.
func (s *Script) SetSource(source []byte) {
	s.source = source
}

// SetBindings sets the bound variable names for the script.
func (s *Script) SetBindings(bindings ...string) {
	s.bindings = bindings
}

// AddBindings adds new bound variable names to the script.
func (s *Script) AddBindings(bindings ...string) {
	s.bindings = append(s.bindings, bindings...)
}

// SetAllowedModules sets the allowed builtin module names for import. If not set, all modules are allowed.
func (s *Script) SetAllowedModules(modules ...string) {
	s.allowedModules = modules
}

// AddCustomModule adds a custom module with the given name and source code.
func (s *Script) AddCustomModule(name string, source []byte) {
	if s.customModules == nil {
		s.customModules = make(map[string][]byte)
	}
	s.customModules[name] = source
}

// EnableFileImport enables file import for the script, allowing it to import other scripts from the specified
// directory.
func (s *Script) EnableFileImport(path string) error {
	dir, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	s.importDir = dir
	s.enableFileImport = true
	return nil
}

// DisableFileImport disables file import for the script.
func (s *Script) DisableFileImport() {
	s.enableFileImport = false
	s.importDir = ""
}

// SetAssignmentMode sets how plain '=' handles unresolved identifiers during compilation.
func (s *Script) SetAssignmentMode(mode compiler.AssignmentMode) {
	s.assignmentMode = mode
}

// Compile compiles the script and returns a Compiled instance containing bytecode and global variable indexes.
func (s *Script) Compile() (*Compiled, error) {
	symbolTable := compiler.NewSymbolTable()
	for idx, name := range vm.BuiltinFunctionNames {
		symbolTable.DefineBuiltin(idx, name)
	}

	globals := make([]core.Value, vm.GlobalsSize)
	for idx, name := range s.bindings {
		symbol := symbolTable.Define(name)
		if symbol.Index != idx {
			panic(fmt.Errorf("wrong symbol index: %d != %d", idx, symbol.Index))
		}
		globals[symbol.Index] = core.Undefined
	}

	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("(main)", -1, len(s.source))

	c := compiler.NewCompiler(s.oc, nil, srcFile, symbolTable, s.allowedModules, s.customModules, nil)
	c.SetAssignmentMode(s.assignmentMode)
	c.EnableFileImport(s.enableFileImport)
	c.SetImportDir(s.importDir)
	if err := c.Compile(srcFile, s.source, nil); err != nil {
		return nil, err
	}

	// reduce globals size
	globals = globals[:symbolTable.MaxSymbols()+1]

	// bindings
	bindings := make(map[string]int, len(s.bindings))
	for _, name := range s.bindings {
		symbol, _, _ := symbolTable.Resolve(name, false)
		if symbol.Scope == compiler.ScopeGlobal {
			bindings[name] = symbol.Index
		}
	}

	return &Compiled{
		bytecode: c.Bytecode(),
		bindings: bindings,
		globals:  globals,
	}, nil
}
