package core

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jokruger/fin128/civil"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// This file is the time type's text and calendar codec: the canonical grammar (ParseTimeText), the layout
// language in both directions (strftime renders, strptime reads — the two directive tables live side by side
// here so they cannot drift apart), the zone loader, and the checked wall-clock builder every construction
// and calendar-arithmetic path goes through. Nothing here normalizes: a part outside its range, a wall clock
// a zone skips, or one it repeats is an error, never a silently different instant.

// ZeroTime is the time type's zero value, `time()`: the unix epoch in UTC. It is the instant whose `int()`
// is 0, and the midnight of `date()`'s zero day, so the two zero values agree. Truthiness compares against
// it (the derived rule: `x.is_true()` iff `x != T()`).
var ZeroTime = time.Unix(0, 0).UTC()

// isZeroTime reports whether t is the zero value — the same INSTANT as ZeroTime, whatever zone views it.
func isZeroTime(t time.Time) bool {
	return t.Equal(ZeroTime)
}

const (
	minTimeYear = 1
	maxTimeYear = 9999
)

// ParseTimeText reads the canonical time text:
//
//	YYYY-MM-DD [ ('T' | ' ') HH:MM [ ':' SS [ '.' f{1,9} ] ] ] [ 'Z' | ±HH:MM | ±HHMM ]
//
// Every field is fixed width and every value must exist (no normalization). A text without a zone suffix is
// UTC; 'Z' and a zero offset are UTC; any other offset is a fixed zone. Anything else is an error that names
// the byte offset where reading stopped.
func ParseTimeText(s string) (time.Time, error) {
	p := timeScanner{s: s}
	y, ok := p.digits(4)
	if !ok {
		return time.Time{}, p.fail("expected a 4-digit year")
	}
	if !p.lit('-') {
		return time.Time{}, p.fail("expected '-' after the year")
	}
	mo, ok := p.digits(2)
	if !ok {
		return time.Time{}, p.fail("expected a 2-digit month")
	}
	if !p.lit('-') {
		return time.Time{}, p.fail("expected '-' after the month")
	}
	d, ok := p.digits(2)
	if !ok {
		return time.Time{}, p.fail("expected a 2-digit day")
	}
	var h, mi, sec, ns int
	if p.lit('T') || p.lit(' ') {
		if h, ok = p.digits(2); !ok {
			return time.Time{}, p.fail("expected a 2-digit hour")
		}
		if !p.lit(':') {
			return time.Time{}, p.fail("expected ':' after the hour")
		}
		if mi, ok = p.digits(2); !ok {
			return time.Time{}, p.fail("expected 2-digit minutes")
		}
		if p.lit(':') {
			if sec, ok = p.digits(2); !ok {
				return time.Time{}, p.fail("expected 2-digit seconds")
			}
			if p.lit('.') {
				start := p.i
				for p.i < len(p.s) && p.i-start < 9 && isASCIIDigit(p.s[p.i]) {
					p.i++
				}
				n := p.i - start
				if n == 0 {
					return time.Time{}, p.fail("expected 1 to 9 fraction digits")
				}
				if p.i < len(p.s) && isASCIIDigit(p.s[p.i]) {
					return time.Time{}, p.fail("more than 9 fraction digits")
				}
				f, _ := strconv.Atoi(p.s[start:p.i])
				for ; n < 9; n++ {
					f *= 10
				}
				ns = f
			}
		}
	}
	loc := time.UTC
	if p.lit('Z') {
		// UTC
	} else if p.i < len(p.s) && (p.s[p.i] == '+' || p.s[p.i] == '-') {
		off, err := p.offset(true)
		if err != nil {
			return time.Time{}, err
		}
		loc = fixedZone(off)
	}
	if p.i != len(p.s) {
		return time.Time{}, p.fail("unexpected text")
	}
	return buildTime(y, mo, d, h, mi, sec, ns, loc, nil)
}

// timeScanner is the cursor both parsers share. Errors carry the byte offset where reading stopped.
type timeScanner struct {
	s string
	i int
}

