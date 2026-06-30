package compiler

import (
	"encoding/binary"
	"fmt"

	"github.com/jokruger/kavun/core/opcode"
)

const (
	spreadNet = 128 // Guess at the stack effect of a function call with a spread argument
)

// ComputeMaxStack returns the maximum operand-stack depth that the given bytecode instruction stream can reach during
// execution.
func ComputeMaxStack(instructions []byte) int {
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

		op := opcode.Opcode(instructions[ip])
		e := analyzeOp(op, instructions, ip+1)

		// The peak observed at this opcode is the higher of the entry height (cur) and the post-op height (cur+e.net).
		// We don't need a separate "transient peak" term because no currently defined opcode reaches a height greater
		// than max(entry, after) — pops happen before pushes from already-on-stack values. If a future opcode does,
		// give its stackEffect a `net` that reflects its peak, or extend stackEffect.
		after := max(cur+int(e.net), 0)
		maxStack = max(maxStack, cur, after)
		next := ip + int(op.Width())

		switch op.Class() {
		case opcode.OpFallThrough:
			push(next, after)
		case opcode.OpConditional:
			push(e.target, after)
			push(next, after)
		case opcode.OpUnconditional:
			push(e.target, after)
		case opcode.OpTerminating:
			// no successor
		}
	}

	return maxStack
}

type stackEffect struct {
	net    int // signed stack height delta
	target int // branch target offset (only when cf is conditional / unconditional jump)
}

// analyzeOp returns the stack/control-flow effect of the opcode at offset ip-1 whose operands begin at opStart in ins.
// The opStart parameter points at the first operand byte.
func analyzeOp(op opcode.Opcode, ins []byte, opStart int) stackEffect {
	var e stackEffect

	switch op {
	// No input, no output
	case opcode.AbortCheck, opcode.Suspend:
		e.net = 0

	// 1 input, 1 output
	case opcode.UnaryBitNot, opcode.UnaryNeg, opcode.UnaryNot, opcode.Immutable, opcode.FormatStaticSpec:
		e.net = 0

	// 1 input, 1 output
	case opcode.IterInit, opcode.IterNext, opcode.IterKey, opcode.IterValue:
		e.net = 0

	// 0 input, 1 output
	case opcode.PushTrue, opcode.PushFalse, opcode.PushUndefined:
		e.net = 1

	// 0 input, 1 output
	case opcode.LoadStaticDecimal8, opcode.LoadStaticString8, opcode.LoadStaticRunes8, opcode.LoadStaticBytes8, opcode.LoadStaticTime8, opcode.LoadStaticFormatSpec8, opcode.LoadStaticCompiledFunction8, opcode.LoadStaticPrimitive8:
		e.net = 1

	// 0 input, 1 output
	case opcode.LoadStaticDecimal16, opcode.LoadStaticString16, opcode.LoadStaticRunes16, opcode.LoadStaticBytes16, opcode.LoadStaticTime16, opcode.LoadStaticFormatSpec16, opcode.LoadStaticCompiledFunction16, opcode.LoadStaticPrimitive16:
		e.net = 1

	// 0 input, 1 output
	case opcode.LoadLocal, opcode.LoadLocalPtr, opcode.LoadFree, opcode.LoadFreePtr, opcode.LoadGlobal8, opcode.LoadGlobal16:
		e.net = 1

	// 0 input, 1 output
	case opcode.LoadBuiltinFunction8, opcode.LoadBuiltinFunction16, opcode.ImportBuiltinModule:
		e.net = 1

	// 1 input, 0 output
	case opcode.Pop, opcode.DefineLocal, opcode.StoreLocal, opcode.StoreFree, opcode.StoreGlobal8, opcode.StoreGlobal16:
		e.net = -1

	// 2 inputs, 1 output
	case opcode.BinaryOp, opcode.Equal, opcode.NotEqual, opcode.Contains, opcode.AccessIndex, opcode.AccessSelector, opcode.FormatRuntimeSpec:
		e.net = -1

	// 3 inputs, 1 output
	case opcode.Slice:
		e.net = -2

	// 4 inputs, 1 output
	case opcode.SliceStep:
		e.net = -3

	// Jump unconditional, no stack effect, 8-bit target
	case opcode.Jump8:
		e.net = 0
		e.target = int(ins[opStart])

	// Jump unconditional, no stack effect, 16-bit target
	case opcode.Jump16:
		e.net = 0
		e.target = int(binary.LittleEndian.Uint16(ins[opStart:]))

	// Jump conditional, no stack effect, 16-bit target
	case opcode.JumpFalsy, opcode.AndJump, opcode.OrJump:
		e.net = -1
		e.target = int(binary.LittleEndian.Uint16(ins[opStart:]))

	// Return, no or 1 output depending on 8-bit operand
	case opcode.Return:
		if ins[opStart] == 1 {
			e.net = -1
		} else {
			e.net = 0
		}

	// N inputs, 1 output, 8-bit operand (in case of record operand = 2 * num elements)
	case opcode.MakeArray8, opcode.MakeRecord8:
		e.net = 1 - int(ins[opStart])

	// N inputs, 1 output, 16-bit operand (in case of record operand = 2 * num elements)
	case opcode.MakeArray16, opcode.MakeRecord16:
		e.net = 1 - int(binary.LittleEndian.Uint16(ins[opStart:]))

	// Call function: 1 + N inputs, 1 output
	case opcode.CallFunction:
		if ins[opStart] != 0 {
			e.net = spreadNet
		} else {
			e.net = 1 - 1 - int(ins[opStart])
		}

	// Call method: 2 + N inputs, 1 output, 8-bit index
	case opcode.CallMethod8:
		if ins[opStart+2] != 0 {
			e.net = spreadNet
		} else {
			e.net = 1 - 2 - int(ins[opStart+1])
		}

	// Call method: 2 + N inputs, 1 output, 16-bit index
	case opcode.CallMethod16:
		if ins[opStart+3] != 0 {
			e.net = spreadNet
		} else {
			e.net = 1 - 2 - int(ins[opStart+2])
		}

	// Make closure: N inputs, 1 output, 8-bit index
	case opcode.MakeClosure8:
		e.net = 1 - int(ins[opStart+1])

	// Make closure: N inputs, 1 output, 16-bit index
	case opcode.MakeClosure16:
		e.net = 1 - int(ins[opStart+2])

	// 1 + N inputs, 0 outputs, 8-bit index
	case opcode.StoreIndexedLocal, opcode.StoreIndexedFree, opcode.StoreIndexedGlobal8, opcode.DeferMethod8:
		e.net = 0 - 1 - int(ins[opStart+1])

	// 1 + N inputs, 0 outputs, 16-bit index
	case opcode.StoreIndexedGlobal16, opcode.DeferMethod16:
		e.net = 0 - 1 - int(ins[opStart+2])

	// 1 + N inputs, 0 outputs
	case opcode.Defer:
		e.net = 0 - 1 - int(ins[opStart])

	default:
		// Unknown opcode: panic to fail loudly. A silent default would silently under-approximate the stack depth and
		// risk runtime overflows.
		panic(fmt.Sprintf("compiler.analyzeOp: unknown opcode 0x%02x (%d) — analyzer must be updated when new opcodes are added", byte(op), int(op)))
	}

	return e
}
