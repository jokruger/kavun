package compiler

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/token"
	"github.com/jokruger/kavun/vm"
)

// DefaultSourceFileExt is the default extension used to resolve file imports.
const DefaultSourceFileExt = ".kvn"

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

// AssignmentMode controls how plain '=' handles unresolved identifiers.
type AssignmentMode int

const (
	// AssignmentModeSmart declares a variable in current scope for unresolved '=' assignments.
	AssignmentModeSmart AssignmentMode = iota

	// AssignmentModeStrict requires variables to already exist for '=' assignments.
	AssignmentModeStrict
)

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
	alloc           *core.Arena
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
	assignmentMode  AssignmentMode
	trace           io.Writer
	indent          int
}

// New creates a Compiler.
func New(
	alloc *core.Arena,
	file *parser.SourceFile,
	symbolTable *vm.SymbolTable,
	constants []core.Value,
	modules vm.ModuleGetter,
	trace io.Writer,
) *Compiler {
	if alloc == nil {
		alloc = core.NewArena(nil)
	}

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
		if bf, ok := core.ResolveBuiltinFunction(fn); ok {
			symbolTable.DefineBuiltin(idx, bf.Name)
		}
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
		assignmentMode:  AssignmentModeSmart,
		trace:           trace,
		modules:         modules,
		compiledModules: make(map[string]*core.CompiledFunction),
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

// Compile compiles the AST node.
func (c *Compiler) Compile(node parser.Node) (err error) {
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
			if err = c.Compile(stmt); err != nil {
				return err
			}
		}

	case *parser.ExprStmt:
		if err = c.Compile(node.Expr); err != nil {
			return err
		}
		if _, err = c.emit(node, bc.OpPop); err != nil {
			return err
		}

	case *parser.IncDecStmt:
		op := token.AddAssign
		if node.Token == token.Dec {
			op = token.SubAssign
		}
		return c.compileAssign(node, []parser.Expr{node.Expr}, []parser.Expr{&parser.IntLit{Value: 1}}, op)

	case *parser.ParenExpr:
		if err = c.Compile(node.Expr); err != nil {
			return err
		}

	case *parser.BinaryExpr:
		if node.Token == token.LAnd || node.Token == token.LOr {
			return c.compileLogical(node)
		}

		if err = c.Compile(node.LHS); err != nil {
			return err
		}
		if err = c.Compile(node.RHS); err != nil {
			return err
		}

		switch node.Token {
		case token.Add:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.Add))
		case token.Sub:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.Sub))
		case token.Mul:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.Mul))
		case token.Quo:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.Quo))
		case token.Rem:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.Rem))
		case token.Greater:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.Greater))
		case token.GreaterEq:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.GreaterEq))
		case token.Less:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.Less))
		case token.LessEq:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.LessEq))
		case token.Equal:
			_, err = c.emit(node, bc.OpEqual)
		case token.NotEqual:
			_, err = c.emit(node, bc.OpNotEqual)
		case token.And:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.And))
		case token.Or:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.Or))
		case token.Xor:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.Xor))
		case token.AndNot:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.AndNot))
		case token.Shl:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.Shl))
		case token.Shr:
			_, err = c.emit(node, bc.OpBinaryOp, int(token.Shr))
		case token.In:
			_, err = c.emit(node, bc.OpContains)
		default:
			return c.errorf(node, "invalid binary operator: %s", node.Token.String())
		}
		if err != nil {
			return err
		}

	case *parser.IntLit:
		_, err = c.emit(node, bc.OpConstant, c.addConstant(core.IntValue(node.Value)))
		if err != nil {
			return err
		}

	case *parser.FloatLit:
		_, err = c.emit(node, bc.OpConstant, c.addConstant(core.FloatValue(node.Value)))
		if err != nil {
			return err
		}

	case *parser.DecimalLit:
		t := core.NewDecimalValue(node.Value)
		_, err = c.emit(node, bc.OpConstant, c.addConstant(t))
		if err != nil {
			return err
		}

	case *parser.BoolLit:
		if node.Value {
			_, err = c.emit(node, bc.OpTrue)
		} else {
			_, err = c.emit(node, bc.OpFalse)
		}
		if err != nil {
			return err
		}

	case *parser.StringLit:
		t := core.NewStringValue(node.Value)
		_, err = c.emit(node, bc.OpConstant, c.addConstant(t))
		if err != nil {
			return err
		}

	case *parser.RunesLit:
		t := core.NewRunesValue(node.Value, false)
		_, err = c.emit(node, bc.OpConstant, c.addConstant(t))
		if err != nil {
			return err
		}

	case *parser.FStringLit:
		if err = c.compileFString(node); err != nil {
			return err
		}

	case *parser.RuneLit:
		_, err = c.emit(node, bc.OpConstant, c.addConstant(core.RuneValue(node.Value)))
		if err != nil {
			return err
		}

	case *parser.UndefinedLit:
		_, err = c.emit(node, bc.OpNull)
		if err != nil {
			return err
		}

	case *parser.UnaryExpr:
		if err = c.Compile(node.Expr); err != nil {
			return err
		}

		switch node.Token {
		case token.Not:
			_, err = c.emit(node, bc.OpLNot)
		case token.Sub:
			_, err = c.emit(node, bc.OpMinus)
		case token.Xor:
			_, err = c.emit(node, bc.OpBComplement)
		case token.Add:
			// do nothing?
		default:
			return c.errorf(node, "invalid unary operator: %s", node.Token.String())
		}
		if err != nil {
			return err
		}

	case *parser.IfStmt:
		// open new symbol table for the statement
		c.symbolTable = c.symbolTable.Fork(true)
		defer func() {
			c.symbolTable = c.symbolTable.Parent(false)
		}()

		if node.Init != nil {
			if err = c.Compile(node.Init); err != nil {
				return err
			}
		}
		if err = c.Compile(node.Cond); err != nil {
			return err
		}

		// first jump placeholder
		jumpPos1, err := c.emit(node, bc.OpJumpFalsy, 0)
		if err != nil {
			return err
		}
		if err = c.Compile(node.Body); err != nil {
			return err
		}
		if node.Else != nil {
			// second jump placeholder
			jumpPos2, err := c.emit(node, bc.OpJump, 0)
			if err != nil {
				return err
			}

			// update first jump offset
			curPos := len(c.currentInstructions())
			if err = c.changeOperand(jumpPos1, curPos); err != nil {
				return err
			}
			if err = c.Compile(node.Else); err != nil {
				return err
			}

			// update second jump offset
			curPos = len(c.currentInstructions())
			if err = c.changeOperand(jumpPos2, curPos); err != nil {
				return err
			}
		} else {
			// update first jump offset
			curPos := len(c.currentInstructions())
			if err = c.changeOperand(jumpPos1, curPos); err != nil {
				return err
			}
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
			pos, err := c.emit(node, bc.OpJump, 0)
			if err != nil {
				return err
			}
			curLoop.Breaks = append(curLoop.Breaks, pos)
		case token.Continue:
			curLoop := c.currentLoop()
			if curLoop == nil {
				return c.errorf(node, "continue not allowed outside loop")
			}
			pos, err := c.emit(node, bc.OpJump, 0)
			if err != nil {
				return err
			}
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
			if err = c.Compile(stmt); err != nil {
				return err
			}
		}

	case *parser.AssignStmt:
		err = c.compileAssign(node, node.LHS, node.RHS, node.Token)
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
			_, err = c.emit(node, bc.OpGetGlobal, symbol.Index)
		case vm.ScopeLocal:
			_, err = c.emit(node, bc.OpGetLocal, symbol.Index)
		case vm.ScopeBuiltin:
			_, err = c.emit(node, bc.OpGetBuiltin, symbol.Index)
		case vm.ScopeFree:
			_, err = c.emit(node, bc.OpGetFree, symbol.Index)
		}
		if err != nil {
			return err
		}

	case *parser.ArrayLit:
		for _, elem := range node.Elements {
			if err = c.Compile(elem); err != nil {
				return err
			}
		}
		_, err = c.emit(node, bc.OpArray, len(node.Elements))
		if err != nil {
			return err
		}

	case *parser.RecordLit:
		for _, e := range node.Elements {
			// key
			t := core.NewStringValue(e.Key)
			_, err = c.emit(node, bc.OpConstant, c.addConstant(t))
			if err != nil {
				return err
			}
			// value
			if err = c.Compile(e.Value); err != nil {
				return err
			}
		}
		_, err = c.emit(node, bc.OpRecord, len(node.Elements)*2)
		if err != nil {
			return err
		}

	case *parser.SelectorExpr: // selector on RHS side
		if err = c.Compile(node.Expr); err != nil {
			return err
		}
		if err = c.Compile(node.Sel); err != nil {
			return err
		}
		_, err = c.emit(node, bc.OpSelect)
		if err != nil {
			return err
		}

	case *parser.IndexExpr:
		if err = c.Compile(node.Expr); err != nil {
			return err
		}
		if err = c.Compile(node.Index); err != nil {
			return err
		}
		_, err = c.emit(node, bc.OpIndex)
		if err != nil {
			return err
		}

	case *parser.SliceExpr:
		if err = c.Compile(node.Expr); err != nil {
			return err
		}
		if node.Low != nil {
			if err = c.Compile(node.Low); err != nil {
				return err
			}
		} else {
			_, err = c.emit(node, bc.OpNull)
			if err != nil {
				return err
			}
		}
		if node.High != nil {
			if err = c.Compile(node.High); err != nil {
				return err
			}
		} else {
			_, err = c.emit(node, bc.OpNull)
			if err != nil {
				return err
			}
		}
		if node.Step != nil {
			if err = c.Compile(node.Step); err != nil {
				return err
			}
			_, err = c.emit(node, bc.OpSliceIndexStep)
			if err != nil {
				return err
			}
		} else {
			_, err = c.emit(node, bc.OpSliceIndex)
			if err != nil {
				return err
			}
		}

	case *parser.FuncLit:
		c.enterScope()

		for _, p := range node.Type.Params.List {
			s := c.symbolTable.Define(p.Name)

			// function arguments is not assigned directly.
			s.LocalAssigned = true
		}

		// Optional named result: define a local right after parameters.
		// It is pre-initialized to undefined (locals start as undefined), can be referenced and assigned by name in the
		// body, and is returned automatically by bare `return` and by exit-after-recover.
		// Encoding: 0 means "no named result"; non-zero N means slot N-1.
		var namedResult int8
		if node.Type.Result != nil {
			rname := node.Type.Result.Name
			if rname == "_" {
				return c.errorf(node, "named result cannot be '_'")
			}
			// Disallow shadowing parameters.
			for _, p := range node.Type.Params.List {
				if p.Name == rname {
					return c.errorf(node, "named result %q conflicts with parameter name", rname)
				}
			}
			s := c.symbolTable.Define(rname)
			s.LocalAssigned = true
			if s.Index >= 127 {
				return c.errorf(node, "named result slot index too large: %d", s.Index)
			}
			namedResult = int8(s.Index) + 1
		}

		if err = c.Compile(node.Body); err != nil {
			return err
		}

		// code optimization
		if err := c.optimizeFunc(node); err != nil {
			return err
		}

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
					_, err = c.emit(node, bc.OpNull)
					if err != nil {
						return err
					}
					_, err = c.emit(node, bc.OpDefineLocal, s.Index)
					if err != nil {
						return err
					}
					s.LocalAssigned = true
				}
				_, err = c.emit(node, bc.OpGetLocalPtr, s.Index)
				if err != nil {
					return err
				}
			case vm.ScopeFree:
				_, err = c.emit(node, bc.OpGetFreePtr, s.Index)
				if err != nil {
					return err
				}
			}
		}

		l := len(node.Type.Params.List)
		if l > 127 {
			return c.errorf(node, "too many function parameters: %d (max: 127)", l)
		}
		compiledFunction := &core.CompiledFunction{
			Instructions:  instructions,
			NumLocals:     numLocals,
			MaxStack:      ComputeMaxStack(instructions),
			NumParameters: int8(l),
			VarArgs:       node.Type.Params.VarArgs,
			SourceMap:     sourceMap,
			NamedResult:   namedResult,
		}
		if len(freeSymbols) > 0 {
			_, err = c.emit(node, bc.OpClosure, c.addConstant(core.CompiledFunctionValue(compiledFunction)), len(freeSymbols))
			if err != nil {
				return err
			}
		} else {
			_, err = c.emit(node, bc.OpConstant, c.addConstant(core.CompiledFunctionValue(compiledFunction)))
			if err != nil {
				return err
			}
		}

	case *parser.ReturnStmt:
		if c.symbolTable.Parent(true) == nil {
			// outside the function
			return c.errorf(node, "return not allowed outside function")
		}

		if node.Result == nil {
			_, err = c.emit(node, bc.OpReturn, 0)
			if err != nil {
				return err
			}
		} else {
			if err = c.Compile(node.Result); err != nil {
				return err
			}
			_, err = c.emit(node, bc.OpReturn, 1)
			if err != nil {
				return err
			}
		}

	case *parser.DeferStmt:
		if c.symbolTable.Parent(true) == nil {
			return c.errorf(node, "defer not allowed outside function")
		}
		switch call := node.Call.(type) {
		case *parser.CallExpr:
			// Evaluate the callee then arguments so they capture current values (Go-style: arguments are evaluated
			// immediately; the call itself is delayed until function exit).
			if err = c.Compile(call.Func); err != nil {
				return err
			}
			for _, arg := range call.Args {
				if err = c.Compile(arg); err != nil {
					return err
				}
			}
			if call.Ellipsis.IsValid() {
				return c.errorf(node, "defer with spread argument is not supported")
			}
			_, err = c.emit(node, bc.OpDefer, len(call.Args))
			if err != nil {
				return err
			}
		case *parser.MethodCallExpr:
			if err = c.Compile(call.Object); err != nil {
				return err
			}
			for _, arg := range call.Args {
				if err = c.Compile(arg); err != nil {
					return err
				}
			}
			if call.Ellipsis.IsValid() {
				return c.errorf(node, "defer with spread argument is not supported")
			}
			methodIdx := c.addConstant(core.NewStringValue(call.MethodName))
			_, err = c.emit(node, bc.OpDeferMethod, methodIdx, len(call.Args))
			if err != nil {
				return err
			}
		default:
			return c.errorf(node, "defer expression must be a call")
		}

	case *parser.CallExpr:
		if err = c.Compile(node.Func); err != nil {
			return err
		}
		for _, arg := range node.Args {
			if err = c.Compile(arg); err != nil {
				return err
			}
		}
		ellipsis := 0
		if node.Ellipsis.IsValid() {
			ellipsis = 1
		}
		_, err = c.emit(node, bc.OpCall, len(node.Args), ellipsis)
		if err != nil {
			return err
		}

	case *parser.MethodCallExpr:
		if err = c.Compile(node.Object); err != nil {
			return err
		}
		for _, arg := range node.Args {
			if err = c.Compile(arg); err != nil {
				return err
			}
		}
		ellipsis := 0
		if node.Ellipsis.IsValid() {
			ellipsis = 1
		}
		t := core.NewStringValue(node.MethodName)
		methodIdx := c.addConstant(t)
		_, err = c.emit(node, bc.OpMethodCall, methodIdx, len(node.Args), ellipsis)
		if err != nil {
			return err
		}

	case *parser.ImportExpr:
		if node.ModuleName == "" {
			return c.errorf(node, "empty module name")
		}

		if mod := c.modules.Get(node.ModuleName); mod != nil {
			if idGetter, ok := c.modules.(interface {
				GetBuiltinModuleID(name string) (uint8, bool)
			}); ok {
				if modID, ok := idGetter.GetBuiltinModuleID(node.ModuleName); ok {
					_, err = c.emit(node, bc.OpImportBuiltinModule, int(modID))
					if err != nil {
						return err
					}
					break
				}
			}

			v, err := mod.Import(c.alloc, node.ModuleName)
			if err != nil {
				return err
			}

			switch v := v.(type) {
			case []byte: // module written in Kavun
				compiled, err := c.compileModule(node, node.ModuleName, v, false)
				if err != nil {
					return err
				}
				_, err = c.emit(node, bc.OpConstant, c.addConstant(core.CompiledFunctionValue(compiled)))
				if err != nil {
					return err
				}
				_, err = c.emit(node, bc.OpCall, 0, 0)
				if err != nil {
					return err
				}
			case core.Value: // builtin module
				_, err = c.emit(node, bc.OpConstant, c.addConstant(v))
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("invalid import value type: %T", v)
			}
		} else if c.allowFileImport {
			moduleName := node.ModuleName
			modulePath, err := c.getPathModule(moduleName)
			if err != nil {
				return c.errorf(node, "module file path error: %s", err.Error())
			}
			moduleSrc, err := os.ReadFile(modulePath)
			if err != nil {
				return c.errorf(node, "module file read error: %s", err.Error())
			}
			compiled, err := c.compileModule(node, modulePath, moduleSrc, true)
			if err != nil {
				return err
			}
			_, err = c.emit(node, bc.OpConstant, c.addConstant(core.CompiledFunctionValue(compiled)))
			if err != nil {
				return err
			}
			_, err = c.emit(node, bc.OpCall, 0, 0)
			if err != nil {
				return err
			}
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
		if err = c.Compile(node.Result); err != nil {
			return err
		}
		_, err = c.emit(node, bc.OpImmutable)
		if err != nil {
			return err
		}
		_, err = c.emit(node, bc.OpReturn, 1)
		if err != nil {
			return err
		}

	case *parser.ImmutableExpr:
		if err = c.Compile(node.Expr); err != nil {
			return err
		}
		_, err = c.emit(node, bc.OpImmutable)
		if err != nil {
			return err
		}

	case *parser.CondExpr:
		if err = c.Compile(node.Cond); err != nil {
			return err
		}

		// first jump placeholder
		jumpPos1, err := c.emit(node, bc.OpJumpFalsy, 0)
		if err != nil {
			return err
		}
		if err = c.Compile(node.True); err != nil {
			return err
		}

		// second jump placeholder
		jumpPos2, err := c.emit(node, bc.OpJump, 0)
		if err != nil {
			return err
		}

		// update first jump offset
		curPos := len(c.currentInstructions())
		if err = c.changeOperand(jumpPos1, curPos); err != nil {
			return err
		}
		if err = c.Compile(node.False); err != nil {
			return err
		}

		// update second jump offset
		curPos = len(c.currentInstructions())
		if err = c.changeOperand(jumpPos2, curPos); err != nil {
			return err
		}
	}

	return nil
}

