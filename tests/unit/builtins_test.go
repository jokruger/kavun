package unit

import (
	"testing"

	"github.com/jokruger/gs/core"
	mock "github.com/jokruger/gs/tests"
	"github.com/jokruger/gs/vm"
)

func Test_builtinDelete(t *testing.T) {
	var builtinDelete core.Value
	for _, f := range vm.BuiltinFuncs {
		if core.ToBuiltinFunction(f).Name == "delete" {
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
		if core.ToBuiltinFunction(f).Name == "splice" {
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
		if core.ToBuiltinFunction(f).Name == "range" {
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
				got, err = got.MethodCall(mock.Vm, "to_array", nil)
				if err != nil {
					t.Errorf("builtinRange() to_array error = %s", err.Error())
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
