package core

import (
	"bytes"
	"fmt"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/binary"
)

type CompiledFunction struct {
	Instructions  []byte
	Free          []*Value
	SourceMap     map[int]Pos
	NumLocals     int // number of local variables (including function parameters)
	MaxStack      int // estimated maximum operand-stack depth which can be reached during execution
	NumParameters int8
	VarArgs       bool
	NamedResult   int8 // local-slot index of function's named result: 0 = no named result, N > 0 means slot N-1
}

// NB: Free compared by length only
func (o *CompiledFunction) Equal(other CompiledFunction) bool {
	if !bytes.Equal(o.Instructions, other.Instructions) {
		return false
	}
	if len(o.Free) != len(other.Free) {
		return false
	}
	if len(o.SourceMap) != len(other.SourceMap) {
		return false
	}
	for k, v := range o.SourceMap {
		if otherV, ok := other.SourceMap[k]; !ok || otherV != v {
			return false
		}
	}
	if o.NumLocals != other.NumLocals || o.MaxStack != other.MaxStack || o.NumParameters != other.NumParameters || o.VarArgs != other.VarArgs || o.NamedResult != other.NamedResult {
		return false
	}
	return true
}

func (o *CompiledFunction) Set(instructions []byte, free []*Value, sourceMap map[int]Pos, numLocals, maxStack int, numParameters int8, varArgs bool, namedResult int8) {
	o.Instructions = instructions
	o.Free = free
	o.SourceMap = sourceMap
	o.NumLocals = numLocals
	o.MaxStack = maxStack
	o.NumParameters = numParameters
	o.VarArgs = varArgs
	o.NamedResult = namedResult
}

func (o *CompiledFunction) HasNamedResult() bool {
	return o.NamedResult != 0
}

// NamedResultSlot returns the local-slot index of the named result. Caller should check HasNamedResult first.
func (o *CompiledFunction) NamedResultSlot() int {
	return int(o.NamedResult) - 1
}

func (o *CompiledFunction) Size() int64 {
	return int64(len(o.Instructions) + len(o.Free) + len(o.SourceMap))
}

func (o *CompiledFunction) EncodeBinary(a *Arena) ([]byte, error) {
	b := binary.AppendBytes(nil, o.Instructions)

	b = binary.AppendUint64(b, uint64(len(o.Free)))
	for i, fv := range o.Free {
		val := Undefined
		if fv != nil {
			val = *fv
		}
		eb, err := val.EncodeBinary(a)
		if err != nil {
			return nil, fmt.Errorf("compiled function free value at index %d: %w", i, err)
		}
		b = binary.AppendBytes(b, eb)
	}

	b = binary.AppendUint64(b, uint64(len(o.SourceMap)))
	for ip, pos := range o.SourceMap {
		b = binary.AppendUint64(b, uint64(ip))
		b = binary.AppendUint64(b, uint64(pos))
	}

	b = binary.AppendUint64(b, uint64(o.NumLocals))
	b = binary.AppendUint64(b, uint64(o.MaxStack))
	b = append(b, byte(o.NumParameters))
	if o.VarArgs {
		b = append(b, 1)
	} else {
		b = append(b, 0)
	}
	b = append(b, byte(o.NamedResult))
	return b, nil
}