// Bytecode returns a compiled bytecode.
func (c *Compiler) Bytecode() *vm.Bytecode {
	mainInsts := append(c.currentInstructions(), bc.OpSuspend)
	return &vm.Bytecode{
		FileSet: c.file.Set(),
		MainFunction: &core.CompiledFunction{
			Instructions: mainInsts,
			MaxStack:     ComputeMaxStack(mainInsts),
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
	// Builtins are pre-seeded global-like values. They may be shadowed in inner scopes (via :=) and reassigned at the
	// top level (via := or =, the latter under smart assignment mode). They have no addressable storage, so compound
	// assignments (+=, -=, etc.) remain disallowed.
	if exists && symbol.Scope == vm.ScopeBuiltin {
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

	if (op == token.Define || (op == token.Assign && numSel == 0 && c.assignmentMode == AssignmentModeSmart && !exists)) && !isFunc {
		symbol = c.symbolTable.Define(ident)
	}

	switch op {
	case token.AddAssign:
		c.emit(node, bc.OpBinaryOp, int(token.Add))
	case token.SubAssign:
		c.emit(node, bc.OpBinaryOp, int(token.Sub))
	case token.MulAssign:
		c.emit(node, bc.OpBinaryOp, int(token.Mul))
	case token.QuoAssign:
		c.emit(node, bc.OpBinaryOp, int(token.Quo))
	case token.RemAssign:
		c.emit(node, bc.OpBinaryOp, int(token.Rem))
	case token.AndAssign:
		c.emit(node, bc.OpBinaryOp, int(token.And))
	case token.OrAssign:
		c.emit(node, bc.OpBinaryOp, int(token.Or))
	case token.AndNotAssign:
		c.emit(node, bc.OpBinaryOp, int(token.AndNot))
	case token.XorAssign:
		c.emit(node, bc.OpBinaryOp, int(token.Xor))
	case token.ShlAssign:
		c.emit(node, bc.OpBinaryOp, int(token.Shl))
	case token.ShrAssign:
		c.emit(node, bc.OpBinaryOp, int(token.Shr))
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
			c.emit(node, bc.OpSetSelGlobal, symbol.Index, numSel)
		} else {
			c.emit(node, bc.OpSetGlobal, symbol.Index)
		}
	case vm.ScopeLocal:
		if numSel > 0 {
			c.emit(node, bc.OpSetSelLocal, symbol.Index, numSel)
		} else {
			if op == token.Define && !symbol.LocalAssigned {
				c.emit(node, bc.OpDefineLocal, symbol.Index)
			} else {
				c.emit(node, bc.OpSetLocal, symbol.Index)
			}
		}

		// mark the symbol as local-assigned
		symbol.LocalAssigned = true
	case vm.ScopeFree:
		if numSel > 0 {
			c.emit(node, bc.OpSetSelFree, symbol.Index, numSel)
		} else {
			c.emit(node, bc.OpSetFree, symbol.Index)
		}
	default:
		panic(fmt.Errorf("invalid assignment variable scope: %s",
			symbol.Scope))
	}
	return nil
}

func (c *Compiler) compileLogical(node *parser.BinaryExpr) (err error) {
	// left side term
	if err = c.Compile(node.LHS); err != nil {
		return err
	}

	// jump position
	var jumpPos int
	if node.Token == token.LAnd {
		jumpPos, err = c.emit(node, bc.OpAndJump, 0)
		if err != nil {
			return err
		}
	} else {
		jumpPos, err = c.emit(node, bc.OpOrJump, 0)
		if err != nil {
			return err
		}
	}

	// right side term
	if err = c.Compile(node.RHS); err != nil {
		return err
	}

	if err = c.changeOperand(jumpPos, len(c.currentInstructions())); err != nil {
		return err
	}

	return nil
}

func (c *Compiler) compileForStmt(stmt *parser.ForStmt) (err error) {
	c.symbolTable = c.symbolTable.Fork(true)
	defer func() {
		c.symbolTable = c.symbolTable.Parent(false)
	}()

	// init statement
	if stmt.Init != nil {
		if err = c.Compile(stmt.Init); err != nil {
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
		postCondPos, err = c.emit(stmt, bc.OpJumpFalsy, 0)
		if err != nil {
			return err
		}
	}

	// enter loop
	loop := c.enterLoop()

	// body statement
	if err = c.Compile(stmt.Body); err != nil {
		c.leaveLoop()
		return err
	}

	c.leaveLoop()

	// post-body position
	postBodyPos := len(c.currentInstructions())

	// post statement
	if stmt.Post != nil {
		if err = c.Compile(stmt.Post); err != nil {
			return err
		}
	}

	// back to condition
	c.emit(stmt, bc.OpJump, preCondPos)

	// post-statement position
	postStmtPos := len(c.currentInstructions())
	if postCondPos >= 0 {
		if err = c.changeOperand(postCondPos, postStmtPos); err != nil {
			return err
		}
	}

	// update all break/continue jump positions
	for _, pos := range loop.Breaks {
		if err = c.changeOperand(pos, postStmtPos); err != nil {
			return err
		}
	}
	for _, pos := range loop.Continues {
		if err = c.changeOperand(pos, postBodyPos); err != nil {
			return err
		}
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
	c.emit(stmt, bc.OpIteratorInit)
	if itSymbol.Scope == vm.ScopeGlobal {
		c.emit(stmt, bc.OpSetGlobal, itSymbol.Index)
	} else {
		c.emit(stmt, bc.OpDefineLocal, itSymbol.Index)
	}

	// pre-condition position
	preCondPos := len(c.currentInstructions())

	// condition
	//  :it.HasMore()
	if itSymbol.Scope == vm.ScopeGlobal {
		c.emit(stmt, bc.OpGetGlobal, itSymbol.Index)
	} else {
		c.emit(stmt, bc.OpGetLocal, itSymbol.Index)
	}
	c.emit(stmt, bc.OpIteratorNext)

	// condition jump position
	postCondPos, err := c.emit(stmt, bc.OpJumpFalsy, 0)
	if err != nil {
		return err
	}

	// enter loop
	loop := c.enterLoop()

	// assign key variable
	if stmt.Key.Name != "_" {
		keySymbol := c.symbolTable.Define(stmt.Key.Name)
		if itSymbol.Scope == vm.ScopeGlobal {
			c.emit(stmt, bc.OpGetGlobal, itSymbol.Index)
		} else {
			c.emit(stmt, bc.OpGetLocal, itSymbol.Index)
		}
		c.emit(stmt, bc.OpIteratorKey)
		if keySymbol.Scope == vm.ScopeGlobal {
			c.emit(stmt, bc.OpSetGlobal, keySymbol.Index)
		} else {
			keySymbol.LocalAssigned = true
			c.emit(stmt, bc.OpDefineLocal, keySymbol.Index)
		}
	}

	// assign value variable
	if stmt.Value.Name != "_" {
		valueSymbol := c.symbolTable.Define(stmt.Value.Name)
		if itSymbol.Scope == vm.ScopeGlobal {
			c.emit(stmt, bc.OpGetGlobal, itSymbol.Index)
		} else {
			c.emit(stmt, bc.OpGetLocal, itSymbol.Index)
		}
		c.emit(stmt, bc.OpIteratorValue)
		if valueSymbol.Scope == vm.ScopeGlobal {
			c.emit(stmt, bc.OpSetGlobal, valueSymbol.Index)
		} else {
			valueSymbol.LocalAssigned = true
			c.emit(stmt, bc.OpDefineLocal, valueSymbol.Index)
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
	c.emit(stmt, bc.OpJump, preCondPos)

	// post-statement position
	postStmtPos := len(c.currentInstructions())
	if err := c.changeOperand(postCondPos, postStmtPos); err != nil {
		return err
	}

	// update all break/continue jump positions
	for _, pos := range loop.Breaks {
		if err := c.changeOperand(pos, postStmtPos); err != nil {
			return err
		}
	}
	for _, pos := range loop.Continues {
		if err := c.changeOperand(pos, postBodyPos); err != nil {
			return err
		}
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
	if err := moduleCompiler.optimizeFunc(node); err != nil {
		return nil, err
	}
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
	child := New(c.alloc, file, symbolTable, nil, c.modules, c.trace)
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
		c.printTrace(fmt.Sprintf("CONST %04d %s", len(c.constants)-1, o.String(c.alloc)))
	}
	return len(c.constants) - 1
}

func (c *Compiler) addInstruction(b []byte) int {
	posNewIns := len(c.currentInstructions())
	c.scopes[c.scopeIndex].Instructions = append(c.currentInstructions(), b...)
	return posNewIns
}

func (c *Compiler) replaceInstruction(pos int, inst []byte) (err error) {
	copy(c.currentInstructions()[pos:], inst)
	if c.trace != nil {
		t, err := vm.FormatInstructions(c.scopes[c.scopeIndex].Instructions[pos:], pos)
		if err != nil {
			return err
		}
		c.printTrace(fmt.Sprintf("REPLC %s", t[0]))
	}
	return nil
}

func (c *Compiler) changeOperand(opPos int, operand ...int) error {
	op := c.currentInstructions()[opPos]
	inst, err := vm.MakeInstruction(op, operand...)
	if err != nil {
		return err
	}
	return c.replaceInstruction(opPos, inst)
}

func (c *Compiler) emit(node parser.Node, opcode bc.Opcode, operands ...int) (int, error) {
	filePos := core.NoPos
	if node != nil {
		filePos = node.Pos()
	}

	inst, err := vm.MakeInstruction(opcode, operands...)
	if err != nil {
		return 0, err
	}

	pos := c.addInstruction(inst)
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

func tracec(c *Compiler, msg string) *Compiler {
	c.printTrace(msg, "{")
	c.indent++
	return c
}

func untracec(c *Compiler) {
	c.indent--
	c.printTrace("}")
}
