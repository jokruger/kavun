// Package kavungen is the logic of cmd/kavun-gen: it reads `//kavun:` directives from Go comments and writes the
// tables that register Kavun's native functions, one kavun_gen.go per package, and the module's reference file
// kavun_reference.json (reference.go) from their doc blocks. It is the only reader of the sources and the validator
// of the declarations; the Markdown reference pages are rendered from kavun_reference.json by cmd/kavun-docs
// (internal/kavundocs), which describes the page layout.
//
// A directive is a comment line starting with `//kavun:` (no space after the slashes); a comment group holds at most
// one. It may sit anywhere in its group, but the last line is the canonical place (gofmt moves it there in a doc
// comment). The other lines of the group, `// ` stripped, are its doc block: its first paragraph (up to the first
// blank comment line, its lines joined with single spaces) is the summary — one sentence ending in a period, a
// second sentence (". " and an upper-case letter) is refused — and the rest is a markdown body.
//
//	//kavun:type <name>                                                 free-standing, in <stem>.go
//	//kavun:member <name> pure|impure <template> -> <result>            on the member body, in <stem>[_members].go
//	//kavun:builtins                                                    free-standing: opens the builtins
//	//kavun:builtin <name> slot=<n> pure|impure <template> -> <result>  on the builtin body
//	//kavun:module <name> id=<go ident> pure|impure                     free-standing: the package's module
//	//kavun:fn <name> slot=<n> pure|impure <template> -> <result>       on the module function body
//	//kavun:const <name> -> <type>                                      on the package-level var holding the value
//	//kavun:names                                                       on a package-level map[member.ID]string var
//
// A type's members live in <stem>_members.go, or in the root file <stem>.go itself (a module-owned type keeps its
// descriptor and bodies in one file). A type name is a name, or module.name for a module-owned type ("fs.file",
// allowed only in a package that declares a //kavun:module); its table is named after the root file's stem, as every
// table is.
//
// A template is a signature template (signature.Parse). The result after ` -> ` is free text to the end of the line
// — `-> array`, `-> [dir, file]`, `-> undefined`, `-> value` where the type depends on the input — and is documentation
// only: it reaches the reference, never the table. Source order carries no meaning: entries may sit anywhere in their
// files, and the reference lists them alphabetically. The generated kavun_gen.go holds, per package: `var <stem>Methods =
// []Method{…}` for each type with a members file (its descriptor says `Methods: <stem>Methods,`; a type whose
// <stem>_members.go does not exist — record, module — gets no table, and its descriptor is untouched; a members file
// that exists must declare at least one member), an init that fills the builtin function table, and an init that
// declares and registers the module.
//
// A module's id= is its type id as the Go expression the init passes to core.NewModule: an identifier the package
// declares (a const or var: an embedder's id) or value.<Ident> (a core/value constant: Kavun's modules,
// id=value.ModuleMath, checked against the constants of core/value/value.go in the Kavun root). No naming convention
// is read.
//
// A member, fn or const name is a builtin member name (core/member/builtin.go, members.<Ident> in the tables) or one
// declared by a //kavun:names of the run: the directive sits in the doc comment of a package-level var holding a
// `map[member.ID]string{Key: "name", …}` literal, every key a constant of the package (a table is a slice literal
// indexed by it) and every value a string literal with a member name's syntax. The names of every package of the run
// are collected first (collectNames), before any table is checked; a builtin member name, a name declared twice and a
// key used twice are refused. The package's kavun_gen.go binds each name with member.Define(key, "name") in the
// initializer of a blank package-level var — before any init function of the package runs (Go initializes every
// package-level variable first), so a type registered by an init function in a file sorting before kavun_gen.go
// finds its members' names bound. The tables name the key bare in its own package and qualified by the package name
// from another package of the run, which then imports it (its import path by `go list`); the key must be exported
// for that, and packages using each other's names (directly or through others) are refused, since the generated
// files would import each other (checkNameCycles). A type registered by a package-level variable initializer is not
// ordered after the binding: register types in init functions.
//
// A module constant is a package-level var holding its core.Value, in a module package, the directive in the var's
// doc comment (no blank line between them): one var, one name — a Go const cannot hold a core.Value and is refused.
// Its name is a member name (below), unique in the module (a function of the same name is refused); `-> <type>` is
// free text, documentation only; its doc block is an entry's, summary required. The module's init calls
// `m.Const(<id>, <var>)` for each (members.<Name>, or a //kavun:names key), by name, after the functions. A module package that needs more (owned
// types) declares `func moduleExtras(m *core.ModuleBuilder)`; the generated init calls it last, before Register. A
// constant added there by hand works but is not in the reference.
//
// A root block (type, module, builtins) may have its own `##` sections, but not one named like the entry list
// kavun-docs adds to its page (`## Members`, `## Functions`, `## Constants` on a module with constants), nor any heading whose slug (reference.Slug) is the name
// of an entry of the same page: the entry's anchor would be taken twice.
//
// Links name what they link to, never a page (links.go): a `kavun:` target (reference.ParseLink —
// kavun:type/<type>[/<member>], kavun:module/<module>[/<entry>], kavun:builtins, kavun:builtin/<name>,
// kavun:doc/<page>, a #heading fragment on a type, module, builtins or doc link), the shorthand `#x` (in the block of
// a type, module or builtins, or of one of its entries: its entry x (a member, function or constant), else the heading of its root block with the
// slug x), or an absolute URL (any scheme), left untouched. A relative target (`string.md`, `../../x.md`) is refused.
// The summary and the body are read through reference.FindLinks, the docs tool's own link reader. The reference holds
// every link in its absolute form: the shorthand is rewritten, and every kavun: target but kavun:doc/ must resolve
// against the run's declarations and the references it loads (reference.Set.Resolve over reference.Merge: a name
// declared twice is refused), each failure reported at its line — Kavun's own reference when the main module is not
// Kavun (<kavun root>/kavun_reference.json, the Kavun module the run builds against) and every -with file; kavun:doc/
// names a page of the rendered site, which kavun-docs checks. Every run builds the reference and resolves the targets; only a run
// over the whole module writes it. A run over part of the module resolves a target only when it declares the
// target's owner (the type, the module, or a builtins block for kavun:builtins and kavun:builtin/) or a loaded
// reference does, whose entry or heading must then exist; a target owned by a package outside the run passes, left to
// the whole-module run.
//
// A body is Markdown, kept verbatim; because gofmt reformats the doc comment of a declaration, no line may be
// indented outside a fenced block, and a fenced block is either unindented or written in gofmt's code-block form (a
// tab before every line), which the reference holds without the tab. Fences are backticks only (`~~~` is refused),
// closed as in CommonMark by a line of at least as many backticks. A `#` or `##` heading appears only in a type,
// module or builtins block; an entry's body uses `####` and deeper to nest under its `###` heading (`###` is
// accepted, at the entry's level). A list must be followed by a blank line: an unindented line right after a list
// item continues that item (CommonMark lazy continuation).
//
// A fenced ```go block of a doc block is an example: a standalone Kavun script that TestDocExamples (kavun_test.go)
// runs and checks through its `// =>` and `// Error:` comments (examples.go, ClassifyLine). A block that is not a runnable
// script is fenced ```go norun [reason]; the reference marks it norun, and a page shows it as a plain ```go fence.
//
// The reference file (package reference) is the same declarations as data, one per module at its root: every type
// (with its owning module when module-owned), member, builtin function and module, each with its purity, signature
// (the template with its arity, variadic and readings), result, summary, doc body and position, and every module
// constant with its type, summary, doc body and position (the directive's line,
// the file relative to the module root). A doc body is a list of blocks: text (Markdown as the page shows it, a
// fenced block of another language included) and examples (each fenced ```go block: its code, the source line of its
// first code line, its norun marker and reason); joined back (reference.Doc.Markdown) they are the body. The
// reference holds no id, slot or type id: those are the binding, in the tables. Its header names the module (origin),
// Kavun's core version (vm/bytecode.go of the Kavun module in use) and, with -docs-url, the module's published docs
// root; it depends on the sources and the flags only. It is generated only by a run that covers the whole main
// module (see Run).
//
// The generator reads what the build compiles: a file excluded by a build constraint (//go:build, a _GOOS/_GOARCH
// suffix) is skipped, so a members file excluded on a build makes its type members-less on that build (in the
// reference, no table).
//
// Packages under <kavun root>/tmp/, the repository's gitignored scratch folder, are never read.
package kavungen
