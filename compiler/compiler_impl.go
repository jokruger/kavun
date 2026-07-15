package compiler

import (
	"fmt"
	"math"
	"os"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/ast/expression"
	"github.com/jokruger/kavun/ast/expression/composite"
	"github.com/jokruger/kavun/ast/expression/scalar"
	"github.com/jokruger/kavun/ast/statement"
	"github.com/jokruger/kavun/core"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/stdlib"
)

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

func (c *Compiler) compileExpression(node ast.Expression) (err error) {
	switch node := node.(type) {
	case *scalar.Bool:
		_, err = c.emit(node, NewPushBool(node.Value))
		if err != nil {
			return err
		}

	case *scalar.Byte:
		_, err = c.emit(node, NewPushByte(node.Value))
		if err != nil {
			return err
		}

	case *scalar.Rune:
		_, err = c.emit(node, NewPushRune(node.Value))
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

	case *scalar.Time:
		i := c.addStaticTime(node.Value)
		_, err = c.emit(node, NewLoadStaticTime(i))
		if err != nil {
			return err
		}

	case *scalar.String:
		i := c.addStaticString(node.Value)
		_, err = c.emit(node, NewLoadStaticString(i))
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

	case *scalar.Runes:
		var v core.Runes
		v.Set(node.Value)
		i := c.addStaticRunes(v)
		_, err = c.emit(node, NewLoadStaticRunes(i))
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

	case *expression.Parenthesis:
		if err = c.CompileNode(node.Expr); err != nil {
			return err
		}

	case *expression.FString:
		if err = c.compileFString(node); err != nil {
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
		return c.compileSliceExpr(node)

	case *expression.Function:
		return c.compileFunctionExpr(node)

	case *expression.Import:
		return c.compileImportExpr(node)

	case *expression.Immutable:
		if err = c.CompileNode(node.Expr); err != nil {
			return err
		}
		_, err = c.emit(node, NewImmutable())
		if err != nil {
			return err
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

	case *expression.Unary:
		return c.compileUnaryExpr(node)

	case *expression.Binary:
		return c.compileBinaryExpr(node)

	case *expression.Ternary:
		return c.compileTernaryExpr(node)

	case *expression.Identifier:
		return c.compileIdentifier(node)

	case *expression.Undefined:
		_, err = c.emit(node, NewPushUndefined())
		if err != nil {
			return err
		}

	default:
		return c.errorf(node, "unknown expression type: %T", node)
	}

	return nil
}

func (c *Compiler) compileStatement(node ast.Statement) (err error) {
	switch node := node.(type) {

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

	case *statement.If:
		return c.compileIfStmt(node)

	case *statement.For:
		return c.compileForStmt(node)

	case *statement.ForIn:
		return c.compileForInStmt(node)

	case *statement.Branch:
		return c.compileBranchStmt(node)

	case *statement.Block:
		return c.compileBlockStmt(node)

	case *statement.Assign:
		err = c.compileAssign(node, node.LHS, node.RHS, node.Token)
		if err != nil {
			return err
		}

	case *statement.Return:
		return c.compileReturnStmt(node)

	case *statement.Defer:
		return c.compileDeferStmt(node)

	case *statement.Export:
		return c.compileExportStmt(node)

	case *statement.Empty:
		// do nothing

	default:
		return c.errorf(node, "unknown statement type: %T", node)
	}

	return nil
}

func (c *Compiler) compileUnaryExpr(node *expression.Unary) (err error) {
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

	return err
}

func (c *Compiler) compileBinaryExpr(node *expression.Binary) (err error) {
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

	return err
}

func (c *Compiler) compileTernaryExpr(node *expression.Ternary) (err error) {
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

	return nil
}

func (c *Compiler) compileIfStmt(node *statement.If) (err error) {
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

	return nil
}

func (c *Compiler) compileBranchStmt(node *statement.Branch) (err error) {
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

	return nil
}

func (c *Compiler) compileBlockStmt(node *statement.Block) (err error) {
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

	return nil
}

func (c *Compiler) compileReturnStmt(node *statement.Return) (err error) {
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

	return nil
}

func (c *Compiler) compileDeferStmt(node *statement.Defer) (err error) {
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

	return nil
}

func (c *Compiler) compileExportStmt(node *statement.Export) (err error) {
	// export statement must be in top-level scope
	if c.scopeIndex != 0 {
		return c.errorf(node, "export not allowed inside function")
	}

	// export statement is simply ignore when compiling non-module code
	if c.parent == nil {
		return nil
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

	return nil
}

func (c *Compiler) compileIdentifier(node *expression.Identifier) (err error) {
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

	return nil
}

func (c *Compiler) compileSliceExpr(node *expression.Slice) (err error) {
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

	return nil
}

func (c *Compiler) compileFunctionExpr(node *expression.Function) (err error) {
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

	return nil
}

func (c *Compiler) compileImportExpr(node *expression.Import) (err error) {
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

	return nil
}
