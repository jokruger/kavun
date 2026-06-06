package unit

import (
	"strings"
	"testing"
	"time"

	"github.com/jokruger/kavun/core"
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

func TestNumObjects(t *testing.T) {
	testCountObjects(t, rta.NewArrayValue(nil, false), 1)
	testCountObjects(t, rta.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		rta.NewArrayValue([]core.Value{core.IntValue(3), core.IntValue(4), core.IntValue(5)}, false),
	}, false), 7)
	testCountObjects(t, core.True, 1)
	testCountObjects(t, core.False, 1)
	testCountObjects(t, rta.NewBuiltinClosureValue("", nil, 0, false), 1)
	testCountObjects(t, rta.NewBytesValue([]byte("foobar"), false), 1)
	testCountObjects(t, core.RuneValue('가'), 1)
	testCountObjects(t, rta.NewCompiledFunctionValue(nil, nil, nil, 0, 0, 0, false, 0), 1)
	testCountObjects(t, rta.NewErrorValue(core.IntValue(5), core.KindUser, false), 2)
	testCountObjects(t, core.FloatValue(19.84), 1)
	testCountObjects(t, rta.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		rta.NewArrayValue([]core.Value{core.IntValue(3), core.IntValue(4), core.IntValue(5)}, true),
	}, true), 7)
	testCountObjects(t, rta.NewRecordValue(map[string]core.Value{
		"k1": core.IntValue(1),
		"k2": core.IntValue(2),
		"k3": rta.NewArrayValue([]core.Value{core.IntValue(3), core.IntValue(4), core.IntValue(5)}, false),
	}, true), 7)
	testCountObjects(t, core.IntValue(1984), 1)
	testCountObjects(t, rta.NewRecordValue(map[string]core.Value{
		"k1": core.IntValue(1),
		"k2": core.IntValue(2),
		"k3": rta.NewArrayValue([]core.Value{core.IntValue(3), core.IntValue(4), core.IntValue(5)}, false),
	}, false), 7)
	testCountObjects(t, rta.NewStringValue("foo bar"), 1)
	testCountObjects(t, rta.NewTimeValue(time.Now()), 1)
	testCountObjects(t, core.Undefined, 1)
}

func testCountObjects(t *testing.T, o core.Value, expected int) {
	require.Equal(t, rta, expected, vm.CountObjects(o))
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
