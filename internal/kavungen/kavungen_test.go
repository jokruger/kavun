package kavungen

import (
	"bytes"
	"flag"
	"go/format"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jokruger/kavun/internal/genfile"
	"github.com/jokruger/kavun/internal/kavundocs"
	"github.com/jokruger/kavun/reference"
)

var update = flag.Bool("update", false, "rewrite the golden files under testdata/ok and the generated files of the embedder module testdata/embed")

// repoRoot answers the repository root, found from this file's position rather than the working directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// got answers the content of the first generated file, "" when there is none.
func got(files []File) string {
	if len(files) == 0 {
		return ""
	}
	return string(files[0].Content)
}

func loadContext(t *testing.T) *Context {
	t.Helper()
	ctx, err := LoadContext(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// TestGeneratedInSync checks both stages of `go generate ./...` against the repository, each with the flags of its
// gen.go line. Stage 1, sources → tables + reference: kavun-gen's -check over the whole repository — every committed
// kavun_gen.go and kavun_reference.json must be what the generator produces now. Stage 2, reference → pages: the
// reference generated in memory, rendered by kavun-docs — every page under docs/reference/ must be what it renders
// (a page there that nothing renders is stale). It first pins the emitter's line width: no generated kavun_gen.go
// line is wider than maxLine.
func TestGeneratedInSync(t *testing.T) {
	// every kavun_gen.go the generator produces for the repository stays within maxLine
	root := repoRoot(t)
	dirs, err := ResolvePatterns(root, []string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	docsURL := genDocsURL(t)
	refPath := filepath.Join(t.TempDir(), reference.FileName)
	files, err := Generate(loadContext(t), SkipScratch(root, dirs), Options{Root: root, Reference: refPath, Origin: KavunModule, DocsURL: docsURL})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	var refContent []byte
	for _, f := range files {
		switch {
		case filepath.Base(f.Path) == GeneratedFile:
			checkLineWidth(t, f.Path, f.Content)
			n++
		case f.Path == refPath:
			refContent = f.Content
		}
	}
	if n == 0 || refContent == nil {
		t.Fatalf("generated %d kavun_gen.go, reference %v", n, refContent != nil)
	}

	// stage 1
	var stdout, stderr strings.Builder
	err = Run(RunConfig{Dir: repoRoot(t), Check: true, DocsURL: docsURL, Patterns: []string{"./..."}, Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		t.Fatalf("%v:\n%s%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() > 0 {
		t.Fatalf("unexpected note: %s", stderr.String())
	}

	// the run above checked kavun_reference.json too, with gen.go's -docs-url: it is Kavun's, against the core
	// version of vm/bytecode.go, and Read takes what Write wrote
	ref, err := reference.ReadFile(filepath.Join(root, reference.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if ref.Origin != KavunModule || ref.CoreVersion != loadContext(t).coreVersion || ref.Docs != docsURL {
		t.Fatalf("unexpected header: origin %q, core_version %q, docs %q", ref.Origin, ref.CoreVersion, ref.Docs)
	}

	// stage 2, from the reference generated above rather than the file on disk, so that it holds whatever stage 1
	// found
	ref, err = reference.Read(bytes.NewReader(refContent))
	if err != nil {
		t.Fatal(err)
	}
	site, err := kavundocs.NewSite([]*reference.Reference{ref}, []*reference.Reference{ref}, nil)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := site.Files(filepath.Join(root, genDocsOut(t)))
	if err != nil {
		t.Fatal(err)
	}
	diffs, err := genfile.Check(pages)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) > 0 {
		t.Fatalf("%d page(s) out of date — run `go generate ./...`:\n%s", len(diffs), strings.Join(diffs, ""))
	}
}

// genDirective is gen.go's kavun-gen line: TestGeneratedInSync checks with the flags `go generate` runs with.
var genDirective = regexp.MustCompile(`(?m)^//go:generate go run \./cmd/kavun-gen (.*)$`)

// genDocsURL answers the -docs-url value of gen.go's `//go:generate go run ./cmd/kavun-gen …` line.
func genDocsURL(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(repoRoot(t), "gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	m := genDirective.FindSubmatch(src)
	if m == nil {
		t.Fatal("gen.go: no `//go:generate go run ./cmd/kavun-gen …` line")
	}
	args := strings.Fields(string(m[1]))
	for i, a := range args {
		if v, ok := strings.CutPrefix(a, "-docs-url="); ok {
			return v
		}
		if a == "-docs-url" && i+1 < len(args) {
			return args[i+1]
		}
	}
	t.Fatalf("gen.go: the kavun-gen line %q has no -docs-url value", m[1])
	return ""
}

// genDocsLine is gen.go's kavun-docs line.
var genDocsLine = regexp.MustCompile(`(?m)^//go:generate go run \./cmd/kavun-docs (.*)$`)

// genDocsOut answers the -out value of gen.go's `//go:generate go run ./cmd/kavun-docs …` line, which must render
// the one reference kavun-gen writes (kavun_reference.json, at the root) and nothing else: TestGeneratedInSync
// renders that reference from memory.
func genDocsOut(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(repoRoot(t), "gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	m := genDocsLine.FindSubmatch(src)
	if m == nil {
		t.Fatal("gen.go: no `//go:generate go run ./cmd/kavun-docs …` line")
	}
	var out string
	var refs []string
	args := strings.Fields(string(m[1]))
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-out" && i+1 < len(args):
			out = args[i+1]
			i++
		case strings.HasPrefix(a, "-out="):
			out = strings.TrimPrefix(a, "-out=")
		case strings.HasPrefix(a, "-"):
			t.Fatalf("gen.go: the kavun-docs line %q has a flag this test does not apply: %s", m[1], a)
		default:
			refs = append(refs, a)
		}
	}
	if out == "" || len(refs) != 1 || refs[0] != reference.FileName {
		t.Fatalf("gen.go: the kavun-docs line %q must be -out <dir> %s", m[1], reference.FileName)
	}
	return filepath.FromSlash(out)
}

// checkLineWidth fails on any line of a generated Go file wider than maxLine columns, a tab counting 4 — the bound
// the emitter's long-entry split (tooLong) exists to keep.
func checkLineWidth(t *testing.T, name string, content []byte) {
	t.Helper()
	for i, line := range strings.Split(string(content), "\n") {
		if w := utf8.RuneCountInString(strings.ReplaceAll(line, "\t", "    ")); w > maxLine {
			t.Errorf("%s:%d: %d columns, over %d", name, i+1, w, maxLine)
		}
	}
}

// TestScratchIsSkipped: a package under <kavun root>/tmp/ — here a members file without its type, which would stop
// generation anywhere else — is not read by a run over ./... .
func TestScratchIsSkipped(t *testing.T) {
	root := repoRoot(t)
	if err := os.MkdirAll(filepath.Join(root, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	scratch, err := os.MkdirTemp(filepath.Join(root, "tmp"), "kavungen-scratch-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(scratch) })
	src, err := os.ReadFile(filepath.Join(root, "core", "array_members.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scratch, "array_members.go"), src, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr strings.Builder
	if err := Run(RunConfig{Dir: root, Check: true, DocsURL: genDocsURL(t), Patterns: []string{"./..."}, Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("the scratch package was read: %v\n%s", err, stdout.String())
	}
	if got := SkipScratch(root, []string{scratch, filepath.Join(root, "core"), filepath.Join(root, "tmpx")}); len(got) != 2 || slices.Contains(got, scratch) {
		t.Fatalf("SkipScratch: %v", got)
	}
}

// TestGolden generates each fixture package under testdata/ok and compares it with its kavun_gen.go.golden and its
// reference file with kavun_reference.json.golden; a fixture with no golden file must generate nothing. The pages
// of each fixture's reference are internal/kavundocs's goldens.
func TestGolden(t *testing.T) {
	ctx := *loadContext(t)
	ctx.coreVersion = "0.0" // the goldens do not change with Kavun's core version
	for _, name := range []string{"core", "ext", "builtins", "module", "module_plain", "none", "pageonly"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", "ok", name)
			refPath := filepath.Join(t.TempDir(), reference.FileName)
			opts := Options{Root: dir, Reference: refPath, Origin: "example.com/fixture"}
			files, err := Generate(&ctx, []string{dir}, opts)
			if err != nil {
				t.Fatal(err)
			}
			var refContent []byte
			goFiles := map[string][]byte{}
			for _, f := range files {
				if f.Path == refPath {
					refContent = f.Content
				} else {
					goFiles[f.Path] = f.Content
				}
			}
			n := 0
			if refContent != nil {
				n = 1
			}
			checkGolden(t, filepath.Join(dir, reference.FileName+".golden"), refContent, n)

			golden := filepath.Join(dir, GeneratedFile+".golden")
			if content := goFiles[filepath.Join(dir, GeneratedFile)]; content != nil {
				checkLineWidth(t, golden, content)
			}
			checkGolden(t, golden, goFiles[filepath.Join(dir, GeneratedFile)], len(goFiles))

			// the core fixture's zz_members.go is excluded by //go:build ignore: its member must not appear
			for _, content := range goFiles {
				if strings.Contains(string(content), "widgetCopy") {
					t.Fatal("a member from a build-excluded file was generated")
				}
			}

			// deterministic: a second run gives the same bytes
			again, err := Generate(&ctx, []string{dir}, opts)
			if err != nil || !slices.EqualFunc(again, files, func(a, b File) bool { return a.Path == b.Path && string(a.Content) == string(b.Content) }) {
				t.Fatal("a second generation differs from the first")
			}
		})
	}
}

// checkGolden compares content with a golden file (rewriting it with -update); a missing golden file means nothing
// may be generated. n is the number of files generated in the golden file's category.
func checkGolden(t *testing.T, golden string, content []byte, n int) {
	t.Helper()
	if *update {
		if content != nil {
			if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, content, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	want, err := os.ReadFile(golden)
	if os.IsNotExist(err) {
		if n != 0 {
			t.Fatalf("expected no generated file for %s, got %d", golden, n)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || content == nil {
		t.Fatalf("expected one generated file for %s, got %d", golden, n)
	}
	if string(content) != string(want) {
		t.Fatalf("output differs from %s (run with -update after checking):\n%s", golden, genfile.UnifiedDiff("golden", "generated", string(want), string(content)))
	}
}

// TestStaleFileIsDeleted: a package that declares nothing any more answers a nil-content file, which Write deletes
// and Check reports while it still exists.
func TestStaleFileIsDeleted(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"plain.go", GeneratedFile} {
		src, err := os.ReadFile(filepath.Join("testdata", "ok", "stale", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), src, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files, err := Generate(loadContext(t), []string{dir}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Content != nil {
		t.Fatalf("expected one deletion, got %v", files)
	}
	if diffs, _ := genfile.Check(files); len(diffs) != 1 {
		t.Fatalf("Check should report the stale file, got %v", diffs)
	}
	if err := genfile.Write(files); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, GeneratedFile)); !os.IsNotExist(err) {
		t.Fatal("the stale file still exists")
	}
	if diffs, _ := genfile.Check(files); len(diffs) != 0 {
		t.Fatalf("Check after Write should be clean, got %v", diffs)
	}
}

// TestForeignGeneratedFileIsRefused: a kavun_gen.go that kavun-gen did not write is never replaced.
func TestForeignGeneratedFileIsRefused(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, GeneratedFile), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Generate(loadContext(t), []string{dir}, Options{})
	if err == nil || !strings.Contains(err.Error(), "was not written by kavun-gen") {
		t.Fatalf("expected a refusal, got %v", err)
	}
}

// TestWriteAndCheck: Write produces exactly what Check then finds in sync, and Check diffs a changed file.
func TestWriteAndCheck(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"path.go"} {
		src, err := os.ReadFile(filepath.Join("testdata", "ok", "module", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), src, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx := loadContext(t)
	files, err := Generate(ctx, []string{dir}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if diffs, _ := genfile.Check(files); len(diffs) != 1 || !strings.Contains(diffs[0], "+package path") {
		t.Fatalf("Check before Write should report the missing file, got %v", diffs)
	}
	if err := genfile.Write(files); err != nil {
		t.Fatal(err)
	}
	if diffs, _ := genfile.Check(files); len(diffs) != 0 {
		t.Fatalf("Check after Write should be clean, got %v", diffs)
	}
	target := filepath.Join(dir, GeneratedFile)
	if err := os.WriteFile(target, append(slices.Clone(files[0].Content), "// edited\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	if diffs, _ := genfile.Check(files); len(diffs) != 1 || !strings.Contains(diffs[0], "-// edited") {
		t.Fatalf("Check should diff the edited file, got %v", diffs)
	}
}

// fixture builds a one-file-per-entry package in memory.
func fixture(files ...string) []sourceFile {
	var out []sourceFile
	for i := 0; i < len(files); i += 2 {
		out = append(out, sourceFile{Name: files[i], Src: []byte(files[i+1])})
	}
	return out
}

// A minimal type root file and a members-file prefix with one open section, shared by the strictness cases.
const (
	widgetRoot     = "package core\n\n// A widget.\n//\n//kavun:type widget\n"
	membersHead    = "package core\n\n// The widget's members.\n\n"
	memberFunc     = "(_ VM, v Value, id member.ID, args []Value) (Value, error) { return v, nil }\n"
	lenMember      = "// Answer the length.\n//\n//kavun:member len pure () -> int\nfunc wLen" + memberFunc
	builtinsHead   = "package vm\n\n//kavun:builtins\n\n"
	nativeFunc     = "(vm core.VM, args []core.Value) (core.Value, error) { return core.Undefined, nil }\n"
	moduleHead     = "package path\n\n// Paths.\n//\n//kavun:module path id=value.ModulePath pure\n\n"
	joinFn         = "// Join.\n//\n//kavun:fn join slot=0 pure (...parts) -> int\nfunc pathJoin" + nativeFunc
	widgetMembersF = "widget_members.go"
)

// TestStrictness: every rule that stops generation, each with the file:line it must name.
func TestStrictness(t *testing.T) {
	ctx := loadContext(t)
	cases := []struct {
		name  string
		files []sourceFile
		want  string // a substring of the error, starting with file:line where the rule has a position
	}{
		// the directive line itself
		{"unknown kind", fixture("a.go", "package x\n\n// S.\n//\n//kavun:frobnicate x\n"),
			"fx/a.go:5: unknown directive kind \"frobnicate\""},
		{"missing kind", fixture("a.go", "package x\n\n//kavun:\n"),
			"fx/a.go:3: malformed directive: missing kind"},
		{"malformed member line", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member len\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: malformed directive: want `//kavun:member"},
		{"malformed purity", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member len pur ()\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: malformed directive: want `pure` or `impure`"},
		{"malformed type line", fixture("widget.go", "package core\n\n// W.\n//\n//kavun:type widget extra\n"),
			"fx/widget.go:5: malformed directive: want `//kavun:type <name>`"},
		{"malformed builtin slot", fixture("b.go", builtinsHead+"// S.\n//\n//kavun:builtin len slot=x pure (x)\nfunc bLen"+nativeFunc),
			"fx/b.go:7: malformed directive: slot \"x\""},
		{"malformed builtin without slot", fixture("b.go", builtinsHead+"// S.\n//\n//kavun:builtin len pure (x) -> int\nfunc bLen"+nativeFunc),
			"fx/b.go:7: malformed directive: want `//kavun:builtin"},
		{"malformed module line", fixture("p.go", "package path\n\n// P.\n//\n//kavun:module path\n"),
			"fx/p.go:5: malformed directive: want `//kavun:module"},
		{"builtins takes no fields", fixture("b.go", "package vm\n\n//kavun:builtins now\n"),
			"fx/b.go:3: malformed directive: `//kavun:builtins` takes no fields"},
		{"two directives in one group", fixture("widget.go", "package core\n\n// W.\n//kavun:type widget\n//kavun:type gadget\n"),
			"fx/widget.go:5: more than one //kavun: directive in one comment group"},

		// removed kinds and keys; the result
		{"section is not a kind", fixture("a.go", "package x\n\n//kavun:section Size\n"),
			"fx/a.go:3: unknown directive kind \"section\""},
		{"shared is not a kind", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:shared is_true MemberIsTrue pure () -> bool\n"),
			"fx/widget_members.go:7: unknown directive kind \"shared\""},
		{"with= is not a key", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member len pure () with=is_empty -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: malformed directive: unknown key \"with=\""},
		{"with= after the result", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member len pure () -> int with=is_empty\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: malformed directive: unknown key \"with=\""},
		{"missing result", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member len pure ()\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: malformed directive: missing ` -> <result>` after the template"},
		{"empty result", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member len pure () ->  \nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: malformed directive: missing ` -> <result>`"},
		{"builtin without result", fixture("b.go", builtinsHead+"// S.\n//\n//kavun:builtin len slot=0 pure (x)\nfunc bLen"+nativeFunc),
			"fx/b.go:7: malformed directive: missing ` -> <result>`"},
		{"fn without result", fixture("p.go", moduleHead+"// J.\n//\n//kavun:fn join slot=0 pure (...parts)\nfunc pathJoin"+nativeFunc),
			"fx/p.go:9: malformed directive: missing ` -> <result>`"},

		// the doc block
		{"missing summary", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:5: //kavun:member needs a doc block opening with a one-sentence summary"},
		{"missing type summary", fixture("widget.go", "package core\n\n//kavun:type widget\n"),
			"fx/widget.go:3: //kavun:type needs a doc block"},
		{"summary without a period", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// Answer the length\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:5: summary \"Answer the length\" must be one sentence ending in a period"},
		{"summary paragraph without a final period", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// Answer the\n// length\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:5: summary \"Answer the length\" must be one sentence ending in a period"},
		{"summary of two sentences", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// Answer the length.\n// More follows.\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:5: summary \"Answer the length. More follows.\" must be one sentence: a second one starts at \"More follows.\""},

		// templates
		{"template fails ParseSignature", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member len pure (a,b) -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: signature \"(a,b)\": parameters must be separated by `, `"},
		{"redundant reading", fixture("b.go", builtinsHead+"// S.\n//\n//kavun:builtin len slot=0 pure (a) | (b) -> int\nfunc bLen"+nativeFunc),
			"fx/b.go:7: signature \"(a) | (b)\": redundant reading"},
		{"missing template", fixture("p.go", moduleHead+"// J.\n//\n//kavun:fn join slot=0 pure\nfunc pathJoin"+nativeFunc),
			"fx/p.go:9: malformed directive: want `//kavun:fn"},

		// placement
		{"member outside a type's files", fixture("widget.go", widgetRoot, "other.go", "package core\n\n"+lenMember),
			"fx/other.go:5: //kavun:member len outside a type's files"},
		{"dotted type name with another module's prefix", fixture("path.go", moduleHead+joinFn, "segment.go", "package path\n\n// A segment.\n//\n//kavun:type fs.segment\n"),
			"fx/segment.go:5: type \"fs.segment\": a module-owned type's prefix is its module's name, \"path\" here"},
		{"dotted type name outside a module package", fixture("widget.go", "package core\n\n// A widget.\n//\n//kavun:type fs.widget\n"),
			"fx/widget.go:5: type \"fs.widget\": a dotted name belongs to a module-owned type"},
		{"builtin name not a plain name", fixture("b.go", builtinsHead+"// N.\n//\n//kavun:builtin Len slot=0 pure (x) -> int\nfunc bLen"+nativeFunc),
			"fx/b.go:7: malformed directive: \"Len\" is not a name"},
		{"duplicate builtin slot", fixture("b.go", builtinsHead+"// N.\n//\n//kavun:builtin len slot=0 pure (x) -> int\nfunc bLen"+nativeFunc+"\n// M.\n//\n//kavun:builtin cap slot=0 pure (x) -> int\nfunc bCap"+nativeFunc),
			"fx/b.go:12: slot 0 is used twice in the builtins"},
		{"members file without its type", fixture(widgetMembersF, membersHead+lenMember),
			"fx/widget_members.go:7: //kavun:member len: widget_members.go pairs with widget.go, which declares no //kavun:type"},
		{"type in a members file", fixture(widgetMembersF, "package core\n\n// W.\n//\n//kavun:type widget\n"),
			"fx/widget_members.go:5: //kavun:type belongs in the type's root file"},
		{"second type in one file", fixture("widget.go", widgetRoot+"\n// G.\n//\n//kavun:type gadget\n", widgetMembersF, membersHead+lenMember),
			"fx/widget.go:9: a second //kavun:type in widget.go"},
		{"type declared twice", fixture("widget.go", widgetRoot, "other.go", widgetRoot, widgetMembersF, membersHead+lenMember),
			"fx/widget.go:5: type \"widget\" is declared twice"},
		{"a root `## Members` on a type with members", fixture("widget.go", "package core\n\n// A widget.\n//\n// ## Members\n//\n//kavun:type widget\n", widgetMembersF, membersHead+lenMember),
			"fx/widget.go:5: a `## Members` heading in the //kavun:type block: kavun-docs adds that section"},
		{"a root `## Functions` on the builtins", fixture("b.go", "package vm\n\n// ## Functions\n//\n//kavun:builtins\n"),
			"fx/b.go:3: a `## Functions` heading in the //kavun:builtins block"},
		{"members file without members", fixture("widget.go", widgetRoot, widgetMembersF, membersHead),
			"fx/widget.go:5: type \"widget\": members file widget_members.go declares no //kavun:member"},
		{"member not attached to a func", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member len pure () -> int\n\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: //kavun:member must be in the doc comment of the Go function it declares"},
		{"member attached to a method", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member len pure () -> int\nfunc (w W) wLen"+memberFunc),
			"fx/widget_members.go:7: //kavun:member must be in the doc comment of the Go function it declares"},
		{"member attached to the wrong shape", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member len pure () -> int\nfunc wLen"+nativeFunc),
			"fx/widget_members.go:7: //kavun:member len: wLen must have the shape func(vm VM, v Value, id member.ID, args []Value) (Value, error)"},
		{"builtin attached to the wrong shape", fixture("b.go", builtinsHead+"// S.\n//\n//kavun:builtin len slot=0 pure (x) -> int\nfunc bLen"+memberFunc),
			"fx/b.go:7: //kavun:builtin len: bLen must have the shape func(vm core.VM, args []core.Value) (core.Value, error)"},
		{"fn not attached", fixture("p.go", moduleHead+"// J.\n//\n//kavun:fn join slot=0 pure (...parts) -> int\n"),
			"fx/p.go:9: //kavun:fn must be in the doc comment of the Go function it declares"},
		{"free-standing kind attached to a declaration", fixture("widget.go", "package core\n\n// W.\n//\n//kavun:type widget\nvar TypeWidget = 1\n"),
			"fx/widget.go:5: //kavun:type must be free-standing"},
		{"directive inside a function body", fixture("a.go", "package x\n\nfunc f() {\n\t// S.\n\t//kavun:builtins\n}\n"),
			"fx/a.go:5: //kavun:builtins inside a function body"},

		// names
		{"unknown member name", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member no_such_member pure () -> int\nfunc wX"+memberFunc),
			"fx/widget_members.go:7: \"no_such_member\" is neither a builtin member name"},
		{"unknown fn name", fixture("p.go", moduleHead+"// J.\n//\n//kavun:fn no_such_member slot=0 pure () -> int\nfunc pathJoin"+nativeFunc),
			"fx/p.go:9: \"no_such_member\" is neither a builtin member name"},
		{"duplicate member", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+lenMember+"\n// Again.\n//\n//kavun:member len pure () -> int\nfunc wLen2"+memberFunc),
			"fx/widget_members.go:12: member \"len\" is declared twice for type \"widget\" (first at fx/widget_members.go:7)"},
		{"pure _in_place member", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member sort_in_place pure () -> int\nfunc wSort"+memberFunc),
			"fx/widget_members.go:7: //kavun:member sort_in_place: an _in_place member mutates its receiver and must be impure"},
		{"impure conversion member", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member int impure ([default]) -> int\nfunc wInt"+memberFunc),
			"fx/widget_members.go:7: //kavun:member int: a conversion member is the conversion int(x) calls: it must be pure with the template ([default])"},
		{"conversion member with another template", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member time pure (zone) -> time\nfunc wTime"+memberFunc),
			"fx/widget_members.go:7: //kavun:member time: a conversion member is the conversion time(x) calls: it must be pure with the template ([default])"},

		// builtins and modules
		{"builtin without builtins", fixture("b.go", "package vm\n\n// S.\n//\n//kavun:builtin len slot=0 pure (x) -> int\nfunc bLen"+nativeFunc),
			"fx/b.go:5: //kavun:builtin len: the package has no //kavun:builtins"},
		{"second builtins", fixture("b.go", builtinsHead+"//kavun:builtins\n"),
			"fx/b.go:5: a second //kavun:builtins (the first is at fx/b.go:3)"},
		{"duplicate builtin slot", fixture("b.go", builtinsHead+"// S.\n//\n//kavun:builtin len slot=0 pure (x) -> int\nfunc bLen"+nativeFunc+"\n// T.\n//\n//kavun:builtin copy slot=0 pure (x) -> int\nfunc bCopy"+nativeFunc),
			"fx/b.go:12: slot 0 is used twice in the builtins (first by len at fx/b.go:7)"},
		{"duplicate builtin name", fixture("b.go", builtinsHead+"// S.\n//\n//kavun:builtin len slot=0 pure (x) -> int\nfunc bLen"+nativeFunc+"\n// T.\n//\n//kavun:builtin len slot=1 pure (x) -> int\nfunc bLen2"+nativeFunc),
			"fx/b.go:12: \"len\" is declared twice in the builtins"},
		{"builtin slot out of range", fixture("b.go", builtinsHead+"// S.\n//\n//kavun:builtin len slot=128 pure (x) -> int\nfunc bLen"+nativeFunc),
			"fx/b.go:7: slot 128 is out of range (0–127)"},
		{"fn without module", fixture("p.go", "package path\n\n"+joinFn),
			"fx/p.go:5: //kavun:fn join: the package has no //kavun:module"},
		{"second module", fixture("p.go", moduleHead+"// Again.\n//\n//kavun:module path id=value.ModulePath pure\n"),
			"fx/p.go:9: a second //kavun:module (the first is at fx/p.go:5): one module per package"},
		{"module without id=", fixture("p.go", "package nope\n\n// N.\n//\n//kavun:module nope pure\n"),
			"fx/p.go:5: malformed directive: want `//kavun:module <name> id=<go ident> pure|impure`"},
		{"module with a malformed id=", fixture("p.go", "package nope\n\n// N.\n//\n//kavun:module nope id=core.ModuleNope pure\n"),
			"fx/p.go:5: malformed directive: id=core.ModuleNope is not a Go identifier of the package or value.<Ident>"},
		{"module id= not a constant of core/value", fixture("p.go", "package nope\n\n// N.\n//\n//kavun:module nope id=value.ModuleNope pure\n"),
			"fx/p.go:5: module \"nope\": id=value.ModuleNope is not a constant of core/value (core/value/value.go)"},
		{"module id= not declared by the package", fixture("p.go", "package nope\n\n// N.\n//\n//kavun:module nope id=moduleNope pure\n"),
			"fx/p.go:5: module \"nope\": id=moduleNope is not a constant or variable of the package"},

		// member names (//kavun:names)
		{"names with fields", fixture("n.go", "package ext\n\n//kavun:names x\nvar memberNames = map[member.ID]string{}\n"),
			"fx/n.go:3: malformed directive: `//kavun:names` takes no fields"},
		{"names on a func", fixture("n.go", "package ext\n\n//kavun:names\nfunc memberNames() {}\n"),
			"fx/n.go:3: //kavun:names must be in the doc comment of the package-level var holding the names"},
		{"names not a map literal", fixture("n.go", "package ext\n\n//kavun:names\nvar memberNames = makeNames()\n"),
			"fx/n.go:3: //kavun:names memberNames: the var must hold a map[member.ID]string{Key: \"name\", …} literal"},
		{"names of another map type", fixture("n.go", "package ext\n\n//kavun:names\nvar memberNames = map[int]string{1: \"double\"}\n"),
			"fx/n.go:3: //kavun:names memberNames: the var must hold a map[member.ID]string"},
		{"names declaring none", fixture("n.go", "package ext\n\n//kavun:names\nvar memberNames = map[member.ID]string{}\n"),
			"fx/n.go:3: //kavun:names memberNames declares no name"},
		{"names key not an identifier", fixture("n.go", "package ext\n\n//kavun:names\nvar memberNames = map[member.ID]string{\n\tmember.ID(2100): \"double\",\n}\n"),
			"fx/n.go:5: //kavun:names memberNames: the key member.ID(2100) is not an identifier of the package"},
		{"names value not a literal", fixture("n.go", "package ext\n\nconst MemberDouble member.ID = 2100\n\n//kavun:names\nvar memberNames = map[member.ID]string{\n\tMemberDouble: doubleName,\n}\n"),
			"fx/n.go:7: //kavun:names memberNames: the name of MemberDouble is not a string literal"},
		{"names value not a member name", fixture("n.go", "package ext\n\nconst MemberDouble member.ID = 2100\n\n//kavun:names\nvar memberNames = map[member.ID]string{\n\tMemberDouble: \"Double\",\n}\n"),
			"fx/n.go:7: //kavun:names memberNames: \"Double\" is not a member name"},
		{"names key not a constant", fixture("n.go", "package ext\n\nvar MemberDouble = member.ID(2100)\n\n//kavun:names\nvar memberNames = map[member.ID]string{\n\tMemberDouble: \"double\",\n}\n"),
			"fx/n.go:7: //kavun:names: the key MemberDouble of \"double\" is not a constant of the package"},
		{"names declaring a builtin name", fixture("n.go", "package ext\n\nconst MemberLen member.ID = 2100\n\n//kavun:names\nvar memberNames = map[member.ID]string{\n\tMemberLen: \"len\",\n}\n"),
			"fx/n.go:7: //kavun:names: \"len\" is a builtin member name"},
		{"names declaring a name twice", fixture("n.go", "package ext\n\nconst (\n\tMemberA member.ID = 2100\n\tMemberB member.ID = 2101\n)\n\n//kavun:names\nvar memberNames = map[member.ID]string{\n\tMemberA: \"double\",\n\tMemberB: \"double\",\n}\n"),
			"fx/n.go:11: //kavun:names: \"double\" is declared twice (first at fx/n.go:10)"},
		{"names using a key twice", fixture("n.go", "package ext\n\nconst MemberA member.ID = 2100\n\n//kavun:names\nvar memberNames = map[member.ID]string{\n\tMemberA: \"double\",\n\tMemberA: \"twice\",\n}\n"),
			"fx/n.go:8: //kavun:names: the key MemberA names \"double\" and \"twice\" (first at fx/n.go:7)"},
		{"a second names", fixture("n.go", "package ext\n\nconst (\n\tMemberA member.ID = 2100\n\tMemberB member.ID = 2101\n)\n\n//kavun:names\nvar memberNames = map[member.ID]string{MemberA: \"double\"}\n\n//kavun:names\nvar moreNames = map[member.ID]string{MemberB: \"twice\"}\n"),
			"fx/n.go:11: a second //kavun:names (the first is at fx/n.go:8): one per package"},
		{"member with an undeclared name", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member triple pure () -> int\nfunc wTriple"+memberFunc),
			"fx/widget_members.go:7: \"triple\" is neither a builtin member name (core/member/builtin.go) nor declared by a //kavun:names of the run"},
		{"duplicate fn slot", fixture("p.go", moduleHead+joinFn+"\n// B.\n//\n//kavun:fn base slot=0 pure (p) -> int\nfunc pathBase"+nativeFunc),
			"fx/p.go:14: slot 0 is used twice in module path (first by join at fx/p.go:9)"},
		{"module extras of the wrong shape", fixture("p.go", moduleHead+joinFn+"\nfunc moduleExtras() {}\n"),
			"fx/p.go:12: func moduleExtras must be `func moduleExtras(m *core.ModuleBuilder)`"},

		// module constants
		{"malformed const without a type", fixture("p.go", moduleHead+"// S.\n//\n//kavun:const pi\nvar constPi = 1\n"),
			"fx/p.go:9: malformed directive: want `//kavun:const <name> -> <type>`"},
		{"malformed const with a bare type", fixture("p.go", moduleHead+"// S.\n//\n//kavun:const pi float\nvar constPi = 1\n"),
			"fx/p.go:9: malformed directive: want `//kavun:const <name> -> <type>`"},
		{"const name not a plain name", fixture("p.go", moduleHead+"// S.\n//\n//kavun:const Pi -> float\nvar constPi = 1\n"),
			"fx/p.go:9: malformed directive: \"Pi\" is not a name"},
		{"const without module", fixture("p.go", "package path\n\n// S.\n//\n//kavun:const pi -> float\nvar constPi = 1\n"),
			"fx/p.go:5: //kavun:const pi: the package has no //kavun:module"},
		{"const on a func", fixture("p.go", moduleHead+"// S.\n//\n//kavun:const pi -> float\nfunc constPi() {}\n"),
			"fx/p.go:9: //kavun:const pi must be in the doc comment of the package-level var holding its value"},
		{"free-standing const", fixture("p.go", moduleHead+"// S.\n//\n//kavun:const pi -> float\n\nvar constPi = 1\n"),
			"fx/p.go:9: //kavun:const pi must be in the doc comment of the package-level var holding its value"},
		{"const on a Go const", fixture("p.go", moduleHead+"// S.\n//\n//kavun:const pi -> float\nconst constPi = 1\n"),
			"fx/p.go:9: //kavun:const pi is on a Go const: the value is a core.Value, which a Go const cannot hold; declare it with var"},
		{"const on a var of two names", fixture("p.go", moduleHead+"// S.\n//\n//kavun:const pi -> float\nvar constPi, constE = 1, 2\n"),
			"fx/p.go:9: //kavun:const pi is on a var declaring 2 names (constPi, constE): one var, one name"},
		{"const on a var group", fixture("p.go", moduleHead+"// S.\n//\n//kavun:const pi -> float\nvar (\n\tconstPi = 1\n\tconstE  = 2\n)\n"),
			"fx/p.go:9: //kavun:const pi is on a var declaring 2 names (constPi, constE)"},
		{"const on the blank var", fixture("p.go", moduleHead+"// S.\n//\n//kavun:const pi -> float\nvar _ = 1\n"),
			"fx/p.go:9: //kavun:const pi is on the blank var `_`"},
		{"const inside a function body", fixture("p.go", moduleHead+"func f() {\n\t// S.\n\t//\n\t//kavun:const pi -> float\n\tvar x = 1\n\t_ = x\n}\n"),
			"fx/p.go:10: //kavun:const inside a function body"},
		{"unknown const name", fixture("p.go", moduleHead+"// S.\n//\n//kavun:const no_such_member -> int\nvar constX = 1\n"),
			"fx/p.go:9: \"no_such_member\" is neither a builtin member name"},
		{"const without summary", fixture("p.go", moduleHead+"//kavun:const pi -> float\nvar constPi = 1\n"),
			"fx/p.go:7: //kavun:const needs a doc block opening with a one-sentence summary"},
		{"## heading in a const body", fixture("p.go", moduleHead+"// S.\n//\n// ## Sub\n//\n//kavun:const pi -> float\nvar constPi = 1\n"),
			"fx/p.go:9: a `#` or `##` heading belongs to the page's structure"},
		{"duplicate const", fixture("p.go", moduleHead+"// S.\n//\n//kavun:const pi -> float\nvar constPi = 1\n\n// T.\n//\n//kavun:const pi -> float\nvar constPi2 = 1\n"),
			"fx/p.go:14: constant \"pi\" is declared twice in module path (first at fx/p.go:9)"},
		{"const named like a function", fixture("p.go", moduleHead+joinFn+"\n// S.\n//\n//kavun:const join -> int\nvar constJoin = 1\n"),
			"fx/p.go:14: constant \"join\" has the name of a function of module path (at fx/p.go:9)"},
		{"a root `## Constants` on a module with constants", fixture("p.go", "package path\n\n// Paths.\n//\n// ## Constants\n//\n//kavun:module path id=value.ModulePath pure\n\n// S.\n//\n//kavun:const pi -> float\nvar constPi = 1\n"),
			"fx/p.go:5: a `## Constants` heading in the //kavun:module block: kavun-docs adds that section"},
		{"module root heading with a constant's anchor", fixture("p.go", "package path\n\n// Paths.\n//\n// ## Pi\n//\n//kavun:module path id=value.ModulePath pure\n\n// S.\n//\n//kavun:const pi -> float\nvar constPi = 1\n"),
			`fx/p.go:5: the heading "Pi" of the //kavun:module block has the anchor #pi, which is the entry pi's on the same page`},
		{"shorthand naming nothing in a const", fixture("p.go", moduleHead+"// S; see [x](#nope).\n//\n//kavun:const pi -> float\nvar constPi = 1\n"),
			"fx/p.go:7: link #nope: module path has no entry nope"},

		// Markdown in doc blocks (S2a-2)
		{"indented line outside a fence (spaces)", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// - item\n//   continued\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:8: doc line is indented outside a fenced block"},
		{"indented line outside a fence (tab)", fixture("widget.go", "package core\n\n// W.\n//\n//\tcode\n//\n//kavun:type widget\n"),
			"fx/widget.go:5: doc line is indented outside a fenced block"},
		{"unclosed fence", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// ```go\n// w.len()\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: unclosed fenced block"},
		{"space-indented line inside a fence", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// ```go\n// f(func() {\n//     g()\n// })\n// ```\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:9: doc line inside a fenced block is indented with spaces"},
		{"fence mixing tab and unindented lines", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// ```go\n// f(func() {\n//\tg()\n// })\n// ```\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:9: a fenced block mixes tab-indented and unindented lines"},
		{"tab fence indented past its tab throughout", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// ```go\n//\t  g()\n// ```\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:8: every line of a tab-indented fenced block is indented past its tab"},
		{"# heading in a member body", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// # Title\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: a `#` or `##` heading belongs to the page's structure"},
		{"## heading in a member body", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// ## Sub\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: a `#` or `##` heading belongs to the page's structure"},
		{"## heading in a builtin body", fixture("b.go", builtinsHead+"// S.\n//\n// ## Sub\n//\n//kavun:builtin len slot=0 pure (x) -> int\nfunc bLen"+nativeFunc),
			"fx/b.go:7: a `#` or `##` heading belongs to the page's structure"},
		{"~~~ fence", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// ~~~go\n// w.len()\n// ~~~\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: a `~~~` fence: only backtick fences are recognised"},
		{"go fence with an unknown word", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// ```go nofmt\n// w.len()\n// ```\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: a `go` fence takes nothing after the language but the `norun` marker"},
		{"fence closed only by as many backticks", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// ````md\n// ```go\n// x\n// ```\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: unclosed fenced block: the doc block ends before a closing line of at least 4 backticks"},

		// links (per package: the form, the shorthand, the heading/entry clash)
		{"relative link in a body", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// See [string](string.md#x).\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: link string.md#x: a relative link names a page of one layout; name the target with a kavun: link"},
		{"relative link in a summary", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// Answer the length; see [guide](../../x.md).\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:5: link ../../x.md: a relative link"},
		{"relative link definition in a root block", fixture("widget.go", "package core\n\n// A widget.\n//\n// [def]: other.md\n//\n//kavun:type widget\n"),
			"fx/widget.go:5: link other.md: a relative link"},
		{"relative link on a summary's second line", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// Answer the length;\n// see [guide](x.md).\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:6: link x.md: a relative link"},
		{"malformed kavun: link", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// See [x](kavun:member/array.splice).\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			`fx/widget_members.go:7: link kavun:member/array.splice: unknown kind "member"`},
		{"shorthand naming nothing", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n// See [x](#nope).\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			"fx/widget_members.go:7: link #nope: type widget has no entry nope and its block no heading with that slug"},
		{"shorthand to another page's heading", fixture("b.go", builtinsHead+"// S; see [x](#durations).\n//\n//kavun:builtin len slot=0 pure (x) -> int\nfunc bLen"+nativeFunc),
			"fx/b.go:5: link #durations: the builtins has no entry durations"},
		{"root heading with an entry's anchor", fixture("widget.go", "package core\n\n// A widget.\n//\n// ### Len\n//\n//kavun:type widget\n", widgetMembersF, membersHead+lenMember),
			`fx/widget.go:5: the heading "Len" of the //kavun:type block has the anchor #len, which is the entry len's on the same page: rename the heading`},
		{"module root heading with a function's anchor", fixture("p.go", "package path\n\n// Paths.\n//\n// ## Join\n//\n//kavun:module path id=value.ModulePath pure\n\n"+joinFn),
			`fx/p.go:5: the heading "Join" of the //kavun:module block has the anchor #join`},

		// Go itself
		{"unparsable Go", fixture("a.go", "package x\n\nfunc {\n"),
			"fx/a.go:3:6: expected 'IDENT'"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			content, _, errs := generateSources(ctx, "fx", c.files)
			if len(errs) == 0 {
				t.Fatalf("expected an error containing %q, generated:\n%s", c.want, content)
			}
			if err := errs.err(); !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected an error containing\n  %q\ngot\n  %v", c.want, err)
			}
		})
	}
}

// TestLinkTargets: with the reference, every shorthand is written in its absolute kavun: form (summary and body,
// examples untouched) and every kavun: target but kavun:doc/ must name something the run declares — a type, a
// member, a module, a module function, a builtin, a heading of the owner's root block — reported at the link's line.
func TestLinkTargets(t *testing.T) {
	ctx := loadContext(t)
	build := func(t *testing.T, memberDoc string) (*reference.Reference, errorList) {
		t.Helper()
		var models []*packageModel
		for _, pkg := range [][]sourceFile{
			fixture("widget.go", "package core\n\n// A widget; see [`len`](#len).\n//\n// ## Notes\n//\n// [Back](#notes).\n//\n//kavun:type widget\n",
				widgetMembersF, membersHead+"// Answer the length; see [the notes](#notes).\n//\n"+memberDoc+"//kavun:member len pure () -> int\nfunc wLen"+memberFunc),
			fixture("b.go", "package vm\n\n// ## Intro\n//\n//kavun:builtins\n\n// S.\n//\n//kavun:builtin len slot=0 pure (x) -> int\nfunc bLen"+nativeFunc),
			fixture("p.go", moduleHead+joinFn),
		} {
			_, m, errs := generateSources(ctx, "fx", pkg)
			if len(errs) > 0 {
				t.Fatal(errs.err())
			}
			models = append(models, m)
		}
		return buildReference(models, "example.com/fixture", "0.0", "", false, nil)
	}

	ref, errs := build(t, "// See [a](#notes), [b](#len), [c](kavun:module/path/join), [d](kavun:builtins#intro), [e](kavun:builtin/len),\n"+
		"// [f](kavun:doc/anything#x), [g](https://example.com/x.md), `[h](#nope)`.\n//\n// ```go\n// x := 1 // [i](#notes)\n// ```\n//\n")
	if len(errs) > 0 {
		t.Fatal(errs.err())
	}
	w := &ref.Types[0]
	if w.Summary != "A widget; see [`len`](kavun:type/widget/len)." || w.Doc.Markdown() != "## Notes\n\n[Back](kavun:type/widget#notes)." {
		t.Fatalf("the root block: %q\n%q", w.Summary, w.Doc.Markdown())
	}
	c := w.Members[0]
	want := "See [a](kavun:type/widget#notes), [b](kavun:type/widget/len), [c](kavun:module/path/join), [d](kavun:builtins#intro), [e](kavun:builtin/len),\n" +
		"[f](kavun:doc/anything#x), [g](https://example.com/x.md), `[h](#nope)`.\n"
	if c.Summary != "Answer the length; see [the notes](kavun:type/widget#notes)." || c.Doc[0].Markdown != want || c.Doc[1].Code != "x := 1 // [i](#notes)" {
		t.Fatalf("the member: %q\n%q\n%q", c.Summary, c.Doc[0].Markdown, c.Doc[1].Code)
	}

	for link, msg := range map[string]string{
		"kavun:type/nope":        "no type nope",
		"kavun:type/widget/nope": "type widget has no member nope",
		"kavun:type/widget#nope": `no heading "nope" in the block of type widget`,
		"kavun:module/nope":      "no module nope",
		"kavun:module/path/nope": "module path has no function or constant nope",
		"kavun:module/path#nope": `no heading "nope" in the block of module path`,
		"kavun:builtin/nope":     "no builtin function nope",
		"kavun:builtins#nope":    `no heading "nope" in the block of the builtins`,
	} {
		_, errs := build(t, "// Text.\n// See [x]("+link+").\n//\n")
		if err := errs.err(); err == nil || !strings.Contains(err.Error(), "fx/widget_members.go:8: link "+link+": "+msg) {
			t.Errorf("%s: got %v, want %q", link, err, msg)
		}
	}
}

// TestPartialRunTargets: a run over part of the module (no reference written) still resolves the targets whose
// owner it declares — a missing member or heading of its own type is refused at its line, a summary's second line
// included — and lets pass a target whose owner another package may declare; the whole-module run refuses that one.
func TestPartialRunTargets(t *testing.T) {
	ctx := loadContext(t)
	write := func(t *testing.T, memberDoc string) string {
		t.Helper()
		dir := t.TempDir()
		for name, src := range map[string]string{
			"widget.go":    widgetRoot,
			widgetMembersF: membersHead + memberDoc + "//kavun:member len pure () -> int\nfunc wLen" + memberFunc,
		} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}
	for doc, want := range map[string]string{
		"// Answer the length.\n//\n// See [x](kavun:type/widget/nope).\n//\n": "widget_members.go:7: link kavun:type/widget/nope: type widget has no member nope",
		"// Answer the length;\n// see [x](kavun:type/widget#nope).\n//\n":     `widget_members.go:6: link kavun:type/widget#nope: no heading "nope" in the block of type widget`,
	} {
		_, err := Generate(ctx, []string{write(t, doc)}, Options{})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("got %v, want %q", err, want)
		}
	}
	elsewhere := write(t, "// Answer the length.\n//\n// See [s](kavun:type/string), [r](kavun:builtin/range), [f](kavun:module/fin/irr).\n//\n")
	if _, err := Generate(ctx, []string{elsewhere}, Options{}); err != nil {
		t.Fatalf("a partial run refused a target of another package: %v", err)
	}
	_, err := Generate(ctx, []string{elsewhere}, Options{Reference: filepath.Join(t.TempDir(), reference.FileName)})
	if err == nil || !strings.Contains(err.Error(), "link kavun:type/string: no type string") {
		t.Fatalf("the whole-module run: %v", err)
	}
}

// TestOtherDirectivesAreIgnored: lines that look like directives but are not ours (another tool's, or `// kavun:`
// with a space) neither count as a directive nor join a doc block.
func TestOtherDirectivesAreIgnored(t *testing.T) {
	ctx := loadContext(t)
	src := membersHead + "// Answer the length.\n//\n//nolint:revive\n//go:noinline\n//kavun:member len pure () -> int\nfunc wLen" + memberFunc +
		"\n// kavun:member is_empty pure () — prose, not a directive\nfunc wIsEmpty" + memberFunc
	scanned, errs := scanPackage("fx", fixture("widget.go", widgetRoot, widgetMembersF, src))
	if len(errs) > 0 {
		t.Fatal(errs.err())
	}
	var got []Directive
	for _, f := range scanned.files {
		got = append(got, f.directives...)
	}
	if len(got) != 2 { // type, member
		t.Fatalf("expected 2 directives, got %d", len(got))
	}
	if d := got[1]; d.Name != "len" || d.Doc.Summary != "Answer the length." || len(d.Doc.Body) != 0 || d.GoFunc != "wLen" {
		t.Fatalf("unexpected member directive %+v", d)
	}
	if _, _, errs := generateSources(ctx, "fx", fixture("widget.go", widgetRoot, widgetMembersF, src)); len(errs) > 0 {
		t.Fatal(errs.err())
	}
}

// TestDocBlock: the summary is the first line, the body keeps its markdown verbatim, and a directive written first
// in its group (accepted, though gofmt moves it last in a doc comment) still takes the other lines as its doc block.
func TestDocBlock(t *testing.T) {
	src := membersHead + "//kavun:member len pure () -> int\n// Answer the length.\n//\n// It counts **elements**.\n//\n// ```go\n// [1].len()   // 1\n// ```\nfunc wLen" + memberFunc
	scanned, errs := scanPackage("fx", fixture(widgetMembersF, src))
	if len(errs) > 0 {
		t.Fatal(errs.err())
	}
	d := scanned.files[0].directives[0]
	want := []string{"It counts **elements**.", "", "```go", "[1].len()   // 1", "```"}
	if d.Doc.Summary != "Answer the length." || !slices.Equal(d.Doc.Body, want) {
		t.Fatalf("unexpected doc block %+v", d.Doc)
	}
	if d.Template != "()" || d.Sig.Arity != 0 || d.Sig.Variadic || !d.Pure {
		t.Fatalf("unexpected directive %+v", d)
	}
}

func TestParseDirective(t *testing.T) {
	cases := []struct {
		text string
		want Directive
	}{
		{"member pad_start pure (n[, fill]) -> array", Directive{Kind: KindMember, Name: "pad_start", Pure: true, Template: "(n[, fill])", Result: "array"}},
		{"member splice impure ([start[, n]]) | (start, n, ...items) -> array", Directive{Kind: KindMember, Name: "splice", Template: "([start[, n]]) | (start, n, ...items)", Result: "array"}},
		{"builtin int slot=6 pure (x[, default]) -> int", Directive{Kind: KindBuiltin, Name: "int", Pure: true, Template: "(x[, default])", Slot: 6, Result: "int"}},
		{"fn split slot=0 impure (p)  ->  [dir, file] ", Directive{Kind: KindFn, Name: "split", Template: "(p)", Result: "[dir, file]"}},
		{"module path id=value.ModulePath pure", Directive{Kind: KindModule, Name: "path", ModuleID: "value.ModulePath", Pure: true}},
		{"module mine id=myModule impure", Directive{Kind: KindModule, Name: "mine", ModuleID: "myModule"}},
		{"names", Directive{Kind: KindNames}},
		{"type array", Directive{Kind: KindType, Name: "array"}},
		{"builtins", Directive{Kind: KindBuiltins}},
	}
	for _, c := range cases {
		got, err := parseDirective(c.text)
		if err != nil {
			t.Fatalf("%q: %v", c.text, err)
		}
		got.Sig = c.want.Sig // compared through Template
		if got.Kind != c.want.Kind || got.Name != c.want.Name || got.Pure != c.want.Pure || got.Template != c.want.Template ||
			got.Slot != c.want.Slot || got.Result != c.want.Result || got.GoFunc != c.want.GoFunc || got.ModuleID != c.want.ModuleID {
			t.Fatalf("%q: got %+v, want %+v", c.text, got, c.want)
		}
	}
}

func TestMethodsVar(t *testing.T) {
	for stem, want := range map[string]string{"array": "arrayMethods", "int_range": "intRangeMethods", "fin_year_fraction": "finYearFractionMethods"} {
		if got := methodsVar(stem); got != want {
			t.Fatalf("methodsVar(%q) = %q, want %q", stem, got, want)
		}
	}
}

// TestGofmtStable: gofmt reformats the doc comment of a declaration (a leading directive moves last, a tab-indented
// fenced block gets blank lines around it); regenerating from the gofmt-formatted sources must give the same Markdown
// and the same Go. testdata/gofmt holds sources that are deliberately not canonical (as .go.txt, so `gofmt -l` does
// not see them); every fixture under testdata/ok is checked too.
func TestGofmtStable(t *testing.T) {
	ctx := loadContext(t)
	dirs := []string{filepath.Join("testdata", "gofmt")}
	for _, name := range []string{"core", "ext", "builtins", "module", "module_plain"} {
		dirs = append(dirs, filepath.Join("testdata", "ok", name))
	}
	for _, dir := range dirs {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			var raw, formatted []sourceFile
			changed := false
			for _, e := range entries {
				name := e.Name()
				if !strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".go.txt") {
					continue
				}
				src, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				if bytes.HasPrefix(src, []byte("//go:build ignore")) {
					continue
				}
				out, err := format.Source(src)
				if err != nil {
					t.Fatal(err)
				}
				changed = changed || !bytes.Equal(out, src)
				name = strings.TrimSuffix(name, ".txt")
				raw = append(raw, sourceFile{Name: name, Src: src})
				formatted = append(formatted, sourceFile{Name: name, Src: out})
			}
			if strings.HasSuffix(dir, "gofmt") && !changed {
				t.Fatal("the testdata/gofmt sources are canonical already: the test would prove nothing")
			}

			goA, mdA := generateInMemory(t, ctx, dir, raw)
			goB, mdB := generateInMemory(t, ctx, dir, formatted)
			if goA != goB {
				t.Fatalf("gofmt changes the generated Go:\n%s", genfile.UnifiedDiff("raw", "gofmt", goA, goB))
			}
			for rel := range mdA {
				if mdA[rel] != mdB[rel] {
					t.Fatalf("gofmt changes %s:\n%s", rel, genfile.UnifiedDiff("raw", "gofmt", mdA[rel], mdB[rel]))
				}
			}
			if len(mdA) != len(mdB) || len(mdA) == 0 {
				t.Fatalf("expected the same pages, got %d and %d", len(mdA), len(mdB))
			}
		})
	}
}

// generateInMemory generates one package from sources: its Go file and its pages (its reference, rendered by
// kavun-docs).
func generateInMemory(t *testing.T, ctx *Context, dir string, files []sourceFile) (string, map[string]string) {
	t.Helper()
	content, model, errs := generateSources(ctx, dir, files)
	if len(errs) > 0 {
		t.Fatal(errs.err())
	}
	ref, errs := buildReference([]*packageModel{model}, "example.com/fixture", "0.0", "", false, nil)
	if len(errs) > 0 {
		t.Fatal(errs.err())
	}
	// a fixture is not Kavun: kavun-docs' own links into Kavun's guide need Kavun's docs URL
	site, err := kavundocs.NewSite([]*reference.Reference{ref}, []*reference.Reference{ref}, map[string]string{reference.KavunOrigin: "https://kavun.example/docs"})
	if err != nil {
		t.Fatal(err)
	}
	pages, err := site.Pages("reference")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for rel, b := range pages {
		out[rel] = string(b)
	}
	return string(content), out
}

// TestGofmtFixtureRendering pins what the non-canonical fixture renders to: the tab-form fence loses its tab and
// gofmt's blank lines, the list, table and headings pass through, and every member is its own anchored entry, listed
// in the index and ordered by name whatever the source order.
func TestGofmtFixtureRendering(t *testing.T) {
	ctx := loadContext(t)
	var files []sourceFile
	for _, name := range []string{"widget.go", "widget_members.go"} {
		src, err := os.ReadFile(filepath.Join("testdata", "gofmt", name+".txt"))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, sourceFile{Name: name, Src: src})
	}
	_, md := generateInMemory(t, ctx, "testdata/gofmt", files)
	page := md["types/widget.md"]
	for _, want := range []string{
		"<a id=\"pad_end\"></a>\n### `pad_end(n[, fill]) -> widget`\n\nPad at the back until length `n`.\n\n- a list item\nwith an unindented continuation line\n",
		"- [`pad_end(n[, fill]) -> widget`](#pad_end)\n- [`pad_start(n[, fill]) -> widget`](#pad_start)\n",
		"### `pad_start(n[, fill]) -> widget`\n\nPad at the front until length `n`.",
		"```go\nw.each(func(x) {\n    print(x)\n\n})\n```",
		"```go\nw.pad_end(3)\n\nw.pad_end(3, 0)\n```",
		"| width | result |\n| --- | --- |\n",
		"### A third-level heading",
		"#### A fourth-level heading",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("the page lacks %q:\n%s", want, page)
		}
	}
	if strings.Index(page, "### `pad_end") > strings.Index(page, "### `pad_start") {
		t.Fatal("the entries are not in alphabetical order")
	}
}

// TestTypeFromTwoPackages: a type two packages of the module declare is refused when the reference is generated.
func TestTypeFromTwoPackages(t *testing.T) {
	dir := filepath.Join("testdata", "ok", "ext")
	twin := t.TempDir()
	for _, name := range []string{"gadget.go", "gadget_members.go"} {
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(twin, name), src, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := Generate(loadContext(t), []string{dir, twin}, Options{Reference: filepath.Join(t.TempDir(), reference.FileName)})
	if err == nil || !strings.Contains(err.Error(), `type "gadget" is declared twice`) {
		t.Fatalf("expected a refusal, got %v", err)
	}
}

// TestNoCascade: a refused doc block drops its directive; the follow-ups its absence trips (an empty section, a type
// with no members) are not reported on top of it.
func TestNoCascade(t *testing.T) {
	ctx := loadContext(t)
	_, _, errs := generateSources(ctx, "fx", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// Answer the length\n//\n//kavun:member len pure () -> int\nfunc wLen"+memberFunc))
	if len(errs) != 1 || !strings.Contains(errs[0], "fx/widget_members.go:5: summary") {
		t.Fatalf("expected the one doc-block error, got:\n%v", errs.err())
	}
}

// TestFenceSpans: backtick fences only, closed by a run at least as long as the opening one.
func TestFenceSpans(t *testing.T) {
	lines := []string{"a", "````md", "```go", "x", "```", "````", "~~~", "```", "y", "``````"}
	want := []fenceSpan{{1, 5}, {7, 9}}
	if got := fenceSpans(lines); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := fenceSpans([]string{"```", "``", "x"}); !slices.Equal(got, []fenceSpan{{0, -1}}) {
		t.Fatalf("unclosed: got %v", got)
	}
}

// TestRenderBody: verbatim, except a fenced block in gofmt's code-block form.
func TestRenderBody(t *testing.T) {
	body := []string{"text", "", "```go", "", "\tf(func() {", "\t    g()", "", "\t})", "", "```", "", "```", "a", "", "b", "```"}
	want := []string{"text", "", "```go", "f(func() {", "    g()", "", "})", "```", "", "```", "a", "", "b", "```"}
	if got := renderBody(body); !slices.Equal(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	// a shorter backtick run inside a longer fence is content, tab form included
	body = []string{"````md", "", "\t```go", "\tx", "\t```", "", "````"}
	want = []string{"````md", "```go", "x", "```", "````"}
	if got := renderBody(body); !slices.Equal(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

// TestPageOnlyStaleGenerated: a package left with only members-less types (a type block, no table) has no Go to generate, so a
// kavun_gen.go from an earlier run is answered with nil content and Write deletes it; the type is still in the
// reference.
func TestPageOnlyStaleGenerated(t *testing.T) {
	ctx := loadContext(t)
	dir := t.TempDir()
	src, err := os.ReadFile(filepath.Join("testdata", "ok", "pageonly", "thing.go"))
	if err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, GeneratedFile)
	for name, content := range map[string]string{"thing.go": string(src), GeneratedFile: generatedHeader + "\npackage pageonly\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	refPath := filepath.Join(t.TempDir(), reference.FileName)
	files, err := Generate(ctx, []string{dir}, Options{Root: dir, Reference: refPath})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Path != stale || files[0].Content != nil || files[1].Path != refPath {
		t.Fatalf("expected the stale %s with nil content, then the reference, got %v", GeneratedFile, files)
	}
	ref, err := reference.Read(bytes.NewReader(files[1].Content))
	if err != nil {
		t.Fatal(err)
	}
	if typ, ok := ref.Types, len(ref.Types) == 1; !ok || typ[0].Name != "thing" || len(typ[0].Members) != 0 {
		t.Fatalf("the members-less type must still be in the reference, got %+v", ref.Types)
	}
	if err := genfile.Write(files[:1]); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("Write must delete the stale %s: %v", GeneratedFile, err)
	}
}

// TestClassifyLine: the example convention — `// =>` is a result check, `// Error:` a raise check, any other comment
// prose — with the comment found outside string, character and raw-string literals.
func TestClassifyLine(t *testing.T) {
	cases := []struct {
		line string
		want LineCheck
	}{
		{`x := 1`, LineCheck{Kind: LineCode, Code: "x := 1"}},
		{`[1].len()   // => 1`, LineCheck{Kind: LineResult, Code: "[1].len()", Comment: "=> 1", Render: "1"}},
		{`[1].push([2])   // => [1, [2]] — the pair stays a pair`, LineCheck{Kind: LineResult, Code: "[1].push([2])", Comment: "=> [1, [2]] — the pair stays a pair", Render: "[1, [2]]"}},
		{`"a // b" + "c"   // => "a // bc"`, LineCheck{Kind: LineResult, Code: `"a // b" + "c"`, Comment: `=> "a // bc"`, Render: `"a // bc"`}},
		{"`//` + '/'   // => \"///\"", LineCheck{Kind: LineResult, Code: "`//` + '/'", Comment: `=> "///"`, Render: `"///"`}},
		{`f"{d["k"]}//"   // => "v//"`, LineCheck{Kind: LineResult, Code: `f"{d["k"]}//"`, Comment: `=> "v//"`, Render: `"v//"`}},
		{`1 / 0   // Error: division_by_zero`, LineCheck{Kind: LineError, Code: "1 / 0", Comment: "Error: division_by_zero", ErrKind: "division_by_zero"}},
		{`[].first()   // Error: invalid_value: (first) empty sequence`, LineCheck{Kind: LineError, Code: "[].first()", Comment: "Error: invalid_value: (first) empty sequence", ErrKind: "invalid_value"}},
		{`"x".int()   // Error: conversion — the parse`, LineCheck{Kind: LineError, Code: `"x".int()`, Comment: "Error: conversion — the parse", ErrKind: "conversion"}},
		{`f(x)   // the callback runs once`, LineCheck{Kind: LineCode, Code: "f(x)", Comment: "the callback runs once"}},
		{`// a whole-line comment`, LineCheck{Kind: LineCode, Code: "", Comment: "a whole-line comment"}},
		{`x   // raises — prose, not the convention`, LineCheck{Kind: LineCode, Code: "x", Comment: "raises — prose, not the convention"}},
	}
	for _, c := range cases {
		if got := ClassifyLine(c.line); got != c.want {
			t.Errorf("ClassifyLine(%q)\n got  %+v\n want %+v", c.line, got, c.want)
		}
	}
}

// TestFenceInfo: the `norun` marker and its reason; a plain fence and another language.
func TestFenceInfo(t *testing.T) {
	cases := []struct {
		line, lang, reason string
		noRun              bool
	}{
		{"```go", "go", "", false},
		{"```go norun", "go", "", true},
		{"```go norun — a signature summary", "go", "a signature summary", true},
		{"```go norun a fragment", "go", "a fragment", true},
		{"````md", "md", "", false},
		{"```", "", "", false},
	}
	for _, c := range cases {
		lang, noRun, reason := fenceInfo(c.line)
		if lang != c.lang || noRun != c.noRun || reason != c.reason {
			t.Errorf("fenceInfo(%q) = %q, %v, %q; want %q, %v, %q", c.line, lang, noRun, reason, c.lang, c.noRun, c.reason)
		}
		if got := pageFence(c.line); c.noRun && got != "```go" || !c.noRun && got != c.line {
			t.Errorf("pageFence(%q) = %q", c.line, got)
		}
	}
}

// TestReferenceRun: the reference file is generated only by a run over the whole module (a partial run says so),
// -reference names another path or ("") none, and the docs field is the -docs-url value: none without it, whatever
// the file on disk holds.
func TestReferenceRun(t *testing.T) {
	root := repoRoot(t)
	run := func(refPath *string, docsURL string, patterns ...string) (string, string, error) {
		var stdout, stderr strings.Builder
		err := Run(RunConfig{Dir: root, Check: true, Reference: refPath, DocsURL: docsURL, Patterns: patterns, Stdout: &stdout, Stderr: &stderr})
		return stdout.String(), stderr.String(), err
	}

	if _, stderr, err := run(nil, "", "./core"); err != nil || !strings.Contains(stderr, "kavun_reference.json not generated: the packages do not cover the whole module") {
		t.Fatalf("a partial run: %v, note %q", err, stderr)
	}

	// another path: missing there, so -check reports it
	other := filepath.Join(t.TempDir(), "ref.json")
	if stdout, _, err := run(&other, "", "./..."); err == nil || !strings.Contains(stdout, "ref.json (on disk)") {
		t.Fatalf("a missing reference must be reported, got %v\n%s", err, stdout)
	}
	// none: nothing reported
	none := ""
	if stdout, stderr, err := run(&none, "", "./..."); err != nil || stderr != "" {
		t.Fatalf("-reference \"\": %v\n%s%s", err, stdout, stderr)
	}

	// a file with a docs field: in sync with the same -docs-url only
	ref, err := reference.ReadFile(filepath.Join(root, reference.FileName))
	if err != nil {
		t.Fatal(err)
	}
	ref.Docs = "https://example.com/docs"
	var buf bytes.Buffer
	if err := ref.Write(&buf); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if stdout, _, err := run(&other, ref.Docs, "./..."); err != nil {
		t.Fatalf("the same -docs-url: %v\n%s", err, stdout)
	}
	if stdout, _, err := run(&other, "", "./..."); err == nil || !strings.Contains(stdout, `-  "docs": "https://example.com/docs",`) {
		t.Fatalf("without -docs-url the docs field must go: %v\n%s", err, stdout)
	}
	if stdout, _, err := run(&other, "https://example.com/other", "./..."); err == nil || !strings.Contains(stdout, `+  "docs": "https://example.com/other",`) {
		t.Fatalf("-docs-url must replace the docs field: %v\n%s", err, stdout)
	}
}

// TestExplicitReferenceKept: with nothing declared, the file at the default location is deleted, a file named by
// -reference is left alone.
func TestExplicitReferenceKept(t *testing.T) {
	ctx := loadContext(t)
	dir := filepath.Join("testdata", "ok", "none")
	path := filepath.Join(t.TempDir(), "mine.json")
	for _, explicit := range []bool{false, true} {
		files, err := Generate(ctx, []string{dir}, Options{Root: dir, Reference: path, ReferenceExplicit: explicit})
		if err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(files, func(f File) bool { return f.Path == path })
		if explicit && i >= 0 || !explicit && (i < 0 || files[i].Content != nil) {
			t.Fatalf("explicit %v: %+v", explicit, files)
		}
	}
}

// TestDocBlocks: a doc body as the reference's blocks — the text between examples verbatim but for gofmt's
// code-block tab (renderBody), each ```go block an example with its code (fence lines, gofmt tab and edge blank lines
// removed), the source line of its first code line, and its norun marker — and Doc.Markdown giving back the body.
func TestDocBlocks(t *testing.T) {
	body := []string{
		"Intro.", "",
		"```go", "", "\tx := 1", "\tx   // => 1", "", "```", "",
		"```md", "", "\tnot an example", "", "```",
		"```go norun — a fragment", "f(x)", "```",
		"```go", "```",
		"Outro.",
	}
	pos := make([]token.Position, len(body))
	for i := range pos {
		pos[i] = token.Position{Filename: "x.go", Line: 100 + i}
	}
	got := docBlocks(Doc{Body: body, BodyPos: pos})
	want := reference.Doc{
		{Kind: reference.BlockText, Markdown: "Intro.\n"},
		{Kind: reference.BlockExample, Code: "x := 1\nx   // => 1", Line: 104},
		{Kind: reference.BlockText, Markdown: "\n```md\nnot an example\n```"},
		{Kind: reference.BlockExample, Code: "f(x)", Line: 115, NoRun: true, Reason: "a fragment"},
		{Kind: reference.BlockExample},
		{Kind: reference.BlockText, Markdown: "Outro."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
	if md, src := got.Markdown(), strings.Join(renderBody(body), "\n"); md != src {
		t.Fatalf("Markdown differs from the source body:\n%s", genfile.UnifiedDiff("source", "reference", src, md))
	}
	if got := docBlocks(Doc{}); got == nil || len(got) != 0 {
		t.Fatalf("an empty body: %#v", got)
	}
}

// TestReferenceMatchesSource generates the repository's reference in memory, reads it back, and checks every entry
// against the directive it came from: the summary, the position, the signature, and the body — Doc.Markdown() must
// be exactly the source's body (renderBody: verbatim but for gofmt's code-block tab; the summary and the body with
// their links in absolute form, linkOwner.absoluteDoc), and the example blocks must carry the source lines
// sourceExamples reads (a line counted in the rendered Markdown would be off after a gofmt-indented fence). The pages
// are rendered from the reference alone (kavun-docs), so TestGeneratedInSync cannot see what the reference loses;
// this test can.
func TestReferenceMatchesSource(t *testing.T) {
	ctx := loadContext(t)
	root := repoRoot(t)
	dirs, err := ResolvePatterns(root, []string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	dirs = SkipScratch(root, dirs)
	refPath := filepath.Join(t.TempDir(), reference.FileName)
	files, err := Generate(ctx, dirs, Options{Root: root, Reference: refPath, Origin: KavunModule})
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(files, func(f File) bool { return f.Path == refPath })
	if i < 0 || files[i].Content == nil {
		t.Fatal("no reference generated")
	}
	ref, err := reference.Read(bytes.NewReader(files[i].Content))
	if err != nil {
		t.Fatal(err)
	}
	set, err := reference.Merge(ref)
	if err != nil {
		t.Fatal(err)
	}

	n := 0
	check := func(what string, o *linkOwner, d Directive, summary string, doc reference.Doc, pos reference.Pos) {
		t.Helper()
		n++
		src, _ := o.absoluteDoc(d.Doc)
		if summary != src.Summary {
			t.Errorf("%s: summary %q, want %q", what, summary, src.Summary)
		}
		if pos.Line != d.Pos.Line || !strings.HasSuffix(filepath.ToSlash(d.Pos.Filename), "/"+pos.File) {
			t.Errorf("%s: pos %v, directive at %s", what, pos, where(d.Pos))
		}
		if got, want := doc.Markdown(), strings.Join(renderBody(src.Body), "\n"); got != want {
			t.Errorf("%s: Doc.Markdown() differs from the source body:\n%s", what, genfile.UnifiedDiff("source", "reference", want, got))
		}
		exs, blocks := sourceExamples(d), doc.Examples()
		if len(exs) != len(blocks) {
			t.Errorf("%s: %d example blocks, %d examples", what, len(blocks), len(exs))
			return
		}
		for i, ex := range exs {
			var lines []string
			first := 0
			for _, l := range ex.lines {
				if strings.TrimSpace(l.text) != "" && first == 0 {
					first = l.pos.Line
				}
				lines = append(lines, l.text)
			}
			b := blocks[i]
			if b.Line != first || b.Code != strings.Join(trimBlank(lines), "\n") || b.NoRun != ex.noRun || b.Reason != ex.reason {
				t.Errorf("%s: example %d is %+v, the source's starts at line %d", what, i, b, first)
			}
		}
	}
	callable := func(what string, o *linkOwner, d Directive, c *reference.Callable, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: not in the reference", what)
			return
		}
		if c.Pure != d.Pure || c.Result != d.Result || c.Signature.Template != d.Template || c.Signature.Arity != d.Sig.Arity || c.Signature.Variadic != d.Sig.Variadic || !reflect.DeepEqual(c.Signature.Readings, d.Sig.Readings) {
			t.Errorf("%s: %+v does not match its directive", what, c)
		}
		check(what, o, d, c.Summary, c.Doc, c.Pos)
	}
	for _, dir := range dirs {
		_, m, problems := generateDir(ctx, dir)
		if len(problems) > 0 {
			t.Fatal(problems.err())
		}
		if m == nil {
			continue
		}
		for _, tm := range m.types {
			typ, ok := set.Type(tm.decl.Name)
			if !ok {
				t.Errorf("type %s: not in the reference", tm.decl.Name)
				continue
			}
			o := newLinkOwner(tm.decl, tm.members)
			check("type "+typ.Name, o, tm.decl, typ.Summary, typ.Doc, typ.Pos)
			if len(typ.Members) != len(tm.members) {
				t.Errorf("type %s: %d members, %d directives", typ.Name, len(typ.Members), len(tm.members))
			}
			for _, d := range tm.members {
				c, ok := typ.Member(d.Name)
				callable(typ.Name+"."+d.Name, o, d, c, ok)
			}
		}
		if m.builtins != nil {
			o := newLinkOwner(*m.builtins, m.builtinFuncs)
			check("builtins", o, *m.builtins, "", ref.Builtins.Doc, ref.Builtins.Pos)
			if len(ref.Builtins.Functions) != len(m.builtinFuncs) {
				t.Errorf("builtins: %d functions, %d directives", len(ref.Builtins.Functions), len(m.builtinFuncs))
			}
			for _, d := range m.builtinFuncs {
				c, ok := set.Builtin(d.Name)
				callable("builtin "+d.Name, o, d, c, ok)
			}
		}
		if d := m.module; d != nil {
			mod, ok := set.Module(d.Name)
			if !ok || mod.Pure != d.Pure {
				t.Errorf("module %s: %+v", d.Name, mod)
				continue
			}
			o := newLinkOwner(*d, m.moduleFuncs).withConsts(m.moduleConsts)
			check("module "+d.Name, o, *d, mod.Summary, mod.Doc, mod.Pos)
			if len(mod.Functions) != len(m.moduleFuncs) {
				t.Errorf("module %s: %d functions, %d directives", d.Name, len(mod.Functions), len(m.moduleFuncs))
			}
			for _, f := range m.moduleFuncs {
				c, ok := mod.Function(f.Name)
				callable(d.Name+"."+f.Name, o, f, c, ok)
			}
			if len(mod.Constants) != len(m.moduleConsts) {
				t.Errorf("module %s: %d constants, %d directives", d.Name, len(mod.Constants), len(m.moduleConsts))
			}
			for _, k := range m.moduleConsts {
				c, ok := mod.Constant(k.Name)
				if !ok || c.Type != k.Result {
					t.Errorf("%s.%s: %+v does not match its directive", d.Name, k.Name, c)
					continue
				}
				check(d.Name+"."+k.Name, o, k, c.Summary, c.Doc, c.Pos)
			}
		}
	}
	if n < 1000 {
		t.Fatalf("only %d entries checked", n)
	}
}

// embedDir is the fixture embedder module: a Go module of its own (testdata/embed/go.mod, Kavun replaced by this
// repository), so the repository's ./... never reads it.
const embedDir = "testdata/embed"

// embedWorkspace makes the go commands a test runs work offline on the fixture embedder module: a temporary
// workspace of the fixture and this repository (whose go.mod and go.sum give Kavun's dependencies, so the fixture
// pins none), no proxy, no inherited flags. It skips the test when the fixture's go.mod is absent: a module zip drops
// a nested module.
func embedWorkspace(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(embedDir, "go.mod")); err != nil {
		t.Skipf("%s/go.mod is absent (a module zip drops nested modules): %v", embedDir, err)
	}
	fixture, err := filepath.Abs(embedDir)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	goLine := regexp.MustCompile(`(?m)^go \S+$`).Find(gomod) // the workspace's go version: the repository's
	work := filepath.Join(t.TempDir(), "go.work")
	if err := os.WriteFile(work, []byte(string(goLine)+"\n\nuse (\n\t"+fixture+"\n\t"+root+"\n)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", work)
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOFLAGS", "")
}

// sourceExample is one fenced `go` block of a directive's doc body as the source holds it: its content lines with
// their positions (gofmt's code-block tab removed, edge blank lines kept), its norun marker and reason.
type sourceExample struct {
	noRun  bool
	reason string
	lines  []sourceLine
}

type sourceLine struct {
	text string
	pos  token.Position
}

// sourceExamples reads the fenced go blocks of one directive's doc body straight from its lines — the reading
// TestReferenceMatchesSource holds the reference's example blocks against.
func sourceExamples(d Directive) []sourceExample {
	body, pos := d.Doc.Body, d.Doc.BodyPos
	var out []sourceExample
	for _, sp := range fenceSpans(body) {
		if sp.close < 0 {
			break
		}
		lang, noRun, reason := fenceInfo(body[sp.open])
		if lang != "go" {
			continue
		}
		ex := sourceExample{noRun: noRun, reason: reason}
		content := body[sp.open+1 : sp.close]
		tabbed := containsTabbed(content)
		for i, l := range content {
			if tabbed {
				l = strings.TrimPrefix(l, "\t")
			}
			ex.lines = append(ex.lines, sourceLine{text: l, pos: pos[sp.open+1+i]})
		}
		out = append(out, ex)
	}
	return out
}

// TestEmbedder runs the two tools on the fixture embedder module as an embedder runs them: kavun-gen over its ./...
// (member names declared by //kavun:names, used bare in their own package and qualified from another; a module with
// its own id=; Kavun's reference loaded, so a kavun:type/string link resolves), then kavun-docs with Kavun's
// reference as a -with (that link renders under Kavun's docs URL, and so do kavun-docs' own links into Kavun's
// guide). Its kavun_gen.go files, kavun_reference.json and docs/reference/ pages are the goldens (-update rewrites
// them). Last, the fixture module's own test builds the generated code and runs a script over it: the names bound
// before the type's init function registers its table, the module registered under its id.
func TestEmbedder(t *testing.T) {
	embedWorkspace(t)
	var stdout, stderr strings.Builder
	err := Run(RunConfig{Dir: embedDir, Check: !*update, Patterns: []string{"./..."}, Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		t.Fatalf("kavun-gen: %v (run with -update after checking)\n%s%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() > 0 {
		t.Fatalf("unexpected note: %s", stderr.String())
	}
	ref, err := reference.ReadFile(filepath.Join(embedDir, reference.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if ref.Origin != "example.com/kavunembed" || len(ref.Types) != 1 || len(ref.Modules) != 1 || len(ref.Modules[0].Constants) != 1 {
		t.Fatalf("unexpected reference: %+v", ref)
	}

	kavunRef := filepath.Join(repoRoot(t), reference.FileName)
	stdout.Reset()
	err = kavundocs.Run(kavundocs.Config{Out: filepath.Join(embedDir, "docs", "reference"), Check: !*update,
		Render: []string{filepath.Join(embedDir, reference.FileName)}, With: []kavundocs.With{{Path: kavunRef}}, Stdout: &stdout})
	if err != nil {
		t.Fatalf("kavun-docs: %v (run with -update after checking)\n%s", err, stdout.String())
	}
	kavun, err := reference.ReadFile(kavunRef)
	if err != nil {
		t.Fatal(err)
	}
	for page, want := range map[string]string{
		"types/gadget.md":    "[`string`](" + kavun.Docs + "/reference/types/string.md)",
		"modules/mymod.md":   "[gadget](../types/gadget.md)",
		"function_matrix.md": "(" + kavun.Docs + "/types.md#member-functions-are-values)",
	} {
		got, err := os.ReadFile(filepath.Join(embedDir, "docs", "reference", filepath.FromSlash(page)))
		if err != nil || !strings.Contains(string(got), want) {
			t.Errorf("%s lacks %q: %v\n%s", page, want, err, got)
		}
	}

	if testing.Short() {
		t.Skip("-short: the fixture module is not built")
	}
	cmd := exec.Command("go", "test", "-count=1", "./...")
	cmd.Dir = embedDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go test in %s: %v\n%s", embedDir, err, out)
	}
}

// TestEmbedderWith: a -with reference joins the references the links resolve against, and one declaring a name the
// run declares too is a conflict.
func TestEmbedderWith(t *testing.T) {
	embedWorkspace(t)
	clash := &reference.Reference{Schema: reference.SchemaVersion, Origin: "example.com/clash",
		Types: []reference.Type{{Name: "gadget", Summary: "Another gadget.", Doc: reference.Doc{}, Members: []reference.Callable{}}}}
	var buf bytes.Buffer
	if err := clash.Write(&buf); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), reference.FileName)
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr strings.Builder
	err := Run(RunConfig{Dir: embedDir, Check: true, With: []string{path}, Patterns: []string{"./..."}, Stdout: &stdout, Stderr: &stderr})
	want := filepath.ToSlash(filepath.Join("testdata", "embed", "gadget", "gadget.go")) + `:23: type "gadget" is declared by example.com/clash too: a type name is declared once across the references`
	if err == nil || !strings.HasSuffix(err.Error(), want) {
		t.Fatalf("expected a merge conflict, got %v", err)
	}
	err = Run(RunConfig{Dir: embedDir, Check: true, With: []string{filepath.Join(t.TempDir(), "missing.json")}, Patterns: []string{"./..."}, Stdout: &stdout, Stderr: &stderr})
	if err == nil || !strings.Contains(err.Error(), "-with:") {
		t.Fatalf("expected a missing -with file, got %v", err)
	}
}

// TestLoadedReferences: a run's links resolve against the references it loads (Kavun's, for an embedder) as well as
// its own declarations, entries and headings included; a type the run declares that a loaded reference declares
// too is a conflict.
func TestLoadedReferences(t *testing.T) {
	ctx := loadContext(t)
	kavun, err := reference.ReadFile(filepath.Join(repoRoot(t), reference.FileName))
	if err != nil {
		t.Fatal(err)
	}
	gen := func(typeName, doc string) error {
		dir := t.TempDir()
		src := "package ext\n\n// A type.\n//\n// " + doc + "\n//\n//kavun:type " + typeName + "\n"
		if err := os.WriteFile(filepath.Join(dir, "gizmo.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Generate(ctx, []string{dir}, Options{Loaded: []*reference.Reference{kavun}})
		return err
	}
	if err := gen("gizmo", "See [s](kavun:type/string), [p](kavun:type/string/pad_start), [r](kavun:builtin/range), [pi](kavun:module/math/pi)."); err != nil {
		t.Fatalf("links into the loaded reference: %v", err)
	}
	for doc, want := range map[string]string{
		"See [s](kavun:type/string/nope).": "kavun:type/string/nope: type string has no member nope",
		"See [s](kavun:module/math/nope).": "kavun:module/math/nope: module math has no function or constant nope",
	} {
		if err := gen("gizmo", doc); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: got %v, want %q", doc, err, want)
		}
	}
	if err := gen("string", "A string again."); err == nil || !strings.Contains(err.Error(), `gizmo.go:7: type "string" is declared by github.com/jokruger/kavun too (core/string.go:`) {
		t.Errorf("a type Kavun declares: got %v", err)
	}
}

// TestNamesAcrossPackages: a member name declared by one package's //kavun:names is used from another package of the
// run by its key, which must then be exported; the run's names are collected before any table is checked.
func TestNamesAcrossPackages(t *testing.T) {
	ctx := loadContext(t)
	write := func(files map[string]string) string {
		dir := t.TempDir()
		for name, src := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}
	names := write(map[string]string{"names.go": "package names\n\nconst (\n\tMemberDouble member.ID = 2048\n\tmemberSecret member.ID = 2049\n)\n\n" +
		"//kavun:names\nvar memberNames = map[member.ID]string{\n\tMemberDouble: \"double\",\n\tmemberSecret: \"secret\",\n}\n"})
	user := func(name string) string {
		return write(map[string]string{
			"widget.go":         widgetRoot,
			"widget_members.go": membersHead + "// S.\n//\n//kavun:member " + name + " pure () -> int\nfunc wX" + memberFunc,
		})
	}
	// the user package sorts first: the names are still known
	_, err := Generate(ctx, []string{user("secret"), names}, Options{})
	if err == nil || !strings.Contains(err.Error(), `"secret" is declared by the //kavun:names of package names`) || !strings.Contains(err.Error(), "under the unexported memberSecret: export the key") {
		t.Fatalf("an unexported key from another package: %v", err)
	}
	if _, err := Generate(ctx, []string{user("double")}, Options{}); err == nil || !strings.Contains(err.Error(), `"double" is neither a builtin member name`) {
		t.Fatalf("a name no package of the run declares: %v", err)
	}
	// a name declared by two packages of the run
	again := write(map[string]string{"more.go": "package more\n\nconst MemberTwice member.ID = 2050\n\n//kavun:names\nvar memberNames = map[member.ID]string{\n\tMemberTwice: \"double\",\n}\n"})
	if _, err := Generate(ctx, []string{names, again}, Options{}); err == nil || !strings.Contains(err.Error(), `more.go:7: //kavun:names: "double" is declared twice`) {
		t.Fatalf("a name declared by two packages: %v", err)
	}
	// a package declaring names only gets the binding, nothing else
	files, err := Generate(ctx, []string{names}, Options{})
	if err != nil || len(files) != 1 || !strings.Contains(got(files), "member.Define(MemberDouble, \"double\")") || strings.Contains(got(files), "Methods") {
		t.Fatalf("names only: %v\n%s", err, got(files))
	}
}

// TestNoFollowOn: a refused //kavun:module or //kavun:names entry is reported once — its functions, constants and
// members using its names are not reported again as lacking a module or a name.
func TestNoFollowOn(t *testing.T) {
	ctx := loadContext(t)
	for name, c := range map[string]struct {
		files []sourceFile
		want  string
	}{
		"module id": {fixture("p.go", "package path\n\n// Paths.\n//\n//kavun:module path id=value.ModuleNope pure\n\n"+joinFn+"\n// S.\n//\n//kavun:const pi -> float\nvar constPi = 1\n"),
			`fx/p.go:5: module "path": id=value.ModuleNope is not a constant of core/value`},
		"malformed module": {fixture("p.go", "package path\n\n// Paths.\n//\n//kavun:module path pure\n\n"+joinFn),
			"fx/p.go:5: malformed directive: want `//kavun:module <name> id=<go ident> pure|impure`"},
		"names key": {fixture("n.go", "package core\n\nvar MemberTriple = member.ID(2048)\n\n//kavun:names\nvar memberNames = map[member.ID]string{\n\tMemberTriple: \"triple\",\n}\n",
			"widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member triple pure () -> int\nfunc wTriple"+memberFunc),
			`fx/n.go:7: //kavun:names: the key MemberTriple of "triple" is not a constant of the package`},
		"names entry": {fixture("n.go", "package core\n\nconst MemberTriple member.ID = 2048\n\n//kavun:names\nvar memberNames = map[member.ID]string{\n\tMemberTriple: \"triple\",\n\tMemberTriple + 1: \"quad\",\n}\n",
			"widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member triple pure () -> int\nfunc wTriple"+memberFunc),
			"fx/n.go:8: //kavun:names memberNames: the key MemberTriple + 1 is not an identifier of the package"},
	} {
		_, _, errs := generateSources(ctx, "fx", c.files)
		if len(errs) != 1 || !strings.Contains(errs[0], c.want) {
			t.Errorf("%s: want the one error %q, got %q", name, c.want, errs)
		}
	}
}

// TestNamesCycle: two packages whose tables use each other's //kavun:names would make their generated files import
// each other; the run refuses it at a directive using the other's name, naming the cycle.
func TestNamesCycle(t *testing.T) {
	ctx := loadContext(t)
	pkg := func(name, key, member, uses string) string {
		dir := filepath.Join(t.TempDir(), name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		files := map[string]string{
			"names.go":          "package " + name + "\n\nconst " + key + " member.ID = 2048\n\n//kavun:names\nvar memberNames = map[member.ID]string{\n\t" + key + ": \"" + member + "\",\n}\n",
			"widget.go":         "package " + name + "\n\n// A " + name + ".\n//\n//kavun:type " + name + "\n",
			"widget_members.go": "package " + name + "\n\n// S.\n//\n//kavun:member " + uses + " pure () -> int\nfunc wX" + memberFunc,
		}
		for f, src := range files {
			if err := os.WriteFile(filepath.Join(dir, f), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}
	a, b := pkg("alpha", "MemberDouble", "double", "triple"), pkg("beta", "MemberTriple", "triple", "double")
	_, err := Generate(ctx, []string{b, a}, Options{})
	want := filepath.ToSlash(filepath.Join(a, "widget_members.go")) + ":5: package alpha uses the //kavun:names of package beta, and the packages' names are used in a cycle (alpha → beta → alpha)"
	if err == nil || strings.Count(err.Error(), "in a cycle") != 1 || !strings.Contains(err.Error(), want) {
		t.Fatalf("want %q once, got %v", want, err)
	}
}
