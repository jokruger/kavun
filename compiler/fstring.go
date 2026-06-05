package compiler

import (
	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/token"
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
func (c *Compiler) compileFString(node *parser.FStringLit) error {
	parts := node.Parts

	// Zero parts: emit an empty string constant.
	if len(parts) == 0 {
		c.emit(node, bc.OpConstant, c.addConstant(c.alloc.NewStringValue("")))
		return nil
	}

	// Single literal-only part: emit a single string constant.
	if len(parts) == 1 && parts[0].Expr == nil {
		c.emit(node, bc.OpConstant, c.addConstant(c.alloc.NewStringValue(parts[0].Literal)))
		return nil
	}

	// General case: emit each part in order; for parts after the first emit an Add to concatenate onto the running
	// accumulator on the stack.
	for i, p := range parts {
		if err := c.emitFStringPart(node, p); err != nil {
			return err
		}
		if i > 0 {
			c.emit(node, bc.OpBinaryOp, int(token.Add))
		}
	}
	return nil
}

func (c *Compiler) emitFStringPart(node *parser.FStringLit, p parser.FStringPart) error {
	if p.Expr == nil {
		c.emit(node, bc.OpConstant, c.addConstant(c.alloc.NewStringValue(p.Literal)))
		return nil
	}
	if err := c.Compile(p.Expr); err != nil {
		return err
	}
	if len(p.SpecExprs) > 0 {
		// Dynamic spec: build the spec string at run time by interleaving SpecLiterals and SpecExprs.
		// Stack layout:  ..., value          (from p.Expr above)
		// We push the spec string on top and emit OpFormatDyn so the VM pops [spec, value] and pushes the formatted
		// result.
		c.emit(node, bc.OpConstant, c.addConstant(c.alloc.NewStringValue(p.SpecLiterals[0])))
		emptySpecIdx := c.addConstant(core.NewFormatSpecValue(emptyFormatSpec, ""))
		for i, e := range p.SpecExprs {
			if err := c.Compile(e); err != nil {
				return err
			}
			// Stringify the inner expression with an empty format spec so any value type is converted to its default
			// textual representation (matches Python's `str(...)` behaviour for nested spec interpolations).
			c.emit(node, bc.OpFormat, emptySpecIdx)
			c.emit(node, bc.OpBinaryOp, int(token.Add))
			if lit := p.SpecLiterals[i+1]; lit != "" {
				c.emit(node, bc.OpConstant, c.addConstant(c.alloc.NewStringValue(lit)))
				c.emit(node, bc.OpBinaryOp, int(token.Add))
			}
		}
		c.emit(node, bc.OpFormatDyn)
		return nil
	}
	specIdx := c.addConstant(core.NewFormatSpecValue(p.Spec, p.SpecText))
	c.emit(node, bc.OpFormat, specIdx)
	return nil
}
