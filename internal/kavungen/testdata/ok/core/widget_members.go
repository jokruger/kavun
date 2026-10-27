package core

import "github.com/jokruger/kavun/core/member"

// Pad at the back until length `n`.
//
// The fill is **one element**:
//
// 1. a numbered item
// 2. another
//
// ```go
// w.pad_end(3)
//
// w.pad_end(3, 0)
// ```
//
// ```go
//
//	w.each(func(x) {
//	    print(x)
//	})
//
// ```
//
// #### A fourth-level heading
//
// | width | result |
// | --- | --- |
// | below the length | unchanged |
//
//kavun:member pad_end pure (n[, fill]) -> widget
func widgetPadEnd(_ VM, v Value, id member.ID, args []Value) (Value, error) { return v, nil }

// Pad at the front until length `n`; see [`pad_end`](#pad_end).
//
//kavun:member pad_start pure (n[, fill]) -> widget
func widgetPadStart(_ VM, v Value, id member.ID, args []Value) (Value, error) { return v, nil }

// Remove `delete_count` parts at `start`, then
// insert the items there.
//
//kavun:member splice pure ([start[, delete_count]]) | (start, delete_count, ...items) -> widget
func widgetSplice(_ VM, v Value, id member.ID, args []Value) (Value, error) { return v, nil }

// Answer the number of parts.
//
//go:noinline
//kavun:member len pure () -> int
func widgetLen(_ VM, v Value, id member.ID, args []Value) (Value, error) { return v, nil }

// Report truthiness.
//
// A wrapper's doc block is rendered like any other.
//
//kavun:member is_true pure () -> bool
func widgetIsTrue(_ VM, v Value, id member.ID, args []Value) (Value, error) { return v, nil }

// Shuffle the parts with the generator `g`.
//
//kavun:member shuffle impure (g) -> widget
func widgetShuffle(_ VM, v Value, id member.ID, args []Value) (Value, error) { return v, nil }

// A helper with no directive is not a member.
func widgetHelper() {}
