package kavundocs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jokruger/kavun/reference"
)

// The links of the pages. A doc block names what it links to (a kavun: link, see package reference); the page
// turns it into a path, by the layout of this package:
//
//   - into a rendered origin: a path relative to the page — "#splice" on the same page, "array.md#splice",
//     "../modules/fin.md#irr"; an entry by its anchor, a heading by its slug;
//   - kavun:doc/<page>[#heading]: a path relative to the page to <docs root>/<page>.md, the docs root being the
//     output root's parent. The file must exist, and with a fragment hold a heading of that slug (or an explicit
//     <a id> anchor); a link that does not is refused with the position of the declaration holding it;
//   - into a -with origin: <docs URL>/reference/<page path>, its docs URL the -with value or its file's docs
//     field; a link into an origin that has neither is refused.
//
// Any other link (an absolute URL) is left as it is.
//
// The text kavun-docs writes itself (a module's grant line, the matrix introduction) links to Kavun's guide pages
// with kavun:doc/ links of Kavun's origin (reference.KavunOrigin), whichever references are rendered: when Kavun's
// reference is rendered, as a path relative to the page, checked like any kavun:doc/ link; else as
// <Kavun's docs URL>/<page>.md, the docs URL of Kavun's -with reference — a page that needs one while Kavun's
// reference is neither rendered nor given a docs URL is refused.

// linker renders the kavun: links of a site's pages and collects the links it cannot render.
type linker struct {
	site     *Site
	docsRoot string            // the directory of the guide pages: the output root's parent
	outBase  string            // the output root's name in the docs root: "reference"
	rendered map[string]bool   // the origins rendered
	guides   map[string]*guide // the guide pages read so far, by page
	problems []string
	reported map[string]bool // the problems already reported
}

// guide is a guide page as a link checks it: whether it exists and its anchors.
type guide struct {
	exists  bool
	anchors map[string]bool
}

// linker answers the linker of the site's pages rendered into out, the output root.
func (s *Site) linker(out string) *linker {
	out = filepath.Clean(out)
	l := &linker{site: s, docsRoot: filepath.Dir(out), outBase: filepath.Base(out), rendered: map[string]bool{}, guides: map[string]*guide{}, reported: map[string]bool{}}
	for _, r := range s.Rendered.References() {
		l.rendered[r.Origin] = true
	}
	return l
}

// render answers md with each kavun: link rendered for the page at from (below the output root); pos names the
// declaration holding md, for a link that cannot be rendered (which is left as it is and reported).
func (l *linker) render(from, md string, pos reference.Pos) string {
	return reference.ReplaceLinks(md, func(link reference.Link) string {
		if !strings.HasPrefix(link.Target, reference.LinkScheme) {
			return link.Target
		}
		href, err := l.href(from, link.Target)
		if err != nil {
			if msg := fmt.Sprintf("%s: %v", pos, err); !l.reported[msg] {
				l.reported[msg] = true // one report for a link repeated in one declaration
				l.problems = append(l.problems, msg)
			}
			return link.Target
		}
		return href
	})
}

// href answers the path or URL of a kavun: link on the page at from.
func (l *linker) href(from, link string) (string, error) {
	t, err := l.site.Links.Resolve(link)
	if err != nil {
		return "", err
	}
	if t.Kind == reference.OwnerDoc {
		if err := l.checkGuide(t); err != nil {
			return "", fmt.Errorf("%s: %v", link, err)
		}
		return relPath(l.outBase+"/"+from, t.Owner+".md") + fragment(t.Fragment), nil
	}
	to := targetPage(t)
	anchor := t.Fragment
	if t.Entry != "" {
		anchor = t.Entry
	}
	if l.rendered[t.Origin] {
		if to == from && anchor != "" {
			return "#" + anchor, nil
		}
		return relPath(from, to) + fragment(anchor), nil
	}
	url := l.site.URLs[t.Origin]
	if url == "" {
		return "", fmt.Errorf("%s: %s is not rendered and has no docs URL (give one with -with <file>=<URL>, or -docs-url to its kavun-gen)", link, t.Origin)
	}
	return url + "/" + referenceDir + "/" + to + fragment(anchor), nil
}

