package hook

type value interface {
	TypeName() string
}

func Const[V any, C any](c C) func(V) C {
	return func(V) C {
		return c
	}
}

func Value[V any, A any](v V, e error) func(V, A) (V, error) {
	return func(V, A) (V, error) {
		return v, e
	}
}

func Self[V any, A any](v V, _ A) (V, error) {
	return v, nil
}
