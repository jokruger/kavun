package compiler

import (
	"encoding/binary"
	"fmt"

	"github.com/jokruger/kavun/core/opcode"
)

// ComputeMaxStack returns the maximum operand-stack depth that the given bytecode instruction stream can reach during
// execution.
//
// The analysis is a flow-sensitive worklist over the instruction stream that handles control flow precisely (unconditional
// jumps don't fall through; conditional jumps explore both arms at the correct height; unreachable code is skipped).
// It relies on the compiler invariant that all paths reaching the same instruction offset do so at equal stack heights.
// When that invariant holds, the analyzer returns the exact peak.
//
// Pre-calculated result is used by the VM to size per-frame stack requirements precisely so that OpCall can guarantee
// the callee has enough room without an arbitrary safety margin.
//
// LIMITATION: spread-call expansion (`f(arr...)`) is data-driven and grows the caller's operand stack by `len(arr)-1`
// slots at runtime. That growth is invisible to this static analysis. The VM bounds-checks the destination before
// expanding the spread and raises a recoverable stack-overflow error if it would exceed the global stack — see
// vm/vm.go OpCall/OpMethodCall.
func ComputeMaxStack(instructions []byte) int {
	n := len(instructions)
	if n == 0 {
		return 0
	}

	// heights[ip] = stack height on entry to the instruction at offset ip; -1 means not yet visited.
	heights := make([]int, n)
	for i := range heights {
		heights[i] = -1
	}

	worklist := []int{0}
	heights[0] = 0
	maxStack := 0

	push := func(ip, h int) {
		if ip < 0 || ip >= n {
			return
		}
		if heights[ip] == -1 {
			heights[ip] = h
			worklist = append(worklist, ip)
		}
		// If already visited, compiler invariants guarantee the same height;
		// we don't re-enqueue (a robust check would assert equality).
	}

	for len(worklist) > 0 {
		ip := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		cur := heights[ip]

		op := opcode.Opcode(instructions[ip])
		e := analyzeOp(op, instructions, ip+1)

		// The peak observed at this opcode is the higher of the entry height (cur) and the post-op height (cur+e.net).
		// We don't need a separate "transient peak" term because no currently defined opcode reaches a height greater
		// than max(entry, after) — pops happen before pushes from already-on-stack values. If a future opcode does,
		// give its stackEffect a `net` that reflects its peak, or extend stackEffect.
		after := max(cur+int(e.net), 0)
		maxStack = max(maxStack, cur, after)
		next := ip + 1 + op.Width()

		switch e.cf {
		case cfFallthrough:
			push(next, after)
		case cfCondJump:
			push(e.target, after)
			push(next, after)
		case cfUncondJump:
			push(e.target, after)
		case cfTerminator:
			// no successor
		}
	}

	return maxStack
}

// controlFlow classifies how execution continues from an opcode.
type controlFlow uint8

const (
	cfFallthrough controlFlow = iota // proceed to next instruction
	cfCondJump                       // may branch to target OR fall through
	cfUncondJump                     // always branch to target; no fall-through
	cfTerminator                     // path ends here (OpReturn, OpSuspend)
)

// stackEffect is the meta-information ComputeMaxStack needs per opcode. Key facts supplied for each opcode:
//
//  1. `net` — signed change to operand-stack height after the opcode executes, relative to the height ENTERING the
//     opcode (positive = push, negative = pop). The analyzer tracks both the entry height and the post-op height as
//     candidate peaks, so an opcode that briefly grows the stack above entry+max(0,net) is NOT representable here
//     (none of the current opcodes do this), but if you add one, extend stackEffect with a "transient peak" field
//     rather than fudging net.
//
//  2. `cf` — control-flow class (see controlFlow constants). If cf is cfCondJump or cfUncondJump, `target` must be the
//     absolute byte offset of the branch destination in the instruction stream.
//
// The number of operand bytes consumed is NOT part of stackEffect; it is looked up via operandBytesOf (which consults
// opcode.OpcodeOperands), keeping the two pieces of metadata in their natural single source of truth.
//
// Short-circuit modeling note: OpAndJump / OpOrJump leave the LHS on the stack when the jump is taken (it becomes the
// expression result) and pop it when execution falls through (the RHS will push a replacement). We model both as net=-1
// with cf=cfCondJump because the join point's height is the same either way: jump-taken keeps 1 slot, fall-through pops
// 1 then RHS pushes 1.
type stackEffect struct {
	net    int8        // signed stack height delta
	cf     controlFlow // control-flow class
	target int         // branch target offset (only when cf is cfCondJump/cfUncondJump)
}

