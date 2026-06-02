package kavun

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"

	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/vm"
)

// Script simplifies compilation and execution of embedded scripts.
type Script struct {
	variables        map[string]*Variable
	modules          vm.ModuleGetter
	input            []byte
	maxConstObjects  int
	assignmentMode   compiler.AssignmentMode
	importDir        string
	enableFileImport bool
}

// NewScript creates a Script instance with an input script.
func NewScript(input []byte) *Script {
	return &Script{
		variables:       make(map[string]*Variable),
		input:           input,
		maxConstObjects: -1,
		assignmentMode:  compiler.AssignmentModeSmart,
	}
}

// Add adds a new variable or updates an existing variable to the script.
func (s *Script) Add(name string, val core.Value) {
	s.variables[name] = NewVariable(name, val)
}

// Remove removes (undefine) an existing variable for the script. It returns false if the variable name is not defined.
func (s *Script) Remove(name string) bool {
	if _, ok := s.variables[name]; !ok {
		return false
	}
	delete(s.variables, name)
	return true
}

// SetImports sets import modules.
func (s *Script) SetImports(modules vm.ModuleGetter) {
	s.modules = modules
}

// SetImportDir sets the initial import directory for script files.
func (s *Script) SetImportDir(dir string) error {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	s.importDir = dir
	return nil
}

// SetMaxConstObjects sets the maximum number of objects in the compiled constants.
func (s *Script) SetMaxConstObjects(n int) {
	s.maxConstObjects = n
}

// SetAssignmentMode sets how plain '=' handles unresolved identifiers during compilation.
func (s *Script) SetAssignmentMode(mode compiler.AssignmentMode) {
	s.assignmentMode = mode
}

// EnableFileImport enables or disables module loading from local files. Local file modules are disabled by default.
func (s *Script) EnableFileImport(enable bool) {
	s.enableFileImport = enable
}

// Compile compiles the script with all the defined variables, and, returns Compiled object.
// If compile-time arena is not provided, a new default arena will be used.
func (s *Script) Compile(a *core.Arena) (*Compiled, error) {
	if a == nil {
		a = core.NewArena(nil)
	}

	symbolTable, globals, err := s.prepCompile()
	if err != nil {
		return nil, err
	}

	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("(main)", -1, len(s.input))
	p := parser.NewParser(srcFile, s.input, nil)
	file, err := p.ParseFile()
	if err != nil {
		return nil, err
	}

	c := compiler.New(a, srcFile, symbolTable, nil, s.modules, nil)
	c.SetAssignmentMode(s.assignmentMode)
	c.EnableFileImport(s.enableFileImport)
	c.SetImportDir(s.importDir)
	if err := c.Compile(file); err != nil {
		return nil, err
	}

	// reduce globals size
	globals = globals[:symbolTable.MaxSymbols()+1]

	// global symbol names to indexes
	globalIndexes := make(map[string]int, len(globals))
	for _, name := range symbolTable.Names() {
		symbol, _, _ := symbolTable.Resolve(name, false)
		if symbol.Scope == vm.ScopeGlobal {
			globalIndexes[name] = symbol.Index
		}
	}

	// remove duplicates from constants
	bytecode := c.Bytecode()
	if err := bytecode.RemoveDuplicates(a); err != nil {
		return nil, err
	}

	// check the constant objects limit
	if s.maxConstObjects >= 0 {
		cnt := bytecode.CountObjects()
		if cnt > s.maxConstObjects {
			return nil, fmt.Errorf("exceeding constant objects limit: %d", cnt)
		}
	}

	return &Compiled{
		bytecode: bytecode,
		index:    globalIndexes,
		globals:  globals,
		runtime:  make([]core.Value, len(globals)),
	}, nil
}

func (s *Script) prepCompile() (symbolTable *vm.SymbolTable, globals []core.Value, err error) {
	names := make([]string, 0, len(s.variables))
	for name := range s.variables {
		names = append(names, name)
	}

	symbolTable = vm.NewSymbolTable()
	for idx, fn := range vm.BuiltinFuncs {
		if bf, ok := core.ResolveBuiltinFunction(fn); ok {
			symbolTable.DefineBuiltin(idx, bf.Name)
		}
	}

	globals = make([]core.Value, vm.GlobalsSize)
	for idx, name := range names {
		symbol := symbolTable.Define(name)
		if symbol.Index != idx {
			panic(fmt.Errorf("wrong symbol index: %d != %d", idx, symbol.Index))
		}
		globals[symbol.Index] = s.variables[name].Value()
	}
	return
}

