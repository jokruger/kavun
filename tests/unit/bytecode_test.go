package unit

import (
	"bytes"
	"testing"
	"time"

	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/tests/require"
	"github.com/jokruger/kavun/vm"
)

func fspecParse(s string) (fspec.FormatSpec, error) {
	return fspec.Parse(s)
}

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

func TestBytecodeConstFormatSpec(t *testing.T) {
	mk := func(text string) core.Value {
		spec, err := fspecParse(text)
		require.NoError(t, err)
		return core.NewFormatSpecValue(spec, text)
	}
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		mk(""),
		mk("d"),
		mk(".2f"),
		mk(">5"),
		mk("0,d"),
	)))
}

func TestBytecodeConstBytes(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray(
		core.NewBytesValue([]byte{}, false),
		core.NewBytesValue([]byte{1, 2, 3}, false),
		core.NewBytesValue([]byte("foo bar"), false),
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
				vm.MustMakeInstruction(bc.OpConstant, 3),
				vm.MustMakeInstruction(bc.OpSetLocal, 0),
				vm.MustMakeInstruction(bc.OpGetGlobal, 0),
				vm.MustMakeInstruction(bc.OpGetFree, 0)),
			core.FloatValue(39.2),
			core.IntValue(192),
			core.NewStringValue("bar"),
		)))

	testBytecodeSerialization(t, bytecodeFileSet(
		concatInsts(
			vm.MustMakeInstruction(bc.OpConstant, 0),
			vm.MustMakeInstruction(bc.OpSetGlobal, 0),
			vm.MustMakeInstruction(bc.OpConstant, 6),
			vm.MustMakeInstruction(bc.OpPop)),
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
				"bytes": core.NewBytesValue(make([]byte, 16), false),
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
				vm.MustMakeInstruction(bc.OpConstant, 3),
				vm.MustMakeInstruction(bc.OpSetLocal, 0),
				vm.MustMakeInstruction(bc.OpGetGlobal, 0),
				vm.MustMakeInstruction(bc.OpGetFree, 0),
				vm.MustMakeInstruction(bc.OpBinaryOp, 11),
				vm.MustMakeInstruction(bc.OpGetFree, 1),
				vm.MustMakeInstruction(bc.OpBinaryOp, 11),
				vm.MustMakeInstruction(bc.OpGetLocal, 0),
				vm.MustMakeInstruction(bc.OpBinaryOp, 11),
				vm.MustMakeInstruction(bc.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MustMakeInstruction(bc.OpConstant, 2),
				vm.MustMakeInstruction(bc.OpSetLocal, 0),
				vm.MustMakeInstruction(bc.OpGetFree, 0),
				vm.MustMakeInstruction(bc.OpGetLocal, 0),
				vm.MustMakeInstruction(bc.OpClosure, 4, 2),
				vm.MustMakeInstruction(bc.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MustMakeInstruction(bc.OpConstant, 1),
				vm.MustMakeInstruction(bc.OpSetLocal, 0),
				vm.MustMakeInstruction(bc.OpGetLocal, 0),
				vm.MustMakeInstruction(bc.OpClosure, 5, 1),
				vm.MustMakeInstruction(bc.OpReturn, 1))),
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
					vm.MustMakeInstruction(bc.OpConstant, 3),
					vm.MustMakeInstruction(bc.OpSetLocal, 0),
					vm.MustMakeInstruction(bc.OpGetGlobal, 0),
					vm.MustMakeInstruction(bc.OpGetFree, 0)),
				core.FloatValue(39.2),
				core.IntValue(192),
				core.NewStringValue("bar"))),
		bytecode(
			concatInsts(), objectsArray(
				core.RuneValue('y'),
				core.FloatValue(93.11),
				compiledFunction(1, 0,
					vm.MustMakeInstruction(bc.OpConstant, 3),
					vm.MustMakeInstruction(bc.OpSetLocal, 0),
					vm.MustMakeInstruction(bc.OpGetGlobal, 0),
					vm.MustMakeInstruction(bc.OpGetFree, 0)),
				core.FloatValue(39.2),
				core.IntValue(192),
				core.NewStringValue("bar"))))

	testBytecodeRemoveDuplicates(t,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(bc.OpConstant, 0),
				vm.MustMakeInstruction(bc.OpConstant, 1),
				vm.MustMakeInstruction(bc.OpConstant, 2),
				vm.MustMakeInstruction(bc.OpConstant, 3),
				vm.MustMakeInstruction(bc.OpConstant, 4),
				vm.MustMakeInstruction(bc.OpConstant, 5),
				vm.MustMakeInstruction(bc.OpConstant, 6),
				vm.MustMakeInstruction(bc.OpConstant, 7),
				vm.MustMakeInstruction(bc.OpConstant, 8),
				vm.MustMakeInstruction(bc.OpClosure, 4, 1)),
			objectsArray(
				core.IntValue(1),
				core.FloatValue(2.0),
				core.RuneValue('3'),
				core.NewStringValue("four"),
				compiledFunction(1, 0,
					vm.MustMakeInstruction(bc.OpConstant, 3),
					vm.MustMakeInstruction(bc.OpConstant, 7),
					vm.MustMakeInstruction(bc.OpSetLocal, 0),
					vm.MustMakeInstruction(bc.OpGetGlobal, 0),
					vm.MustMakeInstruction(bc.OpGetFree, 0)),
				core.IntValue(1),
				core.FloatValue(2.0),
				core.RuneValue('3'),
				core.NewStringValue("four"))),
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(bc.OpConstant, 0),
				vm.MustMakeInstruction(bc.OpConstant, 1),
				vm.MustMakeInstruction(bc.OpConstant, 2),
				vm.MustMakeInstruction(bc.OpConstant, 3),
				vm.MustMakeInstruction(bc.OpConstant, 4),
				vm.MustMakeInstruction(bc.OpConstant, 0),
				vm.MustMakeInstruction(bc.OpConstant, 1),
				vm.MustMakeInstruction(bc.OpConstant, 2),
				vm.MustMakeInstruction(bc.OpConstant, 3),
				vm.MustMakeInstruction(bc.OpClosure, 4, 1)),
			objectsArray(
				core.IntValue(1),
				core.FloatValue(2.0),
				core.RuneValue('3'),
				core.NewStringValue("four"),
				compiledFunction(1, 0,
					vm.MustMakeInstruction(bc.OpConstant, 3),
					vm.MustMakeInstruction(bc.OpConstant, 2),
					vm.MustMakeInstruction(bc.OpSetLocal, 0),
					vm.MustMakeInstruction(bc.OpGetGlobal, 0),
					vm.MustMakeInstruction(bc.OpGetFree, 0)))))

	testBytecodeRemoveDuplicates(t,
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(bc.OpConstant, 0),
				vm.MustMakeInstruction(bc.OpConstant, 1),
				vm.MustMakeInstruction(bc.OpConstant, 2),
				vm.MustMakeInstruction(bc.OpConstant, 3),
				vm.MustMakeInstruction(bc.OpConstant, 4)),
			objectsArray(
				core.IntValue(1),
				core.IntValue(2),
				core.IntValue(3),
				core.IntValue(1),
				core.IntValue(3))),
		bytecode(
			concatInsts(
				vm.MustMakeInstruction(bc.OpConstant, 0),
				vm.MustMakeInstruction(bc.OpConstant, 1),
				vm.MustMakeInstruction(bc.OpConstant, 2),
				vm.MustMakeInstruction(bc.OpConstant, 0),
				vm.MustMakeInstruction(bc.OpConstant, 2)),
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
				vm.MustMakeInstruction(bc.OpConstant, 3),
				vm.MustMakeInstruction(bc.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MustMakeInstruction(bc.OpConstant, 2),
				vm.MustMakeInstruction(bc.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MustMakeInstruction(bc.OpConstant, 1),
				vm.MustMakeInstruction(bc.OpReturn, 1))))
	require.Equal(t, rta, 7, b.CountObjects())
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
	input.RemoveDuplicates(rta)

	require.Equal(t, rta, expected.FileSet, input.FileSet)
	require.Equal(t, rta, expected.MainFunction, input.MainFunction)
	require.Equal(t, rta, expected.Constants, input.Constants)
}

func testBytecodeSerialization(t *testing.T, b *vm.Bytecode) {
	var buf bytes.Buffer
	err := b.Encode(&buf)
	require.NoError(t, err)

	r := &vm.Bytecode{}
	err = r.Decode(rta, bytes.NewReader(buf.Bytes()), nil)
	require.NoError(t, err)

	require.Equal(t, rta, b.FileSet, r.FileSet)
	require.Equal(t, rta, b.MainFunction, r.MainFunction)
	require.Equal(t, rta, b.Constants, r.Constants)
}
