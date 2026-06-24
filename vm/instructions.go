package vm

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/jokruger/kavun/core/opcode"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/bytecode"
)

// MustMakeInstruction is like MakeInstruction but panics if the instruction cannot be created.
func MustMakeInstruction(opcode opcode.Opcode, operands ...int) []byte {
	r, err := MakeInstruction(opcode, operands...)
	if err != nil {
		panic(fmt.Errorf("failed to make instruction: %w", err))
	}
	return r
}

// MakeInstruction returns a bytecode for an opcode and the operands.
func MakeInstruction(opcode opcode.Opcode, operands ...int) ([]byte, error) {
	numOperands := opcode.Operands()

	totalLen := 1
	for _, w := range numOperands {
		totalLen += w
	}

	instruction := make([]byte, totalLen)
	instruction[0] = opcode.Byte()

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
			binary.LittleEndian.PutUint16(instruction[offset:], uint16(o))
		case 4:
			if o < 0 || o > math.MaxUint32 {
				return nil, errs.NewInvalidOperandError(opcode, i, width, o)
			}
			binary.LittleEndian.PutUint32(instruction[offset:], uint32(o))
		case 8:
			binary.LittleEndian.PutUint64(instruction[offset:], uint64(o))
		default:
			panic(fmt.Sprintf("unsupported operand width: %d, opcode %d, index %d", width, opcode, i))
		}
		offset += width
	}
	return instruction, nil
}

// MustFormatInstructions is like FormatInstructions but panics if the instructions cannot be formatted.
func MustFormatInstructions(b []byte, posOffset int) []string {
	r, err := FormatInstructions(b, posOffset)
	if err != nil {
		panic(fmt.Errorf("failed to format instructions: %w", err))
	}
	return r
}

// FormatInstructions returns string representation of bytecode instructions.
func FormatInstructions(b []byte, posOffset int) ([]string, error) {
	var out []string

	i := 0
	for i < len(b) {
		numOperands := opcode.Opcode(b[i]).Operands()
		operands, read, err := bytecode.ReadOperands(numOperands, b[i+1:])
		if err != nil {
			return nil, err
		}

		switch len(numOperands) {
		case 0:
			out = append(out, fmt.Sprintf("%04d %s", posOffset+i, opcode.Opcode(b[i]).String()))
		case 1:
			out = append(out, fmt.Sprintf("%04d %-7s %d", posOffset+i, opcode.Opcode(b[i]).String(), operands[0]))
		case 2:
			out = append(out, fmt.Sprintf("%04d %-7s %-5d %d", posOffset+i, opcode.Opcode(b[i]).String(), operands[0], operands[1]))
		case 3:
			out = append(out, fmt.Sprintf("%04d %-7s %-5d %-5d %d", posOffset+i, opcode.Opcode(b[i]).String(), operands[0], operands[1], operands[2]))
		default:
			panic(fmt.Sprintf("unsupported number of operands: %d", len(numOperands)))
		}
		i += 1 + read
	}
	return out, nil
}
