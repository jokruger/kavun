package compiler

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/parser/expression"
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
		c.emit(node, NewLoadStaticString(i))
		return nil
	}

	// Single literal-only part: emit a single string constant.
	if len(parts) == 1 && parts[0].Expr == nil {
		i := c.addStaticString(parts[0].Literal)
		c.emit(node, NewLoadStaticString(i))
		return nil
	}

	// General case: emit each part in order; for parts after the first emit an Add to concatenate onto the running
	// accumulator on the stack.
	for i, p := range parts {
		if err := c.emitFStringPart(node, p); err != nil {
			return err
		}
		if i > 0 {
			c.emit(node, NewBinaryOp(token.Add))
		}
	}
	return nil
}

func (c *Compiler) emitFStringPart(node *expression.FString, p expression.FStringPart) error {
	if p.Expr == nil {
		i := c.addStaticString(p.Literal)
		c.emit(node, NewLoadStaticString(i))
		return nil
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
		c.emit(node, NewLoadStaticString(i))
		var spec core.FormatSpec
		spec.Set(emptyFormatSpec, "")
		emptySpecIdx := c.addStaticFormatSpec(spec)
		for i, e := range p.SpecExprs {
			if err := c.CompileNode(e); err != nil {
				return err
			}
			// Stringify the inner expression with an empty format spec so any value type is converted to its default
			// textual representation (matches Python's `str(...)` behavior for nested spec interpolations).
			c.emit(node, NewFormatStaticSpec(emptySpecIdx))
			c.emit(node, NewBinaryOp(token.Add))
			if lit := p.SpecLiterals[i+1]; lit != "" {
				i := c.addStaticString(lit)
				c.emit(node, NewLoadStaticString(i))
				c.emit(node, NewBinaryOp(token.Add))
			}
		}
		c.emit(node, NewFormatRuntimeSpec())
		return nil
	}
	var spec core.FormatSpec
	spec.Set(p.Spec, p.SpecText)
	specIdx := c.addStaticFormatSpec(spec)
	c.emit(node, NewFormatStaticSpec(specIdx))
	return nil
}
