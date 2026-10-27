// Package kavungen is the logic of cmd/kavun-gen: it reads `//kavun:` directives from Go comments and writes the
// tables that register Kavun's native functions, one kavun_gen.go per package, and the Markdown reference pages
// under docs/reference/ from their doc blocks (markdown.go).
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
//	//kavun:module <name> pure|impure                                   free-standing: the package's module
//	//kavun:fn <name> slot=<n> pure|impure <template> -> <result>       on the module function body
//
// A type's members live in <stem>_members.go, or in the root file <stem>.go itself (a module-owned type keeps its
// descriptor and bodies in one file). A type name is a name, or module.name for a module-owned type ("fs.file",
// allowed only in a package that declares a //kavun:module); its page is types/<name>.md with the dot as an underscore
// (types/fs_file.md), and its table is named after the root file's stem, as every table is.
//
// A template is a signature template (core.ParseSignature). The result after ` -> ` is free text to the end of the line
// — `-> array`, `-> [dir, file]`, `-> undefined`, `-> value` where the type depends on the input — and is documentation
// only: it reaches the page, never the table. Source order carries no meaning: entries may sit anywhere in their files,
// and a page lists them alphabetically. The generated kavun_gen.go holds, per package: `var <stem>Methods =
// []Method{…}` for each type with a members file (its descriptor says `Methods: <stem>Methods,`; a type whose
// <stem>_members.go does not exist — record, module — gets no table, and its descriptor is untouched; a members file
// that exists must declare at least one member), an init that fills the builtin function table, and an init that
// declares and registers the module. A module package that needs more than its functions (constants, owned types)
// declares `func moduleExtras(m *core.ModuleBuilder)`; the generated init calls it before Register.
//
// The pages: types/<type>.md, builtins.md, stdlib/<module>.md and the stdlib.md index. A type, module or builtins
// page is its title, the root block (summary, then the intro verbatim, with its own `##` sections), then
// `## Members` (`## Functions`): an index with one linked line per entry, then one entry per member in alphabetical
// (byte) order — an explicit anchor <a id="name"></a>, a `###` heading holding the signature in code (every reading
// of the template as name(reading) -> result, joined by " / "), then the summary and the body. The page of a type
// with no members file is its title and root block alone: no `## Members`, no index.
//
// A body is Markdown, kept verbatim; because gofmt reformats the doc comment of a declaration, no line may be
// indented outside a fenced block, and a fenced block is either unindented or written in gofmt's code-block form (a
// tab before every line), which the page renders without the tab. Fences are backticks only (`~~~` is refused),
// closed as in CommonMark by a line of at least as many backticks. A `#` or `##` heading appears only in a type,
// module or builtins block; an entry's body uses `####` and deeper to nest under its `###` heading (`###` is
// accepted, at the entry's level). A list must be followed by a blank line: an unindented line right after a list
// item continues that item (CommonMark lazy continuation).
//
// A fenced ```go block of a doc block is an example: a standalone Kavun script that TestDocExamples (kavun_test.go)
// runs and checks through its `// =>` and `// Error:` comments (examples.go, Examples). A block that is not a runnable
// script is fenced ```go norun [reason]; the page shows it as a plain ```go fence.
//
// The generator reads what the build compiles: a file excluded by a build constraint (//go:build, a _GOOS/_GOARCH
// suffix) is skipped, so a members file excluded on a build makes its type members-less on that build (a
// page, no table).
//
// Packages under <kavun root>/tmp/, the repository's gitignored scratch folder, are never read. docs/reference is
// generated only from a run that covers the whole module; a run over part of it (with an explicit -docs) generates
// the pages of the packages it read and never reports or deletes another page as stale (see Run).
package kavungen
