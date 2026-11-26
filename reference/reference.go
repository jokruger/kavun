// Package reference is the language reference as data: the schema of kavun_reference.json — the file
// cmd/kavun-gen writes at a module's root, describing the types, members, builtin functions and modules that module
// declares — and the helpers that read, write, merge and query it. A docs tool, an editor, tests and an embedder's
// own tools read references through it; it imports the signature package and the standard library only.
//
// The reference is semantic, not presentational: it states what exists, who owns it, what it accepts and what it
// means. Numeric ids and slots (the binding, held by the generated Go tables) are not in it, nor anything that
// follows from names: the twin of an _in_place member (Type.Twin), a module's types (Set.TypesOf).
//
// Every list of a reference is sorted by name (doc blocks keep their order), and Write is the one writer, so a
// reference written twice is the same bytes.
package reference

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/jokruger/kavun/signature"
)

// SchemaVersion is the schema this package reads and writes. It is bumped only by an incompatible change: a field
// added is not one (a reader ignores the fields it does not know), so Read refuses only a greater schema.
const SchemaVersion = 1

// FileName is the name of the reference file at a module's root.
const FileName = "kavun_reference.json"

// KavunOrigin is the origin of Kavun's own reference: the module path of Kavun. A tool that renders Kavun's guide
// pages, or links to them, recognises Kavun's reference by it.
const KavunOrigin = "github.com/jokruger/kavun"

// Reference is one kavun_reference.json: what one Go module declares.
type Reference struct {
	Schema      int       `json:"schema"`
	Origin      string    `json:"origin"`         // the Go module path of the module the file describes
	CoreVersion string    `json:"core_version"`   // Kavun's core version the file was generated against ("1.2")
	Docs        string    `json:"docs,omitempty"` // the URL of the origin's published docs root; "" when none
	Types       []Type    `json:"types"`
	Builtins    *Builtins `json:"builtins,omitempty"` // nil when the module declares no builtin functions
	Modules     []Module  `json:"modules"`
}

// Type is a type with its members. Module names the owning module of a module-owned type ("fin" for
// "fin.year_fraction"); "" otherwise.
type Type struct {
	Name    string     `json:"name"`
	Module  string     `json:"module,omitempty"`
	Summary string     `json:"summary"`
	Doc     Doc        `json:"doc"`
	Pos     Pos        `json:"pos"`
	Members []Callable `json:"members"`
}

// Module is a builtin module with its functions and constants.
type Module struct {
	Name      string     `json:"name"`
	Pure      bool       `json:"pure"` // importable by default; false: needs a grant
	Summary   string     `json:"summary"`
	Doc       Doc        `json:"doc"`
	Pos       Pos        `json:"pos"`
	Functions []Callable `json:"functions"`
	Constants []Constant `json:"constants"`
}

// Builtins is the block of the free builtin functions: an intro (no summary) and the functions.
type Builtins struct {
	Doc       Doc        `json:"doc"`
	Pos       Pos        `json:"pos"`
	Functions []Callable `json:"functions"`
}

// Callable is a member, a builtin function or a module function.
type Callable struct {
	Name      string    `json:"name"`
	Pure      bool      `json:"pure"`
	Signature Signature `json:"signature"`
	Result    string    `json:"result"` // what it answers, free text as declared
	Summary   string    `json:"summary"`
	Doc       Doc       `json:"doc"`
	Pos       Pos       `json:"pos"`
}

// Constant is a module constant. Type is free text as declared.
type Constant struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Summary string `json:"summary"`
	Doc     Doc    `json:"doc"`
	Pos     Pos    `json:"pos"`
}

// Signature is a callable's signature template with what the grammar derives from it: arity (the fewest arguments
// accepted), variadic (any other count accepted) and the readings, in template order.
type Signature struct {
	Template string              `json:"template"`
	Arity    int                 `json:"arity"`
	Variadic bool                `json:"variadic"`
	Readings []signature.Reading `json:"readings"`
}

// SignatureOf answers the Signature of a parsed template.
func SignatureOf(s signature.Shape) Signature {
	return Signature{Template: s.Template, Arity: s.Arity, Variadic: s.Variadic, Readings: s.Readings}
}

// Pos is where a declaration is: its directive's line, in File, relative to the module root and slash-separated.
type Pos struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// String renders a position as file:line.
func (p Pos) String() string {
	return fmt.Sprintf("%s:%d", p.File, p.Line)
}

// The kinds of a doc block.
const (
	BlockText    = "text"    // Markdown, verbatim
	BlockExample = "example" // one fenced `go` block: a Kavun script
)

// Doc is a declaration's body (its summary excluded) as blocks, in order.
type Doc []Block

