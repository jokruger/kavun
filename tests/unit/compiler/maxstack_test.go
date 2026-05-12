package compiler_test

import (
	"testing"

	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/tests/require"
)

// Static cases — small hand-built bytecode snippets with known stack heights.
func TestComputeMaxStack_Static(t *testing.T) {
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
			[]byte{byte(core.OpConstant), 0, 0},
			1,
		},
		{
			"push and pop balances to zero peak of 1",
			[]byte{
				byte(core.OpConstant), 0, 0,
				byte(core.OpPop),
			},
			1,
		},
		{
			"three pushes then pop reaches peak 3",
			[]byte{
				byte(core.OpConstant), 0, 0,
				byte(core.OpConstant), 0, 1,
				byte(core.OpConstant), 0, 2,
				byte(core.OpPop),
				byte(core.OpPop),
				byte(core.OpPop),
			},
			3,
		},
		{
			"binary op: a+b peaks at 2",
			[]byte{
				byte(core.OpConstant), 0, 0,
				byte(core.OpConstant), 0, 1,
				byte(core.OpBinaryOp), 1,
			},
			2,
		},
		{
			"array of 4 elements peaks at 4",
			[]byte{
				byte(core.OpConstant), 0, 0,
				byte(core.OpConstant), 0, 1,
				byte(core.OpConstant), 0, 2,
				byte(core.OpConstant), 0, 3,
				byte(core.OpArray), 0, 4,
			},
			4,
		},
		{
			"call with 3 args peaks at 4 (callee + 3 args)",
			[]byte{
				byte(core.OpGetGlobal), 0, 0, // callee
				byte(core.OpConstant), 0, 0,
				byte(core.OpConstant), 0, 1,
				byte(core.OpConstant), 0, 2,
				byte(core.OpCall), 3, 0,
			},
			4,
		},
		{
			"short-circuit AND balances",
			// Push a, AndJump END, push b, END: result on stack -> peak 1
			[]byte{
				byte(core.OpConstant), 0, 0, // push a
				byte(core.OpAndJump), 0, 0, 0, 9, // jump to END if false
				byte(core.OpConstant), 0, 1, // push b (fall-through)
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
				byte(core.OpConstant), 0, 0, // cond
				byte(core.OpJumpFalsy), 0, 0, 0, 16, // -> ELSE
				byte(core.OpConstant), 0, 1, // then
				byte(core.OpJump), 0, 0, 0, 19, // -> END
				byte(core.OpConstant), 0, 2, // else
				// END
			},
			1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compiler.ComputeMaxStack(tc.ins)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestComputeMaxStack_UnknownOpcodePanics ensures that analyzeOp panics on an
// unknown opcode. This is a guard against forgetting to extend the analyzer
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
if !contains(msg, "unknown opcode") {
t.Fatalf("expected panic message to mention 'unknown opcode', got %q", msg)
}
}()
_ = compiler.ComputeMaxStack(ins)
}

func contains(s, sub string) bool {
for i := 0; i+len(sub) <= len(s); i++ {
if s[i:i+len(sub)] == sub {
return true
}
}
return false
}
