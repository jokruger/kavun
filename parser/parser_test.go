package parser_test

import (
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/token"
)

var rta = core.NewArena(nil)
var testFileSet = parser.NewFileSet()

type scanResult struct {
	Token   token.Token
	Literal string
	Line    int
	Column  int
}

func scanExpect(t *testing.T, input string, mode parser.ScanMode, expected ...scanResult) {
	testFile := testFileSet.AddFile("test", -1, len(input))

	s := parser.NewScanner(
		testFile,
		[]byte(input),
		func(_ parser.SourceFilePos, msg string) { require.Fail(t, msg) },
		mode,
	)

	for idx, e := range expected {
		tok, literal, pos := s.Scan()

		filePos := testFile.Position(pos)

		require.Equal(t, rta, e.Token, tok, "[%d] expected: %s, actual: %s", idx, e.Token.String(), tok.String())
		require.Equal(t, rta, e.Literal, literal)
		require.Equal(t, rta, e.Line, filePos.Line)
		require.Equal(t, rta, e.Column, filePos.Column)
	}

	tok, _, _ := s.Scan()
	require.Equal(t, rta, token.EOF, tok, "more tokens left")
	require.Equal(t, rta, 0, s.ErrorCount())
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := 1
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
		}
	}
	return n
}

type pfn func(int, int) core.Pos            // position conversion function
type expectedFn func(pos pfn) []parser.Stmt // callback function to return expected results

type parseTracer struct {
	out []string
}

func (o *parseTracer) Write(p []byte) (n int, err error) {
	o.out = append(o.out, string(p))
	return len(p), nil
}

