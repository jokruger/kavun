package benchmark

import (
	"testing"

	"github.com/jokruger/gs"
	"github.com/jokruger/gs/alloc"
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/stdlib"
	"github.com/jokruger/gs/vm"
)

func BenchmarkVM(b *testing.B) {
	//src := []byte(`out = range(1, 10000, 1).to_array().reduce(0, (a, b) => a + b * b)`)

	src := []byte(`
foo := func (x) {
	y := 0
	for i := 1; i < x; i++ {
		if !(x % 2 == 0) {
			y = y + i
		} else {
			y = y - i
		}
	}
	return y
}

x := 1
for i := 1; i < 1000; i++ {
	x = x + i - foo(i)
}
`)

	a := alloc.New(0)
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
	c := gs.NewCompiler(a, file.InputFile, symTable, nil, m, nil)
	if err := c.Compile(file); err != nil {
		return nil, err
	}
	bytecode := c.Bytecode()
	bytecode.RemoveDuplicates()
	return bytecode, nil
}

func runVM(a core.Allocator, bytecode *vm.Bytecode) (core.Value, error) {
	globals := make([]core.Value, vm.GlobalsSize)

	v := vm.NewVM(a, bytecode, globals)
	if err := v.Run(); err != nil {
		return core.Undefined, err
	}

	return globals[0], nil
}
