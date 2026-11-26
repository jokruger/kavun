package kavungen

import (
	"bytes"
	"fmt"
	"go/build"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jokruger/kavun/internal/genfile"
	"github.com/jokruger/kavun/reference"
)

// File is one generated file: a kavun_gen.go (the caller's directory joined with GeneratedFile) or the reference
// (Options.Reference). A nil Content means the file must not exist: its package declares nothing any more (a stale
// kavun_gen.go), or the module declares nothing to put in a reference; each is to be deleted.
type File = genfile.File

// Options configures a generation beyond the packages it reads.
type Options struct {
	// Root is the module root: the reference's positions name their files relative to it. "" leaves the package
	// directories as given.
	Root string
	// Reference is the path of the reference file (kavun_reference.json) to generate from the packages; "" generates
	// none. Its positions name files relative to Root, so the packages must be the whole module rooted there.
	Reference string
	// Origin is the reference's origin: the module path of the module the packages make up.
	Origin string
	// DocsURL is the reference's docs: the URL of the module's published docs root; "" for none.
	DocsURL string
	// ReferenceExplicit reports that Reference was named by the caller (-reference), not the default location: when
	// the packages declare nothing, the file there is left alone instead of deleted.
	ReferenceExplicit bool
	// Loaded holds the references of other modules the packages' links may target: Kavun's for an embedder's run,
	// and those given with -with. They are merged with the run's own (reference.Merge: a name declared twice is
	// refused) and never written.
	Loaded []*reference.Reference
}