func expectParse(t *testing.T, input string, fn expectedFn) {
	testFileSet := parser.NewFileSet()
	testFile := testFileSet.AddFile("test", -1, len(input))

	var ok bool
	defer func() {
		if !ok {
			// print trace
			tr := &parseTracer{}
			p := parser.NewParser(testFile, []byte(input), tr)
			actual, _ := p.ParseFile()
			if actual != nil {
				t.Logf("Parsed:\n%s", actual.String())
			}
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	p := parser.NewParser(testFile, []byte(input), nil)
	actual, err := p.ParseFile()
	require.NoError(t, err)

	expected := fn(func(line, column int) core.Pos {
		return core.Pos(int(testFile.LineStart(line)) + (column - 1))
	})
	require.Equal(t, rta, len(expected), len(actual.Stmts))

	for i := 0; i < len(expected); i++ {
		equalStmt(t, expected[i], actual.Stmts[i])
	}

	ok = true
}

func expectParseError(t *testing.T, input string) {
	testFileSet := parser.NewFileSet()
	testFile := testFileSet.AddFile("test", -1, len(input))

	var ok bool
	defer func() {
		if !ok {
			// print trace
			tr := &parseTracer{}
			p := parser.NewParser(testFile, []byte(input), tr)
			_, _ = p.ParseFile()
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	p := parser.NewParser(testFile, []byte(input), nil)
	_, err := p.ParseFile()
	require.Error(t, err)
	ok = true
}

func expectParseString(t *testing.T, input, expected string) {
	var ok bool
	defer func() {
		if !ok {
			// print trace
			tr := &parseTracer{}
			_, _ = parseSource("test", []byte(input), tr)
			t.Logf("Trace:\n%s", strings.Join(tr.out, ""))
		}
	}()

	actual, err := parseSource("test", []byte(input), nil)
	require.NoError(t, err)
	require.Equal(t, rta, expected, actual.String())
	ok = true
}

func stmts(s ...parser.Stmt) []parser.Stmt {
	return s
}

func exprStmt(x parser.Expr) *parser.ExprStmt {
	return &parser.ExprStmt{Expr: x}
}

func assignStmt(lhs, rhs []parser.Expr, token token.Token, pos core.Pos) *parser.AssignStmt {
	return &parser.AssignStmt{LHS: lhs, RHS: rhs, Token: token, TokenPos: pos}
}

func emptyStmt(implicit bool, pos core.Pos) *parser.EmptyStmt {
	return &parser.EmptyStmt{Implicit: implicit, Semicolon: pos}
}

func returnStmt(pos core.Pos, result parser.Expr) *parser.ReturnStmt {
	return &parser.ReturnStmt{Result: result, ReturnPos: pos}
}

func forStmt(
	init parser.Stmt,
	cond parser.Expr,
	post parser.Stmt,
	body *parser.BlockStmt,
	pos core.Pos,
) *parser.ForStmt {
	return &parser.ForStmt{
		Cond: cond, Init: init, Post: post, Body: body, ForPos: pos,
	}
}

func forInStmt(
	key, value *parser.Ident,
	seq parser.Expr,
	body *parser.BlockStmt,
	pos core.Pos,
) *parser.ForInStmt {
	return &parser.ForInStmt{
		Key: key, Value: value, Iterable: seq, Body: body, ForPos: pos,
	}
}

func ifStmt(
	init parser.Stmt,
	cond parser.Expr,
	body *parser.BlockStmt,
	elseStmt parser.Stmt,
	pos core.Pos,
) *parser.IfStmt {
	return &parser.IfStmt{
		Init: init, Cond: cond, Body: body, Else: elseStmt, IfPos: pos,
	}
}

func incDecStmt(
	expr parser.Expr,
	tok token.Token,
	pos core.Pos,
) *parser.IncDecStmt {
	return &parser.IncDecStmt{Expr: expr, Token: tok, TokenPos: pos}
}

func funcType(params *parser.IdentList, pos core.Pos) *parser.FuncType {
	return &parser.FuncType{Params: params, FuncPos: pos}
}

func blockStmt(lbrace, rbrace core.Pos, list ...parser.Stmt) *parser.BlockStmt {
	return &parser.BlockStmt{Stmts: list, LBrace: lbrace, RBrace: rbrace}
}

func ident(name string, pos core.Pos) *parser.Ident {
	return &parser.Ident{Name: name, NamePos: pos}
}

func identList(
	opening, closing core.Pos,
	varArgs bool,
	list ...*parser.Ident,
) *parser.IdentList {
	return &parser.IdentList{
		VarArgs: varArgs, List: list, LParen: opening, RParen: closing,
	}
}

func binaryExpr(
	x, y parser.Expr,
	op token.Token,
	pos core.Pos,
) *parser.BinaryExpr {
	return &parser.BinaryExpr{LHS: x, RHS: y, Token: op, TokenPos: pos}
}

func condExpr(
	cond, trueExpr, falseExpr parser.Expr,
	questionPos, colonPos core.Pos,
) *parser.CondExpr {
	return &parser.CondExpr{
		Cond: cond, True: trueExpr, False: falseExpr,
		QuestionPos: questionPos, ColonPos: colonPos,
	}
}

func unaryExpr(x parser.Expr, op token.Token, pos core.Pos) *parser.UnaryExpr {
	return &parser.UnaryExpr{Expr: x, Token: op, TokenPos: pos}
}

func importExpr(moduleName string, pos core.Pos) *parser.ImportExpr {
	return &parser.ImportExpr{
		ModuleName: moduleName, Token: token.Import, TokenPos: pos,
	}
}

func exprs(list ...parser.Expr) []parser.Expr {
	return list
}

func intLit(value int64, pos core.Pos) *parser.IntLit {
	return &parser.IntLit{Value: value, ValuePos: pos}
}

func floatLit(value float64, pos core.Pos) *parser.FloatLit {
	return &parser.FloatLit{Value: value, ValuePos: pos}
}

func stringLit(value string, pos core.Pos) *parser.StringLit {
	return &parser.StringLit{Value: value, ValuePos: pos}
}

func charLit(value rune, pos core.Pos) *parser.RuneLit {
	return &parser.RuneLit{Value: value, ValuePos: pos, Literal: fmt.Sprintf("'%c'", value)}
}

func boolLit(value bool, pos core.Pos) *parser.BoolLit {
	return &parser.BoolLit{Value: value, ValuePos: pos}
}

func undefinedLit(pos core.Pos) *parser.UndefinedLit {
	return &parser.UndefinedLit{TokenPos: pos}
}

func arrayLit(lbracket, rbracket core.Pos, list ...parser.Expr) *parser.ArrayLit {
	return &parser.ArrayLit{LBrack: lbracket, RBrack: rbracket, Elements: list}
}

func dictElementLit(key string, keyPos core.Pos, colonPos core.Pos, value parser.Expr) *parser.RecordElementLit {
	return &parser.RecordElementLit{
		Key: key, KeyPos: keyPos, ColonPos: colonPos, Value: value,
	}
}

func dictLit(lbrace, rbrace core.Pos, list ...*parser.RecordElementLit) *parser.RecordLit {
	return &parser.RecordLit{LBrace: lbrace, RBrace: rbrace, Elements: list}
}

func funcLit(funcType *parser.FuncType, body *parser.BlockStmt) *parser.FuncLit {
	return &parser.FuncLit{Type: funcType, Body: body}
}

func parenExpr(x parser.Expr, lparen, rparen core.Pos) *parser.ParenExpr {
	return &parser.ParenExpr{Expr: x, LParen: lparen, RParen: rparen}
}

func callExpr(f parser.Expr, lparen, rparen, ellipsis core.Pos, args ...parser.Expr) *parser.CallExpr {
	return &parser.CallExpr{Func: f, LParen: lparen, RParen: rparen, Ellipsis: ellipsis, Args: args}
}

func methodCallExpr(obj parser.Expr, methodName string, methodPos core.Pos, lparen, rparen, ellipsis core.Pos, args ...parser.Expr) *parser.MethodCallExpr {
	return &parser.MethodCallExpr{Object: obj, MethodName: methodName, MethodPos: methodPos, LParen: lparen, RParen: rparen, Ellipsis: ellipsis, Args: args}
}

func indexExpr(x, index parser.Expr, lbrack, rbrack core.Pos) *parser.IndexExpr {
	return &parser.IndexExpr{Expr: x, Index: index, LBrack: lbrack, RBrack: rbrack}
}

func sliceExpr(x, low, high parser.Expr, lbrack, rbrack core.Pos) *parser.SliceExpr {
	return &parser.SliceExpr{Expr: x, Low: low, High: high, LBrack: lbrack, RBrack: rbrack}
}

func sliceExprStep(x, low, high, step parser.Expr, lbrack, rbrack core.Pos) *parser.SliceExpr {
	return &parser.SliceExpr{Expr: x, Low: low, High: high, Step: step, LBrack: lbrack, RBrack: rbrack}
}

func selectorExpr(x, sel parser.Expr) *parser.SelectorExpr {
	return &parser.SelectorExpr{Expr: x, Sel: sel}
}

func equalStmt(t *testing.T, expected, actual parser.Stmt) {
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, rta, expected, actual)

	switch expected := expected.(type) {
	case *parser.ExprStmt:
		equalExpr(t, expected.Expr, actual.(*parser.ExprStmt).Expr)
	case *parser.EmptyStmt:
		require.Equal(t, rta, expected.Implicit, actual.(*parser.EmptyStmt).Implicit)
		require.Equal(t, rta, expected.Semicolon, actual.(*parser.EmptyStmt).Semicolon)
	case *parser.BlockStmt:
		require.Equal(t, rta, expected.LBrace, actual.(*parser.BlockStmt).LBrace)
		require.Equal(t, rta, expected.RBrace, actual.(*parser.BlockStmt).RBrace)
		equalStmts(t, expected.Stmts, actual.(*parser.BlockStmt).Stmts)
	case *parser.AssignStmt:
		equalExprs(t, expected.LHS, actual.(*parser.AssignStmt).LHS)
		equalExprs(t, expected.RHS, actual.(*parser.AssignStmt).RHS)
		require.Equal(t, rta, int(expected.Token), int(actual.(*parser.AssignStmt).Token))
		require.Equal(t, rta, int(expected.TokenPos), int(actual.(*parser.AssignStmt).TokenPos))
	case *parser.IfStmt:
		equalStmt(t, expected.Init, actual.(*parser.IfStmt).Init)
		equalExpr(t, expected.Cond, actual.(*parser.IfStmt).Cond)
		equalStmt(t, expected.Body, actual.(*parser.IfStmt).Body)
		equalStmt(t, expected.Else, actual.(*parser.IfStmt).Else)
		require.Equal(t, rta, expected.IfPos, actual.(*parser.IfStmt).IfPos)
	case *parser.IncDecStmt:
		equalExpr(t, expected.Expr, actual.(*parser.IncDecStmt).Expr)
		require.Equal(t, rta, expected.Token, actual.(*parser.IncDecStmt).Token)
		require.Equal(t, rta, expected.TokenPos, actual.(*parser.IncDecStmt).TokenPos)
	case *parser.ForStmt:
		equalStmt(t, expected.Init, actual.(*parser.ForStmt).Init)
		equalExpr(t, expected.Cond, actual.(*parser.ForStmt).Cond)
		equalStmt(t, expected.Post, actual.(*parser.ForStmt).Post)
		equalStmt(t, expected.Body, actual.(*parser.ForStmt).Body)
		require.Equal(t, rta, expected.ForPos, actual.(*parser.ForStmt).ForPos)
	case *parser.ForInStmt:
		equalExpr(t, expected.Key, actual.(*parser.ForInStmt).Key)
		equalExpr(t, expected.Value, actual.(*parser.ForInStmt).Value)
		equalExpr(t, expected.Iterable, actual.(*parser.ForInStmt).Iterable)
		equalStmt(t, expected.Body, actual.(*parser.ForInStmt).Body)
		require.Equal(t, rta, expected.ForPos, actual.(*parser.ForInStmt).ForPos)
	case *parser.ReturnStmt:
		equalExpr(t, expected.Result, actual.(*parser.ReturnStmt).Result)
		require.Equal(t, rta, expected.ReturnPos, actual.(*parser.ReturnStmt).ReturnPos)
	case *parser.BranchStmt:
		equalExpr(t, expected.Label, actual.(*parser.BranchStmt).Label)
		require.Equal(t, rta, expected.Token, actual.(*parser.BranchStmt).Token)
		require.Equal(t, rta, expected.TokenPos, actual.(*parser.BranchStmt).TokenPos)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func equalExpr(t *testing.T, expected, actual parser.Expr) {
	if expected == nil || reflect.ValueOf(expected).IsNil() {
		require.Nil(t, actual, "expected nil, but got not nil")
		return
	}
	require.NotNil(t, actual, "expected not nil, but got nil")
	require.IsType(t, rta, expected, actual)

	switch expected := expected.(type) {
	case *parser.Ident:
		require.Equal(t, rta, expected.Name, actual.(*parser.Ident).Name)
		require.Equal(t, rta, int(expected.NamePos), int(actual.(*parser.Ident).NamePos))
	case *parser.IntLit:
		require.Equal(t, rta, expected.Value, actual.(*parser.IntLit).Value)
		require.Equal(t, rta, int(expected.ValuePos), int(actual.(*parser.IntLit).ValuePos))
	case *parser.FloatLit:
		require.Equal(t, rta, expected.Value, actual.(*parser.FloatLit).Value)
		require.Equal(t, rta, int(expected.ValuePos), int(actual.(*parser.FloatLit).ValuePos))
	case *parser.BoolLit:
		require.Equal(t, rta, expected.Value, actual.(*parser.BoolLit).Value)
		require.Equal(t, rta, int(expected.ValuePos), int(actual.(*parser.BoolLit).ValuePos))
	case *parser.UndefinedLit:
		require.Equal(t, rta, int(expected.TokenPos), int(actual.(*parser.UndefinedLit).TokenPos))
	case *parser.RuneLit:
		require.Equal(t, rta, expected.Value, actual.(*parser.RuneLit).Value)
		require.Equal(t, rta, int(expected.ValuePos), int(actual.(*parser.RuneLit).ValuePos))
	case *parser.StringLit:
		require.Equal(t, rta, expected.Value, actual.(*parser.StringLit).Value)
		require.Equal(t, rta, int(expected.ValuePos), int(actual.(*parser.StringLit).ValuePos))
	case *parser.ArrayLit:
		require.Equal(t, rta, expected.LBrack, actual.(*parser.ArrayLit).LBrack)
		require.Equal(t, rta, expected.RBrack, actual.(*parser.ArrayLit).RBrack)
		equalExprs(t, expected.Elements, actual.(*parser.ArrayLit).Elements)
	case *parser.RecordLit:
		require.Equal(t, rta, expected.LBrace, actual.(*parser.RecordLit).LBrace)
		require.Equal(t, rta, expected.RBrace, actual.(*parser.RecordLit).RBrace)
		equalMapElements(t, expected.Elements, actual.(*parser.RecordLit).Elements)
	case *parser.BinaryExpr:
		equalExpr(t, expected.LHS, actual.(*parser.BinaryExpr).LHS)
		equalExpr(t, expected.RHS, actual.(*parser.BinaryExpr).RHS)
		require.Equal(t, rta, expected.Token, actual.(*parser.BinaryExpr).Token)
		require.Equal(t, rta, expected.TokenPos, actual.(*parser.BinaryExpr).TokenPos)
	case *parser.UnaryExpr:
		equalExpr(t, expected.Expr, actual.(*parser.UnaryExpr).Expr)
		require.Equal(t, rta, expected.Token, actual.(*parser.UnaryExpr).Token)
		require.Equal(t, rta, expected.TokenPos, actual.(*parser.UnaryExpr).TokenPos)
	case *parser.FuncLit:
		equalFuncType(t, expected.Type, actual.(*parser.FuncLit).Type)
		equalStmt(t, expected.Body, actual.(*parser.FuncLit).Body)
	case *parser.CallExpr:
		equalExpr(t, expected.Func, actual.(*parser.CallExpr).Func)
		require.Equal(t, rta, expected.LParen, actual.(*parser.CallExpr).LParen)
		require.Equal(t, rta, expected.RParen, actual.(*parser.CallExpr).RParen)
		equalExprs(t, expected.Args, actual.(*parser.CallExpr).Args)
	case *parser.MethodCallExpr:
		equalExpr(t, expected.Object, actual.(*parser.MethodCallExpr).Object)
		require.Equal(t, rta, expected.MethodName, actual.(*parser.MethodCallExpr).MethodName)
		require.Equal(t, rta, expected.MethodPos, actual.(*parser.MethodCallExpr).MethodPos)
		require.Equal(t, rta, expected.LParen, actual.(*parser.MethodCallExpr).LParen)
		require.Equal(t, rta, expected.RParen, actual.(*parser.MethodCallExpr).RParen)
		require.Equal(t, rta, expected.Ellipsis, actual.(*parser.MethodCallExpr).Ellipsis)
		equalExprs(t, expected.Args, actual.(*parser.MethodCallExpr).Args)
	case *parser.ParenExpr:
		equalExpr(t, expected.Expr, actual.(*parser.ParenExpr).Expr)
		require.Equal(t, rta, expected.LParen, actual.(*parser.ParenExpr).LParen)
		require.Equal(t, rta, expected.RParen, actual.(*parser.ParenExpr).RParen)
	case *parser.IndexExpr:
		equalExpr(t, expected.Expr, actual.(*parser.IndexExpr).Expr)
		equalExpr(t, expected.Index, actual.(*parser.IndexExpr).Index)
		require.Equal(t, rta, expected.LBrack, actual.(*parser.IndexExpr).LBrack)
		require.Equal(t, rta, expected.RBrack, actual.(*parser.IndexExpr).RBrack)
	case *parser.SliceExpr:
		equalExpr(t, expected.Expr, actual.(*parser.SliceExpr).Expr)
		equalExpr(t, expected.Low, actual.(*parser.SliceExpr).Low)
		equalExpr(t, expected.High, actual.(*parser.SliceExpr).High)
		equalExpr(t, expected.Step, actual.(*parser.SliceExpr).Step)
		require.Equal(t, rta, expected.LBrack, actual.(*parser.SliceExpr).LBrack)
		require.Equal(t, rta, expected.RBrack, actual.(*parser.SliceExpr).RBrack)
	case *parser.SelectorExpr:
		equalExpr(t, expected.Expr, actual.(*parser.SelectorExpr).Expr)
		equalExpr(t, expected.Sel, actual.(*parser.SelectorExpr).Sel)
	case *parser.ImportExpr:
		require.Equal(t, rta, expected.ModuleName, actual.(*parser.ImportExpr).ModuleName)
		require.Equal(t, rta, int(expected.TokenPos), int(actual.(*parser.ImportExpr).TokenPos))
		require.Equal(t, rta, expected.Token, actual.(*parser.ImportExpr).Token)
	case *parser.CondExpr:
		equalExpr(t, expected.Cond, actual.(*parser.CondExpr).Cond)
		equalExpr(t, expected.True, actual.(*parser.CondExpr).True)
		equalExpr(t, expected.False, actual.(*parser.CondExpr).False)
		require.Equal(t, rta, expected.QuestionPos, actual.(*parser.CondExpr).QuestionPos)
		require.Equal(t, rta, expected.ColonPos, actual.(*parser.CondExpr).ColonPos)
	default:
		panic(fmt.Errorf("unknown type: %T", expected))
	}
}

func equalFuncType(t *testing.T, expected, actual *parser.FuncType) {
	require.Equal(t, rta, expected.Params.LParen, actual.Params.LParen)
	require.Equal(t, rta, expected.Params.RParen, actual.Params.RParen)
	equalIdents(t, expected.Params.List, actual.Params.List)
}

func equalIdents(t *testing.T, expected, actual []*parser.Ident) {
	require.Equal(t, rta, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		equalExpr(t, expected[i], actual[i])
	}
}

func equalExprs(t *testing.T, expected, actual []parser.Expr) {
	require.Equal(t, rta, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		equalExpr(t, expected[i], actual[i])
	}
}

func equalStmts(t *testing.T, expected, actual []parser.Stmt) {
	require.Equal(t, rta, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		equalStmt(t, expected[i], actual[i])
	}
}

func equalMapElements(
	t *testing.T,
	expected, actual []*parser.RecordElementLit,
) {
	require.Equal(t, rta, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		require.Equal(t, rta, expected[i].Key, actual[i].Key)
		require.Equal(t, rta, expected[i].KeyPos, actual[i].KeyPos)
		require.Equal(t, rta, expected[i].ColonPos, actual[i].ColonPos)
		equalExpr(t, expected[i].Value, actual[i].Value)
	}
}

func parseSource(
	filename string,
	src []byte,
	trace io.Writer,
) (res *parser.File, err error) {
	fileSet := parser.NewFileSet()
	file := fileSet.AddFile(filename, -1, len(src))
	p := parser.NewParser(file, src, trace)
	return p.ParseFile()
}

func TestScanner_Scan(t *testing.T) {
	var testCases = [...]struct {
		token   token.Token
		literal string
	}{
		{token.Comment, "/* a comment */"},
		{token.Comment, "// a comment \n"},
		{token.Comment, "/*\r*/"},
		{token.Comment, "/**\r/*/"},
		{token.Comment, "/**\r\r/*/"},
		{token.Comment, "//\r\n"},
		{token.Ident, "foobar"},
		{token.Ident, "a۰۱۸"},
		{token.Ident, "foo६४"},
		{token.Ident, "bar９８７６"},
		{token.Ident, "ŝ"},
		{token.Ident, "ŝfoo"},
		{token.Int, "0"},
		{token.Int, "1"},
		{token.Int, "123456789012345678890"},
		{token.Int, "01234567"},
		{token.Int, "0xcafebabe"},
		{token.Float, "0."},
		{token.Float, ".0"},
		{token.Float, "3.14159265"},
		{token.Float, "1e0"},
		{token.Float, "1e+100"},
		{token.Float, "1e-100"},
		{token.Float, "2.71828e-1000"},
		{token.Float, "1f"},
		{token.Float, "1.5f"},
		{token.Decimal, "1d"},
		{token.Decimal, "1.23d"},
		{token.Char, "'a'"},
		{token.Char, "'\\000'"},
		{token.Char, "'\\xFF'"},
		{token.Char, "'\\uff16'"},
		{token.Char, "'\\U0000ff16'"},
		{token.String, "`foobar`"},
		{token.String, "`" + `foo
	                        bar` +
			"`",
		},
		{token.String, "`\r`"},
		{token.String, "`foo\r\nbar`"},
		{token.Add, "+"},
		{token.Sub, "-"},
		{token.Mul, "*"},
		{token.Quo, "/"},
		{token.Rem, "%"},
		{token.And, "&"},
		{token.Or, "|"},
		{token.Xor, "^"},
		{token.Shl, "<<"},
		{token.Shr, ">>"},
		{token.AndNot, "&^"},
		{token.AddAssign, "+="},
		{token.SubAssign, "-="},
		{token.MulAssign, "*="},
		{token.QuoAssign, "/="},
		{token.RemAssign, "%="},
		{token.AndAssign, "&="},
		{token.OrAssign, "|="},
		{token.XorAssign, "^="},
		{token.ShlAssign, "<<="},
		{token.ShrAssign, ">>="},
		{token.AndNotAssign, "&^="},
		{token.LAnd, "&&"},
		{token.LOr, "||"},
		{token.Inc, "++"},
		{token.Dec, "--"},
		{token.Equal, "=="},
		{token.Less, "<"},
		{token.Greater, ">"},
		{token.Assign, "="},
		{token.Not, "!"},
		{token.NotEqual, "!="},
		{token.LessEq, "<="},
		{token.GreaterEq, ">="},
		{token.Define, ":="},
		{token.Ellipsis, "..."},
		{token.LParen, "("},
		{token.LBrack, "["},
		{token.LBrace, "{"},
		{token.Comma, ","},
		{token.Period, "."},
		{token.RParen, ")"},
		{token.RBrack, "]"},
		{token.RBrace, "}"},
		{token.Semicolon, ";"},
		{token.Colon, ":"},
		{token.Break, "break"},
		{token.Continue, "continue"},
		{token.Else, "else"},
		{token.For, "for"},
		{token.Func, "func"},
		{token.If, "if"},
		{token.Return, "return"},
		{token.Export, "export"},
		{token.NotKw, "not"},
		{token.Var, "var"},
	}

	// combine
	var lines []string
	var lineSum int
	lineNos := make([]int, len(testCases))
	columnNos := make([]int, len(testCases))
	for i, tc := range testCases {
		// add 0-2 lines before each test case
		emptyLines := rand.Intn(3)
		for j := 0; j < emptyLines; j++ {
			lines = append(lines, strings.Repeat(" ", rand.Intn(10)))
		}

		// add test case line with some whitespaces around it
		emptyColumns := rand.Intn(10)
		lines = append(lines, fmt.Sprintf("%s%s%s",
			strings.Repeat(" ", emptyColumns),
			tc.literal,
			strings.Repeat(" ", rand.Intn(10))))

		lineNos[i] = lineSum + emptyLines + 1
		lineSum += emptyLines + countLines(tc.literal)
		columnNos[i] = emptyColumns + 1
	}

	// expected results
	var expected []scanResult
	var expectedSkipComments []scanResult
	for i, tc := range testCases {
		// expected literal
		var expectedLiteral string
		switch tc.token {
		case token.Comment:
			// strip CRs in comments
			expectedLiteral = string(parser.StripCR([]byte(tc.literal), tc.literal[1] == '*'))

			//-style comment literal doesn't contain newline
			if expectedLiteral[1] == '/' {
				expectedLiteral = expectedLiteral[:len(expectedLiteral)-1]
			}
		case token.Ident:
			expectedLiteral = tc.literal
		case token.Semicolon:
			expectedLiteral = ";"
		default:
			if tc.token.IsLiteral() {
				// strip CRs in raw string
				expectedLiteral = tc.literal
				if expectedLiteral[0] == '`' {
					expectedLiteral = string(parser.StripCR([]byte(expectedLiteral), false))
				}
			} else if tc.token.IsKeyword() {
				expectedLiteral = tc.literal
			}
		}

		res := scanResult{
			Token:   tc.token,
			Literal: expectedLiteral,
			Line:    lineNos[i],
			Column:  columnNos[i],
		}

		expected = append(expected, res)
		if tc.token != token.Comment {
			expectedSkipComments = append(expectedSkipComments, res)
		}
	}

	scanExpect(t, strings.Join(lines, "\n"), parser.ScanComments|parser.DoNotInsertSemis, expected...)
	scanExpect(t, strings.Join(lines, "\n"), parser.DoNotInsertSemis, expectedSkipComments...)
}

func TestStripCR(t *testing.T) {
	for _, tc := range []struct {
		input  string
		expect string
	}{
		{"//\n", "//\n"},
		{"//\r\n", "//\n"},
		{"//\r\r\r\n", "//\n"},
		{"//\r*\r/\r\n", "//*/\n"},
		{"/**/", "/**/"},
		{"/*\r/*/", "/*/*/"},
		{"/*\r*/", "/**/"},
		{"/**\r/*/", "/**\r/*/"},
		{"/*\r/\r*\r/*/", "/*/*\r/*/"},
		{"/*\r\r\r\r*/", "/**/"},
	} {
		actual := string(parser.StripCR([]byte(tc.input), len(tc.input) >= 2 && tc.input[1] == '*'))
		require.Equal(t, rta, tc.expect, actual)
	}
}

func TestParserError(t *testing.T) {
	err := &parser.Error{Pos: parser.SourceFilePos{
		Offset: 10, Line: 1, Column: 10,
	}, Msg: "test"}
	require.Equal(t, rta, "Parse Error: test\n\tat 1:10", err.Error())
}

func TestParserErrorList(t *testing.T) {
	var list parser.ErrorList
	list.Add(parser.SourceFilePos{Offset: 20, Line: 2, Column: 10}, "error 2")
	list.Add(parser.SourceFilePos{Offset: 30, Line: 3, Column: 10}, "error 3")
	list.Add(parser.SourceFilePos{Offset: 10, Line: 1, Column: 10}, "error 1")
	list.Sort()
	require.Equal(t, rta, "Parse Error: error 1\n\tat 1:10 (and 2 more errors)", list.Error())
}

func TestParseArray(t *testing.T) {
	expectParse(t, "[1, 2, 3]", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(1, 1), p(1, 9),
					intLit(1, p(1, 2)),
					intLit(2, p(1, 5)),
					intLit(3, p(1, 8)))))
	})

	expectParse(t, `
[
	1,
	2,
	3
]`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(2, 1), p(6, 1),
					intLit(1, p(3, 2)),
					intLit(2, p(4, 2)),
					intLit(3, p(5, 2)))))
	})
	expectParse(t, `
[
	1,
	2,
	3

]`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(2, 1), p(7, 1),
					intLit(1, p(3, 2)),
					intLit(2, p(4, 2)),
					intLit(3, p(5, 2)))))
	})

	expectParse(t, `[1, "foo", 12.34]`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(1, 1), p(1, 17),
					intLit(1, p(1, 2)),
					stringLit("foo", p(1, 5)),
					floatLit(12.34, p(1, 12)))))
	})

	expectParse(t, "a = [1, 2, 3]", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(arrayLit(p(1, 5), p(1, 13),
					intLit(1, p(1, 6)),
					intLit(2, p(1, 9)),
					intLit(3, p(1, 12)))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a = [1 + 2, b * 4, [4, c]]", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(arrayLit(p(1, 5), p(1, 26),
					binaryExpr(
						intLit(1, p(1, 6)),
						intLit(2, p(1, 10)),
						token.Add,
						p(1, 8)),
					binaryExpr(
						ident("b", p(1, 13)),
						intLit(4, p(1, 17)),
						token.Mul,
						p(1, 15)),
					arrayLit(p(1, 20), p(1, 25),
						intLit(4, p(1, 21)),
						ident("c", p(1, 24))))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, `[1, 2, 3,]`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(1, 1), p(1, 10),
					intLit(1, p(1, 2)),
					intLit(2, p(1, 5)),
					intLit(3, p(1, 8)))))
	})
	expectParse(t, `
[
	1,
	2,
	3,
]`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(2, 1), p(6, 1),
					intLit(1, p(3, 2)),
					intLit(2, p(4, 2)),
					intLit(3, p(5, 2)))))
	})
	expectParse(t, `
[
	1,
	2,
	3,

]`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				arrayLit(p(2, 1), p(7, 1),
					intLit(1, p(3, 2)),
					intLit(2, p(4, 2)),
					intLit(3, p(5, 2)))))
	})
	expectParseError(t, `[1, 2, 3, ,]`)
}

