package fspec

import (
	"testing"

	"github.com/jokruger/kavun/fspec"
)

func TestParseTemplate_Literals(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain", "hello, world", "hello, world"},
		{"escaped braces", "a {{ b }} c", "a { b } c"},
		{"only escaped braces", "{{}}", "{}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tmpl, err := fspec.ParseTemplate(c.in)
			if err != nil {
				t.Fatalf("ParseTemplate(%q) error: %v", c.in, err)
			}
			if tmpl.Mode != fspec.TemplateModeUnset {
				t.Fatalf("expected mode unset, got %v", tmpl.Mode)
			}
			if c.want == "" {
				if len(tmpl.Segments) != 0 {
					t.Fatalf("expected no segments, got %+v", tmpl.Segments)
				}
				return
			}
			if len(tmpl.Segments) != 1 || tmpl.Segments[0].Kind != fspec.TemplateLiteral || tmpl.Segments[0].Literal != c.want {
				t.Fatalf("unexpected segments: %+v", tmpl.Segments)
			}
		})
	}
}

func TestParseTemplate_NamedAndIndexed(t *testing.T) {
	tmpl, err := fspec.ParseTemplate("hello {x} from {y}!")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Mode != fspec.TemplateModeNamed {
		t.Fatalf("expected Named mode, got %v", tmpl.Mode)
	}
	if len(tmpl.Segments) != 5 {
		t.Fatalf("expected 5 segments, got %d", len(tmpl.Segments))
	}
	if tmpl.Segments[1].Name != "x" || tmpl.Segments[3].Name != "y" {
		t.Fatalf("unexpected names: %+v", tmpl.Segments)
	}

	tmpl, err = fspec.ParseTemplate("hello {0} from {1}!")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Mode != fspec.TemplateModeIndexed {
		t.Fatalf("expected Indexed mode, got %v", tmpl.Mode)
	}
	if tmpl.Segments[1].Index != 0 || tmpl.Segments[3].Index != 1 {
		t.Fatalf("unexpected indices: %+v", tmpl.Segments)
	}
}

func TestParseTemplate_Spec(t *testing.T) {
	t.Run("literal spec", func(t *testing.T) {
		tmpl, err := fspec.ParseTemplate("v={x:.2f}")
		if err != nil {
			t.Fatal(err)
		}
		seg := tmpl.Segments[1]
		if !seg.HasSpec || seg.SpecIsRef {
			t.Fatalf("expected literal spec, got %+v", seg)
		}
		if !seg.Spec.HasPrec || seg.Spec.Precision != 2 || seg.Spec.Verb != 'f' {
			t.Fatalf("unexpected parsed spec: %+v", seg.Spec)
		}
	})
	t.Run("ref spec named", func(t *testing.T) {
		tmpl, err := fspec.ParseTemplate("v={x:{fmt}}")
		if err != nil {
			t.Fatal(err)
		}
		seg := tmpl.Segments[1]
		if !seg.HasSpec || !seg.SpecIsRef || seg.SpecRefName != "fmt" {
			t.Fatalf("unexpected: %+v", seg)
		}
	})
	t.Run("ref spec indexed", func(t *testing.T) {
		tmpl, err := fspec.ParseTemplate("v={0:{1}}")
		if err != nil {
			t.Fatal(err)
		}
		seg := tmpl.Segments[1]
		if !seg.HasSpec || !seg.SpecIsRef || seg.SpecRefIndex != 1 {
			t.Fatalf("unexpected: %+v", seg)
		}
	})
	t.Run("empty spec is no-op", func(t *testing.T) {
		tmpl, err := fspec.ParseTemplate("v={x:}")
		if err != nil {
			t.Fatal(err)
		}
		seg := tmpl.Segments[1]
		if !seg.HasSpec || seg.SpecIsRef {
			t.Fatalf("unexpected: %+v", seg)
		}
	})
}

func TestParseTemplate_Errors(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"bare close", "abc }"},
		{"unterminated", "abc {x"},
		{"empty placeholder", "x={}"},
		{"mixed modes", "{0} and {x}"},
		{"mixed modes reverse", "{x} and {0}"},
		{"expression inside", "{x+1}"},
		{"invalid name", "{1bad}"},
		{"nested ref with literal", "{x:>{w}}"},
		{"two refs", "{x:{a}{b}}"},
		{"ref empty", "{x:{}}"},
		{"unterminated ref", "{x:{abc"},
		{"bad fspec", "{x:zzz}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := fspec.ParseTemplate(c.in); err == nil {
				t.Fatalf("expected error for %q", c.in)
			}
		})
	}
}

func TestParseTemplate_Escapes(t *testing.T) {
	tmpl, err := fspec.ParseTemplate("set = {{ {x} }}")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Mode != fspec.TemplateModeNamed {
		t.Fatalf("mode: %v", tmpl.Mode)
	}
	if len(tmpl.Segments) != 3 {
		t.Fatalf("segs: %+v", tmpl.Segments)
	}
	if tmpl.Segments[0].Literal != "set = { " {
		t.Fatalf("first lit: %q", tmpl.Segments[0].Literal)
	}
	if tmpl.Segments[1].Name != "x" {
		t.Fatalf("placeholder: %+v", tmpl.Segments[1])
	}
	if tmpl.Segments[2].Literal != " }" {
		t.Fatalf("last lit: %q", tmpl.Segments[2].Literal)
	}
}
