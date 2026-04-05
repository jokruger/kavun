package parser

import "github.com/jokruger/gs/core"

// List of opcodes
const (
	OpConstant      = core.Opcode(0)  // Load constant
	OpBComplement   = core.Opcode(1)  // bitwise complement
	OpPop           = core.Opcode(2)  // Pop
	OpTrue          = core.Opcode(3)  // Push true
	OpFalse         = core.Opcode(4)  // Push false
	OpEqual         = core.Opcode(5)  // Equal ==
	OpNotEqual      = core.Opcode(6)  // Not equal !=
	OpMinus         = core.Opcode(7)  // Minus -
	OpLNot          = core.Opcode(8)  // Logical not !
	OpJumpFalsy     = core.Opcode(9)  // Jump if falsy
	OpAndJump       = core.Opcode(10) // Logical AND jump
	OpOrJump        = core.Opcode(11) // Logical OR jump
	OpJump          = core.Opcode(12) // Jump
	OpNull          = core.Opcode(13) // Push null
	OpArray         = core.Opcode(14) // Array object
	OpRecord        = core.Opcode(15) // Record object
	OpError         = core.Opcode(16) // Error object
	OpImmutable     = core.Opcode(17) // Immutable object
	OpIndex         = core.Opcode(18) // Index operation
	OpSliceIndex    = core.Opcode(19) // Slice operation
	OpCall          = core.Opcode(20) // Call function
	OpReturn        = core.Opcode(21) // Return
	OpGetGlobal     = core.Opcode(22) // Get global variable
	OpSetGlobal     = core.Opcode(23) // Set global variable
	OpSetSelGlobal  = core.Opcode(24) // Set global variable using selectors
	OpGetLocal      = core.Opcode(25) // Get local variable
	OpSetLocal      = core.Opcode(26) // Set local variable
	OpDefineLocal   = core.Opcode(27) // Define local variable
	OpSetSelLocal   = core.Opcode(28) // Set local variable using selectors
	OpGetFreePtr    = core.Opcode(29) // Get free variable pointer object
	OpGetFree       = core.Opcode(30) // Get free variables
	OpSetFree       = core.Opcode(31) // Set free variables
	OpGetLocalPtr   = core.Opcode(32) // Get local variable as a pointer
	OpSetSelFree    = core.Opcode(33) // Set free variables using selectors
	OpGetBuiltin    = core.Opcode(34) // Get builtin function
	OpClosure       = core.Opcode(35) // Push closure
	OpIteratorInit  = core.Opcode(36) // Iterator init
	OpIteratorNext  = core.Opcode(37) // Iterator next
	OpIteratorKey   = core.Opcode(38) // Iterator key
	OpIteratorValue = core.Opcode(39) // Iterator value
	OpBinaryOp      = core.Opcode(40) // Binary operation
	OpSuspend       = core.Opcode(41) // Suspend VM
	OpSelect        = core.Opcode(42) // Select operation
	OpMethodCall    = core.Opcode(43) // Call method on object
	// 44...255 are reserved for future use
)

// OpcodeNames are string representation of opcodes.
var OpcodeNames = [...]string{
	OpConstant:      "CONST",
	OpPop:           "POP",
	OpTrue:          "TRUE",
	OpFalse:         "FALSE",
	OpBComplement:   "NEG",
	OpEqual:         "EQL",
	OpNotEqual:      "NEQ",
	OpMinus:         "NEG",
	OpLNot:          "NOT",
	OpJumpFalsy:     "JMPF",
	OpAndJump:       "ANDJMP",
	OpOrJump:        "ORJMP",
	OpJump:          "JMP",
	OpNull:          "NULL",
	OpGetGlobal:     "GETG",
	OpSetGlobal:     "SETG",
	OpSetSelGlobal:  "SETSG",
	OpArray:         "ARR",
	OpRecord:        "RECORD",
	OpError:         "ERROR",
	OpImmutable:     "IMMUT",
	OpIndex:         "INDEX",
	OpSliceIndex:    "SLICE",
	OpCall:          "CALL",
	OpReturn:        "RET",
	OpGetLocal:      "GETL",
	OpSetLocal:      "SETL",
	OpDefineLocal:   "DEFL",
	OpSetSelLocal:   "SETSL",
	OpGetBuiltin:    "BUILTIN",
	OpClosure:       "CLOSURE",
	OpGetFreePtr:    "GETFP",
	OpGetFree:       "GETF",
	OpSetFree:       "SETF",
	OpGetLocalPtr:   "GETLP",
	OpSetSelFree:    "SETSF",
	OpIteratorInit:  "ITER",
	OpIteratorNext:  "ITNXT",
	OpIteratorKey:   "ITKEY",
	OpIteratorValue: "ITVAL",
	OpBinaryOp:      "BINARYOP",
	OpSuspend:       "SUSPEND",
	OpSelect:        "SELECT",
	OpMethodCall:    "MCALL",
}

// OpcodeOperands is the number of operands.
var OpcodeOperands = [...][]int{
	OpConstant:      {2},
	OpPop:           {},
	OpTrue:          {},
	OpFalse:         {},
	OpBComplement:   {},
	OpEqual:         {},
	OpNotEqual:      {},
	OpMinus:         {},
	OpLNot:          {},
	OpJumpFalsy:     {4},
	OpAndJump:       {4},
	OpOrJump:        {4},
	OpJump:          {4},
	OpNull:          {},
	OpGetGlobal:     {2},
	OpSetGlobal:     {2},
	OpSetSelGlobal:  {2, 1},
	OpArray:         {2},
	OpRecord:        {2},
	OpError:         {},
	OpImmutable:     {},
	OpIndex:         {},
	OpSliceIndex:    {},
	OpCall:          {1, 1},
	OpReturn:        {1},
	OpGetLocal:      {1},
	OpSetLocal:      {1},
	OpDefineLocal:   {1},
	OpSetSelLocal:   {1, 1},
	OpGetBuiltin:    {1},
	OpClosure:       {2, 1},
	OpGetFreePtr:    {1},
	OpGetFree:       {1},
	OpSetFree:       {1},
	OpGetLocalPtr:   {1},
	OpSetSelFree:    {1, 1},
	OpIteratorInit:  {},
	OpIteratorNext:  {},
	OpIteratorKey:   {},
	OpIteratorValue: {},
	OpBinaryOp:      {1},
	OpSuspend:       {},
	OpSelect:        {},
	OpMethodCall:    {2, 1, 1}, // method const index, numArgs, spread
}

// ReadOperands reads operands from the bytecode.
func ReadOperands(numOperands []int, ins []byte) ([]int, int) {
	operands := make([]int, 0, len(numOperands))
	var offset int
	for _, width := range numOperands {
		switch width {
		case 1:
			operands = append(operands, int(ins[offset]))
		case 2:
			operands = append(operands, int(ins[offset+1])|int(ins[offset])<<8)
		case 4:
			operands = append(operands, int(ins[offset+3])|int(ins[offset+2])<<8|int(ins[offset+1])<<16|int(ins[offset])<<24)
		}
		offset += width
	}
	return operands, offset
}
