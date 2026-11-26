// Command kavun-gen generates the kavun_gen.go file of every package that declares Kavun native functions with
// `//kavun:` directives, and the module's reference file kavun_reference.json (the language reference as data, see
// package reference) from their doc blocks (see internal/kavungen). The Markdown reference pages are rendered from
// that file by cmd/kavun-docs.
//
//	kavun-gen [-check] [-reference <path>] [-docs-url <url>] [-with <kavun_reference.json>]… <packages…>
//
// The packages are Go package patterns ("./..."); packages under the repository's scratch folder <kavun root>/tmp/
// are skipped. With -check nothing is written: it reports every generated file that differs from the one on disk
// (or that is stale) and exits 1. Any problem in a directive stops generation, names its file:line, and exits 1.
//
// The reference file describes every type, builtin function and module the main module declares, so it is generated
// only when the patterns cover all of the module's packages: at <module root>/kavun_reference.json, or at the
// -reference path; -reference "" generates none. A run over part of the module generates the tables only and says
// so on stderr; with an explicit -reference and nothing declared, the file there is left alone. -docs-url gives the
// reference's docs field, the URL of the module's published docs root; without it the reference has none. The file
// depends on the sources and the flags only, so a -check run takes the flags of the generating run (Kavun's:
// `just generate-check`).
//
// Every kavun: link of a doc block (but kavun:doc/, a guide page checked by kavun-docs) must name something declared
// by the run or by a reference it loads: when the main module is not Kavun, Kavun's own kavun_reference.json, read
// from the Kavun module the run builds against (`go list -m github.com/jokruger/kavun`), and each -with file
// (repeatable), the reference of another module the links may target. A type, module or builtin function declared
// by two of them is refused.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jokruger/kavun/internal/kavungen"
)

func main() {
	check := flag.Bool("check", false, "compare the generated files with those on disk instead of writing them")
	ref := flag.String("reference", "", "the path of the reference file; \"\" for none (default: <module root>/kavun_reference.json when generating the whole module)")
	docsURL := flag.String("docs-url", "", "the URL of the module's published docs root, written into the reference file's docs field (none without it)")
	var with withFlag
	flag.Var(&with, "with", "a `kavun_reference.json` of another module whose declarations the links may target (repeatable); Kavun's own is loaded without it")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: kavun-gen [-check] [-reference <path>] [-docs-url <url>] [-with <kavun_reference.json>]… <packages…>")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	refSet := false
	flag.Visit(func(f *flag.Flag) {
		refSet = refSet || f.Name == "reference"
	})
	if !refSet {
		ref = nil // the default: the module root, when the packages cover the module
	}

	cwd, err := os.Getwd()
	if err == nil {
		err = kavungen.Run(kavungen.RunConfig{Dir: cwd, Check: *check, Reference: ref, DocsURL: *docsURL, With: with, Patterns: flag.Args(), Stdout: os.Stdout, Stderr: os.Stderr})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "kavun-gen:", err)
		os.Exit(1)
	}
}

// withFlag collects the repeated -with values.
type withFlag []string

func (w *withFlag) String() string { return strings.Join(*w, " ") }

func (w *withFlag) Set(s string) error {
	if s == "" {
		return fmt.Errorf("no reference file")
	}
	*w = append(*w, s)
	return nil
}
