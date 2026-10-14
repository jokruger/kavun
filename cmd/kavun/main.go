package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	// The CLI links Go's embedded IANA timezone database so zone names resolve on hosts that ship none
	// (minimal containers). It is only a fallback: ZONEINFO and the system tzdata are read first. A library
	// does not make this choice for its embedder — Go's time/tzdata documentation says so — hence here only.
	_ "time/tzdata"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	_ "github.com/jokruger/kavun/stdlib" // registers the builtin modules
	"github.com/jokruger/kavun/vm"
)

const sourceFileExt = ".kvn"

// keyFileEnv names the default key file: a compiled file is signed and verified with it unless -key says otherwise.
const keyFileEnv = "KAVUN_KEY_FILE"

var (
	compileOutput string
	keyFile       string
	genKey        bool
	showInfo      bool
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
	flag.StringVar(&keyFile, "key", "", "Key file signing (-o) and verifying compiled files (default $"+keyFileEnv+")")
	flag.BoolVar(&genKey, "genkey", false, "Generate a key file (-o)")
	flag.BoolVar(&showInfo, "info", false, "Print a compiled file's metadata")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&resolvePath, "resolve", false, "Resolve relative import paths")
	flag.BoolVar(&strictAssign, "strict-assign", false, "Require variables to be declared before '=' assignment")
	flag.BoolVar(&o0, "O0", false, "Optimization level 0")
	flag.BoolVar(&o1, "O1", false, "Optimization level 1")
	flag.BoolVar(&o2, "O2", false, "Optimization level 2")
	flag.BoolVar(&o3, "O3", false, "Optimization level 3")
}

func main() {
	flag.Parse()
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

	if genKey {
		if compileOutput == "" {
			fmt.Fprintln(os.Stderr, "-genkey needs an output file: kavun -genkey -o app.key")
			os.Exit(1)
		}
		if err := GenerateKey(compileOutput); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Println(compileOutput)
		return
	}

	signer, err := loadSigner(keyFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
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

	if showInfo {
		f, err := os.Open(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file: %s\n", err.Error())
			os.Exit(1)
		}
		defer f.Close()
		if err := PrintInfo(os.Stdout, f, signer); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
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
		err := CompileOnly(inputData, inputFile, compileOutput, oc, signer)
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
		if err := RunCompiled(inputData, signer); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
}

// CompileOnly compiles the source code and writes the compiled binary into outputFile.
func CompileOnly(data []byte, inputFile, outputFile string, oc *compiler.OptimizationConfig, signer vm.Signer) (err error) {
	bytecode, err := compileSrc(data, inputFile, oc)
	if err != nil {
		return
	}

	if outputFile == "" {
		outputFile = basename(inputFile) + ".out"
	}

	out, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
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

	err = bytecode.Encode(out, signer)
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
	machine.AllowAllModules()
	machine.Reset(bytecode, nil)
	err = machine.Run()

	return
}

// RunCompiled reads the compiled binary from file and executes it.
func RunCompiled(data []byte, signer vm.Signer) (err error) {
	bytecode := &vm.Bytecode{}
	err = bytecode.Decode(bytes.NewReader(data), signer)
	if err != nil {
		return
	}

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	machine.AllowAllModules()
	machine.Reset(bytecode, nil)
	err = machine.Run()

	return
}

// GenerateKey writes a new random signing key — vm.MinHMACKeyLen bytes from crypto/rand, one line of hex — to path,
// readable by its owner only. It never overwrites a file: a replaced key would orphan everything signed with it.
func GenerateKey(path string) error {
	key := make([]byte, vm.MinHMACKeyLen)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("generate key: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}
	if _, err := fmt.Fprintln(f, hex.EncodeToString(key)); err != nil {
		_ = f.Close()
		return fmt.Errorf("generate key: %w", err)
	}
	return f.Close()
}

// loadSigner answers the HMAC signer of the key file at path, or of $KAVUN_KEY_FILE when path is empty; nil when
// neither names one (compiled files are then written unsigned and run unchecked).
func loadSigner(path string) (vm.Signer, error) {
	if path == "" {
		path = os.Getenv(keyFileEnv)
	}
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("key file: %w", err)
	}
	key, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("key file %s: not one line of hex: %w", path, err)
	}
	if len(key) < vm.MinHMACKeyLen {
		return nil, fmt.Errorf("key file %s: key too short: %d bytes (min: %d); generate one with -genkey", path, len(key), vm.MinHMACKeyLen)
	}
	return vm.HMACSigner(key), nil
}

// PrintInfo prints a compiled file's metadata beside this engine's: the versions, whether it is signed (and, with a
// signer, whether the signature verifies), the builtin modules it imports and its binding names. It reports what
// Decode would refuse instead of refusing.
func PrintInfo(w io.Writer, r io.Reader, signer vm.Signer) error {
	info, err := vm.Inspect(r, signer)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "core version:  %s (this engine: %s)\n", info.Core, vm.CoreVersion())
	fmt.Fprintf(w, "host version:  %s (this process: %s)\n", info.Host, core.HostVersion())
	switch {
	case !info.Signed && signer == nil:
		fmt.Fprintln(w, "signed:        no")
	case !info.Signed:
		fmt.Fprintln(w, "signed:        no (refused with this key)")
	case signer == nil:
		fmt.Fprintln(w, "signed:        yes (not checked: no key)")
	case info.Verify == nil:
		fmt.Fprintln(w, "signed:        yes, signature valid")
	default:
		fmt.Fprintln(w, "signed:        yes, signature INVALID")
	}
	if info.Bytecode == nil {
		fmt.Fprintf(w, "payload:       unreadable (%v)\n", info.PayloadErr)
		return nil
	}
	imports := make([]string, 0, len(info.Bytecode.Imports))
	for _, t := range info.Bytecode.Imports {
		if def := core.ModuleByType(t); def != nil {
			imports = append(imports, def.Name)
		} else {
			imports = append(imports, fmt.Sprintf("<unknown module %d>", t))
		}
	}
	fmt.Fprintf(w, "imports:       %s\n", listOrNone(imports))
	fmt.Fprintf(w, "bindings:      %s\n", listOrNone(info.Bytecode.Bindings))
	return nil
}

