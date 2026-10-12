package stdlib

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// The io module is the script's console: it prints to the VM's stdout and reads from its stdin (vm.VM.SetStdout,
// vm.VM.SetStdin), never to the process streams directly.
func init() {
	m := core.NewModule(value.ModuleIo, "io", false)
	m.Fn(members.Print, 0, ioPrint, 0, core.Variadic)
	m.Fn(members.Write, 1, ioWrite, 0, core.Variadic)
	m.Fn(members.ReadLine, 2, ioReadLine, 0, core.Variadic)
	m.Fn(members.Read, 3, ioRead, 0, core.Variadic)
	// 4..127 reserved
	if err := m.Register(); err != nil {
		panic(err)
	}
}

// print(values...): the line form — values separated by spaces, then a newline.
func ioPrint(vm core.VM, args []core.Value) (core.Value, error) {
	return done("io.print", writeOut(vm, fmt.Sprintln(printArgs(args)...)))
}

// write(values...): the raw form — values concatenated, no separator, no newline.
func ioWrite(vm core.VM, args []core.Value) (core.Value, error) {
	return done("io.write", writeOut(vm, fmt.Sprint(printArgs(args)...)))
}

func writeOut(vm core.VM, s string) error {
	_, err := io.WriteString(vm.Stdout(), s)
	return err
}

// read_line([default]): one line without its terminator (\n or \r\n). At end of input it raises, unless a default
// is given — then the default is answered as-is. ("" is a valid line, so end of input cannot answer it.)
func ioReadLine(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) > 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("io.read_line", "0 or 1", len(args))
	}
	line, err := stdinReader(vm).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return raiseIO("io.read_line", err)
	}
	if err != nil && line == "" { // end of input; a last line without a terminator is still a line
		if len(args) == 1 {
			return args[0], nil
		}
		return raiseIO("io.read_line", errors.New("end of input"))
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return core.NewStringValue(line), nil
}

// read([n]): without n, the rest of the input; with n, the next n symbols, waiting for them unless the input ends
// first. "" only at end of input, so it needs no default. An undecodable octet is one symbol and comes through as
// itself, as in every text conversion.
func ioRead(vm core.VM, args []core.Value) (core.Value, error) {
	br := stdinReader(vm)
	switch len(args) {
	case 0:
		b, err := io.ReadAll(br)
		if err != nil {
			return raiseIO("io.read", err)
		}
		return core.NewStringValue(string(b)), nil
	case 1:
		if args[0].Type != value.Int {
			return core.Undefined, errs.NewInvalidArgumentTypeError("io.read", "first", "int", args[0].TypeName())
		}
		n := int64(args[0].Data)
		if n < 0 {
			return core.Undefined, errs.NewInvalidValueError(fmt.Sprintf("(io.read) count must not be negative, got %d", n))
		}
		s, err := readSymbols(br, n)
		if err != nil {
			return raiseIO("io.read", err)
		}
		return core.NewStringValue(s), nil
	default:
		return core.Undefined, errs.NewWrongNumArgumentsError("io.read", "0 or 1", len(args))
	}
}

// readSymbols reads up to n symbols, fewer only at end of input. It grows with what arrives, so a large n is a
// bound, not an allocation.
func readSymbols(br *bufio.Reader, n int64) (string, error) {
	var b strings.Builder
	for i := int64(0); i < n; i++ {
		r, size, err := br.ReadRune()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if r == utf8.RuneError && size == 1 { // an undecodable octet: keep the octet, not U+FFFD
			if err := br.UnreadRune(); err != nil {
				return "", err
			}
			c, err := br.ReadByte()
			if err != nil {
				return "", err
			}
			b.WriteByte(c)
			continue
		}
		b.WriteRune(r)
	}
	return b.String(), nil
}

// stdinReader is the VM's buffered stdin; a VM that hands out a plain reader gets a buffer for this read only.
func stdinReader(vm core.VM) *bufio.Reader {
	r := vm.Stdin()
	if br, ok := r.(*bufio.Reader); ok {
		return br
	}
	return bufio.NewReader(r)
}

// printArgs renders each argument the way print shows it: text as its content, everything else as its render.
func printArgs(args []core.Value) []any {
	out := make([]any, 0, len(args))
	for _, arg := range args {
		switch arg.Type {
		case value.Undefined, value.Bytes, value.Array, value.Record, value.Dict, value.IntRange:
			out = append(out, arg.String())

		default:
			s, ok := arg.AsString()
			if !ok {
				s = arg.String()
			}
			out = append(out, s)
		}
	}
	return out
}
