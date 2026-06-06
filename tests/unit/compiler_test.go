package unit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/opcode"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/tests/require"
	"github.com/jokruger/kavun/vm"
)

func TestCompiler_Compile(t *testing.T) {
	expectCompile(t, `1 + 2`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 11),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1; 2`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 - 2`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 12),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 * 2`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 13),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `2 / 1`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 14),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(2),
				intObject(1))))

	expectCompile(t, `1 in 2`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Contains),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 not in 2`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Contains),
				vm.MustMakeInstruction(opcode.LNot),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `true`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.True),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray()))

	expectCompile(t, `false`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.False),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray()))

	expectCompile(t, `1 > 2`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 39),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 < 2`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 38),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 >= 2`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 44),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 <= 2`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 43),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 == 2`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Equal),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 != 2`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.NotEqual),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `true == false`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.True),
				vm.MustMakeInstruction(opcode.False),
				vm.MustMakeInstruction(opcode.Equal),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray()))

	expectCompile(t, `true != false`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.True),
				vm.MustMakeInstruction(opcode.False),
				vm.MustMakeInstruction(opcode.NotEqual),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray()))

	expectCompile(t, `-1`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Minus),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1))))

	expectCompile(t, `!true`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.True),
				vm.MustMakeInstruction(opcode.LNot),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray()))

	expectCompile(t, `if true { 10 }; 3333`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.True),
				vm.MustMakeInstruction(opcode.JumpFalsy, 8),
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(10),
				intObject(3333))))

	expectCompile(t, `if (true) { 10 } else { 20 }; 3333;`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.True),
				vm.MustMakeInstruction(opcode.JumpFalsy, 11),
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Jump, 15),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(10),
				intObject(20),
				intObject(3333))))

	expectCompile(t, `"kami"`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				stringObject("kami"))))

	expectCompile(t, `"ka" + "mi"`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 11),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				stringObject("ka"),
				stringObject("mi"))))

	expectCompile(t, `var a`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Null),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray()))

	expectCompile(t, `var a = 1`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1))))

	expectCompile(t, `a := 1; b := 2; a += b`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.SetGlobal, 1),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.GetGlobal, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 11),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `a := 1; b := 2; a /= b`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.SetGlobal, 1),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.GetGlobal, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 14),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `[]`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Array, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray()))

	expectCompile(t, `[1, 2, 3]`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Array, 3),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3))))

	expectCompile(t, `[1 + 2, 3 - 4, 5 * 6]`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 11),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Constant, 3),
				vm.MustMakeInstruction(opcode.BinaryOp, 12),
				vm.MustMakeInstruction(opcode.Constant, 4),
				vm.MustMakeInstruction(opcode.Constant, 5),
				vm.MustMakeInstruction(opcode.BinaryOp, 13),
				vm.MustMakeInstruction(opcode.Array, 3),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3),
				intObject(4),
				intObject(5),
				intObject(6))))

	expectCompile(t, `{}`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Record, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray()))

	expectCompile(t, `{a: 2, b: 4, c: 6}`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Constant, 3),
				vm.MustMakeInstruction(opcode.Constant, 4),
				vm.MustMakeInstruction(opcode.Constant, 5),
				vm.MustMakeInstruction(opcode.Record, 6),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				stringObject("a"),
				intObject(2),
				stringObject("b"),
				intObject(4),
				stringObject("c"),
				intObject(6))))

	expectCompile(t, `{a: 2 + 3, b: 5 * 6}`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.BinaryOp, 11),
				vm.MustMakeInstruction(opcode.Constant, 3),
				vm.MustMakeInstruction(opcode.Constant, 4),
				vm.MustMakeInstruction(opcode.Constant, 5),
				vm.MustMakeInstruction(opcode.BinaryOp, 13),
				vm.MustMakeInstruction(opcode.Record, 4),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				stringObject("a"),
				intObject(2),
				intObject(3),
				stringObject("b"),
				intObject(5),
				intObject(6))))

	expectCompile(t, `[1, 2, 3][1 + 1]`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Array, 3),
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.BinaryOp, 11),
				vm.MustMakeInstruction(opcode.Index),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3))))

	expectCompile(t, `{a: 2}[2 - 1]`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Record, 2),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.BinaryOp, 12),
				vm.MustMakeInstruction(opcode.Index),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				stringObject("a"),
				intObject(2),
				intObject(1))))

	expectCompile(t, `[1, 2, 3][:]`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Array, 3),
				vm.MustMakeInstruction(opcode.Null),
				vm.MustMakeInstruction(opcode.Null),
				vm.MustMakeInstruction(opcode.SliceIndex),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3))))

	expectCompile(t, `[1, 2, 3][0 : 2]`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Array, 3),
				vm.MustMakeInstruction(opcode.Constant, 3),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.SliceIndex),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3),
				intObject(0))))

	expectCompile(t, `[1, 2, 3][:2]`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Array, 3),
				vm.MustMakeInstruction(opcode.Null),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.SliceIndex),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3))))

	expectCompile(t, `[1, 2, 3][0:]`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Array, 3),
				vm.MustMakeInstruction(opcode.Constant, 3),
				vm.MustMakeInstruction(opcode.Null),
				vm.MustMakeInstruction(opcode.SliceIndex),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3),
				intObject(0))))

	expectCompile(t, `[1, 2, 3][0:3:2]`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Array, 3),
				vm.MustMakeInstruction(opcode.Constant, 3),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.SliceIndexStep),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3),
				intObject(0))))

	expectCompile(t, `f1 := func(a) { return a }; f1([1, 2]...);`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Array, 2),
				vm.MustMakeInstruction(opcode.Call, 1, 1),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				compiledFunction(1, 1,
					vm.MustMakeInstruction(opcode.GetLocal, 0),
					vm.MustMakeInstruction(opcode.Return, 1)),
				intObject(1),
				intObject(2))))

	expectCompile(t, `func() { return 5 + 10 }`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(5),
				intObject(10),
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.Constant, 1),
					vm.MustMakeInstruction(opcode.BinaryOp, 11),
					vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `func() { 5 + 10 }`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(5),
				intObject(10),
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.Constant, 1),
					vm.MustMakeInstruction(opcode.BinaryOp, 11),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.Return, 0)))))

	expectCompile(t, `func() { 1; 2 }`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.Constant, 1),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.Return, 0)))))

	expectCompile(t, `func() { 1; return 2 }`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.Constant, 1),
					vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `func() { if(true) { return 1 } else { return 2 } }`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.True),
					vm.MustMakeInstruction(opcode.JumpFalsy, 9),
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.Return, 1),
					vm.MustMakeInstruction(opcode.Constant, 1),
					vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `func() { 1; if(true) { 2 } else { 3 }; 4 }`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 4),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3),
				intObject(4),
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.True),
					vm.MustMakeInstruction(opcode.JumpFalsy, 15),
					vm.MustMakeInstruction(opcode.Constant, 1),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.Jump, 19),
					vm.MustMakeInstruction(opcode.Constant, 2),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.Constant, 3),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.Return, 0)))))

	expectCompile(t, `func() { }`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.Return, 0)))))

	expectCompile(t, `func() { 24 }()`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Call, 0, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(24),
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.Return, 0)))))

	expectCompile(t, `func() { return 24 }()`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Call, 0, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(24),
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `noArg := func() { 24 }; noArg();`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Call, 0, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(24),
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.Return, 0)))))

	expectCompile(t, `noArg := func() { return 24 }; noArg();`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Call, 0, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(24),
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `n := 55; func() { n };`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(55),
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.GetGlobal, 0),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.Return, 0)))))

	expectCompile(t, `func() { n := 55; return n }`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(55),
				compiledFunction(1, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.DefineLocal, 0),
					vm.MustMakeInstruction(opcode.GetLocal, 0),
					vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `func() { a := 55; b := 77; return a + b }`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(55),
				intObject(77),
				compiledFunction(2, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.DefineLocal, 0),
					vm.MustMakeInstruction(opcode.Constant, 1),
					vm.MustMakeInstruction(opcode.DefineLocal, 1),
					vm.MustMakeInstruction(opcode.GetLocal, 0),
					vm.MustMakeInstruction(opcode.GetLocal, 1),
					vm.MustMakeInstruction(opcode.BinaryOp, 11),
					vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `f1 := func(a) { return a }; f1(24);`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Call, 1, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				compiledFunction(1, 1,
					vm.MustMakeInstruction(opcode.GetLocal, 0),
					vm.MustMakeInstruction(opcode.Return, 1)),
				intObject(24))))

	expectCompile(t, `varTest := func(...a) { return a }; varTest(1,2,3);`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Constant, 3),
				vm.MustMakeInstruction(opcode.Call, 3, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				compiledFunction(1, 1,
					vm.MustMakeInstruction(opcode.GetLocal, 0),
					vm.MustMakeInstruction(opcode.Return, 1)),
				intObject(1), intObject(2), intObject(3))))

	expectCompile(t, `f1 := func(a, b, c) { a; b; return c; }; f1(24, 25, 26);`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Constant, 3),
				vm.MustMakeInstruction(opcode.Call, 3, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				compiledFunction(3, 3,
					vm.MustMakeInstruction(opcode.GetLocal, 0),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.GetLocal, 1),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.GetLocal, 2),
					vm.MustMakeInstruction(opcode.Return, 1)),
				intObject(24),
				intObject(25),
				intObject(26))))

	expectCompile(t, `func() { n := 55; n = 23; return n }`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(55),
				intObject(23),
				compiledFunction(1, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.DefineLocal, 0),
					vm.MustMakeInstruction(opcode.Constant, 1),
					vm.MustMakeInstruction(opcode.SetLocal, 0),
					vm.MustMakeInstruction(opcode.GetLocal, 0),
					vm.MustMakeInstruction(opcode.Return, 1)))))
	expectCompile(t, `len([]);`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.GetBuiltinFunction, 0),
				vm.MustMakeInstruction(opcode.Array, 0),
				vm.MustMakeInstruction(opcode.Call, 1, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray()))

	expectCompile(t, `func() { return len([]) }`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.GetBuiltinFunction, 0),
					vm.MustMakeInstruction(opcode.Array, 0),
					vm.MustMakeInstruction(opcode.Call, 1, 0),
					vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `func(a) { func(b) { return a + b } }`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				compiledFunction(1, 1,
					vm.MustMakeInstruction(opcode.GetFree, 0),
					vm.MustMakeInstruction(opcode.GetLocal, 0),
					vm.MustMakeInstruction(opcode.BinaryOp, 11),
					vm.MustMakeInstruction(opcode.Return, 1)),
				compiledFunction(1, 1,
					vm.MustMakeInstruction(opcode.GetLocalPtr, 0),
					vm.MustMakeInstruction(opcode.Closure, 0, 1),
					vm.MustMakeInstruction(opcode.Pop),
					vm.MustMakeInstruction(opcode.Return, 0)))))

	expectCompile(t, `
