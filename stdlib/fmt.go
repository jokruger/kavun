package stdlib

import (
	"fmt"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/formatter"
)

var fmtModule = map[string]core.Value{
	"print":   core.NewStaticBuiltinFunction("print", fmtPrint, 0, true),
	"printf":  core.NewStaticBuiltinFunction("printf", fmtPrintf, 1, true),
	"println": core.NewStaticBuiltinFunction("println", fmtPrintln, 0, true),
	"sprintf": core.NewStaticBuiltinFunction("sprintf", fmtSprintf, 1, true),
}

func fmtPrint(vm core.VM, args []core.Value) (core.Value, error) {
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return core.UndefinedValue(), err
	}
	_, _ = fmt.Print(printArgs...)
	return core.UndefinedValue(), nil
}

func fmtPrintf(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs == 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("fmt.printf", "at least 1", numArgs)
	}

	format, ok := args[0].AsString()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("fmt.printf", "format", "string", args[0].TypeName())
	}
	if numArgs == 1 {
		fmt.Print(format)
		return core.UndefinedValue(), nil
	}

	s, err := formatter.Format(format, args[1:]...)
	if err != nil {
		return core.UndefinedValue(), err
	}
	fmt.Print(s)
	return core.UndefinedValue(), nil
}

func fmtPrintln(vm core.VM, args []core.Value) (core.Value, error) {
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return core.UndefinedValue(), err
	}
	_, _ = fmt.Println(printArgs...)
	return core.UndefinedValue(), nil
}

func fmtSprintf(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs == 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("fmt.sprintf", "at least 1", numArgs)
	}

	format, ok := args[0].AsString()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("fmt.sprintf", "format", "string", args[0].TypeName())
	}
	if numArgs == 1 {
		return vm.Allocator().NewStringValue(format), nil
	}
	s, err := formatter.Format(format, args[1:]...)
	if err != nil {
		return core.UndefinedValue(), err
	}
	return vm.Allocator().NewStringValue(s), nil
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
