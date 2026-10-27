package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/fin128/civil"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const timeTypeName = "time"

func NewStaticTimeValue(t *time.Time) Value {
	return Value{Type: value.Time, Immutable: true, Ptr: unsafe.Pointer(t)}
}

func NewTimeValue(t time.Time) Value {
	return Value{Type: value.Time, Immutable: true, Ptr: unsafe.Pointer(&t)}
}

// An instant in time, with nanosecond precision and a time zone.
//
// ## Overview
//
// `time` is a domain scalar: an absolute instant plus the zone it is *viewed* in. The zone affects the
// calendar accessors (`hour()`, `day()`, …), calendar arithmetic and the rendered text; it never affects
// identity — two values of the same instant in different zones are equal, and ordering and duration
// arithmetic work on the instant.
//
// Key rules, each expanded below:
//
// - Next to a `time`, a bare `int` **operand** is a duration in **nanoseconds**; a bare `int` in a
// **conversion** is a unix timestamp. Operator = duration, conversion = timestamp — and an int is a duration
// wherever its member says so (`30.seconds()`, `d.in_hours()`).
// - `time` and `int` deliberately do **not** compare: an instant and a bare number have no common order.
// - **One strict text grammar** (below) for `t"…"`, `time(s)` and the canonical layouts — nothing is guessed.
// Any other text shape names its **layout**: `time(s, layout)`, `s.parse_time(layout[, default])`.
// - **A wall clock that does not name exactly one instant raises** — a time a daylight-saving gap skips, or one
// an overlap repeats — whether it is constructed or computed. Nothing normalizes.
// - A zone is an IANA name (`"Europe/Kyiv"`) or `"UTC"`. The host's own zone is never visible: `"Local"` and
// `""` are not zone names, and `clock.now()` answers in UTC.
// - One render surface: `format(spec)` with the format mini-language. One canonical text form: RFC3339 with
// the fraction the instant carries (RFC3339Nano), used by `.string()`, f-strings, and the default
// `format()`.
//
// `time` values are immutable. The range is years 1…9999.
//
// The calendar accessors are all zero-argument, answered in the value's own zone.
//
// ## Construction
//
// ### Text: the canonical grammar
//
// `t"…"` literals, `time(s)` and the layouts `iso`/`isonano`/`date`/`datetime` read exactly this grammar:
//
// | form | reads as |
// | --- | --- |
// | `YYYY-MM-DD` | midnight UTC |
// | `YYYY-MM-DD` + `T` **or a space** + `HH:MM[:SS[.f]]` (1–9 fraction digits) | that wall clock, UTC |
// | any of the above + `Z` / `±HH:MM` / `±HHMM` | that wall clock at the stated offset (`Z` and `+00:00` are UTC) |
//
// ```go
// t"2026-08-29T15:04:05Z"             // a literal: a compile-time constant
// time("2026-08-29")                  // => time("2026-08-29T00:00:00Z")
// time("2026-08-29 15:04:05")         // => time("2026-08-29T15:04:05Z") — the form #datetime writes
// time("2026-08-29T15:04+03:00")      // => time("2026-08-29T15:04:00+03:00")
// time("9999-12-31T23:59:59.9999999") // the far sentinel many backends use reads fine
// ```
//
// Every field is fixed width and every value must exist. These raise `conversion` (a `t"…"` literal that does not
// match is a compile error): unpadded fields (`"2026-8-9"`), slashes (`"12/01/2026"`), month names and free text,
// bare digits (`"20260829"`, `"1700000000"` — name the unit with a layout), lowercase `t`/`z`, and impossible
// values (`"2026-02-30"`, `"2026-08-29T24:00"`).
//
// ```go
// time("12/01/2026")                  // Error: conversion: cannot convert string to time: expected a 4-digit year at offset 0
// time("2026-02-30")                  // Error: conversion: ...: day 30 out of range 1..28
// ```
//
// ### Text: layouts
//
// Any other shape names its layout. On a text receiver the conversion member `s.time([default])` reads the
// canonical grammar, like `time(s)`; reading under a layout is its own member, `s.parse_time(layout[, default])`,
// whose default comes after the layout — so a layout can never be mistaken for a default:
//
// ```go
// time("29/08/2026", "%d/%m/%Y")              // free form: the second argument is the layout
// "29/08/2026".parse_time("%d/%m/%Y")         // member form: the layout is required
// "bad".parse_time("%d/%m/%Y", undefined)     // => undefined — the default, after the layout
// "2026-08-29".parse_time("iso")              // the canonical grammar, spelled explicitly
// "2026-08-29".time()                         // the canonical grammar: the conversion member
// time("29/08/2026 03:04 PM", "%d/%m/%Y %I:%M %p")   // => time("2026-08-29T15:04:00Z")
// time("1700000000", "unix")                  // => time("2023-11-14T22:13:20Z") — the layout names the unit
// ```
//
// | layout | reads |
// | --- | --- |
// | `iso`, `isonano`, `date`, `datetime` | the canonical grammar above |
// | `unix`, `unixms`, `unixmicro`, `unixnano` | an integer, in exactly that unit |
// | `time`, `rfc822` | refused — no date / a 2-digit year and a zone abbreviation |
// | a `%`-template (anything containing `%`) | the directives below; every other byte matches literally |
//
// Readable directives — the inverse of the render directives, over the ones that read back unambiguously:
// `%Y %m %d %e %j %H %I %M %S %f %p %P %z %s` (fixed width, exactly as rendered), `%B %b %A %a` (English
// names, exact case; a weekday that contradicts the date raises), `%n %t %%`. Refused on input: `%y` (a century
// pivot is a guess), `%Z` (an abbreviation names several zones), and the week-based `%G %V %u %w %C`. Unset fields
// default to the zero value's (`1970-01-01T00:00:00Z`). `%I` needs `%p`; `%s` stands alone.
//
// A wrong **layout** is a mistake in the script, not a miss in the data, so it raises `invalid_value` even when a
// default is given:
//
// ```go
// "x".parse_time("%y", undefined)           // Error: invalid_value — parse_time) directive %y cannot be parsed (ambiguous on input
// "x".parse_time("%d/%m/%Y", undefined)     // => undefined — the data missed, the layout is fine
// ```
//
// ### Components
//
// A `dict` or `record` converts through the **components** reading — the same shape `components()` answers.
// Every key is optional and defaults to the zero value's part; an unknown key raises; nothing normalizes:
//
// ```go
// time({year: 2026, month: 8, day: 29})                       // => time("2026-08-29T00:00:00Z")
// time({year: 2026, month: 3, day: 29, hour: 15, zone: "Europe/Kyiv"})   // => time("2026-03-29T15:00:00+03:00")
// time({year: 2026, hour: 1, zone_offset: 3600})              // => time("2026-01-01T01:00:00+01:00") — a fixed offset
// time({})                                                    // => time("1970-01-01T00:00:00Z") — the zero time
// time({yr: 2026})                                            // Error: invalid_value: (time) unknown component "yr"
// time({year: 2026, month: 1, day: 32})                       // Error: invalid_value: (time) day 32 out of range 1..31
// ```
//
// `zone` makes the parts a wall clock in that zone, and the **one-instant rule** applies:
//
// ```go
// time({year: 2026, month: 3, day: 29, hour: 3, minute: 30, zone: "Europe/Kyiv"})
// // Error: invalid_value: (time) 2026-03-29 03:30:00 does not exist in zone Europe/Kyiv (a daylight-saving gap)
// time({year: 2026, month: 10, day: 25, hour: 3, minute: 30, zone: "Europe/Kyiv"})
// // Error: invalid_value: (time) ... occurs twice in zone Europe/Kyiv (a daylight-saving overlap): give zone_offset to pick one
// time({year: 2026, month: 10, day: 25, hour: 3, minute: 30, zone: "Europe/Kyiv", zone_offset: 7200})
// // => time("2026-10-25T03:30:00+02:00") — the second 03:30
// ```
//
// `zone` and `zone_offset` together must agree; `"Local"`, `""` and unknown names raise.
//
// ### Numbers
//
// An `int` in conversion position is a **unix timestamp**; the member name picks the encoding:
//
// ```go
// (0).time()                          // => time("1970-01-01T00:00:00Z") — seconds
// (1700000000).time()                 // => time("2023-11-14T22:13:20Z")
// (1700000000123).time_ms()           // => time("2023-11-14T22:13:20.123Z") — milliseconds
// (1700000000123456).time_micro()     // microseconds
// (1700000000123456789).time_nano()   // nanoseconds
// ```
//
// `float` and `decimal` read as fractional unix seconds:
//
// ```go
// (1700000000.5).time()               // => time("2023-11-14T22:13:20.5Z")
// decimal("1700000000.25").time()     // => time("2023-11-14T22:13:20.25Z")
// ```
//
// Every numeric conversion answers in **UTC** — an integer timestamp is host-independent, so its wall-clock
// reading must be too.
//
// ### The free constructor and the clock
//
// ```go
// time()                     // => time("1970-01-01T00:00:00Z") — the zero time, unix 0
// time(1700000000)           // ≡ (1700000000).time()
// time("2026-08-29")         // the canonical grammar
// time("29/08/2026", "%d/%m/%Y") // a layout (the free form has no default slot; the member does)
// undefined.time(time())     // => time("1970-01-01T00:00:00Z") — the maybe-missing form
//
// clock := import("clock")
// clock.now()                                     // the current instant, in UTC (the clock module needs a grant)
// 1700000000.time() + 500                         // => time("2023-11-14T22:13:20.0000005Z") — seconds, then ns
// ```
//
// On bad input every constructor **raises** a catchable error — none answers an `error` value. The current instant
// is the only world read: it comes from the [`clock`](../stdlib/clock.md) module (`now`, `since`, `until`,
// `sleep`). Calendar facts are members (`t.is_leap_year()`, `t.days_in_month()`).
//
// ## Operators
//
// | expression | result | meaning |
// | --- | --- | --- |
// | `t + n` / `n + t` | `time` | `n` is a duration in **nanoseconds** |
// | `t - n` | `time` | back by `n` nanoseconds |
// | `t2 - t1` | `int` | the elapsed nanoseconds |
// | `t1 < t2` (`<=` `>` `>=`) | `bool` | ordering of instants |
// | `t == x` / `t != x` | `bool` | instant equality; `false`/`true` against any non-time |
// | `n - t` | raises | an int minus an instant has no meaning |
// | `t < n` / `n < t` | raises | deliberate — see below |
// | `t + t2`, `t * n`, … | raises | no instant+instant, no scaling |
//
// ```go
// t := t"2026-08-29T15:04:05.123456789Z"
// t2 := t"2026-08-29T16:04:05.123456789Z"
//
// t + 1000000000            // => time("2026-08-29T15:04:06.123456789Z") — one second later
// t - 1000000000            // => time("2026-08-29T15:04:04.123456789Z")
// t + 30.seconds()          // readable duration spelling — the int's duration member
// t2 - t                    // => 3600000000000 — one hour, in nanoseconds
// t < t2                    // => true
// t == t                    // => true
// ```
//
// A duration is exact elapsed time; to move by **calendar** days or months, use the arithmetic members below.
//
// ### Reading a duration
//
// A duration is nothing special — a plain `int` of nanoseconds. Its members read it in a unit, render it, and
// build one (see [int § Durations](int.md#durations)):
//
// ```go
// t := t"2026-08-29T15:04:05.123456789Z"
// t2 := t"2026-08-29T16:04:05.123456789Z"
// elapsed := t2 - t                  // => 3600000000000
// elapsed / 1.minutes()              // => 60 — int division
// elapsed.in_seconds()               // => 3600.0 — seconds, as a float
// elapsed.in_minutes()               // => 60.0
// elapsed.in_hours()                 // => 1.0
// elapsed.duration_string()          // => "1h0m0s"
// "1h".nanoseconds() == elapsed      // => true — the text form of the same reading
// ```
//
// ### The missing comparisons
//
// **`time` vs `int` comparison is deliberately absent** — an instant and a bare number have no common order
// (nanoseconds since when? seconds or millis?), so both directions raise instead of guessing:
//
// ```go
// t := t"2026-08-29T15:04:05.123456789Z"
// t < 1700000000            // Error: invalid_binary_operator: time < int
// 1700000000 < t            // Error: invalid_binary_operator: int < time
// t == 1788015845           // => false — equality just answers "not the same value"
// ```
//
// Equality and ordering are zone-insensitive — they compare instants:
//
// ```go
// t := t"2026-08-29T15:04:05.123456789Z"
// kyiv := t.in_zone("Europe/Kyiv")
// kyiv == t                 // => true — same instant, different view
// ```
//
// ## Conversions out
//
// Every conversion is `x.T([default])`. An instant **as a number is its unix timestamp** — `int()` is a
// deliberate synonym of `unix()` (the type name answers "as a number", the unit name "in which unit"). Each
// conversion's own entry shows it.
//
// There is deliberately no `bool()` (truthiness is `!!t` / `t.is_true()`), no `byte()`/`rune()`/`bytes()`
// (no octet meaning), and no container targets or `len()` — an instant has no elements.
//
// ## Zone data
//
// Zone names resolve through the host's IANA tzdata (Go's lookup order: `ZONEINFO`, the system copy,
// `$GOROOT/lib/time/zoneinfo.zip`, and the embedded copy the `kavun` CLI links in). Zone rules change, so a
// named-zone result can differ between hosts with different tzdata versions; UTC and fixed offsets never do.
// See [embedding](../../embedding.md#time-zone-data) for pinning tzdata and replaying past results.
//
//kavun:type time

