package vm

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser"
)

// Bytecode is a compiled instructions and constants.
type Bytecode struct {
	FileSet      *parser.SourceFileSet
	MainFunction *core.CompiledFunction
	Constants    []core.Value
}

func appendBinaryUint64(b []byte, v uint64) []byte {
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], v)
	return append(b, tmp[:]...)
}

func appendBinaryBytes(b []byte, payload []byte) []byte {
	b = appendBinaryUint64(b, uint64(len(payload)))
	return append(b, payload...)
}

func readBinaryUint64(data []byte, offset *int, field string) (uint64, error) {
	if len(data)-*offset < 8 {
		return 0, fmt.Errorf("%s: expected 8 bytes, got %d", field, len(data)-*offset)
	}
	v := binary.BigEndian.Uint64(data[*offset : *offset+8])
	*offset += 8
	return v, nil
}

func readBinaryBytes(data []byte, offset *int, field string) ([]byte, error) {
	l, err := readBinaryUint64(data, offset, field+" (length)")
	if err != nil {
		return nil, err
	}
	remaining := len(data) - *offset
	if l > uint64(remaining) {
		return nil, fmt.Errorf("%s: declared length %d exceeds remaining data %d", field, l, remaining)
	}
	end := *offset + int(l)
	b := data[*offset:end]
	*offset = end
	return b, nil
}

// Size of the bytecode in bytes
// (as much as we can calculate it without reflection and black magic)
func (b *Bytecode) Size() int64 {
	return b.MainFunction.Size() + b.FileSet.Size() + int64(len(b.Constants))
}

// Encode writes Bytecode data to the writer.
func (b *Bytecode) Encode(w io.Writer) error {
	arena := core.NewArena(nil)

	var fileSetBuf bytes.Buffer
	if err := gob.NewEncoder(&fileSetBuf).Encode(b.FileSet); err != nil {
		return err
	}

	enc := gob.NewEncoder(w)
	if err := enc.Encode(fileSetBuf.Bytes()); err != nil {
		return err
	}

	mainBytes, err := b.MainFunction.EncodeBinary(arena)
	if err != nil {
		return err
	}
	if err := enc.Encode(mainBytes); err != nil {
		return err
	}

	constants := appendBinaryUint64(nil, uint64(len(b.Constants)))
	for i, c := range b.Constants {
		cb, err := c.EncodeBinary(arena)
		if err != nil {
			return fmt.Errorf("bytecode constant at index %d: %w", i, err)
		}
		constants = appendBinaryBytes(constants, cb)
	}
	return enc.Encode(constants)
}

// CountObjects returns the number of objects found in Constants.
func (b *Bytecode) CountObjects() int {
	n := 0
	for _, c := range b.Constants {
		n += CountObjects(c)
	}
	return n
}

// MustFormatInstructions returns human readable string representations of compiled instructions.
func (b *Bytecode) MustFormatInstructions() []string {
	r, err := FormatInstructions(b.MainFunction.Instructions, 0)
	if err != nil {
		panic(fmt.Errorf("failed to format instructions: %w", err))
	}
	return r
}

// FormatInstructions returns human readable string representations of compiled instructions.
func (b *Bytecode) FormatInstructions() ([]string, error) {
	return FormatInstructions(b.MainFunction.Instructions, 0)
}

// MustFormatConstants returns human readable string representations of compiled constants.
func (b *Bytecode) MustFormatConstants(a *core.Arena) []string {
	r, err := b.FormatConstants(a)
	if err != nil {
		panic(fmt.Errorf("failed to format constants: %w", err))
	}
	return r
}

// FormatConstants returns human readable string representations of compiled constants.
func (b *Bytecode) FormatConstants(a *core.Arena) (output []string, err error) {
	for cidx, cn := range b.Constants {
		if cn.Type == core.VT_COMPILED_FUNCTION {
			f := (*core.CompiledFunction)(cn.Ptr)
			output = append(output, fmt.Sprintf("[% 3d] (Compiled Function|%p)", cidx, f))
			t, err := FormatInstructions(f.Instructions, 0)
			if err != nil {
				return nil, err
			}
			for _, l := range t {
				output = append(output, fmt.Sprintf("     %s", l))
			}
			continue
		}
		output = append(output, fmt.Sprintf("[% 3d] %s (%s|%v)", cidx, cn.String(a), cn.TypeName(a), cn))
	}
	return
}

