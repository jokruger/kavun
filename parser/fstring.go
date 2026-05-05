package parser

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/token"
)

// parseFStringLit parses a token.FString token (already in p.tokenLit / p.pos). The literal is the original source
// including surrounding double quotes — i.e. for `f"hello {x:>5}"` the literal is `"hello {x:>5}"` and p.pos points to
// the opening `"` (the leading `f` is one byte before).
//
// The result is an *FStringLit with all literal segments unescaped and all interpolation expressions sub-parsed; format
// specs are parsed via fspec.Parse so any spec error is reported at compile time. The caller is expected to advance
// past the FString token after this returns.
func (p *Parser) parseFStringLit() Expr {
	startPos := p.pos
	lit := p.tokenLit
	if len(lit) < 2 || lit[0] != '"' || lit[len(lit)-1] != '"' {
		p.error(startPos, "malformed f-string literal")
		return &BadExpr{From: startPos, To: startPos + core.Pos(len(lit))}
	}
	body := lit[1 : len(lit)-1]
	// position of the first character of the body (just after the opening '"')
	bodyStart := startPos + 1

	parts, err := splitFString(body, bodyStart, p)
	if err != nil {
		p.error(startPos, err.Error())
		return &BadExpr{From: startPos, To: startPos + core.Pos(len(lit))}
	}

	return &FStringLit{
		Parts:    parts,
		ValuePos: startPos,
		// EndPos is one past the trailing '"'; the leading 'f' is *before* startPos, so the literal occupies
		// [startPos-1, startPos+len(lit)).
		EndPos:  startPos + core.Pos(len(lit)),
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
func splitFString(body string, bodyOffset core.Pos, p *Parser) ([]FStringPart, error) {
	var parts []FStringPart
	var litBuf bytes.Buffer
	flushLiteral := func() error {
		if litBuf.Len() == 0 {
			return nil
		}
		// Process backslash escapes via strconv.Unquote on a synthetic quoted string. This matches the semantics of
		// regular `"..."` literals.
		quoted := "\"" + litBuf.String() + "\""
		s, err := strconv.Unquote(quoted)
		if err != nil {
			return fmt.Errorf("invalid escape sequence in f-string: %v", err)
		}
		parts = append(parts, FStringPart{Literal: s})
		litBuf.Reset()
		return nil
	}

	i := 0
	n := len(body)
	for i < n {
		ch := body[i]
		switch ch {
		case '\\':
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
			// Apply f-string body escape rules to the expression text so that escapes such as `\"` (used to
			// embed a string literal containing the f-string's own delimiter) are converted back to their
			// literal form before the sub-parser sees them.
			if unescaped, uerr := strconv.Unquote("\"" + exprText + "\""); uerr == nil {
				exprText = unescaped
			}
			expr, perr := p.parseFStringExpr(exprText, bodyOffset+core.Pos(i+1))
			if perr != nil {
				return nil, perr
			}
			spec, ferr := fspec.Parse(specText)
			if ferr != nil {
				return nil, fmt.Errorf("f-string format spec %q: %v", specText, ferr)
			}
			_ = hasSpec
			parts = append(parts, FStringPart{
				Expr:     expr,
				Spec:     spec,
				SpecText: specText,
			})
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

// findFStringExprEnd returns the index of the '}' that closes the '{' whose content begins at start. It tracks balanced
// (), [], {} and skips quoted strings ("..." and '...') so a colon or '}' inside e.g. a record literal or a string does
// not prematurely terminate the expression.
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

// parseFStringExpr parses a Kavun expression embedded inside an f-string.
//
// origin is the absolute parser position where exprText begins in the containing source file; this is used so any
// sub-parser error refers to a position inside the original file rather than a synthetic file.
func (p *Parser) parseFStringExpr(exprText string, origin core.Pos) (Expr, error) {
	// Build a temporary SourceFile that shares the same FileSet so that positions reported by the sub-parser remain
	// meaningful in error messages produced by the host parser. We use a brand-new file because we want the
	// sub-parser's scanner to start at offset 0; that's fine because we report any errors with the host parser's
	// `error` helper using the f-string's overall position.
	subFile := p.file.set.AddFile("<fstring>", -1, len(exprText))
	src := []byte(exprText)
	// Add a trailing newline so the sub-parser's expression scan terminates cleanly at EOF.
	sub := NewParser(subFile, src, nil)
	if sub.token == token.EOF {
		return nil, fmt.Errorf("f-string: empty expression in '{}'")
	}
	expr := sub.parseExpr()
	if sub.token != token.EOF && !(sub.token == token.Semicolon && sub.tokenLit == "\n") {
		return nil, fmt.Errorf("f-string: unexpected token %q after expression %q", sub.tokenLit, exprText)
	}
	if sub.errors.Len() > 0 {
		return nil, fmt.Errorf("f-string expression %q: %v", exprText, sub.errors.Err())
	}
	_ = origin
	return expr, nil
}
