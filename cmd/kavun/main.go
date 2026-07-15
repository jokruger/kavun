package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/vm"
)

const sourceFileExt = ".kvn"

var (
	compileOutput string
	showHelp      bool
	showVersion   bool
	resolvePath   bool
	strictAssign  bool
	o0            bool
	o1            bool
	o2            bool
	o3            bool
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
	flag.BoolVar(&o0, "O0", false, "Optimization level 0")
	flag.BoolVar(&o1, "O1", false, "Optimization level 1")
	flag.BoolVar(&o2, "O2", false, "Optimization level 2")
	flag.BoolVar(&o3, "O3", false, "Optimization level 3")
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

	oc := compiler.O0()
	if o1 {
		oc = compiler.O1()
	} else if o2 {
		oc = compiler.O2()
	} else if o3 {
		oc = compiler.O3()
	}

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
		err := CompileOnly(inputData, inputFile, compileOutput, oc)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	} else if filepath.Ext(inputFile) == sourceFileExt {
		err := CompileAndRun(inputData, inputFile, oc)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	} else {
		if err := RunCompiled(inputData); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
}

// CompileOnly compiles the source code and writes the compiled binary into outputFile.
func CompileOnly(data []byte, inputFile, outputFile string, oc *compiler.OptimizationConfig) (err error) {
	bytecode, err := compileSrc(data, inputFile, oc)
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
func CompileAndRun(data []byte, inputFile string, oc *compiler.OptimizationConfig) (err error) {
	bytecode, err := compileSrc(data, inputFile, oc)
	if err != nil {
		return
	}

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	machine.Reset(bytecode, nil)
	err = machine.Run()

	return
}

// RunCompiled reads the compiled binary from file and executes it.
func RunCompiled(data []byte) (err error) {
	bytecode := &vm.Bytecode{}
	err = bytecode.Decode(bytes.NewReader(data))
	if err != nil {
		return
	}

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	machine.Reset(bytecode, nil)
	err = machine.Run()

	return
}

func compileSrc(src []byte, inputFile string, oc *compiler.OptimizationConfig) (*vm.Bytecode, error) {
	fileSet := ast.NewFileSet()
	srcFile := fileSet.AddFile(filepath.Base(inputFile), -1, len(src))

	c := compiler.NewCompiler(oc, nil, srcFile, nil, nil, nil, nil)
	if strictAssign {
		c.SetAssignmentMode(compiler.AssignmentModeStrict)
	}
	c.EnableFileImport(true)
	if resolvePath {
		c.SetImportDir(filepath.Dir(inputFile))
	}

	if err := c.Compile(srcFile, src, nil); err != nil {
		return nil, err
	}

	return c.Bytecode(), nil
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
	fmt.Println("   -OX       optimization level (X = 0, 1, 2, 3)")
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
