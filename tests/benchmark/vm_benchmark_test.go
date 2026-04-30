package benchmark

import (
	"testing"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/vm"
)

func BenchmarkVM(b *testing.B) {
	//src := []byte(`out = range(1, 10000, 1).array().reduce(0, (a, b) => a + b * b)`)

	src := []byte(`
out = decimal(0)
for i := 0; i < 100; i++ {
	out = out + 1d / decimal(i + 1)
}
`)

	cta := core.NewArena(nil)
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	script := kavun.NewScript(src)
	compiled, err := script.Compile(cta)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("vmRun", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := compiled.Run(rta, machine); err != nil {
				b.Fatal(err)
			}
		}
	})
}
