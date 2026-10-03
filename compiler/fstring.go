package compiler

import (
	"github.com/jokruger/kavun/ast/expression"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/fspec"
)

// emptyFormatSpec is the zero FormatSpec used to coerce dynamic-spec sub-expressions to their default string form.
var emptyFormatSpec = fspec.FormatSpec{}

// compileFString lowers an f-string literal into a sequence of CONST / expr-eval+FMT / ADD operations that build the
// final string at run time.
//
// The number of parts is unbounded and they may be mixed in any order:
//
//	f""            -> CONST ""
//	f"hello"       -> CONST "hello"
//	f"{x}"         -> compile(x) ; FMT <empty-spec>
//	f"a={x} b={y}" -> CONST "a=" ; compile(x) ; FMT spec1 ; ADD ;
//	                  CONST " b=" ; ADD ; compile(y) ; FMT spec2 ; ADD
//
// Each interpolation always lowers to FMT — including when the spec text is the empty string — because the per-type
// Format function decides what an empty FormatSpec means for that type.
func (c *Compiler) compileFString(node *expression.FString) error {
	parts := node.Parts

	// Zero parts: emit an empty string constant.
	if len(parts) == 0 {
		i := c.addStaticString("")
		_, err := c.emit(node, NewLoadStaticString(i))
		return err
	}

	// Single literal-only part: emit a single string constant.
	if len(parts) == 1 && parts[0].Expr == nil {
		i := c.addStaticString(parts[0].Literal)
		_, err := c.emit(node, NewLoadStaticString(i))
		return err
	}

	// General case: emit each part in order; for parts after the first emit an Add to concatenate onto the running
	// accumulator on the stack.
	for i, p := range parts {
		if err := c.emitFStringPart(node, p); err != nil {
			return err
		}
		if i > 0 {
			if _, err := c.emit(node, NewBinaryOp(token.Add)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Compiler) emitFStringPart(node *expression.FString, p expression.FStringPart) error {
	if p.Expr == nil {
		i := c.addStaticString(p.Literal)
		_, err := c.emit(node, NewLoadStaticString(i))
		return err
	}
	if err := c.CompileNode(p.Expr); err != nil {
		return err
	}
	if len(p.SpecExprs) > 0 {
		// Dynamic spec: build the spec string at run time by interleaving SpecLiterals and SpecExprs.
		// Stack layout:  ..., value          (from p.Expr above)
		// We push the spec string on top and emit OpFormatDyn so the VM pops [spec, value] and pushes the formatted
		// result.
		i := c.addStaticString(p.SpecLiterals[0])
		if _, err := c.emit(node, NewLoadStaticString(i)); err != nil {
			return err
		}
		var spec core.FormatSpec
		spec.Set(emptyFormatSpec, "")
		emptySpecIdx := c.addStaticFormatSpec(spec)
		for i, e := range p.SpecExprs {
			if err := c.CompileNode(e); err != nil {
				return err
			}
			// Stringify the inner expression with an empty format spec so any value type is converted to its default
			// textual representation (matches Python's `str(...)` behavior for nested spec interpolations).
			if _, err := c.emit(node, NewFormatStaticSpec(emptySpecIdx)); err != nil {
				return err
			}
			if _, err := c.emit(node, NewBinaryOp(token.Add)); err != nil {
				return err
			}
			if lit := p.SpecLiterals[i+1]; lit != "" {
				i := c.addStaticString(lit)
				if _, err := c.emit(node, NewLoadStaticString(i)); err != nil {
					return err
				}
				if _, err := c.emit(node, NewBinaryOp(token.Add)); err != nil {
					return err
				}
			}
		}
		_, err := c.emit(node, NewFormatRuntimeSpec())
		return err
	}
	var spec core.FormatSpec
	spec.Set(p.Spec, p.SpecText)
	specIdx := c.addStaticFormatSpec(spec)
	_, err := c.emit(node, NewFormatStaticSpec(specIdx))
	return err
}
