package kavun

import (
	"context"
	"fmt"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/vm"
)

// Compiled is a compiled instance of the user script. Use Script.Compile() to create Compiled object.
type Compiled struct {
	bytecode *vm.Bytecode
	index    map[string]int // global variable names
	globals  []core.Value   // global variable values - must be set before each execution
}

// Bind binds the compiled bytecode to the arena. This must be called before running the script or resolving the values,
// and the same arena must be used for allocating any variable values that are set on this Compiled instance.
func (c *Compiled) Bind(a *core.Arena) {
	c.bytecode.Bind(a)
}

// Reset sets all global variable values to Undefined.
func (c *Compiled) Reset() {
	for i := range c.globals {
		c.globals[i] = core.Undefined
	}
}

// Set sets a variable identified by the name to the value. Values must be allocated using same arena.
func (c *Compiled) Set(name string, val core.Value) error {
	i, ok := c.index[name]
	if !ok {
		return fmt.Errorf("variable %s is not defined", name)
	}
	c.globals[i] = val
	return nil
}

// Get returns the value of a variable identified by the name. If the variable is not defined, it returns Undefined.
func (c *Compiled) Get(name string) core.Value {
	v := core.Undefined
	if i, ok := c.index[name]; ok {
		v = c.globals[i]
	}
	return v
}

// GetAll returns a map of all global variable names to their values.
func (c *Compiled) GetAll() map[string]core.Value {
	result := make(map[string]core.Value, len(c.index))
	for name, i := range c.index {
		result[name] = c.globals[i]
	}
	return result
}

// Run binds the script to the provided arena and executes it in the provided virtual machine.
// It is the caller's responsibility to reset arena and set all global variables to new values before calling Run, and
// ensure that same arena is used for allocating each variable value.
func (c *Compiled) Run(v *vm.VM) error {
	v.Reset(a, c.bytecode, c.globals)
	return v.Run()
}

// Run executes the compiled script in the provided virtual machine with a context for cancellation.
// It is the caller's responsibility to reset arena and set all global variables to new values before calling Run, and
// ensure that same arena is used for allocating each variable value.
func (c *Compiled) RunContext(ctx context.Context, a *core.Arena, v *vm.VM) (err error) {
	v.Reset(a, c.bytecode, c.globals)

	ch := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				switch e := r.(type) {
				case string:
					ch <- fmt.Errorf("%s", e)
				case error:
					ch <- e
				default:
					ch <- fmt.Errorf("unknown panic: %v", e)
				}
			}
		}()
		ch <- v.Run()
	}()

	select {
	case <-ctx.Done():
		v.Abort()
		<-ch
		err = ctx.Err()
	case err = <-ch:
	}

	return
}
