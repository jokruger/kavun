package bc

import "fmt"

type Opcode = byte

// List of opcodes
const (
	OpConstant       = Opcode(0)  // Load constant
	OpBComplement    = Opcode(1)  // bitwise complement
	OpPop            = Opcode(2)  // Pop
	OpTrue           = Opcode(3)  // Push true
	OpFalse          = Opcode(4)  // Push false
	OpEqual          = Opcode(5)  // Equal ==
	OpNotEqual       = Opcode(6)  // Not equal !=
	OpMinus          = Opcode(7)  // Minus -
	OpLNot           = Opcode(8)  // Logical not !
	OpJumpFalsy      = Opcode(9)  // Jump if falsy
	OpAndJump        = Opcode(10) // Logical AND jump
	OpOrJump         = Opcode(11) // Logical OR jump
	OpJump           = Opcode(12) // Jump
	OpNull           = Opcode(13) // Push null
	OpArray          = Opcode(14) // Array object
	OpRecord         = Opcode(15) // Record object
	OpContains       = Opcode(16) // Contains operation (x in y)
	OpImmutable      = Opcode(17) // Immutable object
	OpIndex          = Opcode(18) // Index operation
	OpSliceIndex     = Opcode(19) // Slice operation
	OpCall           = Opcode(20) // Call function
	OpReturn         = Opcode(21) // Return
	OpGetGlobal      = Opcode(22) // Get global variable
	OpSetGlobal      = Opcode(23) // Set global variable
	OpSetSelGlobal   = Opcode(24) // Set global variable using selectors
	OpGetLocal       = Opcode(25) // Get local variable
	OpSetLocal       = Opcode(26) // Set local variable
	OpDefineLocal    = Opcode(27) // Define local variable
	OpSetSelLocal    = Opcode(28) // Set local variable using selectors
	OpGetFreePtr     = Opcode(29) // Get free variable pointer object
	OpGetFree        = Opcode(30) // Get free variables
	OpSetFree        = Opcode(31) // Set free variables
	OpGetLocalPtr    = Opcode(32) // Get local variable as a pointer
	OpSetSelFree     = Opcode(33) // Set free variables using selectors
	OpGetBuiltin     = Opcode(34) // Get builtin function
	OpClosure        = Opcode(35) // Push closure
	OpIteratorInit   = Opcode(36) // Iterator init
	OpIteratorNext   = Opcode(37) // Iterator next
	OpIteratorKey    = Opcode(38) // Iterator key
	OpIteratorValue  = Opcode(39) // Iterator value
	OpBinaryOp       = Opcode(40) // Binary operation
	OpSuspend        = Opcode(41) // Suspend VM
	OpSelect         = Opcode(42) // Select operation
	OpMethodCall     = Opcode(43) // Call method on object
	OpSliceIndexStep = Opcode(44) // Slice with step
	OpFormat         = Opcode(45) // Format value with pre-parsed FormatSpec constant
	OpFormatDyn      = Opcode(46) // Format value with runtime-built FormatSpec string popped from the stack
	OpDefer          = Opcode(47) // Register deferred call: pop callee + N args, store on current frame
	OpDeferMethod    = Opcode(48) // Register deferred method call: pop receiver + N args; method name from constants[methodIdx]
	// 49...255 are reserved for future use
)

