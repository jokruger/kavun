package unit

import (
	"testing"

	"github.com/jokruger/kavun/core"
	mock "github.com/jokruger/kavun/tests"
	"github.com/jokruger/kavun/vm"
)

func Test_builtinDelete(t *testing.T) {
	var builtinDelete core.Value
	for _, f := range vm.BuiltinFuncs {
		if (*core.BuiltinFunction)(f.Ptr).Name == "delete" {
			builtinDelete = f
			break
		}
	}
	if builtinDelete.Type == core.VT_UNDEFINED {
		t.Fatal("builtin delete not found")
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
			wantedErr: "invalid delete error: type string does not support delete"},

		{name: "no-args",
			wantedErr: "wrong number of arguments: (delete) expected 2 argument(s), got 0"},

		{name: "empty-args", args: args{[]core.Value{}},
			wantedErr: "wrong number of arguments: (delete) expected 2 argument(s), got 0"},

		{name: "3-args", args: args{[]core.Value{core.NewRecordValue(nil, false), core.NewStringValue(""), core.NewStringValue("")}},
			wantedErr: "wrong number of arguments: (delete) expected 2 argument(s), got 3"},

		{name: "nil-record-no-key", args: args{[]core.Value{core.NewRecordValue(nil, false)}},
			wantedErr: "wrong number of arguments: (delete) expected 2 argument(s), got 1"},

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
			got, err := builtinDelete.Call(mock.Vm, tt.args.args)
			if (err != nil) != (tt.wantedErr != "") {
				t.Errorf("builtinDelete() error = %s, wantedErr %s", err.Error(), tt.wantedErr)
				return
			}
			if tt.wantedErr != "" && (err == nil || err.Error() != tt.wantedErr) {
				t.Errorf("builtinDelete() error = %s, wantedErr %s", err.Error(), tt.wantedErr)
				return
			}
			if tt.want.TypeName() != got.TypeName() {
				t.Errorf("builtinDelete() got type %s, want type %s", got.TypeName(), tt.want.TypeName())
				return
			}
			if !tt.want.Equal(got) {
				t.Errorf("builtinDelete() got %s, want %s", got.String(), tt.want.String())
				return
			}
			if tt.wantedErr == "" && tt.target.Type != core.VT_UNDEFINED {
				if tt.target.TypeName() != tt.args.args[0].TypeName() {
					t.Errorf("builtinDelete() target got type %s, want type %s", tt.args.args[0].TypeName(), tt.target.TypeName())
					return
				}
				if !tt.target.Equal(tt.args.args[0]) {
					t.Errorf("builtinDelete() target got %s, want %s", tt.args.args[0].String(), tt.target.String())
				}
			}
		})
	}
}

