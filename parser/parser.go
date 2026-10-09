package parser

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/ast/expression"
	"github.com/jokruger/kavun/ast/expression/composite"
	"github.com/jokruger/kavun/ast/expression/scalar"
	"github.com/jokruger/kavun/ast/statement"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
)

type bailout struct{}

var stmtStart = map[token.Token]bool{
	tokens.Break:    true,
	tokens.Continue: true,
	tokens.For:      true,
	tokens.If:       true,
	tokens.Return:   true,
	tokens.Export:   true,
	tokens.Var:      true,
}

// Error represents a parser error.
type Error struct {
	Pos ast.SourceFilePos
	Msg string
}

func (e Error) Error() string {
	if e.Pos.Filename != "" || e.Pos.IsValid() {
		return fmt.Sprintf("Parse Error: %s\n\tat %s", e.Msg, e.Pos)
	}
	return fmt.Sprintf("Parse Error: %s", e.Msg)
}

// ErrorList is a collection of parser errors.
type ErrorList []*Error

// Add adds a new parser error to the collection.
func (p *ErrorList) Add(pos ast.SourceFilePos, msg string) {
	*p = append(*p, &Error{pos, msg})
}

// Len returns the number of elements in the collection.
func (p ErrorList) Len() int {
	return len(p)
}

func (p ErrorList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p ErrorList) Less(i, j int) bool {
	e := &p[i].Pos
	f := &p[j].Pos

	if e.Filename != f.Filename {
		return e.Filename < f.Filename
	}
	if e.Line != f.Line {
		return e.Line < f.Line
	}
	if e.Column != f.Column {
		return e.Column < f.Column
	}
	return p[i].Msg < p[j].Msg
}

// Sort sorts the collection.
func (p ErrorList) Sort() {
	sort.Sort(p)
}

