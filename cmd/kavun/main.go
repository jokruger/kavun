package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/vm"
)

const sourceFileExt = ".kvn"

var (
	compileOutput string
	showHelp      bool
	showVersion   bool
	resolvePath   bool
	strictAssign  bool
	version       = "dev"
	commit        = "none"
	date          = "unknown"
)

func init() {
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.StringVar(&compileOutput, "o", "", "Compile output file")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&resolvePath, "resolve", false, "Resolve relative import paths")
	flag.BoolVar(&strictAssign, "strict-assign", false, "Require variables to be declared before '=' assignment")
	flag.Parse()
}

func main() {
	if showHelp {
		doHelp()
		os.Exit(2)
	} else if showVersion {
		ver := "Kavun " + version
		if date != "unknown" {
			ver += " " + date
		}
		if commit != "none" {
			ver += " " + commit
		}
		fmt.Println(ver)
		return
	}

	a := core.NewArena(nil)
	inputFile := flag.Arg(0)
	if inputFile == "" {
		fmt.Fprintln(os.Stderr, "No input file specified")
		return
	}

	inputData, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %s\n", err.Error())
		os.Exit(1)
	}

	inputFile, err = filepath.Abs(inputFile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error file path: %s\n", err)
		os.Exit(1)
	}

	if len(inputData) > 1 && string(inputData[:2]) == "#!" {
		copy(inputData, "//")
	}

	if compileOutput != "" {
		err := CompileOnly(a, inputData, inputFile, compileOutput)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	} else if filepath.Ext(inputFile) == sourceFileExt {
		err := CompileAndRun(a, inputData, inputFile)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	} else {
		if err := RunCompiled(a, inputData); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
}

// CompileOnly compiles the source code and writes the compiled binary into outputFile.
func CompileOnly(a *core.Arena, data []byte, inputFile, outputFile string) (err error) {
	bytecode, err := compileSrc(a, data, inputFile)
	if err != nil {
		return
	}

	if outputFile == "" {
		outputFile = basename(inputFile) + ".out"
	}

	out, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			_ = out.Close()
		} else {
			err = out.Close()
		}
	}()

	err = bytecode.Encode(out)
	if err != nil {
		return
	}
	fmt.Println(outputFile)
	return
}

// CompileAndRun compiles the source code and executes it.
func CompileAndRun(a *core.Arena, data []byte, inputFile string) (err error) {
	bytecode, err := compileSrc(a, data, inputFile)
	if err != nil {
		return
	}

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	machine.Reset(a, bytecode, nil)
	err = machine.Run()

	return
}

// RunCompiled reads the compiled binary from file and executes it.
func RunCompiled(a *core.Arena, data []byte) (err error) {
	bytecode := &vm.Bytecode{}
	err = bytecode.Decode(a, bytes.NewReader(data))
	if err != nil {
		return
	}

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	machine.Reset(a, bytecode, nil)
	err = machine.Run()

	return
}

func compileSrc(a *core.Arena, src []byte, inputFile string) (*vm.Bytecode, error) {
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile(filepath.Base(inputFile), -1, len(src))

	p := parser.NewParser(srcFile, src, nil)
	file, err := p.ParseFile()
	if err != nil {
		return nil, err
	}

	c := compiler.New(a, srcFile, nil, nil, nil, nil)
	if strictAssign {
		c.SetAssignmentMode(compiler.AssignmentModeStrict)
	}
	c.EnableFileImport(true)
	if resolvePath {
		c.SetImportDir(filepath.Dir(inputFile))
	}

	if err := c.Compile(file); err != nil {
		return nil, err
	}

	bytecode := c.Bytecode()
	if err := bytecode.RemoveDuplicates(a); err != nil {
		return nil, err
	}
	return bytecode, nil
}

func doHelp() {
	fmt.Println("Usage:")
	fmt.Println()
	fmt.Println("	kavun [flags] {input-file}")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println()
	fmt.Println("	-o        compile output file")
	fmt.Println("	-strict-assign  require variables to be declared before '=' assignment")
	fmt.Println("	-version  show version")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println()
	fmt.Println("	kavun")
	fmt.Println()
	fmt.Println("	kavun myapp.kvn")
	fmt.Println()
	fmt.Println("	          Compile and run source file (myapp.kvn)")
	fmt.Println("	          Source file must have .kvn extension")
	fmt.Println()
	fmt.Println("	kavun -o myapp myapp.kvn")
	fmt.Println()
	fmt.Println("	          Compile source file (myapp.kvn) into bytecode file (myapp)")
	fmt.Println()
	fmt.Println("	kavun myapp")
	fmt.Println()
	fmt.Println("	          Run bytecode file (myapp)")
	fmt.Println()
	fmt.Println()
}

func basename(s string) string {
	s = filepath.Base(s)
	n := strings.LastIndexByte(s, '.')
	if n > 0 {
		return s[:n]
	}
	return s
}