func (p *timeScanner) fail(what string) error {
	return fmt.Errorf("%s at offset %d", what, p.i)
}

func (p *timeScanner) lit(c byte) bool {
	if p.i < len(p.s) && p.s[p.i] == c {
		p.i++
		return true
	}
	return false
}

// digits reads exactly n ASCII digits.
func (p *timeScanner) digits(n int) (int, bool) {
	if p.i+n > len(p.s) {
		return 0, false
	}
	v := 0
	for k := 0; k < n; k++ {
		c := p.s[p.i+k]
		if !isASCIIDigit(c) {
			return 0, false
		}
		v = v*10 + int(c-'0')
	}
	p.i += n
	return v, true
}

// offset reads ±HHMM, or also ±HH:MM when colon is allowed, and answers it in seconds east of UTC.
func (p *timeScanner) offset(colon bool) (int, error) {
	sign := 1
	if p.s[p.i] == '-' {
		sign = -1
	}
	p.i++
	h, ok := p.digits(2)
	if !ok {
		return 0, p.fail("expected a 2-digit offset hour")
	}
	if colon {
		p.lit(':')
	}
	m, ok := p.digits(2)
	if !ok {
		return 0, p.fail("expected 2-digit offset minutes")
	}
	if h > 23 || m > 59 {
		return 0, p.fail("offset out of range")
	}
	return sign * (h*3600 + m*60), nil
}

func isASCIIDigit(c byte) bool { return c >= '0' && c <= '9' }

// fixedZone answers the location for a fixed offset: UTC itself for zero, so `Z` and `+00:00` read alike.
func fixedZone(off int) *time.Location {
	if off == 0 {
		return time.UTC
	}
	return time.FixedZone("", off)
}

var zoneCache sync.Map // name -> *time.Location

// LoadZone resolves a zone name: an IANA name or "UTC". "" and "Local" are refused — a script names the zone
// it means, and the host's own zone is host state a script must not read by name. Successful lookups are
// cached; the data comes from the host's tzdata (or the embedded copy the CLI links in).
func LoadZone(name string) (*time.Location, error) {
	switch name {
	case "":
		return nil, fmt.Errorf("a zone name is required")
	case "Local":
		return nil, fmt.Errorf(`"Local" is not a zone name: name the zone explicitly`)
	case "UTC":
		return time.UTC, nil
	}
	if l, ok := zoneCache.Load(name); ok {
		return l.(*time.Location), nil
	}
	l, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("unknown time zone %s", name)
	}
	zoneCache.Store(name, l)
	return l, nil
}

// zoneNameOf answers the IANA name of t's location, or "" for UTC and fixed offsets — the zones that are
// fully described by zone_offset alone.
func zoneNameOf(t time.Time) string {
	l := t.Location()
	if l == time.UTC || l.String() == "" || l.String() == "UTC" {
		return ""
	}
	return l.String()
}

// TimeHasNamedZone reports whether t is viewed in a named (IANA) zone, as opposed to UTC or a fixed offset —
// the zones its text form fully describes.
func TimeHasNamedZone(t time.Time) bool {
	return zoneNameOf(t) != ""
}