func TestParseAssignment(t *testing.T) {
	expectParse(t, "a = 5", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(intLit(5, p(1, 5))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a := 5", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(intLit(5, p(1, 6))),
				token.Define,
				p(1, 3)))
	})

	expectParse(t, "var a", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 5))),
				exprs(undefinedLit(p(1, 1))),
				token.Define,
				p(1, 1)))
	})

	expectParse(t, "var a = 5", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 5))),
				exprs(intLit(5, p(1, 9))),
				token.Define,
				p(1, 1)))
	})

	expectParse(t, "a, b = 5, 10", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1)),
					ident("b", p(1, 4))),
				exprs(
					intLit(5, p(1, 8)),
					intLit(10, p(1, 11))),
				token.Assign,
				p(1, 6)))
	})

	expectParse(t, "a, b := 5, 10", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1)),
					ident("b", p(1, 4))),
				exprs(
					intLit(5, p(1, 9)),
					intLit(10, p(1, 12))),
				token.Define,
				p(1, 6)))
	})

	expectParse(t, "a, b = a + 2, b - 8", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1)),
					ident("b", p(1, 4))),
				exprs(
					binaryExpr(
						ident("a", p(1, 8)),
						intLit(2, p(1, 12)),
						token.Add,
						p(1, 10)),
					binaryExpr(
						ident("b", p(1, 15)),
						intLit(8, p(1, 19)),
						token.Sub,
						p(1, 17))),
				token.Assign,
				p(1, 6)))
	})

	expectParse(t, "a = [1, 2, 3]", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(arrayLit(p(1, 5), p(1, 13),
					intLit(1, p(1, 6)),
					intLit(2, p(1, 9)),
					intLit(3, p(1, 12)))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a = [1 + 2, b * 4, [4, c]]", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(arrayLit(p(1, 5), p(1, 26),
					binaryExpr(
						intLit(1, p(1, 6)),
						intLit(2, p(1, 10)),
						token.Add,
						p(1, 8)),
					binaryExpr(
						ident("b", p(1, 13)),
						intLit(4, p(1, 17)),
						token.Mul,
						p(1, 15)),
					arrayLit(p(1, 20), p(1, 25),
						intLit(4, p(1, 21)),
						ident("c", p(1, 24))))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a += 5", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(intLit(5, p(1, 6))),
				token.AddAssign,
				p(1, 3)))
	})

	expectParse(t, "a *= 5 + 10", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(
					binaryExpr(
						intLit(5, p(1, 6)),
						intLit(10, p(1, 10)),
						token.Add,
						p(1, 8))),
				token.MulAssign,
				p(1, 3)))
	})
}

