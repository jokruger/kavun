package gs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/token"
	"github.com/jokruger/gs/vm"
)

// compilationScope represents a compiled instructions and the last two instructions that were emitted.
type compilationScope struct {
	Instructions []byte
	SymbolInit   map[string]bool
	SourceMap    map[int]core.Pos
}

// loop represents a loop construct that the compiler uses to track the current loop.
type loop struct {
	Continues []int
	Breaks    []int
}

// CompilerError represents a compiler error.
type CompilerError struct {
	FileSet *parser.SourceFileSet
	Node    parser.Node
	Err     error
}

func (e *CompilerError) Error() string {
	filePos := e.FileSet.Position(e.Node.Pos())
	return fmt.Sprintf("Compile Error: %s\n\tat %s", e.Err.Error(), filePos)
}

// Compiler compiles the AST into a bytecode.
type Compiler struct {
	alloc           core.Allocator
	file            *parser.SourceFile
	parent          *Compiler
	modulePath      string
	importDir       string
	importFileExt   []string
	constants       []core.Value
	symbolTable     *vm.SymbolTable
	scopes          []compilationScope
	scopeIndex      int
	modules         vm.ModuleGetter
	compiledModules map[string]*core.CompiledFunction
	allowFileImport bool
	loops           []*loop
	loopIndex       int
	trace           io.Writer
	indent          int
}

// NewCompiler creates a Compiler.
func NewCompiler(
	alloc core.Allocator,
	file *parser.SourceFile,
	symbolTable *vm.SymbolTable,
	constants []core.Value,
	modules vm.ModuleGetter,
	trace io.Writer,
) *Compiler {
	mainScope := compilationScope{
		SymbolInit: make(map[string]bool),
		SourceMap:  make(map[int]core.Pos),
	}

	// symbol table
	if symbolTable == nil {
		symbolTable = vm.NewSymbolTable()
	}

	// add builtin functions to the symbol table
	for idx, fn := range vm.BuiltinFuncs {
		// it is safe to cast type because we know that all builtin functions are *value.BuiltinFunction objects
		symbolTable.DefineBuiltin(idx, fn.BuiltinFunction().Name)
	}

	// builtin modules
	if modules == nil {
		modules = vm.NewModuleMap()
	}

	return &Compiler{
		alloc:           alloc,
		file:            file,
		symbolTable:     symbolTable,
		constants:       constants,
		scopes:          []compilationScope{mainScope},
		scopeIndex:      0,
		loopIndex:       -1,
		trace:           trace,
		modules:         modules,
		compiledModules: make(map[string]*core.CompiledFunction),
		importFileExt:   []string{SourceFileExtDefault},
	}
}

