package core

import (
	"fmt"
	"unsafe"

	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/binary"
)

type CompiledFunction struct {
	Instructions  bc.Instructions
	Free          []*Value
	SourceMap     map[int]Pos
	NumLocals     int // number of local variables (including function parameters)
	MaxStack      int // estimated maximum operand-stack depth which can be reached during execution
	NumParameters int
	NamedResult   int // local-slot index of function's named result: 0 = no named result, N > 0 means slot N-1
	VarArgs       bool
}

func NewStaticCompiledFunctionValue(cf *CompiledFunction) Value {
	return Value{Type: value.CompiledFunction, Immutable: true, Ptr: unsafe.Pointer(cf)}
}

func NewCompiledFunctionValue(
	instructions bc.Instructions,
	free []*Value,
	sourceMap map[int]Pos,
	numLocals int,
	maxStack int,
	numParameters int,
	namedResult int,
	varArgs bool,
) Value {
	o := &CompiledFunction{}
	o.Set(instructions, free, sourceMap, numLocals, maxStack, numParameters, namedResult, varArgs)
	return Value{Type: value.CompiledFunction, Immutable: true, Ptr: unsafe.Pointer(o)}
}

func (o *CompiledFunction) Set(instructions bc.Instructions, free []*Value, sourceMap map[int]Pos, numLocals, maxStack int, numParameters int, namedResult int, varArgs bool) {
	o.Instructions = instructions
	o.Free = free
	o.SourceMap = sourceMap
	o.NumLocals = numLocals
	o.MaxStack = maxStack
	o.NumParameters = numParameters
	o.NamedResult = namedResult
	o.VarArgs = varArgs
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

// GobEncode wraps binary encoding so gob does not reflect over fields like Free []*Value (which include unsafe.Pointer).
func (o CompiledFunction) GobEncode() ([]byte, error) {
	return o.EncodeBinary()
}

// GobDecode wraps binary decoding to mirror GobEncode.
func (o *CompiledFunction) GobDecode(data []byte) error {
	if o == nil {
		return fmt.Errorf("compiled function: nil GobDecode receiver")
	}
	return o.DecodeBinary(data)
}

func (o *CompiledFunction) EncodeBinary() ([]byte, error) {
	var b []byte

	t, err := o.Instructions.EncodeBinary()
	if err != nil {
		return nil, fmt.Errorf("compiled function instructions: %w", err)
	}
	b = binary.AppendUint64(b, uint64(len(t)))
	b = append(b, t...)

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
	b = binary.AppendUint64(b, uint64(o.NumParameters))
	b = binary.AppendUint64(b, uint64(o.NamedResult))

	if o.VarArgs {
		b = append(b, 1)
	} else {
		b = append(b, 0)
	}

	return b, nil
}

func (o *CompiledFunction) DecodeBinary(data []byte) error {
	offset := 0

	isWidth, err := binary.ReadUint64(data, &offset, "compiled function instructions")
	if err != nil {
		return err
	}
	var instructions bc.Instructions
	if err := instructions.DecodeBinary(data[offset : offset+int(isWidth)]); err != nil {
		return fmt.Errorf("compiled function instructions: %w", err)
	}
	o.Instructions = instructions
	offset += int(isWidth)

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
	numParameters, err := binary.ReadUint64(data, &offset, "compiled function num parameters")
	if err != nil {
		return err
	}
	namedResult, err := binary.ReadUint64(data, &offset, "compiled function named result")
	if err != nil {
		return err
	}
	varArgs := false
	if offset < len(data) {
		varArgs = data[offset] != 0
		offset++
	}

	o.NumLocals = int(numLocals)
	o.MaxStack = int(maxStack)
	o.NumParameters = int(numParameters)
	o.NamedResult = int(namedResult)
	o.VarArgs = varArgs

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
	Name:         compiledFunctionTypeName,                                    // PURE by contract
	String:       func(v Value) string { return compiledFunctionTypeName(v) }, // PURE by contract
	EncodeBinary: compiledFunctionTypeEncodeBinary,                            // PURE by contract
	DecodeBinary: compiledFunctionTypeDecodeBinary,                            // IMPURE by contract (mutates target)
	IsTrue:       ConstHook(true),                                             // PURE by contract
	IsCallable:   ConstHook(true),                                             // PURE by contract
	IsVariadic:   compiledFunctionTypeIsVariadic,                              // PURE by contract
	Equal:        compiledFunctionTypeEqual,                                   // PURE by contract
	Arity:        compiledFunctionTypeArity,                                   // PURE by contract
	Call:         compiledFunctionTypeCall,                                    // CALLABLE-DEPENDENT by contract
	MethodCall:   compiledFunctionTypeMethodCall,                              // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	IsMethodPure: func(string) bool { return true },                           // All methods are expected to be pure.
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
	f := NewCompiledFunctionValue(nil, nil, nil, 0, 0, 0, 0, false)
	if err := f.DecodeBinary(data); err != nil {
		return fmt.Errorf("compiled function: %w", err)
	}
	*v = f
	return nil
}

// CALLABLE-DEPENDENT: purity is a property of the specific user-defined function. The optimizer folds Call only
// when the interprocedural pass has proven the CompiledFunction pure. See docs/purity.md.
func compiledFunctionTypeCall(vm VM, v Value, args []Value) (Value, error) {
	return vm.Call(v, args)
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
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

func compiledFunctionTypeArity(v Value) int {
	return (*CompiledFunction)(v.Ptr).NumParameters
}
