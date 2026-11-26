// Package kavundocs is the logic of cmd/kavun-docs: it renders the Markdown language reference from one or more
// kavun_reference.json files (package reference), never from Go source — the same public path an embedder's own
// tools take.
//
// The references given as arguments are rendered, merged into one site (reference.Merge). The references given
// with -with are loaded only to resolve links into other origins: their pages live at their docs URL (the -with
// value after "=", by default the file's docs field). Every reference loaded, rendered or not, is merged once, so a
// type, module or builtin function declared twice, or an origin given twice, is refused.
//
// The pages, below the output root: types/<type>.md (the dot of a module-owned type's name as an underscore:
// types/fs_file.md), builtins.md, modules/<module>.md, the module index modules.md and the member-function matrix
// function_matrix.md. A type, module or builtins page is its title, the root block (summary — for a module, then
// its grant line — then the body), then `## Members` (`## Functions`): an index with one linked line per entry, then
// one entry each in alphabetical (byte) order — an explicit anchor <a id="name"></a>, a `###` heading holding the
// signature in code (every reading of the template as name(reading) -> result, joined by " / "), then the summary
// and the body. A type with no members has no `## Members`. A module with constants has `## Constants` after its
// functions, in the same form, each heading holding the name and the type in code (`pi -> float`). The matrix has one column per type with members
// (builtin types, then module-owned types) and one row per member name.
//
// A doc block's links name what they link to (kavun: links, see package reference); each page renders them as
// paths (links.go): into a rendered origin relative to the page ("#splice" on the same page, "array.md#splice",
// "../modules/fin.md#irr"); kavun:doc/<page> relative to <docs root>/<page>.md, the docs root being the output
// root's parent — the file must exist and hold the fragment's heading, or the link is refused with the position of
// the declaration holding it; into a -with origin as <docs URL>/reference/<page path>, refused when the origin has
// no docs URL. The text kavun-docs writes itself — a world module's grant line, the matrix introduction — links to
// Kavun's guide pages (embedding.md, types.md): Kavun's origin (reference.KavunOrigin) rendered, relative and
// checked like a kavun:doc/ link; else under the docs URL of Kavun's -with reference; a page that carries such a link
// while Kavun's reference is neither rendered nor given a docs URL is refused.
//
// The output root is wholly generated: a file there that no page produces is deleted (with -check: reported). Every
// file there must carry a generated header; one that does not (a hand-written page, an image) stops the run before
// anything is written or deleted.
package kavundocs

import (
	"fmt"
	"io"
	"strings"

	"github.com/jokruger/kavun/internal/genfile"
	"github.com/jokruger/kavun/reference"
)

// Config is one kavun-docs invocation.
type Config struct {
	Out    string    // the output root (<docs root>/reference); wholly generated
	Check  bool      // compare with the files on disk instead of writing them
	Render []string  // the reference files rendered (the arguments)
	With   []With    // the reference files loaded for links only (-with)
	Stdout io.Writer // where -check prints its diffs
}

// With is one -with value: a reference file loaded for links only, and the URL of its origin's published docs root
// ("" for the file's docs field).
type With struct {
	Path string
	URL  string
}

// ParseWith parses a -with value, <path>[=<docs URL>]: the path ends at the first "=", so it cannot contain one.
func ParseWith(s string) (With, error) {
	path, url, _ := strings.Cut(s, "=")
	if path == "" {
		return With{}, fmt.Errorf("-with %q: no reference file", s)
	}
	return With{Path: path, URL: url}, nil
}

// Site is what a run works from: the rendered references merged, and every loaded reference merged for resolving
// links, with the docs URL of each origin that is not rendered.
type Site struct {
	Rendered *reference.Set
	Links    *reference.Set
	// URLs maps the origin of each -with reference to the URL of its published docs root ("" when neither the
	// -with value nor the file gives one).
	URLs map[string]string
}

// Load reads the references of a run and merges them: the rendered ones into Site.Rendered, all of them into
// Site.Links. A merge conflict (a name declared twice, an origin given twice) is refused.
func Load(render []string, with []With) (*Site, error) {
	if len(render) == 0 {
		return nil, fmt.Errorf("no reference to render")
	}
	var rendered, all []*reference.Reference
	for _, path := range render {
		ref, err := reference.ReadFile(path)
		if err != nil {
			return nil, err
		}
		rendered = append(rendered, ref)
	}
	all = append(all, rendered...)
	urls := map[string]string{}
	for _, w := range with {
		ref, err := reference.ReadFile(w.Path)
		if err != nil {
			return nil, err
		}
		all = append(all, ref)
		url := w.URL
		if url == "" {
			url = ref.Docs
		}
		urls[ref.Origin] = strings.TrimSuffix(url, "/")
	}
	return NewSite(rendered, all, urls)
}

// NewSite builds a Site from references already read: rendered are rendered, all (rendered included) resolve links,
// urls gives the docs root of each origin that is not rendered.
func NewSite(rendered, all []*reference.Reference, urls map[string]string) (*Site, error) {
	links, err := reference.Merge(all...)
	if err != nil {
		return nil, err
	}
	set, err := reference.Merge(rendered...)
	if err != nil {
		return nil, err
	}
	return &Site{Rendered: set, Links: links, URLs: urls}, nil
}

// Files answers the files of the output root: every page of the site, then a deletion for every file already
// under out that no page produces.
func (s *Site) Files(out string) ([]genfile.File, error) {
	pages, err := s.Pages(out)
	if err != nil {
		return nil, err
	}
	return genfile.Tree(out, pages)
}

// Run is the whole of kavun-docs: it loads the references, renders the pages, and writes them (deleting every other
// file under the output root) or, with Check, prints a diff of every out-of-date file and fails when there is one.
// A file under the output root that is not a generated one (genfile.IsGenerated) fails the run, -check included,
// before anything is written or deleted.
func Run(cfg Config) error {
	if cfg.Out == "" {
		return fmt.Errorf("no output root (-out)")
	}
	site, err := Load(cfg.Render, cfg.With)
	if err != nil {
		return err
	}
	files, err := site.Files(cfg.Out)
	if err != nil {
		return err
	}
	if !cfg.Check {
		return genfile.Write(files)
	}
	diffs, err := genfile.Check(files)
	if err != nil {
		return err
	}
	for _, d := range diffs {
		fmt.Fprint(cfg.Stdout, d)
	}
	if len(diffs) > 0 {
		return fmt.Errorf("%d generated file(s) out of date: run kavun-docs without -check", len(diffs))
	}
	return nil
}