// TypeTime is a time type descriptor.
var TypeTime = ValueTypeDescr{
	Name:         ConstHook(timeTypeName), // PURE by contract
	String:       timeTypeString,          // PURE by contract
	Format:       timeTypeFormat,          // PURE by contract
	Interface:    timeTypeInterface,       // PURE by contract
	EncodeJSON:   timeTypeEncodeJSON,      // PURE by contract
	EncodeBinary: timeTypeEncodeBinary,    // PURE by contract
	DecodeBinary: timeTypeDecodeBinary,    // IMPURE by contract (mutates target)
	IsTrue:       timeTypeIsTrue,          // PURE by contract
	Len:          ConstHook(int64(1)),     // PURE by contract
	Equal:        timeTypeEqual,           // PURE by contract
	BinaryOp:     timeTypeBinaryOp,        // PURE by contract
	AsString:     timeTypeAsString,        // PURE by contract
	AsInt:        timeTypeAsInt,           // PURE by contract
	AsFloat:      timeTypeAsFloat,         // PURE by contract
	AsDecimal:    timeTypeAsDecimal,       // PURE by contract
	AsTime:       timeTypeAsTime,          // PURE by contract
	AsDate:       timeTypeAsDate,          // PURE by contract

	// Every member is pure: zone data is read by name and is the host's tzdata, which the purity contract treats as
	// fixed for a process (docs/purity.md); no member reads the host's own zone.
	Methods: timeMethods,
}

