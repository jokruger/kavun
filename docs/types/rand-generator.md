# rand.generator

A source of random numbers, as `rand.generator(seed)` and `os.rand_generator()` answer it.

## Overview

`rand.generator` is owned by the [`rand`](../stdlib.md#rand) module. It is ONE type over two sources, so anything
that takes a generator — `array.shuffle(g)`, a function of your own — accepts either:

| constructor | source | sequence |
| --- | --- | --- |
| `rand.generator(seed)` (pure) | PCG-DXSM, seeded | the same for the same seed, on every build |
| `os.rand_generator()` (needs the `os` grant) | `crypto/rand` | unpredictable |

Every draw is derived from the source's raw 64-bit output by Kavun's own code (bounded ints without bias, 53-bit
floats, Box–Muller normals, Fisher–Yates permutations), so a seeded sequence depends only on PCG-DXSM — a specified
algorithm — and not on Go's library helpers. A draw moves the generator's state: it is the value's own, never shared
with another script.

```go
g := import("rand").generator(1)
g.int(100)                  // 59 — then 8, then 71: seed 1 always starts this way
g.int(1, 7)                 // a die roll in [1, 7)
g.float()                   // [0, 1)
g.perm(4)                   // a permutation of 0…3
d := import("os").rand_generator()
d.int(1, 7)                 // unpredictable
```

## Operators

| operator | reading |
| --- | --- |
| `==` / `!=` | the same generator (identity — two generators with the same seed are still two) |
| `!` | a generator is always truthy |
| everything else | raises |

## Members

Ranges are half-open, like `range(lo, hi)`.

| member | answers |
| --- | --- |
| `int()` / `int(hi)` / `int(lo, hi)` | an int in [0, 2^63) / [0, hi) / [lo, hi); an empty range raises `invalid_value` |
| `float()` / `float(hi)` / `float(lo, hi)` | a float in [0, 1) / [0, hi) / [lo, hi); bounds must be finite and non-empty |
| `bytes(n)` | `n` random octets (`1 ≤ n ≤` the sequence ceiling) |
| `perm(n)` | a random permutation of `0…n-1` |
| `norm_float()` | a standard normal draw (mean 0, deviation 1) |
| `exp_float()` | an exponential draw (rate 1) |
| `copy()` | seeded: an independent clone at the same state (both then draw the same sequence); entropy: the generator itself |
| `is_true()`, `format([spec])`, `freeze()` | `true`; `<rand.generator:seeded>` / `<rand.generator:entropy>`; the generator itself |

The render names the source, not a constructor expression — no expression rebuilds a state mid-sequence. There is
no `string()`.
