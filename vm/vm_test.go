package vm_test

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/mock"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/vm"
)

func primitive(v core.Value) core.Primitive {
	if v.Type > value.LastPrimitiveType {
		panic(fmt.Errorf("expected primitive type, got: %v", v.Type))
	}
	return core.Primitive{Type: v.Type, Data: v.Data}
}

func fspecParse(s string) (fspec.FormatSpec, error) {
	return fspec.Parse(s)
}

type srcfile struct {
	name string
	size int
}

func bytecode(instructions bc.Instructions, static core.Static) *vm.Bytecode {
	return &vm.Bytecode{
		FileSet:      ast.NewFileSet(),
		MainFunction: &core.CompiledFunction{Instructions: instructions},
		Static:       static,
	}
}

func concatInsts(instructions ...bc.Instruction) bc.Instructions {
	var concat bc.Instructions
	for _, i := range instructions {
		concat = append(concat, i)
	}
	return concat
}

func testBytecodeSerialization(t *testing.T, b *vm.Bytecode) {
	var buf bytes.Buffer
	err := b.Encode(&buf)
	require.NoError(t, err)

	r := &vm.Bytecode{}
	err = r.Decode(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)

	require.Equal(t, b.FileSet, r.FileSet)
	require.Equal(t, b.MainFunction, r.MainFunction)
	require.Equal(t, b.Static, r.Static)
}

func Test_builtinRemove(t *testing.T) {
	builtinRemove, ok := vm.BuiltinFunctions["remove"]
	if !ok {
		t.Fatal("builtin remove not found")
	}
	if builtinRemove.Type == value.Undefined {
		t.Fatal("builtin remove not found")
	}
	type args struct {
		args []core.Value
	}
	tests := []struct {
		name      string
		args      args
		want      core.Value
		wantedErr string
		target    core.Value
	}{
		{name: "invalid-arg", args: args{[]core.Value{core.NewStringValue(""), core.NewStringValue("")}},
			wantedErr: "not_deletable: type string does not support delete"},

		{name: "no-args",
			wantedErr: "wrong_num_arguments: (remove) expected 2 argument(s), got 0"},

		{name: "empty-args", args: args{[]core.Value{}},
			wantedErr: "wrong_num_arguments: (remove) expected 2 argument(s), got 0"},

		{name: "3-args", args: args{[]core.Value{core.NewRecordValue(nil, false), core.NewStringValue(""), core.NewStringValue("")}},
			wantedErr: "wrong_num_arguments: (remove) expected 2 argument(s), got 3"},

		{name: "nil-record-no-key", args: args{[]core.Value{core.NewRecordValue(nil, false)}},
			wantedErr: "wrong_num_arguments: (remove) expected 2 argument(s), got 1"},

		{name: "record-missing-key",
			args: args{
				[]core.Value{
					core.NewRecordValue(map[string]core.Value{
						"key": core.NewStringValue("value"),
					}, false),
					core.NewStringValue("key1")}},
			want:   core.NewRecordValue(map[string]core.Value{"key": core.NewStringValue("value")}, false),
			target: core.NewRecordValue(map[string]core.Value{"key": core.NewStringValue("value")}, false),
		},

		{name: "record-emptied",
			args: args{
				[]core.Value{
					core.NewRecordValue(map[string]core.Value{
						"key": core.NewStringValue("value"),
					}, false),
					core.NewStringValue("key")}},
			want: core.NewRecordValue(map[string]core.Value{}, false),
			// delete() is pure now: the receiver (args[0]) is left untouched, unlike delete_in_place()
			target: core.NewRecordValue(map[string]core.Value{"key": core.NewStringValue("value")}, false),
		},

		{name: "record-multi-keys",
			args: args{
				[]core.Value{
					core.NewRecordValue(map[string]core.Value{
						"key1": core.NewStringValue("value1"),
						"key2": core.IntValue(10),
					}, false),
					core.NewStringValue("key1")}},
			want: core.NewRecordValue(map[string]core.Value{"key2": core.IntValue(10)}, false),
			// delete() is pure now: the receiver (args[0]) is left untouched, unlike delete_in_place()
			target: core.NewRecordValue(map[string]core.Value{"key1": core.NewStringValue("value1"), "key2": core.IntValue(10)}, false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinRemove.Call(mock.Vm, tt.args.args)
			if (err != nil) != (tt.wantedErr != "") {
				t.Errorf("builtinRemove() error = %s, wantedErr %s", err.Error(), tt.wantedErr)
				return
			}
			if tt.wantedErr != "" && (err == nil || err.Error() != tt.wantedErr) {
				t.Errorf("builtinRemove() error = %s, wantedErr %s", err.Error(), tt.wantedErr)
				return
			}
			if tt.want.TypeName() != got.TypeName() {
				t.Errorf("builtinRemove() got type %s, want type %s", got.TypeName(), tt.want.TypeName())
				return
			}
			if !tt.want.Equal(got) {
				t.Errorf("builtinRemove() got %s, want %s", got.String(), tt.want.String())
				return
			}
			if tt.wantedErr == "" && tt.target.Type != value.Undefined {
				if tt.target.TypeName() != tt.args.args[0].TypeName() {
					t.Errorf("builtinRemove() target got type %s, want type %s", tt.args.args[0].TypeName(), tt.target.TypeName())
					return
				}
				if !tt.target.Equal(tt.args.args[0]) {
					t.Errorf("builtinRemove() target got %s, want %s", tt.args.args[0].String(), tt.target.String())
				}
			}
		})
	}
}

