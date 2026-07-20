package kavun

import (
	"context"
	"fmt"
	"maps"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/vm"
)

// Compiled is a compiled instance of the user script. Use Script.Compile() to create Compiled object.
type Compiled struct {
	bytecode *vm.Bytecode
	bindings map[string]int // variable name to index in globals
	globals  []core.Value   // global variables - must be set before each execution
}

// Clone creates a new copy of Compiled.
func (c *Compiled) Clone() (*Compiled, error) {
	clone := &Compiled{
		bytecode: c.bytecode,
		bindings: make(map[string]int, len(c.bindings)),
		globals:  make([]core.Value, len(c.globals)),
	}

	maps.Copy(clone.bindings, c.bindings)
	for i, v := range c.globals {
		t, err := v.Clone()
		if err != nil {
			return nil, err
		}
		clone.globals[i] = t
	}

	return clone, nil
}

// Reset sets all global variable values to Undefined.
func (c *Compiled) Reset() {
	for i := range c.globals {
		c.globals[i] = core.Undefined
	}
}

// Set sets bound variable.
func (c *Compiled) Set(name string, val core.Value) error {
	i, ok := c.bindings[name]
	if !ok {
		return fmt.Errorf("binding for variable '%s' not found", name)
	}
	c.globals[i] = val
	return nil
}

// MustSet sets bound variable. It panics if binding is not found.
func (c *Compiled) MustSet(name string, val core.Value) {
	if err := c.Set(name, val); err != nil {
		panic(err)
	}
}

// Get returns the value of bound variable.
func (c *Compiled) Get(name string) (core.Value, error) {
	if i, ok := c.bindings[name]; ok {
		return c.globals[i], nil
	}
	return core.Undefined, fmt.Errorf("binding for variable '%s' not found", name)
}

// MustGet returns the value of bound variable. It panics if binding is not found.
func (c *Compiled) MustGet(name string) core.Value {
	v, err := c.Get(name)
	if err != nil {
		panic(err)
	}
	return v
}

// GetAll returns a map of all bindings with their values.
func (c *Compiled) GetAll() map[string]core.Value {
	result := make(map[string]core.Value, len(c.bindings))
	for name, i := range c.bindings {
		result[name] = c.globals[i]
	}
	return result
}

// Executes script in the provided virtual machine. It is the caller's responsibility to set all global variables to new
// values before calling Run.
func (c *Compiled) Run(v *vm.VM) error {
	v.Reset(c.bytecode, c.globals)
	return v.Run()
}

// Run executes the compiled script in the provided virtual machine with a context for cancellation. It is the caller's
// responsibility to set all global variables to new values before calling Run.
func (c *Compiled) RunContext(ctx context.Context, v *vm.VM) (err error) {
	v.Reset(c.bytecode, c.globals)

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