func listOrNone(names []string) string {
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}

func compileSrc(src []byte, inputFile string, oc *compiler.OptimizationConfig) (*vm.Bytecode, error) {
	fileSet := ast.NewFileSet()
	srcFile := fileSet.AddFile(filepath.Base(inputFile), -1, len(src))

	c := compiler.NewCompiler(oc, nil, srcFile, nil, nil, nil, nil)
	c.AllowAllModules() // the CLI runs the user's own scripts: every builtin module is granted
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
	fmt.Println("	-key      key file: sign the compiled file (-o), verify a compiled file before running it")
	fmt.Println("	          (default: $KAVUN_KEY_FILE; unset: unsigned, and run without a check)")
	fmt.Println("	-genkey   generate a key file (-o), never overwriting one")
	fmt.Println("	-info     print a compiled file's versions, signature, imports and bindings")
	fmt.Println("	-strict-assign  require variables to be declared before '=' assignment")
	fmt.Println("	-version  show version")
	fmt.Println("   -OX       optimization level (X = 0, 1, 2, 3)")
	fmt.Println()
	fmt.Println("All builtin modules are allowed from the CLI.")
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
	fmt.Println("	kavun -genkey -o app.key")
	fmt.Println("	kavun -key app.key -o myapp myapp.kvn")
	fmt.Println("	kavun -key app.key myapp")
	fmt.Println()
	fmt.Println("	          Sign the compiled file, and refuse to run it unless it verifies")
	fmt.Println()
	fmt.Println("	kavun -info [-key app.key] myapp")
	fmt.Println()
	fmt.Println("	          Show why a compiled file is refused (versions, signature)")
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
