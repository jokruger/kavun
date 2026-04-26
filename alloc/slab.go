package alloc

type clearFunc[T any] func(*T)

type slab[T any] struct {
	buf      []T
	used     int
	fallback int
	clear    clearFunc[T]
}

// newSlab creates a new slab with the given capacity and clear function (a function that releases internal type resources; use nil if not needed).
func newSlab[T any](n int, clear clearFunc[T]) slab[T] {
	return slab[T]{
		buf:   make([]T, n),
		clear: clear,
	}
}

func (s *slab[T]) alloc() *T {
	if s.used < len(s.buf) {
		var zero T
		s.buf[s.used] = zero // zero out the slot before returning it
		t := &s.buf[s.used]
		s.used++
		return t
	}
	s.fallback++
	return new(T) // heap fallback
}

func (s *slab[T]) reset() {
	if s.clear != nil {
		for i := 0; i < s.used; i++ {
			s.clear(&s.buf[i])
		}
	}
	s.used = 0
	s.fallback = 0
}
