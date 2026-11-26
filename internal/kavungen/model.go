package kavungen

import (
	"go/token"
	"slices"
	"strings"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/internal/kavundocs"
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
	dir     string // the package directory, as given

	// names is the //kavun:names of the package, or nil: its generated file binds the names.
	names *Directive
	// run is the member names of the whole run, the builtin ones aside: a member, fn or const may use them.
	run runNames
	// foreign maps the directory of each other package of the run whose names the package's tables use (the
	// generated file imports it) to the first directive using one.
	foreign map[string]token.Position

	types []*typeModel // sorted by Kavun type name

	builtins     *Directive  // the //kavun:builtins root, or nil
	builtinFuncs []Directive // sorted by slot

	module       *Directive  // the //kavun:module root, or nil
	moduleFuncs  []Directive // sorted by slot
	moduleConsts []Directive // sorted by name
	hasExtras    bool

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
	return m.builtins == nil && m.module == nil && m.names == nil
}

// buildModel applies the rules that span directives: the type ↔ members file pairing, names and slots. Source
// order carries no meaning: a page lists its entries alphabetically.
//
// names holds the member names the run declares (collectNames); nil reads the package's own //kavun:names only.
func buildModel(ctx *Context, p *scannedPackage, names runNames) (*packageModel, errorList) {
	var errs errorList
	if names == nil {
		names = collectNames(ctx, []*scannedPackage{p}, &errs)
	}
	m := &packageModel{pkgName: p.pkgName, dir: p.dir, hasExtras: p.hasExtras, run: names, foreign: map[string]token.Position{}}

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
				if x, ok := strings.CutPrefix(d.ModuleID, "value."); ok && !ctx.valueConsts[x] {
					errs.add(d.Pos, "module %q: id=%s is not a constant of core/value (core/value/value.go)", d.Name, d.ModuleID)
					p.refusedModule = true
					continue
				}
				if !strings.HasPrefix(d.ModuleID, "value.") && !p.consts[d.ModuleID] && !p.vars[d.ModuleID] {
					errs.add(d.Pos, "module %q: id=%s is not a constant or variable of the package (the module's type id: one of the package, or value.<Ident>)", d.Name, d.ModuleID)
					p.refusedModule = true
					continue
				}
				m.module = d
			case KindNames:
				if m.names != nil {
					errs.add(d.Pos, "a second //kavun:names (the first is at %s): one per package", where(m.names.Pos))
					continue
				}
				m.names = d
			}
		}
	}

	// a dotted name is a module-owned type's, so it lives in a module's package
	for _, t := range m.types {
		prefix, _, dotted := strings.Cut(t.decl.Name, ".")
		switch {
		case dotted && m.module == nil && !p.refusedModule:
			errs.add(t.decl.Pos, "type %q: a dotted name belongs to a module-owned type, declared in its module's package (one with a //kavun:module)", t.decl.Name)
		case dotted && prefix != m.module.Name:
			errs.add(t.decl.Pos, "type %q: a module-owned type's prefix is its module's name, %q here", t.decl.Name, m.module.Name)
		}
	}

	// Pass 2: the entries, file by file.
	builtinSlots, builtinNames := map[int]Directive{}, map[string]Directive{}
	fnSlots, fnNames := map[int]Directive{}, map[string]Directive{}
	constNames := map[string]Directive{}
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
				if !m.checkMemberName(ctx, d, &errs) {
					failed[f.name] = true // reported (or its name's entry is): not a members file declaring nothing
					continue
				}
				if d.Pure && strings.HasSuffix(d.Name, "_in_place") {
					errs.add(d.Pos, "//kavun:%s %s: an _in_place member mutates its receiver and must be impure", d.Kind, d.Name)
					continue
				}
				if member.IsConversion(member.Lookup(d.Name)) && (!d.Pure || d.Template != conversionTemplate) {
					errs.add(d.Pos, "//kavun:%s %s: a conversion member is the conversion %s(x) calls: it must be pure with the template %s", d.Kind, d.Name, d.Name, conversionTemplate)
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
					if !p.refusedModule {
						errs.add(d.Pos, "//kavun:fn %s: the package has no //kavun:module", d.Name)
					}
					continue
				}
				if !m.checkMemberName(ctx, d, &errs) {
					continue
				}
				addSlotted(ctx, d, fnSlots, fnNames, "module "+m.module.Name, &errs)

			case KindConst:
				if m.module == nil {
					if !p.refusedModule {
						errs.add(d.Pos, "//kavun:const %s: the package has no //kavun:module", d.Name)
					}
					continue
				}
				if !m.checkMemberName(ctx, d, &errs) {
					continue
				}
				if prev, ok := constNames[d.Name]; ok {
					errs.add(d.Pos, "constant %q is declared twice in module %s (first at %s)", d.Name, m.module.Name, where(prev.Pos))
					continue
				}
				constNames[d.Name] = d
			}
		}
	}
	// a constant and a function of one module share the module's namespace (its members and properties)
	for _, d := range sortedByName(constNames) {
		if fn, ok := fnNames[d.Name]; ok {
			errs.add(d.Pos, "constant %q has the name of a function of module %s (at %s)", d.Name, m.module.Name, where(fn.Pos))
			delete(constNames, d.Name)
		}
	}

	// Pass 3: per type with a members file, a non-empty table; a type without one has a page and no table.
	for _, t := range m.types {
		t.hasMembers = t.membersFile || len(t.members) > 0
		if t.membersFile && len(t.members) == 0 && !failed[t.stem+"_members.go"] {
			errs.add(t.decl.Pos, "type %q: members file %s_members.go declares no //kavun:member", t.decl.Name, t.stem)
		}
		if t.hasMembers {
			checkOwnHeading(t.decl, kavundocs.MembersHeading, &errs)
		}
	}
	if m.builtins != nil {
		checkOwnHeading(*m.builtins, kavundocs.FunctionsHeading, &errs)
	}
	if m.module != nil {
		checkOwnHeading(*m.module, kavundocs.FunctionsHeading, &errs)
		if len(constNames) > 0 {
			checkOwnHeading(*m.module, kavundocs.ConstantsHeading, &errs)
		}
	}

	slices.SortFunc(m.types, func(a, b *typeModel) int { return strings.Compare(a.decl.Name, b.decl.Name) })
	m.builtinFuncs = sortedBySlot(builtinSlots)
	m.moduleFuncs = sortedBySlot(fnSlots)
	m.moduleConsts = sortedByName(constNames)
	checkLinks(m, &errs)
	return m, errs
}

