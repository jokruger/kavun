package vm_test

import (
	"testing"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/internal/mock"
	"github.com/jokruger/kavun/vm"
)

func Test_builtinDelete(t *testing.T) {
	rta := core.NewArena(nil)

	builtinDelete, ok := vm.BuiltinFunctions["delete"]
	if !ok {
		t.Fatal("builtin delete not found")
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
		{name: "invalid-arg", args: args{[]core.Value{rta.MustNewStringValue(""), rta.MustNewStringValue("")}},
			wantedErr: "not_deletable: type string does not support delete"},

		{name: "no-args",
			wantedErr: "wrong_num_arguments: (delete) expected 2 argument(s), got 0"},

		{name: "empty-args", args: args{[]core.Value{}},
			wantedErr: "wrong_num_arguments: (delete) expected 2 argument(s), got 0"},

		{name: "3-args", args: args{[]core.Value{rta.MustNewRecordValue(nil, false), rta.MustNewStringValue(""), rta.MustNewStringValue("")}},
			wantedErr: "wrong_num_arguments: (delete) expected 2 argument(s), got 3"},

		{name: "nil-record-no-key", args: args{[]core.Value{rta.MustNewRecordValue(nil, false)}},
			wantedErr: "wrong_num_arguments: (delete) expected 2 argument(s), got 1"},

		{name: "record-missing-key",
			args: args{
				[]core.Value{
					rta.MustNewRecordValue(map[string]core.Value{
						"key": rta.MustNewStringValue("value"),
					}, false),
					rta.MustNewStringValue("key1")}},
			want:   rta.MustNewRecordValue(map[string]core.Value{"key": rta.MustNewStringValue("value")}, false),
			target: rta.MustNewRecordValue(map[string]core.Value{"key": rta.MustNewStringValue("value")}, false),
		},

		{name: "record-emptied",
			args: args{
				[]core.Value{
					rta.MustNewRecordValue(map[string]core.Value{
						"key": rta.MustNewStringValue("value"),
					}, false),
					rta.MustNewStringValue("key")}},
			want:   rta.MustNewRecordValue(map[string]core.Value{}, false),
			target: rta.MustNewRecordValue(map[string]core.Value{}, false),
		},

		{name: "record-multi-keys",
			args: args{
				[]core.Value{
					rta.MustNewRecordValue(map[string]core.Value{
						"key1": rta.MustNewStringValue("value1"),
						"key2": core.IntValue(10),
					}, false),
					rta.MustNewStringValue("key1")}},
			want:   rta.MustNewRecordValue(map[string]core.Value{"key2": core.IntValue(10)}, false),
			target: rta.MustNewRecordValue(map[string]core.Value{"key2": core.IntValue(10)}, false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinDelete.Call(rta, mock.Vm, tt.args.args)
			if (err != nil) != (tt.wantedErr != "") {
				t.Errorf("builtinDelete() error = %s, wantedErr %s", err.Error(), tt.wantedErr)
				return
			}
			if tt.wantedErr != "" && (err == nil || err.Error() != tt.wantedErr) {
				t.Errorf("builtinDelete() error = %s, wantedErr %s", err.Error(), tt.wantedErr)
				return
			}
			if tt.want.TypeName(rta) != got.TypeName(rta) {
				t.Errorf("builtinDelete() got type %s, want type %s", got.TypeName(rta), tt.want.TypeName(rta))
				return
			}
			if !tt.want.Equal(rta, got) {
				t.Errorf("builtinDelete() got %s, want %s", got.String(rta), tt.want.String(rta))
				return
			}
			if tt.wantedErr == "" && tt.target.Type != core.VT_UNDEFINED {
				if tt.target.TypeName(rta) != tt.args.args[0].TypeName(rta) {
					t.Errorf("builtinDelete() target got type %s, want type %s", tt.args.args[0].TypeName(rta), tt.target.TypeName(rta))
					return
				}
				if !tt.target.Equal(rta, tt.args.args[0]) {
					t.Errorf("builtinDelete() target got %s, want %s", tt.args.args[0].String(rta), tt.target.String(rta))
				}
			}
		})
	}
}

