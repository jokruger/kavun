// Command kavun-gen generates the kavun_gen.go file of every package that declares Kavun native functions with
// `//kavun:` directives, and the Markdown reference pages under docs/reference/ from their doc blocks (see
// internal/kavungen).
//
//	kavun-gen [-check] [-docs <dir>] <packages…>
//
// The packages are Go package patterns ("./..."); packages under the repository's scratch folder <kavun root>/tmp/
// are skipped. With -check nothing is written: it reports every generated file that differs from the one on disk
// (or that is stale) and exits 1. Any problem in a directive stops generation, names its file:line, and exits 1.
//
// -docs names the directory the reference pages are generated into; -docs "" generates none. By default the pages
// are generated into <module root>/docs/reference only when the main module is Kavun itself and the patterns cover
// every one of its packages. That directory is wholly generated, so a run whose patterns do not cover the whole
// module never reports or deletes a page as stale: with -docs it generates the pages of the packages it read and
// leaves every other file there alone (and says so on stderr); without it, it generates no page.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jokruger/kavun/internal/kavungen"
)

func main() {
	check := flag.Bool("check", false, "compare the generated files with those on disk instead of writing them")
	docs := flag.String("docs", "", "the directory to generate the Markdown reference pages into; \"\" for none (default: docs/reference when generating all of Kavun)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: kavun-gen [-check] [-docs <dir>] <packages…>")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	docsSet := false
	flag.Visit(func(f *flag.Flag) { docsSet = docsSet || f.Name == "docs" })
	if !docsSet {
		docs = nil // the default: decided once the packages are known
	}

	cwd, err := os.Getwd()
	if err == nil {
		err = kavungen.Run(kavungen.RunConfig{Dir: cwd, Check: *check, Docs: docs, Patterns: flag.Args(), Stdout: os.Stdout, Stderr: os.Stderr})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "kavun-gen:", err)
		os.Exit(1)
	}
}
