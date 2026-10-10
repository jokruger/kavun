package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/jokruger/kavun/core/member/members"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/jokruger/fin128/civil"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// date is a civil calendar day, 0001-01-01…9999-12-31, with no clock and no zone. It is a primitive: Data holds the
// int32 count of days since 1970-01-01 (civil.Date.Days), so a date never allocates and a d"…" literal lives in the
// static pool like an int. The Go side works through fin128's civil.Date, the type the fin module is shaped around.

const dateTypeName = "date"

// ZeroDate is date's zero value, `date()`: 1970-01-01, the day whose int() is 0 and whose midnight is time().
var ZeroDate, _ = civil.FromDays(0)

// DateValue wraps a civil date.
func DateValue(d civil.Date) Value {
	return Value{Type: value.Date, Immutable: true, Data: uint64(uint32(d.Days()))}
}

// dateOf unwraps a date value. Data always holds a valid day: every constructor checks the range.
func dateOf(v Value) civil.Date {
	d, _ := civil.FromDays(int32(uint32(v.Data)))
	return d
}

// TypeDate is the date type descriptor.
var TypeDate = ValueTypeDescr{
	Name:         ConstHook(dateTypeName),                                                     // PURE by contract
	String:       dateTypeString,                                                              // PURE by contract
	Format:       dateTypeFormat,                                                              // PURE by contract
	Interface:    func(v Value) any { return dateOf(v) },                                      // PURE by contract
	EncodeJSON:   dateTypeEncodeJSON,                                                          // PURE by contract
	EncodeBinary: dateTypeEncodeBinary,                                                        // PURE by contract
	DecodeBinary: dateTypeDecodeBinary,                                                        // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return v.Data != 0, nil },                     // PURE by contract: falsy iff date() (1970-01-01, day 0)
	Len:          ConstHook(int64(1)),                                                         // PURE by contract
	Equal:        dateTypeEqual,                                                               // PURE by contract
	BinaryOp:     dateTypeBinaryOp,                                                            // PURE by contract
	AsString:     func(v Value) (string, bool) { return dateOf(v).String(), true },            // PURE by contract
	AsInt:        func(v Value) (int64, bool) { return int64(dateOf(v).Days()), true },        // PURE by contract
	AsTime:       func(v Value) (time.Time, bool) { return dateMidnightUTC(dateOf(v)), true }, // PURE by contract
	AsDate:       func(v Value) (civil.Date, bool) { return dateOf(v), true },                 // PURE by contract

	Methods: []MethodEntry{
		members.IsTrue:       {Fn: memberIsTrue, Pure: true},
		members.String:       {Fn: dateString, Pure: true},
		members.Format:       {Fn: memberFormat, Pure: true},
		members.Copy:         {Fn: memberSelf, Pure: true},
		members.Freeze:       {Fn: memberSelf, Pure: true},
		members.Runes:        {Fn: dateRunes, Pure: true},
		members.Int:          {Fn: dateInt, Pure: true},
		members.Time:         {Fn: dateTime, Pure: true},
		members.Date:         {Fn: dateDate, Pure: true},
		members.Components:   {Fn: dateComponents, Pure: true},
		members.Year:         {Fn: dateYear, Pure: true},
		members.Month:        {Fn: dateMonth, Pure: true},
		members.Day:          {Fn: dateDay, Pure: true},
		members.WeekDay:      {Fn: dateWeekDay, Pure: true},
		members.WeekDayName:  {Fn: dateWeekDayName, Pure: true},
		members.MonthName:    {Fn: dateMonthName, Pure: true},
		members.YearDay:      {Fn: dateYearDay, Pure: true},
		members.DaysInMonth:  {Fn: dateDaysInMonth, Pure: true},
		members.DaysInYear:   {Fn: dateDaysInYear, Pure: true},
		members.IsLeapYear:   {Fn: dateIsLeapYear, Pure: true},
		members.AddYears:     {Fn: dateAddYears, Pure: true},
		members.AddMonths:    {Fn: dateAddMonths, Pure: true},
		members.AddDays:      {Fn: dateAddDays, Pure: true},
		members.StartOfMonth: {Fn: dateStartOfMonth, Pure: true},
		members.EndOfMonth:   {Fn: dateEndOfMonth, Pure: true},
		members.IsEndOfMonth: {Fn: dateIsEndOfMonth, Pure: true},
		members.MonthsSince:  {Fn: dateMonthsSince, Pure: true},
		members.TimeIn:       {Fn: dateTimeIn, Pure: true},
	},
}

func dateMidnightUTC(d civil.Date) time.Time {
	y, m, dd := d.YMD()
	return time.Date(y, time.Month(m), dd, 0, 0, 0, 0, time.UTC)
}

// PURE by contract
func dateTypeString(v Value) string {
	return fmt.Sprintf("date(%q)", dateOf(v).String())
}

// PURE by contract
func dateTypeEncodeJSON(v Value) ([]byte, error) {
	return []byte(`"` + dateOf(v).String() + `"`), nil
}

