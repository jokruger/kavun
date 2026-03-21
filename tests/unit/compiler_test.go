package gs_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jokruger/gs"
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/stdlib"
	"github.com/jokruger/gs/tests/require"
	"github.com/jokruger/gs/value"
	"github.com/jokruger/gs/vm"
)

func TestCompiler_Compile(t *testing.T) {
	expectCompile(t, `1 + 2`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 11),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1; 2`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 - 2`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 12),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 * 2`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 13),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `2 / 1`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 14),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(2),
				intObject(1))))

	expectCompile(t, `true`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpTrue),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray()))

	expectCompile(t, `false`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpFalse),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray()))

	expectCompile(t, `1 > 2`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 39),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 < 2`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 38),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 >= 2`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 44),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 <= 2`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 43),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 == 2`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpEqual),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `1 != 2`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpNotEqual),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `true == false`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpTrue),
				vm.MakeInstruction(parser.OpFalse),
				vm.MakeInstruction(parser.OpEqual),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray()))

	expectCompile(t, `true != false`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpTrue),
				vm.MakeInstruction(parser.OpFalse),
				vm.MakeInstruction(parser.OpNotEqual),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray()))

	expectCompile(t, `-1`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpMinus),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1))))

	expectCompile(t, `!true`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpTrue),
				vm.MakeInstruction(parser.OpLNot),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray()))

	expectCompile(t, `if true { 10 }; 3333`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpTrue),          // 0000
				vm.MakeInstruction(parser.OpJumpFalsy, 10), // 0001
				vm.MakeInstruction(parser.OpConstant, 0),   // 0004
				vm.MakeInstruction(parser.OpPop),           // 0007
				vm.MakeInstruction(parser.OpConstant, 1),   // 0008
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)), // 0011
			objectsArray(
				intObject(10),
				intObject(3333))))

	expectCompile(t, `if (true) { 10 } else { 20 }; 3333;`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpTrue),          // 0000
				vm.MakeInstruction(parser.OpJumpFalsy, 15), // 0001
				vm.MakeInstruction(parser.OpConstant, 0),   // 0004
				vm.MakeInstruction(parser.OpPop),           // 0007
				vm.MakeInstruction(parser.OpJump, 19),      // 0008
				vm.MakeInstruction(parser.OpConstant, 1),   // 0011
				vm.MakeInstruction(parser.OpPop),           // 0014
				vm.MakeInstruction(parser.OpConstant, 2),   // 0015
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)), // 0018
			objectsArray(
				intObject(10),
				intObject(20),
				intObject(3333))))

	expectCompile(t, `"kami"`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				stringObject("kami"))))

	expectCompile(t, `"ka" + "mi"`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 11),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				stringObject("ka"),
				stringObject("mi"))))

	expectCompile(t, `a := 1; b := 2; a += b`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpSetGlobal, 1),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 11),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `a := 1; b := 2; a /= b`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpSetGlobal, 1),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 14),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2))))

	expectCompile(t, `[]`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpArray, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray()))

	expectCompile(t, `[1, 2, 3]`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpArray, 3),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3))))

	expectCompile(t, `[1 + 2, 3 - 4, 5 * 6]`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 11),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpBinaryOp, 12),
				vm.MakeInstruction(parser.OpConstant, 4),
				vm.MakeInstruction(parser.OpConstant, 5),
				vm.MakeInstruction(parser.OpBinaryOp, 13),
				vm.MakeInstruction(parser.OpArray, 3),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
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
				vm.MakeInstruction(parser.OpRecord, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray()))

	expectCompile(t, `{a: 2, b: 4, c: 6}`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpConstant, 4),
				vm.MakeInstruction(parser.OpConstant, 5),
				vm.MakeInstruction(parser.OpRecord, 6),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
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
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpBinaryOp, 11),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpConstant, 4),
				vm.MakeInstruction(parser.OpConstant, 5),
				vm.MakeInstruction(parser.OpBinaryOp, 13),
				vm.MakeInstruction(parser.OpRecord, 4),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
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
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpArray, 3),
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpBinaryOp, 11),
				vm.MakeInstruction(parser.OpIndex),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3))))

	expectCompile(t, `{a: 2}[2 - 1]`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpRecord, 2),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpBinaryOp, 12),
				vm.MakeInstruction(parser.OpIndex),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				stringObject("a"),
				intObject(2),
				intObject(1))))

	expectCompile(t, `[1, 2, 3][:]`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpArray, 3),
				vm.MakeInstruction(parser.OpNull),
				vm.MakeInstruction(parser.OpNull),
				vm.MakeInstruction(parser.OpSliceIndex),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3))))

	expectCompile(t, `[1, 2, 3][0 : 2]`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpArray, 3),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpSliceIndex),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3),
				intObject(0))))

	expectCompile(t, `[1, 2, 3][:2]`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpArray, 3),
				vm.MakeInstruction(parser.OpNull),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpSliceIndex),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3))))

	expectCompile(t, `[1, 2, 3][0:]`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpArray, 3),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpNull),
				vm.MakeInstruction(parser.OpSliceIndex),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3),
				intObject(0))))

	expectCompile(t, `f1 := func(a) { return a }; f1([1, 2]...);`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpArray, 2),
				vm.MakeInstruction(parser.OpCall, 1, 1),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				compiledFunction(1, 1,
					vm.MakeInstruction(parser.OpGetLocal, 0),
					vm.MakeInstruction(parser.OpReturn, 1)),
				intObject(1),
				intObject(2))))

	expectCompile(t, `func() { return 5 + 10 }`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(5),
				intObject(10),
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpConstant, 0),
					vm.MakeInstruction(parser.OpConstant, 1),
					vm.MakeInstruction(parser.OpBinaryOp, 11),
					vm.MakeInstruction(parser.OpReturn, 1)))))

	expectCompile(t, `func() { 5 + 10 }`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(5),
				intObject(10),
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpConstant, 0),
					vm.MakeInstruction(parser.OpConstant, 1),
					vm.MakeInstruction(parser.OpBinaryOp, 11),
					vm.MakeInstruction(parser.OpPop),
					vm.MakeInstruction(parser.OpReturn, 0)))))

	expectCompile(t, `func() { 1; 2 }`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpConstant, 0),
					vm.MakeInstruction(parser.OpPop),
					vm.MakeInstruction(parser.OpConstant, 1),
					vm.MakeInstruction(parser.OpPop),
					vm.MakeInstruction(parser.OpReturn, 0)))))

	expectCompile(t, `func() { 1; return 2 }`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpConstant, 0),
					vm.MakeInstruction(parser.OpPop),
					vm.MakeInstruction(parser.OpConstant, 1),
					vm.MakeInstruction(parser.OpReturn, 1)))))

	expectCompile(t, `func() { if(true) { return 1 } else { return 2 } }`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpTrue),          // 0000
					vm.MakeInstruction(parser.OpJumpFalsy, 11), // 0001
					vm.MakeInstruction(parser.OpConstant, 0),   // 0004
					vm.MakeInstruction(parser.OpReturn, 1),     // 0007
					vm.MakeInstruction(parser.OpConstant, 1),   // 0009
					vm.MakeInstruction(parser.OpReturn, 1)))))  // 0012

	expectCompile(t, `func() { 1; if(true) { 2 } else { 3 }; 4 }`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 4),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(1),
				intObject(2),
				intObject(3),
				intObject(4),
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpConstant, 0),   // 0000
					vm.MakeInstruction(parser.OpPop),           // 0003
					vm.MakeInstruction(parser.OpTrue),          // 0004
					vm.MakeInstruction(parser.OpJumpFalsy, 19), // 0005
					vm.MakeInstruction(parser.OpConstant, 1),   // 0008
					vm.MakeInstruction(parser.OpPop),           // 0011
					vm.MakeInstruction(parser.OpJump, 23),      // 0012
					vm.MakeInstruction(parser.OpConstant, 2),   // 0015
					vm.MakeInstruction(parser.OpPop),           // 0018
					vm.MakeInstruction(parser.OpConstant, 3),   // 0019
					vm.MakeInstruction(parser.OpPop),           // 0022
					vm.MakeInstruction(parser.OpReturn, 0)))))  // 0023

	expectCompile(t, `func() { }`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpReturn, 0)))))

	expectCompile(t, `func() { 24 }()`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpCall, 0, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(24),
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpConstant, 0),
					vm.MakeInstruction(parser.OpPop),
					vm.MakeInstruction(parser.OpReturn, 0)))))

	expectCompile(t, `func() { return 24 }()`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpCall, 0, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(24),
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpConstant, 0),
					vm.MakeInstruction(parser.OpReturn, 1)))))

	expectCompile(t, `noArg := func() { 24 }; noArg();`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpCall, 0, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(24),
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpConstant, 0),
					vm.MakeInstruction(parser.OpPop),
					vm.MakeInstruction(parser.OpReturn, 0)))))

	expectCompile(t, `noArg := func() { return 24 }; noArg();`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpCall, 0, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(24),
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpConstant, 0),
					vm.MakeInstruction(parser.OpReturn, 1)))))

	expectCompile(t, `n := 55; func() { n };`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(55),
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpGetGlobal, 0),
					vm.MakeInstruction(parser.OpPop),
					vm.MakeInstruction(parser.OpReturn, 0)))))

	expectCompile(t, `func() { n := 55; return n }`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(55),
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 0),
					vm.MakeInstruction(parser.OpDefineLocal, 0),
					vm.MakeInstruction(parser.OpGetLocal, 0),
					vm.MakeInstruction(parser.OpReturn, 1)))))

	expectCompile(t, `func() { a := 55; b := 77; return a + b }`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(55),
				intObject(77),
				compiledFunction(2, 0,
					vm.MakeInstruction(parser.OpConstant, 0),
					vm.MakeInstruction(parser.OpDefineLocal, 0),
					vm.MakeInstruction(parser.OpConstant, 1),
					vm.MakeInstruction(parser.OpDefineLocal, 1),
					vm.MakeInstruction(parser.OpGetLocal, 0),
					vm.MakeInstruction(parser.OpGetLocal, 1),
					vm.MakeInstruction(parser.OpBinaryOp, 11),
					vm.MakeInstruction(parser.OpReturn, 1)))))

	expectCompile(t, `f1 := func(a) { return a }; f1(24);`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpCall, 1, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				compiledFunction(1, 1,
					vm.MakeInstruction(parser.OpGetLocal, 0),
					vm.MakeInstruction(parser.OpReturn, 1)),
				intObject(24))))

	expectCompile(t, `varTest := func(...a) { return a }; varTest(1,2,3);`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpCall, 3, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				compiledFunction(1, 1,
					vm.MakeInstruction(parser.OpGetLocal, 0),
					vm.MakeInstruction(parser.OpReturn, 1)),
				intObject(1), intObject(2), intObject(3))))

	expectCompile(t, `f1 := func(a, b, c) { a; b; return c; }; f1(24, 25, 26);`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpCall, 3, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				compiledFunction(3, 3,
					vm.MakeInstruction(parser.OpGetLocal, 0),
					vm.MakeInstruction(parser.OpPop),
					vm.MakeInstruction(parser.OpGetLocal, 1),
					vm.MakeInstruction(parser.OpPop),
					vm.MakeInstruction(parser.OpGetLocal, 2),
					vm.MakeInstruction(parser.OpReturn, 1)),
				intObject(24),
				intObject(25),
				intObject(26))))

	expectCompile(t, `func() { n := 55; n = 23; return n }`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(55),
				intObject(23),
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 0),
					vm.MakeInstruction(parser.OpDefineLocal, 0),
					vm.MakeInstruction(parser.OpConstant, 1),
					vm.MakeInstruction(parser.OpSetLocal, 0),
					vm.MakeInstruction(parser.OpGetLocal, 0),
					vm.MakeInstruction(parser.OpReturn, 1)))))
	expectCompile(t, `len([]);`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpGetBuiltin, 0),
				vm.MakeInstruction(parser.OpArray, 0),
				vm.MakeInstruction(parser.OpCall, 1, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray()))

	expectCompile(t, `func() { return len([]) }`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpGetBuiltin, 0),
					vm.MakeInstruction(parser.OpArray, 0),
					vm.MakeInstruction(parser.OpCall, 1, 0),
					vm.MakeInstruction(parser.OpReturn, 1)))))

	expectCompile(t, `func(a) { func(b) { return a + b } }`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				compiledFunction(1, 1,
					vm.MakeInstruction(parser.OpGetFree, 0),
					vm.MakeInstruction(parser.OpGetLocal, 0),
					vm.MakeInstruction(parser.OpBinaryOp, 11),
					vm.MakeInstruction(parser.OpReturn, 1)),
				compiledFunction(1, 1,
					vm.MakeInstruction(parser.OpGetLocalPtr, 0),
					vm.MakeInstruction(parser.OpClosure, 0, 1),
					vm.MakeInstruction(parser.OpPop),
					vm.MakeInstruction(parser.OpReturn, 0)))))

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
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				compiledFunction(1, 1,
					vm.MakeInstruction(parser.OpGetFree, 0),
					vm.MakeInstruction(parser.OpGetFree, 1),
					vm.MakeInstruction(parser.OpBinaryOp, 11),
					vm.MakeInstruction(parser.OpGetLocal, 0),
					vm.MakeInstruction(parser.OpBinaryOp, 11),
					vm.MakeInstruction(parser.OpReturn, 1)),
				compiledFunction(1, 1,
					vm.MakeInstruction(parser.OpGetFreePtr, 0),
					vm.MakeInstruction(parser.OpGetLocalPtr, 0),
					vm.MakeInstruction(parser.OpClosure, 0, 2),
					vm.MakeInstruction(parser.OpReturn, 1)),
				compiledFunction(1, 1,
					vm.MakeInstruction(parser.OpGetLocalPtr, 0),
					vm.MakeInstruction(parser.OpClosure, 1, 1),
					vm.MakeInstruction(parser.OpReturn, 1)))))

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
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 6),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(55),
				intObject(66),
				intObject(77),
				intObject(88),
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 3),
					vm.MakeInstruction(parser.OpDefineLocal, 0),
					vm.MakeInstruction(parser.OpGetGlobal, 0),
					vm.MakeInstruction(parser.OpGetFree, 0),
					vm.MakeInstruction(parser.OpBinaryOp, 11),
					vm.MakeInstruction(parser.OpGetFree, 1),
					vm.MakeInstruction(parser.OpBinaryOp, 11),
					vm.MakeInstruction(parser.OpGetLocal, 0),
					vm.MakeInstruction(parser.OpBinaryOp, 11),
					vm.MakeInstruction(parser.OpReturn, 1)),
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 2),
					vm.MakeInstruction(parser.OpDefineLocal, 0),
					vm.MakeInstruction(parser.OpGetFreePtr, 0),
					vm.MakeInstruction(parser.OpGetLocalPtr, 0),
					vm.MakeInstruction(parser.OpClosure, 4, 2),
					vm.MakeInstruction(parser.OpReturn, 1)),
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 1),
					vm.MakeInstruction(parser.OpDefineLocal, 0),
					vm.MakeInstruction(parser.OpGetLocalPtr, 0),
					vm.MakeInstruction(parser.OpClosure, 5, 1),
					vm.MakeInstruction(parser.OpReturn, 1)))))

	expectCompile(t, `for i:=0; i<10; i++ {}`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 38),
				vm.MakeInstruction(parser.OpJumpFalsy, 35),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpBinaryOp, 11),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpJump, 6),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(0),
				intObject(10),
				intObject(1))))

	expectCompile(t, `m := {}; for k, v in m {}`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpRecord, 0),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpIteratorInit),
				vm.MakeInstruction(parser.OpSetGlobal, 1),
				vm.MakeInstruction(parser.OpGetGlobal, 1),
				vm.MakeInstruction(parser.OpIteratorNext),
				vm.MakeInstruction(parser.OpJumpFalsy, 41),
				vm.MakeInstruction(parser.OpGetGlobal, 1),
				vm.MakeInstruction(parser.OpIteratorKey),
				vm.MakeInstruction(parser.OpSetGlobal, 2),
				vm.MakeInstruction(parser.OpGetGlobal, 1),
				vm.MakeInstruction(parser.OpIteratorValue),
				vm.MakeInstruction(parser.OpSetGlobal, 3),
				vm.MakeInstruction(parser.OpJump, 13),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray()))

	expectCompile(t, `a := 0; a == 0 && a != 1 || a < 1`,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpSetGlobal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpEqual),
				vm.MakeInstruction(parser.OpAndJump, 25),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpNotEqual),
				vm.MakeInstruction(parser.OpOrJump, 38),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 38),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
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

	expectCompileError(t, `a = 1`,
		"Compile Error: unresolved reference 'a'\n\tat test:1:1")
	expectCompileError(t, `a := a`,
		"Compile Error: unresolved reference 'a'\n\tat test:1:6")
	expectCompileError(t, `a, b := 1, 2`,
		"Compile Error: tuple assignment not allowed\n\tat test:1:1")
	expectCompileError(t, `a.b := 1`,
		"not allowed with selector")
	expectCompileError(t, `a:=1; a:=3`,
		"Compile Error: 'a' redeclared in this block\n\tat test:1:7")

	expectCompileError(t, `return 5`,
		"Compile Error: return not allowed outside function\n\tat test:1:1")
	expectCompileError(t, `func() { break }`,
		"Compile Error: break not allowed outside loop\n\tat test:1:10")
	expectCompileError(t, `func() { continue }`,
		"Compile Error: continue not allowed outside loop\n\tat test:1:10")
	expectCompileError(t, `func() { export 5 }`,
		"Compile Error: export not allowed inside function\n\tat test:1:10")
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
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpSuspend)),
			objectsArray(
				intObject(4),
				intObject(5),
				compiledFunction(0, 0,
					vm.MakeInstruction(parser.OpConstant, 0),
					vm.MakeInstruction(parser.OpDefineLocal, 0),
					vm.MakeInstruction(parser.OpGetLocal, 0),
					vm.MakeInstruction(parser.OpReturn, 1)))))

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
			vm.MakeInstruction(parser.OpConstant, 2),
			vm.MakeInstruction(parser.OpPop),
			vm.MakeInstruction(parser.OpSuspend)),
		objectsArray(
			intObject(5),
			intObject(4),
			compiledFunction(0, 0,
				vm.MakeInstruction(parser.OpTrue),
				vm.MakeInstruction(parser.OpJumpFalsy, 11),
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpReturn, 1),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpReturn, 1)))))

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
			vm.MakeInstruction(parser.OpConstant, 4),
			vm.MakeInstruction(parser.OpPop),
			vm.MakeInstruction(parser.OpSuspend)),
		objectsArray(
			intObject(1),
			intObject(5),
			intObject(10),
			intObject(20),
			compiledFunction(0, 0,
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpDefineLocal, 0),
				vm.MakeInstruction(parser.OpGetLocal, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpEqual),
				vm.MakeInstruction(parser.OpJumpFalsy, 21),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpReturn, 1),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 11),
				vm.MakeInstruction(parser.OpPop),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpReturn, 1)))))

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
			vm.MakeInstruction(parser.OpConstant, 2),
			vm.MakeInstruction(parser.OpPop),
			vm.MakeInstruction(parser.OpSuspend)),
		objectsArray(
			intObject(5),
			intObject(4),
			compiledFunction(0, 0,
				vm.MakeInstruction(parser.OpTrue),
				vm.MakeInstruction(parser.OpJumpFalsy, 11),
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpReturn, 1),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpReturn, 1)))))

	expectCompile(t, `
