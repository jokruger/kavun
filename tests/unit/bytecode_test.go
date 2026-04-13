package unit

import (
	"bytes"
	"testing"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/tests/require"
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
		core.Undefined,
	)))
}

func TestBytecodeConstBool(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		core.True,
		core.False,
	)))
}

func TestBytecodeConstChar(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		core.CharValue('a'),
		core.CharValue('b'),
		core.CharValue('c'),
	)))
}

func TestBytecodeConstInt(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		core.IntValue(1),
		core.IntValue(2),
		core.IntValue(3),
		core.IntValue(1234567890),
	)))
}

func TestBytecodeConstFloat(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		core.FloatValue(0.123),
		core.FloatValue(123456.789),
	)))
}

func TestBytecodeConstString(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		alloc.NewStringValue(""),
		alloc.NewStringValue("foo"),
		alloc.NewStringValue("foo bar"),
	)))
}

func TestBytecodeConstBytes(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		alloc.NewBytesValue([]byte{}),
		alloc.NewBytesValue([]byte{1, 2, 3}),
		alloc.NewBytesValue([]byte("foo bar")),
	)))
}

func TestBytecodeConstTime(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		alloc.NewTimeValue(time.Unix(0, 0)),
		alloc.NewTimeValue(time.Unix(1234567890, 123456789)),
	)))
}

func TestBytecodeConstArray(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		alloc.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.FloatValue(2.0),
			core.CharValue('3'),
			alloc.NewStringValue("four"),
		}, true),
	)))

	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		alloc.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.FloatValue(2.0),
			core.CharValue('3'),
			alloc.NewStringValue("four"),
		}, false),
	)))
}

func TestBytecodeConstMap(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		alloc.NewRecordValue(map[string]core.Value{
			"a": core.IntValue(1),
			"b": core.FloatValue(2.0),
			"c": core.CharValue('3'),
			"d": alloc.NewStringValue("four"),
		}, true),
	)))

	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		alloc.NewRecordValue(map[string]core.Value{
			"a": core.IntValue(1),
			"b": core.FloatValue(2.0),
			"c": core.CharValue('3'),
			"d": alloc.NewStringValue("four"),
		}, false),
	)))
}

func TestBytecodeConstError(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		alloc.NewErrorValue(alloc.NewStringValue("some error")),
	)))
}

