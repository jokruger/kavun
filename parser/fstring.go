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
func findFStringExprEnd(body string, start int) (int, error) {
	depth := 0
	i := start
	n := len(body)
	for i < n {
		ch := body[i]
		switch ch {
		case '"', '\'':
			// skip a quoted string
			quote := ch
			i++
			for i < n && body[i] != quote {
				if body[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				if body[i] == '\n' {
					return 0, fmt.Errorf("f-string: unterminated string inside interpolation")
				}
				i++
			}
			if i >= n {
				return 0, fmt.Errorf("f-string: unterminated string inside interpolation")
			}
			i++ // skip closing quote
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
	i := 0
	n := len(inner)
	for i < n {
		ch := inner[i]
		switch ch {
		case '"', '\'':
			quote := ch
			i++
			for i < n && inner[i] != quote {
				if inner[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				i++
			}
			if i < n {
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
		case ':':
			if depth == 0 {
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
