package kavun

import (
	"context"
	"fmt"
	"strings"

	"github.com/jokruger/kavun/core"
)

// Eval compiles and executes given expr with params, and returns an evaluated value.
// Argument `expr` must be an expression. Otherwise it will fail to compile.
// Expression must not use or define variable "__res__" as it's reserved for the internal usage.
func Eval(ctx context.Context, alloc *core.Arena, expr string, params map[string]core.Value) (any, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty expression")
	}

	script := NewScript(alloc, []byte(fmt.Sprintf("__res__ := (%s)", expr)))
	for pk, pv := range params {
		script.Add(pk, pv)
	}

	compiled, err := script.RunContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("script run: %w", err)
	}

	return compiled.Get("__res__").Value(), nil
}
