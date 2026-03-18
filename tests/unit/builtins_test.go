package gs_test

import (
	"reflect"
	"testing"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
	"github.com/jokruger/gs/vm"
)

func Test_builtinDelete(t *testing.T) {
	var builtinDelete func(args ...core.Object) (core.Object, error)
	for _, f := range vm.GetAllBuiltinFunctions() {
		if f.Name() == "delete" {
			builtinDelete = f.Value()
			break
		}
	}
	if builtinDelete == nil {
		t.Fatal("builtin delete not found")
	}
	type args struct {
		args []core.Object
	}
	tests := []struct {
		name      string
		args      args
		want      core.Object
		wantedErr string
		target    any
	}{
		{name: "invalid-arg", args: args{[]core.Object{value.NewString(""), value.NewString("")}},
			wantedErr: "invalid argument type: delete argument 'first' expects type map, got string"},

		{name: "no-args",
			wantedErr: "wrong number of arguments"},

		{name: "empty-args", args: args{[]core.Object{}},
			wantedErr: "wrong number of arguments"},

		{name: "3-args", args: args{[]core.Object{(*value.Map)(nil), (*value.String)(nil), (*value.String)(nil)}},
			wantedErr: "wrong number of arguments"},

		{name: "nil-map-no-key", args: args{[]core.Object{value.NewMap(nil, false)}},
			wantedErr: "wrong number of arguments"},

		{name: "map-missing-key",
			args: args{
				[]core.Object{
					value.NewMap(map[string]core.Object{
						"key": value.NewString("value"),
					}, false),
					value.NewString("key1")}},
			want: value.UndefinedValue,
			target: value.NewMap(map[string]core.Object{
				"key": value.NewString("value"),
			}, false),
		},

		{name: "map-emptied",
			args: args{
				[]core.Object{
					value.NewMap(map[string]core.Object{
						"key": value.NewString("value"),
					}, false),
					value.NewString("key")}},
			want:   value.UndefinedValue,
			target: value.NewMap(map[string]core.Object{}, false),
		},

		{name: "map-multi-keys",
			args: args{
				[]core.Object{
					value.NewMap(map[string]core.Object{
						"key1": value.NewString("value1"),
						"key2": value.NewInt(10),
					}, false),
					value.NewString("key1")}},
			want: value.UndefinedValue,
			target: value.NewMap(map[string]core.Object{
				"key2": value.NewInt(10)}, false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinDelete(tt.args.args...)
			if (err != nil) != (tt.wantedErr != "") {
				t.Errorf("builtinDelete() error = %v, wantedErr %v", err, tt.wantedErr)
				return
			}
			if tt.wantedErr != "" && (err == nil || err.Error() != tt.wantedErr) {
				t.Errorf("builtinDelete() error = %v, wantedErr %v", err, tt.wantedErr)
				return
			}
			if got != tt.want {
				t.Errorf("builtinDelete() = %v, want %v", got, tt.want)
				return
			}
			if tt.wantedErr == "" && tt.target != nil {
				switch v := tt.args.args[0].(type) {
				case *value.Map, *value.Array:
					if !reflect.DeepEqual(tt.target, tt.args.args[0]) {
						t.Errorf("builtinDelete() objects are not equal, got: %+v, want: %+v", tt.args.args[0], tt.target)
					}
				default:
					t.Errorf("builtinDelete() unsupported arg[0] type %s", v.TypeName())
					return
				}
			}
		})
	}
}

