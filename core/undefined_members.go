package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// undefined's members: one function per entry of TypeUndefined.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func undefinedConvert(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	// the maybe-missing rescue: the conversion members exist on undefined with a MANDATORY default,
	// so a propagated chain can materialize with a typed fallback — d["missing"].int(0) → 0.
	// Absence converts to nothing, so with no default the conversion raises like every T(undefined).
	switch len(args) {
	case 1:
		return args[0], nil
	case 0:
		return Undefined, errs.NewConversionError(undefinedTypeName, name, "value is missing")
	default:
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
}
