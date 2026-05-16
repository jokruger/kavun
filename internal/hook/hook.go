package hook

type value interface {
	TypeName() string
}

func Const[V any, C any](c C) func(V) C {
	return func(V) C {
		return c
	}
}

func Const2[V any, C1 any, C2 any](c1 C1, c2 C2) func(V) (C1, C2) {
	return func(V) (C1, C2) {
		return c1, c2
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
