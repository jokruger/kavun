package vm

import (
	"encoding/gob"
	"fmt"
	"io"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
)

// Bytecode is a compiled instructions and constants.
type Bytecode struct {
	FileSet      *parser.SourceFileSet
	MainFunction *core.CompiledFunction
	Constants    []core.Value
}

// Size of the bytecode in bytes
// (as much as we can calculate it without reflection and black magic)
func (b *Bytecode) Size() int64 {
	return b.MainFunction.Size() + b.FileSet.Size() + int64(len(b.Constants))
}

// Encode writes Bytecode data to the writer.
func (b *Bytecode) Encode(w io.Writer) error {
	enc := gob.NewEncoder(w)
	if err := enc.Encode(b.FileSet); err != nil {
		return err
	}
	if err := enc.Encode(b.MainFunction); err != nil {
		return err
	}
	return enc.Encode(b.Constants)
}

// CountObjects returns the number of objects found in Constants.
func (b *Bytecode) CountObjects() int {
	n := 0
	for _, c := range b.Constants {
		n += CountObjects(c)
	}
	return n
}

// FormatInstructions returns human readable string representations of
// compiled instructions.
func (b *Bytecode) FormatInstructions() []string {
	return FormatInstructions(b.MainFunction.Instructions, 0)
}

// FormatConstants returns human readable string representations of compiled constants.
func (b *Bytecode) FormatConstants() (output []string) {
	for cidx, cn := range b.Constants {
		if cn.IsCompiledFunction() {
			f := core.ToCompiledFunction(cn)
			output = append(output, fmt.Sprintf("[% 3d] (Compiled Function|%p)", cidx, f))
			for _, l := range FormatInstructions(f.Instructions, 0) {
				output = append(output, fmt.Sprintf("     %s", l))
			}
			continue
		}
		output = append(output, fmt.Sprintf("[% 3d] %s (%s|%v)", cidx, cn.String(), cn.TypeName(), cn))
	}
	return
}

// Decode reads Bytecode data from the reader.
func (b *Bytecode) Decode(alloc core.Allocator, r io.Reader, modules *ModuleMap) error {
	if modules == nil {
		modules = NewModuleMap()
	}

	dec := gob.NewDecoder(r)
	if err := dec.Decode(&b.FileSet); err != nil {
		return err
	}
	// TODO: files in b.FileSet.File does not have their 'set' field properly
	//  set to b.FileSet as it's private field and not serialized by gob
	//  encoder/decoder.
	if err := dec.Decode(&b.MainFunction); err != nil {
		return err
	}
	if err := dec.Decode(&b.Constants); err != nil {
		return err
	}
	for i, v := range b.Constants {
		fv, err := fixDecodedObject(alloc, v, modules)
		if err != nil {
			return err
		}
		b.Constants[i] = fv
	}
	return nil
}

// RemoveDuplicates finds and remove the duplicate values in Constants.
// Note this function mutates Bytecode.
func (b *Bytecode) RemoveDuplicates() {
	var deduped []core.Value

	indexMap := make(map[int]int) // mapping from old constant index to new index
	fns := make(map[*core.CompiledFunction]int)
	ints := make(map[int64]int)
	strings := make(map[string]int)
	floats := make(map[float64]int)
	chars := make(map[rune]int)
	bools := make(map[bool]int)
	immutableRecords := make(map[string]int) // for modules

	for curIdx, c := range b.Constants {
		switch c.Type {
		case core.VT_INT:
			if newIdx, ok := ints[core.ToInt(c)]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				ints[core.ToInt(c)] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_FLOAT:
			if newIdx, ok := floats[core.ToFloat(c)]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				floats[core.ToFloat(c)] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_CHAR:
			if newIdx, ok := chars[core.ToChar(c)]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				chars[core.ToChar(c)] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_BOOL:
			if newIdx, ok := bools[core.ToBool(c)]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				bools[core.ToBool(c)] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_COMPILED_FUNCTION:
			cf := core.ToCompiledFunction(c)
			if newIdx, ok := fns[cf]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				fns[cf] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_RECORD:
			cr := (*core.Record)(c.Ptr)
			if !c.IsImmutable() {
				panic(fmt.Errorf("unsupported top-level constant type: %s", c.TypeName()))
			}
			modName := inferModuleName(cr)
			newIdx, ok := immutableRecords[modName]
			if modName != "" && ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				immutableRecords[modName] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_STRING:
			cs := core.ToString(c).Value
			if newIdx, ok := strings[cs]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				strings[cs] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		default:
			panic(fmt.Errorf("unsupported top-level constant type: %s", c.TypeName()))
		}
	}

	// replace with de-duplicated constants
	b.Constants = deduped

	// update CONST instructions with new indexes
	// main function
	updateConstIndexes(b.MainFunction.Instructions, indexMap)
	// other compiled functions in constants
	for _, c := range b.Constants {
		if c.IsCompiledFunction() {
			updateConstIndexes(core.ToCompiledFunction(c).Instructions, indexMap)
		}
	}
}

