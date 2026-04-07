package core

type VM interface {
	Allocator() Allocator
	Abort()
	IsStackEmpty() bool
	Call(*CompiledFunction, []Value) (Value, error)
	Run() error
}
