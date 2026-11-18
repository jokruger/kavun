package parser

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/ast/expression"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/text"
)

// parseFStringLit parses a tokens.FString token (already in p.tokenLit / p.pos). The literal is the original source
// including surrounding double quotes — i.e. for `f"hello {x:>5}"` the literal is `"hello {x:>5}"` and p.pos points to
// the opening `"` (the leading `f` is one byte before).
//
// The result is an *expression.FString with all literal segments unescaped and all interpolation expressions sub-parsed; format
// specs are parsed via fspec.Parse so any spec error is reported at compile time. The caller is expected to advance
// past the FString token after this returns.
func (p *Parser) parseFStringLit() ast.Expression {
	startPos := p.pos
	lit := p.tokenLit
	if len(lit) < 2 || lit[0] != '"' || lit[len(lit)-1] != '"' {
		p.error(startPos, "malformed f-string literal")
		return &expression.Invalid{From: startPos, To: startPos + 1 + core.Pos(len(lit))}
	}
	body := lit[1 : len(lit)-1]
	// position of the first character of the body: the token's position is its leading 'f', then the opening '"'
	bodyStart := startPos + 2

	parts, err := splitFString(body, bodyStart, p)
	if err != nil {
		var pe *fstringPosError
		if errors.As(err, &pe) {
			p.error(pe.pos, pe.msg)
		} else {
			p.error(startPos, err.Error())
		}
		return &expression.Invalid{From: startPos, To: startPos + 1 + core.Pos(len(lit))}
	}

	return &expression.FString{
		Parts:    parts,
		ValuePos: startPos,
		// EndPos is one past the trailing '"': the token starts at its leading 'f' and lit is the quoted body, so
		// the literal occupies [startPos, startPos+1+len(lit)).
		EndPos:  startPos + 1 + core.Pos(len(lit)),
		Literal: lit,
	}
}

// splitFString walks the f-string body once and produces its parts.
//
// body is the source between (but not including) the surrounding double quotes;
// bodyOffset is the absolute parser position of body[0].
//
// Sub-parsing of expression text uses p.file as the source file so error positions are reported within the original
// file.
func splitFString(body string, bodyOffset core.Pos, p *Parser) ([]expression.FStringPart, error) {
	var parts []expression.FStringPart
	var litBuf bytes.Buffer
	flushLiteral := func() error {
		if litBuf.Len() == 0 {
			return nil
		}
		// Process backslash escapes via strconv.Unquote on a synthetic quoted string. This matches the semantics of
		// regular `"..."` literals; each escape was already checked, with its position, by checkStringEscape.
		quoted := "\"" + litBuf.String() + "\""
		s, err := strconv.Unquote(quoted)
		if err != nil {
			return fmt.Errorf("invalid escape sequence in f-string: %v", err)
		}
		parts = append(parts, expression.FStringPart{Literal: s})
		litBuf.Reset()
		return nil
	}

	i := 0
	n := len(body)
	for i < n {
		ch := body[i]
		switch ch {
		case '\\':
			// the scanner steps over every escape of an f-string; the literal text's are checked here, by the
			// string rule, and reported at the escape
			if off, msg := checkStringEscape(body[i:]); msg != "" {
				return nil, &fstringPosError{pos: bodyOffset + core.Pos(i+off), msg: msg}
			}
			// preserve escape sequences as-is in the buffer; strconv.Unquote handles them when we flush.
			litBuf.WriteByte(ch)
			i++
			if i < n {
				litBuf.WriteByte(body[i])
				i++
			}
		case '{':
			if i+1 < n && body[i+1] == '{' {
				litBuf.WriteByte('{')
				i += 2
				continue
			}
			if err := flushLiteral(); err != nil {
				return nil, err
			}
			// find matching '}' (depth-aware over braces, parens, brackets, and respecting nested quoted strings inside
			// the expression)
			end, err := findFStringExprEnd(body, i+1)
			if err != nil {
				return nil, err
			}
			inner := body[i+1 : end]
			exprText, specText, hasSpec := splitFStringExprAndSpec(inner)
			if strings.TrimSpace(exprText) == "" {
				return nil, fmt.Errorf("f-string: empty expression in '{}'")
			}
			// The expression is written exactly as outside the f-string, except that its `"` is written `\"`
			// (the f-string body is a "..." token): undo only that before the sub-parser sees it.
			specOffset := i + 1 + len(exprText) + 1
			exprText = unescapeFStringExpr(exprText)
			expr, perr := p.parseFStringExpr(exprText, bodyOffset+core.Pos(i+1))
			if perr != nil {
				return nil, perr
			}
			specLiterals, specExprs, specErr := splitDynamicSpec(specText, bodyOffset, specOffset, p)
			if specErr != nil {
				return nil, specErr
			}
			part := expression.FStringPart{
				Expr:     expr,
				SpecText: specText,
			}
			if len(specExprs) > 0 {
				// Dynamic spec: defer parsing to run time.
				part.SpecLiterals = specLiterals
				part.SpecExprs = specExprs
			} else {
				// Static spec: parse at compile time so any error is reported now.
				spec, ferr := fspec.Parse(specText)
				if ferr != nil {
					return nil, fmt.Errorf("f-string format spec %q: %v", specText, ferr)
				}
				part.Spec = spec
			}
			_ = hasSpec
			parts = append(parts, part)
			i = end + 1 // skip '}'
		case '}':
			if i+1 < n && body[i+1] == '}' {
				litBuf.WriteByte('}')
				i += 2
				continue
			}
			return nil, fmt.Errorf("f-string: single '}' is not allowed; use '}}' for a literal '}'")
		default:
			litBuf.WriteByte(ch)
			i++
		}
	}
	if err := flushLiteral(); err != nil {
		return nil, err
	}
	return parts, nil
}