func fixDecodedObject(alloc core.Allocator, v core.Value, modules *ModuleMap) (core.Value, error) {
	switch v.Type {
	case core.VT_ARRAY:
		o := (*core.Array)(v.Ptr)
		for i, v := range o.Elements {
			fv, err := fixDecodedObject(alloc, v, modules)
			if err != nil {
				return core.UndefinedValue(), err
			}
			o.Elements[i] = fv
		}

	case core.VT_RECORD:
		o := (*core.Record)(v.Ptr)
		if v.IsImmutable() {
			modName := inferModuleName(o)
			if mod := modules.GetBuiltinModule(modName); mod != nil {
				return mod.AsImmutableRecord(alloc, modName), nil
			}

			for k, v := range o.Elements {
				// encoding of user function not supported
				if v.IsBuiltinFunction() {
					return core.UndefinedValue(), fmt.Errorf("user function not decodable")
				}

				fv, err := fixDecodedObject(alloc, v, modules)
				if err != nil {
					return core.UndefinedValue(), err
				}
				o.Elements[k] = fv
			}
		} else {
			for k, v := range o.Elements {
				fv, err := fixDecodedObject(alloc, v, modules)
				if err != nil {
					return core.UndefinedValue(), err
				}
				o.Elements[k] = fv
			}
		}

	case core.VT_MAP:
		o := (*core.Map)(v.Ptr)
		for k, v := range o.Elements {
			fv, err := fixDecodedObject(alloc, v, modules)
			if err != nil {
				return core.UndefinedValue(), err
			}
			o.Elements[k] = fv
		}
	}

	return v, nil
}

func updateConstIndexes(insts []byte, indexMap map[int]int) {
	i := 0
	for i < len(insts) {
		op := insts[i]
		numOperands := core.OpcodeOperands[op]
		operands, read := core.ReadOperands(numOperands, insts[i+1:])

		switch op {
		case core.OpConstant:
			curIdx := operands[0]
			newIdx, ok := indexMap[curIdx]
			if !ok {
				panic(fmt.Errorf("constant index not found: %d", curIdx))
			}
			copy(insts[i:], MakeInstruction(op, newIdx))

		case core.OpClosure:
			curIdx := operands[0]
			numFree := operands[1]
			newIdx, ok := indexMap[curIdx]
			if !ok {
				panic(fmt.Errorf("constant index not found: %d", curIdx))
			}
			copy(insts[i:], MakeInstruction(op, newIdx, numFree))

		case core.OpMethodCall:
			curIdx := operands[0]
			numArgs := operands[1]
			spread := operands[2]
			newIdx, ok := indexMap[curIdx]
			if !ok {
				panic(fmt.Errorf("constant index not found: %d", curIdx))
			}
			copy(insts[i:], MakeInstruction(op, newIdx, numArgs, spread))
		}

		i += 1 + read
	}
}

func inferModuleName(mod *core.Record) string {
	mn, ok := mod.Elements["__module_name__"]
	if !ok {
		return ""
	}
	if s, ok := mn.AsString(); ok {
		return s
	}
	return ""
}

func init() {
	gob.Register(&core.Value{})
	gob.Register(&core.CompiledFunction{})
	gob.Register(&parser.SourceFileSet{})
	gob.Register(&parser.SourceFile{})
}