// Decode reads Bytecode data from the reader.
func (b *Bytecode) Decode(alloc *core.Arena, r io.Reader, modules *ModuleMap) error {
	if modules == nil {
		modules = NewModuleMap()
	}

	dec := gob.NewDecoder(r)
	var fileSetData []byte
	if err := dec.Decode(&fileSetData); err != nil {
		return err
	}
	b.FileSet = parser.NewFileSet()
	if len(fileSetData) > 0 {
		if err := gob.NewDecoder(bytes.NewReader(fileSetData)).Decode(&b.FileSet); err != nil {
			return err
		}
	}
	// TODO: files in b.FileSet.File does not have their 'set' field properly
	//  set to b.FileSet as it's private field and not serialized by gob
	//  encoder/decoder.

	var mainData []byte
	if err := dec.Decode(&mainData); err != nil {
		return err
	}
	b.MainFunction = &core.CompiledFunction{}
	if err := b.MainFunction.DecodeBinary(alloc, mainData); err != nil {
		return err
	}

	var constantsData []byte
	if err := dec.Decode(&constantsData); err != nil {
		return err
	}
	offset := 0
	count, err := readBinaryUint64(constantsData, &offset, "bytecode constants count")
	if err != nil {
		return err
	}
	if count == 0 {
		b.Constants = nil
	} else {
		b.Constants = make([]core.Value, int(count))
		for i := range b.Constants {
			cb, err := readBinaryBytes(constantsData, &offset, fmt.Sprintf("bytecode constant at index %d", i))
			if err != nil {
				return err
			}
			if err := b.Constants[i].DecodeBinary(alloc, cb); err != nil {
				return fmt.Errorf("bytecode constant at index %d: %w", i, err)
			}
		}
	}
	if offset != len(constantsData) {
		return fmt.Errorf("bytecode constants: trailing %d bytes", len(constantsData)-offset)
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
func (b *Bytecode) RemoveDuplicates(a *core.Arena) error {
	var deduped []core.Value

	indexMap := make(map[int]int) // mapping from old constant index to new index
	fns := make(map[*core.CompiledFunction]int)
	ints := make(map[uint64]int)
	strings := make(map[string]int)
	decimals := make(map[string]int)
	times := make(map[string]int)
	runes := make(map[string]int)
	floats := make(map[uint64]int)
	chars := make(map[uint64]int)
	bools := make(map[uint64]int)
	formatSpecs := make(map[string]int)
	immutableRecords := make(map[string]int) // for modules

	for curIdx, c := range b.Constants {
		switch c.Type {
		case core.VT_INT:
			if newIdx, ok := ints[c.Data]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				ints[c.Data] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_FLOAT:
			if newIdx, ok := floats[c.Data]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				floats[c.Data] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_DECIMAL:
			ds := (*dec128.Dec128)(c.Ptr).String()
			if newIdx, ok := decimals[ds]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				decimals[ds] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_RUNES:
			ds := string((*core.Runes)(c.Ptr).Elements)
			if newIdx, ok := runes[ds]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				runes[ds] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_TIME:
			ds := (*time.Time)(c.Ptr).String()
			if newIdx, ok := times[ds]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				times[ds] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_RUNE:
			if newIdx, ok := chars[c.Data]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				chars[c.Data] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_BOOL:
			if newIdx, ok := bools[c.Data]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				bools[c.Data] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_COMPILED_FUNCTION:
			cf := (*core.CompiledFunction)(c.Ptr)
			if newIdx, ok := fns[cf]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				fns[cf] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_RECORD:
			if !c.Immutable {
				panic(fmt.Errorf("unsupported top-level constant type: %s", c.TypeName(a)))
			}
			cr := (*core.Dict)(c.Ptr)
			modName := inferModuleName(a, cr)
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
			cs := (*core.String)(c.Ptr).Value
			if newIdx, ok := strings[cs]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				strings[cs] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		case core.VT_FORMAT_SPEC:
			fs := (*core.FormatSpecValue)(c.Ptr)
			if newIdx, ok := formatSpecs[fs.Text]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				formatSpecs[fs.Text] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}

		default:
			panic(fmt.Errorf("unsupported top-level constant type: %s", c.TypeName(a)))
		}
	}

	// replace with de-duplicated constants
	b.Constants = deduped

	// update CONST instructions with new indexes

	// main function
	if err := updateConstIndexes(b.MainFunction.Instructions, indexMap); err != nil {
		return err
	}

	// other compiled functions in constants
	for _, c := range b.Constants {
		if c.Type == core.VT_COMPILED_FUNCTION {
			if err := updateConstIndexes((*core.CompiledFunction)(c.Ptr).Instructions, indexMap); err != nil {
				return err
			}
		}
	}

	return nil
}

func fixDecodedObject(a *core.Arena, v core.Value, modules *ModuleMap) (core.Value, error) {
	switch v.Type {
	case core.VT_ARRAY:
		o := (*core.Array)(v.Ptr)
		for i, v := range o.Elements {
			fv, err := fixDecodedObject(a, v, modules)
			if err != nil {
				return core.Undefined, err
			}
			o.Elements[i] = fv
		}

	case core.VT_RECORD:
		if v.Immutable {
			o := (*core.Dict)(v.Ptr)
			modName := inferModuleName(a, o)
			if mod := modules.GetBuiltinModule(modName); mod != nil {
				return mod.AsImmutableRecord(a, modName)
			}
			for k, v := range o.Elements {
				// encoding of user function not supported
				if v.Type == core.VT_BUILTIN_FUNCTION || v.Type == core.VT_COMPILED_FUNCTION {
					return core.Undefined, fmt.Errorf("user function not decodable")
				}
				fv, err := fixDecodedObject(a, v, modules)
				if err != nil {
					return core.Undefined, err
				}
				o.Elements[k] = fv
			}
		} else {
			o := (*core.Dict)(v.Ptr)
			for k, v := range o.Elements {
				fv, err := fixDecodedObject(a, v, modules)
				if err != nil {
					return core.Undefined, err
				}
				o.Elements[k] = fv
			}
		}

	case core.VT_DICT:
		o := (*core.Dict)(v.Ptr)
		for k, v := range o.Elements {
			fv, err := fixDecodedObject(a, v, modules)
			if err != nil {
				return core.Undefined, err
			}
			o.Elements[k] = fv
		}
	}

	return v, nil
}

func updateConstIndexes(insts []byte, indexMap map[int]int) error {
	i := 0
	for i < len(insts) {
		op := insts[i]
		numOperands := bc.OpcodeOperands[op]
		operands, read, err := bc.ReadOperands(numOperands, insts[i+1:])
		if err != nil {
			return err
		}

		switch op {
		case bc.OpConstant:
			curIdx := operands[0]
			newIdx, ok := indexMap[curIdx]
			if !ok {
				panic(fmt.Errorf("constant index not found: %d", curIdx))
			}
			t, err := MakeInstruction(op, newIdx)
			if err != nil {
				return err
			}
			copy(insts[i:], t)

		case bc.OpClosure:
			curIdx := operands[0]
			numFree := operands[1]
			newIdx, ok := indexMap[curIdx]
			if !ok {
				panic(fmt.Errorf("constant index not found: %d", curIdx))
			}
			t, err := MakeInstruction(op, newIdx, numFree)
			if err != nil {
				return err
			}
			copy(insts[i:], t)

		case bc.OpMethodCall:
			curIdx := operands[0]
			numArgs := operands[1]
			spread := operands[2]
			newIdx, ok := indexMap[curIdx]
			if !ok {
				panic(fmt.Errorf("constant index not found: %d", curIdx))
			}
			t, err := MakeInstruction(op, newIdx, numArgs, spread)
			if err != nil {
				return err
			}
			copy(insts[i:], t)

		case bc.OpFormat:
			curIdx := operands[0]
			newIdx, ok := indexMap[curIdx]
			if !ok {
				panic(fmt.Errorf("constant index not found: %d", curIdx))
			}
			t, err := MakeInstruction(op, newIdx)
			if err != nil {
				return err
			}
			copy(insts[i:], t)

		case bc.OpDeferMethod:
			curIdx := operands[0]
			numArgs := operands[1]
			newIdx, ok := indexMap[curIdx]
			if !ok {
				panic(fmt.Errorf("constant index not found: %d", curIdx))
			}
			t, err := MakeInstruction(op, newIdx, numArgs)
			if err != nil {
				return err
			}
			copy(insts[i:], t)
		}

		i += 1 + read
	}

	return nil
}

func inferModuleName(a *core.Arena, mod *core.Dict) string {
	mn, ok := mod.Elements["__module_name__"]
	if !ok {
		return ""
	}
	if s, ok := mn.AsString(a); ok {
		return s
	}
	return ""
}
