package kavungen

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// File is one generated file. A nil Content means the file must not exist: its package declares nothing any more
// (a stale kavun_gen.go), or the docs root holds a page nothing generates; either is to be deleted.
type File struct {
	Path    string // a kavun_gen.go: the caller's directory joined with GeneratedFile; a page: Options.DocsDir joined with its path
	Content []byte
}

// Options configures a generation beyond the packages it reads.
type Options struct {
	// DocsDir is the root the Markdown reference pages are generated into (docs/reference); "" generates none. The
	// directory is wholly generated: a file in it that the generation does not produce is reported as stale.
	DocsDir string
	// Root is the module root: the header of a generated page names its source files relative to it. "" leaves the
	// package directories as given.
	Root string
	// KeepStale leaves the files under DocsDir that the generation does not produce alone — neither reported nor
	// deleted. A run over part of a module sets it: the pages of the packages it did not read are not stale.
	KeepStale bool
}

// Generate reads the packages in dirs and answers the kavun_gen.go of every package that declares something or still
// holds one, and, with opts.DocsDir, every Markdown page under it (including the stale ones, to be deleted). Every
// problem in every package is reported in one error, each line naming file:line. Nothing is written.
func Generate(ctx *Context, dirs []string, opts Options) ([]File, error) {
	var out []File
	var errs errorList
	var models []*packageModel
	for _, dir := range dirs {
		f, model, problems := generateDir(ctx, dir)
		errs = append(errs, problems...)
		if f != nil {
			out = append(out, *f)
		}
		if model != nil && !model.empty() {
			rel, err := relDir(opts.Root, dir)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			model.relDir = rel
			models = append(models, model)
		}
	}
	if err := errs.err(); err != nil {
		return nil, err
	}
	if opts.DocsDir == "" {
		return out, nil
	}

	pages, problems := emitDocs(models)
	if matrix := renderFunctionMatrix(models); matrix != nil && !opts.KeepStale {
		pages[matrixPage] = matrix
	}
	if err := problems.err(); err != nil {
		return nil, err
	}
	docs, err := docFiles(opts.DocsDir, pages, opts.KeepStale)
	if err != nil {
		return nil, err
	}
	return append(out, docs...), nil
}

