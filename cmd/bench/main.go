package main

import (
	"fmt"
	"time"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/vm"
)

type tc struct {
	name string
	src  string
}

var tests = []tc{
	{
		name: "fib1(20)",
		src: `
fib := func(x) {
	if x == 0 {
		return 0
	} else if x == 1 {
		return 1
	}
	return fib(x-1) + fib(x-2)
}
out = fib(20)
`},

	{
		name: "fib2(20)",
		src: `
fib := func(x, a, b) {
	if x == 0 {
		return a
	} else if x == 1 {
		return b
	}
	return fib(x-1, b, a+b)
}
out = fib(20, 0, 1)
`},

	{
		name: "sumPow1",
		src: `
out = 0
for e in range(1, 10000, 1) {
	out = out + e * e
}
`},

	{
		name: "sumPow2",
		src: `
out = range(1, 10000, 1).array().reduce(0, (a, b) => a + b * b)
`},

	{
		name: "closures",
		src: `
out = 0
for i := 0; i < 1000; i++ {
    func(x) {
        out += x
    }(i)
}
`},

	{
		name: "iter",
		src: `
s := range(0, 100, 1).array()
out = 0
for i := 0; i < len(s); i++ {
    for j := 0; j < len(s); j++ {
        s[j] += s[i]
		out += 1
    }
}
`},

	{
		name: "str1",
		src: `
for l := 0; l < 10; l++ {
	x := range(1, 1000, 1).array().map(e => "num" + e)
	if l%2 == 0 {
		x = x.map(e => e.lower())
	} else {
		x = x.map(e => e.upper())
	}
	out = x[l]
}
`},

	{
		name: "str2",
		src: `
text := import("text")
size := 100
s := ""
for r := 0; r < size*2; r++ {
    if r%2 == 0 {
        s += string(rune(r))
    }
}
n := 0
for r := rune(0); r < size*2; r++ {
    if text.contains(s, r) {
        n++
    }
}
out = n
`},

	{
		name: "decimals",
		src: `
out = decimal(0)
for i := 0; i < 1000; i++ {
	out = out + 1.0d / decimal(i + 1)
}
`},

	{
		name: "arena",
		src: `
out = 0
for i := 0; i < 900; i++ {
    base := i % 200
    xs := [base, base+1, base+2, base+3, base+4, base+5, base+6, base+7]

    for x in xs {
        out += x
    }

    bs := xs.bytes()
    for b in bs {
        out += int(b)
    }
}
`},
}

func main() {
	fmt.Printf("%-15s %-25s %-15s %-15s %-15s\n", "Test", "Result", "Parse (sec)", "Compile (sec)", "Run (sec)")
	fmt.Printf("%-15s %-25s %-15s %-15s %-15s\n", "----", "------", "-----------", "-------------", "---------")
	for _, t := range tests {
		compileTime, runTime, res, err := runBench([]byte(t.src))
		if err != nil {
			panic(err)
		}
		fmt.Printf("%-15s %-25s %-15f %-15f\n", t.name, res.String(), compileTime.Seconds(), runTime.Seconds())
	}
}

func runBench(input []byte) (compileTime time.Duration, runTime time.Duration, result core.Value, err error) {
	var compiled *kavun.Compiled // placeholder for compiled script
	cta := core.NewArena(nil)    // compile time arena
	rta := core.NewArena(nil)    // run time arena
	//cta := core.NewArena(&core.ArenaOptions{})
	//rta := core.NewArena(&core.ArenaOptions{})
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize) // virtual machine

	start := time.Now()
	script := kavun.NewScript(input)
	script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))
	script.Add("out", core.Undefined)
	compiled, err = script.Compile(cta)
	if err != nil {
		return
	}
	compileTime = time.Since(start)

	start = time.Now()
	for range 100 {
		if err = compiled.Run(rta, machine); err != nil {
			return
		}
	}
	runTime = time.Since(start)
	result = compiled.GetValue("out")
	return
}
