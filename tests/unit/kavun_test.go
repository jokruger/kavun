package unit

import (
	"strings"
	"testing"

	"github.com/jokruger/kavun/opcode"
	"github.com/jokruger/kavun/tests/require"
	"github.com/jokruger/kavun/vm"
)

func TestInstructions_String(t *testing.T) {
	assertInstructionString(t,
		[][]byte{
			vm.MustMakeInstruction(opcode.Constant, 1),
			vm.MustMakeInstruction(opcode.Constant, 2),
			vm.MustMakeInstruction(opcode.Constant, 65535),
		},
		`0000 CONST   1
0003 CONST   2
0006 CONST   65535`)

	assertInstructionString(t,
		[][]byte{
			vm.MustMakeInstruction(opcode.BinaryOp, 11),
			vm.MustMakeInstruction(opcode.Constant, 2),
			vm.MustMakeInstruction(opcode.Constant, 65535),
		},
		`0000 BINARYOP 11
0002 CONST   2
0005 CONST   65535`)

	assertInstructionString(t,
		[][]byte{
			vm.MustMakeInstruction(opcode.BinaryOp, 11),
			vm.MustMakeInstruction(opcode.GetLocal, 1),
			vm.MustMakeInstruction(opcode.Constant, 2),
			vm.MustMakeInstruction(opcode.Constant, 65535),
		},
		`0000 BINARYOP 11
0002 GETL    1
0004 CONST   2
0007 CONST   65535`)
}

func TestMakeInstruction(t *testing.T) {
	makeInstruction(t, []byte{opcode.Constant.Byte(), 0, 0},
		opcode.Constant, 0)
	makeInstruction(t, []byte{opcode.Constant.Byte(), 0, 1},
		opcode.Constant, 1)
	makeInstruction(t, []byte{opcode.Constant.Byte(), 255, 254},
		opcode.Constant, 65534)
	makeInstruction(t, []byte{opcode.Pop.Byte()}, opcode.Pop)
	makeInstruction(t, []byte{opcode.True.Byte()}, opcode.True)
	makeInstruction(t, []byte{opcode.False.Byte()}, opcode.False)
}

func assertInstructionString(t *testing.T, instructions [][]byte, expected string) {
	concatted := make([]byte, 0)
	for _, e := range instructions {
		concatted = append(concatted, e...)
	}
	require.Equal(t, rta, expected, strings.Join(vm.MustFormatInstructions(concatted, 0), "\n"))
}

func makeInstruction(t *testing.T, expected []byte, opcode opcode.Opcode, operands ...int) {
	inst := vm.MustMakeInstruction(opcode, operands...)
	require.Equal(t, rta, expected, inst)
}
