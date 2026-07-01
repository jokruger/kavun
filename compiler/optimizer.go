package compiler

import (
	"fmt"

	"github.com/jokruger/kavun/core"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/parser"
)

// optimizeFunc performs some code-level optimization for the current function instructions. It also removes unreachable
// (dead code) instructions and adds "returns" instruction if needed.
func (c *Compiler) optimizeFunc(node parser.Node) (err error) {
	// any instructions between RETURN and the function end or instructions between RETURN and jump target position are
	// considered as unreachable.

	// pass 1. identify all jump destinations
	dsts := make(map[int]bool)
	err = iterateInstructions(c.scopes[c.scopeIndex].Instructions, func(pos int, ci bc.Instruction) (bool, error) {
		switch ci.Op {
		case bc.Jump, bc.JumpFalsy, bc.AndJump, bc.OrJump:
			dsts[int(ci.Op3)] = true
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	// pass 2. eliminate dead code
	var newInsts bc.Instructions
	posMap := make(map[int]int) // old position to new position
	var dstIdx int
	var deadCode bool
	err = iterateInstructions(c.scopes[c.scopeIndex].Instructions, func(pos int, ci bc.Instruction) (bool, error) {
		switch {
		case dsts[pos]:
			dstIdx++
			deadCode = false
		case ci.Op == bc.Return:
			if deadCode {
				return true, nil
			}
			deadCode = true
		case deadCode:
			return true, nil
		}
		posMap[pos] = len(newInsts)
		newInsts = append(newInsts, ci)
		return true, nil
	})
	if err != nil {
		return err
	}

	// pass 3. update jump positions
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

	// pass 4. update source map
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
