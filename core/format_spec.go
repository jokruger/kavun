package core

import (
	"fmt"

	"github.com/jokruger/kavun/fspec"
)

const formatSpecTypeName = "format-spec"

// FormatSpec wraps a fully parsed fspec.FormatSpec together with its original textual form. It is an internal value
// kind: it lives only in the constant pool (referenced by OpFormat) and is never visible to user code.
type FormatSpec struct {
	Spec fspec.FormatSpec
	Text string // original mini-language text (without the leading ':')
}

func (f *FormatSpec) Set(spec fspec.FormatSpec, text string) {
	f.Spec = spec
	f.Text = text
}

func (f FormatSpec) Equal(other FormatSpec) bool {
	return f.Spec.Equal(other.Spec) && f.Text == other.Text
}

var TypeFormatSpec = ValueTypeDescr{
	Name:   ConstHook(formatSpecTypeName),
	String: formatSpecTypeString,
	Equal:  formatSpecTypeEqual,
}

func formatSpecTypeString(a *Arena, v Value) string {
	o := a.ResolveFormatSpecValue(v)
	return fmt.Sprintf("format_spec(%q)", o.Text)
}

type formatSpecGob struct {
	Text string
}

func formatSpecTypeEqual(a *Arena, v Value, r Value) bool {
	if r.Type != VT_FORMAT_SPEC {
		return false
	}
	x := a.ResolveFormatSpecValue(v)
	y := a.ResolveFormatSpecValue(r)
	return x.Text == y.Text
}
