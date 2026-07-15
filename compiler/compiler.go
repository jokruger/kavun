package compiler

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/ast/expression"
	"github.com/jokruger/kavun/ast/statement"
	"github.com/jokruger/kavun/core"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
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

func (c *Compiler) compileAssign(node ast.Node, lhs, rhs []ast.Expression, op token.Token) error {
	var err error

	numLHS, numRHS := len(lhs), len(rhs)
	if numLHS > 1 || numRHS > 1 {
		return c.errorf(node, "tuple assignment not allowed")
	}

	// resolve and compile left-hand side
	ident, selectors := resolveAssignLHS(lhs[0])
	numSel := len(selectors)

	if op == token.Define && numSel > 0 {
		// using selector on new variable does not make sense
		return c.errorf(node, "operator ':=' not allowed with selector")
	}

	_, isFunc := rhs[0].(*expression.Function)
	symbol, depth, exists := c.symbolTable.Resolve(ident, false)
	// Builtins are pre-seeded global-like values. They may be shadowed in inner scopes (via :=) and reassigned at the
	// top level (via := or =, the latter under smart assignment mode). They have no addressable storage, so compound
	// assignments (+=, -=, etc.) remain disallowed.
	if exists && symbol.Scope == ScopeBuiltin {
		if op != token.Define && op != token.Assign {
			return c.errorf(node, "cannot assign to builtin '%s'", ident)
		}
		symbol = nil
		exists = false
		depth = 0
	}
	if op == token.Define {
		if depth == 0 && exists {
			return c.errorf(node, "'%s' redeclared in this block", ident)
		}
		if isFunc {
			symbol = c.symbolTable.Define(ident)
		}
	} else {
		if !exists {
			if op == token.Assign && numSel == 0 && c.assignmentMode == AssignmentModeSmart {
				if isFunc {
					symbol = c.symbolTable.Define(ident)
				}
			} else {
				return c.errorf(node, "unresolved reference '%s'", ident)
			}
		}
	}

	// +=, -=, *=, /=
	if op != token.Assign && op != token.Define {
		if err := c.CompileNode(lhs[0]); err != nil {
			return err
		}
	}

	// compile RHSs
	for _, expr := range rhs {
		if err := c.CompileNode(expr); err != nil {
			return err
		}
	}

	if (op == token.Define || (op == token.Assign && numSel == 0 && c.assignmentMode == AssignmentModeSmart && !exists)) && !isFunc {
		symbol = c.symbolTable.Define(ident)
	}

	switch op {
	case token.AddAssign:
		_, err = c.emit(node, NewBinaryOp(token.Add))
	case token.SubAssign:
		_, err = c.emit(node, NewBinaryOp(token.Sub))
	case token.MulAssign:
		_, err = c.emit(node, NewBinaryOp(token.Mul))
	case token.QuoAssign:
		_, err = c.emit(node, NewBinaryOp(token.Quo))
	case token.RemAssign:
		_, err = c.emit(node, NewBinaryOp(token.Rem))
	case token.AndAssign:
		_, err = c.emit(node, NewBinaryOp(token.And))
	case token.OrAssign:
		_, err = c.emit(node, NewBinaryOp(token.Or))
	case token.AndNotAssign:
		_, err = c.emit(node, NewBinaryOp(token.AndNot))
	case token.XorAssign:
		_, err = c.emit(node, NewBinaryOp(token.Xor))
	case token.ShlAssign:
		_, err = c.emit(node, NewBinaryOp(token.Shl))
	case token.ShrAssign:
		_, err = c.emit(node, NewBinaryOp(token.Shr))
	}
	if err != nil {
		return err
	}

	// compile selector expressions (right to left)
	for i := numSel - 1; i >= 0; i-- {
		if err := c.CompileNode(selectors[i]); err != nil {
			return err
		}
	}

	switch symbol.Scope {
	case ScopeGlobal:
		if numSel > 0 {
			_, err = c.emit(node, NewStoreIndexedGlobal(symbol.Index, numSel))
		} else {
			_, err = c.emit(node, NewStoreGlobal(symbol.Index))
		}

	case ScopeLocal:
		if numSel > 0 {
			_, err = c.emit(node, NewStoreIndexedLocal(symbol.Index, numSel))
		} else {
			if op == token.Define && !symbol.LocalAssigned {
				_, err = c.emit(node, NewDefineLocal(symbol.Index))
			} else {
				_, err = c.emit(node, NewStoreLocal(symbol.Index))
			}
		}
		// mark the symbol as local-assigned
		symbol.LocalAssigned = true

	case ScopeFree:
		if numSel > 0 {
			_, err = c.emit(node, NewStoreIndexedFree(symbol.Index, numSel))
		} else {
			_, err = c.emit(node, NewStoreFree(symbol.Index))
		}

	default:
		return fmt.Errorf("invalid assignment variable scope: %s", symbol.Scope)
	}
	if err != nil {
		return err
	}

	return nil
}