// relDir answers dir relative to root, slash-separated; dir itself when root is "".
func relDir(root, dir string) (string, error) {
	if root == "" {
		return filepath.ToSlash(dir), nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absDir)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

// docFiles turns the rendered pages into files under docsDir, in path order, followed (unless keepStale) by a
// deletion for every file already under docsDir that no page produces.
func docFiles(docsDir string, pages map[string][]byte, keepStale bool) ([]File, error) {
	rels := make([]string, 0, len(pages))
	for rel := range pages {
		rels = append(rels, rel)
	}
	slices.Sort(rels)

	var out []File
	produced := map[string]bool{}
	for _, rel := range rels {
		p := filepath.Join(docsDir, filepath.FromSlash(rel))
		produced[p] = true
		out = append(out, File{Path: p, Content: pages[rel]})
	}

	if keepStale {
		return out, nil
	}
	err := filepath.WalkDir(docsDir, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && p == docsDir {
				return fs.SkipAll // no docs root yet: nothing can be stale
			}
			return err
		}
		if !e.IsDir() && !produced[p] {
			out = append(out, File{Path: p}) // stale
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// generateDir generates one package directory: a nil file when it neither declares anything nor holds a
// kavun_gen.go. The model is nil when the package could not be read.
func generateDir(ctx *Context, dir string) (*File, *packageModel, errorList) {
	files, hasGenerated, errs := readPackage(dir)
	if len(errs) > 0 {
		return nil, nil, errs
	}
	target := filepath.Join(dir, GeneratedFile)

	content, model, problems := generateSources(ctx, dir, files)
	if len(problems) > 0 {
		return nil, nil, problems
	}
	if content == nil && !hasGenerated {
		return nil, model, nil
	}
	return &File{Path: target, Content: content}, model, nil
}

// readPackage reads the Go source files of one package directory that the build includes, _test.go files and
// kavun_gen.go excluded; hasGenerated reports that a kavun_gen.go is there (one not written by kavun-gen is refused).
func readPackage(dir string) (files []sourceFile, hasGenerated bool, errs errorList) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		errs = append(errs, err.Error())
		return nil, false, errs
	}

	target := filepath.Join(dir, GeneratedFile)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		// only the files the build includes: a directive in a file excluded by //go:build or a _GOOS/_GOARCH
		// suffix would name a function that is not compiled
		if ok, err := build.Default.MatchFile(dir, name); err != nil {
			errs = append(errs, err.Error())
			continue
		} else if !ok {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if name == GeneratedFile {
			if !bytes.HasPrefix(src, []byte(generatedHeader)) {
				errs = append(errs, fmt.Sprintf("%s: exists but was not written by kavun-gen (no %q header); refusing to replace it", filepath.ToSlash(target), strings.TrimSpace(generatedHeader)))
				return nil, false, errs
			}
			hasGenerated = true
			continue
		}
		files = append(files, sourceFile{Name: name, Src: src})
	}
	return files, hasGenerated, errs
}

// generateSources generates one package from its sources (kavun_gen.go and _test.go files excluded): nil content
// when the package declares nothing. dir only labels the messages. The model is what the Markdown emitter reads.
func generateSources(ctx *Context, dir string, files []sourceFile) ([]byte, *packageModel, errorList) {
	scanned, errs := scanPackage(dir, files)
	model, modelErrs := buildModel(ctx, scanned)
	errs = append(errs, modelErrs...)
	if len(errs) > 0 || model.emptyGo() {
		return nil, model, errs
	}
	content, err := emitGo(ctx, model)
	if err != nil {
		errs = append(errs, err.Error())
		return nil, model, errs
	}
	return content, model, nil
}

// Write applies the files to disk: writes each content, deletes each file whose content is nil.
func Write(files []File) error {
	for _, f := range files {
		if f.Content == nil {
			if err := os.Remove(f.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			continue
		}
		if old, err := os.ReadFile(f.Path); err == nil && bytes.Equal(old, f.Content) {
			continue // unchanged: leave the file and its timestamp alone
		}
		if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(f.Path, f.Content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Check compares the files with what is on disk and answers one unified diff per file that differs (a missing
// file diffs against empty content, a file that must not exist against its content); nil when all are in sync.
func Check(files []File) ([]string, error) {
	var diffs []string
	for _, f := range files {
		old, err := os.ReadFile(f.Path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		if bytes.Equal(old, f.Content) && (old != nil) == (f.Content != nil) {
			continue
		}
		diffs = append(diffs, unifiedDiff(filepath.ToSlash(f.Path)+" (on disk)", filepath.ToSlash(f.Path)+" (generated)", string(old), string(f.Content)))
	}
	return diffs, nil
}

// ResolvePatterns answers the directories of the packages matching Go package patterns ("./..."), resolved by
// `go list` run in dir.
func ResolvePatterns(dir string, patterns []string) ([]string, error) {
	args := append([]string{"list", "-e", "-f", "{{.Dir}}"}, patterns...)
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list %s: %v\n%s", strings.Join(patterns, " "), err, stderr.String())
	}
	return strings.Fields(string(out)), nil
}

// MainModule answers the path and root directory of the main module seen from dir; ok is false when there is not
// exactly one (outside a module, or in a workspace of several).
func MainModule(dir string) (modPath, modDir string, ok bool) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Path}}\t{{.Dir}}")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", "", false
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 1 {
		return "", "", false
	}
	modPath, modDir, ok = strings.Cut(lines[0], "\t")
	return modPath, modDir, ok && modDir != ""
}

// KavunModule is the module path of Kavun itself: generating it writes the reference pages by default.
const KavunModule = "github.com/jokruger/kavun"

// KavunRoot answers the root directory of the github.com/jokruger/kavun module as seen from dir — the repository
// itself, or the module cache copy an embedder's build uses.
func KavunRoot(dir string) (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", KavunModule)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("locating the kavun module: %v\n%s", err, stderr.String())
	}
	return strings.TrimSpace(string(out)), nil
}
