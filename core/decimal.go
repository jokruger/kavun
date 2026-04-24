package core

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/token"
)

// DecimalValue creates new boxed decimal value.
func DecimalValue(d *Decimal) Value {
	return Value{
		Type:  VT_DECIMAL,
		Const: true,
		Ptr:   unsafe.Pointer(d),
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

func decimalTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*Decimal)(v.Ptr)
	alloc := vm.Allocator()

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
		f, err := o.InexactFloat64()
		if err != nil {
			return Undefined, fmt.Errorf("failed to convert decimal to float: %w", err)
		}
		return FloatValue(f), nil

	case "to_int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, err := o.Int64()
		if err != nil {
			return Undefined, fmt.Errorf("failed to convert decimal to int: %w", err)
		}
		return IntValue(i), nil

	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewStringValue(o.String())

	case "is_zero":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(o.IsZero()), nil

	case "is_negative":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(o.IsNegative()), nil

	case "is_positive":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(o.IsPositive()), nil

	case "is_nan":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(o.IsNaN()), nil

	case "error_details":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		e, err := alloc.NewStringValue(o.ErrorDetails().Error())
		if err != nil {
			return Undefined, err
		}
		return vm.Allocator().NewErrorValue(e)

	case "sign":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(o.Sign())), nil

	case "scale":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
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
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.ToScale(uint8(scale))
		return DecimalValue(d), nil

	case "canonical":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.Canonical()
		return DecimalValue(d), nil

	case "next_up":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.NextUp()
		return DecimalValue(d), nil

	case "next_down":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.NextDown()
		return DecimalValue(d), nil

	case "abs":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.Abs()
		return DecimalValue(d), nil

	case "negate":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.Neg()
		return DecimalValue(d), nil

	case "sqrt":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.Sqrt()
		return DecimalValue(d), nil

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
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.RoundDown(uint8(scale))
		return DecimalValue(d), nil

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
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.RoundUp(uint8(scale))
		return DecimalValue(d), nil

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
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.RoundTowardZero(uint8(scale))
		return DecimalValue(d), nil

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
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.RoundAwayFromZero(uint8(scale))
		return DecimalValue(d), nil

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
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.RoundHalfTowardZero(uint8(scale))
		return DecimalValue(d), nil

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
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.RoundHalfAwayFromZero(uint8(scale))
		return DecimalValue(d), nil

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
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.RoundBank(uint8(scale))
		return DecimalValue(d), nil

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
		d, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.Trunc(uint8(scale))
		return DecimalValue(d), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, "decimal")
	}
}

func decimalTypeUnaryOp(v Value, a Allocator, op token.Token) (Value, error) {
	o := (*Decimal)(v.Ptr)

	switch op {
	case token.Sub:
		d, err := a.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = o.Neg()
		return DecimalValue(d), nil

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
		d, err := a.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = l.Add(r)
		return DecimalValue(d), nil
	case token.Sub:
		d, err := a.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = l.Sub(r)
		return DecimalValue(d), nil
	case token.Mul:
		d, err := a.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = l.Mul(r)
		return DecimalValue(d), nil
	case token.Quo:
		d, err := a.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*d = l.Div(r)
		return DecimalValue(d), nil
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