func Test_builtinSplice(t *testing.T) {
	var builtinSplice func(args ...core.Object) (core.Object, error)
	for _, f := range vm.GetAllBuiltinFunctions() {
		if f.Name() == "splice" {
			builtinSplice = f.Value()
			break
		}
	}
	if builtinSplice == nil {
		t.Fatal("builtin splice not found")
	}
	tests := []struct {
		name      string
		args      []core.Object
		deleted   core.Object
		Array     *value.Array
		wantedErr string
	}{
		{name: "no args", args: []core.Object{},
			wantedErr: "wrong number of arguments"},

		{name: "invalid args", args: []core.Object{value.NewMap(nil, false)},
			wantedErr: "invalid argument type: splice argument 'first' expects type mutable array, got map"},

		{name: "invalid args", args: []core.Object{value.NewArray(nil, false), value.NewString("")},
			wantedErr: "invalid argument type: splice argument 'second' expects type int, got string"},

		{name: "negative index", args: []core.Object{value.NewArray(nil, false), value.NewInt(-1)},
			wantedErr: "index out of bounds"},

		{name: "non int count",
			args: []core.Object{
				value.NewArray(nil, false),
				value.NewInt(0),
				value.NewString(""),
			},
			wantedErr: "invalid argument type: splice argument 'third' expects type int, got string"},

		{name: "negative count",
			args: []core.Object{
				value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
				value.NewInt(0),
				value.NewInt(-1),
			},
			wantedErr: "index out of bounds"},

		{name: "insert with zero count",
			args: []core.Object{
				value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
				value.NewInt(0),
				value.NewInt(0),
				value.NewString("b"),
			},
			deleted: value.NewArray([]core.Object{}, false),
			Array:   value.NewArray([]core.Object{value.NewString("b"), value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
		},

		{name: "insert",
			args: []core.Object{
				value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
				value.NewInt(1),
				value.NewInt(0),
				value.NewString("c"),
				value.NewString("d"),
			},
			deleted: value.NewArray([]core.Object{}, false),
			Array:   value.NewArray([]core.Object{value.NewInt(0), value.NewString("c"), value.NewString("d"), value.NewInt(1), value.NewInt(2)}, false),
		},

		{name: "insert with zero count",
			args: []core.Object{
				value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
				value.NewInt(1),
				value.NewInt(0),
				value.NewString("c"),
				value.NewString("d"),
			},
			deleted: value.NewArray([]core.Object{}, false),
			Array:   value.NewArray([]core.Object{value.NewInt(0), value.NewString("c"), value.NewString("d"), value.NewInt(1), value.NewInt(2)}, false),
		},

		{name: "insert with delete",
			args: []core.Object{
				value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
				value.NewInt(1),
				value.NewInt(1),
				value.NewString("c"),
				value.NewString("d"),
			},
			deleted: value.NewArray([]core.Object{value.NewInt(1)}, false),
			Array:   value.NewArray([]core.Object{value.NewInt(0), value.NewString("c"), value.NewString("d"), value.NewInt(2)}, false),
		},

		{name: "insert with delete multi",
			args: []core.Object{
				value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
				value.NewInt(1),
				value.NewInt(2),
				value.NewString("c"),
				value.NewString("d"),
			},
			deleted: value.NewArray([]core.Object{value.NewInt(1), value.NewInt(2)}, false),
			Array:   value.NewArray([]core.Object{value.NewInt(0), value.NewString("c"), value.NewString("d")}, false),
		},

		{name: "delete all with positive count",
			args: []core.Object{
				value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
				value.NewInt(0),
				value.NewInt(3),
			},
			deleted: value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
			Array:   value.NewArray([]core.Object{}, false),
		},

		{name: "delete all with big count",
			args: []core.Object{
				value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
				value.NewInt(0),
				value.NewInt(5),
			},
			deleted: value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
			Array:   value.NewArray([]core.Object{}, false),
		},

		{name: "nothing2",
			args:    []core.Object{value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false)},
			Array:   value.NewArray([]core.Object{}, false),
			deleted: value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
		},

		{name: "pop without count",
			args: []core.Object{
				value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
				value.NewInt(2),
			},
			deleted: value.NewArray([]core.Object{value.NewInt(2)}, false),
			Array:   value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1)}, false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinSplice(tt.args...)
			if (err != nil) != (tt.wantedErr != "") {
				t.Errorf("builtinSplice() error = %v, wantErr %v", err, tt.wantedErr)
				return
			}
			if !reflect.DeepEqual(got, tt.deleted) {
				t.Errorf("builtinSplice() = %v, want %v", got, tt.deleted)
			}
			if (tt.wantedErr != "") && tt.wantedErr != err.Error() {
				t.Errorf("builtinSplice() error = %v, wantedErr %v", err, tt.wantedErr)
			}
			if tt.Array != nil && !reflect.DeepEqual(tt.Array, tt.args[0]) {
				t.Errorf("builtinSplice() arrays are not equal, expected %s, got %s", tt.Array, tt.args[0].(*value.Array))
			}
		})
	}
}

