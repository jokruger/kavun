package stdlib

import (
	"fmt"

	"github.com/jokruger/gs"
	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

var fmtModule = map[string]core.Object{
	"print":   &value.UserFunction{Name: "print", Value: fmtPrint},
	"printf":  &value.UserFunction{Name: "printf", Value: fmtPrintf},
	"println": &value.UserFunction{Name: "println", Value: fmtPrintln},
	"sprintf": &value.UserFunction{Name: "sprintf", Value: fmtSprintf},
}

func fmtPrint(args ...core.Object) (ret core.Object, err error) {
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return nil, err
	}
	_, _ = fmt.Print(printArgs...)
	return nil, nil
}

func fmtPrintf(args ...core.Object) (ret core.Object, err error) {
	numArgs := len(args)
	if numArgs == 0 {
		return nil, gse.ErrWrongNumArguments
	}

	format, ok := args[0].(*value.String)
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "format",
			Expected: "string",
			Found:    args[0].TypeName(),
		}
	}
	if numArgs == 1 {
		fmt.Print(format)
		return nil, nil
	}

	s, err := gs.Format(format.Value, args[1:]...)
	if err != nil {
		return nil, err
	}
	fmt.Print(s)
	return nil, nil
}

func fmtPrintln(args ...core.Object) (ret core.Object, err error) {
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return nil, err
	}
	printArgs = append(printArgs, "\n")
	_, _ = fmt.Print(printArgs...)
	return nil, nil
}

func fmtSprintf(args ...core.Object) (ret core.Object, err error) {
	numArgs := len(args)
	if numArgs == 0 {
		return nil, gse.ErrWrongNumArguments
	}

	format, ok := args[0].(*value.String)
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "format",
			Expected: "string",
			Found:    args[0].TypeName(),
		}
	}
	if numArgs == 1 {
		// okay to return 'format' directly as String is immutable
		return format, nil
	}
	s, err := gs.Format(format.Value, args[1:]...)
	if err != nil {
		return nil, err
	}
	return &value.String{Value: s}, nil
}

func getPrintArgs(args ...core.Object) ([]any, error) {
	var printArgs []any
	l := 0
	for _, arg := range args {
		// TODO: shell we check if arg cannot be converted to string?
		s, _ := arg.ToString()
		slen := len(s)
		// make sure length does not exceed the limit
		if l+slen > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		l += slen
		printArgs = append(printArgs, s)
	}
	return printArgs, nil
}
