package format

import (
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// ValidateContainerSpec rejects fields that have no meaning for container types. Containers accept only the empty verb
// plus width / fill / align. The 'v' verb is dispatched separately by each container's Format method.
func ValidateContainerSpec(typeName string, sp fspec.FormatSpec) error {
	if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.HasPrec || sp.ZeroPad || sp.CoerceZero {
		return errs.NewUnsupportedFormatSpec(typeName, sp)
	}
	if sp.Verb != 0 {
		return errs.NewUnsupportedFormatSpec(typeName, sp)
	}
	return nil
}
