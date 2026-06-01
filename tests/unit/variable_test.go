package unit

import (
	"testing"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/tests/require"
)

type VariableTest struct {
	Name        string
	Value       any
	ValueType   string
	IntValue    int64
	FloatValue  float64
	CharValue   rune
	BoolValue   bool
	StringValue string
	Object      core.Value
	IsUndefined bool
}

func TestVariable(t *testing.T) {
	vars := []VariableTest{
		{
			Name:        "a",
			Value:       int64(1),
			ValueType:   "int",
			IntValue:    1,
			FloatValue:  1.0,
			CharValue:   rune(1),
			BoolValue:   true,
			StringValue: "1",
			Object:      core.IntValue(1),
		},
		{
			Name:        "b",
			Value:       "52.11",
			ValueType:   "string",
			FloatValue:  52.11,
			StringValue: "52.11",
			BoolValue:   false, // cannot be parsed as a boolean, default to false
			Object:      core.NewStringValue("52.11"),
		},
		{
			Name:        "c",
			Value:       true,
			ValueType:   "bool",
			IntValue:    1,
			FloatValue:  0,
			BoolValue:   true,
			StringValue: "true",
			Object:      core.True,
		},
		{
			Name:        "d",
			Value:       nil,
			ValueType:   "undefined",
			Object:      core.Undefined,
			IsUndefined: true,
		},
	}

	for _, tc := range vars {
		o, err := require.FromInterface(rta, tc.Value)
		require.NoError(t, err)

		v := kavun.NewVariable(tc.Name, o)
		val := v.Value()
		require.Equal(t, rta, tc.Value, val.Interface(rta), "Name: %s", tc.Name)
		require.Equal(t, rta, tc.ValueType, v.ValueType(rta), "Name: %s", tc.Name)
		require.Equal(t, rta, tc.IntValue, v.Int(rta), "Name: %s", tc.Name)
		require.Equal(t, rta, tc.FloatValue, v.Float(rta), "Name: %s", tc.Name)
		require.Equal(t, rta, tc.CharValue, v.Rune(rta), "Name: %s", tc.Name)
		require.Equal(t, rta, tc.BoolValue, v.Bool(rta), "Name: %s", tc.Name)
		require.Equal(t, rta, tc.StringValue, v.String(rta), "Name: %s", tc.Name)
		require.Equal(t, rta, tc.Object, v.Object(), "Name: %s", tc.Name)
		require.Equal(t, rta, tc.IsUndefined, v.IsUndefined(), "Name: %s", tc.Name)
	}
}
