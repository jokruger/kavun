package bytecode

import "fmt"

// ReadOperands reads operands from the bytecode.
func ReadOperands(numOperands []int, ins []byte) ([]int, int, error) {
	operands := make([]int, 0, len(numOperands))
	var offset int
	for _, width := range numOperands {
		switch width {
		case 1:
			operands = append(operands, int(ins[offset]))
		case 2:
			operands = append(operands, int(ins[offset+1])|int(ins[offset])<<8)
		default:
			return nil, 0, fmt.Errorf("unsupported operand width: %d", width)
		}
		offset += width
	}
	return operands, offset, nil
}
