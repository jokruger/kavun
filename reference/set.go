package reference

import (
	"fmt"
	"iter"
	"slices"
	"strings"
)

// Set is several references merged: one namespace of types, modules and builtin functions, each entry keeping the
// origin of the reference that declares it. It reads the references it was made from; they must not change
// afterwards.
type Set struct {
	refs     []*Reference
	types    map[string]owned[*Type]
	modules  map[string]owned[*Module]
	builtins map[string]owned[*Callable]
}

// owned is an entry of a Set with the reference that declares it.
type owned[T any] struct {
	v   T
	ref *Reference
}

// Merge merges references into one Set. A type, module or builtin function declared by two references is refused,
// naming both origins, and so are two references of the same origin; references generated against different core
// versions are accepted (each keeps its CoreVersion).
func Merge(refs ...*Reference) (*Set, error) {
	s := &Set{
		refs:     slices.Clone(refs),
		types:    map[string]owned[*Type]{},
		modules:  map[string]owned[*Module]{},
		builtins: map[string]owned[*Callable]{},
	}
	origins := map[string]bool{}
	var problems []string
	for _, ref := range refs {
		if origins[ref.Origin] {
			problems = append(problems, fmt.Sprintf("origin %s is given twice", ref.Origin))
			continue
		}
		origins[ref.Origin] = true
		for i := range ref.Types {
			t := &ref.Types[i]
			if prev, ok := s.types[t.Name]; ok {
				problems = append(problems, fmt.Sprintf("type %s is declared by %s and by %s", t.Name, prev.ref.Origin, ref.Origin))
				continue
			}
			s.types[t.Name] = owned[*Type]{t, ref}
		}
		for i := range ref.Modules {
			m := &ref.Modules[i]
			if prev, ok := s.modules[m.Name]; ok {
				problems = append(problems, fmt.Sprintf("module %s is declared by %s and by %s", m.Name, prev.ref.Origin, ref.Origin))
				continue
			}
			s.modules[m.Name] = owned[*Module]{m, ref}
		}
		if ref.Builtins != nil {
			for i := range ref.Builtins.Functions {
				f := &ref.Builtins.Functions[i]
				if prev, ok := s.builtins[f.Name]; ok {
					problems = append(problems, fmt.Sprintf("builtin %s is declared by %s and by %s", f.Name, prev.ref.Origin, ref.Origin))
					continue
				}
				s.builtins[f.Name] = owned[*Callable]{f, ref}
			}
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("merging references: %s", strings.Join(problems, "; "))
	}
	return s, nil
}

// References answers the merged references, in the order given to Merge.
func (s *Set) References() []*Reference {
	return slices.Clone(s.refs)
}

// Type answers the type of that name.
func (s *Set) Type(name string) (*Type, bool) {
	e, ok := s.types[name]
	return e.v, ok
}

// Module answers the module of that name.
func (s *Set) Module(name string) (*Module, bool) {
	e, ok := s.modules[name]
	return e.v, ok
}

// Builtin answers the builtin function of that name.
func (s *Set) Builtin(name string) (*Callable, bool) {
	e, ok := s.builtins[name]
	return e.v, ok
}

// Member answers the member name of type typ.
func (s *Set) Member(typ, name string) (*Callable, bool) {
	if t, ok := s.Type(typ); ok {
		return t.Member(name)
	}
	return nil, false
}

// Function answers the function name of module module.
func (s *Set) Function(module, name string) (*Callable, bool) {
	if m, ok := s.Module(module); ok {
		return m.Function(name)
	}
	return nil, false
}

// Constant answers the constant name of module module.
func (s *Set) Constant(module, name string) (*Constant, bool) {
	if m, ok := s.Module(module); ok {
		return m.Constant(name)
	}
	return nil, false
}

// OwnerKind is what holds a callable: a type, a module or the builtins block.
type OwnerKind string

// The owner kinds.
const (
	OwnerType     OwnerKind = "type"
	OwnerModule   OwnerKind = "module"
	OwnerBuiltins OwnerKind = "builtins"
	// OwnerDoc is a hand-written guide page beside the reference: never a callable's owner, only a link's target.
	OwnerDoc OwnerKind = "doc"
)

// Entry is one callable of a Set with its owner and origin. Owner is the type's or module's name; "" for a builtin
// function.
type Entry struct {
	Callable *Callable
	Kind     OwnerKind
	Owner    string
	Origin   string
}

// Callables answers every callable of the Set: the members of every type (types by name), then the builtin
// functions, then the functions of every module (modules by name); each list by name.
func (s *Set) Callables() iter.Seq[Entry] {
	return func(yield func(Entry) bool) {
		for _, name := range sortedKeys(s.types) {
			e := s.types[name]
			for i := range e.v.Members {
				if !yield(Entry{&e.v.Members[i], OwnerType, name, e.ref.Origin}) {
					return
				}
			}
		}
		for _, name := range sortedKeys(s.builtins) {
			e := s.builtins[name]
			if !yield(Entry{e.v, OwnerBuiltins, "", e.ref.Origin}) {
				return
			}
		}
		for _, name := range sortedKeys(s.modules) {
			e := s.modules[name]
			for i := range e.v.Functions {
				if !yield(Entry{&e.v.Functions[i], OwnerModule, name, e.ref.Origin}) {
					return
				}
			}
		}
	}
}

// TypesOf answers the types module owns (module-owned types: Type.Module), by name.
func (s *Set) TypesOf(module string) []*Type {
	var out []*Type
	for _, name := range sortedKeys(s.types) {
		if t := s.types[name].v; t.Module == module && module != "" {
			out = append(out, t)
		}
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// Member answers the type's member of that name.
func (t *Type) Member(name string) (*Callable, bool) {
	return find(t.Members, name)
}

// Function answers the module's function of that name.
func (m *Module) Function(name string) (*Callable, bool) {
	return find(m.Functions, name)
}

// Constant answers the module's constant of that name.
func (m *Module) Constant(name string) (*Constant, bool) {
	i := slices.IndexFunc(m.Constants, func(c Constant) bool { return c.Name == name })
	if i < 0 {
		return nil, false
	}
	return &m.Constants[i], true
}

// Function answers the builtin function of that name.
func (b *Builtins) Function(name string) (*Callable, bool) {
	return find(b.Functions, name)
}

func find(cs []Callable, name string) (*Callable, bool) {
	i := slices.IndexFunc(cs, func(c Callable) bool { return c.Name == name })
	if i < 0 {
		return nil, false
	}
	return &cs[i], true
}

// InPlaceSuffix names the mutating twin of a member: x_in_place mutates its receiver where x answers a new value.
const InPlaceSuffix = "_in_place"

// Twin answers the other member of an _in_place pair: x_in_place for x, x for x_in_place; false when the type does
// not declare both.
func (t *Type) Twin(name string) (*Callable, bool) {
	if _, ok := t.Member(name); !ok {
		return nil, false
	}
	if base, ok := strings.CutSuffix(name, InPlaceSuffix); ok {
		return t.Member(base)
	}
	return t.Member(name + InPlaceSuffix)
}
