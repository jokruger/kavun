package stdlib

import (
	"fmt"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/formatter"
	"github.com/jokruger/gs/value"
)

var fmtModule = map[string]core.Value{
	"print":   value.NewStaticBuiltinFunction("print", fmtPrint, 0, true),
	"printf":  value.NewStaticBuiltinFunction("printf", fmtPrintf, 1, true),
	"println": value.NewStaticBuiltinFunction("println", fmtPrintln, 0, true),
	"sprintf": value.NewStaticBuiltinFunction("sprintf", fmtSprintf, 1, true),
}

func fmtPrint(vm core.VM, args ...core.Value) (core.Value, error) {
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return core.NewUndefined(), err
	}
	_, _ = fmt.Print(printArgs...)
	return core.NewUndefined(), nil
}

func fmtPrintf(vm core.VM, args ...core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs == 0 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("fmt.printf", "at least 1", numArgs)
	}

	format, ok := args[0].AsString()
	if !ok {
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("fmt.printf", "format", "string", args[0].TypeName())
	}
	if numArgs == 1 {
		fmt.Print(format)
		return core.NewUndefined(), nil
	}

	s, err := formatter.Format(format, args[1:]...)
	if err != nil {
		return core.NewUndefined(), err
	}
	fmt.Print(s)
	return core.NewUndefined(), nil
}

func fmtPrintln(vm core.VM, args ...core.Value) (core.Value, error) {
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return core.NewUndefined(), err
	}
	_, _ = fmt.Println(printArgs...)
	return core.NewUndefined(), nil
}

func fmtSprintf(vm core.VM, args ...core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs == 0 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("fmt.sprintf", "at least 1", numArgs)
	}

	format, ok := args[0].AsString()
	if !ok {
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("fmt.sprintf", "format", "string", args[0].TypeName())
	}
	if numArgs == 1 {
		t := vm.Allocator().NewString(format)
		return core.NewObject(t, false), nil
	}
	s, err := formatter.Format(format, args[1:]...)
	if err != nil {
		return core.NewUndefined(), err
	}
	t := vm.Allocator().NewString(s)
	return core.NewObject(t, false), nil
}

func getPrintArgs(args ...core.Value) ([]any, error) {
	l := 0
	printArgs := make([]any, 0, len(args))
	for _, arg := range args {
		// TODO: shell we check if arg cannot be converted to string?
		s, _ := arg.AsString()
		slen := len(s)
		// make sure length does not exceed the limit
		if l+slen > core.MaxStringLen {
			return nil, core.NewStringLimitError("fmt.print/println")
		}
		l += slen
		printArgs = append(printArgs, s)
	}
	return printArgs, nil
}
