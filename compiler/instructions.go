package compiler

import (
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
)

func NewAbortCheck() bc.Instruction {
	return bc.Instruction{Op: bc.AbortCheck}
}

func NewSuspend() bc.Instruction {
	return bc.Instruction{Op: bc.Suspend}
}

func NewReturn(hasResult bool) bc.Instruction {
	if hasResult {
		return bc.Instruction{Op: bc.Return, Op1: 1}
	}
	return bc.Instruction{Op: bc.Return, Op1: 0}
}

func NewPop() bc.Instruction {
	return bc.Instruction{Op: bc.Pop}
}

func NewUnaryNeg() bc.Instruction {
	return bc.Instruction{Op: bc.UnaryNeg}
}

func NewUnaryNot() bc.Instruction {
	return bc.Instruction{Op: bc.UnaryNot}
}

func NewUnaryBitNot() bc.Instruction {
	return bc.Instruction{Op: bc.UnaryBitNot}
}

func NewEqual() bc.Instruction {
	return bc.Instruction{Op: bc.Equal}
}

func NewNotEqual() bc.Instruction {
	return bc.Instruction{Op: bc.NotEqual}
}

func NewContains() bc.Instruction {
	return bc.Instruction{Op: bc.Contains}
}

func NewImmutable() bc.Instruction {
	return bc.Instruction{Op: bc.Immutable}
}

func NewAccessIndex() bc.Instruction {
	return bc.Instruction{Op: bc.AccessIndex}
}

func NewAccessSelector() bc.Instruction {
	return bc.Instruction{Op: bc.AccessSelector}
}

func NewSlice() bc.Instruction {
	return bc.Instruction{Op: bc.Slice}
}

func NewSliceStep() bc.Instruction {
	return bc.Instruction{Op: bc.SliceStep}
}

func NewIterInit() bc.Instruction {
	return bc.Instruction{Op: bc.IterInit}
}

func NewIterNext() bc.Instruction {
	return bc.Instruction{Op: bc.IterNext}
}

func NewIterKey() bc.Instruction {
	return bc.Instruction{Op: bc.IterKey}
}

func NewIterValue() bc.Instruction {
	return bc.Instruction{Op: bc.IterValue}
}

func NewFormatRuntimeSpec() bc.Instruction {
	return bc.Instruction{Op: bc.FormatRuntimeSpec}
}

func NewFormatStaticSpec(formatSpecStaticIndex int) bc.Instruction {
	return bc.Instruction{Op: bc.FormatStaticSpec, Op3: uint32(formatSpecStaticIndex)}
}

func NewBinaryOp(tokenID token.Token) bc.Instruction {
	return bc.Instruction{Op: bc.BinaryOp, Op1: uint8(tokenID)}
}

func NewImportBuiltinModule(moduleStaticID int) bc.Instruction {
	return bc.Instruction{Op: bc.ImportBuiltinModule, Op3: uint32(moduleStaticID)}
}

func NewDefineLocal(localIndex int) bc.Instruction {
	return bc.Instruction{Op: bc.DefineLocal, Op3: uint32(localIndex)}
}

func NewLoadLocal(localIndex int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadLocal, Op3: uint32(localIndex)}
}

func NewStoreLocal(localIndex int) bc.Instruction {
	return bc.Instruction{Op: bc.StoreLocal, Op3: uint32(localIndex)}
}

func NewStoreIndexedLocal(localIndex int, numSelectors int) bc.Instruction {
	return bc.Instruction{Op: bc.StoreIndexedLocal, Op3: uint32(localIndex), Op2: uint16(numSelectors)}
}

func NewLoadFree(freeIndex int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadFree, Op3: uint32(freeIndex)}
}

func NewStoreFree(freeIndex int) bc.Instruction {
	return bc.Instruction{Op: bc.StoreFree, Op3: uint32(freeIndex)}
}

func NewStoreIndexedFree(freeIndex int, numSelectors int) bc.Instruction {
	return bc.Instruction{Op: bc.StoreIndexedFree, Op3: uint32(freeIndex), Op2: uint16(numSelectors)}
}

func NewLoadLocalPtr(localIndex int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadLocalPtr, Op3: uint32(localIndex)}
}

func NewLoadFreePtr(freeIndex int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadFreePtr, Op3: uint32(freeIndex)}
}

func NewLoadBuiltinFunction(builtinFuncID int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadBuiltinFunction, Op3: uint32(builtinFuncID)}
}

