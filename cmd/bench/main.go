package main

import (
	"fmt"
	"time"

	"github.com/jokruger/gs"
	"github.com/jokruger/gs/alloc"
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/stdlib"
	"github.com/jokruger/gs/vm"
)

type tc struct {
	name string
	src  string
}

var tests = []tc{
	{
		name: "fib1(30)/10",
		src: `
fib := func(x) {
	if x == 0 {
		return 0
	} else if x == 1 {
		return 1
	}
	return fib(x-1) + fib(x-2)
}
for i := 0; i < 10; i++ {
	out = fib(30)
}
`},

	{
		name: "fib2(30)/10000",
		src: `
fib := func(x, a, b) {
	if x == 0 {
		return a
	} else if x == 1 {
		return b
	}
	return fib(x-1, b, a+b)
}
for i := 0; i < 10000; i++ {
	out = fib(30, 0, 1)
}
`},

	{
		name: "sumPow1/100",
		src: `
for l := 0; l < 100; l++ {
	out = 0
	for e in range(1, 10000, 1) {
		out = out + e * e
	}
}
`},

	{
		name: "sumPow2/100",
		src: `
for l := 0; l < 100; l++ {
	out = range(1, 10000, 1).to_array().reduce(0, (a, b) => a + b * b)
}
`},

	{
		name: "closures/100000",
		src: `
out = 0
for i := 0; i < 100000; i++ {
    func(x) {
        out += x
    }(i)
}
`},

	{
		name: "iter/1000",
		src: `
s := range(0, 1000, 1).to_array()
out = 0
for i := 0; i < len(s); i++ {
    for j := 0; j < len(s); j++ {
        s[j] += s[i]
		out += 1
    }
}
`},

	{
		name: "str1/1000",
		src: `
for l := 0; l < 1000; l++ {
	x := range(1, 1000, 1).to_array().map(e => "num" + e)
	if l%2 == 0 {
		x = x.map(e => e.lower())
	} else {
		x = x.map(e => e.upper())
	}
	out = x[l]
}
`},

	{
		name: "str2/1000",
		src: `
text := import("text")
size := 1000
s := ""
for r := 0; r < size*2; r++ {
    if r%2 == 0 {
        s += string(char(r))
    }
}
n := 0
for r := char(0); r < size*2; r++ {
    if text.contains(s, r) {
        n++
    }
}
out = n
`},
}

func main() {
	fmt.Printf("%-15s %-25s %-15s %-15s %-15s\n", "Test", "Result", "Parse (sec)", "Compile (sec)", "Run (sec)")
	fmt.Printf("%-15s %-25s %-15s %-15s %-15s\n", "----", "------", "-----------", "-------------", "---------")
	for _, t := range tests {
		a := alloc.New()
		parseTime, compileTime, runTime, res, err := runBench(a, []byte(t.src))
		if err != nil {
			panic(err)
		}
		fmt.Printf("%-15s %-25s %-15f %-15f %-15f\n", t.name, res.String(), parseTime.Seconds(), compileTime.Seconds(), runTime.Seconds())
	}
}

func runBench(a core.Allocator, input []byte) (parseTime time.Duration, compileTime time.Duration, runTime time.Duration, result core.Value, err error) {
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

	m := stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	c := gs.NewCompiler(a, file.InputFile, symTable, nil, m, nil)
	if err := c.Compile(file); err != nil {
		return time.Since(start), nil, err
	}

	bytecode := c.Bytecode()
	bytecode.RemoveDuplicates()

	return time.Since(start), bytecode, nil
}

func runVM(a core.Allocator, bytecode *vm.Bytecode) (time.Duration, core.Value, error) {
	globals := make([]core.Value, vm.GlobalsSize)

	start := time.Now()

	v := vm.NewVM(a, bytecode, globals, -1)
	if err := v.Run(); err != nil {
		return time.Since(start), core.UndefinedValue(), err
	}

	return time.Since(start), globals[0], nil
}
