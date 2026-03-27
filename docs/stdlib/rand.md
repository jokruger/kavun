# Module `rand`

```text
rand := import("rand")
```

## Functions

- `int() => int`: non-negative 63-bit integer from the default source.
- `float() => float`: pseudo-random float in `[0.0, 1.0)`.
- `int_n(n int) => int`: pseudo-random integer in `[0, n)`.
- `exp_float() => float`: exponential distribution with rate 1.
- `norm_float() => float`: normal distribution with mean 0 and stddev 1.
- `perm(n int) => [int]`: pseudo-random permutation of `[0, n)`.
- `seed(seed int)`: seed the default source.
- `read(buffer bytes) => int/error`: fill the byte slice with random data.
- `rand(seed int) => generator`: returns a dedicated generator.

### Generators

Records returned by `rand(seed)` expose the same API, scoped to their own source: `int()`, `float()`, `int_n(n)`, `exp_float()`, `norm_float()`, `perm(n)`, `seed(seed)`, `read(bytes)`.