func(a) {
	return func(b) {
		return func(c) {
			return a + b + c
		}
	}
}`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				compiledFunction(1, 1,
					vm.MustMakeInstruction(opcode.GetFree, 0),
					vm.MustMakeInstruction(opcode.GetFree, 1),
					vm.MustMakeInstruction(opcode.BinaryOp, 11),
					vm.MustMakeInstruction(opcode.GetLocal, 0),
					vm.MustMakeInstruction(opcode.BinaryOp, 11),
					vm.MustMakeInstruction(opcode.Return, 1)),
				compiledFunction(1, 1,
					vm.MustMakeInstruction(opcode.GetFreePtr, 0),
					vm.MustMakeInstruction(opcode.GetLocalPtr, 0),
					vm.MustMakeInstruction(opcode.Closure, 0, 2),
					vm.MustMakeInstruction(opcode.Return, 1)),
				compiledFunction(1, 1,
					vm.MustMakeInstruction(opcode.GetLocalPtr, 0),
					vm.MustMakeInstruction(opcode.Closure, 1, 1),
					vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `
g := 55;

func() {
	a := 66;

	return func() {
		b := 77;

		return func() {
			c := 88;

			return g + a + b + c;
		}
	}
}`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 6),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(55),
				intObject(66),
				intObject(77),
				intObject(88),
				compiledFunction(1, 0,
					vm.MustMakeInstruction(opcode.Constant, 3),
					vm.MustMakeInstruction(opcode.DefineLocal, 0),
					vm.MustMakeInstruction(opcode.GetGlobal, 0),
					vm.MustMakeInstruction(opcode.GetFree, 0),
					vm.MustMakeInstruction(opcode.BinaryOp, 11),
					vm.MustMakeInstruction(opcode.GetFree, 1),
					vm.MustMakeInstruction(opcode.BinaryOp, 11),
					vm.MustMakeInstruction(opcode.GetLocal, 0),
					vm.MustMakeInstruction(opcode.BinaryOp, 11),
					vm.MustMakeInstruction(opcode.Return, 1)),
				compiledFunction(1, 0,
					vm.MustMakeInstruction(opcode.Constant, 2),
					vm.MustMakeInstruction(opcode.DefineLocal, 0),
					vm.MustMakeInstruction(opcode.GetFreePtr, 0),
					vm.MustMakeInstruction(opcode.GetLocalPtr, 0),
					vm.MustMakeInstruction(opcode.Closure, 4, 2),
					vm.MustMakeInstruction(opcode.Return, 1)),
				compiledFunction(1, 0,
					vm.MustMakeInstruction(opcode.Constant, 1),
					vm.MustMakeInstruction(opcode.DefineLocal, 0),
					vm.MustMakeInstruction(opcode.GetLocalPtr, 0),
					vm.MustMakeInstruction(opcode.Closure, 5, 1),
					vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `for i:=0; i<10; i++ {}`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 38),
				vm.MustMakeInstruction(opcode.JumpFalsy, 31),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.BinaryOp, 11),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.Jump, 6),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(0),
				intObject(10),
				intObject(1))))

	expectCompile(t, `for var i = 0; i<10; i++ {}`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 38),
				vm.MustMakeInstruction(opcode.JumpFalsy, 31),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.BinaryOp, 11),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.Jump, 6),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(0),
				intObject(10),
				intObject(1))))

	expectCompile(t, `m := {}; for k, v in m {}`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Record, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.IteratorInit),
				vm.MustMakeInstruction(opcode.SetGlobal, 1),
				vm.MustMakeInstruction(opcode.GetGlobal, 1),
				vm.MustMakeInstruction(opcode.IteratorNext),
				vm.MustMakeInstruction(opcode.JumpFalsy, 37),
				vm.MustMakeInstruction(opcode.GetGlobal, 1),
				vm.MustMakeInstruction(opcode.IteratorKey),
				vm.MustMakeInstruction(opcode.SetGlobal, 2),
				vm.MustMakeInstruction(opcode.GetGlobal, 1),
				vm.MustMakeInstruction(opcode.IteratorValue),
				vm.MustMakeInstruction(opcode.SetGlobal, 3),
				vm.MustMakeInstruction(opcode.Jump, 13),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray()))

	expectCompile(t, `a := 0; a == 0 && a != 1 || a < 1`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.SetGlobal, 0),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Equal),
				vm.MustMakeInstruction(opcode.AndJump, 23),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.NotEqual),
				vm.MustMakeInstruction(opcode.OrJump, 34),
				vm.MustMakeInstruction(opcode.GetGlobal, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 38),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(0),
				intObject(1))))

	// unknown module name
	expectCompileError(t, `import("user1")`, "module 'user1' not found")

	// too many errors
	expectCompileError(t, `
