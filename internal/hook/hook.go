package hook

func ReturnConst[V any, C any](c C) func(V) C {
	return func(V) C {
		return c
	}
}

func ReutrnValue[V any](v V, e error) func(V) (V, error) {
	return func(V) (V, error) {
		return v, e
	}
}
