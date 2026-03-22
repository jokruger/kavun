package stdlib

import (
	"fmt"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/formatter"
	"github.com/jokruger/gs/value"
)

var fmtModule = map[string]core.Object{
	"print":   value.NewStaticBuiltinFunction("print", fmtPrint, 0, true),
	"printf":  value.NewStaticBuiltinFunction("printf", fmtPrintf, 1, true),
	"println": value.NewStaticBuiltinFunction("println", fmtPrintln, 0, true),
	"sprintf": value.NewStaticBuiltinFunction("sprintf", fmtSprintf, 1, true),
}

func fmtPrint(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return nil, err
	}
	_, _ = fmt.Print(printArgs...)
	return nil, nil
}

func fmtPrintf(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	numArgs := len(args)
	if numArgs == 0 {
		return nil, core.NewWrongNumArgumentsError("fmt.printf", "at least 1", numArgs)
	}

	format, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("fmt.printf", "format", "string", args[0])
	}
	if numArgs == 1 {
		fmt.Print(format)
		return nil, nil
	}

	s, err := formatter.Format(format, args[1:]...)
	if err != nil {
		return nil, err
	}
	fmt.Print(s)
	return nil, nil
}

func fmtPrintln(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return nil, err
	}
	printArgs = append(printArgs, "\n")
	_, _ = fmt.Print(printArgs...)
	return nil, nil
}

func fmtSprintf(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	numArgs := len(args)
	if numArgs == 0 {
		return nil, core.NewWrongNumArgumentsError("fmt.sprintf", "at least 1", numArgs)
	}

	format, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("fmt.sprintf", "format", "string", args[0])
	}
	if numArgs == 1 {
		return vm.Allocator().NewString(format), nil
	}
	s, err := formatter.Format(format, args[1:]...)
	if err != nil {
		return nil, err
	}
	return vm.Allocator().NewString(s), nil
}

func getPrintArgs(args ...core.Object) ([]any, error) {
	var printArgs []any
	l := 0
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
