package compiler

import (
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/bytecode/opcodes"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/token"
)

func NewAbortCheck() bc.Instruction {
	return bc.Instruction{Op: opcodes.AbortCheck}
}

func NewSuspend() bc.Instruction {
	return bc.Instruction{Op: opcodes.Suspend}
}

func NewReturn(hasResult bool) bc.Instruction {
	if hasResult {
		return bc.Instruction{Op: opcodes.Return, Op1: 1}
	}
	return bc.Instruction{Op: opcodes.Return, Op1: 0}
}

func NewPop() bc.Instruction {
	return bc.Instruction{Op: opcodes.Pop}
}

func NewUnaryNeg() bc.Instruction {
	return bc.Instruction{Op: opcodes.UnaryNeg}
}

func NewUnaryNot() bc.Instruction {
	return bc.Instruction{Op: opcodes.UnaryNot}
}

func NewUnaryBitNot() bc.Instruction {
	return bc.Instruction{Op: opcodes.UnaryBitNot}
}

func NewEqual() bc.Instruction {
	return bc.Instruction{Op: opcodes.Equal}
}

func NewNotEqual() bc.Instruction {
	return bc.Instruction{Op: opcodes.NotEqual}
}

func NewContains() bc.Instruction {
	return bc.Instruction{Op: opcodes.Contains}
}

func NewImmutable(deep bool) bc.Instruction {
	if deep {
		return bc.Instruction{Op: opcodes.Immutable, Op1: 1}
	}
	return bc.Instruction{Op: opcodes.Immutable, Op1: 0}
}

func NewAccessIndex() bc.Instruction {
	return bc.Instruction{Op: opcodes.AccessIndex}
}

func NewSlice() bc.Instruction {
	return bc.Instruction{Op: opcodes.Slice}
}

func NewSliceStep() bc.Instruction {
	return bc.Instruction{Op: opcodes.SliceStep}
}

func NewIterInit() bc.Instruction {
	return bc.Instruction{Op: opcodes.IterInit}
}

func NewIterNext() bc.Instruction {
	return bc.Instruction{Op: opcodes.IterNext}
}

func NewIterKey() bc.Instruction {
	return bc.Instruction{Op: opcodes.IterKey}
}

func NewIterValue() bc.Instruction {
	return bc.Instruction{Op: opcodes.IterValue}
}

func NewIterElem() bc.Instruction {
	return bc.Instruction{Op: opcodes.IterElem}
}

func NewFormatRuntimeSpec() bc.Instruction {
	return bc.Instruction{Op: opcodes.FormatRuntimeSpec}
}

func NewFormatStaticSpec(formatSpecStaticIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.FormatStaticSpec, Op3: uint32(formatSpecStaticIndex)}
}

func NewBinaryOp(tokenID token.Token) bc.Instruction {
	return bc.Instruction{Op: opcodes.BinaryOp, Op1: uint8(tokenID)}
}

func NewImportBuiltinModule(moduleStaticID int) bc.Instruction {
	return bc.Instruction{Op: opcodes.ImportBuiltinModule, Op3: uint32(moduleStaticID)}
}

func NewDefineLocal(localIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.DefineLocal, Op3: uint32(localIndex)}
}

func NewLoadLocal(localIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadLocal, Op3: uint32(localIndex)}
}

func NewStoreLocal(localIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.StoreLocal, Op3: uint32(localIndex)}
}

func NewLoadFree(freeIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadFree, Op3: uint32(freeIndex)}
}

func NewStoreFree(freeIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.StoreFree, Op3: uint32(freeIndex)}
}

func NewLoadLocalPtr(localIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadLocalPtr, Op3: uint32(localIndex)}
}

func NewLoadFreePtr(freeIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadFreePtr, Op3: uint32(freeIndex)}
}

func NewLoadBuiltinFunction(builtinFuncID int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadBuiltinFunction, Op3: uint32(builtinFuncID)}
}

func NewMakeClosure(staticFuncIndex int, numFreeVars int) bc.Instruction {
	return bc.Instruction{Op: opcodes.MakeClosure, Op3: uint32(staticFuncIndex), Op2: uint16(numFreeVars)}
}

func NewLoadGlobal(globalIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadGlobal, Op3: uint32(globalIndex)}
}

func NewStoreGlobal(globalIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.StoreGlobal, Op3: uint32(globalIndex)}
}

func NewMakeArray(numElements int) bc.Instruction {
	return bc.Instruction{Op: opcodes.MakeArray, Op3: uint32(numElements)}
}

