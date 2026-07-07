package compiler_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/vm"
)

func static(vs ...any) core.Static {
	primitive := func(v core.Value) core.Primitive {
		if v.Type > value.LastPrimitiveType {
			panic(fmt.Errorf("expected primitive type, got: %v", v.Type))
		}
		return core.Primitive{Type: v.Type, Data: v.Data}
	}

	static := core.Static{}
	for _, v := range vs {
		switch v := v.(type) {
		case nil:
			static.Primitives = append(static.Primitives, primitive(core.Undefined))
		case bool:
			static.Primitives = append(static.Primitives, primitive(core.BoolValue(v)))
		case byte:
			static.Primitives = append(static.Primitives, primitive(core.ByteValue(v)))
		case rune:
			static.Primitives = append(static.Primitives, primitive(core.RuneValue(v)))
		case int:
			static.Primitives = append(static.Primitives, primitive(core.IntValue(int64(v))))
		case int64:
			static.Primitives = append(static.Primitives, primitive(core.IntValue(v)))
		case float64:
			static.Primitives = append(static.Primitives, primitive(core.FloatValue(v)))
		case dec128.Dec128:
			static.Decimals = append(static.Decimals, v)
		case string:
			static.Strings = append(static.Strings, v)
		case core.Runes:
			static.Runes = append(static.Runes, v)
		case core.Bytes:
			static.Bytes = append(static.Bytes, v)
		case time.Time:
			static.Times = append(static.Times, v)
		case core.FormatSpec:
			static.FormatSpecs = append(static.FormatSpecs, v)
		case core.CompiledFunction:
			static.CompiledFunctions = append(static.CompiledFunctions, v)
		case core.Value:
			static.Primitives = append(static.Primitives, primitive(v))
		default:
			panic(fmt.Sprintf("unsupported static type: %T", v))
		}
	}
	return static
}

func compiledFunction(numLocals int, numParams int, insts ...bc.Instruction) core.CompiledFunction {
	f := &core.CompiledFunction{}
	f.Set(concatInsts(insts...), nil, nil, numLocals, 0, numParams, 0, false)
	return *f
}

func concatInsts(instructions ...bc.Instruction) bc.Instructions {
	var concat bc.Instructions
	for _, i := range instructions {
		concat = append(concat, i)
	}
	return concat
}

func bytecode(instructions bc.Instructions, static core.Static) *vm.Bytecode {
	return &vm.Bytecode{
		FileSet: parser.NewFileSet(),
		MainFunction: &core.CompiledFunction{
			Instructions: instructions,
			MaxStack:     compiler.ComputeMaxStack(instructions),
		},
		Static: static,
	}
}

func countOpcode(inst bc.Instructions, target bc.Opcode) int {
	count := 0
	for _, i := range inst {
		if i.Op == target {
			count++
		}
	}
	return count
}

func hasAbortCheckBeforeBackwardJump(inst bc.Instructions) bool {
	for ip := 0; ip < len(inst); {
		op := inst[ip].Op
		if op == bc.Jump {
			target := int(inst[ip].Op3)
			if target < ip && ip > 0 && inst[ip-1].Op == bc.AbortCheck {
				return true
			}
		}
		ip += 1
	}
	return false
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
	require.True(t, strings.Contains(err.Error(), expected), "expected error string: %s, got: %s", expected, err.Error())
	ok = true
}

func expectCompile(t *testing.T, input string, expected *vm.Bytecode) {
	actual, trace, err := traceCompile(input, nil)
	require.NoError(t, err)

	if !equalBytecode(t, expected, actual) {
		t.Logf("Input:\n%s", input)
		for _, tr := range trace {
			t.Log(tr)
		}
		t.Logf("Expected Bytecode:\n%s", strings.Join(expected.MustFormatInstructions(), "\n"))
		t.Logf("Actual Bytecode:\n%s", strings.Join(actual.MustFormatInstructions(), "\n"))
		panic("bytecode mismatch")
	}
}

func equalBytecode(t *testing.T, expected, actual *vm.Bytecode) bool {
	if !expected.MainFunction.Instructions.Equal(actual.MainFunction.Instructions) {
		t.Logf("Instructions mismatch\nEXPECTED:\n%s\nACTUAL:\n%s", expected.MainFunction.Instructions.String(), actual.MainFunction.Instructions.String())
		return false
	}
	return equalStatic(t, expected.Static, actual.Static)
}

