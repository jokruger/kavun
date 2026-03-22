package core

type NativeFunc = func(VM, ...Object) (Object, error)