// Compile compiles the AST node.
func (c *Compiler) Compile(node parser.Node) error {
	if c.trace != nil {
		if node != nil {
			defer untracec(tracec(c, fmt.Sprintf("%s (%s)",
				node.String(), reflect.TypeOf(node).Elem().Name())))
		} else {
			defer untracec(tracec(c, "<nil>"))
		}
	}

	switch node := node.(type) {
	case *parser.File:
		for _, stmt := range node.Stmts {
			if err := c.Compile(stmt); err != nil {
				return err
			}
		}

	case *parser.ExprStmt:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		c.emit(node, core.OpPop)

	case *parser.IncDecStmt:
		op := token.AddAssign
		if node.Token == token.Dec {
			op = token.SubAssign
		}
		return c.compileAssign(node, []parser.Expr{node.Expr},
			[]parser.Expr{&parser.IntLit{Value: 1}}, op)

	case *parser.ParenExpr:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}

	case *parser.BinaryExpr:
		if node.Token == token.LAnd || node.Token == token.LOr {
			return c.compileLogical(node)
		}

		if err := c.Compile(node.LHS); err != nil {
			return err
		}
		if err := c.Compile(node.RHS); err != nil {
			return err
		}

		switch node.Token {
		case token.Add:
			c.emit(node, core.OpBinaryOp, int(token.Add))
		case token.Sub:
			c.emit(node, core.OpBinaryOp, int(token.Sub))
		case token.Mul:
			c.emit(node, core.OpBinaryOp, int(token.Mul))
		case token.Quo:
			c.emit(node, core.OpBinaryOp, int(token.Quo))
		case token.Rem:
			c.emit(node, core.OpBinaryOp, int(token.Rem))
		case token.Greater:
			c.emit(node, core.OpBinaryOp, int(token.Greater))
		case token.GreaterEq:
			c.emit(node, core.OpBinaryOp, int(token.GreaterEq))
		case token.Less:
			c.emit(node, core.OpBinaryOp, int(token.Less))
		case token.LessEq:
			c.emit(node, core.OpBinaryOp, int(token.LessEq))
		case token.Equal:
			c.emit(node, core.OpEqual)
		case token.NotEqual:
			c.emit(node, core.OpNotEqual)
		case token.And:
			c.emit(node, core.OpBinaryOp, int(token.And))
		case token.Or:
			c.emit(node, core.OpBinaryOp, int(token.Or))
		case token.Xor:
			c.emit(node, core.OpBinaryOp, int(token.Xor))
		case token.AndNot:
			c.emit(node, core.OpBinaryOp, int(token.AndNot))
		case token.Shl:
			c.emit(node, core.OpBinaryOp, int(token.Shl))
		case token.Shr:
			c.emit(node, core.OpBinaryOp, int(token.Shr))
		default:
			return c.errorf(node, "invalid binary operator: %s",
				node.Token.String())
		}

	case *parser.IntLit:
		c.emit(node, core.OpConstant, c.addConstant(core.IntValue(node.Value)))

	case *parser.FloatLit:
		c.emit(node, core.OpConstant, c.addConstant(core.FloatValue(node.Value)))

	case *parser.BoolLit:
		if node.Value {
			c.emit(node, core.OpTrue)
		} else {
			c.emit(node, core.OpFalse)
		}

	case *parser.StringLit:
		if len(node.Value) > core.MaxStringLen {
			return c.error(node, errs.NewStringLimitError("string literal compiler"))
		}
		c.emit(node, core.OpConstant, c.addConstant(c.alloc.NewStringValue(node.Value)))

	case *parser.CharLit:
		c.emit(node, core.OpConstant, c.addConstant(core.CharValue(node.Value)))

	case *parser.UndefinedLit:
		c.emit(node, core.OpNull)

	case *parser.UnaryExpr:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}

		switch node.Token {
		case token.Not:
			c.emit(node, core.OpLNot)
		case token.Sub:
			c.emit(node, core.OpMinus)
		case token.Xor:
			c.emit(node, core.OpBComplement)
		case token.Add:
			// do nothing?
		default:
			return c.errorf(node, "invalid unary operator: %s", node.Token.String())
		}

	case *parser.IfStmt:
		// open new symbol table for the statement
		c.symbolTable = c.symbolTable.Fork(true)
		defer func() {
			c.symbolTable = c.symbolTable.Parent(false)
		}()

		if node.Init != nil {
			if err := c.Compile(node.Init); err != nil {
				return err
			}
		}
		if err := c.Compile(node.Cond); err != nil {
			return err
		}

		// first jump placeholder
		jumpPos1 := c.emit(node, core.OpJumpFalsy, 0)
		if err := c.Compile(node.Body); err != nil {
			return err
		}
		if node.Else != nil {
			// second jump placeholder
			jumpPos2 := c.emit(node, core.OpJump, 0)

			// update first jump offset
			curPos := len(c.currentInstructions())
			c.changeOperand(jumpPos1, curPos)
			if err := c.Compile(node.Else); err != nil {
				return err
			}

			// update second jump offset
			curPos = len(c.currentInstructions())
			c.changeOperand(jumpPos2, curPos)
		} else {
			// update first jump offset
			curPos := len(c.currentInstructions())
			c.changeOperand(jumpPos1, curPos)
		}

	case *parser.ForStmt:
		return c.compileForStmt(node)

	case *parser.ForInStmt:
		return c.compileForInStmt(node)

	case *parser.BranchStmt:
		switch node.Token {
		case token.Break:
			curLoop := c.currentLoop()
			if curLoop == nil {
				return c.errorf(node, "break not allowed outside loop")
			}
			pos := c.emit(node, core.OpJump, 0)
			curLoop.Breaks = append(curLoop.Breaks, pos)
		case token.Continue:
			curLoop := c.currentLoop()
			if curLoop == nil {
				return c.errorf(node, "continue not allowed outside loop")
			}
			pos := c.emit(node, core.OpJump, 0)
			curLoop.Continues = append(curLoop.Continues, pos)
		default:
			panic(fmt.Errorf("invalid branch statement: %s", node.Token.String()))
		}

	case *parser.BlockStmt:
		if len(node.Stmts) == 0 {
			return nil
		}

		c.symbolTable = c.symbolTable.Fork(true)
		defer func() {
			c.symbolTable = c.symbolTable.Parent(false)
		}()

		for _, stmt := range node.Stmts {
			if err := c.Compile(stmt); err != nil {
				return err
			}
		}

	case *parser.AssignStmt:
		err := c.compileAssign(node, node.LHS, node.RHS, node.Token)
		if err != nil {
			return err
		}

	case *parser.Ident:
		symbol, _, ok := c.symbolTable.Resolve(node.Name, false)
		if !ok {
			return c.errorf(node, "unresolved reference '%s'", node.Name)
		}

		switch symbol.Scope {
		case vm.ScopeGlobal:
			c.emit(node, core.OpGetGlobal, symbol.Index)
		case vm.ScopeLocal:
			c.emit(node, core.OpGetLocal, symbol.Index)
		case vm.ScopeBuiltin:
			c.emit(node, core.OpGetBuiltin, symbol.Index)
		case vm.ScopeFree:
			c.emit(node, core.OpGetFree, symbol.Index)
		}

	case *parser.ArrayLit:
		for _, elem := range node.Elements {
			if err := c.Compile(elem); err != nil {
				return err
			}
		}
		c.emit(node, core.OpArray, len(node.Elements))

	case *parser.RecordLit:
		for _, elt := range node.Elements {
			// key
			if len(elt.Key) > core.MaxStringLen {
				return c.error(node, errs.NewStringLimitError("map literal key compiler"))
			}
			c.emit(node, core.OpConstant, c.addConstant(c.alloc.NewStringValue(elt.Key)))

			// value
			if err := c.Compile(elt.Value); err != nil {
				return err
			}
		}
		c.emit(node, core.OpRecord, len(node.Elements)*2)

	case *parser.SelectorExpr: // selector on RHS side
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		if err := c.Compile(node.Sel); err != nil {
			return err
		}
		c.emit(node, core.OpSelect)

	case *parser.IndexExpr:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		if err := c.Compile(node.Index); err != nil {
			return err
		}
		c.emit(node, core.OpIndex)

	case *parser.SliceExpr:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		if node.Low != nil {
			if err := c.Compile(node.Low); err != nil {
				return err
			}
		} else {
			c.emit(node, core.OpNull)
		}
		if node.High != nil {
			if err := c.Compile(node.High); err != nil {
				return err
			}
		} else {
			c.emit(node, core.OpNull)
		}
		c.emit(node, core.OpSliceIndex)

	case *parser.FuncLit:
		c.enterScope()

		for _, p := range node.Type.Params.List {
			s := c.symbolTable.Define(p.Name)

			// function arguments is not assigned directly.
			s.LocalAssigned = true
		}

		if err := c.Compile(node.Body); err != nil {
			return err
		}

		// code optimization
		c.optimizeFunc(node)

		freeSymbols := c.symbolTable.FreeSymbols()
		numLocals := c.symbolTable.MaxSymbols()
		instructions, sourceMap := c.leaveScope()

		for _, s := range freeSymbols {
			switch s.Scope {
			case vm.ScopeLocal:
				if !s.LocalAssigned {
					// Here, the closure is capturing a local variable that's
					// not yet assigned its value. One example is a local
					// recursive function:
					//
					//   func() {
					//     foo := func(x) {
					//       // ..
					//       return foo(x-1)
					//     }
					//   }
					//
					// which translate into
					//
					//   0000 GETL    0
					//   0002 CLOSURE ?     1
					//   0006 DEFL    0
					//
					// . So the local variable (0) is being captured before
					// it's assigned the value.
					//
					// Solution is to transform the code into something like
					// this:
					//
					//   func() {
					//     foo := undefined
					//     foo = func(x) {
					//       // ..
					//       return foo(x-1)
					//     }
					//   }
					//
					// that is equivalent to
					//
					//   0000 NULL
					//   0001 DEFL    0
					//   0003 GETL    0
					//   0005 CLOSURE ?     1
					//   0009 SETL    0
					//
					c.emit(node, core.OpNull)
					c.emit(node, core.OpDefineLocal, s.Index)
					s.LocalAssigned = true
				}
				c.emit(node, core.OpGetLocalPtr, s.Index)
			case vm.ScopeFree:
				c.emit(node, core.OpGetFreePtr, s.Index)
			}
		}

		compiledFunction := &core.CompiledFunction{
			Instructions:  instructions,
			NumLocals:     numLocals,
			NumParameters: len(node.Type.Params.List),
			VarArgs:       node.Type.Params.VarArgs,
			SourceMap:     sourceMap,
		}
		if len(freeSymbols) > 0 {
			c.emit(node, core.OpClosure, c.addConstant(core.CompiledFunctionValue(compiledFunction)), len(freeSymbols))
		} else {
			c.emit(node, core.OpConstant, c.addConstant(core.CompiledFunctionValue(compiledFunction)))
		}

	case *parser.ReturnStmt:
		if c.symbolTable.Parent(true) == nil {
			// outside the function
			return c.errorf(node, "return not allowed outside function")
		}

		if node.Result == nil {
			c.emit(node, core.OpReturn, 0)
		} else {
			if err := c.Compile(node.Result); err != nil {
				return err
			}
			c.emit(node, core.OpReturn, 1)
		}

	case *parser.CallExpr:
		if err := c.Compile(node.Func); err != nil {
			return err
		}
		for _, arg := range node.Args {
			if err := c.Compile(arg); err != nil {
				return err
			}
		}
		ellipsis := 0
		if node.Ellipsis.IsValid() {
			ellipsis = 1
		}
		c.emit(node, core.OpCall, len(node.Args), ellipsis)

	case *parser.MethodCallExpr:
		if err := c.Compile(node.Object); err != nil {
			return err
		}
		for _, arg := range node.Args {
			if err := c.Compile(arg); err != nil {
				return err
			}
		}
		ellipsis := 0
		if node.Ellipsis.IsValid() {
			ellipsis = 1
		}
		methodIdx := c.addConstant(c.alloc.NewStringValue(node.MethodName))
		c.emit(node, core.OpMethodCall, methodIdx, len(node.Args), ellipsis)

	case *parser.ImportExpr:
		if node.ModuleName == "" {
			return c.errorf(node, "empty module name")
		}

		if mod := c.modules.Get(node.ModuleName); mod != nil {
			v, err := mod.Import(c.alloc, node.ModuleName)
			if err != nil {
				return err
			}

			switch v := v.(type) {
			case []byte: // module written in Gs
				compiled, err := c.compileModule(node, node.ModuleName, v, false)
				if err != nil {
					return err
				}
				c.emit(node, core.OpConstant, c.addConstant(core.CompiledFunctionValue(compiled)))
				c.emit(node, core.OpCall, 0, 0)
			case core.Value: // builtin module
				c.emit(node, core.OpConstant, c.addConstant(v))
			default:
				panic(fmt.Errorf("invalid import value type: %T", v))
			}
		} else if c.allowFileImport {
			moduleName := node.ModuleName

			modulePath, err := c.getPathModule(moduleName)
			if err != nil {
				return c.errorf(node, "module file path error: %s",
					err.Error())
			}

			moduleSrc, err := os.ReadFile(modulePath)
			if err != nil {
				return c.errorf(node, "module file read error: %s",
					err.Error())
			}

			compiled, err := c.compileModule(node, modulePath, moduleSrc, true)
			if err != nil {
				return err
			}
			c.emit(node, core.OpConstant, c.addConstant(core.CompiledFunctionValue(compiled)))
			c.emit(node, core.OpCall, 0, 0)
		} else {
			return c.errorf(node, "module '%s' not found", node.ModuleName)
		}

	case *parser.ExportStmt:
		// export statement must be in top-level scope
		if c.scopeIndex != 0 {
			return c.errorf(node, "export not allowed inside function")
		}

		// export statement is simply ignore when compiling non-module code
		if c.parent == nil {
			break
		}
		if err := c.Compile(node.Result); err != nil {
			return err
		}
		c.emit(node, core.OpImmutable)
		c.emit(node, core.OpReturn, 1)

	case *parser.ErrorExpr:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		c.emit(node, core.OpError)

	case *parser.ImmutableExpr:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		c.emit(node, core.OpImmutable)

	case *parser.CondExpr:
		if err := c.Compile(node.Cond); err != nil {
			return err
		}

		// first jump placeholder
		jumpPos1 := c.emit(node, core.OpJumpFalsy, 0)
		if err := c.Compile(node.True); err != nil {
			return err
		}

		// second jump placeholder
		jumpPos2 := c.emit(node, core.OpJump, 0)

		// update first jump offset
		curPos := len(c.currentInstructions())
		c.changeOperand(jumpPos1, curPos)
		if err := c.Compile(node.False); err != nil {
			return err
		}

		// update second jump offset
		curPos = len(c.currentInstructions())
		c.changeOperand(jumpPos2, curPos)
	}

	return nil
}