func Test_builtinRange(t *testing.T) {
	var builtinRange func(args ...core.Object) (core.Object, error)
	for _, f := range vm.GetAllBuiltinFunctions() {
		if f.Name() == "range" {
			builtinRange = f.Value()
			break
		}
	}
	if builtinRange == nil {
		t.Fatal("builtin range not found")
	}
	tests := []struct {
		name      string
		args      []core.Object
		result    *value.Array
		wantedErr string
	}{
		{name: "no args", args: []core.Object{},
			wantedErr: "wrong number of arguments"},

		{name: "single args", args: []core.Object{value.NewMap(nil, false)},
			wantedErr: "wrong number of arguments"},

		{name: "4 args", args: []core.Object{value.NewMap(nil, false), value.NewString(""), value.NewString(""), value.NewString("")},
			wantedErr: "wrong number of arguments"},

		{name: "invalid start", args: []core.Object{value.NewString(""), value.NewString("")},
			wantedErr: "invalid argument type: range argument 'start' expects type int, got string"},

		{name: "invalid stop", args: []core.Object{value.NewInt(0), value.NewString("")},
			wantedErr: "invalid argument type: range argument 'stop' expects type int, got string"},

		{name: "invalid step", args: []core.Object{value.NewInt(0), value.NewInt(0), value.NewString("")},
			wantedErr: "invalid argument type: range argument 'step' expects type int, got string"},

		{name: "zero step", args: []core.Object{value.NewInt(0), value.NewInt(0), value.NewInt(0)}, //must greate than 0
			wantedErr: "range step must be greater than 0"},

		{name: "negative step", args: []core.Object{value.NewInt(0), value.NewInt(0), intObject(-2)}, //must greate than 0
			wantedErr: "range step must be greater than 0"},

		{name: "same bound", args: []core.Object{value.NewInt(0), value.NewInt(0)},
			result: value.NewArray(nil, false),
		},

		{name: "positive range", args: []core.Object{value.NewInt(0), value.NewInt(5)},
			result: value.NewArray([]core.Object{
				intObject(0),
				intObject(1),
				intObject(2),
				intObject(3),
				intObject(4),
			}, false),
		},

		{name: "negative range", args: []core.Object{value.NewInt(0), value.NewInt(-5)},
			result: value.NewArray([]core.Object{
				intObject(0),
				intObject(-1),
				intObject(-2),
				intObject(-3),
				intObject(-4),
			}, false),
		},

		{name: "positive with step", args: []core.Object{value.NewInt(0), value.NewInt(5), value.NewInt(2)},
			result: value.NewArray([]core.Object{
				intObject(0),
				intObject(2),
				intObject(4),
			}, false),
		},

		{name: "negative with step", args: []core.Object{value.NewInt(0), value.NewInt(-10), value.NewInt(2)},
			result: value.NewArray([]core.Object{
				intObject(0),
				intObject(-2),
				intObject(-4),
				intObject(-6),
				intObject(-8),
			}, false),
		},

		{name: "large range", args: []core.Object{intObject(-10), intObject(10), value.NewInt(3)},
			result: value.NewArray([]core.Object{
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
			got, err := builtinRange(tt.args...)
			if (err != nil) != (tt.wantedErr != "") {
				t.Errorf("builtinRange() error = %v, wantErr %v", err, tt.wantedErr)
				return
			}
			if (tt.wantedErr != "") && tt.wantedErr != err.Error() {
				t.Errorf("builtinRange() error = %v, wantedErr %v", err, tt.wantedErr)
			}
			if tt.result != nil && !reflect.DeepEqual(tt.result, got) {
				t.Errorf("builtinRange() arrays are not equal, expected %s, got %s", tt.result, got.(*value.Array))
			}
		})
	}
}
