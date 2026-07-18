package compiler

import (
	"fmt"
	"io"
	"path/filepath"
	"reflect"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/vm"
	"github.com/jokruger/set"
)

// AssignmentMode controls how plain '=' handles unresolved identifiers.
type AssignmentMode int

const (
	// DefaultSourceFileExt is the default extension used to resolve file imports.
	DefaultSourceFileExt = ".kvn"

	// AssignmentModeSmart declares a variable in current scope for unresolved '=' assignments.
	AssignmentModeSmart = AssignmentMode(0)

	// AssignmentModeStrict requires variables to already exist for '=' assignments.
	AssignmentModeStrict = AssignmentMode(1)
)

// CompilerError represents a compiler error.
type CompilerError struct {
	FileSet *ast.SourceFileSet
	Node    ast.Node
	Err     error
}

func (e *CompilerError) Error() string {
	filePos := e.FileSet.Position(e.Node.Pos())
	return fmt.Sprintf("Compile Error: %s\n\tat %s", e.Err.Error(), filePos)
}

// Compiler compiles the AST into a bytecode.
type Compiler struct {
	oc              *OptimizationConfig
	sb              *StaticBuilder
	file            *ast.SourceFile
	parent          *Compiler
	modulePath      string
	importDir       string
	importFileExt   []string
	symbolTable     *SymbolTable
	scopes          []compilationScope
	scopeIndex      int
	allowedModules  set.Set[string]
	customModules   map[string][]byte
	compiledModules map[string]core.CompiledFunction
	allowFileImport bool
	loops           []*loop
	loopIndex       int
	assignmentMode  AssignmentMode
	compilingInit   bool
	trace           io.Writer
	indent          int
}

// New creates a Compiler.
func NewCompiler(
	oc *OptimizationConfig,
	sb *StaticBuilder,
	file *ast.SourceFile,
	symbolTable *SymbolTable,
	allowedModules []string,
	customModules map[string][]byte,
	trace io.Writer,
) *Compiler {
	if oc == nil {
		oc = O0()
	}

	if sb == nil {
		sb = NewStaticBuilder()
	}

	mainScope := compilationScope{
		SymbolInit: make(map[string]bool),
		SourceMap:  make(map[int]core.Pos),
	}

	// symbol table
	if symbolTable == nil {
		symbolTable = NewSymbolTable()
	}

	// add builtin functions to the symbol table
	for idx, name := range vm.BuiltinFunctionNames {
		symbolTable.DefineBuiltin(idx, name)
	}

	var ms set.Set[string]
	if allowedModules != nil {
		ms = set.NewFromSlice(allowedModules)
	}

	if customModules == nil {
		customModules = make(map[string][]byte)
	}

	return &Compiler{
		oc:              oc,
		sb:              sb,
		file:            file,
		symbolTable:     symbolTable,
		scopes:          []compilationScope{mainScope},
		scopeIndex:      0,
		loopIndex:       -1,
		assignmentMode:  AssignmentModeSmart,
		trace:           trace,
		allowedModules:  ms,
		customModules:   customModules,
		compiledModules: make(map[string]core.CompiledFunction),
		importFileExt:   []string{DefaultSourceFileExt},
	}
}

// SetAssignmentMode sets how plain '=' handles unresolved identifiers.
func (c *Compiler) SetAssignmentMode(mode AssignmentMode) {
	switch mode {
	case AssignmentModeSmart, AssignmentModeStrict:
		c.assignmentMode = mode
	default:
		panic(fmt.Errorf("invalid assignment mode: %d", mode))
	}
}

// GetAssignmentMode returns the active assignment mode.
func (c *Compiler) GetAssignmentMode() AssignmentMode {
	return c.assignmentMode
}

// Compile compiles the source file into an optimized bytecode.
func (c *Compiler) Compile(file *ast.SourceFile, src []byte, trace io.Writer) error {
	p := parser.NewParser(file, src, trace)
	f, err := p.ParseFile()
	if err != nil {
		return err
	}

	if err := c.validatePreOptimization(file, c.modulePath, f, snapshotGlobals(c.symbolTable), false); err != nil {
		return err
	}

	n, err := c.Optimize(f)
	if err != nil {
		return err
	}

	return c.CompileNode(n)
}

// Compile compiles the AST node.
func (c *Compiler) CompileNode(node ast.Node) (err error) {
	if c.trace != nil {
		if node != nil {
			defer untracec(tracec(c, fmt.Sprintf("%s (%s)", node.String(), reflect.TypeOf(node).Elem().Name())))
		} else {
			defer untracec(tracec(c, "<nil>"))
		}
	}

	switch node := node.(type) {
	case *ast.File:
		for _, stmt := range node.Stmts {
			if err = c.CompileNode(stmt); err != nil {
				return err
			}
		}

	case ast.Expression:
		if err = c.compileExpression(node); err != nil {
			return err
		}

	case ast.Statement:
		if err = c.compileStatement(node); err != nil {
			return err
		}
	}

	return nil
}

// Bytecode returns a compiled bytecode.
func (c *Compiler) Bytecode() *vm.Bytecode {
	mainInsts := append(c.currentInstructions(), NewSuspend())
	return &vm.Bytecode{
		FileSet: c.file.Set(),
		MainFunction: &core.CompiledFunction{
			Instructions: mainInsts,
			MaxStack:     ComputeMaxStack(mainInsts),
			SourceMap:    c.currentSourceMap(),
		},
		Static: c.sb.Build(),
	}
}

// EnableFileImport enables or disables module loading from local files.
// Local file modules are disabled by default.
func (c *Compiler) EnableFileImport(enable bool) {
	c.allowFileImport = enable
}

// SetImportDir sets the initial import directory path for file imports.
func (c *Compiler) SetImportDir(dir string) {
	c.importDir = dir
}

// SetImportFileExt sets the extension name of the source file for loading local module files.
// Use this method if you want other source file extension than ".kvn".
// This function requires at least one argument, since it will replace the current list of extension name.
func (c *Compiler) SetImportFileExt(exts ...string) error {
	if len(exts) == 0 {
		return fmt.Errorf("missing arg: at least one argument is required")
	}

	for _, ext := range exts {
		if ext != filepath.Ext(ext) || ext == "" {
			return fmt.Errorf("invalid file extension: %s", ext)
		}
	}

	c.importFileExt = exts // Replace the hole current extension list

	return nil
}

// GetImportFileExt returns the current list of extension name.
// These are the complementary suffix of the source file to search and load local module files.
func (c *Compiler) GetImportFileExt() []string {
	return c.importFileExt
}