func (c *Compiler) compileLogical(node *expression.Binary) (err error) {
	// left side term
	if err = c.CompileNode(node.LHS); err != nil {
		return err
	}

	// jump position
	var jumpPos int
	if node.Token == token.LAnd {
		jumpPos, err = c.emit(node, NewAndJump(0))
		if err != nil {
			return err
		}
	} else {
		jumpPos, err = c.emit(node, NewOrJump(0))
		if err != nil {
			return err
		}
	}

	// right side term
	if err = c.CompileNode(node.RHS); err != nil {
		return err
	}

	if err = c.changeJumpAddr(jumpPos, len(c.currentInstructions())); err != nil {
		return err
	}

	return nil
}

func (c *Compiler) compileForStmt(stmt *statement.For) (err error) {
	c.symbolTable = c.symbolTable.Fork(true)
	defer func() {
		c.symbolTable = c.symbolTable.Parent(false)
	}()

	// init statement
	if stmt.Init != nil {
		if err = c.CompileNode(stmt.Init); err != nil {
			return err
		}
	}

	// pre-condition position
	preCondPos := len(c.currentInstructions())

	// condition expression
	postCondPos := -1
	if stmt.Cond != nil {
		if err := c.CompileNode(stmt.Cond); err != nil {
			return err
		}
		// condition jump position
		postCondPos, err = c.emit(stmt, NewJumpFalsy(0))
		if err != nil {
			return err
		}
	}

	// enter loop
	loop := c.enterLoop()

	// body statement
	if err = c.CompileNode(stmt.Body); err != nil {
		c.leaveLoop()
		return err
	}

	c.leaveLoop()

	// post-body position
	postBodyPos := len(c.currentInstructions())

	// post statement
	if stmt.Post != nil {
		if err = c.CompileNode(stmt.Post); err != nil {
			return err
		}
	}
	if _, err = c.emit(stmt, NewAbortCheck()); err != nil {
		return err
	}

	// back to condition
	if _, err = c.emit(stmt, NewJump(preCondPos)); err != nil {
		return err
	}

	// post-statement position
	postStmtPos := len(c.currentInstructions())
	if postCondPos >= 0 {
		if err = c.changeJumpAddr(postCondPos, postStmtPos); err != nil {
			return err
		}
	}

	// update all break/continue jump positions
	for _, pos := range loop.Breaks {
		if err = c.changeJumpAddr(pos, postStmtPos); err != nil {
			return err
		}
	}
	for _, pos := range loop.Continues {
		if err = c.changeJumpAddr(pos, postBodyPos); err != nil {
			return err
		}
	}

	return nil
}