// analyzeOp returns the stack/control-flow effect of the opcode at offset ip-1 whose operands begin at opStart in ins.
//
// HOW TO ADD A NEW OPCODE
//
//  1. Add a case in the matching block below (pure push, pure pop, in-place, jump, call-style, terminator, etc). Return
//     a stackEffect literal with the net stack delta and control-flow class. For variadic-arity ops, read the count
//     from operand bytes and compute net inline.
//  2. If your opcode briefly grows the stack ABOVE max(entry, after) — that is, it has a transient peak that neither
//     the entry height nor the post-op height captures — DO NOT try to encode it via net. Extend stackEffect with a
//     "transient" field and update the worklist loop.
//  3. The default arm panics on unrecognized opcodes; forgetting to add a case is therefore a loud failure, not a
//     silent under-approximation.
//
// Reading operand values: variadic-arity opcodes read N from operand bytes.
// The opStart parameter points at the first operand byte.
func analyzeOp(op opcode.Opcode, ins []byte, opStart int) stackEffect {
	switch op {
	// Abort poll/checkpoint opcode (net 0, falls through)
	case opcode.AbortCheck:
		return stackEffect{net: 0, cf: cfFallthrough}

	// Pure pushes (net +1, falls through)
	case opcode.LoadStaticPrimitive, opcode.LoadStaticDecimal, opcode.LoadStaticString, opcode.LoadStaticRunes, opcode.LoadStaticBytes, opcode.LoadStaticTime, opcode.LoadStaticFormatSpec, opcode.LoadStaticCompiledFunction, opcode.PushTrue, opcode.PushFalse, opcode.PushUndefined, opcode.LoadGlobal, opcode.LoadLocal, opcode.LoadFree, opcode.LoadFreePtr, opcode.LoadLocalPtr, opcode.LoadBuiltinFunction, opcode.ImportBuiltinModule:
		return stackEffect{net: 1, cf: cfFallthrough}

	// Pure pops (net -1, falls through)
	case opcode.Pop, opcode.StoreGlobal, opcode.StoreLocal, opcode.DefineLocal, opcode.StoreFree:
		return stackEffect{net: -1, cf: cfFallthrough}

	// In-place transforms (net 0, falls through)
	case opcode.UnaryBitNot, opcode.UnaryNeg, opcode.UnaryNot, opcode.Immutable, opcode.Format, opcode.IteratorInit, opcode.IteratorNext, opcode.IteratorKey, opcode.IteratorValue:
		return stackEffect{net: 0, cf: cfFallthrough}

	// Pop-2-push-1 binary ops (net -1, falls through)
	case opcode.BinaryOp, opcode.Equal, opcode.NotEqual, opcode.AccessIndex, opcode.Contains, opcode.AccessSelector, opcode.FormatDyn:
		return stackEffect{net: -1, cf: cfFallthrough}

	// Slicing (pops indices, keeps the sliced value on the stack)
	case opcode.SliceIndex: // pops low, high; keeps array
		return stackEffect{net: -2, cf: cfFallthrough}
	case opcode.SliceIndexStep: // pops low, high, step; keeps array
		return stackEffect{net: -3, cf: cfFallthrough}

	// Control flow
	case opcode.JumpFalsy:
		n := int(binary.LittleEndian.Uint16(ins[opStart:]))
		return stackEffect{net: -1, cf: cfCondJump, target: n}
	case opcode.AndJump, opcode.OrJump: // Short-circuit: see stackEffect doc for modeling rationale.
		n := int(binary.LittleEndian.Uint16(ins[opStart:]))
		return stackEffect{net: -1, cf: cfCondJump, target: n}
	case opcode.Jump:
		n := int(binary.LittleEndian.Uint16(ins[opStart:]))
		return stackEffect{net: 0, cf: cfUncondJump, target: n}
	case opcode.Suspend:
		return stackEffect{net: 0, cf: cfTerminator}
	case opcode.Return: // hasResult==1 means a result value was pushed and is popped to return.
		if ins[opStart] == 1 {
			return stackEffect{net: -1, cf: cfTerminator}
		}
		return stackEffect{net: 0, cf: cfTerminator}

	// Variable arity: net depends on an operand-encoded count
	case opcode.Array, opcode.Record: // N elements (or 2*N for records) on stack, replaced by 1 result.
		n := int(binary.LittleEndian.Uint16(ins[opStart:]))
		return stackEffect{net: int8(1 - n), cf: cfFallthrough}
	case opcode.Call: // Pops N args; callee slot is reused for the return value, so net = -N.
		n := int(ins[opStart])
		return stackEffect{net: int8(-n), cf: cfFallthrough}
	case opcode.MethodCall: // numArgs at operand offset 2. Receiver slot is reused for the return value.
		n := int(ins[opStart+2])
		return stackEffect{net: int8(-n), cf: cfFallthrough}
	case opcode.MakeClosure: // Pops NF free vars, pushes 1 closure.
		nf := int(ins[opStart+2])
		return stackEffect{net: int8(1 - nf), cf: cfFallthrough}
	case opcode.StoreIndexedGlobal: // Pops NS selector values + 1 RHS value.
		ns := int(ins[opStart+2])
		return stackEffect{net: int8(-ns - 1), cf: cfFallthrough}
	case opcode.StoreIndexedLocal, opcode.StoreIndexedFree:
		ns := int(ins[opStart+1])
		return stackEffect{net: int8(-ns - 1), cf: cfFallthrough}
	case opcode.Defer: // Pops callee + N args; pushes nothing.
		n := int(ins[opStart])
		return stackEffect{net: int8(-n - 1), cf: cfFallthrough}
	case opcode.DeferMethod: // Pops receiver + N args; pushes nothing.
		n := int(ins[opStart+2])
		return stackEffect{net: int8(-n - 1), cf: cfFallthrough}

	default:
		// Unknown opcode: panic to fail loudly. A silent default would silently under-approximate the stack depth and
		// risk runtime overflows.
		panic(fmt.Sprintf("compiler.analyzeOp: unknown opcode 0x%02x (%d) — analyzer must be updated when new opcodes are added", byte(op), int(op)))
	}
}