r["x"] = {
    @a:1,
    @b:1,
    @c:1,
    @d:1,
    @e:1,
    @f:1,
    @g:1,
    @h:1,
    @i:1,
    @j:1,
    @k:1
}
`, "Parse Error: illegal character U+0040 '@'\n\tat test:3:5 (and 10 more errors)")

	expectCompileError(t, `import("")`, "empty module name")

	expectCompileError(t, `
(func() {
	fn := fn()
})()
`, "unresolved reference 'fn")
}

func TestCompilerErrorReport(t *testing.T) {
	expectCompileError(t, `import("user1")`,
		"Compile Error: module 'user1' not found\n\tat test:1:1")

	_, trace, err := traceCompile(`a = 1`, nil)
	if err != nil {
		for _, tr := range trace {
			t.Log(tr)
		}
	}
	require.NoError(t, err)

	expectCompileError(t, `a := a`, "Compile Error: unresolved reference 'a'\n\tat test:1:6")
	expectCompileError(t, `a, b := 1, 2`, "Compile Error: tuple assignment not allowed\n\tat test:1:1")
	expectCompileError(t, `a.b := 1`, "not allowed with selector")
	expectCompileError(t, `a:=1; a:=3`, "Compile Error: 'a' redeclared in this block\n\tat test:1:7")
	expectCompileError(t, `var a = 1; var a = 3`, "Compile Error: 'a' redeclared in this block\n\tat test:1:16")
	expectCompileError(t, `return 5`, "Compile Error: return not allowed outside function\n\tat test:1:1")
	expectCompileError(t, `func() { break }`, "Compile Error: break not allowed outside loop\n\tat test:1:10")
	expectCompileError(t, `func() { continue }`, "Compile Error: continue not allowed outside loop\n\tat test:1:10")
	expectCompileError(t, `func() { export 5 }`, "Compile Error: export not allowed inside function\n\tat test:1:10")
}

func TestCompilerAssignmentMode(t *testing.T) {
	_, _, err := traceCompileWithMode(`a = 1`, nil, compiler.AssignmentModeSmart)
	require.NoError(t, err)

	_, _, err = traceCompileWithMode(`a = 1`, nil, compiler.AssignmentModeStrict)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unresolved reference 'a'"))

	_, _, err = traceCompileWithMode(`a += 1`, nil, compiler.AssignmentModeSmart)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unresolved reference 'a'"))

	_, _, err = traceCompileWithMode(`a.b = 1`, nil, compiler.AssignmentModeSmart)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unresolved reference 'a'"))
}

func TestCompilerDeadCode(t *testing.T) {
	expectCompile(t, `