func TestBytecode(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray()))

	testBytecodeSerialization(t, bytecode(
		concatInsts(), objectsArray(
			core.CharValue('y'),
			core.FloatValue(93.11),
			compiledFunction(1, 0,
				vm.MakeInstruction(core.OpConstant, 3),
				vm.MakeInstruction(core.OpSetLocal, 0),
				vm.MakeInstruction(core.OpGetGlobal, 0),
				vm.MakeInstruction(core.OpGetFree, 0)),
			core.FloatValue(39.2),
			core.IntValue(192),
			alloc.NewStringValue("bar"),
		)))

	testBytecodeSerialization(t, bytecodeFileSet(
		concatInsts(
			vm.MakeInstruction(core.OpConstant, 0),
			vm.MakeInstruction(core.OpSetGlobal, 0),
			vm.MakeInstruction(core.OpConstant, 6),
			vm.MakeInstruction(core.OpPop)),
		objectsArray(
			core.IntValue(55),
			core.IntValue(66),
			core.IntValue(77),
			core.IntValue(88),
			alloc.NewRecordValue(map[string]core.Value{
				"array": alloc.NewArrayValue([]core.Value{
					core.IntValue(1),
					core.IntValue(2),
					core.IntValue(3),
					core.True,
					core.False,
					core.Undefined,
				}, true),
				"true":  core.True,
				"false": core.False,
				"bytes": alloc.NewBytesValue(make([]byte, 16)),
				"char":  core.CharValue('Y'),
				"error": alloc.NewErrorValue(alloc.NewStringValue("some error")),
				"float": core.FloatValue(-19.84),
				"immutable_array": alloc.NewArrayValue([]core.Value{
					core.IntValue(1),
					core.IntValue(2),
					core.IntValue(3),
					core.True,
					core.False,
					core.Undefined,
				}, true),
				"immutable_map": alloc.NewRecordValue(map[string]core.Value{
					"a": core.IntValue(1),
					"b": core.IntValue(2),
					"c": core.IntValue(3),
					"d": core.True,
					"e": core.False,
					"f": core.Undefined,
				}, true),
				"int": core.IntValue(91),
				"map": alloc.NewRecordValue(map[string]core.Value{
					"a": core.IntValue(1),
					"b": core.IntValue(2),
					"c": core.IntValue(3),
					"d": core.True,
					"e": core.False,
					"f": core.Undefined,
				}, false),
				"string":    alloc.NewStringValue("foo bar"),
				"time":      alloc.NewTimeValue(time.Now()),
				"undefined": core.Undefined,
			}, true),
			compiledFunction(1, 0,
				vm.MakeInstruction(core.OpConstant, 3),
				vm.MakeInstruction(core.OpSetLocal, 0),
				vm.MakeInstruction(core.OpGetGlobal, 0),
				vm.MakeInstruction(core.OpGetFree, 0),
				vm.MakeInstruction(core.OpBinaryOp, 11),
				vm.MakeInstruction(core.OpGetFree, 1),
				vm.MakeInstruction(core.OpBinaryOp, 11),
				vm.MakeInstruction(core.OpGetLocal, 0),
				vm.MakeInstruction(core.OpBinaryOp, 11),
				vm.MakeInstruction(core.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MakeInstruction(core.OpConstant, 2),
				vm.MakeInstruction(core.OpSetLocal, 0),
				vm.MakeInstruction(core.OpGetFree, 0),
				vm.MakeInstruction(core.OpGetLocal, 0),
				vm.MakeInstruction(core.OpClosure, 4, 2),
				vm.MakeInstruction(core.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MakeInstruction(core.OpConstant, 1),
				vm.MakeInstruction(core.OpSetLocal, 0),
				vm.MakeInstruction(core.OpGetLocal, 0),
				vm.MakeInstruction(core.OpClosure, 5, 1),
				vm.MakeInstruction(core.OpReturn, 1))),
		fileSet(srcfile{name: "file1", size: 100},
			srcfile{name: "file2", size: 200})))
}

func TestBytecode_RemoveDuplicates(t *testing.T) {
	testBytecodeRemoveDuplicates(t,
		bytecode(
			concatInsts(), objectsArray(
				core.CharValue('y'),
				core.FloatValue(93.11),
				compiledFunction(1, 0,
					vm.MakeInstruction(core.OpConstant, 3),
					vm.MakeInstruction(core.OpSetLocal, 0),
					vm.MakeInstruction(core.OpGetGlobal, 0),
					vm.MakeInstruction(core.OpGetFree, 0)),
				core.FloatValue(39.2),
				core.IntValue(192),
				alloc.NewStringValue("bar"))),
		bytecode(
			concatInsts(), objectsArray(
				core.CharValue('y'),
				core.FloatValue(93.11),
				compiledFunction(1, 0,
					vm.MakeInstruction(core.OpConstant, 3),
					vm.MakeInstruction(core.OpSetLocal, 0),
					vm.MakeInstruction(core.OpGetGlobal, 0),
					vm.MakeInstruction(core.OpGetFree, 0)),
				core.FloatValue(39.2),
				core.IntValue(192),
				alloc.NewStringValue("bar"))))

	testBytecodeRemoveDuplicates(t,
		bytecode(
			concatInsts(
				vm.MakeInstruction(core.OpConstant, 0),
				vm.MakeInstruction(core.OpConstant, 1),
				vm.MakeInstruction(core.OpConstant, 2),
				vm.MakeInstruction(core.OpConstant, 3),
				vm.MakeInstruction(core.OpConstant, 4),
				vm.MakeInstruction(core.OpConstant, 5),
				vm.MakeInstruction(core.OpConstant, 6),
				vm.MakeInstruction(core.OpConstant, 7),
				vm.MakeInstruction(core.OpConstant, 8),
				vm.MakeInstruction(core.OpClosure, 4, 1)),
			objectsArray(
				core.IntValue(1),
				core.FloatValue(2.0),
				core.CharValue('3'),
				alloc.NewStringValue("four"),
				compiledFunction(1, 0,
					vm.MakeInstruction(core.OpConstant, 3),
					vm.MakeInstruction(core.OpConstant, 7),
					vm.MakeInstruction(core.OpSetLocal, 0),
					vm.MakeInstruction(core.OpGetGlobal, 0),
					vm.MakeInstruction(core.OpGetFree, 0)),
				core.IntValue(1),
				core.FloatValue(2.0),
				core.CharValue('3'),
				alloc.NewStringValue("four"))),
		bytecode(
			concatInsts(
				vm.MakeInstruction(core.OpConstant, 0),
				vm.MakeInstruction(core.OpConstant, 1),
				vm.MakeInstruction(core.OpConstant, 2),
				vm.MakeInstruction(core.OpConstant, 3),
				vm.MakeInstruction(core.OpConstant, 4),
				vm.MakeInstruction(core.OpConstant, 0),
				vm.MakeInstruction(core.OpConstant, 1),
				vm.MakeInstruction(core.OpConstant, 2),
				vm.MakeInstruction(core.OpConstant, 3),
				vm.MakeInstruction(core.OpClosure, 4, 1)),
			objectsArray(
				core.IntValue(1),
				core.FloatValue(2.0),
				core.CharValue('3'),
				alloc.NewStringValue("four"),
				compiledFunction(1, 0,
					vm.MakeInstruction(core.OpConstant, 3),
					vm.MakeInstruction(core.OpConstant, 2),
					vm.MakeInstruction(core.OpSetLocal, 0),
					vm.MakeInstruction(core.OpGetGlobal, 0),
					vm.MakeInstruction(core.OpGetFree, 0)))))

	testBytecodeRemoveDuplicates(t,
		bytecode(
			concatInsts(
				vm.MakeInstruction(core.OpConstant, 0),
				vm.MakeInstruction(core.OpConstant, 1),
				vm.MakeInstruction(core.OpConstant, 2),
				vm.MakeInstruction(core.OpConstant, 3),
				vm.MakeInstruction(core.OpConstant, 4)),
			objectsArray(
				core.IntValue(1),
				core.IntValue(2),
				core.IntValue(3),
				core.IntValue(1),
				core.IntValue(3))),
		bytecode(
			concatInsts(
				vm.MakeInstruction(core.OpConstant, 0),
				vm.MakeInstruction(core.OpConstant, 1),
				vm.MakeInstruction(core.OpConstant, 2),
				vm.MakeInstruction(core.OpConstant, 0),
				vm.MakeInstruction(core.OpConstant, 2)),
			objectsArray(
				core.IntValue(1),
				core.IntValue(2),
				core.IntValue(3))))
}

func TestBytecode_CountObjects(t *testing.T) {
	b := bytecode(
		concatInsts(),
		objectsArray(
			core.IntValue(55),
			core.IntValue(66),
			core.IntValue(77),
			core.IntValue(88),
			compiledFunction(1, 0,
				vm.MakeInstruction(core.OpConstant, 3),
				vm.MakeInstruction(core.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MakeInstruction(core.OpConstant, 2),
				vm.MakeInstruction(core.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MakeInstruction(core.OpConstant, 1),
				vm.MakeInstruction(core.OpReturn, 1))))
	require.Equal(t, 7, b.CountObjects())
}

func fileSet(files ...srcfile) *parser.SourceFileSet {
	fileSet := parser.NewFileSet()
	for _, f := range files {
		fileSet.AddFile(f.name, -1, f.size)
	}
	return fileSet
}

func bytecodeFileSet(instructions []byte, constants []core.Value, fileSet *parser.SourceFileSet) *vm.Bytecode {
	return &vm.Bytecode{
		FileSet:      fileSet,
		MainFunction: &core.CompiledFunction{Instructions: instructions},
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
	err = r.Decode(alloc, bytes.NewReader(buf.Bytes()), nil)
	require.NoError(t, err)

	require.Equal(t, b.FileSet, r.FileSet)
	require.Equal(t, b.MainFunction, r.MainFunction)
	require.Equal(t, b.Constants, r.Constants)
}
