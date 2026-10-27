package rand

import (
	crand "crypto/rand"
	"encoding/binary"
	"math"
	"math/bits"
	mrand "math/rand/v2"
	"unsafe"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
)

// rand.generator is a source of pseudo-random (or, from os.rand_generator(), truly random) numbers, owned by the
// rand module and registered by it (core.ModuleBuilder.Type). ONE type over two sources, so anything taking a
// generator (array.shuffle, a user function) accepts either. Every draw is derived here from the source's raw uint64
// stream — not from math/rand's helpers, whose algorithms Go does not promise to keep — so a seed's sequence
// depends only on PCG-DXSM, a specified algorithm, and is the same on every build. Draws move the generator's
// state (localized state, like an iterator's cursor); two generators are equal only when they are the same one.

const generatorTypeName = "rand.generator"

// Generator is the state behind a rand.generator value.
type Generator struct {
	src source
	pcg *mrand.PCG // the seeded source, for copy(); nil for entropy
}

// source is the raw uint64 stream: PCG-DXSM, or crypto/rand.
type source interface {
	Uint64() uint64
}

type entropy struct{}

func (entropy) Uint64() uint64 {
	var b [8]byte
	_, _ = crand.Read(b[:]) // crypto/rand.Read never returns an error (Go 1.24+): it crashes the process instead
	return binary.LittleEndian.Uint64(b[:])
}

// NewSeededValue is a generator seeded with seed: the same seed, the same sequence.
func NewSeededValue(seed int64) core.Value {
	pcg := mrand.NewPCG(uint64(seed), 0)
	return newValue(&Generator{src: pcg, pcg: pcg})
}

// NewEntropyValue is a generator over crypto/rand (os.rand_generator()).
func NewEntropyValue() core.Value {
	return newValue(&Generator{src: entropy{}})
}

func newValue(g *Generator) core.Value {
	return core.Value{Type: value.RandGenerator, Immutable: true, Ptr: unsafe.Pointer(g)}
}

// GeneratorOf unwraps a rand.generator value.
func GeneratorOf(v core.Value) *Generator {
	return (*Generator)(v.Ptr)
}

// A source of random numbers, as `rand.generator(seed)` and `os.rand_generator()` answer it.
//
// ## Overview
//
// `rand.generator` is owned by the [`rand`](../stdlib/rand.md) module. It is ONE type over two sources, so anything
// that takes a generator — `array.shuffle(g)`, a function of your own — accepts either:
//
// | constructor | source | sequence |
// | --- | --- | --- |
// | `rand.generator(seed)` (pure) | PCG-DXSM, seeded | the same for the same seed, on every build |
// | `os.rand_generator()` (needs the `os` grant) | `crypto/rand` | unpredictable |
//
// Every draw is derived from the source's raw 64-bit output by Kavun's own code (bounded ints without bias, 53-bit
// floats, Box–Muller normals, Fisher–Yates permutations), so a seeded sequence depends only on PCG-DXSM — a specified
// algorithm — and not on Go's library helpers. A draw moves the generator's state: it is the value's own, never shared
// with another script.
//
// ```go
// g := import("rand").generator(1)
// g.int(100)                  // => 59 — then 8, then 71: seed 1 always starts this way
// g.int(1, 7)                 // a die roll in [1, 7)
// g.float()                   // [0, 1)
// g.perm(4)                   // a permutation of 0…3
// d := import("os").rand_generator()
// d.int(1, 7)                 // unpredictable
// ```
//
// ## Operators
//
// | operator | reading |
// | --- | --- |
// | `==` / `!=` | the same generator (identity — two generators with the same seed are still two) |
// | `!` | a generator is always truthy |
// | everything else | raises |
//
// ## Ranges
//
// Ranges are half-open, like `range(lo, hi)`.
//
// ## Render
//
// The render names the source, not a constructor expression — no expression rebuilds a state mid-sequence. There is
// no `string()`.
//
//kavun:type rand.generator

// TypeGenerator is the rand.generator type descriptor.
var TypeGenerator = core.ValueTypeDescr{
	Name:   core.ConstHook(generatorTypeName),       // PURE by contract
	String: generatorTypeString,                     // PURE by contract
	Format: core.RenderFormat,                       // PURE by contract
	IsTrue: core.Const2Hook[bool, error](true, nil), // PURE by contract
	Equal:  core.SamePtrEqual,                       // PURE by contract

	Methods: generatorMethods,
}

// A generator renders by its source; no constructor expression re-creates a state mid-sequence.
func generatorTypeString(v core.Value) string {
	if GeneratorOf(v).pcg == nil {
		return "<" + generatorTypeName + ":entropy>"
	}
	return "<" + generatorTypeName + ":seeded>"
}

// --- the derivations, the same for both sources

// uint64n is a uniform draw in [0, n), n > 0, without modulo bias (Lemire's multiply-and-reject).
func (g *Generator) uint64n(n uint64) uint64 {
	if n&(n-1) == 0 {
		return g.src.Uint64() & (n - 1)
	}
	hi, lo := bits.Mul64(g.src.Uint64(), n)
	if lo < n {
		threshold := -n % n
		for lo < threshold {
			hi, lo = bits.Mul64(g.src.Uint64(), n)
		}
	}
	return hi
}

// float64 is a uniform draw in [0, 1) with 53 random bits.
func (g *Generator) float64() float64 {
	return float64(g.src.Uint64()>>11) / (1 << 53)
}

// normFloat64 is a standard normal draw (Box–Muller, one value per call).
func (g *Generator) normFloat64() float64 {
	u1 := 1 - g.float64() // (0, 1]: log is finite
	u2 := g.float64()
	return math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

// expFloat64 is an exponential draw with rate 1 (inverse CDF).
func (g *Generator) expFloat64() float64 {
	return -math.Log(1 - g.float64())
}