func Test_builtinSplice(t *testing.T) {
	rta := core.NewArena(nil)

	builtinSplice, ok := vm.BuiltinFunctions["splice"]
	if !ok {
		t.Fatal("builtin splice not found")
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
			wantedErr: "wrong_num_arguments: (splice) expected at least 1 argument(s), got 0"},

		{name: "invalid args", args: []core.Value{rta.MustNewRecordValue(nil, false)},
			wantedErr: "invalid_argument_type: (splice) argument first expects type array, got record"},

		{name: "invalid args", args: []core.Value{rta.MustNewArrayValue(nil, false), rta.MustNewStringValue("")},
			wantedErr: "invalid_argument_type: (splice) argument second expects type int, got string"},

		{name: "negative index", args: []core.Value{rta.MustNewArrayValue(nil, false), core.IntValue(-1)},
			wantedErr: "index_out_of_bounds: (splice, start index) -1 out of range [0, 0]"},

		{name: "non int count",
			args: []core.Value{
				rta.MustNewArrayValue(nil, false),
				core.IntValue(0),
				rta.MustNewStringValue(""),
			},
			wantedErr: "invalid_argument_type: (splice) argument third expects type int, got string"},

		{name: "negative count",
			args: []core.Value{
				rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(0),
				core.IntValue(-1),
			},
			wantedErr: "invalid_value: splice delete count must be non-negative"},

		{name: "insert with zero count",
			args: []core.Value{
				rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(0),
				core.IntValue(0),
				rta.MustNewStringValue("b"),
			},
			deleted: rta.MustNewArrayValue([]core.Value{}, false),
			Array:   rta.MustNewArrayValue([]core.Value{rta.MustNewStringValue("b"), core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
		},

		{name: "insert",
			args: []core.Value{
				rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(1),
				core.IntValue(0),
				rta.MustNewStringValue("c"),
				rta.MustNewStringValue("d"),
			},
			deleted: rta.MustNewArrayValue([]core.Value{}, false),
			Array:   rta.MustNewArrayValue([]core.Value{core.IntValue(0), rta.MustNewStringValue("c"), rta.MustNewStringValue("d"), core.IntValue(1), core.IntValue(2)}, false),
		},

		{name: "insert with zero count",
			args: []core.Value{
				rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(1),
				core.IntValue(0),
				rta.MustNewStringValue("c"),
				rta.MustNewStringValue("d"),
			},
			deleted: rta.MustNewArrayValue([]core.Value{}, false),
			Array:   rta.MustNewArrayValue([]core.Value{core.IntValue(0), rta.MustNewStringValue("c"), rta.MustNewStringValue("d"), core.IntValue(1), core.IntValue(2)}, false),
		},

		{name: "insert with delete",
			args: []core.Value{
				rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(1),
				core.IntValue(1),
				rta.MustNewStringValue("c"),
				rta.MustNewStringValue("d"),
			},
			deleted: rta.MustNewArrayValue([]core.Value{core.IntValue(1)}, false),
			Array:   rta.MustNewArrayValue([]core.Value{core.IntValue(0), rta.MustNewStringValue("c"), rta.MustNewStringValue("d"), core.IntValue(2)}, false),
		},

		{name: "insert with delete multi",
			args: []core.Value{
				rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(1),
				core.IntValue(2),
				rta.MustNewStringValue("c"),
				rta.MustNewStringValue("d"),
			},
			deleted: rta.MustNewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false),
			Array:   rta.MustNewArrayValue([]core.Value{core.IntValue(0), rta.MustNewStringValue("c"), rta.MustNewStringValue("d")}, false),
		},

		{name: "delete all with positive count",
			args: []core.Value{
				rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(0),
				core.IntValue(3),
			},
			deleted: rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
			Array:   rta.MustNewArrayValue([]core.Value{}, false),
		},

		{name: "delete all with big count",
			args: []core.Value{
				rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(0),
				core.IntValue(5),
			},
			deleted: rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
			Array:   rta.MustNewArrayValue([]core.Value{}, false),
		},

		{name: "nothing2",
			args:    []core.Value{rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false)},
			deleted: rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
			Array:   rta.MustNewArrayValue([]core.Value{}, false),
		},

		{name: "pop without count",
			args: []core.Value{
				rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1), core.IntValue(2)}, false),
				core.IntValue(2),
			},
			deleted: rta.MustNewArrayValue([]core.Value{core.IntValue(2)}, false),
			Array:   rta.MustNewArrayValue([]core.Value{core.IntValue(0), core.IntValue(1)}, false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinSplice.Call(rta, mock.Vm, tt.args)
			if (err != nil) != (tt.wantedErr != "") {
				t.Errorf("builtinSplice() error = %s, wantErr %s", err.Error(), tt.wantedErr)
				return
			}
			if tt.deleted.TypeName(rta) != got.TypeName(rta) {
				t.Errorf("builtinSplice() got type %s, want type %s", got.TypeName(rta), tt.deleted.TypeName(rta))
				return
			}
			if !tt.deleted.Equal(rta, got) {
				t.Errorf("builtinSplice() got %s, want %s", got.String(rta), tt.deleted.String(rta))
				return
			}
			if (tt.wantedErr != "") && tt.wantedErr != err.Error() {
				t.Errorf("builtinSplice() error = %v, wantedErr %v", err, tt.wantedErr)
			}
			if tt.Array.Type != core.VT_UNDEFINED {
				if tt.Array.TypeName(rta) != tt.args[0].TypeName(rta) {
					t.Errorf("builtinSplice() array got type %s, want type %s", tt.args[0].TypeName(rta), tt.Array.TypeName(rta))
					return
				}
				if !tt.Array.Equal(rta, tt.args[0]) {
					t.Errorf("builtinSplice() array got %s, want %s", tt.args[0].String(rta), tt.Array.String(rta))
				}
			}
		})
	}
}

