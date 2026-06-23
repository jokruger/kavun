package vm

import (
	"encoding/gob"
	"fmt"
	"io"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser"
)

// Bytecode is a compiled instructions and constants.
type Bytecode struct {
	FileSet      *parser.SourceFileSet
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

	// encode bytecode
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

	return
}