func Test_builtinSplice(t *testing.T) {
	var builtinSplice core.Value
	for _, f := range vm.BuiltinFuncs {
		if (*core.BuiltinFunction)(f.Ptr).Name == "splice" {
			builtinSplice = f
			break
		}
	}
	if builtinSplice.Type == core.VT_UNDEFINED {
		t.Fatal("builtin splice not found")
	}
	tests := []struct {
		name      string
		args      []core.Value
		deleted   core.Value
		Array     core.Value
		wantedErr string
	}{
		{name: "no args", args: []core.Value{},
			wantedErr: "wrong number of arguments: (splice) expected at least 1 argument(s), got 0"},

		{name: "invalid args", args: []core.Value{core.NewRecordValue(nil, false)},
			wantedErr: "invalid argument type: (splice) argument first expects type array, got record"},

		{name: "invalid args", args: []core.Value{core.NewArrayValue(nil, false), core.NewStringValue("")},
			wantedErr: "invalid argument type: (splice) argument second expects type int, got string"},

		{name: "negative index", args: []core.Value{core.NewArrayValue(nil, false), core.IntValue(-1)},
			wantedErr: "index out of bounds: (splice, start index) -1 out of range [0, 0]"},

		{name: "non int count",
			args: []core.Value{
				core.NewArrayValue(nil, false),
				core.IntValue(0),
				core.NewStringValue(""),
			},
			wantedErr: "invalid argument type: (splice) argument third expects type int, got string"},

		{name: "negative count",
			args: []core.Value{
				core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(0),
				core.IntValue(-1),
			},
			wantedErr: "logic error: splice delete count must be non-negative"},

		{name: "insert with zero count",
			args: []core.Value{
				core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(0),
				core.IntValue(0),
				core.NewStringValue("b"),
			},
			deleted: core.NewArrayValue([]core.Value{}, false),
			Array:   core.NewArrayValue([]core.Value{core.NewStringValue("b"), core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
		},

		{name: "insert",
			args: []core.Value{
				core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(1),
				core.IntValue(0),
				core.NewStringValue("c"),
				core.NewStringValue("d"),
			},
			deleted: core.NewArrayValue([]core.Value{}, false),
			Array:   core.NewArrayValue([]core.Value{core.IntValue(0), core.NewStringValue("c"), core.NewStringValue("d"), core.IntValue(1), core.IntValue(2)}, false),
		},

		{name: "insert with zero count",
			args: []core.Value{
				core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(1),
				core.IntValue(0),
				core.NewStringValue("c"),
				core.NewStringValue("d"),
			},
			deleted: core.NewArrayValue([]core.Value{}, false),
			Array:   core.NewArrayValue([]core.Value{core.IntValue(0), core.NewStringValue("c"), core.NewStringValue("d"), core.IntValue(1), core.IntValue(2)}, false),
		},

		{name: "insert with delete",
			args: []core.Value{
				core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(1),
				core.IntValue(1),
				core.NewStringValue("c"),
				core.NewStringValue("d"),
			},
			deleted: core.NewArrayValue([]core.Value{core.IntValue(1)}, false),
			Array:   core.NewArrayValue([]core.Value{core.IntValue(0), core.NewStringValue("c"), core.NewStringValue("d"), core.IntValue(2)}, false),
		},

		{name: "insert with delete multi",
			args: []core.Value{
				core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(1),
				core.IntValue(2),
				core.NewStringValue("c"),
				core.NewStringValue("d"),
			},
			deleted: core.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false),
			Array:   core.NewArrayValue([]core.Value{core.IntValue(0), core.NewStringValue("c"), core.NewStringValue("d")}, false),
		},

		{name: "delete all with positive count",
			args: []core.Value{
				core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(0),
				core.IntValue(3),
			},
			deleted: core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
			Array:   core.NewArrayValue([]core.Value{}, false),
		},

		{name: "delete all with big count",
			args: []core.Value{
				core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(0),
				core.IntValue(5),
			},
			deleted: core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
			Array:   core.NewArrayValue([]core.Value{}, false),
		},

		{name: "nothing2",
			args:    []core.Value{core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false)},
			deleted: core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
			Array:   core.NewArrayValue([]core.Value{}, false),
		},

		{name: "pop without count",
			args: []core.Value{
				core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(2),
			},
			deleted: core.NewArrayValue([]core.Value{core.IntValue(2)}, false),
			Array:   core.NewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1)}, false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinSplice.Call(mock.Vm, tt.args)
			if (err != nil) != (tt.wantedErr != "") {
				t.Errorf("builtinSplice() error = %s, wantErr %s", err.Error(), tt.wantedErr)
				return
			}
			if tt.deleted.TypeName() != got.TypeName() {
				t.Errorf("builtinSplice() got type %s, want type %s", got.TypeName(), tt.deleted.TypeName())
				return
			}
			if !tt.deleted.Equal(got) {
				t.Errorf("builtinSplice() got %s, want %s", got.String(), tt.deleted.String())
				return
			}
			if (tt.wantedErr != "") && tt.wantedErr != err.Error() {
				t.Errorf("builtinSplice() error = %v, wantedErr %v", err, tt.wantedErr)
			}
			if tt.Array.Type != core.VT_UNDEFINED {
				if tt.Array.TypeName() != tt.args[0].TypeName() {
					t.Errorf("builtinSplice() array got type %s, want type %s", tt.args[0].TypeName(), tt.Array.TypeName())
					return
				}
				if !tt.Array.Equal(tt.args[0]) {
					t.Errorf("builtinSplice() array got %s, want %s", tt.args[0].String(), tt.Array.String())
				}
			}
		})
	}
}

