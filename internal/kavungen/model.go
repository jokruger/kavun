package kavungen

import (
	"slices"
	"strings"

	"github.com/jokruger/kavun/core/member"
)

// typeModel is one `//kavun:type` with the members of its members file.
type typeModel struct {
	decl    Directive
	stem    string      // the root file's name without ".go": "array", "int_range"
	members []Directive // member directives, in source order
	// membersFile reports that <stem>_members.go exists; it must then declare at least one member.
	membersFile bool
	// hasMembers reports that the type has a table: a members file, or member directives in the root file itself
	// (a module-owned type keeps descriptor and bodies in one file). A type without either (record, module) has a
	// page made of its type block alone and no table.
	hasMembers bool
}

// packageModel is everything one package declares, checked across files and ready to emit.
type packageModel struct {
	pkgName string

	types []*typeModel // sorted by Kavun type name

	builtins     *Directive  // the //kavun:builtins root, or nil
	builtinFuncs []Directive // sorted by slot

	module      *Directive  // the //kavun:module root, or nil
	moduleFuncs []Directive // sorted by slot
	hasExtras   bool

	// relDir is the package directory relative to the module root, slash-separated: the Markdown headers name the
	// source files through it. Set by the caller; the model itself never needs it.
	relDir string
}

// empty reports whether the package declares nothing to generate.
func (m *packageModel) empty() bool {
	return len(m.types) == 0 && m.builtins == nil && m.module == nil
}

// emptyGo reports whether the package declares nothing that has Go to generate: only members-less types (their pages
// are all there is), or nothing at all.
func (m *packageModel) emptyGo() bool {
	for _, t := range m.types {
		if t.hasMembers {
			return false
		}
	}
	return m.builtins == nil && m.module == nil
}

// buildModel applies the rules that span directives: the type ↔ members file pairing, names and slots. Source
// order carries no meaning: a page lists its entries alphabetically.
func buildModel(ctx *Context, p *scannedPackage) (*packageModel, errorList) {
	var errs errorList
	m := &packageModel{pkgName: p.pkgName, hasExtras: p.hasExtras}

	// failed names the files where a directive was refused and dropped: the check its absence would trip (a type
	// with no members) stays quiet there, so the report names the real problem only. Generation fails on that
	// problem anyway.
	failed := map[string]bool{}
	for _, f := range p.files {
		failed[f.name] = f.failed
	}

	// Pass 1: the roots (type, builtins, module), so a members file can be paired whatever the file order.
	fileNames := map[string]bool{}
	for _, f := range p.files {
		fileNames[f.name] = true
	}
	typesByStem := map[string]*typeModel{}
	typeNames := map[string]bool{}
	for _, f := range p.files {
		for i := range f.directives {
			d := &f.directives[i]
			switch d.Kind {
			case KindType:
				stem := strings.TrimSuffix(f.name, ".go")
				switch {
				case strings.HasSuffix(stem, "_members"):
					errs.add(d.Pos, "//kavun:type belongs in the type's root file, not in a members file")
				case typesByStem[stem] != nil:
					errs.add(d.Pos, "a second //kavun:type in %s: one type per root file", f.name)
				case typeNames[d.Name]:
					errs.add(d.Pos, "type %q is declared twice", d.Name)
				default:
					t := &typeModel{decl: *d, stem: stem, membersFile: fileNames[stem+"_members.go"]}
					typesByStem[stem], typeNames[d.Name] = t, true
					m.types = append(m.types, t)
				}
			case KindBuiltins:
				if m.builtins != nil {
					errs.add(d.Pos, "a second //kavun:builtins (the first is at %s)", where(m.builtins.Pos))
					continue
				}
				m.builtins = d
			case KindModule:
				if m.module != nil {
					errs.add(d.Pos, "a second //kavun:module (the first is at %s): one module per package", where(m.module.Pos))
					continue
				}
				if !ctx.moduleIDs[moduleIDConst(d.Name)] {
					errs.add(d.Pos, "module %q has no type id constant value.%s in core/value/value.go", d.Name, moduleIDConst(d.Name))
					continue
				}
				m.module = d
			}
		}
	}

	// a dotted name is a module-owned type's, so it lives in a module's package
	for _, t := range m.types {
		prefix, _, dotted := strings.Cut(t.decl.Name, ".")
		switch {
		case dotted && m.module == nil:
			errs.add(t.decl.Pos, "type %q: a dotted name belongs to a module-owned type, declared in its module's package (one with a //kavun:module)", t.decl.Name)
		case dotted && prefix != m.module.Name:
			errs.add(t.decl.Pos, "type %q: a module-owned type's prefix is its module's name, %q here", t.decl.Name, m.module.Name)
		}
	}

	// Pass 2: the entries, file by file.
	builtinSlots, builtinNames := map[int]Directive{}, map[string]Directive{}
	fnSlots, fnNames := map[int]Directive{}, map[string]Directive{}
	for _, f := range p.files {
		stem := strings.TrimSuffix(f.name, ".go")
		// a members file pairs with its root by stem; a root file may hold its own type's members too
		owner := typesByStem[strings.TrimSuffix(stem, "_members")]

		for _, d := range f.directives {
			switch d.Kind {
			case KindMember:
				switch {
				case owner == nil && strings.HasSuffix(stem, "_members"):
					errs.add(d.Pos, "//kavun:%s %s: %s pairs with %s.go, which declares no //kavun:type", d.Kind, d.Name, f.name, strings.TrimSuffix(stem, "_members"))
					continue
				case owner == nil:
					errs.add(d.Pos, "//kavun:%s %s outside a type's files: a type's members live in its root file <stem>.go (with the //kavun:type) or in <stem>_members.go", d.Kind, d.Name)
					continue
				}
				if !checkMemberName(ctx, d, &errs) {
					continue
				}
				if d.Pure && strings.HasSuffix(d.Name, "_in_place") {
					errs.add(d.Pos, "//kavun:%s %s: an _in_place member mutates its receiver and must be impure", d.Kind, d.Name)
					continue
				}
				if j := slices.IndexFunc(owner.members, func(o Directive) bool { return o.Name == d.Name }); j >= 0 {
					errs.add(d.Pos, "member %q is declared twice for type %q (first at %s)", d.Name, owner.decl.Name, where(owner.members[j].Pos))
					continue
				}
				owner.members = append(owner.members, d)

			case KindBuiltin:
				if m.builtins == nil {
					errs.add(d.Pos, "//kavun:builtin %s: the package has no //kavun:builtins", d.Name)
					continue
				}
				addSlotted(ctx, d, builtinSlots, builtinNames, "the builtins", &errs)

			case KindFn:
				if m.module == nil {
					errs.add(d.Pos, "//kavun:fn %s: the package has no //kavun:module", d.Name)
					continue
				}
				if !checkMemberName(ctx, d, &errs) {
					continue
				}
				addSlotted(ctx, d, fnSlots, fnNames, "module "+m.module.Name, &errs)
			}
		}
	}

	// Pass 3: per type with a members file, a non-empty table; a type without one has a page and no table.
	for _, t := range m.types {
		t.hasMembers = t.membersFile || len(t.members) > 0
		if t.membersFile && len(t.members) == 0 && !failed[t.stem+"_members.go"] {
			errs.add(t.decl.Pos, "type %q: members file %s_members.go declares no //kavun:member", t.decl.Name, t.stem)
		}
		if t.hasMembers {
			checkOwnHeading(t.decl, membersHeading, &errs)
		}
	}
	if m.builtins != nil {
		checkOwnHeading(*m.builtins, functionsHeading, &errs)
	}
	if m.module != nil {
		checkOwnHeading(*m.module, functionsHeading, &errs)
	}

	slices.SortFunc(m.types, func(a, b *typeModel) int { return strings.Compare(a.decl.Name, b.decl.Name) })
	m.builtinFuncs = sortedBySlot(builtinSlots)
	m.moduleFuncs = sortedBySlot(fnSlots)
	return m, errs
}

