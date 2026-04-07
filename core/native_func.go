package core

type NativeFunc = func(VM, []Value) (Value, error)