func Test_builtinRange(t *testing.T) {
	var builtinRange core.Value
	for _, f := range vm.BuiltinFuncs {
		if (*core.BuiltinFunction)(f.Ptr).Name == "range" {
			builtinRange = f
			break
		}
	}
	if builtinRange.Type == core.VT_UNDEFINED {
		t.Fatal("builtin range not found")
	}
	tests := []struct {
		name      string
		args      []core.Value
		result    core.Value
		wantedErr string
	}{
		{name: "no args", args: []core.Value{},
			wantedErr: "wrong number of arguments: (range) expected 2 or 3 argument(s), got 0"},

		{name: "single args", args: []core.Value{core.NewRecordValue(nil, false)},
			wantedErr: "wrong number of arguments: (range) expected 2 or 3 argument(s), got 1"},

		{name: "4 args", args: []core.Value{core.NewRecordValue(nil, false), core.NewStringValue(""), core.NewStringValue(""), core.NewStringValue("")},
			wantedErr: "wrong number of arguments: (range) expected 2 or 3 argument(s), got 4"},

		{name: "invalid start", args: []core.Value{core.NewStringValue(""), core.NewStringValue("")},
			wantedErr: "invalid argument type: (range) argument start expects type int, got string"},

		{name: "invalid stop", args: []core.Value{core.IntValue(0), core.NewStringValue("")},
			wantedErr: "invalid argument type: (range) argument stop expects type int, got string"},

		{name: "invalid step", args: []core.Value{core.IntValue(0), core.IntValue(0), core.NewStringValue("")},
			wantedErr: "invalid argument type: (range) argument step expects type int, got string"},

		{name: "zero step", args: []core.Value{core.IntValue(0), core.IntValue(0), core.IntValue(0)},
			wantedErr: "logic error: range step must be greater than 0, got 0"},

		{name: "negative step", args: []core.Value{core.IntValue(0), core.IntValue(0), intObject(-2)},
			wantedErr: "logic error: range step must be greater than 0, got -2"},

		{name: "same bound", args: []core.Value{core.IntValue(0), core.IntValue(0)},
			result: core.NewArrayValue(nil, false),
		},

		{name: "positive range", args: []core.Value{core.IntValue(0), core.IntValue(5)},
			result: core.NewArrayValue([]core.Value{
				intObject(0),
				intObject(1),
				intObject(2),
				intObject(3),
				intObject(4),
			}, false),
		},

		{name: "negative range", args: []core.Value{core.IntValue(0), core.IntValue(-5)},
			result: core.NewArrayValue([]core.Value{
				intObject(0),
				intObject(-1),
				intObject(-2),
				intObject(-3),
				intObject(-4),
			}, false),
		},

		{name: "positive with step", args: []core.Value{core.IntValue(0), core.IntValue(5), core.IntValue(2)},
			result: core.NewArrayValue([]core.Value{
				intObject(0),
				intObject(2),
				intObject(4),
			}, false),
		},

		{name: "negative with step", args: []core.Value{core.IntValue(0), core.IntValue(-10), core.IntValue(2)},
			result: core.NewArrayValue([]core.Value{
				intObject(0),
				intObject(-2),
				intObject(-4),
				intObject(-6),
				intObject(-8),
			}, false),
		},

		{name: "large range", args: []core.Value{intObject(-10), intObject(10), core.IntValue(3)},
			result: core.NewArrayValue([]core.Value{
				intObject(-10),
				intObject(-7),
				intObject(-4),
				intObject(-1),
				intObject(2),
				intObject(5),
				intObject(8),
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
			if tt.result.Type != core.VT_UNDEFINED {
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
	var builtinFormat core.Value
	for _, f := range vm.BuiltinFuncs {
		if (*core.BuiltinFunction)(f.Ptr).Name == "format" {
			builtinFormat = f
			break
		}
	}
	if builtinFormat.Type == core.VT_UNDEFINED {
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
			wantedErr: "wrong number of arguments: (format) expected 2 argument(s), got 0"},
		{name: "one arg", args: []core.Value{S("hi")},
			wantedErr: "wrong number of arguments: (format) expected 2 argument(s), got 1"},
		{name: "non-string template",
			args:      []core.Value{I(1), arr()},
			wantedErr: "invalid argument type: (format) argument template expects type string, got int"},
		{name: "bad args type",
			args:      []core.Value{S("hi"), I(1)},
			wantedErr: "invalid argument type: (format) argument args expects type array, dict, or record, got int"},

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
			wantedErr: "logic error: format: missing key \"x\""},
		{name: "missing index",
			args:      []core.Value{S("{2}"), arr(S("a"), S("b"))},
			wantedErr: "logic error: format: index 2 out of range [0, 2)"},
		{name: "mode mismatch named template, array args",
			args:      []core.Value{S("{x}"), arr(S("a"))},
			wantedErr: "logic error: format: template uses named placeholders but args is array (expected dict or record)"},
		{name: "mode mismatch indexed template, record args",
			args:      []core.Value{S("{0}"), rec(map[string]core.Value{"0": S("a")})},
			wantedErr: "logic error: format: template uses indexed placeholders but args is record (expected array)"},
		{name: "ref spec wrong type",
			args:      []core.Value{S("{x:{fmt}}"), rec(map[string]core.Value{"x": I(1), "fmt": I(2)})},
			wantedErr: "logic error: format: spec reference must be a string, got int"},
		{name: "ref spec parse error",
			args:      []core.Value{S("{x:{fmt}}"), rec(map[string]core.Value{"x": I(1), "fmt": S("zzz")})},
			wantedErr: "logic error: format: fspec: trailing characters \"zz\" in \"zzz\""},
		{name: "template parse error",
			args:      []core.Value{S("{0} {x}"), arr(S("a"))},
			wantedErr: "logic error: format: cannot mix named and indexed placeholders at offset 4"},
		{name: "bare close brace",
			args:      []core.Value{S("a }"), arr()},
			wantedErr: "logic error: format: unmatched '}' at offset 2 (use '}}' for a literal '}')"},
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