func (c *Compiler) compileForInStmt(stmt *statement.ForIn) error {
	c.symbolTable = c.symbolTable.Fork(true)
	defer func() {
		c.symbolTable = c.symbolTable.Parent(false)
	}()

	// for-in statement is compiled like following:
	//
	//   for :it := iterator(iterable); :it.next();  {
	//     k, v := :it.get()  // DEFINE operator
	//
	//     ... body ...
	//   }
	//
	// ":it" is a local variable but it will not conflict with other user variables
	// because character ":" is not allowed in the variable names.

	// init
	//   :it = iterator(iterable)
	itSymbol := c.symbolTable.Define(":it")
	if err := c.CompileNode(stmt.Iterable); err != nil {
		return err
	}
	if _, err := c.emit(stmt, NewIterInit()); err != nil {
		return err
	}
	if itSymbol.Scope == ScopeGlobal {
		if _, err := c.emit(stmt, NewStoreGlobal(itSymbol.Index)); err != nil {
			return err
		}
	} else {
		if _, err := c.emit(stmt, NewDefineLocal(itSymbol.Index)); err != nil {
			return err
		}
	}

	// pre-condition position
	preCondPos := len(c.currentInstructions())

	// condition
	//  :it.HasMore()
	if itSymbol.Scope == ScopeGlobal {
		if _, err := c.emit(stmt, NewLoadGlobal(itSymbol.Index)); err != nil {
			return err
		}
	} else {
		if _, err := c.emit(stmt, NewLoadLocal(itSymbol.Index)); err != nil {
			return err
		}
	}
	if _, err := c.emit(stmt, NewIterNext()); err != nil {
		return err
	}

	// condition jump position
	postCondPos, err := c.emit(stmt, NewJumpFalsy(0))
	if err != nil {
		return err
	}

	// enter loop
	loop := c.enterLoop()

	// assign key variable
	if stmt.Key.String() != "_" {
		keySymbol := c.symbolTable.Define(stmt.Key.String())
		if itSymbol.Scope == ScopeGlobal {
			if _, err := c.emit(stmt, NewLoadGlobal(itSymbol.Index)); err != nil {
				return err
			}
		} else {
			if _, err := c.emit(stmt, NewLoadLocal(itSymbol.Index)); err != nil {
				return err
			}
		}
		if _, err := c.emit(stmt, NewIterKey()); err != nil {
			return err
		}
		if keySymbol.Scope == ScopeGlobal {
			if _, err := c.emit(stmt, NewStoreGlobal(keySymbol.Index)); err != nil {
				return err
			}
		} else {
			keySymbol.LocalAssigned = true
			if _, err := c.emit(stmt, NewDefineLocal(keySymbol.Index)); err != nil {
				return err
			}
		}
	}

	// assign value variable
	if stmt.Value.String() != "_" {
		valueSymbol := c.symbolTable.Define(stmt.Value.String())
		if itSymbol.Scope == ScopeGlobal {
			if _, err := c.emit(stmt, NewLoadGlobal(itSymbol.Index)); err != nil {
				return err
			}
		} else {
			if _, err := c.emit(stmt, NewLoadLocal(itSymbol.Index)); err != nil {
				return err
			}
		}
		if _, err := c.emit(stmt, NewIterValue()); err != nil {
			return err
		}
		if valueSymbol.Scope == ScopeGlobal {
			if _, err := c.emit(stmt, NewStoreGlobal(valueSymbol.Index)); err != nil {
				return err
			}
		} else {
			valueSymbol.LocalAssigned = true
			if _, err := c.emit(stmt, NewDefineLocal(valueSymbol.Index)); err != nil {
				return err
			}
		}
	}

	// body statement
	if err := c.CompileNode(stmt.Body); err != nil {
		c.leaveLoop()
		return err
	}

	c.leaveLoop()

	// post-body position
	postBodyPos := len(c.currentInstructions())
	if _, err = c.emit(stmt, NewAbortCheck()); err != nil {
		return err
	}

	// back to condition
	if _, err := c.emit(stmt, NewJump(preCondPos)); err != nil {
		return err
	}

	// post-statement position
	postStmtPos := len(c.currentInstructions())
	if err := c.changeJumpAddr(postCondPos, postStmtPos); err != nil {
		return err
	}

	// update all break/continue jump positions
	for _, pos := range loop.Breaks {
		if err := c.changeJumpAddr(pos, postStmtPos); err != nil {
			return err
		}
	}
	for _, pos := range loop.Continues {
		if err := c.changeJumpAddr(pos, postBodyPos); err != nil {
			return err
		}
	}

	return nil
}

func (c *Compiler) checkCyclicImports(node ast.Node, modulePath string) error {
	if c.modulePath == modulePath {
		return c.errorf(node, "cyclic module import: %s", modulePath)
	} else if c.parent != nil {
		return c.parent.checkCyclicImports(node, modulePath)
	}
	return nil
}

