package alloc

func (a *Allocator) NewBytes(capacity int) ([]byte, error) {
	o := make([]byte, 0, capacity)
	return o, nil
}

func (a *Allocator) ReleaseBytes(v []byte) {
}
