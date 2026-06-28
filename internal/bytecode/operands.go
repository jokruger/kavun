package bytecode

import (
	"encoding/binary"
	"fmt"
)

// ReadOperands reads operands from the bytecode.
func ReadOperands(numOperands []int, ins []byte) ([]int, int, error) {
	operands := make([]int, 0, len(numOperands))
	var offset int
	for _, width := range numOperands {
		switch width {
		case 1:
			operands = append(operands, int(ins[offset]))
		case 2:
			operands = append(operands, int(binary.LittleEndian.Uint16(ins[offset:])))
		case 4:
			operands = append(operands, int(binary.LittleEndian.Uint32(ins[offset:])))
		case 8:
			operands = append(operands, int(binary.LittleEndian.Uint64(ins[offset:])))
		default:
			return nil, 0, fmt.Errorf("unsupported operand width: %d", width)
		}
		offset += width
	}
	return operands, offset, nil
}
