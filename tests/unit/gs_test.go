package unit

import (
	"strings"
	"testing"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/tests/require"
	"github.com/jokruger/gs/vm"
)

func TestInstructions_String(t *testing.T) {
	assertInstructionString(t,
		[][]byte{
			vm.MakeInstruction(core.OpConstant, 1),
			vm.MakeInstruction(core.OpConstant, 2),
			vm.MakeInstruction(core.OpConstant, 65535),
		},
		`0000 CONST   1    
0003 CONST   2    
0006 CONST   65535`)

	assertInstructionString(t,
		[][]byte{
			vm.MakeInstruction(core.OpBinaryOp, 11),
			vm.MakeInstruction(core.OpConstant, 2),
			vm.MakeInstruction(core.OpConstant, 65535),
		},
		`0000 BINARYOP 11   
0002 CONST   2    
0005 CONST   65535`)

	assertInstructionString(t,
		[][]byte{
			vm.MakeInstruction(core.OpBinaryOp, 11),
			vm.MakeInstruction(core.OpGetLocal, 1),
			vm.MakeInstruction(core.OpConstant, 2),
			vm.MakeInstruction(core.OpConstant, 65535),
		},
		`0000 BINARYOP 11   
0002 GETL    1    
0004 CONST   2    
0007 CONST   65535`)
}

func TestMakeInstruction(t *testing.T) {
	makeInstruction(t, []byte{core.OpConstant, 0, 0},
		core.OpConstant, 0)
	makeInstruction(t, []byte{core.OpConstant, 0, 1},
		core.OpConstant, 1)
	makeInstruction(t, []byte{core.OpConstant, 255, 254},
		core.OpConstant, 65534)
	makeInstruction(t, []byte{core.OpPop}, core.OpPop)
	makeInstruction(t, []byte{core.OpTrue}, core.OpTrue)
	makeInstruction(t, []byte{core.OpFalse}, core.OpFalse)
}

func TestNumObjects(t *testing.T) {
	testCountObjects(t, alloc.NewArrayValue(nil, false), 1)
	testCountObjects(t, alloc.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		alloc.NewArrayValue([]core.Value{core.IntValue(3), core.IntValue(4), core.IntValue(5)}, false),
	}, false), 7)
	testCountObjects(t, core.True, 1)
	testCountObjects(t, core.False, 1)
	testCountObjects(t, alloc.NewBuiltinFunctionValue("", nil, 0, false), 1)
	testCountObjects(t, alloc.NewBytesValue([]byte("foobar")), 1)
	testCountObjects(t, core.CharValue('가'), 1)
	testCountObjects(t, core.CompiledFunctionValue(&core.CompiledFunction{}), 1)
	testCountObjects(t, alloc.NewErrorValue(core.IntValue(5)), 2)
	testCountObjects(t, core.FloatValue(19.84), 1)
	testCountObjects(t, alloc.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		alloc.NewArrayValue([]core.Value{core.IntValue(3), core.IntValue(4), core.IntValue(5)}, true),
	}, true), 7)
	testCountObjects(t, alloc.NewRecordValue(map[string]core.Value{
		"k1": core.IntValue(1),
		"k2": core.IntValue(2),
		"k3": alloc.NewArrayValue([]core.Value{core.IntValue(3), core.IntValue(4), core.IntValue(5)}, false),
	}, true), 7)
	testCountObjects(t, core.IntValue(1984), 1)
	testCountObjects(t, alloc.NewRecordValue(map[string]core.Value{
		"k1": core.IntValue(1),
		"k2": core.IntValue(2),
		"k3": alloc.NewArrayValue([]core.Value{core.IntValue(3), core.IntValue(4), core.IntValue(5)}, false),
	}, false), 7)
	testCountObjects(t, alloc.NewStringValue("foo bar"), 1)
	testCountObjects(t, alloc.NewTimeValue(time.Now()), 1)
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
	require.Equal(t, expected, strings.Join(vm.FormatInstructions(concatted, 0), "\n"))
}

func makeInstruction(t *testing.T, expected []byte, opcode core.Opcode, operands ...int) {
	inst := vm.MakeInstruction(opcode, operands...)
	require.Equal(t, expected, inst)
}