func() {
	a := 4
	return a

	b := 5 // dead code from here
	c := a
	return b
}`,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Suspend)),
			objectsArray(
				intObject(4),
				intObject(5),
				compiledFunction(0, 0,
					vm.MustMakeInstruction(opcode.Constant, 0),
					vm.MustMakeInstruction(opcode.DefineLocal, 0),
					vm.MustMakeInstruction(opcode.GetLocal, 0),
					vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `
func() {
	if true {
		return 5
		a := 4  // dead code from here
		b := a
		return b
	} else {
		return 4
		c := 5  // dead code from here
		d := c
		return d
	}
}`, bytecode(
		concatInsts(
			vm.MustMakeInstruction(opcode.Constant, 2),
			vm.MustMakeInstruction(opcode.Pop),
			vm.MustMakeInstruction(opcode.Suspend)),
		objectsArray(
			intObject(5),
			intObject(4),
			compiledFunction(0, 0,
				vm.MustMakeInstruction(opcode.True),
				vm.MustMakeInstruction(opcode.JumpFalsy, 9),
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Return, 1),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `
func() {
	a := 1
	for {
		if a == 5 {
			return 10
		}
		5 + 5
		return 20
		b := a
		return b
	}
}`, bytecode(
		concatInsts(
			vm.MustMakeInstruction(opcode.Constant, 4),
			vm.MustMakeInstruction(opcode.Pop),
			vm.MustMakeInstruction(opcode.Suspend)),
		objectsArray(
			intObject(1),
			intObject(5),
			intObject(10),
			intObject(20),
			compiledFunction(0, 0,
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.DefineLocal, 0),
				vm.MustMakeInstruction(opcode.GetLocal, 0),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Equal),
				vm.MustMakeInstruction(opcode.JumpFalsy, 19),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.Return, 1),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.BinaryOp, 11),
				vm.MustMakeInstruction(opcode.Pop),
				vm.MustMakeInstruction(opcode.Constant, 3),
				vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `
