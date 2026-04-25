package benchmark

import (
	"testing"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/alloc"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/vm"
)

func BenchmarkVM(b *testing.B) {
	//src := []byte(`out = range(1, 10000, 1).to_array().reduce(0, (a, b) => a + b * b)`)

	src := []byte(`
out = decimal(0)
for i := 0; i < 1000; i++ {
	out = out + decimal(i)
}
`)

	a := alloc.NewArena()
	astFile, err := parse(src)
	if err != nil {
		b.Fatal(err)
	}
	bytecode, err := compileFile(a, astFile)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("vmRun", func(b *testing.B) {
		var err error

		for i := 0; i < b.N; i++ {
			_, err = runVM(a, bytecode)
		}

		if err != nil {
			b.Fatal(err)
		}
	})
}

func parse(input []byte) (*parser.File, error) {
	fileSet := parser.NewFileSet()
	inputFile := fileSet.AddFile("bench", -1, len(input))
	p := parser.NewParser(inputFile, input, nil)
	return p.ParseFile()
}

func compileFile(a core.Allocator, file *parser.File) (*vm.Bytecode, error) {
	symTable := vm.NewSymbolTable()
	symTable.Define("out")
	m := stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	c := kavun.NewCompiler(a, file.InputFile, symTable, nil, m, nil)
	if err := c.Compile(file); err != nil {
		return nil, err
	}
	bytecode := c.Bytecode()
	bytecode.RemoveDuplicates()
	return bytecode, nil
}

func runVM(a core.Allocator, bytecode *vm.Bytecode) (core.Value, error) {
	globals := make([]core.Value, vm.GlobalsSize)

	a.Reset()
	v := vm.NewVM(a, bytecode, globals)
	if err := v.Run(); err != nil {
		return core.Undefined, err
	}

	return globals[0], nil
}
