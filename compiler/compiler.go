package compiler

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/parser/ast"
	"github.com/jokruger/kavun/parser/expression"
	"github.com/jokruger/kavun/parser/expression/composite"
	"github.com/jokruger/kavun/parser/expression/scalar"
	"github.com/jokruger/kavun/parser/statement"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/vm"
	"github.com/jokruger/set"
)

// DefaultSourceFileExt is the default extension used to resolve file imports.
const DefaultSourceFileExt = ".kvn"

// compilationScope represents a compiled instructions and the last two instructions that were emitted.
type compilationScope struct {
	Instructions bc.Instructions
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
	file            *parser.SourceFile
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
	file *parser.SourceFile,
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
func (c *Compiler) Compile(file *parser.SourceFile, src []byte, trace io.Writer) error {
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
	case *parser.File:
		for _, stmt := range node.Stmts {
			if err = c.CompileNode(stmt); err != nil {
				return err
			}
		}

	case *statement.Expression:
		if err = c.CompileNode(node.Expr); err != nil {
			return err
		}
		if _, err = c.emit(node, NewPop()); err != nil {
			return err
		}

	case *statement.IncDec:
		op := token.AddAssign
		if node.Token == token.Dec {
			op = token.SubAssign
		}
		return c.compileAssign(node, []ast.Expression{node.Expr}, []ast.Expression{&scalar.Int{Value: 1}}, op)

	case *expression.Parenthesis:
		if err = c.CompileNode(node.Expr); err != nil {
			return err
		}

	case *expression.Binary:
		if node.Token == token.LAnd || node.Token == token.LOr {
			return c.compileLogical(node)
		}

		if err = c.CompileNode(node.LHS); err != nil {
			return err
		}
		if err = c.CompileNode(node.RHS); err != nil {
			return err
		}

		switch node.Token {
		case token.Add:
			_, err = c.emit(node, NewBinaryOp(token.Add))
		case token.Sub:
			_, err = c.emit(node, NewBinaryOp(token.Sub))
		case token.Mul:
			_, err = c.emit(node, NewBinaryOp(token.Mul))
		case token.Quo:
			_, err = c.emit(node, NewBinaryOp(token.Quo))
		case token.Rem:
			_, err = c.emit(node, NewBinaryOp(token.Rem))
		case token.Greater:
			_, err = c.emit(node, NewBinaryOp(token.Greater))
		case token.GreaterEq:
			_, err = c.emit(node, NewBinaryOp(token.GreaterEq))
		case token.Less:
			_, err = c.emit(node, NewBinaryOp(token.Less))
		case token.LessEq:
			_, err = c.emit(node, NewBinaryOp(token.LessEq))
		case token.Equal:
			_, err = c.emit(node, NewEqual())
		case token.NotEqual:
			_, err = c.emit(node, NewNotEqual())
		case token.And:
			_, err = c.emit(node, NewBinaryOp(token.And))
		case token.Or:
			_, err = c.emit(node, NewBinaryOp(token.Or))
		case token.Xor:
			_, err = c.emit(node, NewBinaryOp(token.Xor))
		case token.AndNot:
			_, err = c.emit(node, NewBinaryOp(token.AndNot))
		case token.Shl:
			_, err = c.emit(node, NewBinaryOp(token.Shl))
		case token.Shr:
			_, err = c.emit(node, NewBinaryOp(token.Shr))
		case token.In:
			_, err = c.emit(node, NewContains())
		default:
			return c.errorf(node, "invalid binary operator: %s", node.Token.String())
		}
		if err != nil {
			return err
		}

	case *scalar.Int:
		if node.Value >= math.MinInt32 && node.Value <= math.MaxInt32 {
			_, err = c.emit(node, NewPushInt(int32(node.Value)))
		} else {
			i := c.addStaticPrimitive(core.IntValue(node.Value))
			_, err = c.emit(node, NewLoadStaticPrimitive(i))
		}
		if err != nil {
			return err
		}

	case *scalar.Float:
		i := c.addStaticPrimitive(core.FloatValue(node.Value))
		_, err = c.emit(node, NewLoadStaticPrimitive(i))
		if err != nil {
			return err
		}

	case *scalar.Decimal:
		i := c.addStaticDecimal(node.Value)
		_, err = c.emit(node, NewLoadStaticDecimal(i))
		if err != nil {
			return err
		}

	case *scalar.Bool:
		_, err = c.emit(node, NewPushBool(node.Value))
		if err != nil {
			return err
		}

	case *scalar.String:
		i := c.addStaticString(node.Value)
		_, err = c.emit(node, NewLoadStaticString(i))
		if err != nil {
			return err
		}

	case *scalar.Runes:
		var v core.Runes
		v.Set(node.Value)
		i := c.addStaticRunes(v)
		_, err = c.emit(node, NewLoadStaticRunes(i))
		if err != nil {
			return err
		}

	case *scalar.Bytes:
		var v core.Bytes
		v.Set(node.Value)
		i := c.addStaticBytes(v)
		_, err = c.emit(node, NewLoadStaticBytes(i))
		if err != nil {
			return err
		}

	case *scalar.Time:
		i := c.addStaticTime(node.Value)
		_, err = c.emit(node, NewLoadStaticTime(i))
		if err != nil {
			return err
		}

	case *expression.FString:
		if err = c.compileFString(node); err != nil {
			return err
		}

	case *scalar.Rune:
		_, err = c.emit(node, NewPushRune(node.Value))
		if err != nil {
			return err
		}

	case *scalar.Byte:
		_, err = c.emit(node, NewPushByte(node.Value))
		if err != nil {
			return err
		}

	case *expression.Undefined:
		_, err = c.emit(node, NewPushUndefined())
		if err != nil {
			return err
		}

	case *expression.Unary:
		if err = c.CompileNode(node.Expr); err != nil {
			return err
		}

		switch node.Token {
		case token.Not:
			_, err = c.emit(node, NewUnaryNot())
		case token.Sub:
			_, err = c.emit(node, NewUnaryNeg())
		case token.Xor:
			_, err = c.emit(node, NewUnaryBitNot())
		case token.Add:
			// do nothing?
		default:
			return c.errorf(node, "invalid unary operator: %s", node.Token.String())
		}
		if err != nil {
			return err
		}

	case *statement.If:
		// open new symbol table for the statement
		c.symbolTable = c.symbolTable.Fork(true)
		defer func() {
			c.symbolTable = c.symbolTable.Parent(false)
		}()

		if node.Init != nil {
			if err = c.CompileNode(node.Init); err != nil {
				return err
			}
		}
		if err = c.CompileNode(node.Cond); err != nil {
			return err
		}

		// first jump placeholder
		jumpPos1, err := c.emit(node, NewJumpFalsy(0))
		if err != nil {
			return err
		}
		if err = c.CompileNode(node.Body); err != nil {
			return err
		}
		if node.Else != nil {
			// second jump placeholder
			jumpPos2, err := c.emit(node, NewJump(0))
			if err != nil {
				return err
			}

			// update first jump offset
			curPos := len(c.currentInstructions())
			if err = c.changeJumpAddr(jumpPos1, curPos); err != nil {
				return err
			}
			if err = c.CompileNode(node.Else); err != nil {
				return err
			}

			// update second jump offset
			curPos = len(c.currentInstructions())
			if err = c.changeJumpAddr(jumpPos2, curPos); err != nil {
				return err
			}
		} else {
			// update first jump offset
			curPos := len(c.currentInstructions())
			if err = c.changeJumpAddr(jumpPos1, curPos); err != nil {
				return err
			}
		}

	case *statement.For:
		return c.compileForStmt(node)

	case *statement.ForIn:
		return c.compileForInStmt(node)

	case *statement.Branch:
		switch node.Token {
		case token.Break:
			curLoop := c.currentLoop()
			if curLoop == nil {
				return c.errorf(node, "break not allowed outside loop")
			}
			pos, err := c.emit(node, NewJump(0))
			if err != nil {
				return err
			}
			curLoop.Breaks = append(curLoop.Breaks, pos)
		case token.Continue:
			curLoop := c.currentLoop()
			if curLoop == nil {
				return c.errorf(node, "continue not allowed outside loop")
			}
			pos, err := c.emit(node, NewJump(0))
			if err != nil {
				return err
			}
			curLoop.Continues = append(curLoop.Continues, pos)
		default:
			panic(fmt.Errorf("invalid branch statement: %s", node.Token.String()))
		}

	case *statement.Block:
		if len(node.Stmts) == 0 {
			return nil
		}

		c.symbolTable = c.symbolTable.Fork(true)
		defer func() {
			c.symbolTable = c.symbolTable.Parent(false)
		}()

		for _, stmt := range node.Stmts {
			if err = c.CompileNode(stmt); err != nil {
				return err
			}
		}

	case *statement.Assign:
		err = c.compileAssign(node, node.LHS, node.RHS, node.Token)
		if err != nil {
			return err
		}

	case *ast.Identifier:
		symbol, _, ok := c.symbolTable.Resolve(node.Name, false)
		if !ok {
			return c.errorf(node, "unresolved reference '%s'", node.Name)
		}

		switch symbol.Scope {
		case ScopeGlobal:
			_, err = c.emit(node, NewLoadGlobal(symbol.Index))
		case ScopeLocal:
			_, err = c.emit(node, NewLoadLocal(symbol.Index))
		case ScopeBuiltin:
			_, err = c.emit(node, NewLoadBuiltinFunction(symbol.Index))
		case ScopeFree:
			_, err = c.emit(node, NewLoadFree(symbol.Index))
		}
		if err != nil {
			return err
		}

	case *composite.Array:
		for _, elem := range node.Elements {
			if err = c.CompileNode(elem); err != nil {
				return err
			}
		}
		_, err = c.emit(node, NewMakeArray(len(node.Elements)))
		if err != nil {
			return err
		}

	case *composite.Record:
		for _, e := range node.Elements {
			// key
			i := c.addStaticString(e.Key)
			_, err = c.emit(node, NewLoadStaticString(i))
			if err != nil {
				return err
			}
			// value
			if err = c.CompileNode(e.Value); err != nil {
				return err
			}
		}
		n := len(node.Elements) * 2
		_, err = c.emit(node, NewMakeRecord(n))
		if err != nil {
			return err
		}

	case *expression.Selector: // selector on RHS side
		if err = c.CompileNode(node.Expr); err != nil {
			return err
		}
		if err = c.CompileNode(node.Sel); err != nil {
			return err
		}
		_, err = c.emit(node, NewAccessSelector())
		if err != nil {
			return err
		}

	case *expression.Index:
		if err = c.CompileNode(node.Expr); err != nil {
			return err
		}
		if err = c.CompileNode(node.Index); err != nil {
			return err
		}
		_, err = c.emit(node, NewAccessIndex())
		if err != nil {
			return err
		}

	case *expression.Slice:
		if err = c.CompileNode(node.Expr); err != nil {
			return err
		}
		if node.Low != nil {
			if err = c.CompileNode(node.Low); err != nil {
				return err
			}
		} else {
			_, err = c.emit(node, NewPushUndefined())
			if err != nil {
				return err
			}
		}
		if node.High != nil {
			if err = c.CompileNode(node.High); err != nil {
				return err
			}
		} else {
			_, err = c.emit(node, NewPushUndefined())
			if err != nil {
				return err
			}
		}
		if node.Step != nil {
			if err = c.CompileNode(node.Step); err != nil {
				return err
			}
			_, err = c.emit(node, NewSliceStep())
			if err != nil {
				return err
			}
		} else {
			_, err = c.emit(node, NewSlice())
			if err != nil {
				return err
			}
		}

	case *expression.Function:
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
		var namedResult int
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
			namedResult = s.Index + 1
		}

		if err = c.CompileNode(node.Body); err != nil {
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
			case ScopeLocal:
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
					_, err = c.emit(node, NewPushUndefined())
					if err != nil {
						return err
					}
					_, err = c.emit(node, NewDefineLocal(s.Index))
					if err != nil {
						return err
					}
					s.LocalAssigned = true
				}
				_, err = c.emit(node, NewLoadLocalPtr(s.Index))
				if err != nil {
					return err
				}
			case ScopeFree:
				_, err = c.emit(node, NewLoadFreePtr(s.Index))
				if err != nil {
					return err
				}
			}
		}

		l := len(node.Type.Params.List)
		if l > 127 {
			return c.errorf(node, "too many function parameters: %d (max: 127)", l)
		}
		var cf core.CompiledFunction
		cf.Set(instructions, nil, sourceMap, numLocals, ComputeMaxStack(instructions), l, namedResult, node.Type.Params.VarArgs)
		if len(freeSymbols) > 0 {
			i := c.addStaticCompiledFunction(cf)
			_, err = c.emit(node, NewMakeClosure(i, len(freeSymbols)))
			if err != nil {
				return err
			}
		} else {
			i := c.addStaticCompiledFunction(cf)
			_, err = c.emit(node, NewLoadStaticCompiledFunction(i))
			if err != nil {
				return err
			}
		}

	case *statement.Return:
		if c.symbolTable.Parent(true) == nil {
			// outside the function
			return c.errorf(node, "return not allowed outside function")
		}

		if node.Result == nil {
			_, err = c.emit(node, NewReturn(false))
			if err != nil {
				return err
			}
		} else {
			if err = c.CompileNode(node.Result); err != nil {
				return err
			}
			_, err = c.emit(node, NewReturn(true))
			if err != nil {
				return err
			}
		}

	case *statement.Defer:
		if c.symbolTable.Parent(true) == nil {
			return c.errorf(node, "defer not allowed outside function")
		}
		switch call := node.Call.(type) {
		case *expression.Call:
			// Evaluate the callee then arguments so they capture current values (Go-style: arguments are evaluated
			// immediately; the call itself is delayed until function exit).
			if err = c.CompileNode(call.Func); err != nil {
				return err
			}
			for _, arg := range call.Args {
				if err = c.CompileNode(arg); err != nil {
					return err
				}
			}
			if call.Ellipsis.IsValid() {
				return c.errorf(node, "defer with spread argument is not supported")
			}
			_, err = c.emit(node, NewDefer(len(call.Args)))
			if err != nil {
				return err
			}
		case *expression.MethodCall:
			if err = c.CompileNode(call.Object); err != nil {
				return err
			}
			for _, arg := range call.Args {
				if err = c.CompileNode(arg); err != nil {
					return err
				}
			}
			if call.Ellipsis.IsValid() {
				return c.errorf(node, "defer with spread argument is not supported")
			}
			i := c.addStaticString(call.MethodName)
			_, err = c.emit(node, NewDeferMethod(i, len(call.Args)))
			if err != nil {
				return err
			}
		default:
			return c.errorf(node, "defer expression must be a call")
		}

	case *expression.Call:
		if err = c.CompileNode(node.Func); err != nil {
			return err
		}
		for _, arg := range node.Args {
			if err = c.CompileNode(arg); err != nil {
				return err
			}
		}
		_, err = c.emit(node, NewCallFunction(len(node.Args), node.Ellipsis.IsValid()))
		if err != nil {
			return err
		}

	case *expression.MethodCall:
		if err = c.CompileNode(node.Object); err != nil {
			return err
		}
		for _, arg := range node.Args {
			if err = c.CompileNode(arg); err != nil {
				return err
			}
		}
		i := c.addStaticString(node.MethodName)
		_, err = c.emit(node, NewCallMethod(i, len(node.Args), node.Ellipsis.IsValid()))
		if err != nil {
			return err
		}

	case *expression.Import:
		if node.ModuleName == "" {
			return c.errorf(node, "empty module name")
		}

		if mod, ok := stdlib.GetModuleID(node.ModuleName); ok { // builtin module
			if c.allowedModules != nil && !c.allowedModules.Contains(node.ModuleName) {
				return c.errorf(node, "module '%s' is not allowed to import", node.ModuleName)
			}

			_, err = c.emit(node, NewImportBuiltinModule(int(mod)))
			if err != nil {
				return err
			}
		} else if src, ok := c.customModules[node.ModuleName]; ok { // user module from custom source
			compiled, err := c.compileModule(node, node.ModuleName, src, true)
			if err != nil {
				return err
			}
			i := c.addStaticCompiledFunction(compiled)
			_, err = c.emit(node, NewLoadStaticCompiledFunction(i))
			if err != nil {
				return err
			}
			_, err = c.emit(node, NewCallFunction(0, false))
			if err != nil {
				return err
			}
		} else if c.allowFileImport { // user module from local file
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
			i := c.addStaticCompiledFunction(compiled)
			_, err = c.emit(node, NewLoadStaticCompiledFunction(i))
			if err != nil {
				return err
			}
			_, err = c.emit(node, NewCallFunction(0, false))
			if err != nil {
				return err
			}
		} else {
			return c.errorf(node, "module '%s' not found", node.ModuleName)
		}

	case *statement.Export:
		// export statement must be in top-level scope
		if c.scopeIndex != 0 {
			return c.errorf(node, "export not allowed inside function")
		}

		// export statement is simply ignore when compiling non-module code
		if c.parent == nil {
			break
		}
		if err = c.CompileNode(node.Result); err != nil {
			return err
		}
		_, err = c.emit(node, NewImmutable())
		if err != nil {
			return err
		}
		_, err = c.emit(node, NewReturn(true))
		if err != nil {
			return err
		}

	case *expression.Immutable:
		if err = c.CompileNode(node.Expr); err != nil {
			return err
		}
		_, err = c.emit(node, NewImmutable())
		if err != nil {
			return err
		}

	case *expression.Ternary:
		if err = c.CompileNode(node.Cond); err != nil {
			return err
		}

		// first jump placeholder
		jumpPos1, err := c.emit(node, NewJumpFalsy(0))
		if err != nil {
			return err
		}
		if err = c.CompileNode(node.True); err != nil {
			return err
		}

		// second jump placeholder
		jumpPos2, err := c.emit(node, NewJump(0))
		if err != nil {
			return err
		}

		// update first jump offset
		curPos := len(c.currentInstructions())
		if err = c.changeJumpAddr(jumpPos1, curPos); err != nil {
			return err
		}
		if err = c.CompileNode(node.False); err != nil {
			return err
		}

		// update second jump offset
		curPos = len(c.currentInstructions())
		if err = c.changeJumpAddr(jumpPos2, curPos); err != nil {
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
	if stmt.Key.Name != "_" {
		keySymbol := c.symbolTable.Define(stmt.Key.Name)
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
	if stmt.Value.Name != "_" {
		valueSymbol := c.symbolTable.Define(stmt.Value.Name)
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

func (c *Compiler) fork(file *parser.SourceFile, modulePath string, symbolTable *SymbolTable, isFile bool) *Compiler {
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
	case *ast.Identifier:
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
