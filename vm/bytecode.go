package vm

import (
	"encoding/gob"
	"fmt"
	"io"
	"reflect"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/value"
)

// Bytecode is a compiled instructions and constants.
type Bytecode struct {
	FileSet      *parser.SourceFileSet
	MainFunction *value.CompiledFunction
	Constants    []core.Object
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

// FormatConstants returns human readable string representations of
// compiled constants.
func (b *Bytecode) FormatConstants() (output []string) {
	for cidx, cn := range b.Constants {
		switch cn := cn.(type) {
		case *value.CompiledFunction:
			output = append(output, fmt.Sprintf("[% 3d] (Compiled Function|%p)", cidx, &cn))
			for _, l := range FormatInstructions(cn.Instructions, 0) {
				output = append(output, fmt.Sprintf("     %s", l))
			}
		default:
			output = append(output, fmt.Sprintf("[% 3d] %s (%s|%p)",
				cidx, cn, reflect.TypeOf(cn).Elem().Name(), &cn))
		}
	}
	return
}

// Decode reads Bytecode data from the reader.
func (b *Bytecode) Decode(r io.Reader, modules *ModuleMap) error {
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
		fv, err := fixDecodedObject(v, modules)
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
	var deduped []core.Object

	indexMap := make(map[int]int) // mapping from old constant index to new index
	fns := make(map[*value.CompiledFunction]int)
	ints := make(map[int64]int)
	strings := make(map[string]int)
	floats := make(map[float64]int)
	chars := make(map[rune]int)
	immutableRecords := make(map[string]int) // for modules

	for curIdx, c := range b.Constants {
		switch c := c.(type) {
		case *value.CompiledFunction:
			if newIdx, ok := fns[c]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				fns[c] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case *value.Record:
			if !c.IsImmutable() {
				panic(fmt.Errorf("unsupported top-level constant type: %s", c.TypeName()))
			}
			modName := inferModuleName(c)
			newIdx, ok := immutableRecords[modName]
			if modName != "" && ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				immutableRecords[modName] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case *value.Int:
			if newIdx, ok := ints[c.Value()]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				ints[c.Value()] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case *value.String:
			if newIdx, ok := strings[c.Value()]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				strings[c.Value()] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case *value.Float:
			if newIdx, ok := floats[c.Value()]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				floats[c.Value()] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case *value.Char:
			if newIdx, ok := chars[c.Value()]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				chars[c.Value()] = newIdx
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
		switch c := c.(type) {
		case *value.CompiledFunction:
			updateConstIndexes(c.Instructions, indexMap)
		}
	}
}

func fixDecodedObject(o core.Object, modules *ModuleMap) (core.Object, error) {
	switch o := o.(type) {
	case *value.Bool:
		if o.IsFalsy() {
			return value.FalseValue, nil
		}
		return value.TrueValue, nil

	case *value.Undefined:
		return value.UndefinedValue, nil

	case *value.Array:
		for i, v := range o.Value() {
			fv, err := fixDecodedObject(v, modules)
			if err != nil {
				return nil, err
			}
			o.SetAt(i, fv)
		}

	case *value.Record:
		if o.IsImmutable() {
			modName := inferModuleName(o)
			if mod := modules.GetBuiltinModule(modName); mod != nil {
				return mod.AsImmutableRecord(modName), nil
			}

			for k, v := range o.Value() {
				// encoding of user function not supported
				if _, isBuiltinFunction := v.(*value.BuiltinFunction); isBuiltinFunction {
					return nil, fmt.Errorf("user function not decodable")
				}

				fv, err := fixDecodedObject(v, modules)
				if err != nil {
					return nil, err
				}
				o.SetKey(k, fv)
			}
		} else {
			for k, v := range o.Value() {
				fv, err := fixDecodedObject(v, modules)
				if err != nil {
					return nil, err
				}
				o.SetKey(k, fv)
			}
		}

	case *value.Map:
		for k, v := range o.Value() {
			fv, err := fixDecodedObject(v, modules)
			if err != nil {
				return nil, err
			}
			o.SetKey(k, fv)
		}
	}
	return o, nil
}

func updateConstIndexes(insts []byte, indexMap map[int]int) {
	i := 0
	for i < len(insts) {
		op := insts[i]
		numOperands := parser.OpcodeOperands[op]
		_, read := parser.ReadOperands(numOperands, insts[i+1:])

		switch op {
		case parser.OpConstant:
			curIdx := int(insts[i+2]) | int(insts[i+1])<<8
			newIdx, ok := indexMap[curIdx]
			if !ok {
				panic(fmt.Errorf("constant index not found: %d", curIdx))
			}
			copy(insts[i:], MakeInstruction(op, newIdx))
		case parser.OpClosure:
			curIdx := int(insts[i+2]) | int(insts[i+1])<<8
			numFree := int(insts[i+3])
			newIdx, ok := indexMap[curIdx]
			if !ok {
				panic(fmt.Errorf("constant index not found: %d", curIdx))
			}
			copy(insts[i:], MakeInstruction(op, newIdx, numFree))
		}

		i += 1 + read
	}
}

func inferModuleName(mod *value.Record) string {
	mn, ok := mod.Get("__module_name__")
	if !ok {
		return ""
	}
	if s, ok := mn.AsString(); ok {
		return s
	}
	return ""
}

func init() {
	gob.Register(&parser.SourceFileSet{})
	gob.Register(&parser.SourceFile{})

	gob.Register(&value.CompiledFunction{})
	gob.Register(&value.BuiltinFunction{})

	gob.Register(&value.Bool{})
	gob.Register(&value.Char{})
	gob.Register(&value.Int{})
	gob.Register(&value.Float{})
	gob.Register(&value.Bytes{})
	gob.Register(&value.String{})
	gob.Register(&value.Time{})
	gob.Register(&value.Array{})
	gob.Register(&value.Record{})
	gob.Register(&value.Map{})
	gob.Register(&value.Error{})
	gob.Register(&value.Undefined{})
}
