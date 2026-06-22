package core

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
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

func NewCompiledFunctionValue(
	instructions []byte,
	free []*Value,
	sourceMap map[int]Pos,
	numLocals int,
	maxStack int,
	numParameters int8,
	varArgs bool,
	namedResult int8,
) Value {
	o := &CompiledFunction{}
	o.Set(instructions, free, sourceMap, numLocals, maxStack, numParameters, varArgs, namedResult)
	return Value{Type: value.CompiledFunction, Immutable: true, Ptr: unsafe.Pointer(o)}
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

func (o *CompiledFunction) EncodeBinary() ([]byte, error) {
	b := binary.AppendBytes(nil, o.Instructions)

	b = binary.AppendUint64(b, uint64(len(o.Free)))
	for i, fv := range o.Free {
		val := Undefined
		if fv != nil {
			val = *fv
		}
		eb, err := val.EncodeBinary()
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

func (o *CompiledFunction) DecodeBinary(data []byte) error {
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
			if err := fv.DecodeBinary(eb); err != nil {
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
	Name:         compiledFunctionTypeName,
	String:       func(v Value) string { return compiledFunctionTypeName(v) },
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

func compiledFunctionTypeEqual(v Value, r Value) bool {
	if r.Type != value.CompiledFunction {
		return false
	}
	x := (*CompiledFunction)(v.Ptr)
	y := (*CompiledFunction)(r.Ptr)
	return x == y
}

func compiledFunctionTypeName(v Value) string {
	o := (*CompiledFunction)(v.Ptr)
	if o.VarArgs {
		return fmt.Sprintf("<compiled-function/%d+>", o.NumParameters)
	}
	return fmt.Sprintf("<compiled-function/%d>", o.NumParameters)
}

func compiledFunctionTypeEncodeBinary(v Value) ([]byte, error) {
	return (*CompiledFunction)(v.Ptr).EncodeBinary()
}

func compiledFunctionTypeDecodeBinary(v *Value, data []byte) error {
	f := NewCompiledFunctionValue(nil, nil, nil, 0, 0, 0, false, 0)
	if err := f.DecodeBinary(data); err != nil {
		return fmt.Errorf("compiled function: %w", err)
	}
	*v = f
	return nil
}

func compiledFunctionTypeCall(vm VM, v Value, args []Value) (Value, error) {
	return vm.Call(v, args)
}

func compiledFunctionTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func compiledFunctionTypeIsVariadic(v Value) bool {
	return (*CompiledFunction)(v.Ptr).VarArgs
}

func compiledFunctionTypeArity(v Value) int8 {
	return (*CompiledFunction)(v.Ptr).NumParameters
}