// compileModule compiles a module from source code and returns the compiled function of the module.
func (c *Compiler) compileModule(node ast.Node, modulePath string, src []byte, isFile bool) (core.CompiledFunction, error) {
	var cf core.CompiledFunction

	if err := c.checkCyclicImports(node, modulePath); err != nil {
		return cf, err
	}

	var exists bool
	cf, exists = c.loadCompiledModule(modulePath)
	if exists {
		return cf, nil
	}

	modFile := c.file.Set().AddFile(modulePath, -1, len(src))
	p := parser.NewParser(modFile, src, nil)
	f, err := p.ParseFile()
	if err != nil {
		return cf, err
	}
	file, err := c.Optimize(f)
	if err != nil {
		return cf, err
	}

	// inherit builtin functions
	symbolTable := NewSymbolTable()
	for _, sym := range c.symbolTable.BuiltinSymbols() {
		symbolTable.DefineBuiltin(sym.Index, sym.Name)
	}

	// no global scope for the module
	symbolTable = symbolTable.Fork(false)

	// compile module
	moduleCompiler := c.fork(modFile, modulePath, symbolTable, isFile)
	if err := moduleCompiler.CompileNode(file); err != nil {
		return cf, err
	}

	// code optimization
	if err := moduleCompiler.optimizeFunc(node); err != nil {
		return cf, err
	}

	t := moduleCompiler.Bytecode().MainFunction
	t.NumLocals = symbolTable.MaxSymbols()
	cf.Set(t.Instructions, t.Free, t.SourceMap, t.NumLocals, t.MaxStack, t.NumParameters, t.NamedResult, t.VarArgs)
	c.storeCompiledModule(modulePath, cf)
	return cf, nil
}

func (c *Compiler) loadCompiledModule(modulePath string) (cf core.CompiledFunction, ok bool) {
	if c.parent != nil {
		return c.parent.loadCompiledModule(modulePath)
	}
	cf, ok = c.compiledModules[modulePath]
	return
}

func (c *Compiler) storeCompiledModule(modulePath string, cf core.CompiledFunction) {
	if c.parent != nil {
		c.parent.storeCompiledModule(modulePath, cf)
	}
	c.compiledModules[modulePath] = cf
}

func (c *Compiler) enterLoop() *loop {
	loop := &loop{}
	c.loops = append(c.loops, loop)
	c.loopIndex++
	if c.trace != nil {
		c.printTrace("LOOPE", c.loopIndex)
	}
	return loop
}

func (c *Compiler) leaveLoop() {
	if c.trace != nil {
		c.printTrace("LOOPL", c.loopIndex)
	}
	c.loops = c.loops[:len(c.loops)-1]
	c.loopIndex--
}

func (c *Compiler) currentLoop() *loop {
	if c.loopIndex >= 0 {
		return c.loops[c.loopIndex]
	}
	return nil
}

func (c *Compiler) currentInstructions() bc.Instructions {
	return c.scopes[c.scopeIndex].Instructions
}

func (c *Compiler) currentSourceMap() map[int]core.Pos {
	return c.scopes[c.scopeIndex].SourceMap
}

func (c *Compiler) enterScope() {
	scope := compilationScope{
		SymbolInit: make(map[string]bool),
		SourceMap:  make(map[int]core.Pos),
	}
	c.scopes = append(c.scopes, scope)
	c.scopeIndex++
	c.symbolTable = c.symbolTable.Fork(false)
	if c.trace != nil {
		c.printTrace("SCOPE", c.scopeIndex)
	}
}

func (c *Compiler) leaveScope() (instructions bc.Instructions, sourceMap map[int]core.Pos) {
	instructions = c.currentInstructions()
	sourceMap = c.currentSourceMap()
	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--
	c.symbolTable = c.symbolTable.Parent(true)
	if c.trace != nil {
		c.printTrace("SCOPL", c.scopeIndex)
	}
	return
}

func (c *Compiler) fork(file *ast.SourceFile, modulePath string, symbolTable *SymbolTable, isFile bool) *Compiler {
	child := NewCompiler(c.oc, c.sb, file, symbolTable, c.allowedModules.ToSlice(), c.customModules, c.trace)
	child.modulePath = modulePath // module file path
	child.parent = c              // parent to set to current compiler
	child.assignmentMode = c.assignmentMode
	child.allowFileImport = c.allowFileImport
	child.importDir = c.importDir
	child.importFileExt = c.importFileExt
	if isFile && c.importDir != "" {
		child.importDir = filepath.Dir(modulePath)
	}
	return child
}

func (c *Compiler) errorf(node ast.Node, format string, args ...any) error {
	return &CompilerError{
		FileSet: c.file.Set(),
		Node:    node,
		Err:     fmt.Errorf(format, args...),
	}
}