// buildTime is the checked wall-clock builder: every part must be in range (no normalization), and the wall
// clock must name exactly one instant in loc. A clock a DST gap skips raises; a clock a DST overlap repeats
// raises unless offset (seconds east of UTC) picks one of the two. A non-nil offset that matches neither — or,
// for a zone without the ambiguity, the one instant — raises too.
func buildTime(y, mo, d, h, mi, s, ns int, loc *time.Location, offset *int) (time.Time, error) {
	if y < minTimeYear || y > maxTimeYear {
		return time.Time{}, fmt.Errorf("year %d out of range %d..%d", y, minTimeYear, maxTimeYear)
	}
	if mo < 1 || mo > 12 {
		return time.Time{}, fmt.Errorf("month %d out of range 1..12", mo)
	}
	if dim := civil.DaysInMonth(y, civil.Month(mo)); d < 1 || d > dim {
		return time.Time{}, fmt.Errorf("day %d out of range 1..%d", d, dim)
	}
	if h < 0 || h > 23 {
		return time.Time{}, fmt.Errorf("hour %d out of range 0..23", h)
	}
	if mi < 0 || mi > 59 {
		return time.Time{}, fmt.Errorf("minute %d out of range 0..59", mi)
	}
	if s < 0 || s > 59 {
		return time.Time{}, fmt.Errorf("second %d out of range 0..59", s)
	}
	if ns < 0 || ns > 999_999_999 {
		return time.Time{}, fmt.Errorf("nanosecond %d out of range 0..999999999", ns)
	}

	wall := time.Date(y, time.Month(mo), d, h, mi, s, ns, time.UTC)
	matches := func(u time.Time) bool {
		uy, um, ud := u.Date()
		uh, umi, us := u.Clock()
		return uy == y && int(um) == mo && ud == d && uh == h && umi == mi && us == s
	}

	// The instants whose wall clock in loc reads (y..ns): wall shifted back by each offset loc uses around it.
	// A zone changes offset at most once in any short span, so the offsets in force a day either side are
	// every candidate there is.
	var found []time.Time
	seen := map[int]bool{}
	for _, probe := range []time.Time{wall.Add(-36 * time.Hour), wall, wall.Add(36 * time.Hour)} {
		_, off := probe.In(loc).Zone()
		if seen[off] {
			continue
		}
		seen[off] = true
		u := wall.Add(-time.Duration(off) * time.Second).In(loc)
		if matches(u) {
			found = append(found, u)
		}
	}

	switch {
	case len(found) == 0:
		return time.Time{}, fmt.Errorf("%s does not exist in zone %s (a daylight-saving gap)", wall.Format("2006-01-02 15:04:05"), loc)
	case offset != nil:
		for _, u := range found {
			if _, off := u.Zone(); off == *offset {
				return u, nil
			}
		}
		return time.Time{}, fmt.Errorf("zone_offset %d does not match zone %s at %s", *offset, loc, wall.Format("2006-01-02 15:04:05"))
	case len(found) > 1:
		return time.Time{}, fmt.Errorf("%s occurs twice in zone %s (a daylight-saving overlap): give zone_offset to pick one", wall.Format("2006-01-02 15:04:05"), loc)
	}
	return found[0], nil
}

// rebuildWallClock rebuilds t's wall clock on a new calendar day (y, mo, d) in t's own zone, under the
// wall-clock rule. It is how calendar arithmetic keeps the clock.
func rebuildWallClock(t time.Time, y, mo, d int) (time.Time, error) {
	h, mi, s := t.Clock()
	return buildTime(y, mo, d, h, mi, s, t.Nanosecond(), t.Location(), nil)
}

// timeAddDays is t moved by n calendar days in its own zone, the wall clock kept.
func timeAddDays(t time.Time, n int64) (time.Time, error) {
	y, mo, d := t.Date()
	day, ok := civil.New(y, civil.Month(mo), d)
	if !ok || n < -1<<31 || n > 1<<31-1 {
		return time.Time{}, fmt.Errorf("result out of range")
	}
	r, ok := day.AddDays(int32(n))
	if !ok {
		return time.Time{}, fmt.Errorf("result out of range")
	}
	ry, rm, rd := r.YMD()
	return rebuildWallClock(t, ry, int(rm), rd)
}

// timeAddMonths is t moved by n months in its own zone under the end-of-month rule, the wall clock kept.
func timeAddMonths(t time.Time, n int64, rule civil.EOMRule) (time.Time, error) {
	y, mo, d := t.Date()
	day, ok := civil.New(y, civil.Month(mo), d)
	if !ok || n < -1<<31 || n > 1<<31-1 {
		return time.Time{}, fmt.Errorf("result out of range")
	}
	r, ok := day.AddMonths(int32(n), rule)
	if !ok {
		return time.Time{}, fmt.Errorf("result out of range")
	}
	ry, rm, rd := r.YMD()
	return rebuildWallClock(t, ry, int(rm), rd)
}

