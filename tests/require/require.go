package require

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/token"
	"github.com/jokruger/kavun/vm"
)

// NoError asserts err is not an error.
func NoError(t *testing.T, err error, msg ...any) {
	if err != nil {
		failExpectedActual(t, "no error", err, msg...)
	}
}

// Error asserts err is an error.
func Error(t *testing.T, err error, msg ...any) {
	if err == nil {
		failExpectedActual(t, "error", err, msg...)
	}
}

// Nil asserts v is nil.
func Nil(t *testing.T, v any, msg ...any) {
	if !isNil(v) {
		failExpectedActual(t, "nil", v, msg...)
	}
}

// True asserts v is true.
func True(t *testing.T, v bool, msg ...any) {
	if !v {
		failExpectedActual(t, "true", v, msg...)
	}
}

// False asserts vis false.
func False(t *testing.T, v bool, msg ...any) {
	if v {
		failExpectedActual(t, "false", v, msg...)
	}
}

// NotNil asserts v is not nil.
func NotNil(t *testing.T, v any, msg ...any) {
	if isNil(v) {
		failExpectedActual(t, "not nil", v, msg...)
	}
}

// IsType asserts expected and actual are of the same type.
func IsType(t *testing.T, e, a any, msg ...any) {
	switch e := e.(type) {
	case core.Value:
		if a, ok := a.(core.Value); ok {
			if a.Type == e.Type {
				return
			}
			if a.Type == core.VT_MAP || a.Type == core.VT_RECORD {
				if e.Type == core.VT_MAP || e.Type == core.VT_RECORD {
					return
				}
			}
			failExpectedActual(t, e.TypeName(), a.TypeName(), msg...)
		} else {
			failExpectedActual(t, e.TypeName(), reflect.TypeOf(a), msg...)
		}

	case *core.Map:
		if _, ok := a.(*core.Map); ok {
			return
		}
	}

	// test other types as normal
	if reflect.TypeOf(e) != reflect.TypeOf(a) {
		failExpectedActual(t, reflect.TypeOf(e), reflect.TypeOf(a), msg...)
	}
}

// Equal asserts expected and actual are equal.
func Equal(t *testing.T, expected, actual any, msg ...any) {
	e := expected
	a := actual

	if isNil(e) {
		Nil(t, a, "expected nil, but got not nil")
		return
	}
	NotNil(t, a, "expected not nil, but got nil")
	IsType(t, e, a, msg...)

	switch e := e.(type) {
	case int:
		if e != a.(int) {
			failExpectedActual(t, e, a, msg...)
		}

	case int64:
		if e != a.(int64) {
			failExpectedActual(t, e, a, msg...)
		}

	case float64:
		if e != a.(float64) {
			failExpectedActual(t, e, a, msg...)
		}

	case string:
		if e != a.(string) {
			failExpectedActual(t, e, a, msg...)
		}

	case []byte:
		if !bytes.Equal(e, a.([]byte)) {
			failExpectedActual(t, string(e), string(a.([]byte)), msg...)
		}

	case []string:
		if !equalStringSlice(e, a.([]string)) {
			failExpectedActual(t, e, a, msg...)
		}

	case []int:
		if !equalIntSlice(e, a.([]int)) {
			failExpectedActual(t, e, a, msg...)
		}

	case bool:
		if e != a.(bool) {
			failExpectedActual(t, e, a, msg...)
		}

	case rune:
		if e != a.(rune) {
			failExpectedActual(t, e, a, msg...)
		}

	case *vm.Symbol:
		if !equalSymbol(e, a.(*vm.Symbol)) {
			failExpectedActual(t, e, a, msg...)
		}

	case core.Pos:
		if e != a.(core.Pos) {
			failExpectedActual(t, e, a, msg...)
		}

	case token.Token:
		if e != a.(token.Token) {
			failExpectedActual(t, e, a, msg...)
		}

	case []core.Value:
		equalObjectSlice(t, e, a.([]core.Value), msg...)

	case *core.String:
		if e.Value != string(a.(*core.String).Value) {
			failExpectedActual(t, e.Value, string(a.(*core.String).Value), msg...)
		}

	case *core.Array:
		equalObjectSlice(t, e.Elements, a.(*core.Array).Elements, msg...)

	case *core.Bytes:
		if !bytes.Equal(e.Elements, a.(*core.Bytes).Elements) {
			failExpectedActual(t, string(e.Elements), string(a.(*core.Bytes).Elements), msg...)
		}

	case *core.Map:
		if a, ok := a.(*core.Map); ok {
			equalObjectMap(t, e.Elements, a.Elements, msg...)
		}

	case *core.CompiledFunction:
		switch a := a.(type) {
		case *core.CompiledFunction:
			equalCompiledFunction(t, e, a, msg...)
			return
		case core.Value:
			if a.Type == core.VT_COMPILED_FUNCTION {
				equalCompiledFunction(t, e, (*core.CompiledFunction)(a.Ptr), msg...)
			} else {
				failExpectedActual(t, "compiled function", a.TypeName(), msg...)
			}
		default:
			failExpectedActual(t, "compiled function", reflect.TypeOf(a), msg...)
		}

	case core.Value:
		if e.Type == core.VT_COMPILED_FUNCTION {
			switch a := a.(type) {
			case *core.CompiledFunction:
				equalCompiledFunction(t, (*core.CompiledFunction)(e.Ptr), a, msg...)
			case core.Value:
				if a.Type == core.VT_COMPILED_FUNCTION {
					equalCompiledFunction(t, (*core.CompiledFunction)(e.Ptr), (*core.CompiledFunction)(a.Ptr), msg...)
				} else {
					failExpectedActual(t, "compiled function", a.TypeName(), msg...)
				}
			default:
				failExpectedActual(t, "compiled function", reflect.TypeOf(a), msg...)
			}
			return
		}
		if !e.Equal(a.(core.Value)) {
			failExpectedActual(t, e, a, msg...)
		}

	case *parser.SourceFileSet:
		equalFileSet(t, e, a.(*parser.SourceFileSet), msg...)

	case *parser.SourceFile:
		Equal(t, e.Name, a.(*parser.SourceFile).Name, msg...)
		Equal(t, e.Base, a.(*parser.SourceFile).Base, msg...)
		Equal(t, e.Size, a.(*parser.SourceFile).Size, msg...)
		True(t, equalIntSlice(e.Lines, a.(*parser.SourceFile).Lines), msg...)

	case error:
		if e != a.(error) {
			failExpectedActual(t, e, a, msg...)
		}

	case time.Time:
		if !e.Equal(a.(time.Time)) {
			failExpectedActual(t, e, a, msg...)
		}

	default:
		panic(fmt.Errorf("type not implemented: %T", e))
	}
}