func NewMakeClosure(staticFuncIndex int, numFreeVars int) bc.Instruction {
	return bc.Instruction{Op: bc.MakeClosure, Op3: uint32(staticFuncIndex), Op2: uint16(numFreeVars)}
}

func NewLoadGlobal(globalIndex int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadGlobal, Op3: uint32(globalIndex)}
}

func NewStoreGlobal(globalIndex int) bc.Instruction {
	return bc.Instruction{Op: bc.StoreGlobal, Op3: uint32(globalIndex)}
}

func NewStoreIndexedGlobal(globalIndex int, numSelectors int) bc.Instruction {
	return bc.Instruction{Op: bc.StoreIndexedGlobal, Op3: uint32(globalIndex), Op2: uint16(numSelectors)}
}

func NewMakeArray(numElements int) bc.Instruction {
	return bc.Instruction{Op: bc.MakeArray, Op3: uint32(numElements)}
}

func NewMakeRecord(numFields int) bc.Instruction {
	return bc.Instruction{Op: bc.MakeRecord, Op3: uint32(numFields)}
}

func NewCallFunction(numArgs int, isSpread bool) bc.Instruction {
	if isSpread {
		return bc.Instruction{Op: bc.CallFunction, Op2: uint16(numArgs), Op1: 1}
	}
	return bc.Instruction{Op: bc.CallFunction, Op2: uint16(numArgs), Op1: 0}
}

func NewCallMethod(methodIndex int, numArgs int, isSpread bool) bc.Instruction {
	if isSpread {
		return bc.Instruction{Op: bc.CallMethod, Op3: uint32(methodIndex), Op2: uint16(numArgs), Op1: 1}
	}
	return bc.Instruction{Op: bc.CallMethod, Op3: uint32(methodIndex), Op2: uint16(numArgs), Op1: 0}
}

func NewDefer(numArgs int) bc.Instruction {
	return bc.Instruction{Op: bc.Defer, Op2: uint16(numArgs)}
}

func NewDeferMethod(methodIndex int, numArgs int) bc.Instruction {
	return bc.Instruction{Op: bc.DeferMethod, Op3: uint32(methodIndex), Op2: uint16(numArgs)}
}

func NewJump(target int) bc.Instruction {
	return bc.Instruction{Op: bc.Jump, Op3: uint32(int32(target))}
}

func NewJumpFalsy(target int) bc.Instruction {
	return bc.Instruction{Op: bc.JumpFalsy, Op3: uint32(int32(target))}
}

func NewAndJump(target int) bc.Instruction {
	return bc.Instruction{Op: bc.AndJump, Op3: uint32(int32(target))}
}

func NewOrJump(target int) bc.Instruction {
	return bc.Instruction{Op: bc.OrJump, Op3: uint32(int32(target))}
}

func NewPushUndefined() bc.Instruction {
	return bc.Instruction{Op: bc.PushUndefined}
}

func NewPushBool(b bool) bc.Instruction {
	if b {
		return bc.Instruction{Op: bc.PushBool, Op1: 1}
	}
	return bc.Instruction{Op: bc.PushBool, Op1: 0}
}

func NewPushByte(i byte) bc.Instruction {
	return bc.Instruction{Op: bc.PushByte, Op1: uint8(i)}
}

func NewPushRune(i rune) bc.Instruction {
	return bc.Instruction{Op: bc.PushRune, Op3: uint32(i)}
}

func NewPushInt(i int32) bc.Instruction {
	return bc.Instruction{Op: bc.PushInt, Op3: uint32(i)}
}

func NewLoadStaticDecimal(i int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadStaticDecimal, Op3: uint32(i)}
}

func NewLoadStaticString(i int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadStaticString, Op3: uint32(i)}
}

func NewLoadStaticRunes(i int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadStaticRunes, Op3: uint32(i)}
}

func NewLoadStaticBytes(i int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadStaticBytes, Op3: uint32(i)}
}

func NewLoadStaticTime(i int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadStaticTime, Op3: uint32(i)}
}

func NewLoadStaticFormatSpec(i int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadStaticFormatSpec, Op3: uint32(i)}
}

func NewLoadStaticCompiledFunction(i int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadStaticCompiledFunction, Op3: uint32(i)}
}

func NewLoadStaticPrimitive(i int) bc.Instruction {
	return bc.Instruction{Op: bc.LoadStaticPrimitive, Op3: uint32(i)}
}
