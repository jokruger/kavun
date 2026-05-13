package compiler

import (
	"fmt"

	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/vm"
)

// optimizeFunc performs some code-level optimization for the current function instructions. It also removes unreachable
// (dead code) instructions and adds "returns" instruction if needed.
func (c *Compiler) optimizeFunc(node parser.Node) (err error) {
	// any instructions between RETURN and the function end or instructions between RETURN and jump target position are
	// considered as unreachable.

	// pass 1. identify all jump destinations
	dsts := make(map[int]bool)
	err = iterateInstructions(c.scopes[c.scopeIndex].Instructions, func(pos int, opcode bc.Opcode, operands []int) (bool, error) {
		switch opcode {
		case bc.OpJump, bc.OpJumpFalsy, bc.OpAndJump, bc.OpOrJump:
			dsts[operands[0]] = true
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	// pass 2. eliminate dead code
	var newInsts []byte
	posMap := make(map[int]int) // old position to new position
	var dstIdx int
	var deadCode bool
	err = iterateInstructions(c.scopes[c.scopeIndex].Instructions, func(pos int, opcode bc.Opcode, operands []int) (bool, error) {
		switch {
		case dsts[pos]:
			dstIdx++
			deadCode = false
		case opcode == bc.OpReturn:
			if deadCode {
				return true, nil
			}
			deadCode = true
		case deadCode:
			return true, nil
		}
		posMap[pos] = len(newInsts)
		t, err := vm.MakeInstruction(opcode, operands...)
		if err != nil {
			return false, err
		}
		newInsts = append(newInsts, t...)
		return true, nil
	})
	if err != nil {
		return err
	}

	// pass 3. update jump positions
	var lastOp bc.Opcode
	var appendReturn bool
	endPos := len(c.scopes[c.scopeIndex].Instructions)
	newEndPost := len(newInsts)

	err = iterateInstructions(newInsts, func(pos int, opcode bc.Opcode, operands []int) (bool, error) {
		switch opcode {
		case bc.OpJump, bc.OpJumpFalsy, bc.OpAndJump, bc.OpOrJump:
			newDst, ok := posMap[operands[0]]
			if ok {
				t, err := vm.MakeInstruction(opcode, newDst)
				if err != nil {
					return false, err
				}
				copy(newInsts[pos:], t)
			} else if endPos == operands[0] {
				// there's a jump instruction that jumps to the end of function compiler should append "return".
				t, err := vm.MakeInstruction(opcode, newEndPost)
				if err != nil {
					return false, err
				}
				copy(newInsts[pos:], t)
				appendReturn = true
			} else {
				return false, fmt.Errorf("invalid jump position: %d", newDst)
			}
		}
		lastOp = opcode
		return true, nil
	})
	if err != nil {
		return err
	}
	if lastOp != bc.OpReturn {
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
		_, err = c.emit(node, bc.OpReturn, 0)
		if err != nil {
			return err
		}
	}

	return nil
}

func iterateInstructions(b []byte, fn func(pos int, opcode bc.Opcode, operands []int) (bool, error)) error {
	for i := 0; i < len(b); i++ {
		numOperands := bc.OpcodeOperands[b[i]]
		operands, read := bc.ReadOperands(numOperands, b[i+1:])
		r, err := fn(i, b[i], operands)
		if err != nil {
			return err
		}
		if !r {
			break
		}
		i += read
	}
	return nil
}
