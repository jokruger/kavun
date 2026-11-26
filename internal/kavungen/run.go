package kavungen

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jokruger/kavun/internal/genfile"
	"github.com/jokruger/kavun/reference"
)

// RunConfig is one kavun-gen invocation.
type RunConfig struct {
	Dir   string // the directory it runs in: patterns resolve, and messages name files, relative to it
	Check bool   // compare with the files on disk instead of writing them
	// Reference is the -reference value, the path of the reference file (relative to Dir); nil when the flag was
	// not given: <module root>/kavun_reference.json. "" generates none.
	Reference *string
	DocsURL   string // the -docs-url value: the reference's docs field; "" for none
	// With holds the -with values: reference files (kavun_reference.json) of other modules, relative to Dir, whose
	// declarations the links of the run may target.
	With     []string
	Patterns []string // Go package patterns ("./...")
	Stdout   io.Writer
	Stderr   io.Writer
}

// Run is the whole of kavun-gen: it resolves the packages, decides where the reference file goes, generates, and writes or (with Check) prints a diff of every out-of-date file, failing when there is one.
//
// The reference file (kavun_reference.json, see package reference) describes the main module: it is generated only
// when the patterns cover all of its packages, at <module root>/kavun_reference.json or at Reference; a run over part
// of the module generates the tables only and says so on stderr.
//
// The links of the doc blocks are checked against the run's own declarations and the references it loads: when the
// main module is not Kavun, Kavun's reference — <kavun root>/kavun_reference.json, from the Kavun module the run
// builds against —, and every -with file. A name declared by two of them is refused (reference.Merge).
//
// Packages under <kavun root>/tmp/ — the repository's scratch folder — are never read (SkipScratch). The Markdown
// pages are not kavun-gen's: cmd/kavun-docs renders them from the reference file.
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
	if modPath != KavunModule {
		ref, err := reference.ReadFile(filepath.Join(root, reference.FileName))
		if err != nil {
			return fmt.Errorf("loading Kavun's reference from the Kavun module the module builds against, %s (the links may target Kavun's declarations): %v\n"+
				"run the kavun-gen of that same Kavun version — `go run github.com/jokruger/kavun/cmd/kavun-gen ./...` from the module, not an installed binary of another version", root, err)
		}
		opts.Loaded = append(opts.Loaded, ref)
	}
	for _, w := range cfg.With {
		if !filepath.IsAbs(w) {
			w = filepath.Join(cfg.Dir, w)
		}
		ref, err := reference.ReadFile(w)
		if err != nil {
			return fmt.Errorf("-with: %v", err)
		}
		opts.Loaded = append(opts.Loaded, ref)
	}
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
	case cfg.Reference != nil && *cfg.Reference == "":
	case !isModule:
		fmt.Fprintln(cfg.Stderr, "kavun-gen: kavun_reference.json not generated: not in a Go module")
	case !covered:
		fmt.Fprintln(cfg.Stderr, "kavun-gen: kavun_reference.json not generated: the packages do not cover the whole module (run it on ./... from the module root)")
	default:
		opts.Origin = modPath
		opts.Reference = filepath.Join(modDir, reference.FileName)
		if cfg.Reference != nil {
			opts.Reference = *cfg.Reference
			if !filepath.IsAbs(opts.Reference) {
				opts.Reference = filepath.Join(cfg.Dir, opts.Reference)
			}
		}
		opts.DocsURL = cfg.DocsURL
		opts.ReferenceExplicit = cfg.Reference != nil
	}
	// messages name files relative to where the tool runs — when that is the process's working directory, so the
	// relative paths still resolve
	if wd, err := os.Getwd(); err == nil && sameDir(wd, cfg.Dir) {
		for i, d := range dirs {
			if rel, err := filepath.Rel(wd, d); err == nil {
				dirs[i] = rel
			}
		}
		if filepath.IsAbs(opts.Reference) {
			if rel, err := filepath.Rel(wd, opts.Reference); err == nil {
				opts.Reference = rel
			}
		}
	}

	files, err := Generate(ctx, dirs, opts)
	if err != nil {
		return err
	}
	if opts.ReferenceExplicit && !slices.ContainsFunc(files, func(f File) bool { return f.Path == opts.Reference }) {
		fmt.Fprintf(cfg.Stderr, "kavun-gen: %s not generated: the module declares nothing\n", opts.Reference)
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
