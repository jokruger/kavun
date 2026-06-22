package kavun

import (
	"context"
	"fmt"
	"strings"

	"github.com/jokruger/kavun/vm"
)

// Eval compiles and executes given expr with params, and returns an evaluated value.
// Argument `expr` must be an expression. Otherwise it will fail to compile.
// Expression must not use or define variable "__res__" as it's reserved for the internal usage.
func Eval(ctx context.Context, expr string, params map[string]any) (any, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty expression")
	}

	script := NewScript([]byte(fmt.Sprintf("__res__ := (%s)", expr)))
	for pk := range params {
		script.AddGlobals(pk)
	}

	compiled, err := script.Compile()
	if err != nil {
		return nil, fmt.Errorf("script compile: %w", err)
	}
	for k, v := range params {
		nv, err := ValueOf(v)
		if err != nil {
			return nil, fmt.Errorf("convert param %q: %w", k, err)
		}
		compiled.Set(k, nv)
	}

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	if err := compiled.RunContext(ctx, machine); err != nil {
		return nil, fmt.Errorf("script run: %w", err)
	}
	return compiled.Get("__res__"), nil
}