func TestParseBoolean(t *testing.T) {
	expectParse(t, "true", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				boolLit(true, p(1, 1))))
	})

	expectParse(t, "false", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				boolLit(false, p(1, 1))))
	})

	expectParse(t, "true != false", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					boolLit(true, p(1, 1)),
					boolLit(false, p(1, 9)),
					token.NotEqual,
					p(1, 6))))
	})

	expectParse(t, "!false", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				unaryExpr(
					boolLit(false, p(1, 2)),
					token.Not,
					p(1, 1))))
	})

	expectParse(t, `"z" not in "Hello"`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				unaryExpr(
					binaryExpr(
						stringLit("z", p(1, 1)),
						stringLit("Hello", p(1, 12)),
						token.In,
						p(1, 9),
					),
					token.Not,
					p(1, 5),
				)))
	})
}

func TestParseCall(t *testing.T) {
	expectParse(t, "add(1, 2, 3)", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				callExpr(
					ident("add", p(1, 1)),
					p(1, 4), p(1, 12), core.NoPos,
					intLit(1, p(1, 5)),
					intLit(2, p(1, 8)),
					intLit(3, p(1, 11)))))
	})

	expectParse(t, "add(1, 2, v...)", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				callExpr(
					ident("add", p(1, 1)),
					p(1, 4), p(1, 15), p(1, 12),
					intLit(1, p(1, 5)),
					intLit(2, p(1, 8)),
					ident("v", p(1, 11)))))
	})

	expectParse(t, "a = add(1, 2, 3)", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1))),
				exprs(
					callExpr(
						ident("add", p(1, 5)),
						p(1, 8), p(1, 16), core.NoPos,
						intLit(1, p(1, 9)),
						intLit(2, p(1, 12)),
						intLit(3, p(1, 15)))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a, b = add(1, 2, 3)", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1)),
					ident("b", p(1, 4))),
				exprs(
					callExpr(
						ident("add", p(1, 8)),
						p(1, 11), p(1, 19), core.NoPos,
						intLit(1, p(1, 12)),
						intLit(2, p(1, 15)),
						intLit(3, p(1, 18)))),
				token.Assign,
				p(1, 6)))
	})

	expectParse(t, "add(a + 1, 2 * 1, (b + c))", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				callExpr(
					ident("add", p(1, 1)),
					p(1, 4), p(1, 26), core.NoPos,
					binaryExpr(
						ident("a", p(1, 5)),
						intLit(1, p(1, 9)),
						token.Add,
						p(1, 7)),
					binaryExpr(
						intLit(2, p(1, 12)),
						intLit(1, p(1, 16)),
						token.Mul,
						p(1, 14)),
					parenExpr(
						binaryExpr(
							ident("b", p(1, 20)),
							ident("c", p(1, 24)),
							token.Add,
							p(1, 22)),
						p(1, 19), p(1, 25)))))
	})

	expectParseString(t, "a + add(b * c) + d", "((a + add((b * c))) + d)")
	expectParseString(t, "add(a, b, 1, 2 * 3, 4 + 5, add(6, 7 * 8))",
		"add(a, b, 1, (2 * 3), (4 + 5), add(6, (7 * 8)))")
	expectParseString(t, "f1(a) + f2(b) * f3(c)", "(f1(a) + (f2(b) * f3(c)))")
	expectParseString(t, "(f1(a) + f2(b)) * f3(c)",
		"(((f1(a) + f2(b))) * f3(c))")

	expectParse(t, "func(a, b) { a + b }(1, 2)", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				callExpr(
					funcLit(
						funcType(
							identList(
								p(1, 5), p(1, 10),
								false,
								ident("a", p(1, 6)),
								ident("b", p(1, 9))),
							p(1, 1)),
						blockStmt(
							p(1, 12), p(1, 20),
							exprStmt(
								binaryExpr(
									ident("a", p(1, 14)),
									ident("b", p(1, 18)),
									token.Add,
									p(1, 16))))),
					p(1, 21), p(1, 26), core.NoPos,
					intLit(1, p(1, 22)),
					intLit(2, p(1, 25)))))
	})

	expectParse(t, `a.b()`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				methodCallExpr(
					ident("a", p(1, 1)),
					"b",
					p(1, 3),
					p(1, 4), p(1, 5), core.NoPos)))
	})

	expectParse(t, `a.b.c()`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				methodCallExpr(
					selectorExpr(
						ident("a", p(1, 1)),
						stringLit("b", p(1, 3))),
					"c",
					p(1, 5),
					p(1, 6), p(1, 7), core.NoPos)))
	})

	expectParse(t, `a["b"].c()`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				methodCallExpr(
					indexExpr(
						ident("a", p(1, 1)),
						stringLit("b", p(1, 3)),
						p(1, 2), p(1, 6)),
					"c",
					p(1, 8),
					p(1, 9), p(1, 10), core.NoPos)))
	})

	expectParseError(t, `add(...a, 1)`)
	expectParseError(t, `add(a..., 1)`)
	expectParseError(t, `add(a..., b...)`)
	expectParseError(t, `add(1, a..., b...)`)
	expectParseError(t, `add(...)`)
	expectParseError(t, `add(1, ...)`)
	expectParseError(t, `add(1, ..., )`)
	expectParseError(t, `add(...a)`)
}