// Test_builtinDeleteInPlace mirrors Test_builtinDelete's success cases but asserts the mutating behavior
// delete_in_place() preserves from what used to be delete()'s only behavior.
func Test_builtinRemoveInPlace(t *testing.T) {
	builtinRemoveInPlace, ok := vm.BuiltinFunctions["remove_in_place"]
	if !ok {
		t.Fatal("builtin remove_in_place not found")
	}
	if builtinRemoveInPlace.Type == value.Undefined {
		t.Fatal("builtin remove_in_place not found")
	}
	type args struct {
		args []core.Value
	}
	tests := []struct {
		name      string
		args      args
		want      core.Value
		wantedErr string
		target    core.Value
	}{
		{name: "invalid-arg", args: args{[]core.Value{core.NewStringValue(""), core.NewStringValue("")}},
			wantedErr: "not_deletable: type string does not support delete"},

		{name: "no-args",
			wantedErr: "wrong_num_arguments: (remove_in_place) expected 2 argument(s), got 0"},

		{name: "record-emptied",
			args: args{
				[]core.Value{
					core.NewRecordValue(map[string]core.Value{
						"key": core.NewStringValue("value"),
					}, false),
					core.NewStringValue("key")}},
			want:   core.NewRecordValue(map[string]core.Value{}, false),
			target: core.NewRecordValue(map[string]core.Value{}, false),
		},

		{name: "record-multi-keys",
			args: args{
				[]core.Value{
					core.NewRecordValue(map[string]core.Value{
						"key1": core.NewStringValue("value1"),
						"key2": core.IntValue(10),
					}, false),
					core.NewStringValue("key1")}},
			want:   core.NewRecordValue(map[string]core.Value{"key2": core.IntValue(10)}, false),
			target: core.NewRecordValue(map[string]core.Value{"key2": core.IntValue(10)}, false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinRemoveInPlace.Call(mock.Vm, tt.args.args)
			if (err != nil) != (tt.wantedErr != "") {
				t.Errorf("builtinRemoveInPlace() error = %v, wantedErr %s", err, tt.wantedErr)
				return
			}
			if tt.wantedErr != "" && (err == nil || err.Error() != tt.wantedErr) {
				t.Errorf("builtinRemoveInPlace() error = %s, wantedErr %s", err.Error(), tt.wantedErr)
				return
			}
			if tt.wantedErr != "" {
				return
			}
			if tt.want.TypeName() != got.TypeName() {
				t.Errorf("builtinRemoveInPlace() got type %s, want type %s", got.TypeName(), tt.want.TypeName())
				return
			}
			if !tt.want.Equal(got) {
				t.Errorf("builtinRemoveInPlace() got %s, want %s", got.String(), tt.want.String())
				return
			}
			if tt.target.Type != value.Undefined {
				if !tt.target.Equal(tt.args.args[0]) {
					t.Errorf("builtinRemoveInPlace() target got %s, want %s", tt.args.args[0].String(), tt.target.String())
				}
			}
		})
	}
}