// TimeFromComponents rebuilds an instant from its constitutive parts. Every key is optional and defaults to the
// zero time's part (1970-01-01T00:00:00 UTC), so an empty map is the zero time; an UNKNOWN key raises, so a typo
// is an error rather than a silent default. The way back from t.components().
//
// `zone` names an IANA zone and makes the parts a wall clock in it; `zone_offset` alone is a fixed offset; with
// both, the offset picks between the two instants of a daylight-saving overlap and must match the zone. The
// wall clock goes through the checked builder: a part out of range, a clock a gap skips, or an overlap clock
// with no offset raises — nothing normalizes.
func TimeFromComponents(m map[string]Value) (time.Time, error) {
	get := func(key string, dflt int64) (int64, error) {
		v, ok := m[key]
		if !ok {
			return dflt, nil
		}
		i, ok := v.AsInt()
		if !ok || v.Type == value.Float || v.Type == value.Decimal {
			return 0, errs.NewInvalidArgumentTypeError("time", key, "int", v.TypeName())
		}
		return i, nil
	}
	// sorted so that a components map with SEVERAL unknown keys always names the same one: the error text
	// is part of the observable behaviour, and map order must not leak into it
	for _, k := range slices.Sorted(maps.Keys(m)) {
		switch k {
		case "year", "month", "day", "hour", "minute", "second", "nanosecond", "zone", "zone_offset":
		default:
			return time.Time{}, errs.NewInvalidValueError(fmt.Sprintf("(time) unknown component %q", k))
		}
	}
	var parts [7]int
	for i, kd := range []struct {
		key  string
		dflt int64
	}{{"year", 1970}, {"month", 1}, {"day", 1}, {"hour", 0}, {"minute", 0}, {"second", 0}, {"nanosecond", 0}} {
		n, err := get(kd.key, kd.dflt)
		if err != nil {
			return time.Time{}, err
		}
		if n < -1<<31 || n > 1<<31-1 {
			return time.Time{}, errs.NewInvalidValueError(fmt.Sprintf("(time) %s %d out of range", kd.key, n))
		}
		parts[i] = int(n)
	}
	var offset *int
	if _, ok := m["zone_offset"]; ok {
		n, err := get("zone_offset", 0)
		if err != nil {
			return time.Time{}, err
		}
		if n <= -24*3600 || n >= 24*3600 {
			return time.Time{}, errs.NewInvalidValueError(fmt.Sprintf("(time) zone_offset %d out of range", n))
		}
		o := int(n)
		offset = &o
	}
	loc := time.UTC
	if zv, ok := m["zone"]; ok {
		if zv.Type != value.String {
			return time.Time{}, errs.NewInvalidArgumentTypeError("time", "zone", "string", zv.TypeName())
		}
		z, _ := zv.AsString()
		l, err := LoadZone(z)
		if err != nil {
			return time.Time{}, errs.NewInvalidValueError("(time) " + err.Error())
		}
		loc = l
	} else if offset != nil {
		loc = fixedZone(*offset)
	}
	t, err := buildTime(parts[0], parts[1], parts[2], parts[3], parts[4], parts[5], parts[6], loc, offset)
	if err != nil {
		return time.Time{}, errs.NewInvalidValueError("(time) " + err.Error())
	}
	return t, nil
}

