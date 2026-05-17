package internal

import "unsafe"

type Value interface {
	String() string
	TypeName() string
	IsImmutable() bool
	GetPtr() unsafe.Pointer
	AsInt() (int64, bool)
}