func equalStatic(t *testing.T, expected, actual core.Static) bool {
	s := expected
	other := actual

	if len(s.Primitives) != len(other.Primitives) {
		t.Logf("Primitives length mismatch: exp=%d, act=%d", len(s.Primitives), len(other.Primitives))
		return false
	}
	for i := range s.Primitives {
		if s.Primitives[i] != other.Primitives[i] {
			t.Logf("Primitive mismatch at index %d: exp=%v, act=%v", i, s.Primitives[i], other.Primitives[i])
			return false
		}
	}

	if len(s.Decimals) != len(other.Decimals) {
		t.Logf("Decimals length mismatch: exp=%d, act=%d", len(s.Decimals), len(other.Decimals))
		return false
	}
	for i := range s.Decimals {
		if !s.Decimals[i].Equal(other.Decimals[i]) {
			t.Logf("Decimal mismatch at index %d: exp=%v, act=%v", i, s.Decimals[i], other.Decimals[i])
			return false
		}
	}

	if len(s.Strings) != len(other.Strings) {
		t.Logf("Strings length mismatch: exp=%d, act=%d", len(s.Strings), len(other.Strings))
		return false
	}
	for i := range s.Strings {
		if s.Strings[i] != other.Strings[i] {
			t.Logf("String mismatch at index %d: exp=%s, act=%s", i, s.Strings[i], other.Strings[i])
			return false
		}
	}

	if len(s.Runes) != len(other.Runes) {
		t.Logf("Runes length mismatch: exp=%d, act=%d", len(s.Runes), len(other.Runes))
		return false
	}
	for i := range s.Runes {
		if len(s.Runes[i].Elements) != len(other.Runes[i].Elements) {
			t.Logf("Runes elements length mismatch at index %d: exp=%d, act=%d", i, len(s.Runes[i].Elements), len(other.Runes[i].Elements))
			return false
		}
		for j := range s.Runes[i].Elements {
			if s.Runes[i].Elements[j] != other.Runes[i].Elements[j] {
				t.Logf("Rune element mismatch at index %d, element %d: exp=%d, act=%d", i, j, s.Runes[i].Elements[j], other.Runes[i].Elements[j])
				return false
			}
		}
	}

	if len(s.Bytes) != len(other.Bytes) {
		t.Logf("Bytes length mismatch: exp=%d, act=%d", len(s.Bytes), len(other.Bytes))
		return false
	}
	for i := range s.Bytes {
		if len(s.Bytes[i].Elements) != len(other.Bytes[i].Elements) {
			t.Logf("Bytes elements length mismatch at index %d: exp=%d, act=%d", i, len(s.Bytes[i].Elements), len(other.Bytes[i].Elements))
			return false
		}
		for j := range s.Bytes[i].Elements {
			if s.Bytes[i].Elements[j] != other.Bytes[i].Elements[j] {
				t.Logf("Byte element mismatch at index %d, element %d: exp=%d, act=%d", i, j, s.Bytes[i].Elements[j], other.Bytes[i].Elements[j])
				return false
			}
		}
	}

	if len(s.Times) != len(other.Times) {
		t.Logf("Times length mismatch: exp=%d, act=%d", len(s.Times), len(other.Times))
		return false
	}
	for i := range s.Times {
		if !s.Times[i].Equal(other.Times[i]) {
			t.Logf("Time mismatch at index %d: exp=%v, act=%v", i, s.Times[i], other.Times[i])
			return false
		}
	}

	if len(s.FormatSpecs) != len(other.FormatSpecs) {
		t.Logf("FormatSpecs length mismatch: exp=%d, act=%d", len(s.FormatSpecs), len(other.FormatSpecs))
		return false
	}
	for i := range s.FormatSpecs {
		if !s.FormatSpecs[i].Equal(other.FormatSpecs[i]) {
			t.Logf("FormatSpec mismatch at index %d: exp=%v, act=%v", i, s.FormatSpecs[i], other.FormatSpecs[i])
			return false
		}
	}

	if len(s.CompiledFunctions) != len(other.CompiledFunctions) {
		t.Logf("CompiledFunctions length mismatch: exp=%d, act=%d", len(s.CompiledFunctions), len(other.CompiledFunctions))
		return false
	}
	for i := range s.CompiledFunctions {
		if !equalCompiledFunction(t, s.CompiledFunctions[i], other.CompiledFunctions[i]) {
			t.Logf("CompiledFunction mismatch at index %d", i)
			return false
		}
	}

	return true
}

