package vm

import (
	"fmt"

	bc "github.com/jokruger/kavun/core/bytecode"
)

// MustFormatInstructions is like FormatInstructions but panics if the instructions cannot be formatted.
func MustFormatInstructions(is bc.Instructions, posOffset int) []string {
	r, err := FormatInstructions(is, posOffset)
	if err != nil {
		panic(fmt.Errorf("failed to format instructions: %w", err))
	}
	return r
}

// FormatInstructions returns string representation of bytecode instructions.
func FormatInstructions(is bc.Instructions, posOffset int) ([]string, error) {
	var out []string
	for offs, i := range is {
		out = append(out, fmt.Sprintf("%04d %s", posOffset+offs, i.String()))
	}
	return out, nil
}
