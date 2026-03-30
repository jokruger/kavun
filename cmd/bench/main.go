package main

import (
	"fmt"
	"time"

	"github.com/jokruger/gs"
	"github.com/jokruger/gs/alloc"
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/vm"
)

type tc struct {
	name string
	src  string
}

var tests = []tc{
	{
		name: "fib1(30)",
		src: `
fib := func(x) {
	if x == 0 {
		return 0
	} else if x == 1 {
		return 1
	}
	return fib(x-1) + fib(x-2)
}
out = fib(30)
`},

	{
		name: "fib2(30)",
		src: `
fib := func(x, s) {
	if x == 0 {
		return 0 + s
	} else if x == 1 {
		return 1 + s
	}
	return fib(x-1, fib(x-2, s))
}
out = fib(30, 0)
`},

	{
		name: "fib3(1000)",
		src: `
fib := func(x, a, b) {
	if x == 0 {
		return a
	} else if x == 1 {
		return b
	}
	return fib(x-1, b, a+b)
}
out = fib(1000, 0, 1)
`},

	{
		name: "powSum1",
		src: `
x := range(1, 10000, 1)
out = 0
for e in x {
	out = out + e * e
}
`},

	{
		name: "powSum2",
		src: `
x := range(1, 10000, 1)
for i := 0; i < len(x); i++ {
	x[i] = x[i] * x[i]
}
out = 0
for i := 0; i < len(x); i++ {
	out = out + x[i]
}
`},

	{
		name: "powSum3",
		src: `
x := range(1, 10000, 1)
for i, e in x {
	x[i] = e * e
}
out = 0
for e in x {
	out += e
}
`},

	{
		name: "powSum4",
		src: `
x := range(1, 10000, 1)
out = x.map(e => e * e).reduce(0, (a, b) => a + b)
`},

	{
		name: "powSum5",
		src: `
x := range(1, 10000, 1)
out = x.reduce(0, (a, b) => a + b * b)
`},
}

func main() {
	fmt.Printf("%-15s %-25s %-15s %-15s %-15s\n", "Test", "Result", "Parse (sec)", "Compile (sec)", "Run (sec)")
	fmt.Printf("%-15s %-25s %-15s %-15s %-15s\n", "----", "------", "-----------", "-------------", "---------")
	for _, t := range tests {
		a := alloc.NewHeapAllocator()
		parseTime, compileTime, runTime, res, err := runBench(a, []byte(t.src))
		if err != nil {
			panic(err)
		}
		fmt.Printf("%-15s %-25s %-15f %-15f %-15f\n", t.name, res.String(), parseTime.Seconds(), compileTime.Seconds(), runTime.Seconds())
	}
}

func runBench(a core.Allocator, input []byte) (parseTime time.Duration, compileTime time.Duration, runTime time.Duration, result core.Object, err error) {
	var astFile *parser.File
	parseTime, astFile, err = parse(input)
	if err != nil {
		return
	}

	var bytecode *vm.Bytecode
	compileTime, bytecode, err = compileFile(a, astFile)
	if err != nil {
		return
	}

	runTime, result, err = runVM(a, bytecode)

	return
}

func parse(input []byte) (time.Duration, *parser.File, error) {
	fileSet := parser.NewFileSet()
	inputFile := fileSet.AddFile("bench", -1, len(input))

	start := time.Now()

	p := parser.NewParser(inputFile, input, nil)
	file, err := p.ParseFile()
	if err != nil {
		return time.Since(start), nil, err
	}

	return time.Since(start), file, nil
}

func compileFile(a core.Allocator, file *parser.File) (time.Duration, *vm.Bytecode, error) {
	symTable := vm.NewSymbolTable()
	symTable.Define("out")

	start := time.Now()

	c := gs.NewCompiler(a, file.InputFile, symTable, nil, nil, nil)
	if err := c.Compile(file); err != nil {
		return time.Since(start), nil, err
	}

	bytecode := c.Bytecode()
	bytecode.RemoveDuplicates()

	return time.Since(start), bytecode, nil
}

func runVM(a core.Allocator, bytecode *vm.Bytecode) (time.Duration, core.Object, error) {
	globals := make([]core.Object, vm.GlobalsSize)

	start := time.Now()

	v := vm.NewVM(a, bytecode, globals, -1)
	if err := v.Run(); err != nil {
		return time.Since(start), nil, err
	}

	return time.Since(start), globals[0], nil
}