func Test_builtinRange(t *testing.T) {
	builtinRange, ok := vm.BuiltinFunctions["range"]
	if !ok {
		t.Fatal("builtin range not found")
	}
	if builtinRange.Type == value.Undefined {
		t.Fatal("builtin range not found")
	}
	tests := []struct {
		name      string
		args      []core.Value
		result    core.Value
		wantedErr string
	}{
		// range() is the type's zero form — the empty range
		{name: "no args", args: []core.Value{},
			result: core.NewArrayValue(nil, false),
		},

		// a record/dict first argument is the components form
		{name: "components missing start", args: []core.Value{core.NewRecordValue(nil, false)},
			wantedErr: "invalid_value: (range) component start is required"},

		{name: "single int arg", args: []core.Value{core.IntValue(1)},
			wantedErr: "wrong_num_arguments: (range) expected 0, 2 or 3 argument(s), got 1"},

		{name: "4 args", args: []core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(1), core.IntValue(1)},
			wantedErr: "wrong_num_arguments: (range) expected 0, 2 or 3 argument(s), got 4"},

		{name: "invalid start", args: []core.Value{core.NewStringValue(""), core.NewStringValue("")},
			wantedErr: "invalid_argument_type: (range) argument start expects type int, got string"},

		{name: "invalid stop", args: []core.Value{core.IntValue(0), core.NewStringValue("")},
			wantedErr: "invalid_argument_type: (range) argument stop expects type int, got string"},

		{name: "invalid step", args: []core.Value{core.IntValue(0), core.IntValue(0), core.NewStringValue("")},
			wantedErr: "invalid_argument_type: (range) argument step expects type int, got string"},

		{name: "zero step", args: []core.Value{core.IntValue(0), core.IntValue(0), core.IntValue(0)},
			wantedErr: "invalid_value: range step must be greater than 0, got 0"},

		{name: "negative step", args: []core.Value{core.IntValue(0), core.IntValue(0), core.IntValue(-2)},
			wantedErr: "invalid_value: range step must be greater than 0, got -2"},

		{name: "same bound", args: []core.Value{core.IntValue(0), core.IntValue(0)},
			result: core.NewArrayValue(nil, false),
		},

		{name: "positive range", args: []core.Value{core.IntValue(0), core.IntValue(5)},
			result: core.NewArrayValue([]core.Value{
				core.IntValue(0),
				core.IntValue(1),
				core.IntValue(2),
				core.IntValue(3),
				core.IntValue(4),
			}, false),
		},

		{name: "negative range", args: []core.Value{core.IntValue(0), core.IntValue(-5)},
			result: core.NewArrayValue([]core.Value{
				core.IntValue(0),
				core.IntValue(-1),
				core.IntValue(-2),
				core.IntValue(-3),
				core.IntValue(-4),
			}, false),
		},

		{name: "positive with step", args: []core.Value{core.IntValue(0), core.IntValue(5), core.IntValue(2)},
			result: core.NewArrayValue([]core.Value{
				core.IntValue(0),
				core.IntValue(2),
				core.IntValue(4),
			}, false),
		},

		{name: "negative with step", args: []core.Value{core.IntValue(0), core.IntValue(-10), core.IntValue(2)},
			result: core.NewArrayValue([]core.Value{
				core.IntValue(0),
				core.IntValue(-2),
				core.IntValue(-4),
				core.IntValue(-6),
				core.IntValue(-8),
			}, false),
		},

		{name: "large range", args: []core.Value{core.IntValue(-10), core.IntValue(10), core.IntValue(3)},
			result: core.NewArrayValue([]core.Value{
				core.IntValue(-10),
				core.IntValue(-7),
				core.IntValue(-4),
				core.IntValue(-1),
				core.IntValue(2),
				core.IntValue(5),
				core.IntValue(8),
			}, false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinRange.Call(mock.Vm, tt.args)
			if (err != nil) != (tt.wantedErr != "") {
				t.Errorf("builtinRange() error = %s, wantErr %s", err.Error(), tt.wantedErr)
				return
			}
			if (tt.wantedErr != "") && tt.wantedErr != err.Error() {
				t.Errorf("builtinRange() error = %s, wantedErr %s", err.Error(), tt.wantedErr)
				return
			}
			if tt.result.Type != value.Undefined {
				got, err = got.MethodCall(mock.Vm, "array", nil)
				if err != nil {
					t.Errorf("builtinRange() array error = %s", err.Error())
					return
				}
			}
			if tt.result.TypeName() != got.TypeName() {
				t.Errorf("builtinRange() got type %s, want type %s", got.TypeName(), tt.result.TypeName())
				return
			}
			if !tt.result.Equal(got) {
				t.Errorf("builtinRange() got %s, want %s", got.String(), tt.result.String())
				return
			}
		})
	}
}

