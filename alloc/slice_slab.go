package alloc

type sliceSlab[T any] struct {
	buf      [][]T
	cap      int
	used     int
	fallback int
}

func newSliceSlab[T any](n int, cap int) sliceSlab[T] {
	buf := make([][]T, n)
	for i := range buf {
		buf[i] = make([]T, 0, cap)
	}
	return sliceSlab[T]{buf: buf, cap: cap}
}

func (s *sliceSlab[T]) alloc(cap int, resize bool) []T {
	if cap <= s.cap && s.used < len(s.buf) {
		t := s.buf[s.used]
		if resize {
			t = t[:cap]
		}
		s.used++
		return t
	}
	s.fallback++
	if resize {
		return make([]T, cap)
	}
	return make([]T, 0, cap)
}

func (s *sliceSlab[T]) reset() {
	// No need to clear the slices since they never modified (we modify the copy in alloc, not the original) and cannot exceed the original capacity.
	s.used = 0
	s.fallback = 0
}