// PURE by contract
func timeTypeInterface(v Value) any {
	return *(*time.Time)(v.Ptr)
}

// PURE by contract
func timeTypeIsTrue(v Value) (bool, error) {
	return !isZeroTime(*(*time.Time)(v.Ptr)), nil
}

// PURE by contract
func timeTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*time.Time)(v.Ptr)
	y, err := o.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return y, nil
}

// PURE by contract
func timeTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*time.Time)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(*o); err != nil {
		return nil, fmt.Errorf("time: %w", err)
	}
	return buf.Bytes(), nil
}

// IMPURE by contract (mutates target)
func timeTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var t time.Time
	if err := dec.Decode(&t); err != nil {
		return fmt.Errorf("time: %w", err)
	}
	*v = NewTimeValue(t)
	return nil
}

// PURE by contract
func timeTypeString(v Value) string {
	o := (*time.Time)(v.Ptr)
	return fmt.Sprintf("time(%q)", o.Format(time.RFC3339Nano))
}

// PURE by contract
func timeTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return timeTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(timeTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.HasPrec || sp.ZeroPad || sp.CoerceZero || sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	t := (*time.Time)(v.Ptr)

	var body string
	switch sp.Verb {
	case 0:
		// the default render is precision-preserving: the fraction appears iff
		// the instant carries one; #iso is the explicitly seconds-truncating spec
		body = t.Format(time.RFC3339Nano)

	case '#':
		switch sp.Tail {
		case "":
			body = t.Format(time.RFC3339Nano)
		case "iso":
			body = t.Format(time.RFC3339)
		case "isonano":
			body = t.Format(time.RFC3339Nano)
		case "date":
			body = t.Format("2006-01-02")
		case "time":
			body = t.Format("15:04:05")
		case "datetime":
			body = t.Format(time.DateTime)
		case "unix":
			body = strconv.FormatInt(t.Unix(), 10)
		case "unixms":
			body = strconv.FormatInt(t.UnixMilli(), 10)
		case "unixmicro":
			body = strconv.FormatInt(t.UnixMicro(), 10)
		case "unixnano":
			body = strconv.FormatInt(t.UnixNano(), 10)
		case "rfc822":
			body = t.Format(time.RFC822)
		default:
			out, err := strftime(*t, sp.Tail)
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
func timeTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Time:
		o := (*time.Time)(v.Ptr)
		r := (*time.Time)(other.Ptr)
		return o.Equal(*r)
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

func timeTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
		case value.Int:
			switch op {
			case tokens.Add:
				l := int64(other.Data)
				r := (*time.Time)(v.Ptr)
				return NewTimeValue(r.Add(time.Duration(l))), nil
			}
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Time:
		l := *(*time.Time)(v.Ptr)
		r := *(*time.Time)(other.Ptr)
		switch op {
		case tokens.Sub:
			return IntValue(int64(l.Sub(r))), nil
		case tokens.Less:
			return BoolValue(l.Before(r)), nil
		case tokens.Greater:
			return BoolValue(l.After(r)), nil
		case tokens.LessEq:
			return BoolValue(l.Equal(r) || l.Before(r)), nil
		case tokens.GreaterEq:
			return BoolValue(l.Equal(r) || l.After(r)), nil
		}

	case value.Int:
		switch op {
		case tokens.Add:
			l := (*time.Time)(v.Ptr)
			r := int64(other.Data)
			return NewTimeValue(l.Add(time.Duration(r))), nil
		case tokens.Sub:
			l := (*time.Time)(v.Ptr)
			r := int64(other.Data)
			return NewTimeValue(l.Add(time.Duration(-r))), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract
func timeTypeAsString(v Value) (string, bool) {
	// ONE text form: RFC3339 with the fraction the instant carries — the same
	// text format() and f-strings produce, and it round-trips through the parse
	return (*time.Time)(v.Ptr).Format(time.RFC3339Nano), true
}

// PURE by contract
func timeTypeAsInt(v Value) (int64, bool) {
	return (*time.Time)(v.Ptr).Unix(), true
}

// an instant as a number is unix sec.frac; the float form is approximate
// (~100ns at present-day magnitudes), the decimal form exact to the nanosecond
func timeTypeAsFloat(v Value) (float64, bool) {
	o := (*time.Time)(v.Ptr)
	return float64(o.UnixNano()) / 1e9, true
}

func timeTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	o := (*time.Time)(v.Ptr)
	d := dec128.FromInt64(o.UnixNano()).Div(dec128.FromInt64(1e9))
	return d, !d.IsNaN()
}

// parseTimeText is the shared text -> time conversion behind string's, runes' and bytes' AsTime hooks: the
// canonical grammar only (ParseTimeText). Any other layout is named explicitly at the call site.
func parseTimeText(s string) (time.Time, bool) {
	t, err := ParseTimeText(s)
	return t, err == nil
}

// zoneArg reads a zone-name argument through LoadZone: a string naming an IANA zone or "UTC".
func zoneArg(name, pos string, a Value) (*time.Location, error) {
	if a.Type != value.String {
		return nil, errs.NewInvalidArgumentTypeError(name, pos, "string", a.TypeName())
	}
	z, _ := a.AsString()
	loc, err := LoadZone(z)
	if err != nil {
		return nil, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, err))
	}
	return loc, nil
}

// timeTypeAsDate is the civil day t reads in its own zone.
//
// PURE by contract
func timeTypeAsDate(v Value) (civil.Date, bool) {
	y, m, d := (*time.Time)(v.Ptr).Date()
	return civil.New(y, civil.Month(m), d)
}

// PURE by contract
func timeTypeAsTime(v Value) (time.Time, bool) {
	return *(*time.Time)(v.Ptr), true
}