func NewMakeRecord(numFields int) bc.Instruction {
	return bc.Instruction{Op: opcodes.MakeRecord, Op3: uint32(numFields)}
}

func NewCallFunction(numArgs int, isSpread bool) bc.Instruction {
	if isSpread {
		return bc.Instruction{Op: opcodes.CallFunction, Op2: uint16(numArgs), Op1: 1}
	}
	return bc.Instruction{Op: opcodes.CallFunction, Op2: uint16(numArgs), Op1: 0}
}

// NewCallMember emits x.name(args): nargs ≤ 255 (the compiler checks), id the bound member ID or member.Unknown,
// nameIndex the static string index of the name — the name is authoritative, id only a hint.
func NewCallMember(numArgs int, id member.ID, nameIndex int, isSpread bool) bc.Instruction {
	var op bc.Opcode = opcodes.CallMember
	if isSpread {
		op = opcodes.CallMemberSpread
	}
	return bc.Instruction{Op: op, Op1: uint8(numArgs), Op2: uint16(id), Op3: uint32(nameIndex)}
}

func NewDeferMember(numArgs int, id member.ID, nameIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.DeferMember, Op1: uint8(numArgs), Op2: uint16(id), Op3: uint32(nameIndex)}
}

func NewAccessProperty(id member.ID, nameIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.AccessProperty, Op2: uint16(id), Op3: uint32(nameIndex)}
}

func NewAssignProperty(id member.ID, nameIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.AssignProperty, Op2: uint16(id), Op3: uint32(nameIndex)}
}

func NewAssignIndex() bc.Instruction {
	return bc.Instruction{Op: opcodes.AssignIndex}
}

func NewDefer(numArgs int) bc.Instruction {
	return bc.Instruction{Op: opcodes.Defer, Op2: uint16(numArgs)}
}

func NewJump(target int) bc.Instruction {
	return bc.Instruction{Op: opcodes.Jump, Op3: uint32(int32(target))}
}

func NewJumpFalsy(target int) bc.Instruction {
	return bc.Instruction{Op: opcodes.JumpFalsy, Op3: uint32(int32(target))}
}

func NewAndJump(target int) bc.Instruction {
	return bc.Instruction{Op: opcodes.AndJump, Op3: uint32(int32(target))}
}

func NewOrJump(target int) bc.Instruction {
	return bc.Instruction{Op: opcodes.OrJump, Op3: uint32(int32(target))}
}

func NewPushUndefined() bc.Instruction {
	return bc.Instruction{Op: opcodes.PushUndefined}
}

func NewPushBool(b bool) bc.Instruction {
	if b {
		return bc.Instruction{Op: opcodes.PushBool, Op1: 1}
	}
	return bc.Instruction{Op: opcodes.PushBool, Op1: 0}
}

func NewPushByte(i byte) bc.Instruction {
	return bc.Instruction{Op: opcodes.PushByte, Op1: uint8(i)}
}

func NewPushRune(i rune) bc.Instruction {
	return bc.Instruction{Op: opcodes.PushRune, Op3: uint32(i)}
}

func NewPushInt(i int32) bc.Instruction {
	return bc.Instruction{Op: opcodes.PushInt, Op3: uint32(i)}
}

func NewLoadStaticDecimal(i int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadStaticDecimal, Op3: uint32(i)}
}

func NewLoadStaticString(i int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadStaticString, Op3: uint32(i)}
}

func NewLoadStaticRunes(i int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadStaticRunes, Op3: uint32(i)}
}

func NewLoadStaticBytes(i int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadStaticBytes, Op3: uint32(i)}
}

func NewLoadStaticTime(i int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadStaticTime, Op3: uint32(i)}
}

func NewLoadStaticFormatSpec(i int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadStaticFormatSpec, Op3: uint32(i)}
}

func NewLoadStaticCompiledFunction(i int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadStaticCompiledFunction, Op3: uint32(i)}
}

func NewLoadStaticPrimitive(i int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadStaticPrimitive, Op3: uint32(i)}
}

func NewLoadStaticRange(i int) bc.Instruction {
	return bc.Instruction{Op: opcodes.LoadStaticRange, Op3: uint32(i)}
}

func NewUnpack(count int, nameListIndex int) bc.Instruction {
	return bc.Instruction{Op: opcodes.Unpack, Op1: uint8(count), Op3: uint32(nameListIndex)}
}
