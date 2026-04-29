package unit

import (
	"bytes"
	"testing"
	"time"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/tests/require"
	"github.com/jokruger/kavun/vm"
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
		core.RuneValue('a'),
		core.RuneValue('b'),
		core.RuneValue('c'),
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
		core.NewStringValue(""),
		core.NewStringValue("foo"),
		core.NewStringValue("foo bar"),
	)))
}

func TestBytecodeConstBytes(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		core.NewBytesValue([]byte{}),
		core.NewBytesValue([]byte{1, 2, 3}),
		core.NewBytesValue([]byte("foo bar")),
	)))
}

func TestBytecodeConstTime(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		core.NewTimeValue(time.Unix(0, 0)),
		core.NewTimeValue(time.Unix(1234567890, 123456789)),
	)))
}

func TestBytecodeConstArray(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		core.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.FloatValue(2.0),
			core.RuneValue('3'),
			core.NewStringValue("four"),
		}, true),
	)))

	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		core.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.FloatValue(2.0),
			core.RuneValue('3'),
			core.NewStringValue("four"),
		}, false),
	)))
}

func TestBytecodeConstDict(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		core.NewRecordValue(map[string]core.Value{
			"a": core.IntValue(1),
			"b": core.FloatValue(2.0),
			"c": core.RuneValue('3'),
			"d": core.NewStringValue("four"),
		}, true),
	)))

	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		core.NewRecordValue(map[string]core.Value{
			"a": core.IntValue(1),
			"b": core.FloatValue(2.0),
			"c": core.RuneValue('3'),
			"d": core.NewStringValue("four"),
		}, false),
	)))
}

func TestBytecodeConstError(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		core.NewErrorValue(core.NewStringValue("some error")),
	)))
}

func TestBytecode(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray()))

	testBytecodeSerialization(t, bytecode(
		concatInsts(), objectsArray(
			core.RuneValue('y'),
			core.FloatValue(93.11),
			compiledFunction(1, 0,
				vm.MakeInstruction(core.OpConstant, 3),
				vm.MakeInstruction(core.OpSetLocal, 0),
				vm.MakeInstruction(core.OpGetGlobal, 0),
				vm.MakeInstruction(core.OpGetFree, 0)),
			core.FloatValue(39.2),
			core.IntValue(192),
			core.NewStringValue("bar"),
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
			core.NewRecordValue(map[string]core.Value{
				"array": core.NewArrayValue([]core.Value{
					core.IntValue(1),
					core.IntValue(2),
					core.IntValue(3),
					core.True,
					core.False,
					core.Undefined,
				}, true),
				"true":  core.True,
				"false": core.False,
				"bytes": core.NewBytesValue(make([]byte, 16)),
				"char":  core.RuneValue('Y'),
				"error": core.NewErrorValue(core.NewStringValue("some error")),
				"float": core.FloatValue(-19.84),
				"immutable_array": core.NewArrayValue([]core.Value{
					core.IntValue(1),
					core.IntValue(2),
					core.IntValue(3),
					core.True,
					core.False,
					core.Undefined,
				}, true),
				"immutable_dict": core.NewRecordValue(map[string]core.Value{
					"a": core.IntValue(1),
					"b": core.IntValue(2),
					"c": core.IntValue(3),
					"d": core.True,
					"e": core.False,
					"f": core.Undefined,
				}, true),
				"int": core.IntValue(91),
				"dict": core.NewRecordValue(map[string]core.Value{
					"a": core.IntValue(1),
					"b": core.IntValue(2),
					"c": core.IntValue(3),
					"d": core.True,
					"e": core.False,
					"f": core.Undefined,
				}, false),
				"string":    core.NewStringValue("foo bar"),
				"time":      core.NewTimeValue(time.Now()),
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
				core.RuneValue('y'),
				core.FloatValue(93.11),
				compiledFunction(1, 0,
					vm.MakeInstruction(core.OpConstant, 3),
					vm.MakeInstruction(core.OpSetLocal, 0),
					vm.MakeInstruction(core.OpGetGlobal, 0),
					vm.MakeInstruction(core.OpGetFree, 0)),
				core.FloatValue(39.2),
				core.IntValue(192),
				core.NewStringValue("bar"))),
		bytecode(
			concatInsts(), objectsArray(
				core.RuneValue('y'),
				core.FloatValue(93.11),
				compiledFunction(1, 0,
					vm.MakeInstruction(core.OpConstant, 3),
					vm.MakeInstruction(core.OpSetLocal, 0),
					vm.MakeInstruction(core.OpGetGlobal, 0),
					vm.MakeInstruction(core.OpGetFree, 0)),
				core.FloatValue(39.2),
				core.IntValue(192),
				core.NewStringValue("bar"))))

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
				core.RuneValue('3'),
				core.NewStringValue("four"),
				compiledFunction(1, 0,
					vm.MakeInstruction(core.OpConstant, 3),
					vm.MakeInstruction(core.OpConstant, 7),
					vm.MakeInstruction(core.OpSetLocal, 0),
					vm.MakeInstruction(core.OpGetGlobal, 0),
					vm.MakeInstruction(core.OpGetFree, 0)),
				core.IntValue(1),
				core.FloatValue(2.0),
				core.RuneValue('3'),
				core.NewStringValue("four"))),
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
				core.RuneValue('3'),
				core.NewStringValue("four"),
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
	err = r.Decode(core.NewArena(nil), bytes.NewReader(buf.Bytes()), nil)
	require.NoError(t, err)

	require.Equal(t, b.FileSet, r.FileSet)
	require.Equal(t, b.MainFunction, r.MainFunction)
	require.Equal(t, b.Constants, r.Constants)
}
