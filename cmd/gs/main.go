package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jokruger/gs"
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/stdlib"
	"github.com/jokruger/gs/value"
)

const (
	sourceFileExt = ".gs"
	replPrompt    = ">> "
)

var (
	compileOutput string
	showHelp      bool
	showVersion   bool
	resolvePath   bool // TODO Remove this flag at version 3
	version       = "dev"
)

func init() {
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.StringVar(&compileOutput, "o", "", "Compile output file")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&resolvePath, "resolve", false, "Resolve relative import paths")
	flag.Parse()
}

func main() {
	if showHelp {
		doHelp()
		os.Exit(2)
	} else if showVersion {
		fmt.Println(version)
		return
	}

	modules := stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	inputFile := flag.Arg(0)
	if inputFile == "" {
		// REPL
		RunREPL(modules, os.Stdin, os.Stdout)
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
		err := CompileOnly(modules, inputData, inputFile,
			compileOutput)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	} else if filepath.Ext(inputFile) == sourceFileExt {
		err := CompileAndRun(modules, inputData, inputFile)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	} else {
		if err := RunCompiled(modules, inputData); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
}

// CompileOnly compiles the source code and writes the compiled binary into outputFile.
func CompileOnly(modules *gs.ModuleMap, data []byte, inputFile, outputFile string) (err error) {
	bytecode, err := compileSrc(modules, data, inputFile)
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
func CompileAndRun(modules *gs.ModuleMap, data []byte, inputFile string) (err error) {
	bytecode, err := compileSrc(modules, data, inputFile)
	if err != nil {
		return
	}

	machine := gs.NewVM(bytecode, nil, -1)
	err = machine.Run()
	return
}

// RunCompiled reads the compiled binary from file and executes it.
func RunCompiled(modules *gs.ModuleMap, data []byte) (err error) {
	bytecode := &gs.Bytecode{}
	err = bytecode.Decode(bytes.NewReader(data), modules)
	if err != nil {
		return
	}

	machine := gs.NewVM(bytecode, nil, -1)
	err = machine.Run()
	return
}

// RunREPL starts REPL.
func RunREPL(modules *gs.ModuleMap, in io.Reader, out io.Writer) {
	stdin := bufio.NewScanner(in)
	fileSet := parser.NewFileSet()
	globals := make([]core.Object, gs.GlobalsSize)
	symbolTable := gs.NewSymbolTable()
	for idx, fn := range gs.GetAllBuiltinFunctions() {
		symbolTable.DefineBuiltin(idx, fn.Name)
	}

	// embed println function
	symbol := symbolTable.Define("__repl_println__")
	globals[symbol.Index] = &value.UserFunction{
		Name: "println",
		Value: func(args ...core.Object) (ret core.Object, err error) {
			var printArgs []any
			for _, arg := range args {
				if _, isUndefined := arg.(*value.Undefined); isUndefined {
					printArgs = append(printArgs, "<undefined>")
				} else {
					s, _ := arg.ToString()
					printArgs = append(printArgs, s)
				}
			}
			printArgs = append(printArgs, "\n")
			_, _ = fmt.Print(printArgs...)
			return
		},
	}

	var constants []core.Object
	for {
		_, _ = fmt.Fprint(out, replPrompt)
		scanned := stdin.Scan()
		if !scanned {
			return
		}

		line := stdin.Text()
		srcFile := fileSet.AddFile("repl", -1, len(line))
		p := parser.NewParser(srcFile, []byte(line), nil)
		file, err := p.ParseFile()
		if err != nil {
			_, _ = fmt.Fprintln(out, err.Error())
			continue
		}

		file = addPrints(file)
		c := gs.NewCompiler(srcFile, symbolTable, constants, modules, nil)
		if err := c.Compile(file); err != nil {
			_, _ = fmt.Fprintln(out, err.Error())
			continue
		}

		bytecode := c.Bytecode()
		machine := gs.NewVM(bytecode, globals, -1)
		if err := machine.Run(); err != nil {
			_, _ = fmt.Fprintln(out, err.Error())
			continue
		}
		constants = bytecode.Constants
	}
}

func compileSrc(modules *gs.ModuleMap, src []byte, inputFile string) (*gs.Bytecode, error) {
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile(filepath.Base(inputFile), -1, len(src))

	p := parser.NewParser(srcFile, src, nil)
	file, err := p.ParseFile()
	if err != nil {
		return nil, err
	}

	c := gs.NewCompiler(srcFile, nil, nil, modules, nil)
	c.EnableFileImport(true)
	if resolvePath {
		c.SetImportDir(filepath.Dir(inputFile))
	}

	if err := c.Compile(file); err != nil {
		return nil, err
	}

	bytecode := c.Bytecode()
	bytecode.RemoveDuplicates()
	return bytecode, nil
}

func doHelp() {
	fmt.Println("Usage:")
	fmt.Println()
	fmt.Println("	gs [flags] {input-file}")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println()
	fmt.Println("	-o        compile output file")
	fmt.Println("	-version  show version")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println()
	fmt.Println("	gs")
	fmt.Println()
	fmt.Println("	          Start Gs REPL")
	fmt.Println()
	fmt.Println("	gs myapp.gs")
	fmt.Println()
	fmt.Println("	          Compile and run source file (myapp.gs)")
	fmt.Println("	          Source file must have .gs extension")
	fmt.Println()
	fmt.Println("	gs -o myapp myapp.gs")
	fmt.Println()
	fmt.Println("	          Compile source file (myapp.gs) into bytecode file (myapp)")
	fmt.Println()
	fmt.Println("	gs myapp")
	fmt.Println()
	fmt.Println("	          Run bytecode file (myapp)")
	fmt.Println()
	fmt.Println()
}

func addPrints(file *parser.File) *parser.File {
	var stmts []parser.Stmt
	for _, s := range file.Stmts {
		switch s := s.(type) {
		case *parser.ExprStmt:
			stmts = append(stmts, &parser.ExprStmt{
				Expr: &parser.CallExpr{
					Func: &parser.Ident{Name: "__repl_println__"},
					Args: []parser.Expr{s.Expr},
				},
			})
		case *parser.AssignStmt:
			stmts = append(stmts, s)

			stmts = append(stmts, &parser.ExprStmt{
				Expr: &parser.CallExpr{
					Func: &parser.Ident{
						Name: "__repl_println__",
					},
					Args: s.LHS,
				},
			})
		default:
			stmts = append(stmts, s)
		}
	}
	return &parser.File{
		InputFile: file.InputFile,
		Stmts:     stmts,
	}
}

func basename(s string) string {
	s = filepath.Base(s)
	n := strings.LastIndexByte(s, '.')
	if n > 0 {
		return s[:n]
	}
	return s
}