func equalCompiledFunction(t *testing.T, expected, other core.CompiledFunction) bool {
	if !expected.Instructions.Equal(other.Instructions) {
		t.Logf("CompiledFunction instructions mismatch:\nEXPECTED:\n%s\n\nACTUAL:\n%s", expected.Instructions.String(), other.Instructions.String())
		return false
	}
	return true
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

	symTable := compiler.NewSymbolTable()
	for name := range symbols {
		symTable.Define(name)
	}
	for idx, name := range vm.BuiltinFunctionNames {
		symTable.DefineBuiltin(idx, name)
	}

	tr := &compileTracer{}
	c := compiler.NewCompiler(nil, nil, file, symTable, nil, nil, tr)
	c.SetAssignmentMode(mode)
	parsed, err := p.ParseFile()
	if err != nil {
		return
	}

	err = c.Compile(parsed)
	res = c.Bytecode()

	trace = append(trace, fmt.Sprintf("Compiler Trace:\n%s", strings.Join(tr.Out, "")))
	trace = append(trace, fmt.Sprintf("Compiled Constants:\n%s", strings.Join(res.MustFormatStatics(), "\n")))
	trace = append(trace, fmt.Sprintf("Compiled Instructions:\n%s\n", strings.Join(res.MustFormatInstructions(), "\n")))

	return
}

func TestCompiler_CompileBytesLiteral(t *testing.T) {
	expectCompile(t, `b"abc"`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticBytes(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(core.Bytes{Elements: []byte("abc")}),
		),
	)
}

