package compiler

import (
	"fmt"

	bc "github.com/jokruger/kavun/core/bytecode"
)

const (
	spreadNet = 128 // Guess at the stack effect of a function call with a spread argument
)

// ComputeMaxStack returns the maximum operand-stack depth that the given bytecode instruction stream can reach during
// execution.
func ComputeMaxStack(instructions bc.Instructions) int {
	n := len(instructions)
	if n == 0 {
		return 0
	}

	type instrHeight struct {
		height  int
		visited bool
	}

	// heights[ip] = stack height on entry to the instruction at offset ip
	heights := make([]instrHeight, n)

	worklist := []int{0}
	heights[0].visited = true
	maxStack := 0

	push := func(ip, h int) {
		if ip < 0 || ip >= n {
			return
		}
		if !heights[ip].visited {
			heights[ip] = instrHeight{height: h, visited: true}
			worklist = append(worklist, ip)
		}
		// If already visited, compiler invariants guarantee the same height;
		// we don't re-enqueue (a robust check would assert equality).
	}

	for len(worklist) > 0 {
		ip := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		cur := heights[ip].height

		ci := instructions[ip]
		e := analyzeOp(ci)

		// The peak observed at this opcode is the higher of the entry height (cur) and the post-op height (cur+e.net).
		// We don't need a separate "transient peak" term because no currently defined opcode reaches a height greater
		// than max(entry, after) — pops happen before pushes from already-on-stack values. If a future opcode does,
		// give its stackEffect a `net` that reflects its peak, or extend stackEffect.
		after := max(cur+int(e.net), 0)
		maxStack = max(maxStack, cur, after)
		next := ip + 1

		switch ci.Op.Class() {
		case bc.OpFallThrough:
			push(next, after)
		case bc.OpConditional:
			push(e.target, after)
			push(next, after)
		case bc.OpUnconditional:
			push(e.target, after)
		case bc.OpTerminating:
			// no successor
		}
	}

	return maxStack
}

type stackEffect struct {
	net    int // signed stack height delta
	target int // branch target offset (only when cf is conditional / unconditional jump)
}

// analyzeOp returns the stack/control-flow effect of the instruction.
func analyzeOp(ci bc.Instruction) stackEffect {
	var e stackEffect

	switch ci.Op {
	// No input, no output
	case bc.AbortCheck, bc.Suspend:
		e.net = 0

	// 1 input, 1 output
	case bc.UnaryBitNot, bc.UnaryNeg, bc.UnaryNot, bc.Immutable, bc.FormatStaticSpec:
		e.net = 0

	// 1 input, 1 output
	case bc.IterInit, bc.IterNext, bc.IterKey, bc.IterValue:
		e.net = 0

	// 0 input, 1 output
	case bc.PushUndefined, bc.PushBool, bc.PushByte, bc.PushRune, bc.PushInt:
		e.net = 1

	// 0 input, 1 output
	case bc.LoadStaticDecimal, bc.LoadStaticString, bc.LoadStaticRunes, bc.LoadStaticBytes, bc.LoadStaticTime, bc.LoadStaticFormatSpec, bc.LoadStaticCompiledFunction, bc.LoadStaticPrimitive:
		e.net = 1

	// 0 input, 1 output
	case bc.LoadLocal, bc.LoadLocalPtr, bc.LoadFree, bc.LoadFreePtr, bc.LoadGlobal:
		e.net = 1

	// 0 input, 1 output
	case bc.LoadBuiltinFunction, bc.ImportBuiltinModule:
		e.net = 1

	// 1 input, 0 output
	case bc.Pop, bc.DefineLocal, bc.StoreLocal, bc.StoreFree, bc.StoreGlobal:
		e.net = -1

	// 2 inputs, 1 output
	case bc.BinaryOp, bc.Equal, bc.NotEqual, bc.Contains, bc.AccessIndex, bc.AccessSelector, bc.FormatRuntimeSpec:
		e.net = -1

	// 3 inputs, 1 output
	case bc.Slice:
		e.net = -2

	// 4 inputs, 1 output
	case bc.SliceStep:
		e.net = -3

	// Jump unconditional, no stack effect, 16-bit target
	case bc.Jump:
		e.net = 0
		e.target = int(ci.Op3)

	// Jump conditional, no stack effect, 16-bit target
	case bc.JumpFalsy, bc.AndJump, bc.OrJump:
		e.net = -1
		e.target = int(ci.Op3)

	// Return, no or 1 output depending on 8-bit operand
	case bc.Return:
		if ci.Op1 != 0 {
			e.net = -1
		} else {
			e.net = 0
		}

	// N inputs, 1 output, 16-bit operand (in case of record operand = 2 * num elements)
	case bc.MakeArray, bc.MakeRecord:
		e.net = 1 - int(ci.Op3)

	// Call function: 1 + N inputs, 1 output
	case bc.CallFunction:
		if ci.Op1 != 0 {
			e.net = spreadNet
		} else {
			e.net = 1 - 1 - int(ci.Op2)
		}

	// Call method: 2 + N inputs, 1 output, 16-bit index
	case bc.CallMethod:
		if ci.Op1 != 0 {
			e.net = spreadNet
		} else {
			e.net = 1 - 2 - int(ci.Op2)
		}

	// Make closure: N inputs, 1 output, 16-bit index
	case bc.MakeClosure:
		e.net = 1 - int(ci.Op2)

	// 1 + N inputs, 0 outputs, 8-bit index
	case bc.StoreIndexedLocal, bc.StoreIndexedFree:
		e.net = 0 - 1 - int(ci.Op2)

	// 1 + N inputs, 0 outputs, 16-bit index
	case bc.StoreIndexedGlobal, bc.DeferMethod:
		e.net = 0 - 1 - int(ci.Op2)

	// 1 + N inputs, 0 outputs
	case bc.Defer:
		e.net = 0 - 1 - int(ci.Op2)

	default:
		// Unknown opcode: panic to fail loudly. A silent default would silently under-approximate the stack depth and
		// risk runtime overflows.
		panic(fmt.Sprintf("compiler.analyzeOp: unknown opcode %s", ci.String()))
	}

	return e
}