// OpcodeNames are string representation of opcodes.
var OpcodeNames = [...]string{
	OpConstant:       "CONST",
	OpPop:            "POP",
	OpTrue:           "TRUE",
	OpFalse:          "FALSE",
	OpBComplement:    "NEG",
	OpEqual:          "EQL",
	OpNotEqual:       "NEQ",
	OpMinus:          "NEG",
	OpLNot:           "NOT",
	OpJumpFalsy:      "JMPF",
	OpAndJump:        "ANDJMP",
	OpOrJump:         "ORJMP",
	OpJump:           "JMP",
	OpNull:           "NULL",
	OpGetGlobal:      "GETG",
	OpSetGlobal:      "SETG",
	OpSetSelGlobal:   "SETSG",
	OpArray:          "ARR",
	OpRecord:         "RECORD",
	OpImmutable:      "IMMUT",
	OpIndex:          "INDEX",
	OpSliceIndex:     "SLICE",
	OpCall:           "CALL",
	OpSliceIndexStep: "SLICESTEP",
	OpReturn:         "RET",
	OpGetLocal:       "GETL",
	OpSetLocal:       "SETL",
	OpDefineLocal:    "DEFL",
	OpSetSelLocal:    "SETSL",
	OpGetBuiltin:     "BUILTIN",
	OpClosure:        "CLOSURE",
	OpGetFreePtr:     "GETFP",
	OpGetFree:        "GETF",
	OpSetFree:        "SETF",
	OpGetLocalPtr:    "GETLP",
	OpSetSelFree:     "SETSF",
	OpIteratorInit:   "ITER",
	OpIteratorNext:   "ITNXT",
	OpIteratorKey:    "ITKEY",
	OpIteratorValue:  "ITVAL",
	OpBinaryOp:       "BINARYOP",
	OpSuspend:        "SUSPEND",
	OpSelect:         "SELECT",
	OpMethodCall:     "MCALL",
	OpContains:       "CONTAINS",
	OpFormat:         "FMT",
	OpFormatDyn:      "FMTDYN",
	OpDefer:          "DEFER",
	OpDeferMethod:    "DEFERM",
}

// OpcodeOperands describes the number and shape of opcode operands
var OpcodeOperands = [...][]int{
	OpConstant:       {2}, // index
	OpPop:            {},
	OpTrue:           {},
	OpFalse:          {},
	OpBComplement:    {},
	OpEqual:          {},
	OpNotEqual:       {},
	OpMinus:          {},
	OpLNot:           {},
	OpJumpFalsy:      {2}, // new pos
	OpAndJump:        {2}, // new pos
	OpOrJump:         {2}, // new pos
	OpJump:           {2}, // new pos
	OpNull:           {},
	OpGetGlobal:      {2},    // index
	OpSetGlobal:      {2},    // index
	OpSetSelGlobal:   {2, 1}, // index, num selectors
	OpArray:          {2},    // num elements (inline init)
	OpRecord:         {2},    // num elements (inline init)
	OpImmutable:      {},
	OpIndex:          {},
	OpSliceIndex:     {},
	OpCall:           {1, 1}, // num args, is spread (0 or 1)
	OpSliceIndexStep: {},
	OpReturn:         {1},    // has result (0 or 1)
	OpGetLocal:       {1},    // index
	OpSetLocal:       {1},    // index
	OpDefineLocal:    {1},    // index
	OpSetSelLocal:    {1, 1}, // index, num selectors
	OpGetBuiltin:     {1},    // index
	OpClosure:        {2, 1}, // num args, is spread (0 or 1)
	OpGetFreePtr:     {1},    // index
	OpGetFree:        {1},    // index
	OpSetFree:        {1},    // index
	OpGetLocalPtr:    {1},    // index
	OpSetSelFree:     {1, 1}, // index, num selectors
	OpIteratorInit:   {},
	OpIteratorNext:   {},
	OpIteratorKey:    {},
	OpIteratorValue:  {},
	OpBinaryOp:       {1}, // token
	OpSuspend:        {},
	OpSelect:         {},
	OpMethodCall:     {2, 1, 1}, // method const index, num args, spread
	OpContains:       {},
	OpFormat:         {2},    // format spec constant index
	OpFormatDyn:      {},     // pops spec string and value from stack, pushes formatted string
	OpDefer:          {1},    // numArgs (callee + args popped from stack at runtime)
	OpDeferMethod:    {2, 1}, // method const index, num args
}

// ReadOperands reads operands from the bytecode.
func ReadOperands(numOperands []int, ins []byte) ([]int, int, error) {
	operands := make([]int, 0, len(numOperands))
	var offset int
	for _, width := range numOperands {
		switch width {
		case 1:
			operands = append(operands, int(ins[offset]))
		case 2:
			operands = append(operands, int(ins[offset+1])|int(ins[offset])<<8)
		default:
			return nil, 0, fmt.Errorf("unsupported operand width: %d", width)
		}
		offset += width
	}
	return operands, offset, nil
}
