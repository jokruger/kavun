package core

func ConstHook[C any](c C) func(*Arena, Value) C {
	return func(*Arena, Value) C {
		return c
	}
}

func Const2Hook[C1 any, C2 any](c1 C1, c2 C2) func(*Arena, Value) (C1, C2) {
	return func(*Arena, Value) (C1, C2) {
		return c1, c2
	}
}

func ValueHook(v Value, e error) func(*Arena, Value) (Value, error) {
	return func(*Arena, Value) (Value, error) {
		return v, e
	}
}