// kavunText answers text of kavun-docs' own (not of a reference) with its kavun:doc/ links, Kavun's guide pages,
// rendered for the page at from; a link that cannot be rendered is left as it is and reported.
func (l *linker) kavunText(from, md string) string {
	return reference.ReplaceLinks(md, func(link reference.Link) string {
		if !strings.HasPrefix(link.Target, reference.LinkScheme) {
			return link.Target
		}
		href, err := l.kavunGuide(from, link.Target)
		if err != nil {
			if msg := fmt.Sprintf("kavun-docs' own text: %v", err); !l.reported[msg] {
				l.reported[msg] = true // one report for a link every module page carries
				l.problems = append(l.problems, msg)
			}
			return link.Target
		}
		return href
	})
}

// kavunGuide answers the path or URL of a kavun:doc/ link to a page of Kavun's guide on the page at from: relative
// when Kavun's reference is rendered (the page and its heading checked), else under Kavun's docs URL.
func (l *linker) kavunGuide(from, link string) (string, error) {
	t, err := reference.ParseLink(link)
	if err != nil {
		return "", err
	}
	if t.Kind != reference.OwnerDoc {
		return "", fmt.Errorf("%s: not a guide page", link)
	}
	if l.rendered[reference.KavunOrigin] {
		if err := l.checkGuide(t); err != nil {
			return "", fmt.Errorf("%s: %v", link, err)
		}
		return relPath(l.outBase+"/"+from, t.Owner+".md") + fragment(t.Fragment), nil
	}
	url := l.site.URLs[reference.KavunOrigin]
	if url == "" {
		return "", fmt.Errorf("%s: a page of Kavun's guide, and Kavun's reference is neither rendered nor given a docs URL (give Kavun's kavun_reference.json with -with)", link)
	}
	return url + "/" + t.Owner + ".md" + fragment(t.Fragment), nil
}

func fragment(f string) string {
	if f == "" {
		return ""
	}
	return "#" + f
}

// referenceDir is where the reference of an origin lives below its docs URL, by convention: <docs URL>/reference/.
const referenceDir = "reference"

// targetPage answers the page of a link's owner below the output root.
func targetPage(t reference.Target) string {
	switch t.Kind {
	case reference.OwnerType:
		return TypePage(t.Owner)
	case reference.OwnerModule:
		return ModulePage(t.Owner)
	}
	return BuiltinsPage
}

// relPath answers the path of the page to relative to the directory of the page from, both slash-separated paths
// from one root: "array.md", "../modules/fin.md", "types/time.md". Every link between pages is computed by it.
func relPath(from, to string) string {
	var dir []string
	if d := path.Dir(from); d != "." {
		dir = strings.Split(d, "/")
	}
	parts := strings.Split(to, "/")
	i := 0
	for i < len(dir) && i < len(parts)-1 && dir[i] == parts[i] {
		i++
	}
	return strings.Repeat("../", len(dir)-i) + strings.Join(parts[i:], "/")
}

// checkGuide checks a kavun:doc/ target: the page exists under the docs root and has the fragment's anchor.
func (l *linker) checkGuide(t reference.Target) error {
	g, ok := l.guides[t.Owner]
	if !ok {
		g = readGuide(filepath.Join(l.docsRoot, filepath.FromSlash(t.Owner)+".md"))
		l.guides[t.Owner] = g
	}
	switch {
	case !g.exists:
		return fmt.Errorf("no guide page %s.md in %s", t.Owner, l.docsRoot)
	case t.Fragment != "" && !g.anchors[t.Fragment]:
		return fmt.Errorf("%s.md has no heading or anchor %q", t.Owner, t.Fragment)
	}
	return nil
}

// explicitAnchor matches an HTML anchor: <a id="x"> or <a name="x">.
var explicitAnchor = regexp.MustCompile(`<a\s+(?:id|name)="([^"]+)"`)

// readGuide reads a guide page's anchors: the slug of every heading (a repeated slug numbered as GitHub numbers it:
// "x", "x-1", …) and every explicit <a id> or <a name>.
func readGuide(file string) *guide {
	src, err := os.ReadFile(file)
	if err != nil {
		return &guide{}
	}
	g := &guide{exists: true, anchors: map[string]bool{}}
	seen := map[string]int{}
	for _, slug := range (reference.Doc{{Kind: reference.BlockText, Markdown: string(src)}}).Headings() {
		n := seen[slug]
		seen[slug]++
		if n > 0 {
			slug = fmt.Sprintf("%s-%d", slug, n)
		}
		g.anchors[slug] = true
	}
	for _, m := range explicitAnchor.FindAllStringSubmatch(string(src), -1) {
		g.anchors[m[1]] = true
	}
	return g
}
