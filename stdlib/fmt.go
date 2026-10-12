package stdlib

import (
	"fmt"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
)

func init() {
	m := core.NewModule(value.ModuleIo, "fmt", false)
	// 2..127 reserved
	m.Fn(members.Print, 0, fmtPrint, 0, core.Variadic)
	m.Fn(members.Println, 1, fmtPrintln, 0, core.Variadic)
	if err := m.Register(); err != nil {
		panic(err)
	}
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