func TestParseChar(t *testing.T) {
	expectParse(t, `'A'`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				charLit('A', 1)))
	})
	expectParse(t, `'あ'`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				charLit('あ', 1)))
	})

	expectParseError(t, `''`)
	expectParseError(t, `'AB'`)
	expectParseError(t, `'Aあ'`)
}

func TestParseCondExpr(t *testing.T) {
	expectParse(t, "a ? b : c", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				condExpr(
					ident("a", p(1, 1)),
					ident("b", p(1, 5)),
					ident("c", p(1, 9)),
					p(1, 3),
					p(1, 7))))
	})
	expectParse(t, `a ?
b :
c`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				condExpr(
					ident("a", p(1, 1)),
					ident("b", p(1, 5)),
					ident("c", p(1, 9)),
					p(1, 3),
					p(1, 7))))
	})

	expectParseString(t, `a ? b : c`, "(a ? b : c)")
	expectParseString(t, `a + b ? c - d : e * f`,
		"((a + b) ? (c - d) : (e * f))")
	expectParseString(t, `a == b ? c + (d / e) : f ? g : h + i`,
		"((a == b) ? (c + ((d / e))) : (f ? g : (h + i)))")
	expectParseString(t, `(a + b) ? (c - d) : (e * f)`,
		"(((a + b)) ? ((c - d)) : ((e * f)))")
	expectParseString(t, `a + (b ? c : d) - e`, "((a + ((b ? c : d))) - e)")
	expectParseString(t, `a ? b ? c : d : e`, "(a ? (b ? c : d) : e)")
	expectParseString(t, `a := b ? c : d`, "a := (b ? c : d)")
	expectParseString(t, `x := a ? b ? c : d : e`,
		"x := (a ? (b ? c : d) : e)")

	// ? : should be at the end of each line if it's multi-line
	expectParseError(t, `a
? b
: c`)
	expectParseError(t, `a ? (b : e)`)
	expectParseError(t, `(a ? b) : e`)
}

func TestParseForIn(t *testing.T) {
	expectParse(t, "for x in y {}", func(p pfn) []parser.Stmt {
		return stmts(
			forInStmt(
				ident("_", p(1, 5)),
				ident("x", p(1, 5)),
				ident("y", p(1, 10)),
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1)))
	})

	expectParse(t, "for _ in y {}", func(p pfn) []parser.Stmt {
		return stmts(
			forInStmt(
				ident("_", p(1, 5)),
				ident("_", p(1, 5)),
				ident("y", p(1, 10)),
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1)))
	})

	expectParse(t, "for x in [1, 2, 3] {}", func(p pfn) []parser.Stmt {
		return stmts(
			forInStmt(
				ident("_", p(1, 5)),
				ident("x", p(1, 5)),
				arrayLit(
					p(1, 10), p(1, 18),
					intLit(1, p(1, 11)),
					intLit(2, p(1, 14)),
					intLit(3, p(1, 17))),
				blockStmt(p(1, 20), p(1, 21)),
				p(1, 1)))
	})

	expectParse(t, "for x, y in z {}", func(p pfn) []parser.Stmt {
		return stmts(
			forInStmt(
				ident("x", p(1, 5)),
				ident("y", p(1, 8)),
				ident("z", p(1, 13)),
				blockStmt(p(1, 15), p(1, 16)),
				p(1, 1)))
	})

	expectParse(t, "for x, y in {k1: 1, k2: 2} {}", func(p pfn) []parser.Stmt {
		return stmts(
			forInStmt(
				ident("x", p(1, 5)),
				ident("y", p(1, 8)),
				dictLit(
					p(1, 13), p(1, 26),
					dictElementLit(
						"k1", p(1, 14), p(1, 16), intLit(1, p(1, 18))),
					dictElementLit(
						"k2", p(1, 21), p(1, 23), intLit(2, p(1, 25)))),
				blockStmt(p(1, 28), p(1, 29)),
				p(1, 1)))
	})
}

func TestParseFor(t *testing.T) {
	expectParse(t, "for {}", func(p pfn) []parser.Stmt {
		return stmts(
			forStmt(nil, nil, nil, blockStmt(p(1, 5), p(1, 6)), p(1, 1)))
	})

	expectParse(t, "for a == 5 {}", func(p pfn) []parser.Stmt {
		return stmts(
			forStmt(
				nil,
				binaryExpr(
					ident("a", p(1, 5)),
					intLit(5, p(1, 10)),
					token.Equal,
					p(1, 7)),
				nil,
				blockStmt(p(1, 12), p(1, 13)),
				p(1, 1)))
	})

	expectParse(t, "for a := 0; a == 5;  {}", func(p pfn) []parser.Stmt {
		return stmts(
			forStmt(
				assignStmt(
					exprs(ident("a", p(1, 5))),
					exprs(intLit(0, p(1, 10))),
					token.Define, p(1, 7)),
				binaryExpr(
					ident("a", p(1, 13)),
					intLit(5, p(1, 18)),
					token.Equal,
					p(1, 15)),
				nil,
				blockStmt(p(1, 22), p(1, 23)),
				p(1, 1)))
	})

	expectParse(t, "for a := 0; a < 5; a++ {}", func(p pfn) []parser.Stmt {
		return stmts(
			forStmt(
				assignStmt(
					exprs(ident("a", p(1, 5))),
					exprs(intLit(0, p(1, 10))),
					token.Define, p(1, 7)),
				binaryExpr(
					ident("a", p(1, 13)),
					intLit(5, p(1, 17)),
					token.Less,
					p(1, 15)),
				incDecStmt(
					ident("a", p(1, 20)),
					token.Inc, p(1, 21)),
				blockStmt(p(1, 24), p(1, 25)),
				p(1, 1)))
	})

	expectParse(t, "for var i = 0; i < 2; i++ {}", func(p pfn) []parser.Stmt {
		return stmts(
			forStmt(
				assignStmt(
					exprs(ident("i", p(1, 9))),
					exprs(intLit(0, p(1, 13))),
					token.Define, p(1, 5)),
				binaryExpr(
					ident("i", p(1, 16)),
					intLit(2, p(1, 20)),
					token.Less,
					p(1, 18)),
				incDecStmt(
					ident("i", p(1, 23)),
					token.Inc, p(1, 24)),
				blockStmt(p(1, 27), p(1, 28)),
				p(1, 1)))
	})

	expectParse(t, "for ; a < 5; a++ {}", func(p pfn) []parser.Stmt {
		return stmts(
			forStmt(
				nil,
				binaryExpr(
					ident("a", p(1, 7)),
					intLit(5, p(1, 11)),
					token.Less,
					p(1, 9)),
				incDecStmt(
					ident("a", p(1, 14)),
					token.Inc, p(1, 15)),
				blockStmt(p(1, 18), p(1, 19)),
				p(1, 1)))
	})

	expectParse(t, "for a := 0; ; a++ {}", func(p pfn) []parser.Stmt {
		return stmts(
			forStmt(
				assignStmt(
					exprs(ident("a", p(1, 5))),
					exprs(intLit(0, p(1, 10))),
					token.Define, p(1, 7)),
				nil,
				incDecStmt(
					ident("a", p(1, 15)),
					token.Inc, p(1, 16)),
				blockStmt(p(1, 19), p(1, 20)),
				p(1, 1)))
	})

	expectParse(t, "for a == 5 && b != 4 {}", func(p pfn) []parser.Stmt {
		return stmts(
			forStmt(
				nil,
				binaryExpr(
					binaryExpr(
						ident("a", p(1, 5)),
						intLit(5, p(1, 10)),
						token.Equal,
						p(1, 7)),
					binaryExpr(
						ident("b", p(1, 15)),
						intLit(4, p(1, 20)),
						token.NotEqual,
						p(1, 17)),
					token.LAnd,
					p(1, 12)),
				nil,
				blockStmt(p(1, 22), p(1, 23)),
				p(1, 1)))
	})

	expectParse(t, "for (x in y) {}", func(p pfn) []parser.Stmt {
		return stmts(
			forStmt(
				nil,
				parenExpr(
					binaryExpr(
						ident("x", p(1, 6)),
						ident("y", p(1, 11)),
						token.In,
						p(1, 8)),
					p(1, 5),
					p(1, 12)),
				nil,
				blockStmt(p(1, 14), p(1, 15)),
				p(1, 1)))
	})
}