func Test_builtinRange(t *testing.T) {
	rta := core.NewArena(nil)

	builtinRange, ok := vm.BuiltinFunctions["range"]
	if !ok {
		t.Fatal("builtin range not found")
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
			wantedErr: "wrong_num_arguments: (range) expected 2 or 3 argument(s), got 0"},

		{name: "single args", args: []core.Value{rta.MustNewRecordValue(nil, false)},
			wantedErr: "wrong_num_arguments: (range) expected 2 or 3 argument(s), got 1"},

		{name: "4 args", args: []core.Value{rta.MustNewRecordValue(nil, false), rta.MustNewStringValue(""), rta.MustNewStringValue(""), rta.MustNewStringValue("")},
			wantedErr: "wrong_num_arguments: (range) expected 2 or 3 argument(s), got 4"},

		{name: "invalid start", args: []core.Value{rta.MustNewStringValue(""), rta.MustNewStringValue("")},
			wantedErr: "invalid_argument_type: (range) argument start expects type int, got string"},

		{name: "invalid stop", args: []core.Value{core.IntValue(0), rta.MustNewStringValue("")},
			wantedErr: "invalid_argument_type: (range) argument stop expects type int, got string"},

		{name: "invalid step", args: []core.Value{core.IntValue(0), core.IntValue(0), rta.MustNewStringValue("")},
			wantedErr: "invalid_argument_type: (range) argument step expects type int, got string"},

		{name: "zero step", args: []core.Value{core.IntValue(0), core.IntValue(0), core.IntValue(0)},
			wantedErr: "invalid_value: range step must be greater than 0, got 0"},

		{name: "negative step", args: []core.Value{core.IntValue(0), core.IntValue(0), core.IntValue(-2)},
			wantedErr: "invalid_value: range step must be greater than 0, got -2"},

		{name: "same bound", args: []core.Value{core.IntValue(0), core.IntValue(0)},
			result: rta.MustNewArrayValue(nil, false),
		},

		{name: "positive range", args: []core.Value{core.IntValue(0), core.IntValue(5)},
			result: rta.MustNewArrayValue([]core.Value{
				core.IntValue(0),
				core.IntValue(1),
				core.IntValue(2),
				core.IntValue(3),
				core.IntValue(4),
			}, false),
		},

		{name: "negative range", args: []core.Value{core.IntValue(0), core.IntValue(-5)},
			result: rta.MustNewArrayValue([]core.Value{
				core.IntValue(0),
				core.IntValue(-1),
				core.IntValue(-2),
				core.IntValue(-3),
				core.IntValue(-4),
			}, false),
		},

		{name: "positive with step", args: []core.Value{core.IntValue(0), core.IntValue(5), core.IntValue(2)},
			result: rta.MustNewArrayValue([]core.Value{
				core.IntValue(0),
				core.IntValue(2),
				core.IntValue(4),
			}, false),
		},

		{name: "negative with step", args: []core.Value{core.IntValue(0), core.IntValue(-10), core.IntValue(2)},
			result: rta.MustNewArrayValue([]core.Value{
				core.IntValue(0),
				core.IntValue(-2),
				core.IntValue(-4),
				core.IntValue(-6),
				core.IntValue(-8),
			}, false),
		},

		{name: "large range", args: []core.Value{core.IntValue(-10), core.IntValue(10), core.IntValue(3)},
			result: rta.MustNewArrayValue([]core.Value{
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
			got, err := builtinRange.Call(rta, mock.Vm, tt.args)
			if (err != nil) != (tt.wantedErr != "") {
				t.Errorf("builtinRange() error = %s, wantErr %s", err.Error(), tt.wantedErr)
				return
			}
			if (tt.wantedErr != "") && tt.wantedErr != err.Error() {
				t.Errorf("builtinRange() error = %s, wantedErr %s", err.Error(), tt.wantedErr)
				return
			}
			if tt.result.Type != core.VT_UNDEFINED {
				got, err = got.MethodCall(rta, mock.Vm, "array", nil)
				if err != nil {
					t.Errorf("builtinRange() array error = %s", err.Error())
					return
				}
			}
			if tt.result.TypeName(rta) != got.TypeName(rta) {
				t.Errorf("builtinRange() got type %s, want type %s", got.TypeName(rta), tt.result.TypeName(rta))
				return
			}
			if !tt.result.Equal(rta, got) {
				t.Errorf("builtinRange() got %s, want %s", got.String(rta), tt.result.String(rta))
				return
			}
		})
	}
}