// Generate reads the packages in dirs and answers the kavun_gen.go of every package that declares something or still
// holds one, and with opts.Reference the reference file. Every problem in every package is reported in one error, each line naming
// file:line. Nothing is written.
//
// The member names the packages declare by //kavun:names are collected from all of them before any table is checked:
// a package's members may use the names another package of the run declares.
func Generate(ctx *Context, dirs []string, opts Options) ([]File, error) {
	var out []File
	var errs errorList
	var models []*packageModel

	// pass 1: read and scan every package, and collect the run's member names
	var pkgs []*scannedDir
	for _, dir := range dirs {
		files, hasGenerated, problems := readPackage(dir)
		if len(problems) > 0 {
			errs = append(errs, problems...)
			continue
		}
		scanned, problems := scanPackage(dir, files)
		errs = append(errs, problems...)
		pkgs = append(pkgs, &scannedDir{scanned, hasGenerated})
	}
	scanned := make([]*scannedPackage, len(pkgs))
	for i, p := range pkgs {
		scanned[i] = p.pkg
	}
	names := collectNames(ctx, scanned, &errs)

	// pass 2: model every package, then refuse an import cycle the generated files would make
	built := map[*scannedDir]*packageModel{}
	var all []*packageModel
	for _, p := range pkgs {
		model, problems := buildModel(ctx, p.pkg, names)
		errs = append(errs, problems...)
		if len(problems) == 0 {
			built[p] = model
			all = append(all, model)
		}
	}
	checkNameCycles(all, &errs)
	if err := errs.err(); err != nil {
		return nil, err
	}

	// pass 3: emit every package
	importPaths := map[string]string{}
	for _, p := range pkgs {
		model := built[p]
		f, problems := emitScanned(ctx, p, model, importPaths)
		errs = append(errs, problems...)
		if f != nil {
			out = append(out, *f)
		}
		if !model.empty() {
			rel, err := relDir(opts.Root, p.pkg.dir)
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
	// the reference is built for every run, its link targets checked; only a run given opts.Reference — one over
	// the whole module — writes it
	var ref *reference.Reference
	if len(models) > 0 {
		var problems errorList
		ref, problems = buildReference(models, opts.Origin, ctx.coreVersion, opts.DocsURL, opts.Reference == "", opts.Loaded)
		if err := problems.err(); err != nil {
			return nil, err
		}
	}
	if opts.Reference != "" {
		f, err := referenceFile(ref, opts.Reference)
		if err != nil {
			return nil, err
		}
		if f.Content != nil || !opts.ReferenceExplicit {
			out = append(out, f)
		}
	}
	return out, nil
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

// scannedDir is one package directory of a run, scanned.
type scannedDir struct {
	pkg          *scannedPackage
	hasGenerated bool // a kavun_gen.go is there
}

// generateDir generates one package directory on its own (its //kavun:names the only ones): a nil file when it
// neither declares anything nor holds a kavun_gen.go. The model is nil when the package could not be read.
func generateDir(ctx *Context, dir string) (*File, *packageModel, errorList) {
	files, hasGenerated, errs := readPackage(dir)
	if len(errs) > 0 {
		return nil, nil, errs
	}
	scanned, errs := scanPackage(dir, files)
	names := collectNames(ctx, []*scannedPackage{scanned}, &errs)
	if len(errs) > 0 {
		return nil, nil, errs
	}
	model, errs := buildModel(ctx, scanned, names)
	if len(errs) > 0 {
		return nil, nil, errs
	}
	f, errs := emitScanned(ctx, &scannedDir{scanned, hasGenerated}, model, nil)
	return f, model, errs
}

// emitScanned generates the kavun_gen.go of one modelled package directory: a nil file when it neither declares
// anything nor holds one. importPaths caches the import path of each package directory whose names another package
// uses (resolved by `go list`).
func emitScanned(ctx *Context, p *scannedDir, model *packageModel, importPaths map[string]string) (*File, errorList) {
	var errs errorList
	dir := p.pkg.dir
	var content []byte
	if !model.emptyGo() {
		for d := range model.foreign {
			if _, ok := importPaths[d]; ok {
				continue
			}
			path, err := importPath(d)
			if err != nil {
				errs = append(errs, err.Error())
				return nil, errs
			}
			importPaths[d] = path
		}
		var err error
		if content, err = emitGo(ctx, model, importPaths); err != nil {
			errs = append(errs, err.Error())
			return nil, errs
		}
	}
	if content == nil && !p.hasGenerated {
		return nil, nil
	}
	return &File{Path: filepath.Join(dir, GeneratedFile), Content: content}, nil
}

// checkNameCycles refuses packages whose tables use each other's //kavun:names, directly or through others: each
// generated file imports the packages whose names it uses, so the imports would form a cycle. The cycle is reported
// once, at the directive of its first package (in directory order) that uses the next one's names.
func checkNameCycles(models []*packageModel, errs *errorList) {
	byDir := map[string]*packageModel{}
	for _, m := range models {
		byDir[m.dir] = m
	}
	const (
		unseen = iota
		open
		done
	)
	state := map[string]int{}
	var stack []string
	var visit func(dir string)
	visit = func(dir string) {
		state[dir] = open
		stack = append(stack, dir)
		m := byDir[dir]
		for _, next := range slices.Sorted(maps.Keys(m.foreign)) {
			if byDir[next] == nil {
				continue
			}
			switch state[next] {
			case unseen:
				visit(next)
			case open:
				cycle := append(slices.Clone(stack[slices.Index(stack, next):]), next)
				names := make([]string, len(cycle))
				for i, d := range cycle {
					names[i] = byDir[d].pkgName
				}
				from := byDir[cycle[0]]
				errs.add(from.foreign[cycle[1]], "package %s uses the //kavun:names of package %s, and the packages' names are used in a cycle (%s): the generated files would import each other; declare the names in one package",
					from.pkgName, names[1], strings.Join(names, " → "))
			}
		}
		stack = stack[:len(stack)-1]
		state[dir] = done
	}
	dirs := slices.Sorted(maps.Keys(byDir))
	for _, d := range dirs {
		if state[d] == unseen {
			visit(d)
		}
	}
}

// importPath answers the import path of the package in dir, by `go list`.
func importPath(dir string) (string, error) {
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}", ".")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s: the import path of the package: %v\n%s", filepath.ToSlash(dir), err, stderr.String())
	}
	return strings.TrimSpace(string(out)), nil
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
// when the package declares nothing. dir only labels the messages. The model is what the reference emitter reads.
func generateSources(ctx *Context, dir string, files []sourceFile) ([]byte, *packageModel, errorList) {
	scanned, errs := scanPackage(dir, files)
	model, modelErrs := buildModel(ctx, scanned, nil)
	errs = append(errs, modelErrs...)
	if len(errs) > 0 || model.emptyGo() {
		return nil, model, errs
	}
	content, err := emitGo(ctx, model, nil)
	if err != nil {
		errs = append(errs, err.Error())
		return nil, model, errs
	}
	return content, model, nil
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

// MainModule answers the path and root directory of the main module seen from dir: the module whose go.mod holds
// dir (`go env GOMOD`), also in a workspace of several; ok is false outside a module.
func MainModule(dir string) (modPath, modDir string, ok bool) {
	gomod, err := goOutput(dir, "env", "GOMOD")
	if err != nil || gomod == "" || gomod == os.DevNull {
		return "", "", false
	}
	modDir = filepath.Dir(gomod)
	out, err := goOutput(dir, "list", "-m", "-f", "{{.Path}}\t{{.Dir}}")
	if err != nil {
		return "", "", false
	}
	for _, line := range strings.Split(out, "\n") {
		if path, d, found := strings.Cut(line, "\t"); found && sameDir(d, modDir) {
			return path, modDir, true
		}
	}
	return "", "", false
}

// goOutput runs a go command in dir and answers its trimmed standard output.
func goOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// KavunModule is the module path of Kavun itself.
const KavunModule = reference.KavunOrigin

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