func TestParseFunction(t *testing.T) {
	expectParse(t, "a = func(b, c, d) { return d }", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1))),
				exprs(
					funcLit(
						funcType(
							identList(p(1, 9), p(1, 17), false,
								ident("b", p(1, 10)),
								ident("c", p(1, 13)),
								ident("d", p(1, 16))),
							p(1, 5)),
						blockStmt(p(1, 19), p(1, 30),
							returnStmt(p(1, 21), ident("d", p(1, 28)))))),
				token.Assign,
				p(1, 3)))
	})
}

func TestParseVariadicFunction(t *testing.T) {
	expectParse(t, "a = func(...args) { return args }", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1))),
				exprs(
					funcLit(
						funcType(
							identList(
								p(1, 9), p(1, 17),
								true,
								ident("args", p(1, 13)),
							), p(1, 5)),
						blockStmt(p(1, 19), p(1, 33),
							returnStmt(p(1, 21),
								ident("args", p(1, 28)),
							),
						),
					),
				),
				token.Assign,
				p(1, 3)))
	})
}

func TestParseVariadicFunctionWithArgs(t *testing.T) {
	expectParse(t, "a = func(x, y, ...z) { return z }", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					ident("a", p(1, 1))),
				exprs(
					funcLit(
						funcType(
							identList(
								p(1, 9), p(1, 20),
								true,
								ident("x", p(1, 10)),
								ident("y", p(1, 13)),
								ident("z", p(1, 19)),
							), p(1, 5)),
						blockStmt(p(1, 22), p(1, 33),
							returnStmt(p(1, 24),
								ident("z", p(1, 31)),
							),
						),
					),
				),
				token.Assign,
				p(1, 3)))
	})

	expectParseError(t, "a = func(x, y, ...z, invalid) { return z }")
	expectParseError(t, "a = func(...args, invalid) { return args }")
}

func TestParseIf(t *testing.T) {
	expectParse(t, "if a == 5 {}", func(p pfn) []parser.Stmt {
		return stmts(
			ifStmt(
				nil,
				binaryExpr(
					ident("a", p(1, 4)),
					intLit(5, p(1, 9)),
					token.Equal,
					p(1, 6)),
				blockStmt(
					p(1, 11), p(1, 12)),
				nil,
				p(1, 1)))
	})

	expectParse(t, "if a == 5 && b != 3 {}", func(p pfn) []parser.Stmt {
		return stmts(
			ifStmt(
				nil,
				binaryExpr(
					binaryExpr(
						ident("a", p(1, 4)),
						intLit(5, p(1, 9)),
						token.Equal,
						p(1, 6)),
					binaryExpr(
						ident("b", p(1, 14)),
						intLit(3, p(1, 19)),
						token.NotEqual,
						p(1, 16)),
					token.LAnd,
					p(1, 11)),
				blockStmt(
					p(1, 21), p(1, 22)),
				nil,
				p(1, 1)))
	})

	expectParse(t, "if var a = 5; a {}", func(p pfn) []parser.Stmt {
		return stmts(
			ifStmt(
				assignStmt(
					exprs(ident("a", p(1, 8))),
					exprs(intLit(5, p(1, 12))),
					token.Define,
					p(1, 4)),
				ident("a", p(1, 15)),
				blockStmt(
					p(1, 17), p(1, 18)),
				nil,
				p(1, 1)))
	})

	expectParse(t, "if a == 5 { a = 3; a = 1 }", func(p pfn) []parser.Stmt {
		return stmts(
			ifStmt(
				nil,
				binaryExpr(
					ident("a", p(1, 4)),
					intLit(5, p(1, 9)),
					token.Equal,
					p(1, 6)),
				blockStmt(
					p(1, 11), p(1, 26),
					assignStmt(
						exprs(ident("a", p(1, 13))),
						exprs(intLit(3, p(1, 17))),
						token.Assign,
						p(1, 15)),
					assignStmt(
						exprs(ident("a", p(1, 20))),
						exprs(intLit(1, p(1, 24))),
						token.Assign,
						p(1, 22))),
				nil,
				p(1, 1)))
	})

	expectParse(t, "if a == 5 { a = 3; a = 1 } else { a = 2; a = 4 }",
		func(p pfn) []parser.Stmt {
			return stmts(
				ifStmt(
					nil,
					binaryExpr(
						ident("a", p(1, 4)),
						intLit(5, p(1, 9)),
						token.Equal,
						p(1, 6)),
					blockStmt(
						p(1, 11), p(1, 26),
						assignStmt(
							exprs(ident("a", p(1, 13))),
							exprs(intLit(3, p(1, 17))),
							token.Assign,
							p(1, 15)),
						assignStmt(
							exprs(ident("a", p(1, 20))),
							exprs(intLit(1, p(1, 24))),
							token.Assign,
							p(1, 22))),
					blockStmt(
						p(1, 33), p(1, 48),
						assignStmt(
							exprs(ident("a", p(1, 35))),
							exprs(intLit(2, p(1, 39))),
							token.Assign,
							p(1, 37)),
						assignStmt(
							exprs(ident("a", p(1, 42))),
							exprs(intLit(4, p(1, 46))),
							token.Assign,
							p(1, 44))),
					p(1, 1)))
		})

	expectParse(t, `
if a == 5 {
	b = 3
	c = 1
} else if d == 3 {
	e = 8
	f = 3
} else {
	g = 2
	h = 4
}`, func(p pfn) []parser.Stmt {
		return stmts(
			ifStmt(
				nil,
				binaryExpr(
					ident("a", p(2, 4)),
					intLit(5, p(2, 9)),
					token.Equal,
					p(2, 6)),
				blockStmt(
					p(2, 11), p(5, 1),
					assignStmt(
						exprs(ident("b", p(3, 2))),
						exprs(intLit(3, p(3, 6))),
						token.Assign,
						p(3, 4)),
					assignStmt(
						exprs(ident("c", p(4, 2))),
						exprs(intLit(1, p(4, 6))),
						token.Assign,
						p(4, 4))),
				ifStmt(
					nil,
					binaryExpr(
						ident("d", p(5, 11)),
						intLit(3, p(5, 16)),
						token.Equal,
						p(5, 13)),
					blockStmt(
						p(5, 18), p(8, 1),
						assignStmt(
							exprs(ident("e", p(6, 2))),
							exprs(intLit(8, p(6, 6))),
							token.Assign,
							p(6, 4)),
						assignStmt(
							exprs(ident("f", p(7, 2))),
							exprs(intLit(3, p(7, 6))),
							token.Assign,
							p(7, 4))),
					blockStmt(
						p(8, 8), p(11, 1),
						assignStmt(
							exprs(ident("g", p(9, 2))),
							exprs(intLit(2, p(9, 6))),
							token.Assign,
							p(9, 4)),
						assignStmt(
							exprs(ident("h", p(10, 2))),
							exprs(intLit(4, p(10, 6))),
							token.Assign,
							p(10, 4))),
					p(5, 8)),
				p(2, 1)))
	})

	expectParse(t, "if a := 3; a < b {}", func(p pfn) []parser.Stmt {
		return stmts(
			ifStmt(
				assignStmt(
					exprs(ident("a", p(1, 4))),
					exprs(intLit(3, p(1, 9))),
					token.Define, p(1, 6)),
				binaryExpr(
					ident("a", p(1, 12)),
					ident("b", p(1, 16)),
					token.Less, p(1, 14)),
				blockStmt(
					p(1, 18), p(1, 19)),
				nil,
				p(1, 1)))
	})

	expectParse(t, "if a++; a < b {}", func(p pfn) []parser.Stmt {
		return stmts(
			ifStmt(
				incDecStmt(ident("a", p(1, 4)), token.Inc, p(1, 5)),
				binaryExpr(
					ident("a", p(1, 9)),
					ident("b", p(1, 13)),
					token.Less, p(1, 11)),
				blockStmt(
					p(1, 15), p(1, 16)),
				nil,
				p(1, 1)))
	})

	expectParseError(t, `if {}`)
	expectParseError(t, `if a == b { } else a != b { }`)
	expectParseError(t, `if a == b { } else if { }`)
	expectParseError(t, `else { }`)
	expectParseError(t, `if ; {}`)
	expectParseError(t, `if a := 3; {}`)
	expectParseError(t, `if ; a < 3 {}`)
}

func TestParseImport(t *testing.T) {
	expectParse(t, `a := import("mod1")`, func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(importExpr("mod1", p(1, 6))),
				token.Define, p(1, 3)))
	})

	expectParse(t, `import("mod1").var1`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					importExpr("mod1", p(1, 1)),
					stringLit("var1", p(1, 16)))))
	})

	expectParse(t, `import("mod1").func1()`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				methodCallExpr(
					importExpr("mod1", p(1, 1)),
					"func1",
					p(1, 16),
					p(1, 21), p(1, 22), core.NoPos)))
	})

	expectParse(t, `for x, y in import("mod1") {}`, func(p pfn) []parser.Stmt {
		return stmts(
			forInStmt(
				ident("x", p(1, 5)),
				ident("y", p(1, 8)),
				importExpr("mod1", p(1, 13)),
				blockStmt(p(1, 28), p(1, 29)),
				p(1, 1)))
	})
}

