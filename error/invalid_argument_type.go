package error

import "fmt"

type InvalidArgumentTypeError struct {
	Name     string
	Expected string
	Found    string
}

func (e *InvalidArgumentTypeError) Error() string {
	return fmt.Sprintf("invalid type for argument '%s': expected %s, found %s", e.Name, e.Expected, e.Found)
}
