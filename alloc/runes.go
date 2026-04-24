package alloc

func (a *Allocator) NewRunes(capacity int) ([]rune, error) {
	o := make([]rune, 0, capacity)
	return o, nil
}

func (a *Allocator) ReleaseRunes(r []rune) {
}
