package benchmarks

import (
	"testing"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/vm"
)

func BenchmarkVM(b *testing.B) {
	src := []byte(`
out = 0
for i := 0; i < 1000; i++ {
    func(x) {
        out += x
    }(i)
}
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