func (c *Compiler) addStaticPrimitive(v core.Value) int {
	if c.parent != nil {
		return c.parent.addStaticPrimitive(v)
	}
	n := c.sb.AddPrimitive(v)
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("CONST %04d %s", n, v.String()))
	}
	return n
}

func (c *Compiler) addStaticDecimal(v dec128.Dec128) int {
	if c.parent != nil {
		return c.parent.addStaticDecimal(v)
	}
	n := c.sb.AddDecimal(v)
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("CONST %04d dec(%s)", n, v.String()))
	}
	return n
}

func (c *Compiler) addStaticString(v string) int {
	if c.parent != nil {
		return c.parent.addStaticString(v)
	}
	n := c.sb.AddString(v)
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("CONST %04d %q", n, v))
	}
	return n
}

func (c *Compiler) addStaticRunes(v core.Runes) int {
	if c.parent != nil {
		return c.parent.addStaticRunes(v)
	}
	n := c.sb.AddRunes(v)
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("CONST %04d u%s", n, strconv.Quote(string(v.Elements))))
	}
	return n
}

func (c *Compiler) addStaticBytes(v core.Bytes) int {
	if c.parent != nil {
		return c.parent.addStaticBytes(v)
	}
	n := c.sb.AddBytes(v)
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("CONST %04d b%s", n, strconv.Quote(string(v.Elements))))
	}
	return n
}

func (c *Compiler) addStaticTime(v time.Time) int {
	if c.parent != nil {
		return c.parent.addStaticTime(v)
	}
	n := c.sb.AddTime(v)
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("CONST %04d time(%q)", n, v.Format(time.RFC3339Nano)))
	}
	return n
}

func (c *Compiler) addStaticFormatSpec(v core.FormatSpec) int {
	if c.parent != nil {
		return c.parent.addStaticFormatSpec(v)
	}
	n := c.sb.AddFormatSpec(v)
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("CONST %04d format_spec(%q)", n, v.Text))
	}
	return n
}

func (c *Compiler) addStaticCompiledFunction(v core.CompiledFunction) int {
	if c.parent != nil {
		return c.parent.addStaticCompiledFunction(v)
	}
	n := c.sb.AddCompiledFunction(v)
	if c.trace != nil {
		var s string
		if v.VarArgs {
			s = fmt.Sprintf("<compiled-function/%d+>", v.NumParameters)
		} else {
			s = fmt.Sprintf("<compiled-function/%d>", v.NumParameters)
		}
		c.printTrace(fmt.Sprintf("CONST %04d %s", n, s))
	}
	return n
}

func (c *Compiler) addInstruction(i bc.Instruction) int {
	pos := len(c.scopes[c.scopeIndex].Instructions)
	c.scopes[c.scopeIndex].Instructions = append(c.scopes[c.scopeIndex].Instructions, i)
	return pos
}

func (c *Compiler) replaceInstruction(pos int, i bc.Instruction) (err error) {
	c.scopes[c.scopeIndex].Instructions[pos] = i
	if c.trace != nil {
		t, err := vm.FormatInstructions(c.scopes[c.scopeIndex].Instructions[pos:], pos)
		if err != nil {
			return err
		}
		c.printTrace(fmt.Sprintf("REPLC %s", t[0]))
	}
	return nil
}

func (c *Compiler) changeJumpAddr(pos int, addr int) error {
	c.scopes[c.scopeIndex].Instructions[pos].Op3 = uint32(addr)
	return nil
}

func (c *Compiler) emit(node ast.Node, i bc.Instruction) (int, error) {
	filePos := core.NoPos
	if node != nil {
		filePos = node.Pos()
	}

	pos := c.addInstruction(i)
	c.scopes[c.scopeIndex].SourceMap[pos] = filePos
	if c.trace != nil {
		t, err := vm.FormatInstructions(c.scopes[c.scopeIndex].Instructions[pos:], pos)
		if err != nil {
			return 0, err
		}
		c.printTrace(fmt.Sprintf("EMIT  %s", t[0]))
	}

	return pos, nil
}

func (c *Compiler) printTrace(a ...any) {
	const (
		dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
		n    = len(dots)
	)

	i := 2 * c.indent
	for i > n {
		_, _ = fmt.Fprint(c.trace, dots)
		i -= n
	}
	_, _ = fmt.Fprint(c.trace, dots[0:i])
	_, _ = fmt.Fprintln(c.trace, a...)
}

