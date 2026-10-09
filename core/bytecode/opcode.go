package bytecode

import (
	"fmt"

	"github.com/jokruger/kavun/core/bytecode/opcodes"
)

type Opcode byte

func (op Opcode) String() string {
	if int(op) >= len(opInfo) || opInfo[op].Name == "" {
		return fmt.Sprintf("UNKNOWN_OPCODE %d", op)
	}
	return opInfo[op].Name
}

func (op Opcode) Class() OpClass {
	if int(op) >= len(opInfo) {
		return OpUnknown
	}
	return opInfo[op].Class
}

type OpClass byte

const (
	OpUnknown       = OpClass(0)
	OpFallThrough   = OpClass(1) // Proceed to next instruction
	OpConditional   = OpClass(2) // Conditional jump
	OpUnconditional = OpClass(3) // Unconditional jump
	OpTerminating   = OpClass(4) // Terminating instruction (return, abort, etc.)
)

type OpDescr struct {
	Name  string
	Class OpClass // Instruction class (fall-through, conditional jump, etc.)
}

// opInfo describes every opcode, indexed by it.
var opInfo = [...]OpDescr{
	opcodes.AbortCheck:                 {"ABORT_CHECK", OpFallThrough},
	opcodes.Suspend:                    {"SUSPEND", OpTerminating},
	opcodes.Return:                     {"RETURN", OpTerminating},
	opcodes.Jump:                       {"JUMP", OpUnconditional},
	opcodes.JumpFalsy:                  {"JUMP_FALSY", OpConditional},
	opcodes.AndJump:                    {"AND_JUMP", OpConditional},
	opcodes.OrJump:                     {"OR_JUMP", OpConditional},
	opcodes.Pop:                        {"POP", OpFallThrough},
	opcodes.Unpack:                     {"UNPACK", OpFallThrough},
	opcodes.Immutable:                  {"IMMUTABLE", OpFallThrough},
	opcodes.UnaryNeg:                   {"UNARY_NEG", OpFallThrough},
	opcodes.UnaryNot:                   {"UNARY_NOT", OpFallThrough},
	opcodes.UnaryBitNot:                {"UNARY_BITNOT", OpFallThrough},
	opcodes.Equal:                      {"EQUAL", OpFallThrough},
	opcodes.NotEqual:                   {"NOT_EQUAL", OpFallThrough},
	opcodes.Contains:                   {"CONTAINS", OpFallThrough},
	opcodes.Slice:                      {"SLICE", OpFallThrough},
	opcodes.SliceStep:                  {"SLICE_STEP", OpFallThrough},
	opcodes.IterInit:                   {"ITER_INIT", OpFallThrough},
	opcodes.IterNext:                   {"ITER_NEXT", OpFallThrough},
	opcodes.IterKey:                    {"ITER_KEY", OpFallThrough},
	opcodes.IterValue:                  {"ITER_VALUE", OpFallThrough},
	opcodes.IterElem:                   {"ITER_ELEM", OpFallThrough},
	opcodes.AccessIndex:                {"ACCESS_INDEX", OpFallThrough},
	opcodes.AccessProperty:             {"ACCESS_PROPERTY", OpFallThrough},
	opcodes.AssignIndex:                {"ASSIGN_INDEX", OpFallThrough},
	opcodes.AssignProperty:             {"ASSIGN_PROPERTY", OpFallThrough},
	opcodes.BinaryOp:                   {"BINARY_OP", OpFallThrough},
	opcodes.CallFunction:               {"CALL_FUNCTION", OpFallThrough},
	opcodes.CallMember:                 {"CALL_MEMBER", OpFallThrough},
	opcodes.CallMemberSpread:           {"CALL_MEMBER_SPREAD", OpFallThrough},
	opcodes.Defer:                      {"DEFER", OpFallThrough},
	opcodes.DeferMember:                {"DEFER_MEMBER", OpFallThrough},
	opcodes.FormatRuntimeSpec:          {"FORMAT_RUNTIME_SPEC", OpFallThrough},
	opcodes.FormatStaticSpec:           {"FORMAT_STATIC_SPEC", OpFallThrough},
	opcodes.ImportBuiltinModule:        {"IMPORT_BUILTIN_MODULE", OpFallThrough},
	opcodes.DefineLocal:                {"DEFINE_LOCAL", OpFallThrough},
	opcodes.StoreLocal:                 {"STORE_LOCAL", OpFallThrough},
	opcodes.StoreFree:                  {"STORE_FREE", OpFallThrough},
	opcodes.StoreGlobal:                {"STORE_GLOBAL", OpFallThrough},
	opcodes.MakeClosure:                {"MAKE_CLOSURE", OpFallThrough},
	opcodes.MakeArray:                  {"MAKE_ARRAY", OpFallThrough},
	opcodes.MakeRecord:                 {"MAKE_RECORD", OpFallThrough},
	opcodes.PushUndefined:              {"PUSH_UNDEFINED", OpFallThrough},
	opcodes.PushBool:                   {"PUSH_BOOL", OpFallThrough},
	opcodes.PushByte:                   {"PUSH_BYTE", OpFallThrough},
	opcodes.PushRune:                   {"PUSH_RUNE", OpFallThrough},
	opcodes.PushInt:                    {"PUSH_INT", OpFallThrough},
	opcodes.LoadLocal:                  {"LOAD_LOCAL", OpFallThrough},
	opcodes.LoadLocalPtr:               {"LOAD_LOCAL_PTR", OpFallThrough},
	opcodes.LoadFree:                   {"LOAD_FREE", OpFallThrough},
	opcodes.LoadFreePtr:                {"LOAD_FREE_PTR", OpFallThrough},
	opcodes.LoadBuiltinFunction:        {"LOAD_BUILTIN_FUNCTION", OpFallThrough},
	opcodes.LoadGlobal:                 {"LOAD_GLOBAL", OpFallThrough},
	opcodes.LoadStaticDecimal:          {"LOAD_STATIC_DECIMAL", OpFallThrough},
	opcodes.LoadStaticString:           {"LOAD_STATIC_STRING", OpFallThrough},
	opcodes.LoadStaticRunes:            {"LOAD_STATIC_RUNES", OpFallThrough},
	opcodes.LoadStaticBytes:            {"LOAD_STATIC_BYTES", OpFallThrough},
	opcodes.LoadStaticTime:             {"LOAD_STATIC_TIME", OpFallThrough},
	opcodes.LoadStaticFormatSpec:       {"LOAD_STATIC_FORMAT_SPEC", OpFallThrough},
	opcodes.LoadStaticRange:            {"LOAD_STATIC_RANGE", OpFallThrough},
	opcodes.LoadStaticCompiledFunction: {"LOAD_STATIC_COMPILED_FUNCTION", OpFallThrough},
	opcodes.LoadStaticPrimitive:        {"LOAD_STATIC_PRIMITIVE", OpFallThrough},
}