// checkMemberName resolves a member, module-function or constant name: a builtin member name (the generated code
// names its members constant), or one declared by a //kavun:names of the run (the generated code names its key,
// qualified from another package, which must then export it).
func (m *packageModel) checkMemberName(ctx *Context, d Directive, errs *errorList) bool {
	if ctx.memberIdents[d.Name] != "" {
		if member.Lookup(d.Name) == member.Unknown {
			errs.add(d.Pos, "%q is not a member name member.Lookup knows (add it to core/member first)", d.Name)
			return false
		}
		return true
	}
	def, ok := m.run[d.Name]
	if ok && def.refused {
		return false // its //kavun:names entry is reported
	}
	if !ok {
		errs.add(d.Pos, "%q is neither a builtin member name (core/member/builtin.go) nor declared by a //kavun:names of the run", d.Name)
		return false
	}
	if def.dir != m.dir {
		if !token.IsExported(def.Key) {
			errs.add(d.Pos, "%q is declared by the //kavun:names of package %s (%s) under the unexported %s: export the key to use it from another package", d.Name, def.pkgName, where(def.Pos), def.Key)
			return false
		}
		if _, ok := m.foreign[def.dir]; !ok {
			m.foreign[def.dir] = d.Pos
		}
	}
	return true
}

// runNames maps each member name a run declares by //kavun:names to its declaration.
type runNames map[string]nameDef

// nameDef is one //kavun:names entry with its package.
type nameDef struct {
	NameDecl
	dir     string // the package directory, as given
	pkgName string
	// refused marks an entry refused by collectNames (reported there): a member using the name is not reported again.
	refused bool
}

// collectNames collects the //kavun:names of every package of a run, before any table is checked: a name declared
// twice (in the run, or as a builtin member name), a key used twice, and a key that is not a constant of its
// package are refused.
func collectNames(ctx *Context, pkgs []*scannedPackage, errs *errorList) runNames {
	out := runNames{}
	for _, p := range pkgs {
		keys := map[string]NameDecl{}
		for _, f := range p.files {
			for _, d := range f.directives {
				if d.Kind != KindNames {
					continue
				}
				for _, n := range d.Names {
					switch prev, dup := out[n.Name]; {
					case ctx.memberIdents[n.Name] != "" || member.Lookup(n.Name) != member.Unknown:
						errs.add(n.Pos, "//kavun:names: %q is a builtin member name: use it as it is, without declaring it", n.Name)
					case dup:
						errs.add(n.Pos, "//kavun:names: %q is declared twice (first at %s)", n.Name, where(prev.Pos))
					case !p.consts[n.Key]:
						errs.add(n.Pos, "//kavun:names: the key %s of %q is not a constant of the package (a member.ID constant: the tables are indexed by it)", n.Key, n.Name)
						out[n.Name] = nameDef{NameDecl: n, dir: p.dir, pkgName: p.pkgName, refused: true}
					case keys[n.Key].Key != "":
						errs.add(n.Pos, "//kavun:names: the key %s names %q and %q (first at %s)", n.Key, keys[n.Key].Name, n.Name, where(keys[n.Key].Pos))
						out[n.Name] = nameDef{NameDecl: n, dir: p.dir, pkgName: p.pkgName, refused: true}
					default:
						keys[n.Key] = n
						out[n.Name] = nameDef{NameDecl: n, dir: p.dir, pkgName: p.pkgName}
					}
				}
			}
		}
	}
	return out
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

func sortedByName(names map[string]Directive) []Directive {
	out := make([]Directive, 0, len(names))
	for _, d := range names {
		out = append(out, d)
	}
	slices.SortFunc(out, func(a, b Directive) int { return strings.Compare(a.Name, b.Name) })
	return out
}

func sortedBySlot(slots map[int]Directive) []Directive {
	out := make([]Directive, 0, len(slots))
	for _, d := range slots {
		out = append(out, d)
	}
	slices.SortFunc(out, func(a, b Directive) int { return a.Slot - b.Slot })
	return out
}

// checkOwnHeading refuses a root block's `##` heading equal to the one cmd/kavun-docs adds to that page (the entry
// list's "Members", "Functions" or "Constants"): the page would carry the heading twice, and the second's anchor would shift.
func checkOwnHeading(root Directive, heading string, errs *errorList) {
	for _, s := range root.Sections {
		if s.Level == 2 && strings.EqualFold(s.Title, heading) {
			errs.add(s.Pos, "a `## %s` heading in the //kavun:%s block: kavun-docs adds that section to the page; name yours differently", s.Title, root.Kind)
		}
	}
}
