package io

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
)

// The script's console: print, write and read on the VM's standard streams.
//
// It writes to the VM's stdout and reads from its stdin — the process streams unless the host redirects them
// (`VM.SetStdout`, `VM.SetStdin`, see [embedding.md § Standard streams](../../embedding.md#standard-streams)).
//
// | | line-oriented | raw stream |
// | --- | --- | --- |
// | output | `print(...values)` | `write(...values)` |
// | input | `read_line([default])` | `read([n])` |
//
// Example:
//
// ```go
// io := import("io")
// io.print("sum:", 20 + 22)            // sum: 42
// io.write("a", "b"); io.write("c\n")  // abc
// name := io.read_line("anonymous")    // one line of input, or "anonymous" at end of input
// head := io.read(4)                   // the next 4 symbols ("" at end of input)
// rest := io.read()                    // everything after them ("" when nothing is left)
// ```
//
// Both readers share one input buffer, so `read_line` followed by `read` loses nothing. Octets that are not valid
// UTF-8 come through unchanged; each counts as one symbol, as in every text conversion.
//
// Failures: `io` (an output or input stream the world refused, or end of input for `read_line` without a default);
// `invalid_value` (a negative count); `invalid_argument_type`; `wrong_num_arguments`.
//
//kavun:module io impure

// Print the values separated by spaces, then a newline.
//
// Text prints as its content, everything else as its render.
//
//kavun:fn print slot=0 impure (...values) -> undefined
func ioPrint(vm core.VM, args []core.Value) (core.Value, error) {
	return modkit.Done("io.print", writeOut(vm, fmt.Sprintln(printArgs(args)...)))
}

// Write the values concatenated — no separator, no newline.
//
// The building block for output assembled piece by piece.
//
//kavun:fn write slot=1 impure (...values) -> undefined
func ioWrite(vm core.VM, args []core.Value) (core.Value, error) {
	return modkit.Done("io.write", writeOut(vm, fmt.Sprint(printArgs(args)...)))
}

func writeOut(vm core.VM, s string) error {
	_, err := io.WriteString(vm.Stdout(), s)
	return err
}

// Read one line of input without its terminator (`\n` or `\r\n`).
//
// A last line with no terminator is still a line. At end of input it raises `io` (`(io.read_line) end of input`),
// unless a default is given — then the default is answered as-is. (`""` is a valid line, so end of input cannot answer
// it.)
//
//kavun:fn read_line slot=2 impure ([default]) -> string
func ioReadLine(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) > 1 {
		return core.Undefined, core.ErrArgCount
	}
	line, err := stdinReader(vm).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return modkit.RaiseIO("io.read_line", err)
	}
	if err != nil && line == "" { // end of input; a last line without a terminator is still a line
		if len(args) == 1 {
			return args[0], nil
		}
		return modkit.RaiseIO("io.read_line", errors.New("end of input"))
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return core.NewStringValue(line), nil
}

// Read the rest of the input, or with `n`, the next `n` symbols.
//
// Without `n`, the rest of the input. With `n`, the next `n` symbols — the unit of a string's `len()` and indexing —
// waiting for them unless the input ends first, so it answers fewer only at end of input. `""` only at end of input
// (and for `n == 0`), so a loop `for s := io.read(64); s != ""; s = io.read(64)` ends cleanly. A negative `n` raises
// `invalid_value`; `n` bounds the read, it is not an allocation.
//
//kavun:fn read slot=3 impure ([n]) -> string
func ioRead(vm core.VM, args []core.Value) (core.Value, error) {
	br := stdinReader(vm)
	switch len(args) {
	case 0:
		b, err := io.ReadAll(br)
		if err != nil {
			return modkit.RaiseIO("io.read", err)
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
			return modkit.RaiseIO("io.read", err)
		}
		return core.NewStringValue(s), nil
	default:
		return core.Undefined, core.ErrArgCount
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
