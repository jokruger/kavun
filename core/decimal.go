package core

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/token"
)

// DecimalValue creates new boxed decimal value.
func DecimalValue(d *Decimal) Value {
	return Value{
		Ptr:  unsafe.Pointer(d),
		Type: VT_DECIMAL,
	}
}

// NewDecimalValue creates new (heap-allocated) boxed decimal value.
func NewDecimalValue(d Decimal) Value {
	o := &d
	return DecimalValue(o)
}

/* Decimal type methods */

func decimalTypeName(v Value) string {
	return "decimal"
}

func decimalTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Decimal)(v.Ptr)
	return o.MarshalJSON()
}

func decimalTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Decimal)(v.Ptr)
	return o.MarshalBinary()
}

func decimalTypeDecodeBinary(v *Value, data []byte) error {
	var d Decimal
	if err := d.UnmarshalBinary(data); err != nil {
		return fmt.Errorf("failed to decode decimal: %w", err)
	}
	*v = NewDecimalValue(d)
	return nil
}

func decimalTypeString(v Value) string {
	o := (*Decimal)(v.Ptr)
	return o.String()
}

func decimalTypeInterface(v Value) any {
	o := (*Decimal)(v.Ptr)
	return *o
}

func decimalTypeIsTrue(v Value) bool {
	o := (*Decimal)(v.Ptr)
	return !o.IsZero()
}

func decimalTypeAsInt(v Value) (int64, bool) {
	o := (*Decimal)(v.Ptr)
	i, err := o.Int64()
	if err != nil {
		return 0, false
	}
	return i, true
}

func decimalTypeAsString(v Value) (string, bool) {
	o := (*Decimal)(v.Ptr)
	return o.String(), true
}

func decimalTypeAsFloat(v Value) (float64, bool) {
	o := (*Decimal)(v.Ptr)
	f, err := o.InexactFloat64()
	if err != nil {
		return 0, false
	}
	return f, true
}

func decimalTypeAsDecimal(v Value) (Decimal, bool) {
	o := (*Decimal)(v.Ptr)
	return *o, true
}

func decimalTypeAsBool(v Value) (bool, bool) {
	o := (*Decimal)(v.Ptr)
	return !o.IsZero(), true
}

func decimalTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsDecimal()
	if !ok {
		return false
	}
	l := (*Decimal)(v.Ptr)
	return l.Equal(r)
}

func decimalTypeCopy(v Value, a Allocator) (Value, error) {
	o := (*Decimal)(v.Ptr)
	return a.NewDecimalValue(*o)
}

func decimalTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "to_float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		f, err := o.InexactFloat64()
		if err != nil {
			return Undefined, fmt.Errorf("failed to convert decimal to float: %w", err)
		}
		return FloatValue(f), nil

	case "to_int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		i, err := o.Int64()
		if err != nil {
			return Undefined, fmt.Errorf("failed to convert decimal to int: %w", err)
		}
		return IntValue(i), nil

	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewStringValue(o.String())

	case "is_zero":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return BoolValue(o.IsZero()), nil

	case "is_negative":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return BoolValue(o.IsNegative()), nil

	case "is_positive":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return BoolValue(o.IsPositive()), nil

	case "is_nan":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return BoolValue(o.IsNaN()), nil

	case "error_details":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		e, err := vm.Allocator().NewStringValue(o.ErrorDetails().Error())
		if err != nil {
			return Undefined, err
		}
		return vm.Allocator().NewErrorValue(e)

	case "sign":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return IntValue(int64(o.Sign())), nil

	case "scale":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return IntValue(int64(o.Scale())), nil

	case "to_scale":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.ToScale(uint8(scale)))

	case "canonical":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.Canonical())

	case "next_up":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.NextUp())

	case "next_down":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.NextDown())

	case "abs":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.Abs())

	case "negate":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.Neg())

	case "sqrt":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.Sqrt())

	case "round_down":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.RoundDown(uint8(scale)))

	case "round_up":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.RoundUp(uint8(scale)))

	case "round_toward_zero":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.RoundTowardZero(uint8(scale)))

	case "round_away_from_zero":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.RoundAwayFromZero(uint8(scale)))

	case "round_half_toward_zero":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.RoundHalfTowardZero(uint8(scale)))

	case "round_half_away_from_zero":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.RoundHalfAwayFromZero(uint8(scale)))

	case "round_bank":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.RoundBank(uint8(scale)))

	case "trunc":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		o := (*Decimal)(v.Ptr)
		return vm.Allocator().NewDecimalValue(o.Trunc(uint8(scale)))

	default:
		return Undefined, errs.NewInvalidMethodError(name, "decimal")
	}
}

func decimalTypeUnaryOp(v Value, a Allocator, op token.Token) (Value, error) {
	o := (*Decimal)(v.Ptr)
	switch op {
	case token.Sub:
		return a.NewDecimalValue(o.Neg())

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

func decimalTypeBinaryOp(v Value, a Allocator, op token.Token, rhs Value) (Value, error) {
	r, ok := rhs.AsDecimal()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	l := (*Decimal)(v.Ptr)
	switch op {
	case token.Add:
		return a.NewDecimalValue(l.Add(r))
	case token.Sub:
		return a.NewDecimalValue(l.Sub(r))
	case token.Mul:
		return a.NewDecimalValue(l.Mul(r))
	case token.Quo:
		return a.NewDecimalValue(l.Div(r))
	case token.Less:
		return BoolValue(l.LessThan(r)), nil
	case token.Greater:
		return BoolValue(l.GreaterThan(r)), nil
	case token.LessEq:
		return BoolValue(l.LessThanOrEqual(r)), nil
	case token.GreaterEq:
		return BoolValue(l.GreaterThanOrEqual(r)), nil
	default:
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}
}
