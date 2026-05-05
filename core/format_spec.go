package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/fspec"
)

// FormatSpecValue wraps a fully parsed fspec.FormatSpec together with its original textual form. It is an internal
// value kind: it lives only in the constant pool (referenced by OpFormat) and is never visible to user code.
type FormatSpecValue struct {
	Spec fspec.FormatSpec
	Text string // original mini-language text (without the leading ':')
}

// NewFormatSpecValue boxes a parsed FormatSpec for the constant pool.
func NewFormatSpecValue(spec fspec.FormatSpec, text string) Value {
	o := &FormatSpecValue{Spec: spec, Text: text}
	return Value{
		Type:  VT_FORMAT_SPEC,
		Const: true,
		Ptr:   unsafe.Pointer(o),
	}
}

func formatSpecTypeName(v Value) string {
	return "format-spec"
}

func formatSpecTypeString(v Value) string {
	o := (*FormatSpecValue)(v.Ptr)
	return fmt.Sprintf("format_spec(%q)", o.Text)
}

type formatSpecGob struct {
	Text string
}

func formatSpecTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*FormatSpecValue)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(formatSpecGob{Text: o.Text}); err != nil {
		return nil, fmt.Errorf("format_spec: %w", err)
	}
	return buf.Bytes(), nil
}

func formatSpecTypeDecodeBinary(v *Value, data []byte) error {
	var g formatSpecGob
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&g); err != nil {
		return fmt.Errorf("format_spec: %w", err)
	}
	spec, err := fspec.Parse(g.Text)
	if err != nil {
		return fmt.Errorf("format_spec: re-parse %q: %w", g.Text, err)
	}
	o := &FormatSpecValue{Spec: spec, Text: g.Text}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func formatSpecTypeEqual(v Value, r Value) bool {
	if r.Type != VT_FORMAT_SPEC {
		return false
	}
	a := (*FormatSpecValue)(v.Ptr)
	b := (*FormatSpecValue)(r.Ptr)
	return a.Text == b.Text
}