func() {
	if true {
		return
	}

    return

    return 123
}`, bytecode(
		concatInsts(
			vm.MakeInstruction(parser.OpConstant, 1),
			vm.MakeInstruction(parser.OpPop),
			vm.MakeInstruction(parser.OpSuspend)),
		objectsArray(
			intObject(123),
			compiledFunction(0, 0,
				vm.MakeInstruction(parser.OpTrue),
				vm.MakeInstruction(parser.OpJumpFalsy, 8),
				vm.MakeInstruction(parser.OpReturn, 0),
				vm.MakeInstruction(parser.OpReturn, 0),
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpReturn, 1)))))
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
			vm.MakeInstruction(parser.OpConstant, 0),
			vm.MakeInstruction(parser.OpSetGlobal, 0),
			vm.MakeInstruction(parser.OpGetGlobal, 0),
			vm.MakeInstruction(parser.OpJumpFalsy, 31),
			vm.MakeInstruction(parser.OpConstant, 1),
			vm.MakeInstruction(parser.OpSetGlobal, 0),
			vm.MakeInstruction(parser.OpGetGlobal, 0),
			vm.MakeInstruction(parser.OpSetGlobal, 1),
			vm.MakeInstruction(parser.OpJump, 43),
			vm.MakeInstruction(parser.OpConstant, 2),
			vm.MakeInstruction(parser.OpSetGlobal, 0),
			vm.MakeInstruction(parser.OpGetGlobal, 0),
			vm.MakeInstruction(parser.OpSetGlobal, 2),
			vm.MakeInstruction(parser.OpSuspend)),
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
			vm.MakeInstruction(parser.OpConstant, 3),
			vm.MakeInstruction(parser.OpPop),
			vm.MakeInstruction(parser.OpSuspend)),
		objectsArray(
			intObject(1),
			intObject(2),
			intObject(3),
			compiledFunction(0, 0,
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpDefineLocal, 0),
				vm.MakeInstruction(parser.OpGetLocal, 0),
				vm.MakeInstruction(parser.OpJumpFalsy, 26),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpSetLocal, 0),
				vm.MakeInstruction(parser.OpGetLocal, 0),
				vm.MakeInstruction(parser.OpDefineLocal, 1),
				vm.MakeInstruction(parser.OpJump, 35),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpSetLocal, 0),
				vm.MakeInstruction(parser.OpGetLocal, 0),
				vm.MakeInstruction(parser.OpDefineLocal, 1),
				vm.MakeInstruction(parser.OpReturn, 0)))))
}

func TestCompiler_custom_extension(t *testing.T) {
	pathFileSource := "../testdata/issue286/test.yb"

	modules := stdlib.GetModuleMap(stdlib.AllModuleNames()...)

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

	c := gs.NewCompiler(srcFile, nil, nil, modules, nil)
	c.EnableFileImport(true)
	c.SetImportDir(filepath.Dir(pathFileSource))

	// Search for "*.gs" and ".yb" (custom extension)
	c.SetImportFileExt(".gs", ".yb")

	err = c.Compile(file)
	require.NoError(t, err)
}

func TestCompilerNewCompiler_default_file_extension(t *testing.T) {
	modules := stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	input := "{}"
	fileSet := parser.NewFileSet()
	file := fileSet.AddFile("test", -1, len(input))

	c := gs.NewCompiler(file, nil, nil, modules, nil)
	c.EnableFileImport(true)

	require.Equal(t, []string{".gs"}, c.GetImportFileExt(), "newly created compiler object must contain the default extension")
}

func TestCompilerSetImportExt_extension_name_validation(t *testing.T) {
	c := new(gs.Compiler) // Instantiate a new compiler object with no initialization

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
		{[]string{".gs"}, []string{".gs"}, false, "well-formed extension should not return an error"},
		{[]string{""}, []string{".gs"}, true, "empty extension name should return an error"},
		{[]string{"foo"}, []string{".gs"}, true, "name without dot prefix should return an error"},
		{[]string{"foo.bar"}, []string{".gs"}, true, "malformed extension should return an error"},
		{[]string{"foo."}, []string{".gs"}, true, "malformed extension should return an error"},
		{[]string{".yb"}, []string{".yb"}, false, "name with dot prefix should be added"},
		{[]string{".foo", ".bar"}, []string{".foo", ".bar"}, false, "it should replace instead of appending"},
	} {
		err := c.SetImportFileExt(test.extensions...)
		if test.requireErr {
			require.Error(t, err, test.msgFail)
		}

		expect := test.expect
		actual := c.GetImportFileExt()
		require.Equal(t, expect, actual, test.msgFail)
	}
}

func concatInsts(instructions ...[]byte) []byte {
	var concat []byte
	for _, i := range instructions {
		concat = append(concat, i...)
	}
	return concat
}

func bytecode(instructions []byte, constants []core.Object) *vm.Bytecode {
	return &vm.Bytecode{
		FileSet:      parser.NewFileSet(),
		MainFunction: &value.CompiledFunction{Instructions: instructions},
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
	require.Equal(t, expected.MainFunction, actual.MainFunction)
	equalConstants(t, expected.Constants, actual.Constants)
}

func equalConstants(t *testing.T, expected, actual []core.Object) {
	require.Equal(t, len(expected), len(actual))
	for i := 0; i < len(expected); i++ {
		require.Equal(t, expected[i], actual[i])
	}
}

type compileTracer struct {
	Out []string
}

func (o *compileTracer) Write(p []byte) (n int, err error) {
	o.Out = append(o.Out, string(p))
	return len(p), nil
}

func traceCompile(input string, symbols map[string]core.Object) (res *vm.Bytecode, trace []string, err error) {
	fileSet := parser.NewFileSet()
	file := fileSet.AddFile("test", -1, len(input))

	p := parser.NewParser(file, []byte(input), nil)

	symTable := vm.NewSymbolTable()
	for name := range symbols {
		symTable.Define(name)
	}
	for idx, fn := range vm.BuiltinFuncs {
		symTable.DefineBuiltin(idx, fn.Name())
	}

	tr := &compileTracer{}
	c := gs.NewCompiler(file, symTable, nil, nil, tr)
	parsed, err := p.ParseFile()
	if err != nil {
		return
	}

	err = c.Compile(parsed)
	res = c.Bytecode()
	res.RemoveDuplicates()
	{
		trace = append(trace, fmt.Sprintf("Compiler Trace:\n%s",
			strings.Join(tr.Out, "")))
		trace = append(trace, fmt.Sprintf("Compiled Constants:\n%s",
			strings.Join(res.FormatConstants(), "\n")))
		trace = append(trace, fmt.Sprintf("Compiled Instructions:\n%s\n",
			strings.Join(res.FormatInstructions(), "\n")))
	}
	if err != nil {
		return
	}
	return
}

func objectsArray(o ...core.Object) []core.Object {
	return o
}

func intObject(v int64) *value.Int {
	return value.NewInt(v)
}

func stringObject(v string) *value.String {
	return value.NewString(v)
}

func compiledFunction(numLocals, numParams int, insts ...[]byte) *value.CompiledFunction {
	return &value.CompiledFunction{
		Instructions:  concatInsts(insts...),
		NumLocals:     numLocals,
		NumParameters: numParams,
	}
}
