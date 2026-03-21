package gs_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/tests/require"
	"github.com/jokruger/gs/value"
	"github.com/jokruger/gs/vm"
)

func TestInstructions_String(t *testing.T) {
	assertInstructionString(t,
		[][]byte{
			vm.MakeInstruction(parser.OpConstant, 1),
			vm.MakeInstruction(parser.OpConstant, 2),
			vm.MakeInstruction(parser.OpConstant, 65535),
		},
		`0000 CONST   1    
0003 CONST   2    
0006 CONST   65535`)

	assertInstructionString(t,
		[][]byte{
			vm.MakeInstruction(parser.OpBinaryOp, 11),
			vm.MakeInstruction(parser.OpConstant, 2),
			vm.MakeInstruction(parser.OpConstant, 65535),
		},
		`0000 BINARYOP 11   
0002 CONST   2    
0005 CONST   65535`)

	assertInstructionString(t,
		[][]byte{
			vm.MakeInstruction(parser.OpBinaryOp, 11),
			vm.MakeInstruction(parser.OpGetLocal, 1),
			vm.MakeInstruction(parser.OpConstant, 2),
			vm.MakeInstruction(parser.OpConstant, 65535),
		},
		`0000 BINARYOP 11   
0002 GETL    1    
0004 CONST   2    
0007 CONST   65535`)
}

func TestMakeInstruction(t *testing.T) {
	makeInstruction(t, []byte{parser.OpConstant, 0, 0},
		parser.OpConstant, 0)
	makeInstruction(t, []byte{parser.OpConstant, 0, 1},
		parser.OpConstant, 1)
	makeInstruction(t, []byte{parser.OpConstant, 255, 254},
		parser.OpConstant, 65534)
	makeInstruction(t, []byte{parser.OpPop}, parser.OpPop)
	makeInstruction(t, []byte{parser.OpTrue}, parser.OpTrue)
	makeInstruction(t, []byte{parser.OpFalse}, parser.OpFalse)
}

func TestNumObjects(t *testing.T) {
	testCountObjects(t, value.NewArray(nil, false), 1)
	testCountObjects(t, value.NewArray([]core.Object{
		value.NewInt(1),
		value.NewInt(2),
		value.NewArray([]core.Object{value.NewInt(3), value.NewInt(4), value.NewInt(5)}, false),
	}, false), 7)
	testCountObjects(t, value.TrueValue, 1)
	testCountObjects(t, value.FalseValue, 1)
	testCountObjects(t, value.NewBuiltinFunction("", nil, 0, false), 1)
	testCountObjects(t, value.NewBytes([]byte("foobar")), 1)
	testCountObjects(t, value.NewChar('가'), 1)
	testCountObjects(t, &value.CompiledFunction{}, 1)
	testCountObjects(t, value.NewError(value.NewInt(5)), 2)
	testCountObjects(t, value.NewFloat(19.84), 1)
	testCountObjects(t, value.NewArray([]core.Object{
		value.NewInt(1),
		value.NewInt(2),
		value.NewArray([]core.Object{value.NewInt(3), value.NewInt(4), value.NewInt(5)}, true),
	}, true), 7)
	testCountObjects(t, value.NewRecord(map[string]core.Object{
		"k1": value.NewInt(1),
		"k2": value.NewInt(2),
		"k3": value.NewArray([]core.Object{value.NewInt(3), value.NewInt(4), value.NewInt(5)}, false),
	}, true), 7)
	testCountObjects(t, value.NewInt(1984), 1)
	testCountObjects(t, value.NewRecord(map[string]core.Object{
		"k1": value.NewInt(1),
		"k2": value.NewInt(2),
		"k3": value.NewArray([]core.Object{value.NewInt(3), value.NewInt(4), value.NewInt(5)}, false),
	}, false), 7)
	testCountObjects(t, value.NewString("foo bar"), 1)
	testCountObjects(t, value.NewTime(time.Now()), 1)
	testCountObjects(t, value.UndefinedValue, 1)
}

func testCountObjects(t *testing.T, o core.Object, expected int) {
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