// Fail marks the function as having failed but continues execution.
func Fail(t *testing.T, msg ...any) {
	t.Logf("\nError trace:\n\t%s\n%s", strings.Join(errorTrace(), "\n\t"), message(msg...))
	t.Fail()
}

func failExpectedActual(t *testing.T, expected, actual any, msg ...any) {
	var addMsg string
	if len(msg) > 0 {
		addMsg = "\nMessage:  " + message(msg...)
	}

	t.Logf("\nError trace:\n\t%s\nExpected: %v\nActual:   %v%s", strings.Join(errorTrace(), "\n\t"), expected, actual, addMsg)
	t.FailNow()
}

func message(formatArgs ...any) string {
	var format string
	var args []any
	if len(formatArgs) > 0 {
		format = formatArgs[0].(string)
	}
	if len(formatArgs) > 1 {
		args = formatArgs[1:]
	}
	return fmt.Sprintf(format, args...)
}

func equalIntSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalSymbol(a, b *vm.Symbol) bool {
	return a.Name == b.Name &&
		a.Index == b.Index &&
		a.Scope == b.Scope
}

func equalObjectSlice(t *testing.T, expected, actual []core.Value, msg ...any) {
	Equal(t, len(expected), len(actual), msg...)
	for i := 0; i < len(expected); i++ {
		Equal(t, expected[i], actual[i], msg...)
	}
}

func equalFileSet(t *testing.T, expected, actual *parser.SourceFileSet, msg ...any) {
	Equal(t, len(expected.Files), len(actual.Files), msg...)
	for i, f := range expected.Files {
		Equal(t, f, actual.Files[i], msg...)
	}
	Equal(t, expected.Base, actual.Base)
	Equal(t, expected.LastFile, actual.LastFile)
}

func equalObjectMap(t *testing.T, expected, actual map[string]core.Value, msg ...any) {
	Equal(t, len(expected), len(actual), msg...)
	for key, expectedVal := range expected {
		actualVal := actual[key]
		Equal(t, expectedVal, actualVal, msg...)
	}
}

func equalCompiledFunction(t *testing.T, expected, actual *core.CompiledFunction, msg ...any) {
	Equal(t, vm.FormatInstructions(expected.Instructions, 0), vm.FormatInstructions(actual.Instructions, 0), msg...)
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	value := reflect.ValueOf(v)
	kind := value.Kind()
	return kind >= reflect.Chan && kind <= reflect.Slice && value.IsNil()
}

func errorTrace() []string {
	var pc uintptr
	file := ""
	line := 0
	var ok bool
	name := ""

	var callers []string
	for i := 0; ; i++ {
		pc, file, line, ok = runtime.Caller(i)
		if !ok {
			break
		}

		if file == "<autogenerated>" {
			break
		}

		f := runtime.FuncForPC(pc)
		if f == nil {
			break
		}
		name = f.Name()

		if name == "testing.tRunner" {
			break
		}

		parts := strings.Split(file, "/")
		file = parts[len(parts)-1]
		if len(parts) > 1 {
			dir := parts[len(parts)-2]
			if dir != "require" ||
				file == "mock_test.go" {
				callers = append(callers, fmt.Sprintf("%s:%d", file, line))
			}
		}

		// Drop the package
		segments := strings.Split(name, ".")
		name = segments[len(segments)-1]
		if isTest(name, "Test") ||
			isTest(name, "Benchmark") ||
			isTest(name, "Example") {
			break
		}
	}
	return callers
}

func isTest(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) { // "Test" is ok
		return true
	}
	r, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(r)
}
