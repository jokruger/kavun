package core

import (
	"encoding/binary"
	"errors"
	"fmt"
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

// DateOf unwraps a date value. Data always holds a valid day: every constructor checks the range.
func DateOf(v Value) civil.Date {
	d, _ := civil.FromDays(int32(uint32(v.Data)))
	return d
}

// A civil calendar day — `0001-01-01` … `9999-12-31` — with no clock and no zone.
//
// ## Overview
//
// `date` is a domain scalar for the questions that are about *days*, not instants: a due date, a value date, a
// birthday, the day a posting is booked. It has no time of day and no zone, so two dates compare and subtract
// exactly, and nothing about them depends on where the script runs. When a day has to meet the clock, the
// conversion names the zone (see [the time bridge](#the-time-bridge)).
//
// Key rules, each expanded below:
//
// - Next to a `date`, a bare `int` **operand** is a number of **days**; a bare `int` in a **conversion** is a day
// count since `1970-01-01` (epoch days). Operator = days, conversion = epoch days.
// - One strict text form, `YYYY-MM-DD`, for `d"…"`, `date(s)`, `.string()`, f-strings and JSON. Any other text
// shape names its layout: `date(s, layout)`, `s.parse_date(layout[, default])`.
// - Month and year arithmetic states its end-of-month rule — `"clamp"` or `"last_day"` — with no default.
// - `date` and `time` / `int` deliberately do **not** order against each other; `==` across them is `false`.
//
// `date` values are immutable. A date is a primitive: it never allocates, and a `d"…"` literal is a compile-time
// constant.
//
// ## Construction
//
// ```go
// d"2026-01-31"                       // literal: a compile-time constant; a bad literal is a compile error
// date("2026-01-31")                  // the canonical text
// date("31/01/2026", "%d/%m/%Y")      // a layout — the free form's second argument
// "2026-01-31".date()                 // member form on text: the canonical text, as date(s) reads it
// "31/01/2026".parse_date("%d/%m/%Y") // member form under a layout: the layout is required
// "bad".parse_date("date", undefined) // => undefined — the default comes after the layout
// date({year: 2026, month: 1, day: 31})   // components (every key optional, defaulting to date()'s part)
// (20484).date()                      // epoch days: d"2026-01-31"
// date(t"2026-03-31T23:30:00Z")       // a time's civil day, in the time's own zone
// date()                              // => date("1970-01-01") — the zero value
// undefined.date(d"2026-01-01")       // => date("2026-01-01") — the maybe-missing form
// ```
//
// The text form is exactly `YYYY-MM-DD`: fixed width, a day that exists, nothing after it. Everything else raises
// `conversion`; nothing normalizes:
//
// ```go
// date("2026-1-31")                   // Error: conversion: cannot convert string to date: expected a 2-digit month at offset 5
// date("2026-02-30")                  // Error: conversion: ...: day 30 out of range 1..28
// date("2026-01-31T00:00:00Z")        // Error: conversion: ...: unexpected text (a date has no clock or zone) at offset 10
// date({year: 2026, month: 2, day: 29})   // Error: invalid_value: (date) day 29 out of range 1..28
// date(3652059)                       // Error: conversion: cannot convert int to date — one day past 9999-12-31
// ```
//
// **Layouts** for text are the `date` alias (the canonical form) or a `%`-template of **date directives**:
// `%Y %m %d %e %j %B %b %A %a %n %t %%` (fixed width; English names, exact case; a weekday that contradicts the
// date raises). Clock and zone directives (`%H %M %S %z` …) raise on a date, and so do the ambiguous `%y %Z` and
// the week-based `%G %V %u %w %C`. A wrong layout raises `invalid_value` even when a default is given — it is a
// mistake in the script, not a miss in the data:
//
// ```go
// date("Saturday 31 January 2026", "%A %d %B %Y")   // => date("2026-01-31")
// date("2026-031", "%Y-%j")                         // => date("2026-01-31") — day of year
// "x".parse_date("%H", undefined)                   // Error: invalid_value: directive %H is not available on a date
// "x".parse_date("%d/%m/%Y", undefined)             // => undefined — the data missed, the layout is fine
// ```
//
// ## Operators
//
// | expression | result | meaning |
// | --- | --- | --- |
// | `d + n` / `n + d` | `date` | `n` days later |
// | `d - n` | `date` | `n` days earlier |
// | `d2 - d1` | `int` | the days between (negative when `d2` is earlier) |
// | `d1 < d2` (`<=` `>` `>=`) | `bool` | calendar order |
// | `d == x` / `d != x` | `bool` | the same day; `false`/`true` against any non-date |
// | a result outside `0001-01-01` … `9999-12-31` | raises | `invalid_value` |
// | `n - d`, `d + d`, `d * n`, `d / n`, `d ± float` | raises | no meaning for a day |
// | `d < n`, `d < t` (and the reverse) | raises | a day has no common order with a number or an instant |
// | `d + t` | raises | a day is not a duration |
//
// ```go
// d := d"2026-01-31"
// d + 1                       // => date("2026-02-01")
// d - 31                      // => date("2025-12-31")
// d"2026-03-01" - d"2026-02-01"   // => 28
// d < d"2026-02-01"           // => true
// d == 20484                  // => false — use d.int() == 20484
// d < 5                       // Error: invalid_binary_operator: date < int
// min(d, d"2026-02-01")       // => date("2026-01-31") — the global min/max work, the type has no min member
// ```
//
// ## The time bridge
//
// A `date` has no zone, so every move to or from a `time` names one:
//
// ```go
// t := t"2026-03-31T23:30:00Z"
// t.date()                       // => date("2026-03-31") — the civil day in the time's own zone
// t.date_in("Europe/Kyiv")       // => date("2026-04-01") — the same instant is already April 1st in Kyiv
// t.in_zone("Europe/Kyiv").date()  // => date("2026-04-01") — the same thing
//
// d := d"2026-01-31"
// d.time()                       // => time("2026-01-31T00:00:00Z") — midnight UTC; ≡ time(d)
// d.time_in("Europe/Kyiv")       // => time("2026-01-31T00:00:00+02:00") — the start of the day in Kyiv
// ```
//
// **Which day an instant falls on depends on the zone** — the first line above is a real hazard for bookings:
// near midnight, UTC and local days differ. Name the zone the business day belongs to.
//
// `time_in(zone)` answers the **first instant whose local date is the day**: midnight, or — on a day whose
// midnight a daylight-saving change skips — the instant the gap ends. So `d.time_in(z).date_in(z) == d` always
// holds:
//
// ```go
// d"2026-09-06".time_in("America/Santiago")   // => time("2026-09-06T01:00:00-03:00") — Chile skips 00:00–01:00
// ```
//
// `"Local"` and `""` are not zone names; an unknown zone raises.
//
// ## Conversions out
//
// Every conversion is `x.T([default])`; each has its own entry.
//
// `json.encode(d)` is `"2026-01-31"`; `json.decode` never produces a date (JSON has no date type), so a config
// reads it back with `date(cfg.due)`. The binary codec is 4 bytes and re-validates the range on decode.
//
// ## For hosts
//
// `kavun.ValueOf` accepts fin128's `civil.Date` directly, and a date's `Interface()` is a `civil.Date`.
//
//kavun:type date

// TypeDate is the date type descriptor.
var TypeDate = ValueTypeDescr{
	Name:         ConstHook(dateTypeName),                                                     // PURE by contract
	String:       dateTypeString,                                                              // PURE by contract
	Format:       dateTypeFormat,                                                              // PURE by contract
	Interface:    func(v Value) any { return DateOf(v) },                                      // PURE by contract
	EncodeJSON:   dateTypeEncodeJSON,                                                          // PURE by contract
	EncodeBinary: dateTypeEncodeBinary,                                                        // PURE by contract
	DecodeBinary: dateTypeDecodeBinary,                                                        // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return v.Data != 0, nil },                     // PURE by contract: falsy iff date() (1970-01-01, day 0)
	Len:          ConstHook(int64(1)),                                                         // PURE by contract
	Equal:        dateTypeEqual,                                                               // PURE by contract
	BinaryOp:     dateTypeBinaryOp,                                                            // PURE by contract
	AsString:     func(v Value) (string, bool) { return DateOf(v).String(), true },            // PURE by contract
	AsInt:        func(v Value) (int64, bool) { return int64(DateOf(v).Days()), true },        // PURE by contract
	AsTime:       func(v Value) (time.Time, bool) { return dateMidnightUTC(DateOf(v)), true }, // PURE by contract
	AsDate:       func(v Value) (civil.Date, bool) { return DateOf(v), true },                 // PURE by contract

	Methods: dateMethods,
}

func dateMidnightUTC(d civil.Date) time.Time {
	y, m, dd := d.YMD()
	return time.Date(y, time.Month(m), dd, 0, 0, 0, 0, time.UTC)
}

// PURE by contract
func dateTypeString(v Value) string {
	return fmt.Sprintf("date(%q)", DateOf(v).String())
}

// PURE by contract
func dateTypeEncodeJSON(v Value) ([]byte, error) {
	return []byte(`"` + DateOf(v).String() + `"`), nil
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
	d := DateOf(v)
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
	d := DateOf(v)
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
		o := DateOf(other)
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

// textDateMember is the text receivers' `date([default])` conversion: the strict YYYY-MM-DD form (ParseDateText),
// as the free `date(text)` reads it. A miss answers the default as-is, or raises naming why.
func textDateMember(name, from, text string, args []Value) (Value, error) {
	if len(args) > 1 {
		return Undefined, ErrArgCount
	}
	d, err := ParseDateText(text)
	if err == nil {
		return DateValue(d), nil
	}
	if len(args) == 1 {
		return args[0], nil
	}
	return Undefined, errs.NewConversionError(from, name, err.Error())
}

// textParseDateMember is the text receivers' `parse_date(layout[, default])`: the text read under a layout
// (ParseDateLayout; "date" is the strict YYYY-MM-DD form), as for parse_time.
func textParseDateMember(name, from, text string, args []Value) (Value, error) {
	if len(args) > 2 {
		return Undefined, ErrArgCount
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
