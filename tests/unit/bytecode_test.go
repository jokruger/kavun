package gs_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/tests/require"
	"github.com/jokruger/gs/value"
	"github.com/jokruger/gs/vm"
)

type srcfile struct {
	name string
	size int
}

func TestBytecodeEmpty(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray()))
}

func TestBytecodeConstUndefined(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.UndefinedValue,
	)))
}

func TestBytecodeConstBool(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.TrueValue,
		value.FalseValue,
	)))
}

func TestBytecodeConstChar(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.NewChar('a'),
		value.NewChar('b'),
		value.NewChar('c'),
	)))
}

func TestBytecodeConstInt(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.NewInt(1),
		value.NewInt(2),
		value.NewInt(3),
		value.NewInt(1234567890),
	)))
}

func TestBytecodeConstFloat(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.NewFloat(0.123),
		value.NewFloat(123456.789),
	)))
}

func TestBytecodeConstString(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.NewString(""),
		value.NewString("foo"),
		value.NewString("foo bar"),
	)))
}

func TestBytecodeConstBytes(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.NewBytes([]byte{}),
		value.NewBytes([]byte{1, 2, 3}),
		value.NewBytes([]byte("foo bar")),
	)))
}

func TestBytecodeConstTime(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.NewTime(time.Unix(0, 0)),
		value.NewTime(time.Unix(1234567890, 123456789)),
	)))
}

func TestBytecodeConstArray(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.NewArray([]core.Object{
			value.NewInt(1),
			value.NewFloat(2.0),
			value.NewChar('3'),
			value.NewString("four"),
		}, true),
	)))

	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.NewArray([]core.Object{
			value.NewInt(1),
			value.NewFloat(2.0),
			value.NewChar('3'),
			value.NewString("four"),
		}, false),
	)))
}

func TestBytecodeConstMap(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.NewMap(map[string]core.Object{
			"a": value.NewInt(1),
			"b": value.NewFloat(2.0),
			"c": value.NewChar('3'),
			"d": value.NewString("four"),
		}, true),
	)))

	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.NewMap(map[string]core.Object{
			"a": value.NewInt(1),
			"b": value.NewFloat(2.0),
			"c": value.NewChar('3'),
			"d": value.NewString("four"),
		}, false),
	)))
}

func TestBytecodeConstError(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		value.NewError(value.NewString("some error")),
	)))
}

func TestBytecode(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray()))

	testBytecodeSerialization(t, bytecode(
		concatInsts(), objectsArray(
			value.NewChar('y'),
			value.NewFloat(93.11),
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpSetLocal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpGetFree, 0)),
			value.NewFloat(39.2),
			value.NewInt(192),
			value.NewString("bar"),
		)))

	testBytecodeSerialization(t, bytecodeFileSet(
		concatInsts(
			vm.MakeInstruction(parser.OpConstant, 0),
			vm.MakeInstruction(parser.OpSetGlobal, 0),
			vm.MakeInstruction(parser.OpConstant, 6),
			vm.MakeInstruction(parser.OpPop)),
		objectsArray(
			value.NewInt(55),
			value.NewInt(66),
			value.NewInt(77),
			value.NewInt(88),
			value.NewMap(map[string]core.Object{
				"array": value.NewArray([]core.Object{
					value.NewInt(1),
					value.NewInt(2),
					value.NewInt(3),
					value.TrueValue,
					value.FalseValue,
					value.UndefinedValue,
				}, true),
				"true":  value.TrueValue,
				"false": value.FalseValue,
				"bytes": value.NewBytes(make([]byte, 16)),
				"char":  value.NewChar('Y'),
				"error": value.NewError(value.NewString("some error")),
				"float": value.NewFloat(-19.84),
				"immutable_array": value.NewArray([]core.Object{
					value.NewInt(1),
					value.NewInt(2),
					value.NewInt(3),
					value.TrueValue,
					value.FalseValue,
					value.UndefinedValue,
				}, true),
				"immutable_map": value.NewMap(map[string]core.Object{
					"a": value.NewInt(1),
					"b": value.NewInt(2),
					"c": value.NewInt(3),
					"d": value.TrueValue,
					"e": value.FalseValue,
					"f": value.UndefinedValue,
				}, true),
				"int": value.NewInt(91),
				"map": value.NewMap(map[string]core.Object{
					"a": value.NewInt(1),
					"b": value.NewInt(2),
					"c": value.NewInt(3),
					"d": value.TrueValue,
					"e": value.FalseValue,
					"f": value.UndefinedValue,
				}, false),
				"string":    value.NewString("foo bar"),
				"time":      value.NewTime(time.Now()),
				"undefined": value.UndefinedValue,
			}, true),
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpSetLocal, 0),
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
				vm.MakeInstruction(parser.OpSetLocal, 0),
				vm.MakeInstruction(parser.OpGetFree, 0),
				vm.MakeInstruction(parser.OpGetLocal, 0),
				vm.MakeInstruction(parser.OpClosure, 4, 2),
				vm.MakeInstruction(parser.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpSetLocal, 0),
				vm.MakeInstruction(parser.OpGetLocal, 0),
				vm.MakeInstruction(parser.OpClosure, 5, 1),
				vm.MakeInstruction(parser.OpReturn, 1))),
		fileSet(srcfile{name: "file1", size: 100},
			srcfile{name: "file2", size: 200})))
}