// Block is one block of a doc body. A text block holds Markdown (a fenced block that is not a `go` fence stays
// inside it). An example block holds the code of one fenced `go` block without its fence lines; Line is the source
// line of its first code line (the following lines map one to one), NoRun marks an example that is not run, Reason
// says why ("" when not given).
type Block struct {
	Kind     string `json:"kind"`
	Markdown string `json:"markdown,omitempty"`
	Code     string `json:"code,omitempty"`
	Line     int    `json:"line,omitempty"`
	NoRun    bool   `json:"norun,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// MarshalJSON writes a block by its kind: a text block always carries its markdown (an empty one as ""), an example
// block omits it when empty; every other field is omitted when empty.
func (b Block) MarshalJSON() ([]byte, error) {
	type omitting Block // the struct tags of Block, without this method
	var v any = omitting(b)
	if b.Kind == BlockText {
		v = struct {
			Kind     string `json:"kind"`
			Markdown string `json:"markdown"`
			Code     string `json:"code,omitempty"`
			Line     int    `json:"line,omitempty"`
			NoRun    bool   `json:"norun,omitempty"`
			Reason   string `json:"reason,omitempty"`
		}(b)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // the encoder of Write keeps `<`, `>` and `&` literal; so must the block's own
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// Read reads a reference. Fields it does not know are ignored; a schema greater than SchemaVersion (or none) is
// refused. Every list of the answer is non-nil, so a reference read back compares equal to the one written.
func Read(r io.Reader) (*Reference, error) {
	var ref Reference
	if err := json.NewDecoder(r).Decode(&ref); err != nil {
		return nil, fmt.Errorf("reading a reference: %w", err)
	}
	switch {
	case ref.Schema > SchemaVersion:
		return nil, fmt.Errorf("reference %s: schema %d is newer than this reader's (%d); update github.com/jokruger/kavun", ref.Origin, ref.Schema, SchemaVersion)
	case ref.Schema < 1:
		return nil, fmt.Errorf("reference %s: no schema version: not a kavun_reference.json", ref.Origin)
	}
	ref.normalize()
	return &ref, nil
}

// ReadFile reads the reference file at path.
func ReadFile(path string) (*Reference, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ref, err := Read(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return ref, nil
}

// Write writes the reference as kavun_reference.json: every list sorted by name (doc blocks and readings keep
// their order), nil lists as [], two-space indent, `<`, `>` and `&` literal, a trailing newline. The reference
// itself is not changed. It is the one writer: the same reference is always the same bytes.
func (r *Reference) Write(w io.Writer) error {
	out := r.clone()
	out.normalize()
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// clone answers a copy deep enough that normalize does not touch r: every list the writer sorts or fills.
func (r *Reference) clone() *Reference {
	out := *r
	out.Types = slices.Clone(r.Types)
	for i := range out.Types {
		out.Types[i].Members = slices.Clone(out.Types[i].Members)
	}
	if r.Builtins != nil {
		b := *r.Builtins
		b.Functions = slices.Clone(b.Functions)
		out.Builtins = &b
	}
	out.Modules = slices.Clone(r.Modules)
	for i := range out.Modules {
		out.Modules[i].Functions = slices.Clone(out.Modules[i].Functions)
		out.Modules[i].Constants = slices.Clone(out.Modules[i].Constants)
	}
	return &out
}

// normalize puts the reference in its one form: every list sorted by name, no nil list. It changes the slices it
// sorts in place; Write calls it on a clone.
func (r *Reference) normalize() {
	r.Types = sortedByName(r.Types, func(t Type) string { return t.Name })
	for i := range r.Types {
		t := &r.Types[i]
		t.Doc = nonNil(t.Doc)
		t.Members = normalizeCallables(t.Members)
	}
	if r.Builtins != nil {
		r.Builtins.Doc = nonNil(r.Builtins.Doc)
		r.Builtins.Functions = normalizeCallables(r.Builtins.Functions)
	}
	r.Modules = sortedByName(r.Modules, func(m Module) string { return m.Name })
	for i := range r.Modules {
		m := &r.Modules[i]
		m.Doc = nonNil(m.Doc)
		m.Functions = normalizeCallables(m.Functions)
		m.Constants = sortedByName(m.Constants, func(c Constant) string { return c.Name })
		for j := range m.Constants {
			m.Constants[j].Doc = nonNil(m.Constants[j].Doc)
		}
	}
}

func normalizeCallables(cs []Callable) []Callable {
	cs = sortedByName(cs, func(c Callable) string { return c.Name })
	for i := range cs {
		c := &cs[i]
		c.Doc = nonNil(c.Doc)
		c.Signature.Readings = slices.Clone(nonNil(c.Signature.Readings)) // a clone: Write must not touch the caller's
		for j := range c.Signature.Readings {
			c.Signature.Readings[j].Params = nonNil(c.Signature.Readings[j].Params)
		}
	}
	return cs
}

func sortedByName[T any](s []T, name func(T) string) []T {
	s = nonNil(s)
	slices.SortStableFunc(s, func(a, b T) int { return strings.Compare(name(a), name(b)) })
	return s
}

func nonNil[S ~[]E, E any](s S) S {
	if s == nil {
		return S{}
	}
	return s
}
