# Embedding

## Minimal example

```go
package main

import (
	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/alloc"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/vm"
)

func main() {
	// Inline script source
	src := []byte(`
fmt := import("fmt")
fmt.println("Hello Kavun!")
`)

	// Parse -> AST
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("inline", -1, len(src))
	p := parser.NewParser(srcFile, src, nil)
	file, err := p.ParseFile()
	if err != nil {
		panic(err)
	}

	// Compile -> bytecode
	a := alloc.New(0) // 0 => no allocation cap
	modules := stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	c := kavun.NewCompiler(a, srcFile, nil, nil, modules, nil)
	if err := c.Compile(file); err != nil {
		panic(err)
	}
	bytecode := c.Bytecode()

	// Run in VM
	machine := vm.NewVM(a, bytecode, nil)
	if err := machine.Run(); err != nil {
		panic(err)
	}
}
```

## Runtime components

- Parser (`parser` package): transforms source bytes into AST (`*parser.File`) and reports syntax errors with source positions.
- Compiler (`kavun.NewCompiler`): transforms AST into VM bytecode. It needs allocator, source file metadata, symbol table/constants (optional for simple use), and module getter.
- Allocator (`alloc.New`): controls runtime object allocation. `alloc.New(0)` means effectively unlimited allocations; non-zero can be used as a safety limit.
- VM (`vm.NewVM`): executes compiled bytecode.

`kavun.NewScript` is a higher-level helper around the same pipeline when you prefer convenience over low-level control.

## Imports and host modules

Use a module map to control what `import("...")` can load.

```go
modules := vm.NewModuleMap()

// Add selected stdlib modules only.
modules.AddMap(stdlib.GetModuleMap("math", "json"))

// Add a host builtin module.
modules.AddBuiltinModule("host", map[string]core.Value{
	"answer": core.IntValue(42),
})

// Add a source module from bytes.
modules.AddSourceModule("helpers", []byte(`
export add := func(a, b) { return a + b }
`))
```

For local file imports, enable them on compiler/script and set an import directory.

```go
c.EnableFileImport(true)
c.SetImportDir("./scripts")
// Optional custom extensions, default is .kvn
_ = c.SetImportFileExt(".kvn", ".yb")
```

## User-defined data types

Kavun types are registered via `core.SetValueType`. User-defined type IDs must be greater than or equal to `core.VT_USER_DEFINED`.

```go
package mytypes

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/token"
)

const VT_COUNTER = core.VT_USER_DEFINED + 1

type Counter struct {
	N int64
}

func NewCounterValue(n int64) core.Value {
	return core.Value{Ptr: unsafe.Pointer(&Counter{N: n}), Type: VT_COUNTER}
}

func toCounter(v core.Value) *Counter {
	return (*Counter)(v.Ptr)
}

func init() {
	core.SetValueType(VT_COUNTER, core.ValueType{
		Name:   func(core.Value) string { return "counter" },
		String: func(v core.Value) string { return fmt.Sprintf("Counter(%d)", toCounter(v).N) },
		IsTrue: func(v core.Value) bool { return toCounter(v).N != 0 },
		BinaryOp: func(v core.Value, a *core.Arena, op token.Token, rhs core.Value) (core.Value, error) {
			if op != token.Add || rhs.Type != core.VT_INT {
				return core.Undefined, fmt.Errorf("unsupported op")
			}
			return NewCounterValue(toCounter(v).N + int64(rhs.Data)), nil
		},
	})
}
```

Expose values/functions to scripts through globals (low-level symbol table flow) or the `kavun.Script` API.

```go
s := kavun.NewScript(a, []byte(`out := c + 2`))
s.Add("c", mytypes.NewCounterValue(40))
compiled, err := s.Run()
if err != nil {
	panic(err)
}
fmt.Println(compiled.Get("out").Value().String()) // Counter(42)
```

## Host function injection (CLI-style)

The Kavun CLI REPL injects a host function into globals through a symbol table entry. The same pattern works in embedded apps.

```go
a := alloc.New(0)
modules := stdlib.GetModuleMap(stdlib.AllModuleNames()...)

src := []byte(`print("Hello Kavun!")`)
fileSet := parser.NewFileSet()
srcFile := fileSet.AddFile("inline", -1, len(src))
file, err := parser.NewParser(srcFile, src, nil).ParseFile()
if err != nil {
	panic(err)
}

symbolTable := vm.NewSymbolTable()
for idx, fn := range vm.BuiltinFuncs {
	// Builtins must be pre-registered in the same symbol table.
	symbolTable.DefineBuiltin(idx, (*core.BuiltinFunction)(fn.Ptr).Name)
}

globals := make([]core.Value, vm.GlobalsSize)
printSym := symbolTable.Define("print")
printFn, err := a.NewBuiltinFunctionValue(
	"print",
	func(vm core.VM, args []core.Value) (core.Value, error) {
		for _, arg := range args {
			fmt.Print(arg.String())
		}
		fmt.Println()
		return core.Undefined, nil
	},
	1,
	false,
)
if err != nil {
	panic(err)
}
globals[printSym.Index] = printFn

c := kavun.NewCompiler(a, srcFile, symbolTable, nil, modules, nil)
if err := c.Compile(file); err != nil {
	panic(err)
}

machine := vm.NewVM(a, c.Bytecode(), globals)
if err := machine.Run(); err != nil {
	panic(err)
}
```

## Set variables before run, read variables after run

Use `kavun.Script` when you want host-provided inputs and easy output extraction.

```go
package main

import (
	"fmt"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/alloc"
	"github.com/jokruger/kavun/core"
)

func main() {
	a := alloc.New(0)

	src := []byte(`
sum := x + y
message := "Hello, " + name
`)

	s := kavun.NewScript(a, src)

	// Set globals before execution.
	s.Add("x", core.IntValue(20))
	s.Add("y", core.IntValue(22))
	name, err := a.NewStringValue("Kavun")
	if err != nil {
		panic(err)
	}
	s.Add("name", name)

	compiled, err := s.Run()
	if err != nil {
		panic(err)
	}

	// Read globals after execution.
	sum, _ := compiled.Get("sum").Value().AsInt()
	msg, _ := compiled.Get("message").Value().AsString()
	fmt.Println(sum) // 42
	fmt.Println(msg) // Hello, Kavun
}
```

You can also update a compiled global between runs with `compiled.Set(name, value)`.

```go
// Reuse compiled bytecode with updated inputs.
if err := compiled.Set("x", core.IntValue(50)); err != nil {
	panic(err)
}
if err := compiled.Set("y", core.IntValue(7)); err != nil {
	panic(err)
}
if err := compiled.Run(); err != nil {
	panic(err)
}

sum2, _ := compiled.Get("sum").Value().AsInt()
fmt.Println(sum2) // 57
```

## Practical notes

- `alloc.Allocator` is single-threaded; do not share one allocator across concurrent executions.
- Use the same allocator instance for the full pipeline of one execution unit (parse/compile/run) so values produced during compilation and values allocated at runtime follow the same lifetime model.
- Use separate allocator instances for separate VM instances (especially when running VMs concurrently) to avoid cross-VM ownership and reuse issues.
- `bytecode.RemoveDuplicates()` is optional and mainly a size optimization.
- Keep module exposure explicit (`vm.NewModuleMap`) for sandboxed embedding.