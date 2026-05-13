package unit

import (
	"strings"
	"testing"
	"time"

	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/tests/require"
	"github.com/jokruger/kavun/vm"
)

func TestInstructions_String(t *testing.T) {
	assertInstructionString(t,
		[][]byte{
			vm.MustMakeInstruction(bc.OpConstant, 1),
			vm.MustMakeInstruction(bc.OpConstant, 2),
			vm.MustMakeInstruction(bc.OpConstant, 65535),
		},
		`0000 CONST   1
0003 CONST   2
0006 CONST   65535`)

	assertInstructionString(t,
		[][]byte{
			vm.MustMakeInstruction(bc.OpBinaryOp, 11),
			vm.MustMakeInstruction(bc.OpConstant, 2),
			vm.MustMakeInstruction(bc.OpConstant, 65535),
		},
		`0000 BINARYOP 11
0002 CONST   2
0005 CONST   65535`)

	assertInstructionString(t,
		[][]byte{
			vm.MustMakeInstruction(bc.OpBinaryOp, 11),
			vm.MustMakeInstruction(bc.OpGetLocal, 1),
			vm.MustMakeInstruction(bc.OpConstant, 2),
			vm.MustMakeInstruction(bc.OpConstant, 65535),
		},
		`0000 BINARYOP 11
0002 GETL    1
0004 CONST   2
0007 CONST   65535`)
}

func TestMakeInstruction(t *testing.T) {
	makeInstruction(t, []byte{bc.OpConstant, 0, 0},
		bc.OpConstant, 0)
	makeInstruction(t, []byte{bc.OpConstant, 0, 1},
		bc.OpConstant, 1)
	makeInstruction(t, []byte{bc.OpConstant, 255, 254},
		bc.OpConstant, 65534)
	makeInstruction(t, []byte{bc.OpPop}, bc.OpPop)
	makeInstruction(t, []byte{bc.OpTrue}, bc.OpTrue)
	makeInstruction(t, []byte{bc.OpFalse}, bc.OpFalse)
}

func TestNumObjects(t *testing.T) {
	testCountObjects(t, core.NewArrayValue(nil, false), 1)
	testCountObjects(t, core.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.NewArrayValue([]core.Value{core.IntValue(3), core.IntValue(4), core.IntValue(5)}, false),
	}, false), 7)
	testCountObjects(t, core.True, 1)
	testCountObjects(t, core.False, 1)
	testCountObjects(t, core.NewBuiltinFunctionValue("", nil, 0, false), 1)
	testCountObjects(t, core.NewBytesValue([]byte("foobar"), false), 1)
	testCountObjects(t, core.RuneValue('가'), 1)
	testCountObjects(t, core.CompiledFunctionValue(&core.CompiledFunction{}), 1)
	testCountObjects(t, core.NewErrorValue(core.IntValue(5)), 2)
	testCountObjects(t, core.FloatValue(19.84), 1)
	testCountObjects(t, core.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.NewArrayValue([]core.Value{core.IntValue(3), core.IntValue(4), core.IntValue(5)}, true),
	}, true), 7)
	testCountObjects(t, core.NewRecordValue(map[string]core.Value{
		"k1": core.IntValue(1),
		"k2": core.IntValue(2),
		"k3": core.NewArrayValue([]core.Value{core.IntValue(3), core.IntValue(4), core.IntValue(5)}, false),
	}, true), 7)
	testCountObjects(t, core.IntValue(1984), 1)
	testCountObjects(t, core.NewRecordValue(map[string]core.Value{
		"k1": core.IntValue(1),
		"k2": core.IntValue(2),
		"k3": core.NewArrayValue([]core.Value{core.IntValue(3), core.IntValue(4), core.IntValue(5)}, false),
	}, false), 7)
	testCountObjects(t, core.NewStringValue("foo bar"), 1)
	testCountObjects(t, core.NewTimeValue(time.Now()), 1)
	testCountObjects(t, core.Undefined, 1)
}

func testCountObjects(t *testing.T, o core.Value, expected int) {
	require.Equal(t, expected, vm.CountObjects(o))
}

func assertInstructionString(t *testing.T, instructions [][]byte, expected string) {
	concatted := make([]byte, 0)
	for _, e := range instructions {
		concatted = append(concatted, e...)
	}
	require.Equal(t, expected, strings.Join(vm.MustFormatInstructions(concatted, 0), "\n"))
}

func makeInstruction(t *testing.T, expected []byte, opcode bc.Opcode, operands ...int) {
	inst := vm.MustMakeInstruction(opcode, operands...)
	require.Equal(t, expected, inst)
}