func Test_builtinFormat(t *testing.T) {
	rta := core.NewArena(nil)

	builtinFormat, ok := vm.BuiltinFunctions["format"]
	if !ok {
		t.Fatal("builtin format not found")
	}
	if builtinFormat.Type == core.VT_UNDEFINED {
		t.Fatal("builtin format not found")
	}

	rec := func(m map[string]core.Value) core.Value { return rta.MustNewRecordValue(m, false) }
	dict := func(m map[string]core.Value) core.Value { return rta.MustNewDictValue(m, false) }
	arr := func(vs ...core.Value) core.Value { return rta.MustNewArrayValue(vs, false) }
	S := rta.MustNewStringValue
	I := core.IntValue

	tests := []struct {
		name      string
		args      []core.Value
		want      string
		wantedErr string
	}{
		{name: "no args",
			wantedErr: "wrong_num_arguments: (format) expected 2 argument(s), got 0"},
		{name: "one arg", args: []core.Value{S("hi")},
			wantedErr: "wrong_num_arguments: (format) expected 2 argument(s), got 1"},
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
			wantedErr: "unsupported_format_spec: format: fspec: trailing characters \"zz\" in \"zzz\""},
		{name: "template parse error",
			args:      []core.Value{S("{0} {x}"), arr(S("a"))},
			wantedErr: "unsupported_format_spec: format: cannot mix named and indexed placeholders at offset 4"},
		{name: "bare close brace",
			args:      []core.Value{S("a }"), arr()},
			wantedErr: "unsupported_format_spec: format: unmatched '}' at offset 2 (use '}}' for a literal '}')"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinFormat.Call(rta, mock.Vm, tt.args)
			if tt.wantedErr != "" {
				if err == nil || err.Error() != tt.wantedErr {
					t.Fatalf("expected error %q, got err=%v val=%v", tt.wantedErr, err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			s, ok := got.AsString(rta)
			if !ok {
				t.Fatalf("expected string result, got %s", got.TypeName(rta))
			}
			if s != tt.want {
				t.Fatalf("got %q, want %q", s, tt.want)
			}
		})
	}
}
