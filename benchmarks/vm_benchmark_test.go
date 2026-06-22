package benchmarks

import (
	"testing"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/vm"
)

func BenchmarkVM(b *testing.B) {
	src := []byte(`
fib := func(x) {
	if x == 0 {
		return 0
	} else if x == 1 {
		return 1
	}
	return fib(x-1) + fib(x-2)
}
out = fib(20)
`)

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	script := kavun.NewScript(src, "out")
	compiled, err := script.Compile()
	if err != nil {
		b.Fatal(err)
	}

	b.Run("vmRun", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			compiled.Reset()
			if err := compiled.Run(machine); err != nil {
				b.Fatal(err)
			}
		}
	})
}