func() {
	if true {
		return 5
		a := 4  // dead code from here
		b := a
		return b
	} else {
		return 4
		c := 5  // dead code from here
		d := c
		return d
	}
}`, bytecode(
		concatInsts(
			vm.MustMakeInstruction(opcode.Constant, 2),
			vm.MustMakeInstruction(opcode.Pop),
			vm.MustMakeInstruction(opcode.Suspend)),
		objectsArray(
			intObject(5),
			intObject(4),
			compiledFunction(0, 0,
				vm.MustMakeInstruction(opcode.True),
				vm.MustMakeInstruction(opcode.JumpFalsy, 9),
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Return, 1),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.Return, 1)))))

	expectCompile(t, `
func() {
	if true {
		return
	}

    return

    return 123
}`, bytecode(
		concatInsts(
			vm.MustMakeInstruction(opcode.Constant, 1),
			vm.MustMakeInstruction(opcode.Pop),
			vm.MustMakeInstruction(opcode.Suspend)),
		objectsArray(
			intObject(123),
			compiledFunction(0, 0,
				vm.MustMakeInstruction(opcode.True),
				vm.MustMakeInstruction(opcode.JumpFalsy, 6),
				vm.MustMakeInstruction(opcode.Return, 0),
				vm.MustMakeInstruction(opcode.Return, 0),
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.Return, 1)))))
}

func TestCompilerScopes(t *testing.T) {
	expectCompile(t, `
