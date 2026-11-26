package reference_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jokruger/kavun/reference"
	"github.com/jokruger/kavun/signature"
)

// callable builds a callable from a template, as the generator does.
func callable(t *testing.T, name, template string, pure bool, doc ...reference.Block) reference.Callable {
	t.Helper()
	sh, err := signature.Parse(template)
	if err != nil {
		t.Fatal(err)
	}
	return reference.Callable{
		Name: name, Pure: pure, Signature: reference.SignatureOf(sh), Result: "value",
		Summary: "Does " + name + ".", Doc: append(reference.Doc{}, doc...), Pos: reference.Pos{File: "x/" + name + ".go", Line: 7},
	}
}

// sample answers a reference in normal form (sorted, no nil list) with every kind of entry and block.
func sample(t *testing.T, origin string) *reference.Reference {
	t.Helper()
	text := reference.Block{Kind: reference.BlockText, Markdown: "See [x](kavun:type/array/splice) & <b>.\n"}
	ex := reference.Block{Kind: reference.BlockExample, Code: "[1].len()   // => 1", Line: 12}
	skip := reference.Block{Kind: reference.BlockExample, Code: "f(x)", Line: 20, NoRun: true, Reason: "a fragment"}
	return &reference.Reference{
		Schema: reference.SchemaVersion, Origin: origin, CoreVersion: "1.2", Docs: "https://example.com/docs",
		Types: []reference.Type{
			{Name: "array", Summary: "An array.", Doc: reference.Doc{text, ex}, Pos: reference.Pos{File: "core/array.go", Line: 3},
				Members: []reference.Callable{
					callable(t, "map", "(f)", true),
					callable(t, "map_in_place", "(f)", false),
					callable(t, "splice", "([start[, delete_count]]) | (start, delete_count, ...items)", true, text, ex, skip),
				}},
			{Name: "fs.file", Module: "fs", Summary: "A file.", Doc: reference.Doc{}, Pos: reference.Pos{File: "stdlib/fs/file.go", Line: 9},
				Members: []reference.Callable{}},
		},
		Builtins: &reference.Builtins{Doc: reference.Doc{text}, Pos: reference.Pos{File: "vm/builtins.go", Line: 1},
			Functions: []reference.Callable{callable(t, "len", "(x)", true), callable(t, "range", "(stop) | (start, stop[, step])", true)}},
		Modules: []reference.Module{
			{Name: "fs", Pure: false, Summary: "Files.", Doc: reference.Doc{}, Pos: reference.Pos{File: "stdlib/fs/fs.go", Line: 2},
				Functions: []reference.Callable{callable(t, "close", "()", false), callable(t, "open", "(path)", false)},
				Constants: []reference.Constant{{Name: "sep", Type: "string", Summary: "The separator.", Doc: reference.Doc{}, Pos: reference.Pos{File: "stdlib/fs/fs.go", Line: 30}}}},
		},
	}
}

