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
func (c *Compiler) optimizeFunc(node parser.Node) {
	// any instructions between RETURN and the function end or instructions between RETURN and jump target position are
	// considered as unreachable.

	// pass 1. identify all jump destinations
	dsts := make(map[int]bool)
	iterateInstructions(c.scopes[c.scopeIndex].Instructions,
		func(pos int, opcode bc.Opcode, operands []int) bool {
			switch opcode {
			case bc.OpJump, bc.OpJumpFalsy,
				bc.OpAndJump, bc.OpOrJump:
				dsts[operands[0]] = true
			}
			return true
		})

	// pass 2. eliminate dead code
	var newInsts []byte
	posMap := make(map[int]int) // old position to new position
	var dstIdx int
	var deadCode bool
	iterateInstructions(c.scopes[c.scopeIndex].Instructions,
		func(pos int, opcode bc.Opcode, operands []int) bool {
			switch {
			case dsts[pos]:
				dstIdx++
				deadCode = false
			case opcode == bc.OpReturn:
				if deadCode {
					return true
				}
				deadCode = true
			case deadCode:
				return true
			}
			posMap[pos] = len(newInsts)
			newInsts = append(newInsts, vm.MakeInstruction(opcode, operands...)...)
			return true
		})

	// pass 3. update jump positions
	var lastOp bc.Opcode
	var appendReturn bool
	endPos := len(c.scopes[c.scopeIndex].Instructions)
	newEndPost := len(newInsts)

	iterateInstructions(newInsts,
		func(pos int, opcode bc.Opcode, operands []int) bool {
			switch opcode {
			case bc.OpJump, bc.OpJumpFalsy, bc.OpAndJump,
				bc.OpOrJump:
				newDst, ok := posMap[operands[0]]
				if ok {
					copy(newInsts[pos:], vm.MakeInstruction(opcode, newDst))
				} else if endPos == operands[0] {
					// there's a jump instruction that jumps to the end of
					// function compiler should append "return".
					copy(newInsts[pos:], vm.MakeInstruction(opcode, newEndPost))
					appendReturn = true
				} else {
					panic(fmt.Errorf("invalid jump position: %d", newDst))
				}
			}
			lastOp = opcode
			return true
		})
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
		c.emit(node, bc.OpReturn, 0)
	}
}

func iterateInstructions(b []byte, fn func(pos int, opcode bc.Opcode, operands []int) bool) {
	for i := 0; i < len(b); i++ {
		numOperands := bc.OpcodeOperands[b[i]]
		operands, read := bc.ReadOperands(numOperands, b[i+1:])
		if !fn(i, b[i], operands) {
			break
		}
		i += read
	}
}