func TestCompiler_CompileByteCharLiteral(t *testing.T) {
	expectCompile(t, `'A'`,
		bytecode(
			bc.Instructions{
				compiler.NewPushRune('A'),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(),
		),
	)

	expectCompile(t, `b'A'`,
		bytecode(
			bc.Instructions{
				compiler.NewPushByte(byte('A')),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(),
		),
	)
}

func TestCompiler_CompileTimeLiteral(t *testing.T) {
	v := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	expectCompile(t, `t"2024-01-01T00:00:00Z"`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticTime(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(v),
		),
	)
}

func TestCompiler_Compile(t *testing.T) {
	expectCompile(t, `1 + 2`,
		bytecode(
			bc.Instructions{
				compiler.NewPushInt(1),
				compiler.NewPushInt(2),
				compiler.NewBinaryOp(11),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static()))

	expectCompile(t, `1.0 + 2.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(11),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `1.0; 2.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewPop(),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `1.0 - 2.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(12),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `1.0 * 2.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(13),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `2.0 / 1.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(14),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				2.0,
				1.0)))

	expectCompile(t, `1.0 in 2.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewContains(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `1.0 not in 2.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewContains(),
				compiler.NewUnaryNot(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `true`,
		bytecode(
			bc.Instructions{
				compiler.NewPushBool(true),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static()))

	expectCompile(t, `false`,
		bytecode(
			bc.Instructions{
				compiler.NewPushBool(false),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static()))

	expectCompile(t, `1.0 > 2.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(39),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `1.0 < 2.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(38),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `1.0 >= 2.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(44),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `1.0 <= 2.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(43),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `1.0 == 2.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewEqual(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `1.0 != 2.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewNotEqual(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `true == false`,
		bytecode(
			bc.Instructions{
				compiler.NewPushBool(true),
				compiler.NewPushBool(false),
				compiler.NewEqual(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static()))

	expectCompile(t, `true != false`,
		bytecode(
			bc.Instructions{
				compiler.NewPushBool(true),
				compiler.NewPushBool(false),
				compiler.NewNotEqual(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static()))

	expectCompile(t, `-1.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewUnaryNeg(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0)))

	expectCompile(t, `!true`,
		bytecode(
			bc.Instructions{
				compiler.NewPushBool(true),
				compiler.NewUnaryNot(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static()))

	expectCompile(t, `if true { 10.0 }; 3333.0`,
		bytecode(
			bc.Instructions{
				compiler.NewPushBool(true),
				compiler.NewJumpFalsy(4),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewPop(),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				10.0,
				3333.0)))

	expectCompile(t, `if (true) { 10.0 } else { 20.0 }; 3333.0;`,
		bytecode(
			bc.Instructions{
				compiler.NewPushBool(true),
				compiler.NewJumpFalsy(5),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewPop(),
				compiler.NewJump(7),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewPop(),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				10.0,
				20.0,
				3333.0)))

	expectCompile(t, `"kami"`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticString(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				"kami")))

	expectCompile(t, `"ka" + "mi"`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticString(0),
				compiler.NewLoadStaticString(1),
				compiler.NewBinaryOp(11),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				"ka",
				"mi")))

	expectCompile(t, `var a`,
		bytecode(
			bc.Instructions{
				compiler.NewPushUndefined(),
				compiler.NewStoreGlobal(0),
				compiler.NewSuspend(),
			},
			static()))

	expectCompile(t, `var a = 1.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewStoreGlobal(0),
				compiler.NewSuspend(),
			},
			static(
				1.0)))

	expectCompile(t, `a := 1.0; b := 2.0; a += b`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewStoreGlobal(1),
				compiler.NewLoadGlobal(0),
				compiler.NewLoadGlobal(1),
				compiler.NewBinaryOp(11),
				compiler.NewStoreGlobal(0),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `a := 1.0; b := 2.0; a /= b`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewStoreGlobal(1),
				compiler.NewLoadGlobal(0),
				compiler.NewLoadGlobal(1),
				compiler.NewBinaryOp(14),
				compiler.NewStoreGlobal(0),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0)))

	expectCompile(t, `[]`,
		bytecode(
			bc.Instructions{
				compiler.NewMakeArray(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static()))

	expectCompile(t, `[1.0, 2.0, 3.0]`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewMakeArray(3),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				3.0)))

	expectCompile(t, `[1.0 + 2.0, 3.0 - 4.0, 5.0 * 6.0]`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(11),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewLoadStaticPrimitive(3),
				compiler.NewBinaryOp(12),
				compiler.NewLoadStaticPrimitive(4),
				compiler.NewLoadStaticPrimitive(5),
				compiler.NewBinaryOp(13),
				compiler.NewMakeArray(3),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				3.0,
				4.0,
				5.0,
				6.0)))

	expectCompile(t, `{}`,
		bytecode(
			bc.Instructions{
				compiler.NewMakeRecord(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static()))

	expectCompile(t, `{a: 2.0, b: 4.0, c: 6.0}`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticString(0),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticString(1),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewLoadStaticString(2),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewMakeRecord(6),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				"a",
				2.0,
				"b",
				4.0,
				"c",
				6.0)))

	expectCompile(t, `{a: 2.0 + 3.0, b: 5.0 * 6.0}`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticString(0),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(11),
				compiler.NewLoadStaticString(1),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewLoadStaticPrimitive(3),
				compiler.NewBinaryOp(13),
				compiler.NewMakeRecord(4),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				"a",
				2.0,
				3.0,
				"b",
				5.0,
				6.0)))

	expectCompile(t, `[1.0, 2.0, 3.0][1.0 + 1.0]`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewMakeArray(3),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewBinaryOp(11),
				compiler.NewAccessIndex(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				3.0)))

	expectCompile(t, `{a: 2.0}[2.0 - 1.0]`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticString(0),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewMakeRecord(2),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(12),
				compiler.NewAccessIndex(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				"a",
				2.0,
				1.0)))

	expectCompile(t, `[1.0, 2.0, 3.0][:]`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewMakeArray(3),
				compiler.NewPushUndefined(),
				compiler.NewPushUndefined(),
				compiler.NewSlice(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				3.0)))

	expectCompile(t, `[1.0, 2.0, 3.0][0.0 : 2.0]`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewMakeArray(3),
				compiler.NewLoadStaticPrimitive(3),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewSlice(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				3.0,
				0.0)))

	expectCompile(t, `[1.0, 2.0, 3.0][:2.0]`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewMakeArray(3),
				compiler.NewPushUndefined(),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewSlice(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				3.0)))

	expectCompile(t, `[1.0, 2.0, 3.0][0.0:]`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewMakeArray(3),
				compiler.NewLoadStaticPrimitive(3),
				compiler.NewPushUndefined(),
				compiler.NewSlice(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				3.0,
				0.0)))

	expectCompile(t, `[1.0, 2.0, 3.0][0.0:3.0:2.0]`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewMakeArray(3),
				compiler.NewLoadStaticPrimitive(3),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewSliceStep(),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				3.0,
				0.0)))

	expectCompile(t, `f1 := func(a) { return a }; f1([1.0, 2.0]...);`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadGlobal(0),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewMakeArray(2),
				compiler.NewCallFunction(1, true),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				compiledFunction(1, 1,
					compiler.NewLoadLocal(0),
					compiler.NewReturn(true)),
				1.0,
				2.0)))

	expectCompile(t, `func() { return 5.0 + 10.0 }`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				5.0,
				10.0,
				compiledFunction(0, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewLoadStaticPrimitive(1),
					compiler.NewBinaryOp(11),
					compiler.NewReturn(true)))))

	expectCompile(t, `func() { 5.0 + 10.0 }`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				5.0,
				10.0,
				compiledFunction(0, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewLoadStaticPrimitive(1),
					compiler.NewBinaryOp(11),
					compiler.NewPop(),
					compiler.NewReturn(false)))))

	expectCompile(t, `func() { 1.0; 2.0 }`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				compiledFunction(0, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewPop(),
					compiler.NewLoadStaticPrimitive(1),
					compiler.NewPop(),
					compiler.NewReturn(false)))))

	expectCompile(t, `func() { 1.0; return 2.0 }`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				compiledFunction(0, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewPop(),
					compiler.NewLoadStaticPrimitive(1),
					compiler.NewReturn(true)))))

	expectCompile(t, `func() { if(true) { return 1.0 } else { return 2.0 } }`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				compiledFunction(0, 0,
					compiler.NewPushBool(true),
					compiler.NewJumpFalsy(4),
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewReturn(true),
					compiler.NewLoadStaticPrimitive(1),
					compiler.NewReturn(true)))))

	expectCompile(t, `func() { 1.0; if(true) { 2.0 } else { 3.0 }; 4.0 }`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				3.0,
				4.0,
				compiledFunction(0, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewPop(),
					compiler.NewPushBool(true),
					compiler.NewJumpFalsy(7),
					compiler.NewLoadStaticPrimitive(1),
					compiler.NewPop(),
					compiler.NewJump(9),
					compiler.NewLoadStaticPrimitive(2),
					compiler.NewPop(),
					compiler.NewLoadStaticPrimitive(3),
					compiler.NewPop(),
					compiler.NewReturn(false)))))

	expectCompile(t, `func() { }`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				compiledFunction(0, 0,
					compiler.NewReturn(false)))))

	expectCompile(t, `func() { 24.0 }()`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewCallFunction(0, false),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				24.0,
				compiledFunction(0, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewPop(),
					compiler.NewReturn(false)))))

	expectCompile(t, `func() { return 24.0 }()`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewCallFunction(0, false),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				24.0,
				compiledFunction(0, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewReturn(true)))))

	expectCompile(t, `noArg := func() { 24.0 }; noArg();`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadGlobal(0),
				compiler.NewCallFunction(0, false),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				24.0,
				compiledFunction(0, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewPop(),
					compiler.NewReturn(false)))))

	expectCompile(t, `noArg := func() { return 24.0 }; noArg();`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadGlobal(0),
				compiler.NewCallFunction(0, false),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				24.0,
				compiledFunction(0, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewReturn(true)))))

	expectCompile(t, `n := 55.0; func() { n };`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				55.0,
				compiledFunction(0, 0,
					compiler.NewLoadGlobal(0),
					compiler.NewPop(),
					compiler.NewReturn(false)))))

	expectCompile(t, `func() { n := 55.0; return n }`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				55.0,
				compiledFunction(1, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewDefineLocal(0),
					compiler.NewLoadLocal(0),
					compiler.NewReturn(true)))))

	expectCompile(t, `func() { a := 55.0; b := 77.0; return a + b }`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				55.0,
				77.0,
				compiledFunction(2, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewDefineLocal(0),
					compiler.NewLoadStaticPrimitive(1),
					compiler.NewDefineLocal(1),
					compiler.NewLoadLocal(0),
					compiler.NewLoadLocal(1),
					compiler.NewBinaryOp(11),
					compiler.NewReturn(true)))))

	expectCompile(t, `f1 := func(a) { return a }; f1(24.0);`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadGlobal(0),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewCallFunction(1, false),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				compiledFunction(1, 1,
					compiler.NewLoadLocal(0),
					compiler.NewReturn(true)),
				24.0)))

	expectCompile(t, `varTest := func(...a) { return a }; varTest(1.0,2.0,3.0);`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadGlobal(0),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewCallFunction(3, false),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				compiledFunction(1, 1,
					compiler.NewLoadLocal(0),
					compiler.NewReturn(true)),
				1.0, 2.0, 3.0)))

	expectCompile(t, `f1 := func(a, b, c) { a; b; return c; }; f1(24.0, 25.0, 26.0);`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadGlobal(0),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewCallFunction(3, false),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				compiledFunction(3, 3,
					compiler.NewLoadLocal(0),
					compiler.NewPop(),
					compiler.NewLoadLocal(1),
					compiler.NewPop(),
					compiler.NewLoadLocal(2),
					compiler.NewReturn(true)),
				24.0,
				25.0,
				26.0)))

	expectCompile(t, `func() { n := 55.0; n = 23.0; return n }`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				55.0,
				23.0,
				compiledFunction(1, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewDefineLocal(0),
					compiler.NewLoadStaticPrimitive(1),
					compiler.NewStoreLocal(0),
					compiler.NewLoadLocal(0),
					compiler.NewReturn(true)))))
	expectCompile(t, `len([]);`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadBuiltinFunction(0),
				compiler.NewMakeArray(0),
				compiler.NewCallFunction(1, false),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static()))

	expectCompile(t, `func() { return len([]) }`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				compiledFunction(0, 0,
					compiler.NewLoadBuiltinFunction(0),
					compiler.NewMakeArray(0),
					compiler.NewCallFunction(1, false),
					compiler.NewReturn(true)))))

	expectCompile(t, `func(a) { func(b) { return a + b } }`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(1),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				compiledFunction(1, 1,
					compiler.NewLoadFree(0),
					compiler.NewLoadLocal(0),
					compiler.NewBinaryOp(11),
					compiler.NewReturn(true)),
				compiledFunction(1, 1,
					compiler.NewLoadLocalPtr(0),
					compiler.NewMakeClosure(0, 1),
					compiler.NewPop(),
					compiler.NewReturn(false)))))

	expectCompile(t, `
func(a) {
	return func(b) {
		return func(c) {
			return a + b + c
		}
	}
}`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(2),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				compiledFunction(1, 1,
					compiler.NewLoadFree(0),
					compiler.NewLoadFree(1),
					compiler.NewBinaryOp(11),
					compiler.NewLoadLocal(0),
					compiler.NewBinaryOp(11),
					compiler.NewReturn(true)),
				compiledFunction(1, 1,
					compiler.NewLoadFreePtr(0),
					compiler.NewLoadLocalPtr(0),
					compiler.NewMakeClosure(0, 2),
					compiler.NewReturn(true)),
				compiledFunction(1, 1,
					compiler.NewLoadLocalPtr(0),
					compiler.NewMakeClosure(1, 1),
					compiler.NewReturn(true)))))

	expectCompile(t, `
g := 55.0;

func() {
	a := 66.0;

	return func() {
		b := 77.0;

		return func() {
			c := 88.0;

			return g + a + b + c;
		}
	}
}`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadStaticCompiledFunction(2),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				55.0,
				66.0,
				77.0,
				88.0,
				compiledFunction(1, 0,
					compiler.NewLoadStaticPrimitive(3),
					compiler.NewDefineLocal(0),
					compiler.NewLoadGlobal(0),
					compiler.NewLoadFree(0),
					compiler.NewBinaryOp(11),
					compiler.NewLoadFree(1),
					compiler.NewBinaryOp(11),
					compiler.NewLoadLocal(0),
					compiler.NewBinaryOp(11),
					compiler.NewReturn(true)),
				compiledFunction(1, 0,
					compiler.NewLoadStaticPrimitive(2),
					compiler.NewDefineLocal(0),
					compiler.NewLoadFreePtr(0),
					compiler.NewLoadLocalPtr(0),
					compiler.NewMakeClosure(0, 2),
					compiler.NewReturn(true)),
				compiledFunction(1, 0,
					compiler.NewLoadStaticPrimitive(1),
					compiler.NewDefineLocal(0),
					compiler.NewLoadLocalPtr(0),
					compiler.NewMakeClosure(1, 1),
					compiler.NewReturn(true)))))

	expectCompile(t, `for i:=0.0; i<10.0; i++ {}`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadGlobal(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(38),
				compiler.NewJumpFalsy(12),
				compiler.NewLoadGlobal(0),
				compiler.NewPushInt(1),
				compiler.NewBinaryOp(11),
				compiler.NewStoreGlobal(0),
				compiler.NewAbortCheck(),
				compiler.NewJump(2),
				compiler.NewSuspend(),
			},
			static(
				0.0,
				10.0)))

	expectCompile(t, `for var i = 0.0; i<10.0; i++ {}`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadGlobal(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(38),
				compiler.NewJumpFalsy(12),
				compiler.NewLoadGlobal(0),
				compiler.NewPushInt(1),
				compiler.NewBinaryOp(11),
				compiler.NewStoreGlobal(0),
				compiler.NewAbortCheck(),
				compiler.NewJump(2),
				compiler.NewSuspend(),
			},
			static(
				0.0,
				10.0)))

	expectCompile(t, `m := {}; for k, v in m {}`,
		bytecode(
			bc.Instructions{
				compiler.NewMakeRecord(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadGlobal(0),
				compiler.NewIterInit(),
				compiler.NewStoreGlobal(1),
				compiler.NewLoadGlobal(1),
				compiler.NewIterNext(),
				compiler.NewJumpFalsy(16),
				compiler.NewLoadGlobal(1),
				compiler.NewIterKey(),
				compiler.NewStoreGlobal(2),
				compiler.NewLoadGlobal(1),
				compiler.NewIterValue(),
				compiler.NewStoreGlobal(3),
				compiler.NewAbortCheck(),
				compiler.NewJump(5),
				compiler.NewSuspend(),
			},
			static()))

	expectCompile(t, `a := 0.0; a == 0.0 && a != 1.0 || a < 1.0`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewStoreGlobal(0),
				compiler.NewLoadGlobal(0),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewEqual(),
				compiler.NewAndJump(9),
				compiler.NewLoadGlobal(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewNotEqual(),
				compiler.NewOrJump(13),
				compiler.NewLoadGlobal(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(38),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				0.0,
				1.0)))

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

func TestCompiler_AbortCheckEmission(t *testing.T) {
	loopRes, _, err := traceCompile(`for i := 0; i < 3; i++ {}`, nil)
	require.NoError(t, err)
	require.Equal(t, 1, countOpcode(loopRes.MainFunction.Instructions, bc.AbortCheck))
	require.True(t, hasAbortCheckBeforeBackwardJump(loopRes.MainFunction.Instructions))

	forInRes, _, err := traceCompile(`m := {}; for k, v in m {}`, nil)
	require.NoError(t, err)
	require.Equal(t, 1, countOpcode(forInRes.MainFunction.Instructions, bc.AbortCheck))
	require.True(t, hasAbortCheckBeforeBackwardJump(forInRes.MainFunction.Instructions))

	linearRes, _, err := traceCompile(`a := 1; b := 2; a + b`, nil)
	require.NoError(t, err)
	require.Equal(t, 0, countOpcode(linearRes.MainFunction.Instructions, bc.AbortCheck))
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
	a := 4.0
	return a

	b := 5.0 // dead code from here
	c := a
	return b
}`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				4.0,
				5.0,
				compiledFunction(0, 0,
					compiler.NewLoadStaticPrimitive(0),
					compiler.NewDefineLocal(0),
					compiler.NewLoadLocal(0),
					compiler.NewReturn(true)))))

	expectCompile(t, `
func() {
	if true {
		return 5.0
		a := 4.0  // dead code from here
		b := a
		return b
	} else {
		return 4.0
		c := 5.0  // dead code from here
		d := c
		return d
	}
}`, bytecode(
		bc.Instructions{
			compiler.NewLoadStaticCompiledFunction(0),
			compiler.NewPop(),
			compiler.NewSuspend(),
		},
		static(
			5.0,
			4.0,
			compiledFunction(0, 0,
				compiler.NewPushBool(true),
				compiler.NewJumpFalsy(4),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewReturn(true),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewReturn(true)))))

	expectCompile(t, `
func() {
	a := 1.0
	for {
		if a == 5.0 {
			return 10.0
		}
		5.0 + 5.0
		return 20.0
		b := a
		return b
	}
}`, bytecode(
		bc.Instructions{
			compiler.NewLoadStaticCompiledFunction(0),
			compiler.NewPop(),
			compiler.NewSuspend(),
		},
		static(
			1.0,
			5.0,
			10.0,
			20.0,
			compiledFunction(0, 0,
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewDefineLocal(0),
				compiler.NewLoadLocal(0),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewEqual(),
				compiler.NewJumpFalsy(8),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewReturn(true),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewBinaryOp(11),
				compiler.NewPop(),
				compiler.NewLoadStaticPrimitive(3),
				compiler.NewReturn(true)))))

	expectCompile(t, `
func() {
	if true {
		return 5.0
		a := 4.0  // dead code from here
		b := a
		return b
	} else {
		return 4.0
		c := 5.0  // dead code from here
		d := c
		return d
	}
}`, bytecode(
		bc.Instructions{
			compiler.NewLoadStaticCompiledFunction(0),
			compiler.NewPop(),
			compiler.NewSuspend(),
		},
		static(
			5.0,
			4.0,
			compiledFunction(0, 0,
				compiler.NewPushBool(true),
				compiler.NewJumpFalsy(4),
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewReturn(true),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewReturn(true)))))

	expectCompile(t, `
func() {
	if true {
		return
	}

    return

    return 123.0
}`, bytecode(
		bc.Instructions{
			compiler.NewLoadStaticCompiledFunction(0),
			compiler.NewPop(),
			compiler.NewSuspend(),
		},
		static(
			123.0,
			compiledFunction(0, 0,
				compiler.NewPushBool(true),
				compiler.NewJumpFalsy(3),
				compiler.NewReturn(false),
				compiler.NewReturn(false)))))

	expectCompile(t, `
func() {
	return
	if true {
		return 1.0
	} else {
		return 2.0
	}
}`,
		bytecode(
			bc.Instructions{
				compiler.NewLoadStaticCompiledFunction(0),
				compiler.NewPop(),
				compiler.NewSuspend(),
			},
			static(
				1.0,
				2.0,
				compiledFunction(0, 0,
					compiler.NewReturn(false)))))
}

func TestCompilerScopes(t *testing.T) {
	expectCompile(t, `
if a := 1.0; a {
    a = 2.0
	b := a
} else {
    a = 3.0
	b := a
}`, bytecode(
		bc.Instructions{
			compiler.NewLoadStaticPrimitive(0),
			compiler.NewStoreGlobal(0),
			compiler.NewLoadGlobal(0),
			compiler.NewJumpFalsy(9),
			compiler.NewLoadStaticPrimitive(1),
			compiler.NewStoreGlobal(0),
			compiler.NewLoadGlobal(0),
			compiler.NewStoreGlobal(1),
			compiler.NewJump(13),
			compiler.NewLoadStaticPrimitive(2),
			compiler.NewStoreGlobal(0),
			compiler.NewLoadGlobal(0),
			compiler.NewStoreGlobal(2),
			compiler.NewSuspend(),
		},
		static(
			1.0,
			2.0,
			3.0)))

	expectCompile(t, `
if var a = 1.0; a {
    a = 2.0
	b := a
} else {
    a = 3.0
	b := a
}`, bytecode(
		bc.Instructions{
			compiler.NewLoadStaticPrimitive(0),
			compiler.NewStoreGlobal(0),
			compiler.NewLoadGlobal(0),
			compiler.NewJumpFalsy(9),
			compiler.NewLoadStaticPrimitive(1),
			compiler.NewStoreGlobal(0),
			compiler.NewLoadGlobal(0),
			compiler.NewStoreGlobal(1),
			compiler.NewJump(13),
			compiler.NewLoadStaticPrimitive(2),
			compiler.NewStoreGlobal(0),
			compiler.NewLoadGlobal(0),
			compiler.NewStoreGlobal(2),
			compiler.NewSuspend(),
		},
		static(
			1.0,
			2.0,
			3.0)))

	expectCompile(t, `
func() {
	if a := 1.0; a {
    	a = 2.0
		b := a
	} else {
    	a = 3.0
		b := a
	}
}`, bytecode(
		bc.Instructions{
			compiler.NewLoadStaticCompiledFunction(0),
			compiler.NewPop(),
			compiler.NewSuspend(),
		},
		static(
			1.0,
			2.0,
			3.0,
			compiledFunction(0, 0,
				compiler.NewLoadStaticPrimitive(0),
				compiler.NewDefineLocal(0),
				compiler.NewLoadLocal(0),
				compiler.NewJumpFalsy(9),
				compiler.NewLoadStaticPrimitive(1),
				compiler.NewStoreLocal(0),
				compiler.NewLoadLocal(0),
				compiler.NewDefineLocal(1),
				compiler.NewJump(13),
				compiler.NewLoadStaticPrimitive(2),
				compiler.NewStoreLocal(0),
				compiler.NewLoadLocal(0),
				compiler.NewDefineLocal(1),
				compiler.NewReturn(false)))))
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

	c := compiler.NewCompiler(nil, nil, srcFile, nil, nil, nil, nil)
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

	c := compiler.NewCompiler(nil, nil, file, nil, nil, nil, nil)
	c.EnableFileImport(true)

	require.Equal(t, []string{".kvn"}, c.GetImportFileExt(), "newly created compiler object must contain the default extension")
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
		require.Equal(t, expect, actual, test.msgFail)
	}
}