func TestParseIndex(t *testing.T) {
	expectParse(t, "[1, 2, 3][1]", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					intLit(1, p(1, 11)),
					p(1, 10), p(1, 12))))
	})

	expectParse(t, "[1, 2, 3][5 - a]", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					binaryExpr(
						intLit(5, p(1, 11)),
						ident("a", p(1, 15)),
						token.Sub,
						p(1, 13)),
					p(1, 10), p(1, 16))))
	})

	expectParse(t, "[1, 2, 3][5 : a]", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				sliceExpr(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					intLit(5, p(1, 11)),
					ident("a", p(1, 15)),
					p(1, 10), p(1, 16))))
	})

	expectParse(t, "[1, 2, 3][a + 3 : b - 8]", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				sliceExpr(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					binaryExpr(
						ident("a", p(1, 11)),
						intLit(3, p(1, 15)),
						token.Add,
						p(1, 13)),
					binaryExpr(
						ident("b", p(1, 19)),
						intLit(8, p(1, 23)),
						token.Sub,
						p(1, 21)),
					p(1, 10), p(1, 24))))
	})

	expectParse(t, "[1, 2, 3][0:3:2]", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				sliceExprStep(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					intLit(0, p(1, 11)),
					intLit(3, p(1, 13)),
					intLit(2, p(1, 15)),
					p(1, 10), p(1, 16))))
	})

	expectParse(t, "[1, 2, 3][::-1]", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				sliceExprStep(
					arrayLit(p(1, 1), p(1, 9),
						intLit(1, p(1, 2)),
						intLit(2, p(1, 5)),
						intLit(3, p(1, 8))),
					nil,
					nil,
					unaryExpr(
						intLit(1, p(1, 14)),
						token.Sub,
						p(1, 13)),
					p(1, 10), p(1, 15))))
	})

	expectParse(t, `{a: 1, b: 2}["b"]`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					dictLit(p(1, 1), p(1, 12),
						dictElementLit(
							"a", p(1, 2), p(1, 3), intLit(1, p(1, 5))),
						dictElementLit(
							"b", p(1, 8), p(1, 9), intLit(2, p(1, 11)))),
					stringLit("b", p(1, 14)),
					p(1, 13), p(1, 17))))
	})

	expectParse(t, `{a: 1, b: 2}[a + b]`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				indexExpr(
					dictLit(p(1, 1), p(1, 12),
						dictElementLit(
							"a", p(1, 2), p(1, 3), intLit(1, p(1, 5))),
						dictElementLit(
							"b", p(1, 8), p(1, 9), intLit(2, p(1, 11)))),
					binaryExpr(
						ident("a", p(1, 14)),
						ident("b", p(1, 18)),
						token.Add,
						p(1, 16)),
					p(1, 13), p(1, 19))))
	})
}

func TestParseLogical(t *testing.T) {
	expectParse(t, "a && 5 || true", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					binaryExpr(
						ident("a", p(1, 1)),
						intLit(5, p(1, 6)),
						token.LAnd,
						p(1, 3)),
					boolLit(true, p(1, 11)),
					token.LOr,
					p(1, 8))))
	})

	expectParse(t, "a || 5 && true", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					ident("a", p(1, 1)),
					binaryExpr(
						intLit(5, p(1, 6)),
						boolLit(true, p(1, 11)),
						token.LAnd,
						p(1, 8)),
					token.LOr,
					p(1, 3))))
	})

	expectParse(t, "a && (5 || true)", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					ident("a", p(1, 1)),
					parenExpr(
						binaryExpr(
							intLit(5, p(1, 7)),
							boolLit(true, p(1, 12)),
							token.LOr,
							p(1, 9)),
						p(1, 6), p(1, 16)),
					token.LAnd,
					p(1, 3))))
	})
}

func TestParseDict(t *testing.T) {
	expectParse(t, "{ key1: 1, key2: \"2\", key3: true }", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				dictLit(p(1, 1), p(1, 34),
					dictElementLit("key1", p(1, 3), p(1, 7), intLit(1, p(1, 9))),
					dictElementLit("key2", p(1, 12), p(1, 16), stringLit("2", p(1, 18))),
					dictElementLit("key3", p(1, 23), p(1, 27), boolLit(true, p(1, 29))))))
	})

	expectParse(t, "{ \"key1\": 1 }", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				dictLit(p(1, 1), p(1, 13),
					dictElementLit("key1", p(1, 3), p(1, 9), intLit(1, p(1, 11))))))
	})

	expectParse(t, `{ key1: 1, }`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				dictLit(p(1, 1), p(1, 12),
					dictElementLit("key1", p(1, 3), p(1, 7), intLit(1, p(1, 9))))))
	})

	expectParse(t, "a = { key1: 1, key2: \"2\", key3: true }",
		func(p pfn) []parser.Stmt {
			return stmts(assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(dictLit(p(1, 5), p(1, 38),
					dictElementLit("key1", p(1, 7), p(1, 11), intLit(1, p(1, 13))),
					dictElementLit("key2", p(1, 16), p(1, 20), stringLit("2", p(1, 22))),
					dictElementLit("key3", p(1, 27), p(1, 31), boolLit(true, p(1, 33))))),
				token.Assign,
				p(1, 3)))
		})

	expectParse(t, "a = { key1: 1, key2: \"2\", key3: { k1: `bar`, k2: 4 } }",
		func(p pfn) []parser.Stmt {
			return stmts(assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(dictLit(p(1, 5), p(1, 54),
					dictElementLit("key1", p(1, 7), p(1, 11), intLit(1, p(1, 13))),
					dictElementLit("key2", p(1, 16), p(1, 20), stringLit("2", p(1, 22))),
					dictElementLit(
						"key3", p(1, 27), p(1, 31),
						dictLit(p(1, 33), p(1, 52),
							dictElementLit(
								"k1", p(1, 35),
								p(1, 37), stringLit("bar", p(1, 39))),
							dictElementLit(
								"k2", p(1, 46),
								p(1, 48), intLit(4, p(1, 50))))))),
				token.Assign,
				p(1, 3)))
		})

	expectParse(t, `
{
	key1: 1,
	key2: "2",
	key3: true
}`, func(p pfn) []parser.Stmt {
		return stmts(exprStmt(
			dictLit(p(2, 1), p(6, 1),
				dictElementLit(
					"key1", p(3, 2), p(3, 6), intLit(1, p(3, 8))),
				dictElementLit(
					"key2", p(4, 2), p(4, 6), stringLit("2", p(4, 8))),
				dictElementLit(
					"key3", p(5, 2), p(5, 6), boolLit(true, p(5, 8))))))
	})

	expectParse(t, `
{
	key1: 1,
	key2: "2",
	key3: true,
}`, func(p pfn) []parser.Stmt {
		return stmts(exprStmt(
			dictLit(p(2, 1), p(6, 1),
				dictElementLit(
					"key1", p(3, 2), p(3, 6), intLit(1, p(3, 8))),
				dictElementLit(
					"key2", p(4, 2), p(4, 6), stringLit("2", p(4, 8))),
				dictElementLit(
					"key3", p(5, 2), p(5, 6), boolLit(true, p(5, 8))))))
	})

	expectParse(t, `{
key1: 1,
key2: 2,
}`, func(p pfn) []parser.Stmt {
		return stmts(exprStmt(
			dictLit(p(1, 1), p(4, 1),
				dictElementLit(
					"key1", p(2, 1), p(2, 5), intLit(1, p(2, 7))),
				dictElementLit(
					"key2", p(3, 1), p(3, 5), intLit(2, p(3, 7))))))
	})
}

func TestParsePrecedence(t *testing.T) {
	expectParseString(t, `a + b + c`, `((a + b) + c)`)
	expectParseString(t, `a + b * c`, `(a + (b * c))`)
	expectParseString(t, `x = 2 * 1 + 3 / 4`, `x = ((2 * 1) + (3 / 4))`)
	expectParseString(t, `a + b in c + d`, `((a + b) in (c + d))`)
	expectParseString(t, `a || b in c`, `(a || (b in c))`)
}