func write(t *testing.T, r *reference.Reference) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := r.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func read(t *testing.T, data []byte) *reference.Reference {
	t.Helper()
	r, err := reference.Read(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestRoundTrip: Read(Write(x)) == x for a reference in normal form, and Write is deterministic.
func TestRoundTrip(t *testing.T) {
	x := sample(t, "example.com/a")
	data := write(t, x)
	if got := read(t, data); !reflect.DeepEqual(got, x) {
		t.Fatalf("Read(Write(x)) != x\n got  %+v\n want %+v", got, x)
	}
	if again := write(t, read(t, data)); !bytes.Equal(again, data) {
		t.Fatal("writing what was read gives other bytes")
	}
}

// TestWriteForm: sorted by name at every level (doc blocks and readings keep their order), nil lists as [],
// optional fields omitted when empty, `pure` always present, HTML characters literal, two-space indent, one
// trailing newline; the reference written is not changed.
func TestWriteForm(t *testing.T) {
	x := sample(t, "example.com/a")
	x.Docs = ""
	slices.Reverse(x.Types)
	slices.Reverse(x.Types[1].Members)
	slices.Reverse(x.Builtins.Functions)
	x.Types[0].Members = nil // fs.file
	before := sample(t, "example.com/a")
	before.Docs = ""
	slices.Reverse(before.Types)
	slices.Reverse(before.Types[1].Members)
	slices.Reverse(before.Builtins.Functions)
	before.Types[0].Members = nil

	data := write(t, x)
	if !reflect.DeepEqual(x, before) {
		t.Fatal("Write changed the reference")
	}
	s := string(data)
	for _, want := range []string{
		"{\n  \"schema\": 1,\n  \"origin\": \"example.com/a\",\n  \"core_version\": \"1.2\",\n  \"types\": [\n    {\n      \"name\": \"array\",",
		`"members": []`,
		`"module": "fs"`,
		`"pure": false`,
		`& <b>`,
		`"norun": true`,
		`"reason": "a fragment"`,
		`"params": []`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in\n%s", want, s)
		}
	}
	for _, absent := range []string{`"docs"`, `\u003c`, `\u0026`, `"line": 0`, `"norun": false`, `"code": ""`, "\t"} {
		if strings.Contains(s, absent) {
			t.Errorf("unexpected %q", absent)
		}
	}
	if !strings.HasSuffix(s, "}\n") || strings.HasSuffix(s, "\n\n") {
		t.Error("want exactly one trailing newline")
	}
	if strings.Index(s, `"name": "map"`) > strings.Index(s, `"name": "map_in_place"`) || strings.Index(s, `"name": "len"`) > strings.Index(s, `"name": "range"`) {
		t.Error("entries are not sorted by name")
	}
	if strings.Index(s, `"See [x]`) > strings.Index(s, `"[1].len()`) {
		t.Error("doc blocks changed order")
	}
	// the module-owned type has no module field on the others
	if strings.Count(s, `"module":`) != 1 {
		t.Errorf("module field count %d", strings.Count(s, `"module":`))
	}
}

// TestBlockJSON: a text block always carries markdown, "" included; an example omits it; both read back as written.
func TestBlockJSON(t *testing.T) {
	x := sample(t, "o")
	x.Types[1].Doc = reference.Doc{{Kind: reference.BlockText}, {Kind: reference.BlockExample, Code: "x", Line: 3}}
	data := write(t, x)
	want := "\"doc\": [\n        {\n          \"kind\": \"text\",\n          \"markdown\": \"\"\n        },\n" +
		"        {\n          \"kind\": \"example\",\n          \"code\": \"x\",\n          \"line\": 3\n        }\n      ]"
	if !strings.Contains(string(data), want) {
		t.Fatalf("missing %s in\n%s", want, data)
	}
	if got := read(t, data); !reflect.DeepEqual(got, x) {
		t.Fatal("Read(Write(x)) != x")
	}
}

// TestRead: unknown fields are ignored, missing lists read as empty, a greater or missing schema is refused.
func TestRead(t *testing.T) {
	r := read(t, []byte(`{"schema": 1, "origin": "o", "core_version": "1.2", "future": {"x": 1},
		"types": [{"name": "t", "color": "red"}]}`))
	if r.Origin != "o" || len(r.Types) != 1 || r.Types[0].Members == nil || r.Types[0].Doc == nil || r.Modules == nil || r.Builtins != nil {
		t.Fatalf("unexpected %+v", r)
	}
	for _, bad := range []string{
		`{"schema": 2, "origin": "o"}`,
		`{"origin": "o"}`,
		`{"schema": 1,`,
		`[]`,
	} {
		if _, err := reference.Read(strings.NewReader(bad)); err == nil {
			t.Errorf("Read(%s) succeeded", bad)
		}
	}
	if _, err := reference.Read(strings.NewReader(`{"schema": 2, "origin": "o"}`)); err == nil || !strings.Contains(err.Error(), "schema 2 is newer") {
		t.Errorf("unexpected error %v", err)
	}
}

// TestReadFile: the file round trip, and the error names the path.
func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, reference.FileName)
	x := sample(t, "example.com/a")
	if err := os.WriteFile(p, write(t, x), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := reference.ReadFile(p)
	if err != nil || !reflect.DeepEqual(got, x) {
		t.Fatalf("ReadFile: %v", err)
	}
	if err := os.WriteFile(p, []byte(`{"schema": 9}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := reference.ReadFile(p); err == nil || !strings.Contains(err.Error(), p) {
		t.Fatalf("want an error naming %s, got %v", p, err)
	}
	if _, err := reference.ReadFile(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("a missing file read")
	}
}

// TestKavunReference: the repository's own kavun_reference.json reads, and writes back to the same bytes.
func TestKavunReference(t *testing.T) {
	p := filepath.Join("..", reference.FileName)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	r := read(t, data)
	if got := write(t, r); !bytes.Equal(got, data) {
		t.Fatal("Write(Read(kavun_reference.json)) differs from the file")
	}
	if r.Origin != "github.com/jokruger/kavun" || len(r.Types) == 0 || r.Builtins == nil || len(r.Modules) == 0 {
		t.Fatalf("unexpected header or content: %s, %d types, %d modules", r.Origin, len(r.Types), len(r.Modules))
	}
}

// TestDocMarkdown: blocks joined with newlines; an example as a ```go fence longer than any backtick run its code
// starts a line with; a norun example shows as a plain go fence.
func TestDocMarkdown(t *testing.T) {
	cases := []struct {
		doc  reference.Doc
		want string
	}{
		{nil, ""},
		{reference.Doc{{Kind: reference.BlockText, Markdown: "a\n"}, {Kind: reference.BlockExample, Code: "x"}, {Kind: reference.BlockText, Markdown: "b"}},
			"a\n\n```go\nx\n```\nb"},
		{reference.Doc{{Kind: reference.BlockExample, Code: "f(x)", NoRun: true, Reason: "r"}}, "```go\nf(x)\n```"},
		{reference.Doc{{Kind: reference.BlockExample}}, "```go\n```"},
		{reference.Doc{{Kind: reference.BlockExample, Code: "s := `\n```\n`"}}, "````go\ns := `\n```\n`\n````"},
		{reference.Doc{{Kind: reference.BlockExample, Code: "  `````x"}}, "``````go\n  `````x\n``````"},
		{reference.Doc{{Kind: reference.BlockExample, Code: "a ``` b"}}, "```go\na ``` b\n```"},
		{reference.Doc{{Kind: reference.BlockText, Markdown: "x"}, {Kind: reference.BlockText, Markdown: ""}, {Kind: reference.BlockText, Markdown: "y"}}, "x\n\ny"},
	}
	for _, c := range cases {
		if got := c.doc.Markdown(); got != c.want {
			t.Errorf("Markdown(%+v)\n got  %q\n want %q", c.doc, got, c.want)
		}
	}
}

// TestDocExamples: the example blocks in order, text blocks left out.
func TestDocExamples(t *testing.T) {
	a := reference.Block{Kind: reference.BlockExample, Code: "a", Line: 3}
	b := reference.Block{Kind: reference.BlockExample, Code: "b", Line: 9, NoRun: true}
	doc := reference.Doc{{Kind: reference.BlockText, Markdown: "t"}, a, {Kind: reference.BlockText, Markdown: "u"}, b}
	if got := doc.Examples(); !reflect.DeepEqual(got, []reference.Block{a, b}) {
		t.Fatalf("got %+v", got)
	}
	if got := (reference.Doc{}).Examples(); got != nil {
		t.Fatalf("got %+v", got)
	}
}

// TestDocLinks: the kavun: targets of inline links, autolinks and link definitions in text blocks, in order, repeats
// included; not in a fenced block (``` or ~~~, indented up to three spaces), a code span or an example; other
// targets ignored.
func TestDocLinks(t *testing.T) {
	doc := reference.Doc{
		{Kind: reference.BlockText, Markdown: "See [splice](kavun:type/array/splice) and [x](kavun:type/string#undecodable-octets \"title\").\n" +
			"[web](https://example.com) [rel](string.md#x) [same](#y) `[code](kavun:type/no)` [again](kavun:type/array/splice)\n" +
			"[angle](<kavun:module/fin/irr>)\n" +
			"```md\n[fenced](kavun:type/no)\n```\n" +
			"[def]: kavun:doc/embedding#handling-errors\n" +
			"``x`` [after](kavun:builtin/range) `` `unclosed"},
		{Kind: reference.BlockExample, Code: "// [ex](kavun:type/no)"},
		{Kind: reference.BlockText, Markdown: "[b](kavun:builtins)"},
		{Kind: reference.BlockText, Markdown: "<kavun:type/int> then [i](kavun:type/int#durations) <https://x> <kavun:module/os>\n" +
			"~~~\n[tilde](kavun:type/no)\n```\n<kavun:type/no>\n~~~~\n" +
			"   ```md\n[indented](kavun:type/no)\n  ```\n" +
			"    ```\n[four spaces is no fence](kavun:type/a)\n" +
			"~~~ info\n~~~ not a close\n[in](kavun:type/no)\n ~~~\n" +
			"```not`a fence\n[after](kavun:type/b)"},
	}
	want := []string{
		"kavun:type/array/splice", "kavun:type/string#undecodable-octets", "kavun:type/array/splice",
		"kavun:module/fin/irr", "kavun:doc/embedding#handling-errors", "kavun:builtin/range", "kavun:builtins",
		"kavun:type/int", "kavun:type/int#durations", "kavun:module/os", "kavun:type/a", "kavun:type/b",
	}
	if got := doc.Links(); !slices.Equal(got, want) {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

// TestMerge: a merged Set answers every lookup with the entry's origin; the same name twice or the same origin twice
// is refused naming both origins; different core versions are accepted.
func TestMerge(t *testing.T) {
	a := sample(t, "example.com/a")
	b := &reference.Reference{Schema: 1, Origin: "example.com/b", CoreVersion: "1.1",
		Types: []reference.Type{{Name: "widget", Members: []reference.Callable{callable(t, "double", "()", true)}},
			{Name: "ext.thing", Module: "ext", Members: []reference.Callable{}}},
		Modules: []reference.Module{{Name: "ext", Functions: []reference.Callable{callable(t, "run", "(x)", true)}}},
	}
	s, err := reference.Merge(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if refs := s.References(); len(refs) != 2 || refs[0] != a || refs[1] != b {
		t.Fatalf("References: %v", refs)
	}
	if ty, ok := s.Type("widget"); !ok || ty != &b.Types[0] {
		t.Fatal("Type")
	}
	if _, ok := s.Type("nope"); ok {
		t.Fatal("Type: a missing type found")
	}
	if c, ok := s.Member("array", "splice"); !ok || c != &a.Types[0].Members[2] {
		t.Fatal("Member")
	}
	for _, miss := range [][2]string{{"array", "nope"}, {"nope", "splice"}} {
		if _, ok := s.Member(miss[0], miss[1]); ok {
			t.Fatalf("Member%v found", miss)
		}
	}
	if m, ok := s.Module("fs"); !ok || m != &a.Modules[0] {
		t.Fatal("Module")
	}
	if c, ok := s.Function("ext", "run"); !ok || c.Name != "run" {
		t.Fatal("Function")
	}
	if _, ok := s.Function("nope", "run"); ok {
		t.Fatal("Function: a missing module found")
	}
	if c, ok := s.Constant("fs", "sep"); !ok || c.Type != "string" {
		t.Fatal("Constant")
	}
	if _, ok := s.Constant("fs", "nope"); ok {
		t.Fatal("Constant: a missing constant found")
	}
	if c, ok := s.Builtin("range"); !ok || c.Name != "range" {
		t.Fatal("Builtin")
	}
	if tg, err := s.Resolve("kavun:type/widget"); err != nil || tg.Origin != "example.com/b" {
		t.Fatalf("Resolve: %v %v", tg, err)
	}
	if tg, err := s.Resolve("kavun:builtin/len"); err != nil || tg.Origin != "example.com/a" {
		t.Fatalf("Resolve: %v %v", tg, err)
	}
	if _, err := s.Resolve("kavun:module/nope"); err == nil {
		t.Fatal("Resolve: a missing module found")
	}
	if got := s.TypesOf("fs"); len(got) != 1 || got[0].Name != "fs.file" {
		t.Fatalf("TypesOf(fs): %v", got)
	}
	if got := s.TypesOf(""); got != nil {
		t.Fatalf("TypesOf(\"\"): %v", got)
	}

	var got []string
	for e := range s.Callables() {
		got = append(got, string(e.Kind)+" "+e.Owner+" "+e.Callable.Name+" "+e.Origin)
	}
	want := []string{
		"type array map example.com/a", "type array map_in_place example.com/a", "type array splice example.com/a",
		"type widget double example.com/b",
		"builtins  len example.com/a", "builtins  range example.com/a",
		"module ext run example.com/b", "module fs close example.com/a", "module fs open example.com/a",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Callables:\n got  %q\n want %q", got, want)
	}
	n := 0
	for range s.Callables() {
		n++
		break
	}
	if n != 1 {
		t.Fatal("Callables does not stop")
	}

	conflicts := []struct {
		refs []*reference.Reference
		want string
	}{
		{[]*reference.Reference{a, sample(t, "example.com/a")}, "origin example.com/a is given twice"},
		{[]*reference.Reference{a, sample(t, "example.com/c")}, "type array is declared by example.com/a and by example.com/c"},
		{[]*reference.Reference{a, {Schema: 1, Origin: "x", Modules: []reference.Module{{Name: "fs"}}}}, "module fs is declared by example.com/a and by x"},
		{[]*reference.Reference{a, {Schema: 1, Origin: "x", Builtins: &reference.Builtins{Functions: []reference.Callable{{Name: "len"}}}}}, "builtin len is declared by example.com/a and by x"},
	}
	for _, c := range conflicts {
		if _, err := reference.Merge(c.refs...); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("want %q, got %v", c.want, err)
		}
	}
}

// TestTwin: x ↔ x_in_place, both ways, only when the type declares both.
func TestTwin(t *testing.T) {
	ty := &sample(t, "o").Types[0]
	if c, ok := ty.Twin("map"); !ok || c.Name != "map_in_place" {
		t.Fatal("Twin(map)")
	}
	if c, ok := ty.Twin("map_in_place"); !ok || c.Name != "map" {
		t.Fatal("Twin(map_in_place)")
	}
	for _, name := range []string{"splice", "nope", "nope_in_place"} {
		if _, ok := ty.Twin(name); ok {
			t.Errorf("Twin(%s) found", name)
		}
	}
}

// TestSlug: GitHub's anchors — lower case, spaces to hyphens, punctuation and backticks dropped, letters of any
// script, numbers, `_` and `-` kept.
func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Undecodable octets":          "undecodable-octets",
		"Rate and charge tables":      "rate-and-charge-tables",
		"`splice` and friends":        "splice-and-friends",
		"`pad_start(n[, fill]) -> x`": "pad_startn-fill---x",
		"The module grant":            "the-module-grant",
		"  Trimmed  ":                 "trimmed",
		"Two  spaces":                 "two--spaces",
		"x.y/z: (a), b!":              "xyz-a-b",
		"Durations — and time":        "durations--and-time",
		"Ünïcödé Straße":              "ünïcödé-straße",
		"Кавун і час":                 "кавун-і-час",
		"C++ & Go's <tags>":           "c--gos-tags",
		"pre-existing_name 2":         "pre-existing_name-2",
		"x² ½ Ⅳ 3":                    "x---3",
		"a‿b ＿c":                      "a‿b-＿c",
		"emoji 🍉 here":                "emoji--here",
		"":                            "",
	}
	for in, want := range cases {
		if got := reference.Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

var (
	headingLine  = regexp.MustCompile(`(?m)^#{1,6} (.+)$`)
	anchorTag    = regexp.MustCompile(`<a id="([^"]+)"></a>`)
	samePageLink = regexp.MustCompile(`\]\(#([^)]+)\)`)
)

// TestSlugOnPages: every same-page link of the generated pages under docs/reference names an explicit anchor or
// the Slug of a heading of its page — Slug agrees with the anchors the pages already link to.
func TestSlugOnPages(t *testing.T) {
	var pages []string
	err := filepath.WalkDir(filepath.Join("..", "docs", "reference"), func(p string, e os.DirEntry, err error) error {
		if err == nil && !e.IsDir() && strings.HasSuffix(p, ".md") {
			pages = append(pages, p)
		}
		return err
	})
	if err != nil || len(pages) == 0 {
		t.Fatalf("no pages: %v", err)
	}
	links := 0
	for _, p := range pages {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		// fenced blocks hold no headings
		var prose []string
		fence := ""
		for _, line := range strings.Split(string(data), "\n") {
			run := len(line) - len(strings.TrimLeft(line, "`"))
			switch {
			case fence != "":
				if run >= len(fence) && strings.TrimSpace(line[run:]) == "" {
					fence = ""
				}
			case run >= 3:
				fence = line[:run]
			default:
				prose = append(prose, line)
			}
		}
		text := strings.Join(prose, "\n")
		anchors := map[string]bool{}
		for _, m := range anchorTag.FindAllStringSubmatch(text, -1) {
			anchors[m[1]] = true
		}
		for _, m := range headingLine.FindAllStringSubmatch(text, -1) {
			anchors[reference.Slug(m[1])] = true
		}
		for _, m := range samePageLink.FindAllStringSubmatch(text, -1) {
			links++
			if !anchors[m[1]] {
				t.Errorf("%s: link #%s names no anchor", p, m[1])
			}
		}
	}
	if links < 1000 {
		t.Fatalf("only %d same-page links", links)
	}
}

// TestFindLinks: every link destination of a text with its offset and line — inline, angle-bracketed, autolink to
// any scheme, link definition — outside fenced blocks and code spans; ReplaceLinks rewrites exactly those.
func TestFindLinks(t *testing.T) {
	md := "a [x](string.md#y) `[c](no)` [s](#z \"t\")\n" +
		"<https://e.com/a> <b> [u](<kavun:type/int>)\n" +
		"```\n[f](no)\n```\n" +
		"[def]: ../x.md\n" +
		"<mailto:a@b.c>"
	want := []reference.Link{
		{Target: "string.md#y", Offset: 6, Line: 0},
		{Target: "#z", Offset: 33, Line: 0},
		{Target: "https://e.com/a", Offset: 42, Line: 1},
		{Target: "kavun:type/int", Offset: 68, Line: 1},
		{Target: "../x.md", Offset: 108, Line: 5},
		{Target: "mailto:a@b.c", Offset: 117, Line: 6},
	}
	got := reference.FindLinks(md)
	if !slices.Equal(got, want) {
		t.Fatalf("got  %v\nwant %v", got, want)
	}
	for _, l := range got {
		if md[l.Offset:l.Offset+len(l.Target)] != l.Target {
			t.Fatalf("offset of %q is wrong", l.Target)
		}
	}
	out := reference.ReplaceLinks(md, func(l reference.Link) string { return "<" + strings.ToUpper(l.Target) + ">" })
	wantOut := "a [x](<STRING.MD#Y>) `[c](no)` [s](<#Z> \"t\")\n" +
		"<<HTTPS://E.COM/A>> <b> [u](<<KAVUN:TYPE/INT>>)\n" +
		"```\n[f](no)\n```\n" +
		"[def]: <../X.MD>\n" +
		"<<MAILTO:A@B.C>>"
	if out != wantOut {
		t.Fatalf("ReplaceLinks:\n%s\nwant\n%s", out, wantOut)
	}
}

// TestDocHeadings: the slugs of the ATX headings of the text blocks, outside fenced blocks, closing #s dropped.
func TestDocHeadings(t *testing.T) {
	doc := reference.Doc{
		{Kind: reference.BlockText, Markdown: "## Rate and charge tables\ntext #not\n```\n## fenced\n```\n### `x.y` form ##\n#nospace"},
		{Kind: reference.BlockExample, Code: "## no"},
		{Kind: reference.BlockText, Markdown: "   # Top"},
	}
	want := []string{"rate-and-charge-tables", "xy-form", "top"}
	if got := doc.Headings(); !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if text, ok := reference.HeadingText("## Components ##"); !ok || text != "Components" {
		t.Fatalf("HeadingText: %q %v", text, ok)
	}
}

// TestParseLink: every link form and its refusals; String writes a target back as its link.
func TestParseLink(t *testing.T) {
	for _, c := range []struct {
		link string
		want reference.Target
	}{
		{"kavun:type/array", reference.Target{Kind: reference.OwnerType, Owner: "array"}},
		{"kavun:type/array/splice", reference.Target{Kind: reference.OwnerType, Owner: "array", Entry: "splice"}},
		{"kavun:type/string#undecodable-octets", reference.Target{Kind: reference.OwnerType, Owner: "string", Fragment: "undecodable-octets"}},
		{"kavun:type/fin.year_fraction", reference.Target{Kind: reference.OwnerType, Owner: "fin.year_fraction"}},
		{"kavun:module/fin/irr", reference.Target{Kind: reference.OwnerModule, Owner: "fin", Entry: "irr"}},
		{"kavun:module/fin#day-count", reference.Target{Kind: reference.OwnerModule, Owner: "fin", Fragment: "day-count"}},
		{"kavun:builtins", reference.Target{Kind: reference.OwnerBuiltins}},
		{"kavun:builtins#conversions", reference.Target{Kind: reference.OwnerBuiltins, Fragment: "conversions"}},
		{"kavun:builtin/range", reference.Target{Kind: reference.OwnerBuiltins, Entry: "range"}},
		{"kavun:doc/embedding#handling-errors", reference.Target{Kind: reference.OwnerDoc, Owner: "embedding", Fragment: "handling-errors"}},
		{"kavun:doc/guides/x", reference.Target{Kind: reference.OwnerDoc, Owner: "guides/x"}},
	} {
		got, err := reference.ParseLink(c.link)
		if err != nil || got != c.want {
			t.Fatalf("%s: got %+v, %v", c.link, got, err)
		}
		if got.String() != c.link {
			t.Fatalf("String: %q, want %q", got.String(), c.link)
		}
	}
	for _, bad := range []string{
		"type/array", "kavun:", "kavun:type", "kavun:type/", "kavun:type/a/b/c", "kavun:type//b", "kavun:member/array.splice",
		"kavun:builtins/len", "kavun:builtin", "kavun:builtin/a/b", "kavun:doc", "kavun:doc/x.md", "kavun:doc/../x",
		"kavun:type/array#", "kavun:type/array/splice#x", "kavun:builtin/range#x",
	} {
		if _, err := reference.ParseLink(bad); err == nil {
			t.Fatalf("%s: accepted", bad)
		}
	}
}

// TestResolve: a link resolves to an existing owner, entry or root-block heading, with the origin declaring it; a
// doc link is parsed only.
func TestResolve(t *testing.T) {
	a := sample(t, "example.com/a")
	a.Types[0].Doc = append(reference.Doc{{Kind: reference.BlockText, Markdown: "## Callbacks\ntext"}}, a.Types[0].Doc...)
	a.Builtins.Doc = reference.Doc{{Kind: reference.BlockText, Markdown: "## Conversions"}}
	b := &reference.Reference{Schema: 1, Origin: "example.com/b",
		Types:   []reference.Type{{Name: "widget", Members: []reference.Callable{callable(t, "double", "()", true)}}},
		Modules: []reference.Module{{Name: "ext", Functions: []reference.Callable{callable(t, "run", "(x)", true)}}},
	}
	s, err := reference.Merge(a, b)
	if err != nil {
		t.Fatal(err)
	}
	for link, origin := range map[string]string{
		"kavun:type/array":            "example.com/a",
		"kavun:type/array/splice":     "example.com/a",
		"kavun:type/array#callbacks":  "example.com/a",
		"kavun:type/widget/double":    "example.com/b",
		"kavun:module/fs/open":        "example.com/a",
		"kavun:module/fs/sep":         "example.com/a",
		"kavun:module/ext":            "example.com/b",
		"kavun:builtins#conversions":  "example.com/a",
		"kavun:builtin/range":         "example.com/a",
		"kavun:doc/embedding#nowhere": "",
	} {
		tg, err := s.Resolve(link)
		if err != nil || tg.Origin != origin || tg.String() != link {
			t.Fatalf("%s: %+v, %v", link, tg, err)
		}
	}
	for link, msg := range map[string]string{
		"kavun:type/nope":          "no type nope",
		"kavun:type/array/nope":    "type array has no member nope",
		"kavun:type/array#nope":    `no heading "nope" in the block of type array`,
		"kavun:module/nope/x":      "no module nope",
		"kavun:module/fs/nope":     "module fs has no function or constant nope",
		"kavun:module/fs#nope":     `no heading "nope" in the block of module fs`,
		"kavun:builtin/nope":       "no builtin function nope",
		"kavun:builtins#nope":      `no heading "nope" in the block of the builtins`,
		"kavun:frob/x":             `unknown kind "frob"`,
		"kavun:type/array/a#frag":  "takes no fragment",
		"https://example.com/x.md": "not a kavun: link",
	} {
		if _, err := s.Resolve(link); err == nil || !strings.Contains(err.Error(), msg) {
			t.Fatalf("%s: got %v, want %q", link, err, msg)
		}
	}
	none, err := reference.Merge(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := none.Resolve("kavun:builtins"); err == nil {
		t.Fatal("kavun:builtins resolved with no builtins block")
	}
}