// PURE by contract
func dateTypeEncodeBinary(v Value) ([]byte, error) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(v.Data))
	return b, nil
}

// IMPURE by contract (mutates target)
func dateTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) != 4 {
		return fmt.Errorf("date: expected 4 bytes, got %d", len(data))
	}
	d, ok := civil.FromDays(int32(binary.LittleEndian.Uint32(data)))
	if !ok {
		return fmt.Errorf("date: day count out of range")
	}
	*v = DateValue(d)
	return nil
}

// dateDirectives are the strftime directives that mean something for a day with no clock and no zone.
const dateDirectives = "YyCmdejBbAauwVGnt%"

// PURE by contract
func dateTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return dateTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(dateTypeName, sp, fspec.AlignLeft), nil
	}
	if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.HasPrec || sp.ZeroPad || sp.CoerceZero || sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	d := dateOf(v)
	var body string
	switch sp.Verb {
	case 0:
		body = d.String()
	case '#':
		switch sp.Tail {
		case "", "date", "iso":
			body = d.String()
		default:
			if !strings.ContainsRune(sp.Tail, '%') {
				return "", errs.NewFormattingError(fmt.Sprintf("date: layout %q is not available on a date (it has no clock or zone)", sp.Tail))
			}
			for i := 0; i < len(sp.Tail); i++ {
				if sp.Tail[i] == '%' && i+1 < len(sp.Tail) {
					i++
					if !strings.ContainsRune(dateDirectives, rune(sp.Tail[i])) {
						return "", errs.NewFormattingError(fmt.Sprintf("date: directive %%%c is not available on a date (it has no clock or zone)", sp.Tail[i]))
					}
				}
			}
			out, err := strftime(dateMidnightUTC(d), sp.Tail)
			if err != nil {
				return "", err
			}
			body = out
		}
	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	return fspec.ApplyGenerics(body, sp, fspec.AlignLeft), nil
}

// PURE by contract.
func dateTypeEqual(v Value, other Value, final bool) bool {
	if other.Type == value.Date {
		return v.Data == other.Data
	}
	if final {
		return false
	}
	return ValueTypes[other.Type].Equal(other, v, true)
}

// dateShiftDays is d moved by n days, raising when the result leaves 0001-01-01…9999-12-31.
func dateShiftDays(d civil.Date, n int64) (Value, error) {
	if n >= -1<<31 && n <= 1<<31-1 {
		if r, ok := d.AddDays(int32(n)); ok {
			return DateValue(r), nil
		}
	}
	return Undefined, errs.NewInvalidValueError(fmt.Sprintf("date out of range: %s %+d days", d, n))
}

// PURE by contract
func dateTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	d := dateOf(v)
	if reflected {
		// n + d: days, as d + n
		if other.Type == value.Int && op == tokens.Add {
			return dateShiftDays(d, int64(other.Data))
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}
	switch other.Type {
	case value.Int:
		switch op {
		case tokens.Add:
			return dateShiftDays(d, int64(other.Data))
		case tokens.Sub:
			n := int64(other.Data)
			if n == -1<<63 {
				return Undefined, errs.NewInvalidValueError("date out of range")
			}
			return dateShiftDays(d, -n)
		}
	case value.Date:
		o := dateOf(other)
		switch op {
		case tokens.Sub:
			return IntValue(int64(d.Sub(o))), nil
		case tokens.Less:
			return BoolValue(d.Before(o)), nil
		case tokens.Greater:
			return BoolValue(d.After(o)), nil
		case tokens.LessEq:
			return BoolValue(!d.After(o)), nil
		case tokens.GreaterEq:
			return BoolValue(!d.Before(o)), nil
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), other.TypeName())
	}
	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// dateStartIn is the first instant whose wall-clock date in loc is d: midnight when midnight exists (the earlier
// one if a fall-back repeats it), otherwise — a DST gap swallowing midnight — the instant the gap ends. So
// d.time_in(z).date_in(z) == d always holds.
func dateStartIn(d civil.Date, loc *time.Location) time.Time {
	y, m, dd := d.YMD()
	if c := wallClockInstants(y, int(m), dd, 0, 0, 0, 0, loc); len(c) > 0 {
		return c[0]
	}
	// in a gap: Go normalizes the missing midnight to SOME instant beside the gap — the gap ends where the zone
	// period around that instant does (or begins, if Go moved forward), so the first instant of d is whichever of
	// them reads d
	u := time.Date(y, time.Month(m), dd, 0, 0, 0, 0, loc)
	reads := func(c time.Time) bool {
		cy, cm, cd := c.In(loc).Date()
		return cy == y && int(cm) == int(m) && cd == dd
	}
	start, end := u.ZoneBounds()
	switch {
	case !end.IsZero() && reads(end) && !reads(u):
		return end.In(loc)
	case !start.IsZero() && reads(start):
		return start.In(loc)
	}
	return u
}

