package error

import "fmt"

type DecodeLengthError struct {
	Type     string
	Expected int
	Found    int
	Context  string
}

func (e *DecodeLengthError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("decode %s from binary (%s): invalid length: expected %d bytes, found %d", e.Type, e.Context, e.Expected, e.Found)
	}
	return fmt.Sprintf("decode %s from binary: invalid length: expected %d bytes, found %d", e.Type, e.Expected, e.Found)
}