func TestBytecode_RemoveDuplicates(t *testing.T) {
	testBytecodeRemoveDuplicates(t,
		bytecode(
			concatInsts(), objectsArray(
				value.NewChar('y'),
				value.NewFloat(93.11),
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 3),
					vm.MakeInstruction(parser.OpSetLocal, 0),
					vm.MakeInstruction(parser.OpGetGlobal, 0),
					vm.MakeInstruction(parser.OpGetFree, 0)),
				value.NewFloat(39.2),
				value.NewInt(192),
				value.NewString("bar"))),
		bytecode(
			concatInsts(), objectsArray(
				value.NewChar('y'),
				value.NewFloat(93.11),
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 3),
					vm.MakeInstruction(parser.OpSetLocal, 0),
					vm.MakeInstruction(parser.OpGetGlobal, 0),
					vm.MakeInstruction(parser.OpGetFree, 0)),
				value.NewFloat(39.2),
				value.NewInt(192),
				value.NewString("bar"))))

	testBytecodeRemoveDuplicates(t,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpConstant, 4),
				vm.MakeInstruction(parser.OpConstant, 5),
				vm.MakeInstruction(parser.OpConstant, 6),
				vm.MakeInstruction(parser.OpConstant, 7),
				vm.MakeInstruction(parser.OpConstant, 8),
				vm.MakeInstruction(parser.OpClosure, 4, 1)),
			objectsArray(
				value.NewInt(1),
				value.NewFloat(2.0),
				value.NewChar('3'),
				value.NewString("four"),
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 3),
					vm.MakeInstruction(parser.OpConstant, 7),
					vm.MakeInstruction(parser.OpSetLocal, 0),
					vm.MakeInstruction(parser.OpGetGlobal, 0),
					vm.MakeInstruction(parser.OpGetFree, 0)),
				value.NewInt(1),
				value.NewFloat(2.0),
				value.NewChar('3'),
				value.NewString("four"))),
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpConstant, 4),
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpClosure, 4, 1)),
			objectsArray(
				value.NewInt(1),
				value.NewFloat(2.0),
				value.NewChar('3'),
				value.NewString("four"),
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 3),
					vm.MakeInstruction(parser.OpConstant, 2),
					vm.MakeInstruction(parser.OpSetLocal, 0),
					vm.MakeInstruction(parser.OpGetGlobal, 0),
					vm.MakeInstruction(parser.OpGetFree, 0)))))

	testBytecodeRemoveDuplicates(t,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpConstant, 4)),
			objectsArray(
				value.NewInt(1),
				value.NewInt(2),
				value.NewInt(3),
				value.NewInt(1),
				value.NewInt(3))),
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 2)),
			objectsArray(
				value.NewInt(1),
				value.NewInt(2),
				value.NewInt(3))))
}

func TestBytecode_CountObjects(t *testing.T) {
	b := bytecode(
		concatInsts(),
		objectsArray(
			value.NewInt(55),
			value.NewInt(66),
			value.NewInt(77),
			value.NewInt(88),
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpReturn, 1))))
	require.Equal(t, 7, b.CountObjects())
}

func fileSet(files ...srcfile) *parser.SourceFileSet {
	fileSet := parser.NewFileSet()
	for _, f := range files {
		fileSet.AddFile(f.name, -1, f.size)
	}
	return fileSet
}

func bytecodeFileSet(instructions []byte, constants []core.Object, fileSet *parser.SourceFileSet) *vm.Bytecode {
	return &vm.Bytecode{
		FileSet:      fileSet,
		MainFunction: &vm.CompiledFunction{Instructions: instructions},
		Constants:    constants,
	}
}

func testBytecodeRemoveDuplicates(t *testing.T, input, expected *vm.Bytecode) {
	input.RemoveDuplicates()

	require.Equal(t, expected.FileSet, input.FileSet)
	require.Equal(t, expected.MainFunction, input.MainFunction)
	require.Equal(t, expected.Constants, input.Constants)
}

func testBytecodeSerialization(t *testing.T, b *vm.Bytecode) {
	var buf bytes.Buffer
	err := b.Encode(&buf)
	require.NoError(t, err)

	r := &vm.Bytecode{}
	err = r.Decode(bytes.NewReader(buf.Bytes()), nil)
	require.NoError(t, err)

	require.Equal(t, b.FileSet, r.FileSet)
	require.Equal(t, b.MainFunction, r.MainFunction)
	require.Equal(t, b.Constants, r.Constants)
}