// ParseDateText reads the canonical date text: exactly YYYY-MM-DD, every field fixed width, the day must exist.
func ParseDateText(s string) (civil.Date, error) {
	p := timeScanner{s: s}
	y, ok := p.digits(4)
	if !ok {
		return civil.Date{}, p.fail("expected a 4-digit year")
	}
	if !p.lit('-') {
		return civil.Date{}, p.fail("expected '-' after the year")
	}
	m, ok := p.digits(2)
	if !ok {
		return civil.Date{}, p.fail("expected a 2-digit month")
	}
	if !p.lit('-') {
		return civil.Date{}, p.fail("expected '-' after the month")
	}
	d, ok := p.digits(2)
	if !ok {
		return civil.Date{}, p.fail("expected a 2-digit day")
	}
	if p.i != len(p.s) {
		return civil.Date{}, p.fail("unexpected text (a date has no clock or zone)")
	}
	return newDate(y, m, d)
}

// newDate is the checked (y, m, d) constructor: every part in range, nothing normalizes.
func newDate(y, m, d int) (civil.Date, error) {
	if y < civil.MinYear || y > civil.MaxYear {
		return civil.Date{}, fmt.Errorf("year %d out of range %d..%d", y, civil.MinYear, civil.MaxYear)
	}
	if m < 1 || m > 12 {
		return civil.Date{}, fmt.Errorf("month %d out of range 1..12", m)
	}
	if dim := civil.DaysInMonth(y, civil.Month(m)); d < 1 || d > dim {
		return civil.Date{}, fmt.Errorf("day %d out of range 1..%d", d, dim)
	}
	r, _ := civil.New(y, civil.Month(m), d)
	return r, nil
}

// CheckDateLayout validates a date parse layout: the `date` alias, or a %-template of date directives only
// (no clock, no zone). A wrong layout is a *LayoutError.
func CheckDateLayout(layout string) error {
	if !strings.ContainsRune(layout, '%') {
		if layout == "date" {
			return nil
		}
		return layoutErr("unknown date layout %q: expected \"date\" or a %%-template", layout)
	}
	for k := 0; k < len(layout); k++ {
		if layout[k] == '%' && k+1 < len(layout) {
			k++
			switch layout[k] {
			case 'Y', 'm', 'd', 'e', 'j', 'B', 'b', 'A', 'a', 'n', 't', '%':
			case 'y', 'G', 'V', 'u', 'w', 'C':
				// the time-layout check names why these are refused
			default:
				if strings.ContainsRune("HIMSfpPzsZ", rune(layout[k])) {
					return layoutErr("directive %%%c is not available on a date (it has no clock or zone)", layout[k])
				}
			}
		}
	}
	return CheckTimeLayout(layout)
}

// ParseDateLayout reads s as a date under a layout.
func ParseDateLayout(s, layout string) (civil.Date, error) {
	if err := CheckDateLayout(layout); err != nil {
		return civil.Date{}, err
	}
	if layout == "date" {
		return ParseDateText(s)
	}
	t, err := strptime(s, layout)
	if err != nil {
		return civil.Date{}, err
	}
	y, m, d := t.Date()
	return newDate(y, int(m), d)
}

// textDateMember is the text receivers' `date` member: `s.date(layout[, default])`, the layout required, as for time.
func textDateMember(name, from, text string, args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1 or 2", len(args))
	}
	if args[0].Type != value.String {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first (layout)", "string", args[0].TypeName())
	}
	layout, _ := args[0].AsString()
	d, err := ParseDateLayout(text, layout)
	if err == nil {
		return DateValue(d), nil
	}
	var le *LayoutError
	if errors.As(err, &le) {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, err))
	}
	if len(args) == 2 {
		return args[1], nil
	}
	return Undefined, errs.NewConversionError(from, name, err.Error())
}

// DateFromComponents rebuilds a date from {year, month, day}: every key optional (defaulting to date()'s part),
// an unknown key raises, nothing normalizes. The way back from d.components().
func DateFromComponents(m map[string]Value) (civil.Date, error) {
	for _, k := range slices.Sorted(maps.Keys(m)) {
		switch k {
		case "year", "month", "day":
		default:
			return civil.Date{}, errs.NewInvalidValueError(fmt.Sprintf("(date) unknown component %q", k))
		}
	}
	part := func(key string, dflt int64) (int, error) {
		v, ok := m[key]
		if !ok {
			return int(dflt), nil
		}
		if v.Type != value.Int {
			return 0, errs.NewInvalidArgumentTypeError("date", key, "int", v.TypeName())
		}
		n := int64(v.Data)
		if n < -1<<31 || n > 1<<31-1 {
			return 0, errs.NewInvalidValueError(fmt.Sprintf("(date) %s %d out of range", key, n))
		}
		return int(n), nil
	}
	y, err := part("year", 1970)
	if err != nil {
		return civil.Date{}, err
	}
	mo, err := part("month", 1)
	if err != nil {
		return civil.Date{}, err
	}
	d, err := part("day", 1)
	if err != nil {
		return civil.Date{}, err
	}
	r, err := newDate(y, mo, d)
	if err != nil {
		return civil.Date{}, errs.NewInvalidValueError("(date) " + err.Error())
	}
	return r, nil
}