// Compiled is a compiled instance of the user script. Use Script.Compile() to create Compiled object.
type Compiled struct {
	bytecode *vm.Bytecode
	index    map[string]int // global symbol name to index
	globals  []core.Value
	runtime  []core.Value // global variables during execution
}

// Set replaces the value of a global variable identified by the name (must be used before script execution).
// An error will be returned if the name was not defined during compilation.
func (c *Compiled) Set(name string, val core.Value) error {
	i, ok := c.index[name]
	if !ok {
		return fmt.Errorf("'%s' is not defined", name)
	}
	c.globals[i] = val
	return nil
}

// Run executes the compiled script in the virtual machine.
func (c *Compiled) Run(a *core.Arena, v *vm.VM) error {
	if err := c.prepareRun(a, v); err != nil {
		return err
	}
	return v.Run()
}

// RunContext is like Run but includes a context.
func (c *Compiled) RunContext(ctx context.Context, a *core.Arena, v *vm.VM) (err error) {
	if err := c.prepareRun(a, v); err != nil {
		return err
	}

	ch := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				switch e := r.(type) {
				case string:
					ch <- fmt.Errorf("%s", e)
				case error:
					ch <- e
				default:
					ch <- fmt.Errorf("unknown panic: %v", e)
				}
			}
		}()
		ch <- v.Run()
	}()

	select {
	case <-ctx.Done():
		v.Abort()
		<-ch
		err = ctx.Err()
	case err = <-ch:
	}

	return
}

// Size of compiled script in bytes (as much as we can calculate it without reflection and black magic)
func (c *Compiled) Size() int64 {
	return c.bytecode.Size() + int64(len(c.index)+len(c.globals))
}

// Clone creates a new copy of Compiled.
func (c *Compiled) Clone(a *core.Arena) (*Compiled, error) {
	if a == nil {
		a = core.NewArena(nil)
	}

	clone := &Compiled{
		bytecode: c.bytecode,
		index:    make(map[string]int, len(c.index)),
		globals:  make([]core.Value, len(c.globals)),
		runtime:  make([]core.Value, len(c.globals)),
	}

	maps.Copy(clone.index, c.index)
	for i, v := range c.globals {
		t, err := v.Clone(a)
		if err != nil {
			return nil, err
		}
		clone.globals[i] = t
	}

	return clone, nil
}

// GetValue returns a value identified by the name.
// Must be used right after script execution to get the updated value. Otherwise, the result in ambiguous.
func (c *Compiled) GetValue(name string) core.Value {
	v := core.Undefined
	if i, ok := c.index[name]; ok {
		v = c.runtime[i]
	}
	return v
}

// Get returns a variable identified by the name.
// Must be used right after script execution to get the updated variable. Otherwise, the result in ambiguous.
func (c *Compiled) Get(name string) *Variable {
	return NewVariable(name, c.GetValue(name))
}

// GetAll returns all the variables that are defined by the compiled script.
// Must be used right after script execution to get the updated variables. Otherwise, the result in ambiguous.
func (c *Compiled) GetAll() []*Variable {
	vars := make([]*Variable, 0, len(c.index))
	for name, idx := range c.index {
		v := c.runtime[idx]
		vars = append(vars, NewVariable(name, v))
	}
	return vars
}

func (c *Compiled) prepareRun(a *core.Arena, v *vm.VM) error {
	if a == nil {
		return fmt.Errorf("runtime allocator is nil")
	}
	if v == nil {
		return fmt.Errorf("vm is nil")
	}

	a.Reset()
	for i, v := range c.globals {
		t, err := v.Clone(a)
		if err != nil {
			return err
		}
		c.runtime[i] = t
	}
	v.Reset(a, c.bytecode, c.runtime)
	return nil
}
