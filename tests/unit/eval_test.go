package unit

import (
	"context"
	"testing"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/tests/require"
)

func TestEval(t *testing.T) {
	alloc := core.NewArena(nil)

	eval := func(expr string, params map[string]any, expected any) {
		e, err := require.FromInterface(alloc, expected)
		require.NoError(t, err)
		ctx := context.Background()
		ps := make(map[string]core.Value)
		for k, v := range params {
			o, err := require.FromInterface(alloc, v)
			require.NoError(t, err)
			ps[k] = o
		}
		actual, err := kavun.Eval(ctx, expr, ps)
		require.NoError(t, err)
		require.Equal(t, e, actual)
	}

	eval(`undefined`, nil, nil)
	eval(`1`, nil, int64(1))
	eval(`19 + 23`, nil, int64(42))
	eval(`"foo bar"`, nil, "foo bar")
	eval(`[1, 2, 3][1]`, nil, int64(2))

	eval(
		`5 + p`,
		map[string]any{
			"p": 7,
		},
		int64(12),
	)
	eval(
		`"seven is " + p`,
		map[string]any{
			"p": 7,
		},
		"seven is 7",
	)
	eval(
		`"" + a + b`,
		map[string]any{
			"a": 7,
			"b": " is seven",
		},
		"7 is seven",
	)

	eval(
		`a ? "success" : "fail"`,
		map[string]any{
			"a": 1,
		},
		"success",
	)

	// f-strings
	eval(`f""`, nil, "")
	eval(`f"hello"`, nil, "hello")
	eval(`f"hello, {name}"`, map[string]any{"name": "world"}, "hello, world")
	eval(`f"a={x} b={y}"`, map[string]any{"x": 1, "y": 2}, "a=1 b=2")
	eval(`f"pi={pi:.2f}"`, map[string]any{"pi": 3.14159}, "pi=3.14")
	eval(`f"int={n:5d}"`, map[string]any{"n": 42}, "int=   42")
	eval(`f"{{literal}}"`, nil, "{literal}")
	eval(`f"{x+y}"`, map[string]any{"x": 1, "y": 2}, "3")
	eval(`f"{x:}{y:}"`, map[string]any{"x": 1, "y": 2}, "12")
	eval(`f"prefix"`, nil, "prefix")
	eval(`f"{n}!"`, map[string]any{"n": 42}, "42!")
	eval(`f"{a}-{b}-{c}"`, map[string]any{"a": 1, "b": 2, "c": 3}, "1-2-3")
}