func (p ErrorList) Error() string {
	switch len(p) {
	case 0:
		return "no errors"
	case 1:
		return p[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", p[0], len(p)-1)
}

// Err returns an error.
func (p ErrorList) Err() error {
	if len(p) == 0 {
		return nil
	}
	return p
}

// Parser parses the Kavun source files.
type Parser struct {
	file      *ast.SourceFile
	errors    ErrorList
	scanner   *Scanner
	pos       core.Pos
	token     token.Token
	tokenLit  string
	exprLevel int      // < 0: in control clause, >= 0: in expression
	forInLHS  bool     // parsing potential for-in lhs; keep `in` for stmt parsing
	forInNest int      // parentheses nesting while parsing potential for-in lhs
	syncPos   core.Pos // last sync position
	syncCount int      // number of advance calls without progress
	trace     bool
	indent    int
	traceOut  io.Writer
}

// NewParser creates a Parser.
func NewParser(file *ast.SourceFile, src []byte, trace io.Writer) *Parser {
	p := &Parser{
		file:     file,
		trace:    trace != nil,
		traceOut: trace,
	}
	p.scanner = NewScanner(p.file, src,
		func(pos ast.SourceFilePos, msg string) {
			p.errors.Add(pos, msg)
		}, 0)
	p.next()
	return p
}

// ParseFile parses the source and returns an AST file unit.
func (p *Parser) ParseFile() (file *ast.File, err error) {
	defer func() {
		if e := recover(); e != nil {
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

		p.errors.Sort()
		err = p.errors.Err()
	}()

	if p.trace {
		defer untracep(tracep(p, "File"))
	}

	if p.errors.Len() > 0 {
		return nil, p.errors.Err()
	}

	stmts := p.parseStmtList()
	p.expect(tokens.EOF)
	if p.errors.Len() > 0 {
		return nil, p.errors.Err()
	}

	file = &ast.File{
		InputFile: p.file,
		Stmts:     stmts,
	}
	return
}

func (p *Parser) parseExpr() ast.Expression {
	if p.trace {
		defer untracep(tracep(p, "Expression"))
	}

	expr := p.parseBinaryExpr(token.LowestPrec + 1)

	// range literal: "low..high" or "low..high:step", sugar for range(low, high[, step]). Kept out of the
	// binary-operator precedence table (see token.Precedence) so plain parseBinaryExpr/parseExprNoRange never
	// consume it — that's what lets parseIndexOrSlice tell "arr[low..high]" (a slice) apart from "arr[<range
	// value>]" (an index). Building the range here rather than in parseBinaryExpr's own loop, then feeding the
	// result back into continueBinaryExpr, lets looser operators that follow attach normally, e.g.
	// "1..5 == range(1,5)" parses as "(1..5) == range(1,5)", not a dangling '==' after the range.
	if p.token == tokens.DotDot {
		expr = p.parseRangeExpr(expr)
		expr = p.continueBinaryExpr(expr, token.LowestPrec+1)
	}

	// ternary conditional expression
	if p.token == tokens.Question {
		return p.parseCondExpr(expr)
	}
	return expr
}

// parseExprNoRange parses a full expression except for the trailing "..high[:step]" range suffix. It is used by
// parseExpr (which adds the range suffix back on) and by parseIndexOrSlice, which must see an unconsumed DotDot
// token itself in order to treat it as an alternate low/high separator (see parseIndexOrSlice).
func (p *Parser) parseExprNoRange() ast.Expression {
	return p.parseBinaryExpr(token.LowestPrec + 1)
}

// rangeHighPrec is the minimum precedence used when parsing the "high" (and "step") operand of a bare range
// literal. It sits one above comparison operators' precedence (see token.Precedence) so that e.g.
// "for x in 1..n < len(arr)" parses as "(1..n) < len(arr)" rather than swallowing "n < len(arr)" into the range's
// high bound.
const rangeHighPrec = 4

// parseRangeExpr parses the "..high[:step]" suffix of a bare range literal, given the already-parsed low operand,
// producing an *expression.Range — a distinct node type from *expression.Slice, since a bare range is a
// value-producing expression (sugar for range(low, high[, step])), not an operation on a receiver. Only used outside
// of index/slice brackets (see parseExpr); arr[low..high] is handled entirely within parseIndexOrSlice instead,
// producing an *expression.Slice as always.
func (p *Parser) parseRangeExpr(low ast.Expression) ast.Expression {
	if p.trace {
		defer untracep(tracep(p, "RangeExpression"))
	}

	p.expect(tokens.DotDot)
	high := p.parseBinaryExpr(rangeHighPrec)

	var step ast.Expression
	if p.token == tokens.Colon {
		p.next()
		step = p.parseBinaryExpr(rangeHighPrec)
	}

	return &expression.Range{
		Low:  low,
		High: high,
		Step: step,
	}
}

func (p *Parser) parseBinaryExpr(prec1 int) ast.Expression {
	if p.trace {
		defer untracep(tracep(p, "BinaryExpression"))
	}

	return p.continueBinaryExpr(p.parseUnaryExpr(), prec1)
}

// continueBinaryExpr runs the binary-operator precedence climb starting from an already-parsed operand x, rather
// than parsing x itself via parseUnaryExpr. Factored out of parseBinaryExpr so parseExpr can build a range literal
// out-of-band (see parseRangeExpr) and then resume the climb with the range as the new left operand, letting looser
// trailing operators (comparison, logical) attach to it normally.
func (p *Parser) continueBinaryExpr(x ast.Expression, prec1 int) ast.Expression {
	for {
		op, prec := p.token, p.token.Precedence()
		if p.forInLHS && p.forInNest == 0 && op == tokens.In {
			return x
		}
		if op == tokens.NotKw && p.peekToken() == tokens.In {
			if p.forInLHS && p.forInNest == 0 {
				return x
			}

			const notInPrec = 3 // same precedence level as `in`
			if notInPrec < prec1 {
				return x
			}

			notPos := p.expect(tokens.NotKw)
			inPos := p.expect(tokens.In)
			y := p.parseBinaryExpr(notInPrec + 1)
			x = &expression.Unary{
				Token:    tokens.Not,
				TokenPos: notPos,
				Expr: &expression.Binary{
					LHS:      x,
					RHS:      y,
					Token:    tokens.In,
					TokenPos: inPos,
				},
			}
			continue
		}
		if prec < prec1 {
			return x
		}

		pos := p.expect(op)

		y := p.parseBinaryExpr(prec + 1)

		x = &expression.Binary{
			LHS:      x,
			RHS:      y,
			Token:    op,
			TokenPos: pos,
		}
	}
}

func (p *Parser) parseCondExpr(cond ast.Expression) ast.Expression {
	questionPos := p.expect(tokens.Question)
	trueExpr := p.parseExpr()
	colonPos := p.expect(tokens.Colon)
	falseExpr := p.parseExpr()

	return &expression.Ternary{
		Cond:        cond,
		True:        trueExpr,
		False:       falseExpr,
		QuestionPos: questionPos,
		ColonPos:    colonPos,
	}
}

func (p *Parser) parseUnaryExpr() ast.Expression {
	if p.trace {
		defer untracep(tracep(p, "UnaryExpression"))
	}

	switch p.token {
	case tokens.Add, tokens.Sub, tokens.Not, tokens.Xor:
		pos, op := p.pos, p.token
		p.next()
		x := p.parseUnaryExpr()
		return &expression.Unary{
			Token:    op,
			TokenPos: pos,
			Expr:     x,
		}
	}
	return p.parsePrimaryExpr()
}

func (p *Parser) parsePrimaryExpr() ast.Expression {
	if p.trace {
		defer untracep(tracep(p, "PrimaryExpression"))
	}

	x := p.parseOperand()

L:
	for {
		switch p.token {
		case tokens.Period:
			p.next()

			switch p.token {
			case tokens.Ident:
				x = p.parseSelector(x)
			default:
				pos := p.pos
				p.errorExpected(pos, "selector")
				p.advance(stmtStart)
				return &expression.Invalid{From: pos, To: p.pos}
			}
		case tokens.LBrack:
			x = p.parseIndexOrSlice(x)
		case tokens.LParen:
			x = p.parseCall(x)
		default:
			break L
		}
	}
	return x
}

func (p *Parser) parseCall(x ast.Expression) ast.Expression {
	if p.trace {
		defer untracep(tracep(p, "Call"))
	}

	lparen := p.expect(tokens.LParen)
	p.exprLevel++

	var list []ast.Expression
	var ellipsis core.Pos
	for p.token != tokens.RParen && p.token != tokens.EOF && !ellipsis.IsValid() {
		list = append(list, p.parseExpr())
		if p.token == tokens.Ellipsis {
			ellipsis = p.pos
			p.next()
		}
		if !p.expectComma(tokens.RParen, "call argument") {
			break
		}
	}

	p.exprLevel--
	rparen := p.expect(tokens.RParen)

	// Distinguish method call from regular function call.
	if sel, ok := x.(*expression.Selector); ok {
		// parseSelector currently stores selector as StringLit from an identifier.
		if method, ok := sel.Sel.(*scalar.String); ok {
			return &expression.MethodCall{
				Object:     sel.Expr,
				MethodName: method.Value,
				MethodPos:  method.ValuePos,
				LParen:     lparen,
				RParen:     rparen,
				Ellipsis:   ellipsis,
				Args:       list,
			}
		}
		// Defensive fallback: if selector is ever not identifier-based.
		return &expression.Invalid{From: sel.Pos(), To: rparen}
	}

	return &expression.Call{
		Func:     x,
		LParen:   lparen,
		RParen:   rparen,
		Ellipsis: ellipsis,
		Args:     list,
	}
}

func (p *Parser) expectComma(closing token.Token, want string) bool {
	if p.token == tokens.Comma {
		p.next()

		if p.token == closing {
			p.errorExpected(p.pos, want)
			return false
		}
		return true
	}

	if p.token == tokens.Semicolon && p.tokenLit == "\n" {
		p.next()
	}
	return false
}

func (p *Parser) parseIndexOrSlice(x ast.Expression) ast.Expression {
	if p.trace {
		defer untracep(tracep(p, "IndexOrSlice"))
	}

	lbrack := p.expect(tokens.LBrack)
	p.exprLevel++

	// "arr[low:high:step]" and "arr[low..high:step]" are equivalent spellings of the same slice: the low/high
	// separator may be ':' or '..', but the step separator (if present) is always ':'. Operands are parsed with
	// parseExprNoRange rather than parseExpr so a bare DotDot is left unconsumed for the checks below instead of
	// being swallowed into a range value by parseExpr's own range handling.
	var index [3]ast.Expression
	if p.token != tokens.Colon && p.token != tokens.DotDot {
		index[0] = p.parseExprNoRange()
	}
	numSeps := 0
	if p.token == tokens.Colon || p.token == tokens.DotDot {
		numSeps++
		p.next()

		if p.token != tokens.RBrack && p.token != tokens.EOF {
			if p.token != tokens.Colon {
				index[1] = p.parseExprNoRange()
			}
		}
	}
	if p.token == tokens.Colon {
		numSeps++
		p.next()

		if p.token != tokens.RBrack && p.token != tokens.EOF {
			index[2] = p.parseExprNoRange()
		}
	}

	p.exprLevel--
	rbrack := p.expect(tokens.RBrack)

	if numSeps > 0 {
		// slice expression
		return &expression.Slice{
			Expr:   x,
			LBrack: lbrack,
			RBrack: rbrack,
			Low:    index[0],
			High:   index[1],
			Step:   index[2],
		}
	}
	return &expression.Index{
		Expr:   x,
		LBrack: lbrack,
		RBrack: rbrack,
		Index:  index[0],
	}
}

func (p *Parser) parseSelector(x ast.Expression) ast.Expression {
	if p.trace {
		defer untracep(tracep(p, "Selector"))
	}

	sel := p.parseIdent()
	return &expression.Selector{Expr: x, Sel: &scalar.String{
		Value:    sel.Name,
		ValuePos: sel.NamePos,
		Literal:  sel.Name,
	}}
}

func (p *Parser) parseOperand() ast.Expression {
	if p.trace {
		defer untracep(tracep(p, "Operand"))
	}

	switch p.token {
	case tokens.Ident:
		// try parse as lambda
		if p.isLambdaHead() {
			return p.parseLambda()
		}

		// default to parsing as identifier
		return p.parseIdent()

	case tokens.Int:
		v, err := strconv.ParseInt(p.tokenLit, 0, 64)
		if err == strconv.ErrRange {
			p.error(p.pos, "number out of range")
		} else if err != nil {
			p.error(p.pos, "invalid integer")
		}
		x := &scalar.Int{
			Value:    v,
			ValuePos: p.pos,
			Literal:  p.tokenLit,
		}
		p.next()
		return x

	case tokens.Float:
		v, err := strconv.ParseFloat(strings.TrimSuffix(p.tokenLit, "f"), 64)
		if err == strconv.ErrRange {
			p.error(p.pos, "number out of range")
		} else if err != nil {
			p.error(p.pos, "invalid float")
		}
		x := &scalar.Float{
			Value:    v,
			ValuePos: p.pos,
			Literal:  p.tokenLit,
		}
		p.next()
		return x

	case tokens.Decimal:
		v := dec128.FromString(strings.TrimSuffix(p.tokenLit, "d"))
		if v.IsNaN() {
			p.error(p.pos, "invalid decimal literal")
		}
		x := &scalar.Decimal{
			Value:    v,
			ValuePos: p.pos,
			Literal:  p.tokenLit,
		}
		p.next()
		return x

	case tokens.Char:
		return p.parseCharLit()

	case tokens.ByteChar:
		return p.parseByteLit()

	case tokens.String:
		v, _ := strconv.Unquote(p.tokenLit)
		x := &scalar.String{
			Value:    v,
			ValuePos: p.pos,
			Literal:  p.tokenLit,
		}
		p.next()
		return x

	case tokens.RunesString:
		v, _ := strconv.Unquote(p.tokenLit)
		x := &scalar.Runes{
			// the same total decode the runtime uses: an octet the literal wrote that is not a
			// symbol (u"\xff") becomes its escape, never U+FFFD, so u"..." and "..." hold the
			// same text and round-trip to the same octets
			Value:    core.DecodeText(v),
			ValuePos: p.pos,
			Literal:  p.tokenLit,
		}
		p.next()
		return x

	case tokens.BytesString:
		v, _ := strconv.Unquote(p.tokenLit)
		x := &scalar.Bytes{
			Value:    []byte(v),
			ValuePos: p.pos,
			Literal:  p.tokenLit,
		}
		p.next()
		return x

	case tokens.TimeString:
		return p.parseTimeLit()

	case tokens.DateString:
		return p.parseDateLit()

	case tokens.RawString:
		// Strip surrounding quotes and only unescape \"
		raw := p.tokenLit[1 : len(p.tokenLit)-1]
		raw = strings.ReplaceAll(raw, `\"`, `"`)
		x := &scalar.String{
			Value:    raw,
			ValuePos: p.pos,
			Literal:  p.tokenLit,
		}
		p.next()
		return x

	case tokens.FString:
		x := p.parseFStringLit()
		p.next()
		return x

	case tokens.True:
		x := &scalar.Bool{
			Value:    true,
			ValuePos: p.pos,
			Literal:  p.tokenLit,
		}
		p.next()
		return x

	case tokens.False:
		x := &scalar.Bool{
			Value:    false,
			ValuePos: p.pos,
			Literal:  p.tokenLit,
		}
		p.next()
		return x

	case tokens.Undefined:
		x := &scalar.Undefined{TokenPos: p.pos}
		p.next()
		return x

	case tokens.Import:
		return p.parseImportExpr()

	case tokens.LParen:
		// try parse as lambda
		if p.isLambdaHead() {
			return p.parseLambda()
		}

		// default to parenthesized expression
		lparen := p.pos
		p.next()
		if p.forInLHS {
			p.forInNest++
		}
		p.exprLevel++
		x := p.parseExpr()
		p.exprLevel--
		if p.forInLHS {
			p.forInNest--
		}
		rparen := p.expect(tokens.RParen)
		return &expression.Parenthesis{
			LParen: lparen,
			Expr:   x,
			RParen: rparen,
		}

	case tokens.LBrack: // array literal
		return p.parseArrayLit()

	case tokens.LBrace: // record literal
		return p.parseRecordLit()

	case tokens.Func: // function literal
		return p.parseFuncLit()

	default:
		p.errorExpected(p.pos, "operand")
	}

	pos := p.pos
	p.advance(stmtStart)
	return &expression.Invalid{From: pos, To: p.pos}
}

func (p *Parser) parseImportExpr() ast.Expression {
	pos := p.pos
	p.next()
	p.expect(tokens.LParen)
	if p.token != tokens.String {
		p.errorExpected(p.pos, "module name")
		p.advance(stmtStart)
		return &expression.Invalid{From: pos, To: p.pos}
	}

	// module name
	moduleName, _ := strconv.Unquote(p.tokenLit)
	expr := &expression.Import{
		ModuleName: moduleName,
		Token:      tokens.Import,
		TokenPos:   pos,
	}

	p.next()
	p.expect(tokens.RParen)
	return expr
}

func (p *Parser) parseCharLit() ast.Expression {
	if n := len(p.tokenLit); n >= 3 {
		code, _, _, err := strconv.UnquoteChar(p.tokenLit[1:n-1], '\'')
		if err == nil {
			x := &scalar.Rune{
				Value:    code,
				ValuePos: p.pos,
				Literal:  p.tokenLit,
			}
			p.next()
			return x
		}
	}

	pos := p.pos
	p.error(pos, "illegal char literal")
	p.next()
	return &expression.Invalid{
		From: pos,
		To:   p.pos,
	}
}

func (p *Parser) parseByteLit() ast.Expression {
	if n := len(p.tokenLit); n >= 3 {
		code, _, _, err := strconv.UnquoteChar(p.tokenLit[1:n-1], '\'')
		if err == nil && code <= 255 {
			x := &scalar.Byte{
				Value:    byte(code),
				ValuePos: p.pos,
				Literal:  p.tokenLit,
			}
			p.next()
			return x
		}
	}

	pos := p.pos
	p.error(pos, "illegal byte literal")
	p.next()
	return &expression.Invalid{
		From: pos,
		To:   p.pos,
	}
}

func (p *Parser) parseTimeLit() ast.Expression {
	v, err := strconv.Unquote(p.tokenLit)
	if err == nil {
		parsed, perr := core.ParseTimeText(v)
		if perr == nil {
			x := &scalar.Time{
				Value:    parsed,
				ValuePos: p.pos,
				Literal:  p.tokenLit,
			}
			p.next()
			return x
		}
	}

	pos := p.pos
	p.error(pos, "illegal time literal")
	p.next()
	return &expression.Invalid{From: pos, To: p.pos}
}

func (p *Parser) parseDateLit() ast.Expression {
	v, err := strconv.Unquote(p.tokenLit)
	if err == nil {
		parsed, perr := core.ParseDateText(v)
		if perr == nil {
			x := &scalar.Date{
				Value:    parsed,
				ValuePos: p.pos,
				Literal:  p.tokenLit,
			}
			p.next()
			return x
		}
	}

	pos := p.pos
	p.error(pos, "illegal date literal")
	p.next()
	return &expression.Invalid{From: pos, To: p.pos}
}

func (p *Parser) parseFuncLit() ast.Expression {
	if p.trace {
		defer untracep(tracep(p, "FuncLit"))
	}

	typ := p.parseFuncType()
	p.exprLevel++
	body := p.parseBody()
	p.exprLevel--
	return &expression.Function{
		Type: typ,
		Body: body,
	}
}

func (p *Parser) parseArrayLit() ast.Expression {
	if p.trace {
		defer untracep(tracep(p, "ArrayLit"))
	}

	lbrack := p.expect(tokens.LBrack)
	p.exprLevel++

	var elements []ast.Expression
	for p.token != tokens.RBrack && p.token != tokens.EOF {
		elements = append(elements, p.parseExpr())

		if p.token == tokens.Comma {
			p.next()
			if p.token == tokens.RBrack {
				break
			}
			continue
		}

		if p.token == tokens.Semicolon && p.tokenLit == "\n" {
			p.next()
		}
		break
	}

	p.exprLevel--
	rbrack := p.expect(tokens.RBrack)
	return &composite.Array{
		Elements: elements,
		LBrack:   lbrack,
		RBrack:   rbrack,
	}
}

func (p *Parser) parseFuncType() *expression.FunctionType {
	if p.trace {
		defer untracep(tracep(p, "FuncType"))
	}

	pos := p.expect(tokens.Func)
	params := p.parseIdentList()
	var result *expression.Identifier
	if p.token == tokens.Ident {
		// Optional named result: `func(args) name { ... }`.
		// Disallow on a new line — the identifier must be on the same line as the closing paren of the parameter list.
		result = p.parseIdent()
	}
	return &expression.FunctionType{
		FuncPos: pos,
		Params:  params,
		Result:  result,
	}
}

func (p *Parser) parseBody() *statement.Block {
	if p.trace {
		defer untracep(tracep(p, "Body"))
	}

	lbrace := p.expect(tokens.LBrace)
	list := p.parseStmtList()
	rbrace := p.expect(tokens.RBrace)
	return &statement.Block{
		LBrace: lbrace,
		RBrace: rbrace,
		Stmts:  list,
	}
}

func (p *Parser) parseStmtList() (list []ast.Statement) {
	if p.trace {
		defer untracep(tracep(p, "StatementList"))
	}

	for p.token != tokens.RBrace && p.token != tokens.EOF {
		list = append(list, p.parseStmt())
	}
	return
}

func (p *Parser) parseIdent() *expression.Identifier {
	pos := p.pos
	name := "_"

	if p.token == tokens.Ident {
		name = p.tokenLit
		p.next()
	} else {
		p.expect(tokens.Ident)
	}
	return &expression.Identifier{
		NamePos: pos,
		Name:    name,
	}
}

func (p *Parser) parseIdentList() *expression.Identifiers {
	if p.trace {
		defer untracep(tracep(p, "IdentList"))
	}

	var params []*expression.Identifier
	lparen := p.expect(tokens.LParen)
	isVarArgs := false
	if p.token != tokens.RParen {
		if p.token == tokens.Ellipsis {
			isVarArgs = true
			p.next()
		}

		params = append(params, p.parseIdent())
		for !isVarArgs && p.token == tokens.Comma {
			p.next()
			if p.token == tokens.Ellipsis {
				isVarArgs = true
				p.next()
			}
			params = append(params, p.parseIdent())
		}
	}

	rparen := p.expect(tokens.RParen)
	return &expression.Identifiers{
		LParen:  lparen,
		RParen:  rparen,
		VarArgs: isVarArgs,
		List:    params,
	}
}

func (p *Parser) parseStmt() (stmt ast.Statement) {
	if p.trace {
		defer untracep(tracep(p, "Statement"))
	}

	switch p.token {
	case // simple statements
		tokens.Func, tokens.Ident, tokens.Int,
		tokens.Float, tokens.Decimal, tokens.Char, tokens.ByteChar, tokens.String, tokens.RunesString, tokens.BytesString,
		tokens.TimeString, tokens.DateString, tokens.RawString, tokens.FString, tokens.True, tokens.False,
		tokens.Undefined, tokens.Import, tokens.Var, tokens.LParen, tokens.LBrace,
		tokens.LBrack, tokens.Add, tokens.Sub, tokens.Mul, tokens.And, tokens.Xor,
		tokens.Not:
		s := p.parseSimpleStmt(false)
		p.expectSemi()
		return s
	case tokens.Return:
		return p.parseReturnStmt()
	case tokens.Defer:
		return p.parseDeferStmt()
	case tokens.Export:
		return p.parseExportStmt()
	case tokens.If:
		return p.parseIfStmt()
	case tokens.For:
		return p.parseForStmt()
	case tokens.Break, tokens.Continue:
		return p.parseBranchStmt(p.token)
	case tokens.Semicolon:
		s := &statement.Empty{Semicolon: p.pos, Implicit: p.tokenLit == "\n"}
		p.next()
		return s
	case tokens.RBrace:
		// semicolon may be omitted before a closing "}"
		return &statement.Empty{Semicolon: p.pos, Implicit: true}
	default:
		pos := p.pos
		p.errorExpected(pos, "statement")
		p.advance(stmtStart)
		return &statement.Invalid{From: pos, To: p.pos}
	}
}

func (p *Parser) parseForStmt() ast.Statement {
	if p.trace {
		defer untracep(tracep(p, "ForStmt"))
	}

	pos := p.expect(tokens.For)

	// for {}
	if p.token == tokens.LBrace {
		body := p.parseBlockStmt()
		p.expectSemi()

		return &statement.For{
			ForPos: pos,
			Body:   body,
		}
	}

	prevLevel := p.exprLevel
	p.exprLevel = -1

	var s1 ast.Statement
	if p.token != tokens.Semicolon { // skipping init
		s1 = p.parseSimpleStmt(true)
	}

	// for _ in seq {}            or
	// for value in seq {}        or
	// for key, value in seq {}
	if forInStmt, isForIn := s1.(*statement.ForIn); isForIn {
		forInStmt.ForPos = pos
		p.exprLevel = prevLevel
		forInStmt.Body = p.parseBlockStmt()
		p.expectSemi()
		return forInStmt
	}

	// for init; cond; post {}
	var s2, s3 ast.Statement
	if p.token == tokens.Semicolon {
		p.next()
		if p.token != tokens.Semicolon {
			s2 = p.parseSimpleStmt(false) // cond
		}
		p.expect(tokens.Semicolon)
		if p.token != tokens.LBrace {
			s3 = p.parseSimpleStmt(false) // post
		}
	} else {
		// for cond {}
		s2 = s1
		s1 = nil
	}

	// body
	p.exprLevel = prevLevel
	body := p.parseBlockStmt()
	p.expectSemi()
	cond := p.makeExpr(s2, "condition expression")
	return &statement.For{
		ForPos: pos,
		Init:   s1,
		Cond:   cond,
		Post:   s3,
		Body:   body,
	}
}

func (p *Parser) parseBranchStmt(tok token.Token) ast.Statement {
	if p.trace {
		defer untracep(tracep(p, "BranchStmt"))
	}

	pos := p.expect(tok)

	var label *expression.Identifier
	if p.token == tokens.Ident {
		label = p.parseIdent()
	}
	p.expectSemi()
	return &statement.Branch{
		Token:    tok,
		TokenPos: pos,
		Label:    label,
	}
}

func (p *Parser) parseIfStmt() ast.Statement {
	if p.trace {
		defer untracep(tracep(p, "IfStmt"))
	}

	pos := p.expect(tokens.If)
	init, cond := p.parseIfHeader()
	body := p.parseBlockStmt()

	var elseStmt ast.Statement
	if p.token == tokens.Else {
		p.next()

		switch p.token {
		case tokens.If:
			elseStmt = p.parseIfStmt()
		case tokens.LBrace:
			elseStmt = p.parseBlockStmt()
			p.expectSemi()
		default:
			p.errorExpected(p.pos, "if or {")
			elseStmt = &statement.Invalid{From: p.pos, To: p.pos}
		}
	} else {
		p.expectSemi()
	}
	return &statement.If{
		IfPos: pos,
		Init:  init,
		Cond:  cond,
		Body:  body,
		Else:  elseStmt,
	}
}

func (p *Parser) parseBlockStmt() *statement.Block {
	if p.trace {
		defer untracep(tracep(p, "BlockStmt"))
	}

	lbrace := p.expect(tokens.LBrace)
	list := p.parseStmtList()
	rbrace := p.expect(tokens.RBrace)
	return &statement.Block{
		LBrace: lbrace,
		RBrace: rbrace,
		Stmts:  list,
	}
}

func (p *Parser) parseIfHeader() (init ast.Statement, cond ast.Expression) {
	if p.token == tokens.LBrace {
		p.error(p.pos, "missing condition in if statement")
		cond = &expression.Invalid{From: p.pos, To: p.pos}
		return
	}

	outer := p.exprLevel
	p.exprLevel = -1
	if p.token == tokens.Semicolon {
		p.error(p.pos, "missing init in if statement")
		return
	}
	init = p.parseSimpleStmt(false)

	var condStmt ast.Statement
	switch p.token {
	case tokens.LBrace:
		condStmt = init
		init = nil
	case tokens.Semicolon:
		p.next()
		condStmt = p.parseSimpleStmt(false)
	default:
		p.error(p.pos, "missing condition in if statement")
	}

	if condStmt != nil {
		cond = p.makeExpr(condStmt, "boolean expression")
	}
	if cond == nil {
		cond = &expression.Invalid{From: p.pos, To: p.pos}
	}
	p.exprLevel = outer
	return
}

func (p *Parser) makeExpr(s ast.Statement, want string) ast.Expression {
	if s == nil {
		return nil
	}

	if es, isExpr := s.(*statement.Expression); isExpr {
		return es.Expr
	}

	found := "simple statement"
	if _, isAss := s.(*statement.Assign); isAss {
		found = "assignment"
	}
	p.error(s.Pos(), fmt.Sprintf("expected %s, found %s", want, found))
	return &expression.Invalid{From: s.Pos(), To: p.safePos(s.End())}
}

func (p *Parser) parseReturnStmt() ast.Statement {
	if p.trace {
		defer untracep(tracep(p, "ReturnStmt"))
	}

	pos := p.pos
	p.expect(tokens.Return)

	var x ast.Expression
	if p.token != tokens.Semicolon && p.token != tokens.RBrace {
		x = p.parseExpr()
	}
	p.expectSemi()
	return &statement.Return{
		ReturnPos: pos,
		Result:    x,
	}
}

func (p *Parser) parseDeferStmt() ast.Statement {
	if p.trace {
		defer untracep(tracep(p, "DeferStmt"))
	}

	pos := p.expect(tokens.Defer)
	x := p.parseExpr()
	p.expectSemi()

	switch x.(type) {
	case *expression.Call, *expression.MethodCall:
		return &statement.Defer{DeferPos: pos, Call: x}
	}
	p.error(x.Pos(), "expression in defer must be a function or method call")
	return &statement.Invalid{From: pos, To: p.safePos(x.End())}
}

func (p *Parser) parseExportStmt() ast.Statement {
	if p.trace {
		defer untracep(tracep(p, "ExportStmt"))
	}

	pos := p.pos
	p.expect(tokens.Export)
	x := p.parseExpr()
	p.expectSemi()
	return &statement.Export{
		ExportPos: pos,
		Result:    x,
	}
}

func (p *Parser) parseVarStmt() ast.Statement {
	if p.trace {
		defer untracep(tracep(p, "VarStmt"))
	}

	pos := p.expect(tokens.Var)
	ident := p.parseIdent()

	rhs := ast.Expression(&scalar.Undefined{TokenPos: pos})
	if p.token == tokens.Assign {
		p.next()
		rhs = p.parseExpr()
	}

	return &statement.Assign{
		LHS:      []ast.Expression{ident},
		RHS:      []ast.Expression{rhs},
		Token:    tokens.Define,
		TokenPos: pos,
	}
}

func (p *Parser) parseSimpleStmt(forIn bool) ast.Statement {
	if p.trace {
		defer untracep(tracep(p, "SimpleStmt"))
	}

	if p.token == tokens.Var {
		return p.parseVarStmt()
	}

	prevForInLHS := p.forInLHS
	if forIn {
		p.forInLHS = true
		p.forInNest = 0
	}
	x := p.parseExprList()
	p.forInLHS = prevForInLHS
	if !p.forInLHS {
		p.forInNest = 0
	}

	switch p.token {
	case tokens.Assign, tokens.Define: // assignment statement
		pos, tok := p.pos, p.token
		p.next()
		y := p.parseExprList()
		return &statement.Assign{
			LHS:      x,
			RHS:      y,
			Token:    tok,
			TokenPos: pos,
		}
	case tokens.In:
		if forIn {
			p.next()
			y := p.parseExpr()

			var key, value *expression.Identifier
			var ok bool
			solo := false
			switch len(x) {
			case 1:
				solo = true
				key = &expression.Identifier{Name: "_", NamePos: x[0].Pos()}

				value, ok = x[0].(*expression.Identifier)
				if !ok {
					p.errorExpected(x[0].Pos(), "identifier")
					value = &expression.Identifier{Name: "_", NamePos: x[0].Pos()}
				}
			case 2:
				key, ok = x[0].(*expression.Identifier)
				if !ok {
					p.errorExpected(x[0].Pos(), "identifier")
					key = &expression.Identifier{Name: "_", NamePos: x[0].Pos()}
				}
				value, ok = x[1].(*expression.Identifier)
				if !ok {
					p.errorExpected(x[1].Pos(), "identifier")
					value = &expression.Identifier{Name: "_", NamePos: x[1].Pos()}
				}
			}
			return &statement.ForIn{
				Key:      key,
				Value:    value,
				Iterable: y,
				Solo:     solo,
			}
		}
	}

	if len(x) > 1 {
		p.errorExpected(x[0].Pos(), "1 expression")
		// continue with first expression
	}

	switch p.token {
	case tokens.Define,
		tokens.AddAssign, tokens.SubAssign, tokens.MulAssign, tokens.QuoAssign,
		tokens.RemAssign, tokens.AndAssign, tokens.OrAssign, tokens.XorAssign,
		tokens.ShlAssign, tokens.ShrAssign, tokens.AndNotAssign:
		pos, tok := p.pos, p.token
		p.next()
		y := p.parseExpr()
		return &statement.Assign{
			LHS:      []ast.Expression{x[0]},
			RHS:      []ast.Expression{y},
			Token:    tok,
			TokenPos: pos,
		}
	case tokens.Inc, tokens.Dec:
		// increment or decrement statement
		s := &statement.IncDec{Expr: x[0], Token: p.token, TokenPos: p.pos}
		p.next()
		return s
	}
	return &statement.Expression{Expr: x[0]}
}

func (p *Parser) parseExprList() (list []ast.Expression) {
	if p.trace {
		defer untracep(tracep(p, "ExpressionList"))
	}

	list = append(list, p.parseExpr())
	for p.token == tokens.Comma {
		p.next()
		list = append(list, p.parseExpr())
	}
	return
}

func (p *Parser) parseRecordElementLit() *composite.RecordElement {
	if p.trace {
		defer untracep(tracep(p, "RecordElementLit"))
	}

	pos := p.pos
	name := "_"
	switch p.token {
	case tokens.Ident:
		name = p.tokenLit
	case tokens.String:
		v, _ := strconv.Unquote(p.tokenLit)
		name = v
	default:
		p.errorExpected(pos, "record key")
	}

	p.next()
	colonPos := p.expect(tokens.Colon)
	valueExpr := p.parseExpr()
	return &composite.RecordElement{
		Key:      name,
		KeyPos:   pos,
		ColonPos: colonPos,
		Value:    valueExpr,
	}
}

func (p *Parser) parseRecordLit() *composite.Record {
	if p.trace {
		defer untracep(tracep(p, "RecordLit"))
	}

	lbrace := p.expect(tokens.LBrace)
	p.exprLevel++

	var elements []*composite.RecordElement
	for p.token != tokens.RBrace && p.token != tokens.EOF {
		elements = append(elements, p.parseRecordElementLit())

		if p.token == tokens.Comma {
			p.next()
			if p.token == tokens.RBrace {
				break
			}
			continue
		}

		if p.token == tokens.Semicolon && p.tokenLit == "\n" {
			p.next()
		}
		break
	}

	p.exprLevel--
	rbrace := p.expect(tokens.RBrace)
	return &composite.Record{
		LBrace:   lbrace,
		RBrace:   rbrace,
		Elements: elements,
	}
}

func (p *Parser) parseLambda() ast.Expression {
	if p.trace {
		defer untracep(tracep(p, "Lambda"))
	}

	var params *expression.Identifiers
	switch p.token {
	case tokens.Ident: // x =>
		fpos := p.pos
		arg := p.parseIdent()
		params = &expression.Identifiers{
			LParen:  fpos,
			VarArgs: false,
			List:    []*expression.Identifier{arg},
			RParen:  p.pos,
		}
	case tokens.LParen: // () =>
		params = p.parseIdentList()
	default:
		p.errorExpected(p.pos, "lambda parameter list")
	}

	apos := p.expect(tokens.Arrow)

	var body *statement.Block
	bkp := p.scanner.Backup()
	if p.token == tokens.LBrace {
		// => { ... }
		p.scanner.Restore(bkp)
		body = p.parseBody()
	} else {
		// => expr
		p.scanner.Restore(bkp)
		expr := p.parseExpr()
		body = &statement.Block{
			LBrace: apos,
			RBrace: expr.End(),
			Stmts: []ast.Statement{
				&statement.Return{
					ReturnPos: apos,
					Result:    expr,
				},
			},
		}
	}

	return &expression.Function{
		Type: &expression.FunctionType{
			FuncPos: params.LParen,
			Params:  params,
		},
		Body: body,
	}
}

func (p *Parser) isLambdaHead() bool {
	// x =>
	if p.token == tokens.Ident {
		bkp := p.scanner.Backup()
		t, _, _ := p.scanner.Scan()
		if t == tokens.Arrow {
			p.scanner.Restore(bkp)
			return true
		}
		p.scanner.Restore(bkp)
	}

	// () =>
	if p.token == tokens.LParen {
		bkp := p.scanner.Backup()
		t, _, _ := p.scanner.Scan()
		arg := false
		for {
			switch {
			case t == tokens.RParen:
				t, _, _ = p.scanner.Scan()
				if t == tokens.Arrow {
					p.scanner.Restore(bkp)
					return true
				}
				p.scanner.Restore(bkp)
				return false

			case t == tokens.Ident && !arg:
				t, _, _ = p.scanner.Scan()
				arg = true
				continue

			case t == tokens.Comma && arg:
				t, _, _ = p.scanner.Scan()
				arg = false
				continue

			default:
				p.scanner.Restore(bkp)
				return false
			}
		}
	}

	return false
}

func (p *Parser) expect(token token.Token) core.Pos {
	pos := p.pos

	if p.token != token {
		p.errorExpected(pos, "'"+token.String()+"'")
	}
	p.next()
	return pos
}

func (p *Parser) expectSemi() {
	switch p.token {
	case tokens.RParen, tokens.RBrace:
		// semicolon is optional before a closing ')' or '}'
	case tokens.Comma:
		// permit a ',' instead of a ';' but complain
		p.errorExpected(p.pos, "';'")
		fallthrough
	case tokens.Semicolon:
		p.next()
	default:
		p.errorExpected(p.pos, "';'")
		p.advance(stmtStart)
	}
}

func (p *Parser) advance(to map[token.Token]bool) {
	for ; p.token != tokens.EOF; p.next() {
		if to[p.token] {
			if p.pos == p.syncPos && p.syncCount < 10 {
				p.syncCount++
				return
			}
			if p.pos > p.syncPos {
				p.syncPos = p.pos
				p.syncCount = 0
				return
			}
		}
	}
}

func (p *Parser) error(pos core.Pos, msg string) {
	filePos := p.file.Position(pos)

	n := len(p.errors)
	if n > 0 && p.errors[n-1].Pos.Line == filePos.Line {
		// discard errors reported on the same line
		return
	}
	if n > 10 {
		// too many errors; terminate early
		panic(bailout{})
	}
	p.errors.Add(filePos, msg)
}

func (p *Parser) errorExpected(pos core.Pos, msg string) {
	msg = "expected " + msg
	if pos == p.pos {
		// error happened at the current position: provide more specific
		switch {
		case p.token == tokens.Semicolon && p.tokenLit == "\n":
			msg += ", found newline"
		case p.token.IsLiteral():
			msg += ", found " + p.tokenLit
		default:
			msg += ", found '" + p.token.String() + "'"
		}
	}
	p.error(pos, msg)
}

func (p *Parser) next() {
	if p.trace && p.pos.IsValid() {
		s := p.token.String()
		switch {
		case p.token.IsLiteral():
			p.printTrace(s, p.tokenLit)
		case p.token.IsOperator(), p.token.IsKeyword():
			p.printTrace(`"` + s + `"`)
		default:
			p.printTrace(s)
		}
	}
	p.token, p.tokenLit, p.pos = p.scanner.Scan()
}

func (p *Parser) peekToken() token.Token {
	bkp := p.scanner.Backup()
	tok, _, _ := p.scanner.Scan()
	p.scanner.Restore(bkp)
	return tok
}

func (p *Parser) printTrace(a ...any) {
	const (
		dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
		n    = len(dots)
	)

	filePos := p.file.Position(p.pos)
	_, _ = fmt.Fprintf(p.traceOut, "%5d: %5d:%3d: ", p.pos, filePos.Line,
		filePos.Column)
	i := 2 * p.indent
	for i > n {
		_, _ = fmt.Fprint(p.traceOut, dots)
		i -= n
	}
	_, _ = fmt.Fprint(p.traceOut, dots[0:i])
	_, _ = fmt.Fprintln(p.traceOut, a...)
}

func (p *Parser) safePos(pos core.Pos) core.Pos {
	fileBase := p.file.Base
	fileSize := p.file.Size

	if int(pos) < fileBase || int(pos) > fileBase+fileSize {
		return core.Pos(fileBase + fileSize)
	}
	return pos
}

func tracep(p *Parser, msg string) *Parser {
	p.printTrace(msg, "(")
	p.indent++
	return p
}

func untracep(p *Parser) {
	p.indent--
	p.printTrace(")")
}