// EOMRuleArg reads an end-of-month rule argument: exactly "clamp" or "last_day". There is no default.
func EOMRuleArg(name, pos string, a Value) (civil.EOMRule, error) {
	if a.Type != value.String {
		return 0, errs.NewInvalidArgumentTypeError(name, pos, "string", a.TypeName())
	}
	s, _ := a.AsString()
	switch s {
	case "clamp":
		return civil.EOMClamp, nil
	case "last_day":
		return civil.EOMLastDay, nil
	}
	return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) unknown end-of-month rule %q, expected one of: clamp, last_day", name, s))
}

// ParseTimeLayout reads s under a layout: a format alias, or a %-template (anything containing '%').
//
//	iso, isonano, date, datetime        the canonical grammar (ParseTimeText)
//	unix, unixms, unixmicro, unixnano   an integer, in exactly that unit
//	time, rfc822                        refused: no date / a 2-digit year and a zone abbreviation
//
// A layout that is itself wrong (unknown alias or directive, a refused directive, %I without %p, ...) is a
// *LayoutError, reported before any text is read: it is a mistake in the script, never a miss in the data,
// so a caller's default must not swallow it.
func ParseTimeLayout(s, layout string) (time.Time, error) {
	if err := CheckTimeLayout(layout); err != nil {
		return time.Time{}, err
	}
	if strings.ContainsRune(layout, '%') {
		return strptime(s, layout)
	}
	switch layout {
	case "iso", "isonano", "date", "datetime":
		return ParseTimeText(s)
	case "unix", "unixms", "unixmicro", "unixnano":
		n, err := parseStrictInt(s)
		if err != nil {
			return time.Time{}, err
		}
		var t time.Time
		switch layout {
		case "unix":
			t = time.Unix(n, 0)
		case "unixms":
			t = time.UnixMilli(n)
		case "unixmicro":
			t = time.UnixMicro(n)
		default:
			t = time.Unix(0, n)
		}
		t = t.UTC()
		if t.Year() < minTimeYear || t.Year() > maxTimeYear {
			return time.Time{}, fmt.Errorf("result out of range")
		}
		return t, nil
	}
	return time.Time{}, layoutErr("unknown layout %q", layout) // unreachable: CheckTimeLayout admitted it
}

// LayoutError is a layout that cannot be used for reading, whatever the text.
type LayoutError struct{ msg string }

func (e *LayoutError) Error() string { return e.msg }

func layoutErr(format string, a ...any) error { return &LayoutError{msg: fmt.Sprintf(format, a...)} }

// CheckTimeLayout validates a parse layout statically: the alias exists and reads back, or every directive of
// the template is one strptime reads and the combination is coherent.
func CheckTimeLayout(layout string) error {
	if !strings.ContainsRune(layout, '%') {
		switch layout {
		case "iso", "isonano", "date", "datetime", "unix", "unixms", "unixmicro", "unixnano":
			return nil
		case "time", "rfc822":
			return layoutErr("layout %q cannot be parsed: it has no date / a 2-digit year and a zone abbreviation", layout)
		}
		return layoutErr("unknown layout %q: expected an alias (iso, isonano, date, datetime, unix, unixms, unixmicro, unixnano) or a %%-template", layout)
	}
	seen := map[byte]bool{}
	for k := 0; k < len(layout); k++ {
		if layout[k] != '%' {
			continue
		}
		if k+1 >= len(layout) {
			return layoutErr("trailing '%%' in layout %q", layout)
		}
		k++
		dir := layout[k]
		switch dir {
		case 'Y', 'm', 'd', 'e', 'j', 'H', 'I', 'M', 'S', 'f', 'p', 'P', 'B', 'b', 'A', 'a', 'z', 's':
			if seen[dir] {
				return layoutErr("directive %%%c appears twice in layout %q", dir, layout)
			}
			seen[dir] = true
		case 'n', 't', '%':
		case 'y', 'Z', 'G', 'V', 'u', 'w', 'C':
			return layoutErr("directive %%%c cannot be parsed (ambiguous on input)", dir)
		default:
			return layoutErr("unknown directive %%%c in layout %q", dir, layout)
		}
	}
	ampm := seen['p'] || seen['P']
	switch {
	case seen['s'] && len(seen) > 1:
		return layoutErr("%%s cannot be combined with other date or time directives")
	case seen['I'] && seen['H']:
		return layoutErr("%%I and %%H cannot be combined")
	case seen['I'] && !ampm:
		return layoutErr("%%I needs %%p")
	case ampm && !seen['I']:
		return layoutErr("%%p needs %%I")
	case seen['p'] && seen['P']:
		return layoutErr("%%p and %%P cannot be combined")
	case seen['B'] && seen['b'], seen['A'] && seen['a'], seen['d'] && seen['e']:
		return layoutErr("a field appears twice in layout %q", layout)
	case (seen['B'] || seen['b']) && seen['m']:
		return layoutErr("a field appears twice in layout %q", layout)
	}
	return nil
}