func (o *CompiledFunction) DecodeBinary(a *Arena, data []byte) error {
	offset := 0

	insts, err := binary.ReadBytes(data, &offset, "compiled function instructions")
	if err != nil {
		return err
	}
	o.Instructions = append(o.Instructions[:0], insts...)

	freeCount, err := binary.ReadUint64(data, &offset, "compiled function free values count")
	if err != nil {
		return err
	}
	if freeCount == 0 {
		o.Free = nil
	} else {
		o.Free = make([]*Value, int(freeCount))
		for i := range o.Free {
			eb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("compiled function free value at index %d", i))
			if err != nil {
				return err
			}
			var fv Value
			if err := fv.DecodeBinary(a, eb); err != nil {
				return fmt.Errorf("compiled function free value at index %d: %w", i, err)
			}
			o.Free[i] = &fv
		}
	}

	sourceMapCount, err := binary.ReadUint64(data, &offset, "compiled function source map count")
	if err != nil {
		return err
	}
	if sourceMapCount == 0 {
		o.SourceMap = nil
	} else {
		o.SourceMap = make(map[int]Pos, int(sourceMapCount))
		for i := 0; i < int(sourceMapCount); i++ {
			ip, err := binary.ReadUint64(data, &offset, fmt.Sprintf("compiled function source map entry %d ip", i))
			if err != nil {
				return err
			}
			pos, err := binary.ReadUint64(data, &offset, fmt.Sprintf("compiled function source map entry %d pos", i))
			if err != nil {
				return err
			}
			o.SourceMap[int(ip)] = Pos(pos)
		}
	}

	numLocals, err := binary.ReadUint64(data, &offset, "compiled function num locals")
	if err != nil {
		return err
	}
	maxStack, err := binary.ReadUint64(data, &offset, "compiled function max stack")
	if err != nil {
		return err
	}
	if len(data)-offset < 3 {
		return fmt.Errorf("compiled function: expected 3 bytes for parameters/flags, got %d", len(data)-offset)
	}
	o.NumLocals = int(numLocals)
	o.MaxStack = int(maxStack)
	o.NumParameters = int8(data[offset])
	o.VarArgs = data[offset+1] != 0
	o.NamedResult = int8(data[offset+2])
	offset += 3

	if offset != len(data) {
		return fmt.Errorf("compiled function: trailing %d bytes", len(data)-offset)
	}
	return nil
}

func (o *CompiledFunction) SourcePos(ip int) Pos {
	for ip >= 0 {
		if p, ok := o.SourceMap[ip]; ok {
			return p
		}
		ip--
	}
	return NoPos
}

var TypeCompiledFunction = ValueTypeDescr{
	Pin:          func(a *Arena, v Value) { a.PinCompiledFunctionValue(v) },
	Retain:       func(a *Arena, v Value) { a.RetainCompiledFunctionValue(v) },
	Release:      func(a *Arena, v Value) { a.ReleaseCompiledFunctionValue(v) },
	Name:         compiledFunctionTypeName,
	String:       func(a *Arena, v Value) string { return compiledFunctionTypeName(a, v) },
	EncodeBinary: compiledFunctionTypeEncodeBinary,
	DecodeBinary: compiledFunctionTypeDecodeBinary,
	IsTrue:       ConstHook(true),
	IsCallable:   ConstHook(true),
	IsVariadic:   compiledFunctionTypeIsVariadic,
	Equal:        compiledFunctionTypeEqual,
	Arity:        compiledFunctionTypeArity,
	Call:         compiledFunctionTypeCall,
	MethodCall:   compiledFunctionTypeMethodCall,
}

func compiledFunctionTypeEqual(a *Arena, v Value, r Value) bool {
	if r.Type != VT_COMPILED_FUNCTION {
		return false
	}
	x := a.ResolveCompiledFunctionValue(v)
	y := a.ResolveCompiledFunctionValue(r)
	return x == y
}

func compiledFunctionTypeName(a *Arena, v Value) string {
	o := a.ResolveCompiledFunctionValue(v)
	if o.VarArgs {
		return fmt.Sprintf("<compiled-function/%d+>", o.NumParameters)
	}
	return fmt.Sprintf("<compiled-function/%d>", o.NumParameters)
}

func compiledFunctionTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
	return a.ResolveCompiledFunctionValue(v).EncodeBinary(a)
}

func compiledFunctionTypeDecodeBinary(a *Arena, v *Value, data []byte) error {
	f, err := a.NewCompiledFunctionValue(nil, nil, nil, 0, 0, 0, false, 0)
	if err != nil {
		return fmt.Errorf("compiled function: %w", err)
	}
	if err := f.DecodeBinary(a, data); err != nil {
		a.ReleaseCompiledFunctionValue(f)
		return fmt.Errorf("compiled function: %w", err)
	}
	// we are not releasing old value here because it should be managed by caller Value.DecodeBinary
	*v = f
	return nil
}

func compiledFunctionTypeCall(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	return vm.Call(v, args)
}

func compiledFunctionTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
	}
}

func compiledFunctionTypeIsVariadic(a *Arena, v Value) bool {
	return a.ResolveCompiledFunctionValue(v).VarArgs
}

func compiledFunctionTypeArity(a *Arena, v Value) int8 {
	return a.ResolveCompiledFunctionValue(v).NumParameters
}
