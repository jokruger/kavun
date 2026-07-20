package core

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/fspec"
)

const formatSpecTypeName = "format-spec"

// FormatSpec wraps a fully parsed fspec.FormatSpec together with its original textual form. It is an internal value
// kind: it lives only in the constant pool (referenced by OpFormat) and is never visible to user code.
type FormatSpec struct {
	Spec fspec.FormatSpec
	Text string // original mini-language text (without the leading ':')
}

func NewStaticFormatSpecValue(fs *FormatSpec) Value {
	return Value{Type: value.FormatSpec, Immutable: true, Ptr: unsafe.Pointer(fs)}
}

func (f *FormatSpec) Set(spec fspec.FormatSpec, text string) {
	f.Spec = spec
	f.Text = text
}

func (f FormatSpec) Equal(other FormatSpec) bool {
	return f.Spec.Equal(other.Spec) && f.Text == other.Text
}

var TypeFormatSpec = ValueTypeDescr{
	Name:   ConstHook(formatSpecTypeName), // PURE by contract
	String: formatSpecTypeString,          // PURE by contract
	Equal:  formatSpecTypeEqual,           // PURE by contract
}

func formatSpecTypeString(v Value) string {
	o := (*FormatSpec)(v.Ptr)
	return fmt.Sprintf("format_spec(%q)", o.Text)
}

type formatSpecGob struct {
	Text string
}

func formatSpecTypeEqual(v Value, r Value) bool {
	if r.Type != value.FormatSpec {
		return false
	}
	x := (*FormatSpec)(v.Ptr)
	y := (*FormatSpec)(r.Ptr)
	return x.Text == y.Text
}
