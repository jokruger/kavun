package core

import (
	"time"

	"github.com/jokruger/gs/token"
)

// Object represents an object in the VM.
type Object interface {
	TypeName() string // return type name
	String() string   // return string representation of the object (should be valid GS string literal if possible)
	Interface() any   // return underlying value as empty interface
	Arity() int       // return number of positional arguments (or minimum number of variadic arguments) for callable objects

	BinaryOp(VM, token.Token, Object) (Object, error) // return result of binary operation with another object
	Equals(Object) bool                               // return whether the object is equal to the value of another object
	Copy(Allocator) Object                            // return a copy of the object
	Access(VM, Object, Opcode) (Object, error)        // return result of accessing the object at the given index with the given mode (index or selector)
	Assign(idx, val Object) error                     // return result of setting the value of the object at the given index
	Iterate(Allocator) Iterator                       // return an Iterator for the object
	Call(VM, ...Object) (Object, error)               // return result of calling the object with the given arguments

	IsUndefined() bool // return whether the object is undefined
	IsTrue() bool      // return whether the object is true (i.e. truthy)
	IsFalse() bool     // return whether the object is false (i.e. falsy)
	IsIterable() bool  // return whether the object is iterable (i.e. can be used in a for loop)
	IsCallable() bool  // return whether the object is callable (i.e. can be called like a function)
	IsImmutable() bool // return whether the object is immutable (i.e. cannot be modified after creation)
	IsVariadic() bool  // return whether the object is variadic (i.e. can accept a variable number of arguments)

	AsString() (string, bool)  // return string value and whether the conversion was successful
	AsInt() (int64, bool)      // return int value and whether the conversion was successful
	AsFloat() (float64, bool)  // return float value and whether the conversion was successful
	AsBool() (bool, bool)      // return bool value and whether the conversion was successful
	AsRune() (rune, bool)      // return rune value and whether the conversion was successful
	AsBytes() ([]byte, bool)   // return byte slice value and whether the conversion was successful
	AsTime() (time.Time, bool) // return time value and whether the conversion was successful
}