func TestParseSelector(t *testing.T) {
	expectParse(t, "a.b", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					ident("a", p(1, 1)),
					stringLit("b", p(1, 3)))))
	})

	expectParse(t, "a.b.c", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					selectorExpr(
						ident("a", p(1, 1)),
						stringLit("b", p(1, 3))),
					stringLit("c", p(1, 5)))))
	})

	expectParse(t, "{k1:1}.k1", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					dictLit(
						p(1, 1), p(1, 6),
						dictElementLit(
							"k1", p(1, 2), p(1, 4), intLit(1, p(1, 5)))),
					stringLit("k1", p(1, 8)))))

	})
	expectParse(t, "{k1:{v1:1}}.k1.v1", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				selectorExpr(
					selectorExpr(
						dictLit(
							p(1, 1), p(1, 11),
							dictElementLit("k1", p(1, 2), p(1, 4),
								dictLit(p(1, 5), p(1, 10),
									dictElementLit(
										"v1", p(1, 6),
										p(1, 8), intLit(1, p(1, 9)))))),
						stringLit("k1", p(1, 13))),
					stringLit("v1", p(1, 16)))))
	})

	expectParse(t, "a.b = 4", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						ident("a", p(1, 1)),
						stringLit("b", p(1, 3)))),
				exprs(intLit(4, p(1, 7))),
				token.Assign, p(1, 5)))
	})

	expectParse(t, "a.b.c = 4", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						selectorExpr(
							ident("a", p(1, 1)),
							stringLit("b", p(1, 3))),
						stringLit("c", p(1, 5)))),
				exprs(intLit(4, p(1, 9))),
				token.Assign, p(1, 7)))
	})

	expectParse(t, "a.b.c = 4 + 5", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						selectorExpr(
							ident("a", p(1, 1)),
							stringLit("b", p(1, 3))),
						stringLit("c", p(1, 5)))),
				exprs(
					binaryExpr(
						intLit(4, p(1, 9)),
						intLit(5, p(1, 13)),
						token.Add,
						p(1, 11))),
				token.Assign, p(1, 7)))
	})

	expectParse(t, "a[0].c = 4", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							ident("a", p(1, 1)),
							intLit(0, p(1, 3)),
							p(1, 2), p(1, 4)),
						stringLit("c", p(1, 6)))),
				exprs(intLit(4, p(1, 10))),
				token.Assign, p(1, 8)))
	})

	expectParse(t, "a.b[0].c = 4", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							selectorExpr(
								ident("a", p(1, 1)),
								stringLit("b", p(1, 3))),
							intLit(0, p(1, 5)),
							p(1, 4), p(1, 6)),
						stringLit("c", p(1, 8)))),
				exprs(intLit(4, p(1, 12))),
				token.Assign, p(1, 10)))
	})

	expectParse(t, "a.b[0][2].c = 4", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							indexExpr(
								selectorExpr(
									ident("a", p(1, 1)),
									stringLit("b", p(1, 3))),
								intLit(0, p(1, 5)),
								p(1, 4), p(1, 6)),
							intLit(2, p(1, 8)),
							p(1, 7), p(1, 9)),
						stringLit("c", p(1, 11)))),
				exprs(intLit(4, p(1, 15))),
				token.Assign, p(1, 13)))
	})

	expectParse(t, `a.b["key1"][2].c = 4`, func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							indexExpr(
								selectorExpr(
									ident("a", p(1, 1)),
									stringLit("b", p(1, 3))),
								stringLit("key1", p(1, 5)),
								p(1, 4), p(1, 11)),
							intLit(2, p(1, 13)),
							p(1, 12), p(1, 14)),
						stringLit("c", p(1, 16)))),
				exprs(intLit(4, p(1, 20))),
				token.Assign, p(1, 18)))
	})

	expectParse(t, "a[0].b[2].c = 4", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(
					selectorExpr(
						indexExpr(
							selectorExpr(
								indexExpr(
									ident("a", p(1, 1)),
									intLit(0, p(1, 3)),
									p(1, 2), p(1, 4)),
								stringLit("b", p(1, 6))),
							intLit(2, p(1, 8)),
							p(1, 7), p(1, 9)),
						stringLit("c", p(1, 11)))),
				exprs(intLit(4, p(1, 15))),
				token.Assign, p(1, 13)))
	})

	expectParseError(t, `a.(b.c)`)
}

func TestParseSemicolon(t *testing.T) {
	expectParse(t, "1", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	expectParse(t, "1;", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	expectParse(t, "1;;", func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))),
			emptyStmt(false, p(1, 3)))
	})

	expectParse(t, `1
`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))))
	})

	expectParse(t, `1
;`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))),
			emptyStmt(false, p(2, 1)))
	})

	expectParse(t, `1;
;`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(intLit(1, p(1, 1))),
			emptyStmt(false, p(2, 1)))
	})
}

func TestParseMultilineSelectorContinuation(t *testing.T) {
	actual, err := parseSource("test", []byte(`x := [1, 2, 3]
   .sort()
   .sum()`), nil)
	require.NoError(t, err)
	require.Equal(t, rta, "x := [1, 2, 3].sort().sum()", actual.String())

	actual, err = parseSource("test", []byte(`result := [1, 2, 3, 4]
  .filter(x => x % 2 == 0)
  .map(x => x * x)
  .sum()`), nil)
	require.NoError(t, err)
	require.Equal(t, rta, "result := [1, 2, 3, 4].filter(func(x) {return ((x % 2) == 0)}).map(func(x) {return (x * x)}).sum()", actual.String())
}

func TestParseString(t *testing.T) {
	expectParse(t, `a = "foo\nbar"`, func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(stringLit("foo\nbar", p(1, 5))),
				token.Assign,
				p(1, 3)))
	})

	expectParse(t, "a = `raw string`", func(p pfn) []parser.Stmt {
		return stmts(
			assignStmt(
				exprs(ident("a", p(1, 1))),
				exprs(stringLit("raw string", p(1, 5))),
				token.Assign,
				p(1, 3)))
	})
}

func TestParseInt(t *testing.T) {
	testCases := []string{
		// All valid digits
		"1234567890",
		"0b10",
		"0o12345670",
		"0x123456789abcdef0",
		"0x123456789ABCDEF0",

		// Alternative base prefixes
		"010",
		"0B10",
		"0O10",
		"0X10",

		// Invalid digits
		"0b2",
		"08",
		"0o8",
		"1a",
		"0xg",

		// Range errors
		"9223372036854775807",
		"9223372036854775808", // invalid: range error

		// Examples from specification (https://go.dev/ref/spec#Integer_literals)
		"42",
		"4_2",
		"0600",
		"0_600",
		"0o600",
		"0O600", // second character is capital letter 'O'
		"0xBadFace",
		"0xBad_Face",
		"0x_67_7a_2f_cc_40_c6",
		"170141183460469231731687303715884105727",
		"170_141183_460469_231731_687303_715884_105727",
		"42_",        // invalid: _ must separate successive digits
		"4__2",       // invalid: only one _ at a time
		"0_xBadFace", // invalid: _ must separate successive digits
	}

	for _, num := range testCases {
		t.Run(num, func(t *testing.T) {
			expected, err := strconv.ParseInt(num, 0, 64)
			if err == nil {
				expectParse(t, num, func(p pfn) []parser.Stmt {
					return stmts(exprStmt(intLit(expected, p(1, 1))))
				})
			} else {
				expectParseError(t, num)
			}
		})
	}
}

func TestParseFloat(t *testing.T) {
	testCases := []string{
		// Different placements of decimal point
		".0",
		"0.",
		"0.0",
		"00.0",
		"00.00",
		"0.0.0",
		"0..0",

		// Ignoring leading zeros
		"010.0",
		"00010.0",
		"08.0",
		"0a.0", // ivalid: hex character

		// Exponents
		"1e1",
		"1E1",
		"1e1.1",
		"1e+1",
		"1e-1",
		"1e+-1",
		"0x1p1",
		"0x10p1",

		// Examples from language specifcation (https://go.dev/ref/spec#Floating-point_literals)
		"0.",
		"72.40",
		"072.40", // == 72.40
		"2.71828",
		"1.e+0",
		"6.67428e-11",
		"1E6",
		".25",
		".12345E+5",
		"1_5.",        // == 15.0
		"0.15e+0_2",   // == 15.0
		"0x1p-2",      // == 0.25
		"0x2.p10",     // == 2048.0
		"0x1.Fp+0",    // == 1.9375
		"0X.8p-0",     // == 0.5
		"0X_1FFFP-16", // == 0.1249847412109375
		"0x.p1",       // invalid: mantissa has no digits
		"1p-2",        // invalid: p exponent requires hexadecimal mantissa
		"0x1.5e-2",    // invalid: hexadecimal mantissa requires p exponent
		"1_.5",        // invalid: _ must separate successive digits
		"1._5",        // invalid: _ must separate successive digits
		"1.5_e1",      // invalid: _ must separate successive digits
		"1.5e_1",      // invalid: _ must separate successive digits
		"1.5e1_",      // invalid: _ must separate successive digits
	}

	for _, num := range testCases {
		t.Run(num, func(t *testing.T) {
			expected, err := strconv.ParseFloat(num, 64)
			if err == nil {
				expectParse(t, num, func(p pfn) []parser.Stmt {
					return stmts(exprStmt(floatLit(expected, p(1, 1))))
				})
			} else {
				expectParseError(t, num)
			}
		})
	}
}

func TestMismatchBrace(t *testing.T) {
	expectParseError(t, `
fmt := import("fmt")
out := 0
if 3 == 1 {
	out = 1
}
} else {
	out = 2
}
fmt.println(out)
	`)
}

func TestParseNumberExpressions(t *testing.T) {
	expectParse(t, `0x15e+2`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					intLit(0x15e, p(1, 1)),
					intLit(2, p(1, 7)),
					token.Add,
					p(1, 6))))
	})

	expectParse(t, `0-_42`, func(p pfn) []parser.Stmt {
		return stmts(
			exprStmt(
				binaryExpr(
					intLit(0, p(1, 1)),
					ident("_42", p(1, 3)),
					token.Sub,
					p(1, 2))))
	})
}

func TestIdentListString(t *testing.T) {
	identListVar := &parser.IdentList{
		List: []*parser.Ident{
			{Name: "a"},
			{Name: "b"},
			{Name: "c"},
		},
		VarArgs: true,
	}

	expectedVar := "(a, b, ...c)"
	if str := identListVar.String(); str != expectedVar {
		t.Fatalf("expected string of %#v to be %s, got %s",
			identListVar, expectedVar, str)
	}

	identList := &parser.IdentList{
		List: []*parser.Ident{
			{Name: "a"},
			{Name: "b"},
			{Name: "c"},
		},
		VarArgs: false,
	}

	expected := "(a, b, c)"
	if str := identList.String(); str != expected {
		t.Fatalf("expected string of %#v to be %s, got %s",
			identList, expected, str)
	}
}

func TestScanner_NoSemicolonBeforeSelector(t *testing.T) {
	input := "a\n  .b"
	testFile := testFileSet.AddFile("test", -1, len(input))

	s := parser.NewScanner(
		testFile,
		[]byte(input),
		func(_ parser.SourceFilePos, msg string) { require.Fail(t, msg) },
		0,
	)

	tok, _, _ := s.Scan()
	require.Equal(t, rta, token.Ident, tok)

	tok, _, _ = s.Scan()
	require.Equal(t, rta, token.Period, tok)

	tok, _, _ = s.Scan()
	require.Equal(t, rta, token.Ident, tok)
}
