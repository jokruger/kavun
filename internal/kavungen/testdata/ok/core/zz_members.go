//go:build ignore

package core

import "github.com/jokruger/kavun/core/member"

// Excluded by the build constraint, so never read.
//
//kavun:member copy pure () -> widget
func widgetCopy(_ VM, v Value, id member.ID, args []Value) (Value, error) { return v, nil }
