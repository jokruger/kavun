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

	BinaryOp(VM, token.Token, Value) (Value, error) // return result of binary operation with another object
	Equals(Value) bool                              // return whether the object is equal to the value of another object
	Copy(Allocator) Value                           // return a copy of the object
	Method(VM, string, []Value) (Value, error)      // return result of calling the method with the given name and arguments on the object
	Access(VM, Value, Opcode) (Value, error)        // return result of accessing the object at the given index with the given mode (index or selector)
	Assign(idx, val Value) error                    // return result of setting the value of the object at the given index
	Iterate(Allocator) Iterator                     // return an Iterator for the object
	Call(VM, []Value) (Value, error)                // return result of calling the object with the given arguments

	IsUndefined() bool        // return whether the object is undefined
	IsString() bool           // return whether the object is a string
	IsInt() bool              // return whether the object is an int
	IsFloat() bool            // return whether the object is a float
	IsBool() bool             // return whether the object is a bool
	IsChar() bool             // return whether the object is a char
	IsBytes() bool            // return whether the object is a byte slice
	IsTime() bool             // return whether the object is a time
	IsArray() bool            // return whether the object is an array
	IsError() bool            // return whether the object is an error
	IsMap() bool              // return whether the object is a map
	IsRecord() bool           // return whether the object is a record
	IsCompiledFunction() bool // return whether the object is a compiled function
	IsBuiltinFunction() bool  // return whether the object is a builtin function
	IsTrue() bool             // return whether the object is in true state (for non-boolean objects this is usually related to whether the object is not empty/zero)
	IsFalse() bool            // return whether the object is in false state (for non-boolean objects this is usually related to whether the object is empty/zero)
	IsIterable() bool         // return whether the object is iterable (i.e. can be used in a for loop)
	IsCallable() bool         // return whether the object is callable (i.e. can be called like a function)
	IsVariadic() bool         // return whether the object is variadic (i.e. can accept a variable number of arguments)
	IsImmutable() bool        // return whether the object is immutable (i.e. the elements of container object cannot be modified)

	AsString() (string, bool)  // return string value and whether the conversion was successful
	AsInt() (int64, bool)      // return int value and whether the conversion was successful
	AsFloat() (float64, bool)  // return float value and whether the conversion was successful
	AsBool() (bool, bool)      // return bool value and whether the conversion was successful
	AsChar() (rune, bool)      // return rune value and whether the conversion was successful
	AsBytes() ([]byte, bool)   // return byte slice value and whether the conversion was successful
	AsTime() (time.Time, bool) // return time value and whether the conversion was successful
}