// Bytecode returns a compiled bytecode.
func (c *Compiler) Bytecode() *vm.Bytecode {
	return &vm.Bytecode{
		FileSet: c.file.Set(),
		MainFunction: &core.CompiledFunction{
			Instructions: append(c.currentInstructions(), core.OpSuspend),
			SourceMap:    c.currentSourceMap(),
		},
		Constants: c.constants,
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
// Use this method if you want other source file extension than ".gs".
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

func (c *Compiler) compileAssign(node parser.Node, lhs, rhs []parser.Expr, op token.Token) error {
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

	_, isFunc := rhs[0].(*parser.FuncLit)
	symbol, depth, exists := c.symbolTable.Resolve(ident, false)
	if op == token.Define {
		if depth == 0 && exists {
			return c.errorf(node, "'%s' redeclared in this block", ident)
		}
		if isFunc {
			symbol = c.symbolTable.Define(ident)
		}
	} else {
		if !exists {
			return c.errorf(node, "unresolved reference '%s'", ident)
		}
	}

	// +=, -=, *=, /=
	if op != token.Assign && op != token.Define {
		if err := c.Compile(lhs[0]); err != nil {
			return err
		}
	}

	// compile RHSs
	for _, expr := range rhs {
		if err := c.Compile(expr); err != nil {
			return err
		}
	}

	if op == token.Define && !isFunc {
		symbol = c.symbolTable.Define(ident)
	}

	switch op {
	case token.AddAssign:
		c.emit(node, core.OpBinaryOp, int(token.Add))
	case token.SubAssign:
		c.emit(node, core.OpBinaryOp, int(token.Sub))
	case token.MulAssign:
		c.emit(node, core.OpBinaryOp, int(token.Mul))
	case token.QuoAssign:
		c.emit(node, core.OpBinaryOp, int(token.Quo))
	case token.RemAssign:
		c.emit(node, core.OpBinaryOp, int(token.Rem))
	case token.AndAssign:
		c.emit(node, core.OpBinaryOp, int(token.And))
	case token.OrAssign:
		c.emit(node, core.OpBinaryOp, int(token.Or))
	case token.AndNotAssign:
		c.emit(node, core.OpBinaryOp, int(token.AndNot))
	case token.XorAssign:
		c.emit(node, core.OpBinaryOp, int(token.Xor))
	case token.ShlAssign:
		c.emit(node, core.OpBinaryOp, int(token.Shl))
	case token.ShrAssign:
		c.emit(node, core.OpBinaryOp, int(token.Shr))
	}

	// compile selector expressions (right to left)
	for i := numSel - 1; i >= 0; i-- {
		if err := c.Compile(selectors[i]); err != nil {
			return err
		}
	}

	switch symbol.Scope {
	case vm.ScopeGlobal:
		if numSel > 0 {
			c.emit(node, core.OpSetSelGlobal, symbol.Index, numSel)
		} else {
			c.emit(node, core.OpSetGlobal, symbol.Index)
		}
	case vm.ScopeLocal:
		if numSel > 0 {
			c.emit(node, core.OpSetSelLocal, symbol.Index, numSel)
		} else {
			if op == token.Define && !symbol.LocalAssigned {
				c.emit(node, core.OpDefineLocal, symbol.Index)
			} else {
				c.emit(node, core.OpSetLocal, symbol.Index)
			}
		}

		// mark the symbol as local-assigned
		symbol.LocalAssigned = true
	case vm.ScopeFree:
		if numSel > 0 {
			c.emit(node, core.OpSetSelFree, symbol.Index, numSel)
		} else {
			c.emit(node, core.OpSetFree, symbol.Index)
		}
	default:
		panic(fmt.Errorf("invalid assignment variable scope: %s",
			symbol.Scope))
	}
	return nil
}

func (c *Compiler) compileLogical(node *parser.BinaryExpr) error {
	// left side term
	if err := c.Compile(node.LHS); err != nil {
		return err
	}

	// jump position
	var jumpPos int
	if node.Token == token.LAnd {
		jumpPos = c.emit(node, core.OpAndJump, 0)
	} else {
		jumpPos = c.emit(node, core.OpOrJump, 0)
	}

	// right side term
	if err := c.Compile(node.RHS); err != nil {
		return err
	}

	c.changeOperand(jumpPos, len(c.currentInstructions()))
	return nil
}

func (c *Compiler) compileForStmt(stmt *parser.ForStmt) error {
	c.symbolTable = c.symbolTable.Fork(true)
	defer func() {
		c.symbolTable = c.symbolTable.Parent(false)
	}()

	// init statement
	if stmt.Init != nil {
		if err := c.Compile(stmt.Init); err != nil {
			return err
		}
	}

	// pre-condition position
	preCondPos := len(c.currentInstructions())

	// condition expression
	postCondPos := -1
	if stmt.Cond != nil {
		if err := c.Compile(stmt.Cond); err != nil {
			return err
		}
		// condition jump position
		postCondPos = c.emit(stmt, core.OpJumpFalsy, 0)
	}

	// enter loop
	loop := c.enterLoop()

	// body statement
	if err := c.Compile(stmt.Body); err != nil {
		c.leaveLoop()
		return err
	}

	c.leaveLoop()

	// post-body position
	postBodyPos := len(c.currentInstructions())

	// post statement
	if stmt.Post != nil {
		if err := c.Compile(stmt.Post); err != nil {
			return err
		}
	}

	// back to condition
	c.emit(stmt, core.OpJump, preCondPos)

	// post-statement position
	postStmtPos := len(c.currentInstructions())
	if postCondPos >= 0 {
		c.changeOperand(postCondPos, postStmtPos)
	}

	// update all break/continue jump positions
	for _, pos := range loop.Breaks {
		c.changeOperand(pos, postStmtPos)
	}
	for _, pos := range loop.Continues {
		c.changeOperand(pos, postBodyPos)
	}
	return nil
}

func (c *Compiler) compileForInStmt(stmt *parser.ForInStmt) error {
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
	if err := c.Compile(stmt.Iterable); err != nil {
		return err
	}
	c.emit(stmt, core.OpIteratorInit)
	if itSymbol.Scope == vm.ScopeGlobal {
		c.emit(stmt, core.OpSetGlobal, itSymbol.Index)
	} else {
		c.emit(stmt, core.OpDefineLocal, itSymbol.Index)
	}

	// pre-condition position
	preCondPos := len(c.currentInstructions())

	// condition
	//  :it.HasMore()
	if itSymbol.Scope == vm.ScopeGlobal {
		c.emit(stmt, core.OpGetGlobal, itSymbol.Index)
	} else {
		c.emit(stmt, core.OpGetLocal, itSymbol.Index)
	}
	c.emit(stmt, core.OpIteratorNext)

	// condition jump position
	postCondPos := c.emit(stmt, core.OpJumpFalsy, 0)

	// enter loop
	loop := c.enterLoop()

	// assign key variable
	if stmt.Key.Name != "_" {
		keySymbol := c.symbolTable.Define(stmt.Key.Name)
		if itSymbol.Scope == vm.ScopeGlobal {
			c.emit(stmt, core.OpGetGlobal, itSymbol.Index)
		} else {
			c.emit(stmt, core.OpGetLocal, itSymbol.Index)
		}
		c.emit(stmt, core.OpIteratorKey)
		if keySymbol.Scope == vm.ScopeGlobal {
			c.emit(stmt, core.OpSetGlobal, keySymbol.Index)
		} else {
			keySymbol.LocalAssigned = true
			c.emit(stmt, core.OpDefineLocal, keySymbol.Index)
		}
	}

	// assign value variable
	if stmt.Value.Name != "_" {
		valueSymbol := c.symbolTable.Define(stmt.Value.Name)
		if itSymbol.Scope == vm.ScopeGlobal {
			c.emit(stmt, core.OpGetGlobal, itSymbol.Index)
		} else {
			c.emit(stmt, core.OpGetLocal, itSymbol.Index)
		}
		c.emit(stmt, core.OpIteratorValue)
		if valueSymbol.Scope == vm.ScopeGlobal {
			c.emit(stmt, core.OpSetGlobal, valueSymbol.Index)
		} else {
			valueSymbol.LocalAssigned = true
			c.emit(stmt, core.OpDefineLocal, valueSymbol.Index)
		}
	}

	// body statement
	if err := c.Compile(stmt.Body); err != nil {
		c.leaveLoop()
		return err
	}

	c.leaveLoop()

	// post-body position
	postBodyPos := len(c.currentInstructions())

	// back to condition
	c.emit(stmt, core.OpJump, preCondPos)

	// post-statement position
	postStmtPos := len(c.currentInstructions())
	c.changeOperand(postCondPos, postStmtPos)

	// update all break/continue jump positions
	for _, pos := range loop.Breaks {
		c.changeOperand(pos, postStmtPos)
	}
	for _, pos := range loop.Continues {
		c.changeOperand(pos, postBodyPos)
	}
	return nil
}

func (c *Compiler) checkCyclicImports(
	node parser.Node,
	modulePath string,
) error {
	if c.modulePath == modulePath {
		return c.errorf(node, "cyclic module import: %s", modulePath)
	} else if c.parent != nil {
		return c.parent.checkCyclicImports(node, modulePath)
	}
	return nil
}

func (c *Compiler) compileModule(node parser.Node, modulePath string, src []byte, isFile bool) (*core.CompiledFunction, error) {
	if err := c.checkCyclicImports(node, modulePath); err != nil {
		return nil, err
	}

	compiledModule, exists := c.loadCompiledModule(modulePath)
	if exists {
		return compiledModule, nil
	}

	modFile := c.file.Set().AddFile(modulePath, -1, len(src))
	p := parser.NewParser(modFile, src, nil)
	file, err := p.ParseFile()
	if err != nil {
		return nil, err
	}

	// inherit builtin functions
	symbolTable := vm.NewSymbolTable()
	for _, sym := range c.symbolTable.BuiltinSymbols() {
		symbolTable.DefineBuiltin(sym.Index, sym.Name)
	}

	// no global scope for the module
	symbolTable = symbolTable.Fork(false)

	// compile module
	moduleCompiler := c.fork(modFile, modulePath, symbolTable, isFile)
	if err := moduleCompiler.Compile(file); err != nil {
		return nil, err
	}

	// code optimization
	moduleCompiler.optimizeFunc(node)
	compiledFunc := moduleCompiler.Bytecode().MainFunction
	compiledFunc.NumLocals = symbolTable.MaxSymbols()
	c.storeCompiledModule(modulePath, compiledFunc)
	return compiledFunc, nil
}

func (c *Compiler) loadCompiledModule(modulePath string) (mod *core.CompiledFunction, ok bool) {
	if c.parent != nil {
		return c.parent.loadCompiledModule(modulePath)
	}
	mod, ok = c.compiledModules[modulePath]
	return
}

func (c *Compiler) storeCompiledModule(modulePath string, module *core.CompiledFunction) {
	if c.parent != nil {
		c.parent.storeCompiledModule(modulePath, module)
	}
	c.compiledModules[modulePath] = module
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

func (c *Compiler) currentInstructions() []byte {
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

func (c *Compiler) leaveScope() (instructions []byte, sourceMap map[int]core.Pos) {
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

func (c *Compiler) fork(file *parser.SourceFile, modulePath string, symbolTable *vm.SymbolTable, isFile bool) *Compiler {
	child := NewCompiler(c.alloc, file, symbolTable, nil, c.modules, c.trace)
	child.modulePath = modulePath // module file path
	child.parent = c              // parent to set to current compiler
	child.allowFileImport = c.allowFileImport
	child.importDir = c.importDir
	child.importFileExt = c.importFileExt
	if isFile && c.importDir != "" {
		child.importDir = filepath.Dir(modulePath)
	}
	return child
}

func (c *Compiler) error(node parser.Node, err error) error {
	return &CompilerError{
		FileSet: c.file.Set(),
		Node:    node,
		Err:     err,
	}
}

func (c *Compiler) errorf(node parser.Node, format string, args ...any) error {
	return &CompilerError{
		FileSet: c.file.Set(),
		Node:    node,
		Err:     fmt.Errorf(format, args...),
	}
}

func (c *Compiler) addConstant(o core.Value) int {
	if c.parent != nil {
		// module compilers will use their parent's constants array
		return c.parent.addConstant(o)
	}
	c.constants = append(c.constants, o)
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("CONST %04d %s", len(c.constants)-1, o.String()))
	}
	return len(c.constants) - 1
}

func (c *Compiler) addInstruction(b []byte) int {
	posNewIns := len(c.currentInstructions())
	c.scopes[c.scopeIndex].Instructions = append(c.currentInstructions(), b...)
	return posNewIns
}

func (c *Compiler) replaceInstruction(pos int, inst []byte) {
	copy(c.currentInstructions()[pos:], inst)
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("REPLC %s", vm.FormatInstructions(c.scopes[c.scopeIndex].Instructions[pos:], pos)[0]))
	}
}

func (c *Compiler) changeOperand(opPos int, operand ...int) {
	op := c.currentInstructions()[opPos]
	inst := vm.MakeInstruction(op, operand...)
	c.replaceInstruction(opPos, inst)
}

// optimizeFunc performs some code-level optimization for the current function
// instructions. It also removes unreachable (dead code) instructions and adds
// "returns" instruction if needed.
func (c *Compiler) optimizeFunc(node parser.Node) {
	// any instructions between RETURN and the function end
	// or instructions between RETURN and jump target position
	// are considered as unreachable.

	// pass 1. identify all jump destinations
	dsts := make(map[int]bool)
	iterateInstructions(c.scopes[c.scopeIndex].Instructions,
		func(pos int, opcode core.Opcode, operands []int) bool {
			switch opcode {
			case core.OpJump, core.OpJumpFalsy,
				core.OpAndJump, core.OpOrJump:
				dsts[operands[0]] = true
			}
			return true
		})

	// pass 2. eliminate dead code
	var newInsts []byte
	posMap := make(map[int]int) // old position to new position
	var dstIdx int
	var deadCode bool
	iterateInstructions(c.scopes[c.scopeIndex].Instructions,
		func(pos int, opcode core.Opcode, operands []int) bool {
			switch {
			case dsts[pos]:
				dstIdx++
				deadCode = false
			case opcode == core.OpReturn:
				if deadCode {
					return true
				}
				deadCode = true
			case deadCode:
				return true
			}
			posMap[pos] = len(newInsts)
			newInsts = append(newInsts, vm.MakeInstruction(opcode, operands...)...)
			return true
		})

	// pass 3. update jump positions
	var lastOp core.Opcode
	var appendReturn bool
	endPos := len(c.scopes[c.scopeIndex].Instructions)
	newEndPost := len(newInsts)

	iterateInstructions(newInsts,
		func(pos int, opcode core.Opcode, operands []int) bool {
			switch opcode {
			case core.OpJump, core.OpJumpFalsy, core.OpAndJump,
				core.OpOrJump:
				newDst, ok := posMap[operands[0]]
				if ok {
					copy(newInsts[pos:], vm.MakeInstruction(opcode, newDst))
				} else if endPos == operands[0] {
					// there's a jump instruction that jumps to the end of
					// function compiler should append "return".
					copy(newInsts[pos:], vm.MakeInstruction(opcode, newEndPost))
					appendReturn = true
				} else {
					panic(fmt.Errorf("invalid jump position: %d", newDst))
				}
			}
			lastOp = opcode
			return true
		})
	if lastOp != core.OpReturn {
		appendReturn = true
	}

	// pass 4. update source map
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
		c.emit(node, core.OpReturn, 0)
	}
}

