package gs

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/value"
	"github.com/jokruger/gs/vm"
)

// Script can simplify compilation and execution of embedded scripts.
type Script struct {
	alloc            core.Allocator
	variables        map[string]*Variable
	modules          vm.ModuleGetter
	input            []byte
	maxAllocs        int64
	maxConstObjects  int
	enableFileImport bool
	importDir        string
}

// NewScript creates a Script instance with an input script.
func NewScript(alloc core.Allocator, input []byte) *Script {
	return &Script{
		alloc:           alloc,
		variables:       make(map[string]*Variable),
		input:           input,
		maxAllocs:       -1,
		maxConstObjects: -1,
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

// SetMaxAllocs sets the maximum number of objects allocations during the run time.
// Compiled script will return gse.ErrObjectAllocLimit error if it exceeds this limit.
func (s *Script) SetMaxAllocs(n int64) {
	s.maxAllocs = n
}

// SetMaxConstObjects sets the maximum number of objects in the compiled constants.
func (s *Script) SetMaxConstObjects(n int) {
	s.maxConstObjects = n
}

// EnableFileImport enables or disables module loading from local files. Local file modules are disabled by default.
func (s *Script) EnableFileImport(enable bool) {
	s.enableFileImport = enable
}

// Compile compiles the script with all the defined variables, and, returns Compiled object.
func (s *Script) Compile() (*Compiled, error) {
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

	c := NewCompiler(s.alloc, srcFile, symbolTable, nil, s.modules, nil)
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
	bytecode.RemoveDuplicates()

	// check the constant objects limit
	if s.maxConstObjects >= 0 {
		cnt := bytecode.CountObjects()
		if cnt > s.maxConstObjects {
			return nil, fmt.Errorf("exceeding constant objects limit: %d", cnt)
		}
	}
	return &Compiled{
		alloc:         s.alloc,
		globalIndexes: globalIndexes,
		bytecode:      bytecode,
		globals:       globals,
		maxAllocs:     s.maxAllocs,
	}, nil
}

// Run compiles and runs the scripts. Use returned compiled object to access global variables.
func (s *Script) Run() (compiled *Compiled, err error) {
	compiled, err = s.Compile()
	if err != nil {
		return
	}
	err = compiled.Run()
	return
}

// RunContext is like Run but includes a context.
func (s *Script) RunContext(ctx context.Context) (compiled *Compiled, err error) {
	compiled, err = s.Compile()
	if err != nil {
		return
	}
	err = compiled.RunContext(ctx)
	return
}

func (s *Script) prepCompile() (symbolTable *vm.SymbolTable, globals []core.Value, err error) {
	names := make([]string, 0, len(s.variables))
	for name := range s.variables {
		names = append(names, name)
	}

	symbolTable = vm.NewSymbolTable()
	for idx, fn := range vm.BuiltinFuncs {
		// it is safe to cast type because we know that all values in vm.BuiltinFuncs are *value.BuiltinFunction objects
		symbolTable.DefineBuiltin(idx, fn.Object().(*value.BuiltinFunction).Name())
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
	alloc         core.Allocator
	globalIndexes map[string]int // global symbol name to index
	bytecode      *vm.Bytecode
	globals       []core.Value
	maxAllocs     int64
	lock          sync.RWMutex
}

// Run executes the compiled script in the virtual machine.
func (c *Compiled) Run() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	v := vm.NewVM(c.alloc, c.bytecode, c.globals, c.maxAllocs)
	return v.Run()
}

// RunContext is like Run but includes a context.
func (c *Compiled) RunContext(ctx context.Context) (err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	v := vm.NewVM(c.alloc, c.bytecode, c.globals, c.maxAllocs)
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
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.bytecode.Size() + int64(len(c.globalIndexes)+len(c.globals))
}

// Clone creates a new copy of Compiled. Cloned copies are safe for concurrent use by multiple goroutines.
func (c *Compiled) Clone() *Compiled {
	c.lock.RLock()
	defer c.lock.RUnlock()

	clone := &Compiled{
		alloc:         c.alloc,
		globalIndexes: c.globalIndexes,
		bytecode:      c.bytecode,
		globals:       make([]core.Value, len(c.globals)),
		maxAllocs:     c.maxAllocs,
	}

	// copy global objects
	for idx, g := range c.globals {
		clone.globals[idx] = g.Copy(c.alloc)
	}

	return clone
}

// IsDefined returns true if the variable name is defined (has value) before or after the execution.
func (c *Compiled) IsDefined(name string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	idx, ok := c.globalIndexes[name]
	if !ok {
		return false
	}
	v := c.globals[idx]
	return !v.IsUndefined()
}

// Get returns a variable identified by the name.
func (c *Compiled) Get(name string) *Variable {
	c.lock.RLock()
	defer c.lock.RUnlock()

	v := core.NewUndefined()
	if idx, ok := c.globalIndexes[name]; ok {
		v = c.globals[idx]
	}

	return NewVariable(name, v)
}

// GetAll returns all the variables that are defined by the compiled script.
func (c *Compiled) GetAll() []*Variable {
	c.lock.RLock()
	defer c.lock.RUnlock()

	vars := make([]*Variable, 0, len(c.globalIndexes))
	for name, idx := range c.globalIndexes {
		v := c.globals[idx]
		vars = append(vars, NewVariable(name, v))
	}
	return vars
}

// Set replaces the value of a global variable identified by the name.
// An error will be returned if the name was not defined during compilation.
func (c *Compiled) Set(name string, val core.Value) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	idx, ok := c.globalIndexes[name]
	if !ok {
		return fmt.Errorf("'%s' is not defined", name)
	}
	c.globals[idx] = val
	return nil
}
