package vm

import (
	"encoding/gob"
	"fmt"
	"io"
	"strings"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// BytecodeMagic opens every encoded Bytecode: a fixed prefix plus a format version. Decode refuses any other
// header, so bytecode from an incompatible build fails loudly instead of decoding into different instructions.
//
// CHANGE THE VERSION EVERY TIME BYTECODE COMPATIBILITY BREAKS: an opcode added, removed or renumbered (core/bytecode),
// an instruction's operand layout or meaning changed, a field of Bytecode/core.Static/core.CompiledFunction changed,
// or any type's EncodeBinary/DecodeBinary format changed. Keep the length: Decode reads exactly len(BytecodeMagic).
const BytecodeMagic = bytecodeMagicPrefix + "001"

const bytecodeMagicPrefix = "KVN"

// Bytecode is a compiled instructions and constants.
type Bytecode struct {
	FileSet      *ast.SourceFileSet
	MainFunction *core.CompiledFunction
	Static       core.Static
}

// Encode writes Bytecode data to the writer.
func (b *Bytecode) Encode(w io.Writer) error {
	// validate main function - it should not be nil and should not have free variables
	if b.MainFunction == nil {
		return fmt.Errorf("main function is nil")
	}
	if len(b.MainFunction.Free) > 0 {
		return fmt.Errorf("main function should not have free variables, but has %d", len(b.MainFunction.Free))
	}

	// validate static - compiled functions in static should not have free variables
	for i, cf := range b.Static.CompiledFunctions {
		if len(cf.Free) > 0 {
			return fmt.Errorf("compiled function at static index %d should not have free variables, but has %d", i, len(cf.Free))
		}
	}

	// encode bytecode: the format header, then the gob stream
	if _, err := io.WriteString(w, BytecodeMagic); err != nil {
		return fmt.Errorf("failed to encode bytecode: %w", err)
	}
	enc := gob.NewEncoder(w)
	if err := enc.Encode(*b); err != nil {
		return fmt.Errorf("failed to encode bytecode: %w", err)
	}
	return nil
}

// Decode reads Bytecode data from the reader.
// NB: files in b.FileSet.File does not have their 'set' field properly set to b.FileSet as it's private field and not
// serialized by gob encoder/decoder.
func (b *Bytecode) Decode(r io.Reader) error {
	header := make([]byte, len(BytecodeMagic))
	if _, err := io.ReadFull(r, header); err != nil {
		return fmt.Errorf("failed to decode bytecode: not Kavun bytecode (missing header): %w", err)
	}
	if h := string(header); h != BytecodeMagic {
		if strings.HasPrefix(h, bytecodeMagicPrefix) {
			return fmt.Errorf("failed to decode bytecode: incompatible bytecode version %s (this build reads %s); recompile from source", h[len(bytecodeMagicPrefix):], BytecodeMagic[len(bytecodeMagicPrefix):])
		}
		return fmt.Errorf("failed to decode bytecode: not Kavun bytecode (bad header %q)", h)
	}

	dec := gob.NewDecoder(r)
	if err := dec.Decode(b); err != nil {
		return fmt.Errorf("failed to decode bytecode: %w", err)
	}

	// validate main function - it should not be nil and should not have free variables
	if b.MainFunction == nil {
		return fmt.Errorf("main function is nil")
	}
	if len(b.MainFunction.Free) > 0 {
		return fmt.Errorf("main function should not have free variables, but has %d", len(b.MainFunction.Free))
	}

	// validate static - compiled functions in static should not have free variables
	for i, cf := range b.Static.CompiledFunctions {
		if len(cf.Free) > 0 {
			return fmt.Errorf("compiled function at static index %d should not have free variables, but has %d", i, len(cf.Free))
		}
	}

	// derived data the hot loop relies on (the cached string rune counts); idempotent
	b.Static.BuildStringLens()

	return nil
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

// MustFormatStatics returns human readable string representations of compiled static values.
func (b *Bytecode) MustFormatStatics() []string {
	r, err := b.FormatStatics()
	if err != nil {
		panic(fmt.Errorf("failed to format constants: %w", err))
	}
	return r
}

// FormatStatics returns human readable string representations of compiled static values.
func (b *Bytecode) FormatStatics() (output []string, err error) {
	for i, v := range b.Static.Primitives {
		output = append(output, fmt.Sprintf("[% 3d] %s (%s|%v)", i, v.Value().String(), v.Value().TypeName(), v))
	}

	for i, v := range b.Static.Decimals {
		output = append(output, fmt.Sprintf("[% 3d] %s (decimal)", i, v.String()))
	}

	for i, v := range b.Static.Strings {
		output = append(output, fmt.Sprintf("[% 3d] %s (string)", i, v))
	}

	for i, v := range b.Static.Runes {
		output = append(output, fmt.Sprintf("[% 3d] %s (runes)", i, string(v.Elements)))
	}

	for i, v := range b.Static.Bytes {
		output = append(output, fmt.Sprintf("[% 3d] %v (bytes)", i, v.Elements))
	}

	for i, v := range b.Static.Times {
		output = append(output, fmt.Sprintf("[% 3d] %s (time)", i, v.Format("2006-01-02T15:04:05.999999999Z07:00")))
	}

	for i, v := range b.Static.FormatSpecs {
		output = append(output, fmt.Sprintf("[% 3d] %s (format spec)", i, v.Text))
	}

	for i, v := range b.Static.CompiledFunctions {
		output = append(output, fmt.Sprintf("[% 3d] (compiled function)", i))
		t, err := FormatInstructions(v.Instructions, 0)
		if err != nil {
			return nil, err
		}
		for _, l := range t {
			output = append(output, fmt.Sprintf("     %s", l))
		}
		continue
	}

	for i, v := range b.Static.NameLists {
		output = append(output, fmt.Sprintf("[% 3d] %v (name list)", i, v))
	}

	for i, v := range b.Static.Ranges {
		output = append(output, fmt.Sprintf("[% 3d] range(%d, %d, %d) (range)", i, v.Start, v.Stop, v.Step))
	}

	return
}