func (c *Compiler) emit(node parser.Node, opcode core.Opcode, operands ...int) int {
	filePos := core.NoPos
	if node != nil {
		filePos = node.Pos()
	}

	inst := vm.MakeInstruction(opcode, operands...)
	pos := c.addInstruction(inst)
	c.scopes[c.scopeIndex].SourceMap[pos] = filePos
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("EMIT  %s", vm.FormatInstructions(c.scopes[c.scopeIndex].Instructions[pos:], pos)[0]))
	}
	return pos
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

func resolveAssignLHS(expr parser.Expr) (name string, selectors []parser.Expr) {
	switch term := expr.(type) {
	case *parser.SelectorExpr:
		name, selectors = resolveAssignLHS(term.Expr)
		selectors = append(selectors, term.Sel)
		return
	case *parser.IndexExpr:
		name, selectors = resolveAssignLHS(term.Expr)
		selectors = append(selectors, term.Index)
	case *parser.Ident:
		name = term.Name
	}
	return
}

func iterateInstructions(b []byte, fn func(pos int, opcode core.Opcode, operands []int) bool) {
	for i := 0; i < len(b); i++ {
		numOperands := core.OpcodeOperands[b[i]]
		operands, read := core.ReadOperands(numOperands, b[i+1:])
		if !fn(i, b[i], operands) {
			break
		}
		i += read
	}
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