// unescapeFStringExpr turns the text of an f-string interpolation back into the expression it spells. Inside
// `{...}` an expression is written exactly as outside the f-string, except that `"` is written `\"` — so `\"`
// is the one sequence undone; every other backslash sequence (`\n`, `\xff`, `\\`) reaches the sub-parser
// untouched and means there what it means anywhere else. Read pairwise, so the `\"` of `\\\"` is found and
// the `\\` before it is kept.
func unescapeFStringExpr(s string) string {
	if !strings.Contains(s, `\"`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			if s[i+1] == '"' {
				b.WriteByte('"')
			} else {
				b.WriteByte('\\')
				b.WriteByte(s[i+1])
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// fstringPosError is an f-string error reported at its own position rather than at the f-string's start.
type fstringPosError struct {
	pos core.Pos
	msg string
}

func (e *fstringPosError) Error() string { return e.msg }

// checkStringEscape checks the escape at the start of s (s[0] is the backslash) by the rule of a string literal,
// as the scanner checks one: it answers "" when the escape is well-formed, else the scanner's message and the
// offset in s it reports at (the character after the backslash, or the offending digit).
func checkStringEscape(s string) (off int, msg string) {
	if len(s) < 2 {
		return 1, "escape sequence not terminated"
	}
	var n int
	var base, max uint32
	i := 2
	switch s[1] {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '"':
		return 0, ""
	case '0', '1', '2', '3', '4', '5', '6', '7':
		n, base, max, i = 3, 8, 255, 1
	case 'x':
		n, base, max = 2, 16, 255
	case 'u':
		n, base, max = 4, 16, unicode.MaxRune
	case 'U':
		n, base, max = 8, 16, unicode.MaxRune
	default:
		return 1, "unknown escape sequence"
	}
	var x uint32
	for ; n > 0; n-- {
		if i >= len(s) {
			return i, "escape sequence not terminated"
		}
		ch, _ := utf8.DecodeRuneInString(s[i:])
		d := uint32(digitVal(ch))
		if d >= base {
			return i, fmt.Sprintf("illegal character %#U in escape sequence", ch)
		}
		x = x*base + d
		i++
	}
	if x > max {
		return 1, "escape sequence is invalid Unicode code point"
	}
	if 0xD800 <= x && x < 0xE000 {
		if text.IsEscapeRune(rune(x)) {
			return 1, octetEscapeHint(s[:i], rune(x))
		}
		return 1, "escape sequence is invalid Unicode code point"
	}
	return 0, ""
}

// skipFStringStringLit advances past a string literal embedded in an f-string interpolation expression.
// In the f-string body, the surrounding quote of an embedded `"..."` literal is written `\"` (because a bare
// `"` would terminate the f-string itself), while `'...'` rune literals appear with bare `'` delimiters since
// the f-string scanner does not stop on `'`.
//
// `i` points at the opening delimiter byte (a `\` for `\"` or a `'` for `'`). On success it returns the index
// just past the closing delimiter. On unterminated string it returns an error.
func skipFStringStringLit(body string, i int) (int, error) {
	n := len(body)
	var openLen int     // 1 for ', 2 for \"
	var closeQuote byte // " or '
	if body[i] == '\\' {
		if i+1 >= n || (body[i+1] != '"' && body[i+1] != '\'') {
			return 0, fmt.Errorf("internal: skipFStringStringLit called on non-quote escape")
		}
		closeQuote = body[i+1]
		openLen = 2
	} else {
		closeQuote = body[i]
		openLen = 1
	}
	i += openLen
	for i < n {
		switch body[i] {
		case '\n':
			return 0, fmt.Errorf("f-string: unterminated string inside interpolation")
		case '\\':
			// inside an embedded "..." literal the closer is `\"`; for embedded '...' (rune) the closer is bare `'`.
			if openLen == 2 && i+1 < n && body[i+1] == closeQuote {
				return i + 2, nil
			}
			// any other `\X` escape sequence – skip 2 bytes so e.g. `\\` doesn't accidentally pair with a later quote.
			if i+1 < n {
				i += 2
			} else {
				i++
			}
		default:
			if openLen == 1 && body[i] == closeQuote {
				return i + 1, nil
			}
			i++
		}
	}
	return 0, fmt.Errorf("f-string: unterminated string inside interpolation")
}

// findFStringExprEnd returns the index of the '}' that closes the '{' whose content begins at start. It tracks balanced
// (), [], {} and skips quoted strings ("..." and '...') so a colon or '}' inside e.g. a record literal or a string does
// not prematurely terminate the expression.
func findFStringExprEnd(body string, start int) (int, error) {
	depth := 0
	i := start
	n := len(body)
	for i < n {
		ch := body[i]
		switch ch {
		case '\\':
			// `\"` opens a string literal in the underlying expression; any other `\X` is an opaque escape.
			if i+1 < n && (body[i+1] == '"' || body[i+1] == '\'') {
				next, err := skipFStringStringLit(body, i)
				if err != nil {
					return 0, err
				}
				i = next
			} else if i+1 < n {
				i += 2
			} else {
				i++
			}
		case '\'':
			next, err := skipFStringStringLit(body, i)
			if err != nil {
				return 0, err
			}
			i = next
		case '(', '[', '{':
			depth++
			i++
		case ')', ']':
			depth--
			i++
		case '}':
			if depth == 0 {
				return i, nil
			}
			depth--
			i++
		default:
			i++
		}
	}
	return 0, fmt.Errorf("f-string: missing '}' to close interpolation")
}

// splitFStringExprAndSpec splits the inside of a `{...}` placeholder into the expression text and the optional
// format-spec text. The split is the first top-level ':' (i.e. ':' that is not inside parens/brackets/braces or a
// quoted string). A leading '::' style does not occur because Kavun does not have a `::` operator inside expressions.
func splitFStringExprAndSpec(inner string) (expr, spec string, hasSpec bool) {
	depth := 0
	ternary := 0
	i := 0
	n := len(inner)
	for i < n {
		ch := inner[i]
		switch ch {
		case '\\':
			if i+1 < n && (inner[i+1] == '"' || inner[i+1] == '\'') {
				if next, err := skipFStringStringLit(inner, i); err == nil {
					i = next
				} else {
					i += 2
				}
			} else if i+1 < n {
				i += 2
			} else {
				i++
			}
		case '\'':
			if next, err := skipFStringStringLit(inner, i); err == nil {
				i = next
			} else {
				i++
			}
		case '(', '[', '{':
			depth++
			i++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
			i++
		case '?':
			if depth == 0 {
				ternary++
			}
			i++
		case ':':
			if depth == 0 {
				if ternary > 0 {
					ternary--
					i++
					continue
				}
				return inner[:i], inner[i+1:], true
			}
			i++
		default:
			i++
		}
	}
	return inner, "", false
}

// splitDynamicSpec scans a format-spec text for nested `{...}` placeholders (Python-style dynamic format specs).
//
// On success it returns:
//   - literals: len(N+1) literal text segments
//   - exprs:    len(N) parsed Kavun expressions corresponding to each `{...}` placeholder
//
// The runtime spec string is then `literals[0] + str(exprs[0]) + literals[1] + ... + literals[N]`.
//
// When specText contains no `{`, both slices are nil — the caller should fall back to the static path.
//
// Only one level of nesting is allowed: a `{` inside a placeholder body is a compile error.
//
// bodyOffset+specOffset is the absolute parser position of specText[0] in the original source file; it is used to make
// any sub-parser error refer to the correct location.
func splitDynamicSpec(specText string, bodyOffset core.Pos, specOffset int, p *Parser) (literals []string, exprs []ast.Expression, err error) {
	if !strings.ContainsRune(specText, '{') && !strings.ContainsRune(specText, '}') {
		return nil, nil, nil
	}

	var lit bytes.Buffer
	i := 0
	n := len(specText)
	for i < n {
		ch := specText[i]
		switch ch {
		case '{':
			// `{{` is a literal `{` (mirrors the outer f-string body convention).
			if i+1 < n && specText[i+1] == '{' {
				lit.WriteByte('{')
				i += 2
				continue
			}
			// flush current literal segment
			literals = append(literals, lit.String())
			lit.Reset()
			// find matching '}' — nested '{' is forbidden
			j := i + 1
			for j < n && specText[j] != '}' {
				if specText[j] == '{' {
					return nil, nil, fmt.Errorf("f-string: nested '{' inside format spec is not allowed (only one level of nesting)")
				}
				j++
			}
			if j >= n {
				return nil, nil, fmt.Errorf("f-string: missing '}' to close format-spec interpolation")
			}
			inner := specText[i+1 : j]
			if strings.TrimSpace(inner) == "" {
				return nil, nil, fmt.Errorf("f-string: empty expression in format spec '{}'")
			}
			// the same rule as the expression: only `\"` is undone before sub-parsing.
			inner = unescapeFStringExpr(inner)
			expr, perr := p.parseFStringExpr(inner, bodyOffset+core.Pos(specOffset+i+1))
			if perr != nil {
				return nil, nil, perr
			}
			exprs = append(exprs, expr)
			i = j + 1
		case '}':
			if i+1 < n && specText[i+1] == '}' {
				lit.WriteByte('}')
				i += 2
				continue
			}
			return nil, nil, fmt.Errorf("f-string: unexpected '}' inside format spec")
		default:
			lit.WriteByte(ch)
			i++
		}
	}
	literals = append(literals, lit.String())
	return literals, exprs, nil
}

// parseFStringExpr parses a Kavun expression embedded inside an f-string.
//
// origin is the absolute parser position where exprText begins in the containing source file; this is used so any
// sub-parser error refers to a position inside the original file rather than a synthetic file.
func (p *Parser) parseFStringExpr(exprText string, origin core.Pos) (ast.Expression, error) {
	// Build a temporary SourceFile that shares the same FileSet so that positions reported by the sub-parser remain
	// meaningful in error messages produced by the host parser. We use a brand-new file because we want the
	// sub-parser's scanner to start at offset 0; that's fine because we report any errors with the host parser's
	// `error` helper using the f-string's overall position.
	subFile := p.file.Set().AddFile("<fstring>", -1, len(exprText))
	src := []byte(exprText)
	// Add a trailing newline so the sub-parser's expression scan terminates cleanly at EOF.
	sub := NewParser(subFile, src, nil)
	if sub.token == tokens.EOF {
		return nil, fmt.Errorf("f-string: empty expression in '{}'")
	}
	expr := sub.parseExpr()
	if sub.token != tokens.EOF && !(sub.token == tokens.Semicolon && sub.tokenLit == "\n") {
		return nil, fmt.Errorf("f-string: unexpected token %q after expression %q", sub.tokenLit, exprText)
	}
	if sub.errors.Len() > 0 {
		return nil, fmt.Errorf("f-string expression %q: %v", exprText, sub.errors.Err())
	}
	_ = origin
	return expr, nil
}
