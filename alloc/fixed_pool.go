package alloc

type fixedPool[T any] struct {
	buf      []T
	used     int
	fallback int
}

func newFixedPool[T any](n int) fixedPool[T] {
	return fixedPool[T]{
		buf: make([]T, n),
	}
}

func (p *fixedPool[T]) alloc() *T {
	if p.used < len(p.buf) {
		t := &p.buf[p.used]
		p.used++
		return t
	}
	p.fallback++
	return new(T) // heap fallback
}

func (p *fixedPool[T]) reset() {
	var zero T
	for i := 0; i < p.used; i++ {
		p.buf[i] = zero
	}
	p.used = 0
	p.fallback = 0
}
