package kavungen

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// RunConfig is one kavun-gen invocation.
type RunConfig struct {
	Dir      string   // the directory it runs in: patterns resolve, and messages name files, relative to it
	Check    bool     // compare with the files on disk instead of writing them
	Docs     *string  // the -docs value; nil when the flag was not given
	Patterns []string // Go package patterns ("./...")
	Stdout   io.Writer
	Stderr   io.Writer
}

// Run is the whole of kavun-gen: it resolves the packages, decides where the reference pages go, generates, and
// writes or (with Check) prints a diff of every out-of-date file, failing when there is one.
//
// Packages under <kavun root>/tmp/ — the repository's scratch folder — are never read (SkipScratch). The pages
// go to Docs when given ("" for none); by default to <module root>/docs/reference, but only when the main module
// is Kavun and the patterns cover all of its packages. docs/reference is wholly generated, so a run over part of
// the module never reports or deletes a page as stale: an explicit -docs then generates the pages of the packages
// it read and leaves every other file alone, and the default generates none.
func Run(cfg RunConfig) error {
	root, err := KavunRoot(cfg.Dir)
	if err != nil {
		return err
	}
	ctx, err := LoadContext(root)
	if err != nil {
		return err
	}
	dirs, err := ResolvePatterns(cfg.Dir, cfg.Patterns)
	if err != nil {
		return err
	}
	dirs = SkipScratch(root, dirs)

	opts := Options{Root: cfg.Dir}
	modPath, modDir, isModule := MainModule(cfg.Dir)
	covered := false
	if isModule {
		opts.Root = modDir
		all, err := ResolvePatterns(modDir, []string{"./..."})
		if err != nil {
			return err
		}
		covered = covers(dirs, SkipScratch(root, all))
	}
	switch {
	case cfg.Docs != nil:
		opts.DocsDir = *cfg.Docs
		if opts.DocsDir != "" && !covered {
			opts.KeepStale = true
			fmt.Fprintln(cfg.Stderr, "kavun-gen: the packages do not cover the whole module: their pages are generated, but no page is reported or deleted as stale")
		}
	case isModule && modPath == KavunModule:
		if covered {
			opts.DocsDir = filepath.Join(modDir, "docs", "reference")
		} else {
			fmt.Fprintln(cfg.Stderr, "kavun-gen: reference pages not generated: the packages do not cover the whole module (run it on ./... from the module root, or pass -docs)")
		}
	}

	// messages name files relative to where the tool runs — when that is the process's working directory, so the
	// relative paths still resolve
	if wd, err := os.Getwd(); err == nil && sameDir(wd, cfg.Dir) {
		for i, d := range dirs {
			if rel, err := filepath.Rel(wd, d); err == nil {
				dirs[i] = rel
			}
		}
		if filepath.IsAbs(opts.DocsDir) {
			if rel, err := filepath.Rel(wd, opts.DocsDir); err == nil {
				opts.DocsDir = rel
			}
		}
	}

	files, err := Generate(ctx, dirs, opts)
	if err != nil {
		return err
	}
	if !cfg.Check {
		return Write(files)
	}
	diffs, err := Check(files)
	if err != nil {
		return err
	}
	for _, d := range diffs {
		fmt.Fprint(cfg.Stdout, d)
	}
	if len(diffs) > 0 {
		return fmt.Errorf("%d generated file(s) out of date: run `go generate ./...`", len(diffs))
	}
	return nil
}

// SkipScratch drops the package directories at or under <kavunRoot>/tmp, the repository's gitignored scratch
// folder: a probe copied there must neither be generated nor stop generation.
func SkipScratch(kavunRoot string, dirs []string) []string {
	scratch, err := filepath.Abs(filepath.Join(kavunRoot, "tmp"))
	if err != nil {
		return dirs
	}
	return slices.DeleteFunc(slices.Clone(dirs), func(d string) bool {
		abs, err := filepath.Abs(d)
		return err == nil && (abs == scratch || strings.HasPrefix(abs, scratch+string(filepath.Separator)))
	})
}

// sameDir reports whether two paths name the same directory.
func sameDir(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	return errA == nil && errB == nil && absA == absB
}

// covers reports whether every directory of all is among dirs.
func covers(dirs, all []string) bool {
	for _, d := range all {
		if !slices.Contains(dirs, d) {
			return false
		}
	}
	return true
}
