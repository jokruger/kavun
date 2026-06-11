package compiler_test

import (
	"strings"
	"testing"

	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/opcode"
	"github.com/jokruger/kavun/test"
)

// Static cases — small hand-built bytecode snippets with known stack heights.
func TestComputeMaxStack_Static(t *testing.T) {
	rta := core.NewArena(nil)

	cases := []struct {
		name string
		ins  []byte
		want int
	}{
		{
			"empty",
			[]byte{},
			0,
		},
		{
			"single constant push",
			[]byte{byte(opcode.StaticStringValue), 0, 0},
			1,
		},
		{
			"push and pop balances to zero peak of 1",
			[]byte{
				byte(opcode.StaticStringValue), 0, 0,
				byte(opcode.Pop),
			},
			1,
		},
		{
			"three pushes then pop reaches peak 3",
			[]byte{
				byte(opcode.StaticPrimitiveValue), 0, 0,
				byte(opcode.StaticStringValue), 0, 1,
				byte(opcode.StaticRunesValue), 0, 2,
				byte(opcode.Pop),
				byte(opcode.Pop),
				byte(opcode.Pop),
			},
			3,
		},
		{
			"binary op: a+b peaks at 2",
			[]byte{
				byte(opcode.StaticPrimitiveValue), 0, 0,
				byte(opcode.StaticPrimitiveValue), 0, 1,
				byte(opcode.BinaryOp), 1,
			},
			2,
		},
		{
			"array of 4 elements peaks at 4",
			[]byte{
				byte(opcode.StaticPrimitiveValue), 0, 0,
				byte(opcode.StaticPrimitiveValue), 0, 1,
				byte(opcode.StaticPrimitiveValue), 0, 2,
				byte(opcode.StaticPrimitiveValue), 0, 3,
				byte(opcode.Array), 0, 4,
			},
			4,
		},
		{
			"call with 3 args peaks at 4 (callee + 3 args)",
			[]byte{
				byte(opcode.GetGlobal), 0, 0, // callee
				byte(opcode.StaticPrimitiveValue), 0, 0,
				byte(opcode.StaticPrimitiveValue), 0, 1,
				byte(opcode.StaticPrimitiveValue), 0, 2,
				byte(opcode.Call), 3, 0,
			},
			4,
		},
		{
			"short-circuit AND balances",
			// Push a, AndJump END, push b, END: result on stack -> peak 1
			[]byte{
				byte(opcode.StaticPrimitiveValue), 0, 0, // push a
				byte(opcode.AndJump), 0, 9, // jump to END if false
				byte(opcode.StaticPrimitiveValue), 0, 1, // push b (fall-through)
				// END: result is one value
			},
			1,
		},
		{
			"if/else both arms balance",
			// 0: push cond           (3 bytes)
			// 3: JumpFalsy -> 16     (5 bytes)
			// 8: push then           (3 bytes)
			// 11: Jump -> 19         (5 bytes)
			// 16: push else          (3 bytes)
			// 19: <end>
			[]byte{
				byte(opcode.StaticPrimitiveValue), 0, 0, // cond
				byte(opcode.JumpFalsy), 0, 16, // -> ELSE
				byte(opcode.StaticPrimitiveValue), 0, 1, // then
				byte(opcode.Jump), 0, 19, // -> END
				byte(opcode.StaticPrimitiveValue), 0, 2, // else
				// END
			},
			1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compiler.ComputeMaxStack(tc.ins)
			test.Equal(t, rta, tc.want, got)
		})
	}
}

// Ensures that analyzeOp panics on an unknown opcode. This is a guard against forgetting to extend the analyzer
// when a new opcode is introduced.
func TestComputeMaxStack_UnknownOpcodePanics(t *testing.T) {
	// 0xFF is well outside the range of currently defined opcodes.
	ins := []byte{0xFF}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic on unknown opcode, got nil")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T: %v", r, r)
		}
		if !strings.Contains(msg, "unknown opcode") {
			t.Fatalf("expected panic message to mention 'unknown opcode', got %q", msg)
		}
	}()
	_ = compiler.ComputeMaxStack(ins)
}