func Test_builtinFormat(t *testing.T) {
	builtinFormat, ok := vm.BuiltinFunctions["format"]
	if !ok {
		t.Fatal("builtin format not found")
	}
	if builtinFormat.Type == value.Undefined {
		t.Fatal("builtin format not found")
	}

	rec := func(m map[string]core.Value) core.Value { return core.NewRecordValue(m, false) }
	dict := func(m map[string]core.Value) core.Value { return core.NewDictValue(m, false) }
	arr := func(vs ...core.Value) core.Value { return core.NewArrayValue(vs, false) }
	S := core.NewStringValue
	I := core.IntValue

	tests := []struct {
		name      string
		args      []core.Value
		want      string
		wantedErr string
	}{
		{name: "no args",
			wantedErr: "wrong_num_arguments: (format) expected 1 or 2 argument(s), got 0"},
		// format(x) renders any value — the 1-arg render form
		{name: "one arg", args: []core.Value{S("hi")}, want: "hi"},
		{name: "one arg int", args: []core.Value{I(5)}, want: "5"},
		{name: "non-string template",
			args:      []core.Value{I(1), arr()},
			wantedErr: "invalid_argument_type: (format) argument template expects type string, got int"},
		{name: "bad args type",
			args:      []core.Value{S("hi"), I(1)},
			wantedErr: "invalid_argument_type: (format) argument args expects type array, dict, or record, got int"},

		{name: "empty template indexed args",
			args: []core.Value{S(""), arr()}, want: ""},
		{name: "literal only",
			args: []core.Value{S("hello"), arr()}, want: "hello"},
		{name: "escaped braces",
			args: []core.Value{S("a {{ b }} c"), arr()}, want: "a { b } c"},

		{name: "named record",
			args: []core.Value{S("hello {x} from {y}!"),
				rec(map[string]core.Value{"x": S("kavun"), "y": S("Kherson")})},
			want: "hello kavun from Kherson!"},
		{name: "named dict",
			args: []core.Value{S("hello {x}"),
				dict(map[string]core.Value{"x": S("world")})},
			want: "hello world"},

		{name: "indexed array",
			args: []core.Value{S("hello {0} from {1}!"),
				arr(S("kavun"), S("Kherson"))},
			want: "hello kavun from Kherson!"},
		{name: "indexed array reuse",
			args: []core.Value{S("{0}-{1}-{0}"),
				arr(S("a"), S("b"))},
			want: "a-b-a"},

		{name: "literal spec",
			args: []core.Value{S("{x:05d}"),
				rec(map[string]core.Value{"x": I(42)})},
			want: "00042"},
		{name: "ref spec named",
			args: []core.Value{S("{x:{fmt}}"),
				rec(map[string]core.Value{"x": I(42), "fmt": S("05d")})},
			want: "00042"},
		{name: "ref spec indexed",
			args: []core.Value{S("{0:{1}}"),
				arr(I(42), S("05d"))},
			want: "00042"},

		{name: "missing named key",
			args:      []core.Value{S("{x}"), rec(map[string]core.Value{})},
			wantedErr: "invalid_value: format: missing key \"x\""},
		{name: "missing index",
			args:      []core.Value{S("{2}"), arr(S("a"), S("b"))},
			wantedErr: "index_out_of_bounds: (format) 2 out of range [0, 2]"},
		{name: "mode mismatch named template, array args",
			args:      []core.Value{S("{x}"), arr(S("a"))},
			wantedErr: "invalid_argument_type: (format) argument args expects type dict or record, got array"},
		{name: "mode mismatch indexed template, record args",
			args:      []core.Value{S("{0}"), rec(map[string]core.Value{"0": S("a")})},
			wantedErr: "invalid_argument_type: (format) argument args expects type array, got record"},
		{name: "ref spec wrong type",
			args:      []core.Value{S("{x:{fmt}}"), rec(map[string]core.Value{"x": I(1), "fmt": I(2)})},
			wantedErr: "invalid_argument_type: (format) argument spec ref expects type string, got int"},
		{name: "ref spec parse error",
			args:      []core.Value{S("{x:{fmt}}"), rec(map[string]core.Value{"x": I(1), "fmt": S("zzz")})},
			wantedErr: "unsupported_format_spec: (format) trailing characters \"zz\" in \"zzz\""},
		{name: "template parse error",
			args:      []core.Value{S("{0} {x}"), arr(S("a"))},
			wantedErr: "unsupported_format_spec: (format) cannot mix named and indexed placeholders at offset 4"},
		{name: "bare close brace",
			args:      []core.Value{S("a }"), arr()},
			wantedErr: "unsupported_format_spec: (format) unmatched '}' at offset 2 (use '}}' for a literal '}')"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinFormat.Call(mock.Vm, tt.args)
			if tt.wantedErr != "" {
				if err == nil || err.Error() != tt.wantedErr {
					t.Fatalf("expected error %q, got err=%v val=%v", tt.wantedErr, err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			s, ok := got.AsString()
			if !ok {
				t.Fatalf("expected string result, got %s", got.TypeName())
			}
			if s != tt.want {
				t.Fatalf("got %q, want %q", s, tt.want)
			}
		})
	}
}

func TestBytecodeEmpty(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), core.Static{}))
}