// checkMemberName resolves a member or module-function name: it must be a bound builtin member name, so the
// generated code can name its members constant.
func checkMemberName(ctx *Context, d Directive, errs *errorList) bool {
	if member.Lookup(d.Name) == member.Unknown {
		errs.add(d.Pos, "%q is not a member name member.Lookup knows (add it to core/member first)", d.Name)
		return false
	}
	if ctx.memberIdents[d.Name] == "" {
		errs.add(d.Pos, "%q is not a builtin member name (core/member/builtin.go)", d.Name)
		return false
	}
	return true
}

// addSlotted records a builtin or module function, refusing a slot out of range or a slot or a name used twice in
// the same set.
func addSlotted(ctx *Context, d Directive, slots map[int]Directive, names map[string]Directive, set string, errs *errorList) {
	if d.Slot >= ctx.slotSize {
		errs.add(d.Pos, "slot %d is out of range (0–%d)", d.Slot, ctx.slotSize-1)
		return
	}
	if prev, ok := slots[d.Slot]; ok {
		errs.add(d.Pos, "slot %d is used twice in %s (first by %s at %s)", d.Slot, set, prev.Name, where(prev.Pos))
		return
	}
	if prev, ok := names[d.Name]; ok {
		errs.add(d.Pos, "%q is declared twice in %s (first at %s)", d.Name, set, where(prev.Pos))
		return
	}
	slots[d.Slot], names[d.Name] = d, d
}

func sortedBySlot(slots map[int]Directive) []Directive {
	out := make([]Directive, 0, len(slots))
	for _, d := range slots {
		out = append(out, d)
	}
	slices.SortFunc(out, func(a, b Directive) int { return a.Slot - b.Slot })
	return out
}

// checkOwnHeading refuses a root block's `##` heading equal to the one the generator adds to that page (the entry
// list's "Members" or "Functions"): the page would carry the heading twice, and the second's anchor would shift.
func checkOwnHeading(root Directive, heading string, errs *errorList) {
	for _, s := range root.Sections {
		if strings.EqualFold(s.Title, heading) {
			errs.add(s.Pos, "a `## %s` heading in the //kavun:%s block: the generator adds that section to the page; name yours differently", s.Title, root.Kind)
		}
	}
}