func (c *Compiler) getPathModule(moduleName string) (pathFile string, err error) {
	for _, ext := range c.importFileExt {
		nameFile := moduleName

		if !strings.HasSuffix(nameFile, ext) {
			nameFile += ext
		}

		pathFile, err = filepath.Abs(filepath.Join(c.importDir, nameFile))
		if err != nil {
			continue
		}

		// Check if file exists
		if _, err := os.Stat(pathFile); !errors.Is(err, os.ErrNotExist) {
			return pathFile, nil
		}
	}

	return "", fmt.Errorf("module '%s' not found at: %s", moduleName, pathFile)
}

func resolveAssignLHS(expr ast.Expression) (name string, selectors []ast.Expression) {
	switch term := expr.(type) {
	case *expression.Selector:
		name, selectors = resolveAssignLHS(term.Expr)
		selectors = append(selectors, term.Sel)
		return
	case *expression.Index:
		name, selectors = resolveAssignLHS(term.Expr)
		selectors = append(selectors, term.Index)
	case *expression.Identifier:
		name = term.Name
	}
	return
}

func tracec(c *Compiler, msg string) *Compiler {
	c.printTrace(msg, "{")
	c.indent++
	return c
}

func untracec(c *Compiler) {
	c.indent--
	c.printTrace("}")
}

// optimizeFunc performs some code-level optimization for the current function instructions. It also removes unreachable
// (dead code) instructions and adds "returns" instruction if needed.
func (c *Compiler) optimizeFunc(node ast.Node) (err error) {
	// any instructions between RETURN and the function end or instructions between RETURN and jump target position are
	// considered as unreachable.

	// pass 1. eliminate dead code
	// Only jump targets discovered from already-reachable instructions may revive code.
	// This avoids reviving unreachable blocks via jumps that themselves are in dead code.
	var newInsts bc.Instructions
	posMap := make(map[int]int) // old position to new position
	reachableDsts := make(map[int]bool)
	var deadCode bool
	err = iterateInstructions(c.scopes[c.scopeIndex].Instructions, func(pos int, ci bc.Instruction) (bool, error) {
		switch {
		case reachableDsts[pos]:
			deadCode = false
		case deadCode:
			return true, nil
		}

		posMap[pos] = len(newInsts)
		newInsts = append(newInsts, ci)

		switch ci.Op {
		case bc.Jump, bc.JumpFalsy, bc.AndJump, bc.OrJump:
			reachableDsts[int(ci.Op3)] = true
		case bc.Return:
			deadCode = true
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	// pass 2. update jump positions
	var li bc.Instruction
	var appendReturn bool
	endPos := len(c.scopes[c.scopeIndex].Instructions)
	newEndPost := len(newInsts)

	err = iterateInstructions(newInsts, func(pos int, ci bc.Instruction) (bool, error) {
		switch ci.Op {
		case bc.Jump, bc.JumpFalsy, bc.AndJump, bc.OrJump:
			newDst, ok := posMap[int(ci.Op3)]
			if ok {
				t := ci
				t.Op3 = uint32(newDst)
				newInsts[pos] = t
			} else if endPos == int(ci.Op3) {
				// there's a jump instruction that jumps to the end of function compiler should append "return".
				t := ci
				t.Op3 = uint32(newEndPost)
				newInsts[pos] = t
				appendReturn = true
			} else {
				return false, fmt.Errorf("invalid jump position: %d", newDst)
			}
		}
		li = ci
		return true, nil
	})
	if err != nil {
		return err
	}
	if li.Op != bc.Return {
		appendReturn = true
	}

	// pass 3. update source map
	newSourceMap := make(map[int]core.Pos)
	for pos, srcPos := range c.scopes[c.scopeIndex].SourceMap {
		newPos, ok := posMap[pos]
		if ok {
			newSourceMap[newPos] = srcPos
		}
	}
	c.scopes[c.scopeIndex].Instructions = newInsts
	c.scopes[c.scopeIndex].SourceMap = newSourceMap

	// append "return"
	if appendReturn {
		_, err = c.emit(node, NewReturn(false))
		if err != nil {
			return err
		}
	}

	return nil
}

func iterateInstructions(is bc.Instructions, fn func(int, bc.Instruction) (bool, error)) error {
	for pos, i := range is {
		r, err := fn(pos, i)
		if err != nil {
			return err
		}
		if !r {
			break
		}
	}
	return nil
}