func TestBytecodeConstUndefined(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), core.Static{
		Primitives: []core.Primitive{primitive(core.Undefined)},
	}))
}

func TestBytecodeConstBool(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), core.Static{
		Primitives: []core.Primitive{
			primitive(core.True),
			primitive(core.False),
		},
	}))
}

func TestBytecodeConstChar(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), core.Static{
		Primitives: []core.Primitive{
			primitive(core.RuneValue('a')),
			primitive(core.RuneValue('b')),
			primitive(core.RuneValue('c')),
		},
	}))
}

func TestBytecodeConstInt(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), core.Static{
		Primitives: []core.Primitive{
			primitive(core.IntValue(1)),
			primitive(core.IntValue(2)),
			primitive(core.IntValue(3)),
			primitive(core.IntValue(1234567890)),
		},
	}))
}

func TestBytecodeConstFloat(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), core.Static{
		Primitives: []core.Primitive{
			primitive(core.FloatValue(0.123)),
			primitive(core.FloatValue(123456.789)),
		},
	}))
}

func TestBytecodeConstString(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), core.Static{
		Strings: []string{"", "foo", "foo bar"},
	}))
}

func TestBytecodeConstFormatSpec(t *testing.T) {
	mk := func(text string) core.FormatSpec {
		spec, err := fspecParse(text)
		require.NoError(t, err)
		var sv core.FormatSpec
		sv.Set(spec, text)
		return sv
	}
	testBytecodeSerialization(t, bytecode(concatInsts(), core.Static{
		FormatSpecs: []core.FormatSpec{
			mk(""),
			mk("d"),
			mk(".2f"),
			mk(">5"),
			mk("0,d"),
		},
	}))
}

func runVM(t *testing.T, instructions bc.Instructions, st core.Static) []core.Value {
	t.Helper()
	m := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	globals := make([]core.Value, vm.GlobalsSize)
	m.Reset(bytecode(instructions, st), globals)
	require.NoError(t, m.Run())
	return globals
}

func TestVM_PushOpcodes(t *testing.T) {
	globals := runVM(t, concatInsts(
		compiler.NewPushBool(true),
		compiler.NewStoreGlobal(0),
		compiler.NewPushBool(false),
		compiler.NewStoreGlobal(1),
		compiler.NewPushByte(255),
		compiler.NewStoreGlobal(2),
		compiler.NewPushRune('A'),
		compiler.NewStoreGlobal(3),
		compiler.NewPushInt(-1),
		compiler.NewStoreGlobal(4),
		compiler.NewPushInt(math.MinInt32),
		compiler.NewStoreGlobal(5),
		compiler.NewSuspend(),
	), core.Static{})

	require.Equal(t, core.BoolValue(true), globals[0])
	require.Equal(t, core.BoolValue(false), globals[1])
	require.Equal(t, core.ByteValue(255), globals[2])
	require.Equal(t, core.RuneValue('A'), globals[3])
	require.Equal(t, core.IntValue(-1), globals[4])
	require.Equal(t, core.IntValue(math.MinInt32), globals[5])
}

func TestVM_JumpUsesInstructionIndex(t *testing.T) {
	globals := runVM(t, concatInsts(
		compiler.NewPushInt(1),
		compiler.NewStoreGlobal(0),
		compiler.NewJump(5),
		compiler.NewPushInt(2),
		compiler.NewStoreGlobal(0),
		compiler.NewPushInt(3),
		compiler.NewStoreGlobal(0),
		compiler.NewSuspend(),
	), core.Static{})

	require.Equal(t, core.IntValue(3), globals[0])
}