if a := 1; a {
    a = 2
	b := a
} else {
    a = 3
	b := a
}`, bytecode(
		concatInsts(
			vm.MustMakeInstruction(opcode.Constant, 0),
			vm.MustMakeInstruction(opcode.SetGlobal, 0),
			vm.MustMakeInstruction(opcode.GetGlobal, 0),
			vm.MustMakeInstruction(opcode.JumpFalsy, 27),
			vm.MustMakeInstruction(opcode.Constant, 1),
			vm.MustMakeInstruction(opcode.SetGlobal, 0),
			vm.MustMakeInstruction(opcode.GetGlobal, 0),
			vm.MustMakeInstruction(opcode.SetGlobal, 1),
			vm.MustMakeInstruction(opcode.Jump, 39),
			vm.MustMakeInstruction(opcode.Constant, 2),
			vm.MustMakeInstruction(opcode.SetGlobal, 0),
			vm.MustMakeInstruction(opcode.GetGlobal, 0),
			vm.MustMakeInstruction(opcode.SetGlobal, 2),
			vm.MustMakeInstruction(opcode.Suspend)),
		objectsArray(
			intObject(1),
			intObject(2),
			intObject(3))))

	expectCompile(t, `
if var a = 1; a {
    a = 2
	b := a
} else {
    a = 3
	b := a
}`, bytecode(
		concatInsts(
			vm.MustMakeInstruction(opcode.Constant, 0),
			vm.MustMakeInstruction(opcode.SetGlobal, 0),
			vm.MustMakeInstruction(opcode.GetGlobal, 0),
			vm.MustMakeInstruction(opcode.JumpFalsy, 27),
			vm.MustMakeInstruction(opcode.Constant, 1),
			vm.MustMakeInstruction(opcode.SetGlobal, 0),
			vm.MustMakeInstruction(opcode.GetGlobal, 0),
			vm.MustMakeInstruction(opcode.SetGlobal, 1),
			vm.MustMakeInstruction(opcode.Jump, 39),
			vm.MustMakeInstruction(opcode.Constant, 2),
			vm.MustMakeInstruction(opcode.SetGlobal, 0),
			vm.MustMakeInstruction(opcode.GetGlobal, 0),
			vm.MustMakeInstruction(opcode.SetGlobal, 2),
			vm.MustMakeInstruction(opcode.Suspend)),
		objectsArray(
			intObject(1),
			intObject(2),
			intObject(3))))

	expectCompile(t, `
