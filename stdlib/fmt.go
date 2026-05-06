package stdlib

import (
	"fmt"

	"github.com/jokruger/kavun/core"
)

var fmtModule = map[string]core.Value{
	"print":   core.NewBuiltinFunctionValue("print", fmtPrint, 0, true),
	"println": core.NewBuiltinFunctionValue("println", fmtPrintln, 0, true),
}

func fmtPrint(vm core.VM, args []core.Value) (core.Value, error) {
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return core.Undefined, err
	}
	_, _ = fmt.Print(printArgs...)
	return core.Undefined, nil
}

func fmtPrintln(vm core.VM, args []core.Value) (core.Value, error) {
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return core.Undefined, err
	}
	_, _ = fmt.Println(printArgs...)
	return core.Undefined, nil
}

func getPrintArgs(args ...core.Value) ([]any, error) {
	printArgs := make([]any, 0, len(args))
	for _, arg := range args {
		switch arg.Type {
		case core.VT_UNDEFINED, core.VT_BYTES, core.VT_ARRAY, core.VT_RECORD, core.VT_DICT, core.VT_INT_RANGE:
			printArgs = append(printArgs, arg.String())

		default:
			s, ok := arg.AsString()
			if !ok {
				s = arg.String()
			}
			printArgs = append(printArgs, s)
		}
	}

	return printArgs, nil
}
