package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/vm"
)

type Args = map[string]core.Value

type tc struct {
	name string
	src  string
	args Args
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
out = fib(N)
`, args: Args{"N": core.IntValue(20)}},

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
out = fib(N, 0, 1)
`, args: Args{"N": core.IntValue(20)}},

	{
		name: "sumPow1",
		src: `
out = 0
for e in range(1, N, 1) {
	out = out + e * e
}
`, args: Args{"N": core.IntValue(10000)}},

	{
		name: "sumPow2",
		src: `
out = range(1, N, 1).array().reduce(0, (a, b) => a + b * b)
`, args: Args{"N": core.IntValue(10000)}},

	{
		name: "closures",
		src: `
out = 0
for i := 0; i < N; i++ {
    func(x) {
        out += x
    }(i)
}
`, args: Args{"N": core.IntValue(1000)}},

	{
		name: "iter",
		src: `
s := range(0, N, 1).array()
out = 0
for i := 0; i < len(s); i++ {
    for j := 0; j < len(s); j++ {
        s[j] += s[i]
		out += 1
    }
}
`, args: Args{"N": core.IntValue(100)}},

	{
		name: "str1",
		src: `
for l := 0; l < N1; l++ {
	x := range(1, N2, 1).array().map(e => "num" + e)
	if l%2 == 0 {
		x = x.map(e => e.lower())
	} else {
		x = x.map(e => e.upper())
	}
	out = x[l]
}
`, args: Args{"N1": core.IntValue(10), "N2": core.IntValue(1000)}},

	{
		name: "str2",
		src: `
text := import("text")
size := N
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
`, args: Args{"N": core.IntValue(100)}},

	{
		name: "decimals",
		src: `
out = decimal(0)
for i := 0; i < N; i++ {
	out = out + 1.0d / decimal(i + 1)
}
`, args: Args{"N": core.IntValue(1000)}},

	{
		name: "arena",
		src: `
out = 0
for i := 0; i < N; i++ {
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
`, args: Args{"N": core.IntValue(900)}},
}

const (
	runWarmup   = 10
	runMeasured = 100

	compileWarmup   = 3
	compileMeasured = 10

	regressionThreshold = 0.03

	baselineFile = "bench-baseline.json"
	currentFile  = "bench-current.json"

	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorReset = "\033[0m"
)

type metrics struct {
	Name       string  `json:"name"`
	CompileAvg float64 `json:"compile_avg"`
	CompileMin float64 `json:"compile_min"`
	RunAvg     float64 `json:"run_avg"`
	RunMin     float64 `json:"run_min"`
	Result     string  `json:"result"`
}

func main() {
	baseline := loadBaseline(baselineFile)
	current := make([]metrics, 0, len(tests))

	fmt.Printf("%-15s %-14s %-14s %-14s %-10s\n", "Test", "Compile-avg", "Run-avg(s)", "Run-min(s)", "Delta(min)")
	fmt.Printf("%-15s %-14s %-14s %-14s %-10s\n", "----", "-----------", "----------", "----------", "----------")

	for _, t := range tests {
		m, err := runBench(t)
		if err != nil {
			panic(err)
		}
		current = append(current, m)
		b, hasBaseline := baseline[m.Name]
		printRow(m, b, hasBaseline)
	}

	saveMetrics(currentFile, current)
}

func runBench(t tc) (metrics, error) {
	input := []byte(t.src)
	args := make([]string, 0, 1+len(t.args))
	args = append(args, "out")
	for k := range t.args {
		args = append(args, k)
	}

	var compileDurations []time.Duration
	var compiled *kavun.Compiled
	for i := 0; i < compileWarmup+compileMeasured; i++ {
		script := kavun.NewScript(input, args...)
		start := time.Now()
		c, err := script.Compile()
		d := time.Since(start)
		if err != nil {
			return metrics{}, err
		}
		if i >= compileWarmup {
			compileDurations = append(compileDurations, d)
		}
		compiled = c
	}

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	var runDurations []time.Duration
	for i := 0; i < runWarmup+runMeasured; i++ {
		compiled.Reset()
		for k, v := range t.args {
			compiled.Set(k, v)
		}
		start := time.Now()
		err := compiled.Run(machine)
		d := time.Since(start)
		if err != nil {
			return metrics{}, err
		}
		if i >= runWarmup {
			runDurations = append(runDurations, d)
		}
	}

	v, err := compiled.Get("out")
	if err != nil {
		return metrics{}, err
	}

	cAvg, cMin := stats(compileDurations)
	rAvg, rMin := stats(runDurations)

	return metrics{
		Name:       t.name,
		CompileAvg: cAvg.Seconds(),
		CompileMin: cMin.Seconds(),
		RunAvg:     rAvg.Seconds(),
		RunMin:     rMin.Seconds(),
		Result:     v.String(),
	}, nil
}

func stats(ds []time.Duration) (avg, min time.Duration) {
	if len(ds) == 0 {
		return 0, 0
	}
	min = ds[0]
	var sum time.Duration
	for _, d := range ds {
		sum += d
		if d < min {
			min = d
		}
	}
	avg = sum / time.Duration(len(ds))
	return
}

func printRow(m metrics, b metrics, hasBaseline bool) {
	diffText := "NEW"
	diffColor := ""

	if hasBaseline {
		if b.Result != m.Result {
			panic(fmt.Sprintf("Result mismatch for test %s: baseline=%s, current=%s", m.Name, b.Result, m.Result))
		}

		if b.RunMin > 0 {
			diff := (m.RunMin - b.RunMin) / b.RunMin
			diffText = fmt.Sprintf("%+.2f%%", diff*100)
			if math.Abs(diff) > regressionThreshold {
				if diff > 0 {
					diffColor = colorRed
				} else {
					diffColor = colorGreen
				}
			}
		}
	}

	fmt.Printf("%-15s %-14.9f %-14.9f %-14.9f %s\n", m.Name, m.CompileAvg, m.RunAvg, m.RunMin, colorize(diffText, 10, diffColor))
}

func colorize(text string, width int, color string) string {
	padded := fmt.Sprintf("%-*s", width, text)
	if color == "" || os.Getenv("NO_COLOR") != "" {
		return padded
	}
	return color + padded + colorReset
}

func loadBaseline(path string) map[string]metrics {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]metrics{}
	}
	var list []metrics
	if err := json.Unmarshal(data, &list); err != nil {
		return map[string]metrics{}
	}
	out := make(map[string]metrics, len(list))
	for _, m := range list {
		out[m.Name] = m
	}
	return out
}

func saveMetrics(path string, list []metrics) {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		panic(err)
	}
}
