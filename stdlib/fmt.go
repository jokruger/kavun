package stdlib

import (
	"fmt"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/module"
	"github.com/jokruger/kavun/core/value"
)

func init() {
	// 2..127 reserved
	InitModule("fmt", module.Fmt, nil, map[uint64]*core.BuiltinFunction{
		0: core.NewBuiltinFunction("print", fmtPrint, 0, true),
		1: core.NewBuiltinFunction("println", fmtPrintln, 0, true),
	})
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
		case value.Undefined, value.Bytes, value.Array, value.Record, value.Dict, value.IntRange:
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
