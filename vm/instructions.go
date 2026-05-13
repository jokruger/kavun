package vm

import (
	"fmt"
	"math"

	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/errs"
)

// MustMakeInstruction is like MakeInstruction but panics if the instruction cannot be created.
func MustMakeInstruction(opcode bc.Opcode, operands ...int) []byte {
	r, err := MakeInstruction(opcode, operands...)
	if err != nil {
		panic(fmt.Errorf("failed to make instruction: %w", err))
	}
	return r
}

// MakeInstruction returns a bytecode for an opcode and the operands.
func MakeInstruction(opcode bc.Opcode, operands ...int) ([]byte, error) {
	numOperands := bc.OpcodeOperands[opcode]

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
			if o < 0 || o > math.MaxUint8 {
				return nil, errs.NewInvalidOperandError(opcode, i, width, o)
			}
			instruction[offset] = byte(o)
		case 2:
			if o < 0 || o > math.MaxUint16 {
				return nil, errs.NewInvalidOperandError(opcode, i, width, o)
			}
			n := uint16(o)
			instruction[offset] = byte(n >> 8)
			instruction[offset+1] = byte(n)
		case 4:
			if o < 0 || o > math.MaxUint32 {
				return nil, errs.NewInvalidOperandError(opcode, i, width, o)
			}
			n := uint32(o)
			instruction[offset] = byte(n >> 24)
			instruction[offset+1] = byte(n >> 16)
			instruction[offset+2] = byte(n >> 8)
			instruction[offset+3] = byte(n)
		default:
			panic(fmt.Sprintf("unsupported operand width: %d, opcode %d, index %d", width, opcode, i))
		}
		offset += width
	}
	return instruction, nil
}

// FormatInstructions returns string representation of bytecode instructions.
func FormatInstructions(b []byte, posOffset int) []string {
	var out []string

	i := 0
	for i < len(b) {
		numOperands := bc.OpcodeOperands[b[i]]
		operands, read := bc.ReadOperands(numOperands, b[i+1:])

		switch len(numOperands) {
		case 0:
			out = append(out, fmt.Sprintf("%04d %s", posOffset+i, bc.OpcodeNames[b[i]]))
		case 1:
			out = append(out, fmt.Sprintf("%04d %-7s %d", posOffset+i, bc.OpcodeNames[b[i]], operands[0]))
		case 2:
			out = append(out, fmt.Sprintf("%04d %-7s %-5d %d", posOffset+i, bc.OpcodeNames[b[i]], operands[0], operands[1]))
		case 3:
			out = append(out, fmt.Sprintf("%04d %-7s %-5d %-5d %d", posOffset+i, bc.OpcodeNames[b[i]], operands[0], operands[1], operands[2]))
		default:
			panic(fmt.Sprintf("unsupported number of operands: %d", len(numOperands)))
		}
		i += 1 + read
	}
	return out
}
