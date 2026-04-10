package vm

import (
	"fmt"

	"github.com/jokruger/gs/core"
)

// MakeInstruction returns a bytecode for an opcode and the operands.
func MakeInstruction(opcode core.Opcode, operands ...int) []byte {
	numOperands := core.OpcodeOperands[opcode]

	totalLen := 1
	for _, w := range numOperands {
		totalLen += w
	}

	instruction := make([]byte, totalLen)
	instruction[0] = opcode

	offset := 1
	for i, o := range operands {
		width := numOperands[i]
		switch width {
		case 1:
			instruction[offset] = byte(o)
		case 2:
			n := uint16(o)
			instruction[offset] = byte(n >> 8)
			instruction[offset+1] = byte(n)
		case 4:
			n := uint32(o)
			instruction[offset] = byte(n >> 24)
			instruction[offset+1] = byte(n >> 16)
			instruction[offset+2] = byte(n >> 8)
			instruction[offset+3] = byte(n)
		}
		offset += width
	}
	return instruction
}

// FormatInstructions returns string representation of bytecode instructions.
func FormatInstructions(b []byte, posOffset int) []string {
	var out []string

	i := 0
	for i < len(b) {
		numOperands := core.OpcodeOperands[b[i]]
		operands, read := core.ReadOperands(numOperands, b[i+1:])

		switch len(numOperands) {
		case 0:
			out = append(out, fmt.Sprintf("%04d %-7s", posOffset+i, core.OpcodeNames[b[i]]))
		case 1:
			out = append(out, fmt.Sprintf("%04d %-7s %-5d", posOffset+i, core.OpcodeNames[b[i]], operands[0]))
		case 2:
			out = append(out, fmt.Sprintf("%04d %-7s %-5d %-5d", posOffset+i, core.OpcodeNames[b[i]], operands[0], operands[1]))
		case 3:
			out = append(out, fmt.Sprintf("%04d %-7s %-5d %-5d %-5d", posOffset+i, core.OpcodeNames[b[i]], operands[0], operands[1], operands[2]))
		}
		i += 1 + read
	}
	return out
}
