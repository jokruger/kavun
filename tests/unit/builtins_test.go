package gs_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
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
		wantErr   bool
		wantedErr error
		target    any
	}{
		{name: "invalid-arg", args: args{[]core.Object{value.NewString(""),
			value.NewString("")}}, wantErr: true,
			wantedErr: gse.ErrInvalidArgumentType{Name: "first", Expected: "map", Found: "string"},
		},
		{name: "no-args",
			wantErr: true, wantedErr: gse.ErrWrongNumArguments},
		{name: "empty-args", args: args{[]core.Object{}}, wantErr: true,
			wantedErr: gse.ErrWrongNumArguments,
		},
		{name: "3-args", args: args{[]core.Object{
			(*value.Map)(nil), (*value.String)(nil), (*value.String)(nil)}},
			wantErr: true, wantedErr: gse.ErrWrongNumArguments,
		},
		{name: "nil-map-empty-key",
			args: args{[]core.Object{value.NewMap(nil, false), value.NewString("")}},
			want: value.UndefinedValue,
		},
		//{name: "nil-map-nonstr-key",
		//	args: args{[]core.Object{
		//		value.NewMap(nil, false), value.NewInt(0)}}, wantErr: true,
		//	wantedErr: gse.ErrInvalidArgumentType{
		//		Name: "second", Expected: "string", Found: "int"},
		//},
		{name: "nil-map-no-key",
			args: args{[]core.Object{value.NewMap(nil, false)}}, wantErr: true,
			wantedErr: gse.ErrWrongNumArguments,
		},
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
			if (err != nil) != tt.wantErr {
				t.Errorf("builtinDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !errors.Is(err, tt.wantedErr) {
				if err.Error() != tt.wantedErr.Error() {
					t.Errorf("builtinDelete() error = %v, wantedErr %v", err, tt.wantedErr)
					return
				}
			}
			if got != tt.want {
				t.Errorf("builtinDelete() = %v, want %v", got, tt.want)
				return
			}
			if !tt.wantErr && tt.target != nil {
				switch v := tt.args.args[0].(type) {
				case *value.Map, *value.Array:
					if !reflect.DeepEqual(tt.target, tt.args.args[0]) {
						t.Errorf("builtinDelete() objects are not equal "+
							"got: %+v, want: %+v", tt.args.args[0], tt.target)
					}
				default:
					t.Errorf("builtinDelete() unsuporrted arg[0] type %s",
						v.TypeName())
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
		wantErr   bool
		wantedErr error
	}{
		{name: "no args", args: []core.Object{}, wantErr: true,
			wantedErr: gse.ErrWrongNumArguments,
		},
		{name: "invalid args", args: []core.Object{value.NewMap(nil, false)},
			wantErr: true,
			wantedErr: gse.ErrInvalidArgumentType{
				Name: "first", Expected: "array", Found: "map"},
		},
		{name: "invalid args",
			args:    []core.Object{value.NewArray(nil, false), value.NewString("")},
			wantErr: true,
			wantedErr: gse.ErrInvalidArgumentType{
				Name: "second", Expected: "int", Found: "string"},
		},
		{name: "negative index",
			args:      []core.Object{value.NewArray(nil, false), value.NewInt(-1)},
			wantErr:   true,
			wantedErr: gse.ErrIndexOutOfBounds},
		{name: "non int count",
			args: []core.Object{
				value.NewArray(nil, false),
				value.NewInt(0),
				value.NewString(""),
			},
			wantErr: true,
			wantedErr: gse.ErrInvalidArgumentType{
				Name: "third", Expected: "int", Found: "string"},
		},
		{name: "negative count",
			args: []core.Object{
				value.NewArray([]core.Object{value.NewInt(0), value.NewInt(1), value.NewInt(2)}, false),
				value.NewInt(0),
				value.NewInt(-1),
			},
			wantErr:   true,
			wantedErr: gse.ErrIndexOutOfBounds,
		},
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
			if (err != nil) != tt.wantErr {
				t.Errorf("builtinSplice() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.deleted) {
				t.Errorf("builtinSplice() = %v, want %v", got, tt.deleted)
			}
			if tt.wantErr && tt.wantedErr.Error() != err.Error() {
				t.Errorf("builtinSplice() error = %v, wantedErr %v",
					err, tt.wantedErr)
			}
			if tt.Array != nil && !reflect.DeepEqual(tt.Array, tt.args[0]) {
				t.Errorf("builtinSplice() arrays are not equal expected"+
					" %s, got %s", tt.Array, tt.args[0].(*value.Array))
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
		wantErr   bool
		wantedErr error
	}{
		{name: "no args", args: []core.Object{}, wantErr: true,
			wantedErr: gse.ErrWrongNumArguments,
		},
		{name: "single args", args: []core.Object{value.NewMap(nil, false)},
			wantErr:   true,
			wantedErr: gse.ErrWrongNumArguments,
		},
		{name: "4 args", args: []core.Object{value.NewMap(nil, false), value.NewString(""), value.NewString(""), value.NewString("")},
			wantErr:   true,
			wantedErr: gse.ErrWrongNumArguments,
		},
		{name: "invalid start",
			args:    []core.Object{value.NewString(""), value.NewString("")},
			wantErr: true,
			wantedErr: gse.ErrInvalidArgumentType{
				Name: "start", Expected: "int", Found: "string"},
		},
		{name: "invalid stop",
			args:    []core.Object{value.NewInt(0), value.NewString("")},
			wantErr: true,
			wantedErr: gse.ErrInvalidArgumentType{
				Name: "stop", Expected: "int", Found: "string"},
		},
		{name: "invalid step",
			args:    []core.Object{value.NewInt(0), value.NewInt(0), value.NewString("")},
			wantErr: true,
			wantedErr: gse.ErrInvalidArgumentType{
				Name: "step", Expected: "int", Found: "string"},
		},
		{name: "zero step",
			args:      []core.Object{value.NewInt(0), value.NewInt(0), value.NewInt(0)}, //must greate than 0
			wantErr:   true,
			wantedErr: gse.ErrInvalidRangeStep,
		},
		{name: "negative step",
			args:      []core.Object{value.NewInt(0), value.NewInt(0), intObject(-2)}, //must greate than 0
			wantErr:   true,
			wantedErr: gse.ErrInvalidRangeStep,
		},
		{name: "same bound",
			args:    []core.Object{value.NewInt(0), value.NewInt(0)},
			wantErr: false,
			result:  value.NewArray(nil, false),
		},
		{name: "positive range",
			args:    []core.Object{value.NewInt(0), value.NewInt(5)},
			wantErr: false,
			result: value.NewArray([]core.Object{
				intObject(0),
				intObject(1),
				intObject(2),
				intObject(3),
				intObject(4),
			}, false),
		},
		{name: "negative range",
			args:    []core.Object{value.NewInt(0), value.NewInt(-5)},
			wantErr: false,
			result: value.NewArray([]core.Object{
				intObject(0),
				intObject(-1),
				intObject(-2),
				intObject(-3),
				intObject(-4),
			}, false),
		},

		{name: "positive with step",
			args:    []core.Object{value.NewInt(0), value.NewInt(5), value.NewInt(2)},
			wantErr: false,
			result: value.NewArray([]core.Object{
				intObject(0),
				intObject(2),
				intObject(4),
			}, false),
		},

		{name: "negative with step",
			args:    []core.Object{value.NewInt(0), value.NewInt(-10), value.NewInt(2)},
			wantErr: false,
			result: value.NewArray([]core.Object{
				intObject(0),
				intObject(-2),
				intObject(-4),
				intObject(-6),
				intObject(-8),
			}, false),
		},

		{name: "large range",
			args:    []core.Object{intObject(-10), intObject(10), value.NewInt(3)},
			wantErr: false,
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
			if (err != nil) != tt.wantErr {
				t.Errorf("builtinRange() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.wantedErr.Error() != err.Error() {
				t.Errorf("builtinRange() error = %v, wantedErr %v",
					err, tt.wantedErr)
			}
			if tt.result != nil && !reflect.DeepEqual(tt.result, got) {
				t.Errorf("builtinRange() arrays are not equal expected"+
					" %s, got %s", tt.result, got.(*value.Array))
			}
		})
	}
}
