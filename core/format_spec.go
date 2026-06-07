package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/fspec"
)

const formatSpecTypeName = "format-spec"

// FormatSpec wraps a fully parsed fspec.FormatSpec together with its original textual form. It is an internal value
// kind: it lives only in the constant pool (referenced by OpFormat) and is never visible to user code.
type FormatSpec struct {
	Spec fspec.FormatSpec
	Text string // original mini-language text (without the leading ':')
}

var TypeFormatSpec = ValueTypeDescr{
	Name:         ConstHook(formatSpecTypeName),
	String:       formatSpecTypeString,
	EncodeBinary: formatSpecTypeEncodeBinary,
	DecodeBinary: formatSpecTypeDecodeBinary,
	Equal:        formatSpecTypeEqual,
}

func formatSpecTypeString(_ *Arena, v Value) string {
	o := (*FormatSpec)(v.Ptr)
	return fmt.Sprintf("format_spec(%q)", o.Text)
}

type formatSpecGob struct {
	Text string
}

func formatSpecTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
	o := (*FormatSpec)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(formatSpecGob{Text: o.Text}); err != nil {
		return nil, fmt.Errorf("format_spec: %w", err)
	}
	return buf.Bytes(), nil
}

func formatSpecTypeDecodeBinary(a *Arena, v *Value, data []byte) error {
	var g formatSpecGob
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&g); err != nil {
		return fmt.Errorf("format_spec: %w", err)
	}
	spec, err := fspec.Parse(g.Text)
	if err != nil {
		return fmt.Errorf("format_spec: re-parse %q: %w", g.Text, err)
	}
	o := &FormatSpec{Spec: spec, Text: g.Text}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func formatSpecTypeEqual(a *Arena, v Value, r Value) bool {
	if r.Type != VT_FORMAT_SPEC {
		return false
	}
	x := (*FormatSpec)(v.Ptr)
	y := (*FormatSpec)(r.Ptr)
	return x.Text == y.Text
}