// textTimeMember is the text receivers' `time` member: `s.time(layout[, default])`. Text has no single way to
// become a time, so the layout is REQUIRED — the conversion's trailing default comes after it, where a layout
// can never be mistaken for one. A miss answers the default as-is, or raises naming why.
func textTimeMember(name, from, text string, args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1 or 2", len(args))
	}
	if args[0].Type != value.String {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first (layout)", "string", args[0].TypeName())
	}
	layout, _ := args[0].AsString()
	t, err := ParseTimeLayout(text, layout)
	if err == nil {
		return NewTimeValue(t), nil
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

// parseStrictInt reads an optional '-' and one or more ASCII digits, nothing else.
func parseStrictInt(s string) (int64, error) {
	body := strings.TrimPrefix(s, "-")
	if body == "" {
		return 0, fmt.Errorf("expected an integer")
	}
	for i := 0; i < len(body); i++ {
		if !isASCIIDigit(body[i]) {
			return 0, fmt.Errorf("expected an integer at offset %d", i+len(s)-len(body))
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("integer out of range")
	}
	return n, nil
}

var (
	monthNames   = [...]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	weekdayNames = [...]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
)

// strptime reads s under a %-template — the inverse of strftime, over the directives that read back
// unambiguously:
//
//	%Y %m %d %H %M %S %f %j %e %I %p %P %z %s   fixed width, exactly as strftime writes them
//	%B %b %A %a                                 English names, exact case; a weekday that contradicts the
//	                                            date raises
//	%n %t %%                                    literal newline, tab, '%'
//
// Refused on input: %y (a century pivot is a guess), %Z (an abbreviation names several zones), and the
// week-based %G %V %u %w %C — CheckTimeLayout rejects those (and incoherent combinations) before strptime
// runs. Every other byte must match literally. Unset fields default to the zero value's
// (1970-01-01T00:00:00Z); the result goes through the same checked builder as every construction.
func strptime(s, layout string) (time.Time, error) {
	p := timeScanner{s: s}
	y, mo, d, h, mi, sec, ns := 1970, 1, 1, 0, 0, 0, 0
	yday, h12, weekday := -1, -1, -1
	pm := -1 // -1 unset, 0 AM, 1 PM
	hasMD := false
	var offset *int
	unix := int64(0)
	hasUnix := false

	for k := 0; k < len(layout); k++ {
		c := layout[k]
		if c != '%' {
			if !p.lit(c) {
				return time.Time{}, p.fail(fmt.Sprintf("expected %q", string(c)))
			}
			continue
		}
		k++ // CheckTimeLayout guarantees a directive follows
		var ok bool
		switch dir := layout[k]; dir {
		case 'Y':
			y, ok = p.digits(4)
		case 'm':
			mo, ok = p.digits(2)
			hasMD = true
		case 'd':
			d, ok = p.digits(2)
			hasMD = true
		case 'e':
			if p.lit(' ') {
				d, ok = p.digits(1)
			} else {
				d, ok = p.digits(2)
			}
			hasMD = true
		case 'j':
			yday, ok = p.digits(3)
		case 'H':
			h, ok = p.digits(2)
		case 'I':
			h12, ok = p.digits(2)
		case 'M':
			mi, ok = p.digits(2)
		case 'S':
			sec, ok = p.digits(2)
		case 'f':
			var us int
			us, ok = p.digits(6)
			ns = us * 1000
		case 'p', 'P':
			am, pmText := "AM", "PM"
			if dir == 'P' {
				am, pmText = "am", "pm"
			}
			switch {
			case strings.HasPrefix(p.s[p.i:], am):
				pm, ok = 0, true
				p.i += 2
			case strings.HasPrefix(p.s[p.i:], pmText):
				pm, ok = 1, true
				p.i += 2
			}
		case 'B', 'b':
			for i, n := range monthNames {
				if dir == 'b' {
					n = n[:3]
				}
				if strings.HasPrefix(p.s[p.i:], n) {
					mo, ok = i+1, true
					p.i += len(n)
					break
				}
			}
			hasMD = true
		case 'A', 'a':
			for i, n := range weekdayNames {
				if dir == 'a' {
					n = n[:3]
				}
				if strings.HasPrefix(p.s[p.i:], n) {
					weekday, ok = i, true
					p.i += len(n)
					break
				}
			}
		case 'z':
			if p.i < len(p.s) && (p.s[p.i] == '+' || p.s[p.i] == '-') {
				off, err := p.offset(false)
				if err != nil {
					return time.Time{}, err
				}
				offset, ok = &off, true
			}
		case 's':
			start := p.i
			if p.lit('-') {
				// sign
			}
			for p.i < len(p.s) && isASCIIDigit(p.s[p.i]) {
				p.i++
			}
			n, err := parseStrictInt(p.s[start:p.i])
			if err != nil {
				return time.Time{}, err
			}
			unix, ok, hasUnix = n, true, true
		case 'n':
			ok = p.lit('\n')
		case 't':
			ok = p.lit('\t')
		case '%':
			ok = p.lit('%')
		default:
			return time.Time{}, layoutErr("unknown directive %%%c", dir) // unreachable: CheckTimeLayout admitted it
		}
		if !ok {
			return time.Time{}, p.fail(fmt.Sprintf("text does not match %%%c", layout[k]))
		}
	}
	if p.i != len(p.s) {
		return time.Time{}, p.fail("unexpected text")
	}

	if hasUnix {
		return ParseTimeLayout(strconv.FormatInt(unix, 10), "unix")
	}
	if h12 >= 0 {
		if h12 < 1 || h12 > 12 {
			return time.Time{}, fmt.Errorf("hour %d out of range 1..12", h12)
		}
		h = h12 % 12
		if pm == 1 {
			h += 12
		}
	}
	if yday >= 0 {
		days := civil.DaysInYear(y)
		if yday < 1 || yday > days {
			return time.Time{}, fmt.Errorf("day of year %d out of range 1..%d", yday, days)
		}
		jd := time.Date(y, 1, yday, 0, 0, 0, 0, time.UTC)
		if hasMD && (int(jd.Month()) != mo || jd.Day() != d) {
			return time.Time{}, fmt.Errorf("day of year %d contradicts the month and day", yday)
		}
		mo, d = int(jd.Month()), jd.Day()
	}
	loc := time.UTC
	if offset != nil {
		loc = fixedZone(*offset)
	}
	t, err := buildTime(y, mo, d, h, mi, sec, ns, loc, nil)
	if err != nil {
		return time.Time{}, err
	}
	if weekday >= 0 && int(t.Weekday()) != weekday {
		return time.Time{}, fmt.Errorf("weekday %s contradicts %s", weekdayNames[weekday], t.Format("2006-01-02"))
	}
	return t, nil
}

// strftime renders t using a Python-style layout containing %-directives. Supported codes:
//
//	%Y  4-digit year                    %B  full month name        %p  AM / PM
//	%y  2-digit year                    %b  abbreviated month name %P  am / pm
//	%C  century   (00-99)               %A  full weekday name      %j  day of year (001-366)
//	%m  month     (01-12)               %a  abbreviated weekday    %s  unix seconds
//	%d  day       (01-31)               %u  ISO weekday   (1-7)    %f  microseconds (000000-999999)
//	%e  day, space-padded ( 1-31)       %w  weekday       (0-6)    %Z  timezone abbreviation
//	%H  hour 24h  (00-23)               %V  ISO week      (01-53)  %z  timezone offset (-0700)
//	%I  hour 12h  (01-12)               %G  ISO week-numbering year
//	%M  minute    (00-59)               %n  literal newline
//	%S  second    (00-59)               %t  literal tab
//	%%  literal '%'
//
// An unknown directive returns an error. strptime above is its inverse over the unambiguous subset.
func strftime(t time.Time, layout string) (string, error) {
	var b strings.Builder
	b.Grow(len(layout) + 8)
	for i := 0; i < len(layout); i++ {
		c := layout[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		if i+1 >= len(layout) {
			return "", errs.NewFormattingError(fmt.Sprintf("time: trailing '%%' in format %q", layout))
		}
		i++
		switch layout[i] {
		case 'Y':
			fmt.Fprintf(&b, "%04d", t.Year())
		case 'y':
			y := t.Year() % 100
			if y < 0 {
				y = -y
			}
			fmt.Fprintf(&b, "%02d", y)
		case 'C':
			c := t.Year() / 100
			if c < 0 {
				c = -c
			}
			fmt.Fprintf(&b, "%02d", c)
		case 'm':
			fmt.Fprintf(&b, "%02d", int(t.Month()))
		case 'd':
			fmt.Fprintf(&b, "%02d", t.Day())
		case 'e':
			fmt.Fprintf(&b, "%2d", t.Day())
		case 'H':
			fmt.Fprintf(&b, "%02d", t.Hour())
		case 'I':
			h := t.Hour() % 12
			if h == 0 {
				h = 12
			}
			fmt.Fprintf(&b, "%02d", h)
		case 'M':
			fmt.Fprintf(&b, "%02d", t.Minute())
		case 'S':
			fmt.Fprintf(&b, "%02d", t.Second())
		case 'p':
			if t.Hour() < 12 {
				b.WriteString("AM")
			} else {
				b.WriteString("PM")
			}
		case 'P':
			if t.Hour() < 12 {
				b.WriteString("am")
			} else {
				b.WriteString("pm")
			}
		case 'B':
			b.WriteString(t.Month().String())
		case 'b':
			b.WriteString(t.Month().String()[:3])
		case 'A':
			b.WriteString(t.Weekday().String())
		case 'a':
			b.WriteString(t.Weekday().String()[:3])
		case 'u':
			// ISO 8601 weekday: 1=Mon … 7=Sun.
			wd := int(t.Weekday())
			if wd == 0 {
				wd = 7
			}
			fmt.Fprintf(&b, "%d", wd)
		case 'w':
			// POSIX weekday: 0=Sun … 6=Sat.
			fmt.Fprintf(&b, "%d", int(t.Weekday()))
		case 'V':
			// ISO 8601 week of year (01-53).
			_, week := t.ISOWeek()
			fmt.Fprintf(&b, "%02d", week)
		case 'G':
			// ISO 8601 week-numbering year.
			year, _ := t.ISOWeek()
			fmt.Fprintf(&b, "%04d", year)
		case 'j':
			fmt.Fprintf(&b, "%03d", t.YearDay())
		case 'Z':
			b.WriteString(t.Format("MST"))
		case 'z':
			b.WriteString(t.Format("-0700"))
		case 'f':
			fmt.Fprintf(&b, "%06d", t.Nanosecond()/1000)
		case 's':
			fmt.Fprintf(&b, "%d", t.Unix())
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case '%':
			b.WriteByte('%')
		default:
			return "", errs.NewFormattingError(fmt.Sprintf("time: unknown strftime directive %%%c in %q", layout[i], layout))
		}
	}
	return b.String(), nil
}