func() {
	if a := 1; a {
    	a = 2
		b := a
	} else {
    	a = 3
		b := a
	}
}`, bytecode(
		concatInsts(
			vm.MustMakeInstruction(opcode.Constant, 3),
			vm.MustMakeInstruction(opcode.Pop),
			vm.MustMakeInstruction(opcode.Suspend)),
		objectsArray(
			intObject(1),
			intObject(2),
			intObject(3),
			compiledFunction(0, 0,
				vm.MustMakeInstruction(opcode.Constant, 0),
				vm.MustMakeInstruction(opcode.DefineLocal, 0),
				vm.MustMakeInstruction(opcode.GetLocal, 0),
				vm.MustMakeInstruction(opcode.JumpFalsy, 22),
				vm.MustMakeInstruction(opcode.Constant, 1),
				vm.MustMakeInstruction(opcode.SetLocal, 0),
				vm.MustMakeInstruction(opcode.GetLocal, 0),
				vm.MustMakeInstruction(opcode.DefineLocal, 1),
				vm.MustMakeInstruction(opcode.Jump, 31),
				vm.MustMakeInstruction(opcode.Constant, 2),
				vm.MustMakeInstruction(opcode.SetLocal, 0),
				vm.MustMakeInstruction(opcode.GetLocal, 0),
				vm.MustMakeInstruction(opcode.DefineLocal, 1),
				vm.MustMakeInstruction(opcode.Return, 0)))))
}

func TestCompiler_custom_extension(t *testing.T) {
	pathFileSource := "../testdata/issue286/test.yb"

	src, err := os.ReadFile(pathFileSource)
	require.NoError(t, err)

	// Escape hashbang
	if len(src) > 1 && string(src[:2]) == "#!" {
		copy(src, "//")
	}

	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile(filepath.Base(pathFileSource), -1, len(src))

	p := parser.NewParser(srcFile, src, nil)
	file, err := p.ParseFile()
	require.NoError(t, err)

	c := compiler.New(cta, srcFile, nil, nil, nil, nil)
	c.EnableFileImport(true)
	c.SetImportDir(filepath.Dir(pathFileSource))

	// Search for "*.kvn" and ".yb" (custom extension)
	c.SetImportFileExt(".kvn", ".yb")

	err = c.Compile(file)
	require.NoError(t, err)
}

func TestCompilerNew_default_file_extension(t *testing.T) {
	input := "{}"
	fileSet := parser.NewFileSet()
	file := fileSet.AddFile("test", -1, len(input))

	c := compiler.New(cta, file, nil, nil, nil, nil)
	c.EnableFileImport(true)

	require.Equal(t, rta, []string{".kvn"}, c.GetImportFileExt(), "newly created compiler object must contain the default extension")
}

func TestCompilerSetImportExt_extension_name_validation(t *testing.T) {
	c := new(compiler.Compiler) // Instantiate a new compiler object with no initialization

	// Test of empty arg
	err := c.SetImportFileExt()

	require.Error(t, err, "empty arg should return an error")

	// Test of various arg types
	for _, test := range []struct {
		extensions []string
		expect     []string
		requireErr bool
		msgFail    string
	}{
		{[]string{".kvn"}, []string{".kvn"}, false, "well-formed extension should not return an error"},
		{[]string{""}, []string{".kvn"}, true, "empty extension name should return an error"},
		{[]string{"foo"}, []string{".kvn"}, true, "name without dot prefix should return an error"},
		{[]string{"foo.bar"}, []string{".kvn"}, true, "malformed extension should return an error"},
		{[]string{"foo."}, []string{".kvn"}, true, "malformed extension should return an error"},
		{[]string{".yb"}, []string{".yb"}, false, "name with dot prefix should be added"},
		{[]string{".foo", ".bar"}, []string{".foo", ".bar"}, false, "it should replace instead of appending"},
	} {
		err := c.SetImportFileExt(test.extensions...)
		if test.requireErr {
			require.Error(t, err, test.msgFail)
		}

		expect := test.expect
		actual := c.GetImportFileExt()
		require.Equal(t, rta, expect, actual, test.msgFail)
	}
}

func concatInsts(instructions ...[]byte) []byte {
	var concat []byte
	for _, i := range instructions {
		concat = append(concat, i...)
	}
	return concat
}

func bytecode(instructions []byte, constants []core.Value) *vm.Bytecode {
	return &vm.Bytecode{
		FileSet:      parser.NewFileSet(),
		MainFunction: &core.CompiledFunction{Instructions: instructions},
		Constants:    constants,
	}
}

func expectCompile(t *testing.T, input string, expected *vm.Bytecode) {
	actual, trace, err := traceCompile(input, nil)

	var ok bool
	defer func() {
		if !ok {
			for _, tr := range trace {
				t.Log(tr)
			}
		}
	}()

	require.NoError(t, err)
	equalBytecode(t, expected, actual)
	ok = true
}

func expectCompileError(t *testing.T, input, expected string) {
	_, trace, err := traceCompile(input, nil)

	var ok bool
	defer func() {
		if !ok {
			for _, tr := range trace {
				t.Log(tr)
			}
		}
	}()

	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), expected),
		"expected error string: %s, got: %s", expected, err.Error())
	ok = true
}

func equalBytecode(t *testing.T, expected, actual *vm.Bytecode) {
	require.Equal(t, rta, expected.MainFunction, actual.MainFunction)
	equalConstants(t, expected.Constants, actual.Constants)
}

func equalConstants(t *testing.T, expected, actual []core.Value) {
	require.Equal(t, rta, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		require.Equal(t, rta, expected[i], actual[i])
	}
}

type compileTracer struct {
	Out []string
}

func (o *compileTracer) Write(p []byte) (n int, err error) {
	o.Out = append(o.Out, string(p))
	return len(p), nil
}

func traceCompile(input string, symbols map[string]core.Value) (res *vm.Bytecode, trace []string, err error) {
	return traceCompileWithMode(input, symbols, compiler.AssignmentModeSmart)
}

func traceCompileWithMode(input string, symbols map[string]core.Value, mode compiler.AssignmentMode) (res *vm.Bytecode, trace []string, err error) {
	fileSet := parser.NewFileSet()
	file := fileSet.AddFile("test", -1, len(input))

	p := parser.NewParser(file, []byte(input), nil)

	symTable := vm.NewSymbolTable()
	for name := range symbols {
		symTable.Define(name)
	}
	for idx, name := range vm.BuiltinFunctionNames {
		symTable.DefineBuiltin(idx, name)
	}

	tr := &compileTracer{}
	c := compiler.New(cta, file, symTable, nil, nil, tr)
	c.SetAssignmentMode(mode)
	parsed, err := p.ParseFile()
	if err != nil {
		return
	}

	err = c.Compile(parsed)
	res = c.Bytecode()
	if err == nil {
		err = res.RemoveDuplicates(rta)
	}

	trace = append(trace, fmt.Sprintf("Compiler Trace:\n%s", strings.Join(tr.Out, "")))
	trace = append(trace, fmt.Sprintf("Compiled Constants:\n%s", strings.Join(res.MustFormatConstants(rta), "\n")))
	trace = append(trace, fmt.Sprintf("Compiled Instructions:\n%s\n", strings.Join(res.MustFormatInstructions(), "\n")))

	return
}

func objectsArray(o ...core.Value) []core.Value {
	return o
}

func intObject(v int64) core.Value {
	return core.IntValue(v)
}

func stringObject(v string) core.Value {
	return rta.NewStringValue(v)
}

func compiledFunction(numLocals int, numParams int8, insts ...[]byte) core.Value {
	return rta.NewCompiledFunctionValue(concatInsts(insts...), nil, nil, numLocals, 0, numParams, false, 0)
}
