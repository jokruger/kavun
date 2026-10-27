package kavungen

import (
	"bytes"
	"flag"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

var update = flag.Bool("update", false, "rewrite the golden files under testdata/ok")

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

// TestGeneratedInSync runs kavun-gen's -check over the whole repository, as `go generate ./...` would generate it:
// every committed kavun_gen.go and every page under docs/reference/ must be what the generator produces now (a
// page there that nothing generates is stale). It first pins the emitter's line width: no generated kavun_gen.go
// line is wider than maxLine.
func TestGeneratedInSync(t *testing.T) {
	// every kavun_gen.go the generator produces for the repository stays within maxLine
	root := repoRoot(t)
	dirs, err := ResolvePatterns(root, []string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	files, err := Generate(loadContext(t), SkipScratch(root, dirs), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, f := range files {
		if filepath.Base(f.Path) == GeneratedFile {
			checkLineWidth(t, f.Path, f.Content)
			n++
		}
	}
	if n == 0 {
		t.Fatal("no kavun_gen.go generated")
	}

	var stdout, stderr strings.Builder
	err = Run(RunConfig{Dir: repoRoot(t), Check: true, Patterns: []string{"./..."}, Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		t.Fatalf("%v — run `go generate ./...`:\n%s%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() > 0 {
		t.Fatalf("unexpected note: %s", stderr.String())
	}
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
	if err := Run(RunConfig{Dir: root, Check: true, Patterns: []string{"./..."}, Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("the scratch package was read: %v\n%s", err, stdout.String())
	}
	if got := SkipScratch(root, []string{scratch, filepath.Join(root, "core"), filepath.Join(root, "tmpx")}); len(got) != 2 || slices.Contains(got, scratch) {
		t.Fatalf("SkipScratch: %v", got)
	}
}

// TestPartialDocs: with -docs and patterns that do not cover the module, the pages of the packages read are
// generated, and nothing else under the docs root is reported (or deleted) as stale; over ./... it is.
func TestPartialDocs(t *testing.T) {
	root := repoRoot(t)
	docsDir := t.TempDir()
	// every page ./core generates (one per type moved so far), plus a page no package generates
	pages := map[string][]byte{"types/other.md": []byte("another package's page\n")}
	typePages, err := filepath.Glob(filepath.Join(root, "docs", "reference", "types", "*.md"))
	if err != nil || len(typePages) == 0 {
		t.Fatalf("no type pages: %v", err)
	}
	for _, p := range typePages {
		content, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		pages["types/"+filepath.Base(p)] = content
	}
	for rel, content := range pages {
		p := filepath.Join(docsDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var stdout, stderr strings.Builder
	if err := Run(RunConfig{Dir: root, Check: true, Docs: &docsDir, Patterns: []string{"./core"}, Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("a partial run reported: %v\n%s", err, stdout.String())
	}
	// the matrix needs every column, so a partial run neither writes nor checks it (were it produced, its absence
	// from docsDir would have been reported above)
	if !strings.Contains(stderr.String(), "no page is reported or deleted as stale") {
		t.Fatalf("expected the note, got %q", stderr.String())
	}

	stdout.Reset()
	err = Run(RunConfig{Dir: root, Check: true, Docs: &docsDir, Patterns: []string{"./..."}, Stdout: &stdout, Stderr: io.Discard})
	if err == nil || !strings.Contains(stdout.String(), "other.md (on disk)") {
		t.Fatalf("a whole-module run should report the stale page, got %v\n%s", err, stdout.String())
	}
}

// TestGolden generates each fixture package under testdata/ok and compares it with its kavun_gen.go.golden and its
// Markdown pages with reference/<page>.golden; a fixture with no golden file must generate nothing, and every page
// golden must be generated.
func TestGolden(t *testing.T) {
	ctx := loadContext(t)
	for _, name := range []string{"core", "ext", "builtins", "module", "module_plain", "none", "pageonly"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", "ok", name)
			docsDir := t.TempDir()
			files, err := Generate(ctx, []string{dir}, Options{DocsDir: docsDir, Root: dir})
			if err != nil {
				t.Fatal(err)
			}
			goFiles, pages := splitGenerated(t, files, docsDir)

			golden := filepath.Join(dir, GeneratedFile+".golden")
			if content := goFiles[filepath.Join(dir, GeneratedFile)]; content != nil {
				checkLineWidth(t, golden, content)
			}
			checkGolden(t, golden, goFiles[filepath.Join(dir, GeneratedFile)], len(goFiles))

			goldenDocs := filepath.Join(dir, "reference")
			for rel, content := range pages {
				checkGolden(t, filepath.Join(goldenDocs, filepath.FromSlash(rel))+".golden", content, 1)
			}
			if !*update {
				_ = filepath.WalkDir(goldenDocs, func(p string, e os.DirEntry, err error) error {
					if err == nil && !e.IsDir() {
						rel, _ := filepath.Rel(goldenDocs, strings.TrimSuffix(p, ".golden"))
						if _, ok := pages[filepath.ToSlash(rel)]; !ok {
							t.Errorf("%s: the page is not generated", p)
						}
					}
					return nil
				})
			}

			// the core fixture's zz_members.go is excluded by //go:build ignore: its member must not appear
			for _, content := range goFiles {
				if strings.Contains(string(content), "widgetCopy") {
					t.Fatal("a member from a build-excluded file was generated")
				}
			}

			// deterministic: a second run gives the same bytes
			again, err := Generate(ctx, []string{dir}, Options{DocsDir: docsDir, Root: dir})
			if err != nil || !slices.EqualFunc(again, files, func(a, b File) bool { return a.Path == b.Path && string(a.Content) == string(b.Content) }) {
				t.Fatal("a second generation differs from the first")
			}
		})
	}
}

// splitGenerated separates the kavun_gen.go files (by path) from the pages (by slash path below docsDir).
func splitGenerated(t *testing.T, files []File, docsDir string) (goFiles map[string][]byte, pages map[string][]byte) {
	t.Helper()
	goFiles, pages = map[string][]byte{}, map[string][]byte{}
	for _, f := range files {
		if rel, err := filepath.Rel(docsDir, f.Path); err == nil && !strings.HasPrefix(rel, "..") {
			if f.Content == nil {
				t.Fatalf("unexpected stale page %s", rel)
			}
			pages[filepath.ToSlash(rel)] = f.Content
			continue
		}
		goFiles[f.Path] = f.Content
	}
	return goFiles, pages
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
		t.Fatalf("output differs from %s (run with -update after checking):\n%s", golden, unifiedDiff("golden", "generated", string(want), string(content)))
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
	if diffs, _ := Check(files); len(diffs) != 1 {
		t.Fatalf("Check should report the stale file, got %v", diffs)
	}
	if err := Write(files); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, GeneratedFile)); !os.IsNotExist(err) {
		t.Fatal("the stale file still exists")
	}
	if diffs, _ := Check(files); len(diffs) != 0 {
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
	if diffs, _ := Check(files); len(diffs) != 1 || !strings.Contains(diffs[0], "+package path") {
		t.Fatalf("Check before Write should report the missing file, got %v", diffs)
	}
	if err := Write(files); err != nil {
		t.Fatal(err)
	}
	if diffs, _ := Check(files); len(diffs) != 0 {
		t.Fatalf("Check after Write should be clean, got %v", diffs)
	}
	target := filepath.Join(dir, GeneratedFile)
	if err := os.WriteFile(target, append(slices.Clone(files[0].Content), "// edited\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	if diffs, _ := Check(files); len(diffs) != 1 || !strings.Contains(diffs[0], "-// edited") {
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
	moduleHead     = "package path\n\n// Paths.\n//\n//kavun:module path pure\n\n"
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
			"fx/widget.go:5: a `## Members` heading in the //kavun:type block: the generator adds that section"},
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
			"fx/widget_members.go:7: \"no_such_member\" is not a member name member.Lookup knows"},
		{"unknown fn name", fixture("p.go", moduleHead+"// J.\n//\n//kavun:fn no_such_member slot=0 pure () -> int\nfunc pathJoin"+nativeFunc),
			"fx/p.go:9: \"no_such_member\" is not a member name member.Lookup knows"},
		{"duplicate member", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+lenMember+"\n// Again.\n//\n//kavun:member len pure () -> int\nfunc wLen2"+memberFunc),
			"fx/widget_members.go:12: member \"len\" is declared twice for type \"widget\" (first at fx/widget_members.go:7)"},
		{"pure _in_place member", fixture("widget.go", widgetRoot, widgetMembersF, membersHead+"// S.\n//\n//kavun:member sort_in_place pure () -> int\nfunc wSort"+memberFunc),
			"fx/widget_members.go:7: //kavun:member sort_in_place: an _in_place member mutates its receiver and must be impure"},

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
		{"second module", fixture("p.go", moduleHead+"// Again.\n//\n//kavun:module path pure\n"),
			"fx/p.go:9: a second //kavun:module (the first is at fx/p.go:5): one module per package"},
		{"module without a type id", fixture("p.go", "package nope\n\n// N.\n//\n//kavun:module nope pure\n"),
			"fx/p.go:5: module \"nope\" has no type id constant value.ModuleNope"},
		{"duplicate fn slot", fixture("p.go", moduleHead+joinFn+"\n// B.\n//\n//kavun:fn base slot=0 pure (p) -> int\nfunc pathBase"+nativeFunc),
			"fx/p.go:14: slot 0 is used twice in module path (first by join at fx/p.go:9)"},
		{"module extras of the wrong shape", fixture("p.go", moduleHead+joinFn+"\nfunc moduleExtras() {}\n"),
			"fx/p.go:12: func moduleExtras must be `func moduleExtras(m *core.ModuleBuilder)`"},

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
		{"module path pure", Directive{Kind: KindModule, Name: "path", Pure: true}},
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
			got.Slot != c.want.Slot || got.Result != c.want.Result || got.GoFunc != c.want.GoFunc {
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

func TestUnifiedDiff(t *testing.T) {
	got := unifiedDiff("a", "b", "1\n2\n3\n", "1\n3\n4\n")
	want := "--- a\n+++ b\n@@ -1,3 +1,3 @@\n 1\n-2\n 3\n+4\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
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
				t.Fatalf("gofmt changes the generated Go:\n%s", unifiedDiff("raw", "gofmt", goA, goB))
			}
			for rel := range mdA {
				if mdA[rel] != mdB[rel] {
					t.Fatalf("gofmt changes %s:\n%s", rel, unifiedDiff("raw", "gofmt", mdA[rel], mdB[rel]))
				}
			}
			if len(mdA) != len(mdB) || len(mdA) == 0 {
				t.Fatalf("expected the same pages, got %d and %d", len(mdA), len(mdB))
			}
		})
	}
}

// generateInMemory generates one package from sources: its Go file and its pages.
func generateInMemory(t *testing.T, ctx *Context, dir string, files []sourceFile) (string, map[string]string) {
	t.Helper()
	content, model, errs := generateSources(ctx, dir, files)
	if len(errs) > 0 {
		t.Fatal(errs.err())
	}
	pages, errs := emitDocs([]*packageModel{model})
	if len(errs) > 0 {
		t.Fatal(errs.err())
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

// TestStdlibIndex: the module index has one row per module in name order, with the grant spelled out.
func TestStdlibIndex(t *testing.T) {
	docsDir := t.TempDir()
	root := filepath.Join("testdata", "ok")
	files, err := Generate(loadContext(t), []string{filepath.Join(root, "module_plain"), filepath.Join(root, "module")}, Options{DocsDir: docsDir, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	_, pages := splitGenerated(t, files, docsDir)
	want := "<!-- Code generated by kavun-gen from module/path.go, module_plain/os.go; DO NOT EDIT. -->\n" +
		"# Standard library\n\n| module | grant | summary |\n| --- | --- | --- |\n" +
		"| [os](stdlib/os.md) | needs a grant | The operating system: a world module. |\n" +
		"| [path](stdlib/path.md) | importable by default | Path text: joining and splitting file paths. |\n"
	if got := string(pages["stdlib.md"]); got != want {
		t.Fatalf("stdlib.md:\n%s", unifiedDiff("want", "got", want, got))
	}
	// stdlib.md, the two module pages, the module-owned type's page (types/path_segment.md) and the matrix
	if len(pages) != 5 || pages["types/path_segment.md"] == nil || pages[matrixPage] == nil {
		t.Fatalf("expected stdlib.md, two module pages, types/path_segment.md and %s, got %d pages", matrixPage, len(pages))
	}
}

// TestStaleDocs: a file under the docs root that no page produces is reported by Check and deleted by Write; a page
// two packages would both write is refused.
func TestStaleDocs(t *testing.T) {
	ctx := loadContext(t)
	docsDir := t.TempDir()
	stale := filepath.Join(docsDir, "types", "gone.md")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("testdata", "ok", "ext")
	files, err := Generate(ctx, []string{dir}, Options{DocsDir: docsDir, Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	// ext's kavun_gen.go is missing on disk (its golden is not the file), its page is new, gone.md is stale
	diffs, err := Check(files)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(diffs, func(d string) bool { return strings.Contains(d, "gone.md (on disk)") && strings.Contains(d, "-old") }) {
		t.Fatalf("Check should report the stale page, got %v", diffs)
	}
	var docs []File
	for _, f := range files {
		if strings.HasPrefix(f.Path, docsDir) {
			docs = append(docs, f)
		}
	}
	if err := Write(docs); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("the stale page still exists")
	}
	if _, err := os.Stat(filepath.Join(docsDir, "types", "gadget.md")); err != nil {
		t.Fatalf("the page was not written: %v", err)
	}
	if diffs, _ := Check(docs); len(diffs) != 0 {
		t.Fatalf("Check after Write should be clean, got %v", diffs)
	}

	// KeepStale: a file nothing produces is neither reported nor deleted
	if err := os.WriteFile(stale, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err = Generate(ctx, []string{dir}, Options{DocsDir: docsDir, Root: dir, KeepStale: true})
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(files, func(f File) bool { return f.Content == nil }) {
		t.Fatal("KeepStale still reports a stale page")
	}

	// the same type from two packages
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
	_, err = Generate(ctx, []string{dir, twin}, Options{DocsDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "types/gadget.md would be generated twice") {
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
// kavun_gen.go from an earlier run is answered with nil content and Write deletes it; the page is still emitted.
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
	docsDir := t.TempDir()
	files, err := Generate(ctx, []string{dir}, Options{DocsDir: docsDir, Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	goFiles, pages := splitGenerated(t, files, docsDir)
	if content, ok := goFiles[stale]; !ok || content != nil {
		t.Fatalf("the stale %s must be answered with nil content, got %v", GeneratedFile, goFiles)
	}
	if _, ok := pages["types/thing.md"]; !ok {
		t.Fatalf("the members-less type's page must still be generated, got %v", pages)
	}
	if err := Write(files); err != nil {
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

// TestExamples: Examples answers every fenced go block of the doc blocks, with the position of each line, the
// gofmt tab removed, and the norun marker read; a fence of another language is not an example.
func TestExamples(t *testing.T) {
	dir := t.TempDir()
	src := membersHead + "// Answer the length.\n//\n// ```go\n// [1].len()   // => 1\n// ```\n//\n// ```go norun — a fragment\n//\tlen(x)\n// ```\n//\n// ```md\n// text\n// ```\n//\n//kavun:member len pure () -> int\nfunc wLen" + memberFunc
	for name, content := range map[string]string{"widget.go": widgetRoot, widgetMembersF: src} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Examples([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 examples, got %+v", got)
	}
	run, skip := got[0], got[1]
	if run.NoRun || run.Owner != "member len" || run.Pos.Line != 7 || len(run.Lines) != 1 || run.Lines[0].Text != "[1].len()   // => 1" || run.Lines[0].Pos.Line != 8 {
		t.Fatalf("unexpected first example %+v", run)
	}
	if !skip.NoRun || skip.Reason != "a fragment" || len(skip.Lines) != 1 || skip.Lines[0].Text != "len(x)" || skip.Lines[0].Pos.Line != 12 {
		t.Fatalf("unexpected second example %+v", skip)
	}
}
