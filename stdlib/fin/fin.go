package fin

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/fin128"
	"github.com/jokruger/fin128/civil"
	"github.com/jokruger/fin128/daycount"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// The fin module binds github.com/jokruger/fin128: deterministic financial arithmetic over dec128, with every
// rounding stated. The argument rules are the module's contract, shared by every function:
//
//   - money, rates and factors are decimal or int — a float raises (determinism is the product);
//   - counts (n, period, life, m) are int;
//   - enum arguments are exact canonical names: timing "arrears" | "advance", frequency "annual" … "daily";
//   - every quantizing function ENDS with (scale, mode), read by decimal's own parser;
//   - an optional solver spec — a record/dict over {lo, hi, tolerance, max_iter}, laid over the frozen
//     fin128.DefaultSolver — sits just before (scale, mode) and is detected by argument count.
//
// Every fin128 failure becomes a Kavun error exactly once, in finRaise.

// Deterministic financial arithmetic — time value of money, cashflows, depreciation, rate conversion — backed by
// [fin128](https://github.com/jokruger/fin128) over the same dec128 library `decimal` uses.
//
// Every function either answers an exact `decimal` at the rounding you stated or raises; nothing is approximated,
// nothing defaults.
//
// ```go
// fin := import("fin")
// fin.payment(0.005d, 60, 25000, 0, "arrears", 2, "half_even")   // => -483.32d — a 60-month loan at 0.5% a month
// ```
//
// **The argument contract**, shared by every function:
//
// - Money, rates and factors are `decimal` or `int`. A `float` **raises** (`argument rate expects type decimal or
// int, got float`) — binary floating point cannot spell most decimal rates, and an approximate rate is a wrong
// posting. Counts (`n`, `period`, `life`, `m`, `num`, `den`) are `int`.
// - Every quantizing function **ends with `(scale, mode)`**, read exactly as `decimal`'s `*_round` members read
// them: scale 0…19, mode one of `ceiling`, `floor`, `down`, `up`, `half_down`, `half_up`, `half_even`. There is no
// default scale or mode.
// - Enum arguments are **exact canonical names** (case-sensitive): `timing` is `"arrears"` (payments at period end)
// or `"advance"` (at period start); `frequency` is `"annual"`, `"semiannual"`, `"quarterly"`, `"monthly"`,
// `"weekly"` or `"daily"`. An unknown name raises and lists the accepted ones.
// - Sign convention: money flowing **in** is positive, **out** negative — a positive `pv` (the amount borrowed)
// gives a negative `payment` (the amount paid back).
// - The solver-driven functions (`rate`, `irr`) take an optional **solver spec** just before `(scale, mode)`: a
// record or dict with any of `lo`, `hi` (the search bracket), `tolerance` (in ulps at the output scale) and
// `max_iter`, laid over the frozen defaults `fin.default_solver()` answers (`{lo: -0.9999d, hi: 9, tolerance: 1,
// max_iter: 100}`). An unknown key raises. The solver is bisection, so its answer depends only on the bracket and
// the tolerance.
//
// **Failures.** Data the formula cannot use raises `invalid_value` with fin128's reason —
// `(fin.payment) the number of periods must be positive`, `(fin.irr) the bracket does not contain a root`,
// `(fin.straight_line) the salvage value is above the cost`. A wrong argument type raises `invalid_argument_type`,
// a wrong count `wrong_num_arguments`, a division by zero `division_by_zero`.
//
// ## Function groups
//
// Each group's functions, linked to their entries:
//
// | group | functions |
// | --- | --- |
// | time value of money | [`payment`](#payment), [`present_value`](#present_value), [`future_value`](#future_value), [`periods`](#periods) → `[whole, remainder]`, [`rate`](#rate), [`annuity_factor_pv`](#annuity_factor_pv), [`annuity_factor_fv`](#annuity_factor_fv) |
// | schedule (period `1…n`) | [`balance`](#balance) — the balance after the period; [`charge_part`](#charge_part) / [`principal_part`](#principal_part) — the period payment's interest and principal parts |
// | cashflows | [`npv`](#npv), [`irr`](#irr), [`mirr`](#mirr) — `cf` is an array of `decimal\|int`, one per period |
// | depreciation | [`straight_line`](#straight_line), [`sum_of_digits`](#sum_of_digits), [`declining_balance`](#declining_balance), [`declining_rate_from_factor`](#declining_rate_from_factor), [`declining_rate_from_salvage`](#declining_rate_from_salvage) |
// | rates and factors | [`nominal_to_effective`](#nominal_to_effective), [`effective_to_nominal`](#effective_to_nominal) (`m` compounding periods a year), [`compound_factor`](#compound_factor), [`discount_factor`](#discount_factor), [`per_year`](#per_year) → `int` |
// | day count | [`year_fraction`](#year_fraction) (the constructor), [`year_fraction_between`](#year_fraction_between), [`year_fraction_between_final`](#year_fraction_between_final), [`days_between`](#days_between), [`days_between_final`](#days_between_final), [`days_in_year`](#days_in_year), [`conventions`](#conventions), [`is_year_fraction`](#is_year_fraction) — see [fin.year_fraction](../types/fin_year_fraction.md) |
// | interest over a fraction | [`accrue_simple`](#accrue_simple), [`accrue_compound`](#accrue_compound), [`compound_factor_for`](#compound_factor_for), [`discount_factor_for`](#discount_factor_for) |
// | discount instruments | [`discount_price`](#discount_price), [`discount_rate`](#discount_rate), [`discount_yield`](#discount_yield) |
// | dated cashflows | [`xnpv`](#xnpv), [`xirr`](#xirr) — `flows` is an array of `{date, amount}`, ascending by date |
// | rate and charge tables | [`tiered_rates`](#tiered_rates), [`tiered_charges`](#tiered_charges), [`dated_rates`](#dated_rates), [`dated_charges`](#dated_charges), and [`is_tiered_rates`](#is_tiered_rates) … — see [Rate and charge tables](#rate-and-charge-tables) |
// | exact helpers | [`apply_rate`](#apply_rate) (amount × rate, rounded once), [`exact_product`](#exact_product) (exact, or raises), [`mul_div_round`](#mul_div_round) (a × b × num / den, rounded once) |
// | units (exact) | [`to_percent`](#to_percent), [`from_percent`](#from_percent), [`to_basis_points`](#to_basis_points), [`from_basis_points`](#from_basis_points) |
// | solver | [`root`](#root) — x where `f(x) = 0`; [`brackets`](#brackets) → `bool`; [`default_solver`](#default_solver) → record |
//
// Spreadsheet equivalents, for migrations: `payment` ≈ `PMT`, `present_value` ≈ `PV`, `future_value` ≈ `FV`,
// `periods` ≈ `NPER` (split into whole periods and a remainder), `rate` ≈ `RATE`, `charge_part` ≈ `IPMT`,
// `principal_part` ≈ `PPMT`, `straight_line` ≈ `SLN`, `sum_of_digits` ≈ `SYD`. **`npv` is not the spreadsheet's
// `NPV`**: here `cf[0]` sits at t = 0 and is not discounted (the definition, as numpy-financial has it); a
// spreadsheet discounts its first value by one period. Pass the initial outlay as `cf[0]`, or shift the array.
//
// Quantize then recompute: a function's result is already rounded to the scale you asked for, and each function
// rounds its own result once. So two results rounded independently — `balance` and `principal_part` of the same
// period — can disagree by a cent with their difference. Build a schedule from **one** rounded series and derive the
// rest from it (see [the loan example](../../examples.md#a-loan-schedule-with-fin)).
//
// ## Day count
//
// **Conventions** are named by their market spelling and read through fin128's alias table, so `"ACT/365F"`,
// `"30/360 US"` or `"Actual/360"` work as products store them; `fin.conventions()` lists the canonical names
// (`30B/360`, `30E-ISDA/360`, `30E/360`, `30E3/360`, `30U/360`, `ACT/360`, `ACT/365`, `ACT/366`, `ACT/ACT`,
// `NL/365`). An unknown name raises and lists them. The `_final` variants take `end` as the contract's termination
// date — only `30E-ISDA/360` distinguishes it, when the termination falls at the end of February.
//
// `days_between` and `days_in_year` answer the **convention's** count, not the calendar's:
// `fin.days_between(d"2026-02-01", d"2026-03-01", "30E/360")` is `30` (the calendar's `d2 - d1` is `28`), and
// `fin.days_in_year(d"2028-03-01", "ACT/360")` is `360` (the calendar's `d"2028-03-01".days_in_year()` is `366`).
//
// **Dated cashflows** discount each flow by the year fraction from the **first** flow's date, so the flows must be in
// ascending date order — an unsorted array raises rather than being sorted for you:
//
// ```go
//
//	fin := import("fin")
//	flows := [{date: d"2026-01-01", amount: -10000}, {date: d"2026-06-30", amount: 2750},
//	          {date: d"2027-01-01", amount: 4250}, {date: d"2027-06-30", amount: 3250},
//	          {date: d"2028-01-01", amount: 2750}]
//	fin.xnpv(0.09d, flows, "ACT/365F", 2, "half_even")   // => 1706.87d
//	fin.xirr(flows, "ACT/365F", 10, "half_even")         // => 0.2449124168d
//
// ```
//
// Each flow is a record or dict with exactly `date` (a `date`) and `amount` (`decimal|int`).
//
// Not yet available: conventions outside the named set (a custom year length, a host-defined name table, a
// script-defined convention) — a stated fraction, `fin.year_fraction(n, d)`, covers the functions that take `f`.
//
// ## The solver: `root` and `brackets`
//
// `fin.root(f, [solver,] scale, mode)` finds the `x` where a function crosses zero, by bisection over the
// solver bracket — the same solver `rate`, `irr` and `xirr` use, so its answer depends only on the bracket and the
// tolerance. `fin.brackets(f[, solver])` answers whether `f` changes sign across the bracket at all: it tells a
// bracket that merely missed (widen `hi`) from a function with no root.
//
// ```go
//	fin := import("fin")
//	flows := [{date: d"2026-01-01", amount: -10000}, {date: d"2026-06-30", amount: 2750},
//	          {date: d"2027-01-01", amount: 4250}, {date: d"2027-06-30", amount: 3250},
//	          {date: d"2028-01-01", amount: 2750}]
//	fin.root(x => x * x - 2, 10, "half_even")                     // => 1.4142135624d
//	fin.root(r => fin.xnpv(r, flows, "ACT/365F", 12, "half_even"), 10, "half_even")   // xirr, spelled out
//	fin.brackets(x => x - 20)                                     // => false — the default bracket stops at 9
//	fin.root(x => x - 20, {hi: 50}, 10, "half_even")              // => 20.0000000000d
// ```
//
// The callback contract:
//
// - `f` is any callable that **accepts one argument** — `func(x)`, `func(x, ...rest)`, `func(...args)`, a builtin or
// a bound member that does (`decimal`, `1d.copy_sign`); one that does not raises `invalid_argument_type`, and a
// lambda adapts it (`x => g(x, 2)`). It is called with a `decimal` and returns a `decimal` or `int` — anything else
// raises `invalid_argument_type`.
// - A raise inside `f` stops the search and surfaces **as itself** — the script sees its own error, never "the bracket
// does not contain a root". After the first raise `f` is not called again. A fatal error (`stack_overflow`) stays
// fatal.
// - `f` is called at most `max_iter + 2` times (102 with the defaults).
// - `f` may itself call `fin.root`. The result is deterministic only if `f` is: a callback that reads the clock or
// other host state makes the answer depend on it. Calls that run a callback are never folded at compile time.
//
// ## Rate and charge tables
//
// Four immutable, validated table types for product parameters — a savings ladder, a fee schedule, a teaser rate
// that reverts, a standing charge that changes on a date. Each is built once, from a one-line **text form** (what a
// product configuration stores) or from **band records** (when a script computes the table), and then asked
// questions. Like `fin.year_fraction`, each type is named after its constructor.
//
// | type | text form | band record |
// | --- | --- | --- |
// | `fin.tiered_rates` | `"0:0.005, 1000:0.007, 10000:0.009"` — from 0, open at the top | `{from, rate}` |
// | `fin.tiered_charges` | `"0:0.015+2.00, 1000:0.01; min=25, max=500"` — a rate plus a fixed amount per band, the charge clamped | `{from, rate, fixed}`, with `(bands, min, max)`; `max` 0 = no cap |
// | `fin.dated_rates` | `"2026-01-01:0.04, 2026-07-01:0.02"` — each band from a date to the next | `{from: date, rate}` |
// | `fin.dated_charges` | `"2024-01-01:25.00, 2025-01-01:30.00"` | `{from: date, amount}` |
//
// A rate is written as a rate, never a percent (`0.007` is 0.7%), and the digits are kept as written (`"0:0.50"`
// stays `"0:0.50"`). Values are `decimal|int` — a `float` raises. Every ordering and anchoring rule (ascending
// bands, the first tiered band at 0, `max` not below `min`) is fin128's, and raises the same way from text or
// records. The empty text raises: a table always has a band.
//
// `rule` is `"whole"` (the band the amount lands in applies to all of it) or `"marginal"` (each slice at its own
// band's rate). A member a table does not have does not exist: `tc.rate(…)` is `invalid_method` (a fixed charge is
// money, not a rate), as is `dc.apply(…)`.
//
// **Values, not containers.** A table has no `len()`, iteration, indexing or `in`; `for b in t.bands()` reads it.
// `bands()` answers the records in constructor shape, so `fin.tiered_rates(t.bands()) == t`. Two tables are equal
// when they have the same bands, compared numerically (`"0:0.50"` equals `"0:0.5"`; for `tiered_charges` the bounds
// too) — but `"0:0, 1000:0"` is **not** `"0:0"`: under `"marginal"` an extra band changes the parts. There is no
// ordering. Each type's default (`fin.tiered_rates()` …) is the zero-rate table (`"0:0"`, or `"0001-01-01:0"` for the
// dated ones), the one falsy table.
//
// **In configuration**, store the text form: `json.encode` writes a table as its text, and `json.decode` gives the
// string back for `fin.tiered_rates(cfg.rates)`. (JSON numbers decode as `float`, which the record form refuses.)
//
//kavun:module fin pure

// moduleExtras declares what the //kavun:fn directives do not: the module's constants and owned types.
func moduleExtras(m *core.ModuleBuilder) {
	m.Type(value.FinYearFraction, TypeYearFraction)
	m.Type(value.FinTieredRates, TypeTieredRates)
	m.Type(value.FinTieredCharges, TypeTieredCharges)
	m.Type(value.FinDatedRates, TypeDatedRates)
	m.Type(value.FinDatedCharges, TypeDatedCharges)
}

// finRaise translates a fin128/dec128 error through Error, the single translation point shared with the
// fin types' members.
func finRaise(name string, err error) (core.Value, error) {
	return core.Undefined, Error("fin."+name, err)
}

// finResult answers a fin128 (value, error) pair: the error through finRaise, else the decimal.
func finResult(name string, d dec128.Dec128, err error) (core.Value, error) {
	if err != nil {
		return finRaise(name, err)
	}
	return core.DecimalResult("fin."+name, d)
}

// finArgs reads one call's arguments by position; the first failure sticks and every later read is a no-op, so a
// function body reads its whole signature and checks err once.
type finArgs struct {
	name string
	args []core.Value
	err  error
}

// newFinArgs starts reading a fin function's arguments. The dispatcher has checked the template's floor; counts,
// when given, are every count the template accepts — the body's bound or gap on its rest ("(f[, solver])": 1, 2) —
// and any other count is refused with core.ErrArgCount.
func newFinArgs(name string, args []core.Value, counts ...int) *finArgs {
	a := &finArgs{name: name, args: args}
	if len(counts) > 0 && !slices.Contains(counts, len(args)) {
		a.err = core.ErrArgCount
	}
	return a
}

func (a *finArgs) dec(i int, pos string) dec128.Dec128 {
	if a.err != nil {
		return dec128.Dec128{}
	}
	d, err := core.DecimalOperandArg("fin."+a.name, pos, a.args[i])
	a.err = err
	return d
}

func (a *finArgs) int(i int, pos string) int64 {
	if a.err != nil {
		return 0
	}
	v := a.args[i]
	if v.Type != value.Int {
		a.err = errs.NewInvalidArgumentTypeError("fin."+a.name, pos, "int", v.TypeName())
		return 0
	}
	return int64(v.Data)
}

func (a *finArgs) int32(i int, pos string) int32 {
	n := a.int(i, pos)
	if a.err == nil && (n < -1<<31 || n > 1<<31-1) {
		a.err = errs.NewInvalidValueError(fmt.Sprintf("(fin.%s) %s %d out of range", a.name, pos, n))
	}
	return int32(n)
}

// name reads an exact enum name, listing the accepted ones on a miss.
func (a *finArgs) enum(i int, pos string, names []string) int {
	if a.err != nil {
		return 0
	}
	v := a.args[i]
	if v.Type != value.String {
		a.err = errs.NewInvalidArgumentTypeError("fin."+a.name, pos, "string", v.TypeName())
		return 0
	}
	s, _ := v.AsString()
	for k, n := range names {
		if s == n {
			return k
		}
	}
	a.err = errs.NewInvalidValueError(fmt.Sprintf("(fin.%s) unknown %s %q, expected one of: %s", a.name, pos, s, strings.Join(names, ", ")))
	return 0
}

func (a *finArgs) timing(i int) fin128.Timing {
	return []fin128.Timing{fin128.Arrears, fin128.Advance}[a.enum(i, "timing", []string{"arrears", "advance"})]
}

var finFrequencies = []civil.Frequency{civil.Annual, civil.Semiannual, civil.Quarterly, civil.Monthly, civil.Weekly, civil.Daily}

func (a *finArgs) frequency(i int) civil.Frequency {
	names := make([]string, len(finFrequencies))
	for k, f := range finFrequencies {
		names[k] = f.String()
	}
	return finFrequencies[a.enum(i, "frequency", names)]
}

// fraction reads a fin.year_fraction argument.
func (a *finArgs) fraction(i int, pos string) daycount.Fraction {
	if a.err != nil {
		return daycount.Fraction{}
	}
	v := a.args[i]
	if v.Type != value.FinYearFraction {
		a.err = errs.NewInvalidArgumentTypeError("fin."+a.name, pos, "fin.year_fraction", v.TypeName())
		return daycount.Fraction{}
	}
	return YearFractionOf(v)
}

// date reads a date argument.
func (a *finArgs) date(i int, pos string) civil.Date {
	if a.err != nil {
		return civil.Date{}
	}
	v := a.args[i]
	if v.Type != value.Date {
		a.err = errs.NewInvalidArgumentTypeError("fin."+a.name, pos, "date", v.TypeName())
		return civil.Date{}
	}
	d, _ := v.AsDate()
	return d
}

// convention reads a day-count convention by name (ConventionArg: market spellings and aliases).
func (a *finArgs) convention(i int) daycount.Convention {
	if a.err != nil {
		return daycount.Convention{}
	}
	c, err := ConventionArg("fin."+a.name, "convention", a.args[i])
	a.err = err
	return c
}

// rounding reads the trailing (scale, mode) pair at i, i+1 — decimal's own parser and messages.
func (a *finArgs) rounding(i int) fin128.Rounding {
	if a.err != nil {
		return fin128.Rounding{}
	}
	scale, mode, err := core.ScaleModeArgs("fin."+a.name, a.args, i)
	if err != nil {
		a.err = err
		return fin128.Rounding{}
	}
	return fin128.NewRounding(scale, mode)
}

// flows reads an array of decimal|int cashflows.
func (a *finArgs) flows(i int, pos string) []dec128.Dec128 {
	if a.err != nil {
		return nil
	}
	v := a.args[i]
	if v.Type != value.Array {
		a.err = errs.NewInvalidArgumentTypeError("fin."+a.name, pos, "array", v.TypeName())
		return nil
	}
	elems := (*core.Array)(v.Ptr).Elements
	out := make([]dec128.Dec128, len(elems))
	for k, e := range elems {
		d, err := core.DecimalOperandArg("fin."+a.name, fmt.Sprintf("%s[%d]", pos, k), e)
		if err != nil {
			a.err = err
			return nil
		}
		out[k] = d
	}
	return out
}

// solver reads the optional solver spec at i when the call carries one (hasSpec), laid over DefaultSolver. The
// keys are lo, hi, tolerance, max_iter; an unknown key — guess included, which bisection never reads — raises,
// and an unusable combination is refused here, so fin128's ErrSolverSpec stays unreachable.
func (a *finArgs) solver(i int, hasSpec bool) fin128.SolverSpec {
	s := fin128.DefaultSolver()
	if a.err != nil || !hasSpec {
		return s
	}
	v := a.args[i]
	var m map[string]core.Value
	switch v.Type {
	case value.Record:
		m = (*core.Record)(v.Ptr).Elements
	case value.Dict:
		m = (*core.Dict)(v.Ptr).Elements
	default:
		a.err = errs.NewInvalidArgumentTypeError("fin."+a.name, "solver", "record or dict", v.TypeName())
		return s
	}
	ctx := "fin." + a.name
	for _, k := range slices.Sorted(maps.Keys(m)) {
		e := m[k]
		switch k {
		case "lo", "hi":
			d, err := core.DecimalOperandArg(ctx, "solver."+k, e)
			if err != nil {
				a.err = err
				return s
			}
			if k == "lo" {
				s.Lo = d
			} else {
				s.Hi = d
			}
		case "tolerance", "max_iter":
			if e.Type != value.Int {
				a.err = errs.NewInvalidArgumentTypeError(ctx, "solver."+k, "int", e.TypeName())
				return s
			}
			if k == "tolerance" {
				s.Tolerance = int64(e.Data)
			} else {
				s.MaxIter = int(int64(e.Data))
			}
		default:
			a.err = errs.NewInvalidValueError(fmt.Sprintf("(%s) unknown solver key %q, expected: lo, hi, tolerance, max_iter", ctx, k))
			return s
		}
	}
	switch {
	case s.MaxIter <= 0:
		a.err = errs.NewInvalidValueError(fmt.Sprintf("(%s) solver max_iter must be positive", ctx))
	case s.Tolerance < 0:
		a.err = errs.NewInvalidValueError(fmt.Sprintf("(%s) solver tolerance must not be negative", ctx))
	case !s.Hi.GreaterThan(s.Lo):
		a.err = errs.NewInvalidValueError(fmt.Sprintf("(%s) solver hi must be greater than lo", ctx))
	}
	return s
}

// ---------------------------------------------------------------------------------------------------------------
// units (exact)

func finUnit(name string, args []core.Value, f func(dec128.Dec128) dec128.Dec128) (core.Value, error) {
	a := newFinArgs(name, args)
	v := a.dec(0, "value")
	if a.err != nil {
		return core.Undefined, a.err
	}
	return core.DecimalResult("fin."+name, f(v))
}

// Answer `v` × 100, exactly.
//
//kavun:fn to_percent slot=0 pure (v) -> decimal
func finToPercent(_ core.VM, args []core.Value) (core.Value, error) {
	return finUnit("to_percent", args, fin128.ToPercent)
}

// Answer `v` ÷ 100, exactly.
//
//kavun:fn from_percent slot=1 pure (v) -> decimal
func finFromPercent(_ core.VM, args []core.Value) (core.Value, error) {
	return finUnit("from_percent", args, fin128.FromPercent)
}

// Answer `v` × 10000, exactly.
//
//kavun:fn to_basis_points slot=2 pure (v) -> decimal
func finToBasisPoints(_ core.VM, args []core.Value) (core.Value, error) {
	return finUnit("to_basis_points", args, fin128.ToBasisPoints)
}

// Answer `v` ÷ 10000, exactly.
//
//kavun:fn from_basis_points slot=3 pure (v) -> decimal
func finFromBasisPoints(_ core.VM, args []core.Value) (core.Value, error) {
	return finUnit("from_basis_points", args, fin128.FromBasisPoints)
}

// ---------------------------------------------------------------------------------------------------------------
// rounding / exact

// Answer `amount` × `rate`, rounded once.
//
//kavun:fn apply_rate slot=4 pure (amount, rate, scale, mode) -> decimal
func finApplyRate(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("apply_rate", args)
	amount, rate, out := a.dec(0, "amount"), a.dec(1, "rate"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.ApplyRate(amount, rate, out)
	return finResult(a.name, d, err)
}

// Answer `a` × `b` exactly, or raise when it cannot be held exactly.
//
//kavun:fn exact_product slot=5 pure (a, b) -> decimal
func finExactProduct(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("exact_product", args)
	x, y := a.dec(0, "a"), a.dec(1, "b")
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.ExactProduct(x, y)
	return finResult(a.name, d, err)
}

// Answer `a` × `b` × `num` ÷ `den`, rounded once.
//
//kavun:fn mul_div_round slot=6 pure (a, b, num, den, scale, mode) -> decimal
func finMulDivRound(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("mul_div_round", args)
	x, y, num, den, out := a.dec(0, "a"), a.dec(1, "b"), a.int(2, "num"), a.int(3, "den"), a.rounding(4)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.MulDivRound(x, y, num, den, out)
	return finResult(a.name, d, err)
}

// ---------------------------------------------------------------------------------------------------------------
// rate conversion

// Convert a nominal annual rate compounded `m` times a year to the effective annual rate.
//
// ```go
// fin := import("fin")
// fin.nominal_to_effective(0.12d, 12, 10, "half_even")                     // => 0.1268250301d
// ```
//
//kavun:fn nominal_to_effective slot=7 pure (nominal, m, scale, mode) -> decimal
func finNominalToEffective(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("nominal_to_effective", args)
	r, m, out := a.dec(0, "nominal"), a.int32(1, "m"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.NominalToEffective(r, m, out)
	return finResult(a.name, d, err)
}

// Convert an effective annual rate to the nominal annual rate compounded `m` times a year.
//
//kavun:fn effective_to_nominal slot=8 pure (effective, m, scale, mode) -> decimal
func finEffectiveToNominal(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("effective_to_nominal", args)
	r, m, out := a.dec(0, "effective"), a.int32(1, "m"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.EffectiveToNominal(r, m, out)
	return finResult(a.name, d, err)
}

// Answer how many periods a year a `frequency` name has.
//
//kavun:fn per_year slot=9 pure (frequency) -> int
func finPerYear(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("per_year", args)
	f := a.frequency(0)
	if a.err != nil {
		return core.Undefined, a.err
	}
	return core.IntValue(int64(f.PerYear())), nil
}

// ---------------------------------------------------------------------------------------------------------------
// factors

// Answer the compound factor (1 + `rate`) ^ `n`.
//
//kavun:fn compound_factor slot=10 pure (rate, n, scale, mode) -> decimal
func finCompoundFactor(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("compound_factor", args)
	rate, n, out := a.dec(0, "rate"), a.int(1, "n"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.CompoundFactor(rate, n, out)
	return finResult(a.name, d, err)
}

// Answer the discount factor 1 ÷ (1 + `rate`) ^ `n`.
//
//kavun:fn discount_factor slot=11 pure (rate, n, scale, mode) -> decimal
func finDiscountFactor(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("discount_factor", args)
	rate, n, out := a.dec(0, "rate"), a.int(1, "n"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.DiscountFactor(rate, n, out)
	return finResult(a.name, d, err)
}

// ---------------------------------------------------------------------------------------------------------------
// annuity / TVM

// Answer the future-value annuity factor.
//
//kavun:fn annuity_factor_fv slot=12 pure (rate, n, timing, scale, mode) -> decimal
func finAnnuityFactorFV(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("annuity_factor_fv", args)
	rate, n, t, out := a.dec(0, "rate"), a.int(1, "n"), a.timing(2), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.AnnuityFactorFV(rate, n, t, out)
	return finResult(a.name, d, err)
}

// Answer the present-value annuity factor.
//
//kavun:fn annuity_factor_pv slot=13 pure (rate, n, timing, scale, mode) -> decimal
func finAnnuityFactorPV(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("annuity_factor_pv", args)
	rate, n, t, out := a.dec(0, "rate"), a.int(1, "n"), a.timing(2), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.AnnuityFactorPV(rate, n, t, out)
	return finResult(a.name, d, err)
}

// Answer the periodic payment that takes `pv` to `fv` over `n` periods (`PMT`).
//
//kavun:fn payment slot=14 pure (rate, n, pv, fv, timing, scale, mode) -> decimal
func finPayment(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("payment", args)
	rate, n, pv, fv, t, out := a.dec(0, "rate"), a.int(1, "n"), a.dec(2, "pv"), a.dec(3, "fv"), a.timing(4), a.rounding(5)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.Payment(rate, n, pv, fv, t, out)
	return finResult(a.name, d, err)
}

// Answer the present value of a payment stream (`PV`).
//
// ```go
// fin := import("fin")
// fin.present_value(0.005d, 60, -483.32d, 0, "arrears", 2, "half_even")   // => 25000.00d
// ```
//
//kavun:fn present_value slot=15 pure (rate, n, pmt, fv, timing, scale, mode) -> decimal
func finPresentValue(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("present_value", args)
	rate, n, pmt, fv, t, out := a.dec(0, "rate"), a.int(1, "n"), a.dec(2, "pmt"), a.dec(3, "fv"), a.timing(4), a.rounding(5)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.PresentValue(rate, n, pmt, fv, t, out)
	return finResult(a.name, d, err)
}

// Answer the future value of a payment stream (`FV`).
//
//kavun:fn future_value slot=16 pure (rate, n, pmt, pv, timing, scale, mode) -> decimal
func finFutureValue(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("future_value", args)
	rate, n, pmt, pv, t, out := a.dec(0, "rate"), a.int(1, "n"), a.dec(2, "pmt"), a.dec(3, "pv"), a.timing(4), a.rounding(5)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.FutureValue(rate, n, pmt, pv, t, out)
	return finResult(a.name, d, err)
}

// Answer the number of periods (`NPER`), split into whole periods and a remainder.
//
// ```go
// fin := import("fin")
// n, rest := fin.periods(0.005d, -500, 25000, 0, "arrears", 6, "half_even")   // => [57, 0.679593d]
// ```
//
//kavun:fn periods slot=17 pure (rate, pmt, pv, fv, timing, scale, mode) -> [whole, remainder]
func finPeriods(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("periods", args)
	rate, pmt, pv, fv, t, out := a.dec(0, "rate"), a.dec(1, "pmt"), a.dec(2, "pv"), a.dec(3, "fv"), a.timing(4), a.rounding(5)
	if a.err != nil {
		return core.Undefined, a.err
	}
	whole, rest, err := fin128.Periods(rate, pmt, pv, fv, t, out)
	if err != nil {
		return finRaise(a.name, err)
	}
	r, err := core.DecimalResult("fin."+a.name, rest)
	if err != nil {
		return core.Undefined, err
	}
	return core.NewArrayValue([]core.Value{core.IntValue(whole), r}, false), nil
}

// Solve for the periodic rate (`RATE`), with an optional solver spec.
//
// ```go
// fin := import("fin")
// fin.rate(60, -483.32d, 25000, 0, "arrears", 10, "half_even")             // => 0.0049999973d
// fin.rate(60, -483.32d, 25000, 0, "arrears", {hi: 1}, 10, "half_even")    // a narrower bracket, same answer
// ```
//
//kavun:fn rate slot=18 pure (n, pmt, pv, fv, timing, scale, mode) | (n, pmt, pv, fv, timing, solver, scale, mode) -> decimal
func finRate(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("rate", args, 7, 8)
	spec := len(args) == 8
	n, pmt, pv, fv, t := a.int(0, "n"), a.dec(1, "pmt"), a.dec(2, "pv"), a.dec(3, "fv"), a.timing(4)
	s := a.solver(5, spec)
	out := a.rounding(len(args) - 2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.Rate(n, pmt, pv, fv, t, s, out)
	return finResult(a.name, d, err)
}

// ---------------------------------------------------------------------------------------------------------------
// schedule

func finSchedule(name string, args []core.Value, f func(rate dec128.Dec128, period, n int64, pv, fv dec128.Dec128, t fin128.Timing, out fin128.Rounding) (dec128.Dec128, error)) (core.Value, error) {
	a := newFinArgs(name, args)
	rate, period, n, pv, fv, t, out := a.dec(0, "rate"), a.int(1, "period"), a.int(2, "n"), a.dec(3, "pv"), a.dec(4, "fv"), a.timing(5), a.rounding(6)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := f(rate, period, n, pv, fv, t, out)
	return finResult(name, d, err)
}

// Answer the balance owed after `period` (1…`n`).
//
// ```go
// fin := import("fin")
// fin.balance(0.005d, 12, 60, 25000, 0, "arrears", 2, "half_even")         // => 20579.92d — owed after a year
// ```
//
//kavun:fn balance slot=19 pure (rate, period, n, pv, fv, timing, scale, mode) -> decimal
func finBalance(_ core.VM, args []core.Value) (core.Value, error) {
	return finSchedule("balance", args, fin128.Balance)
}

// Answer the interest part of the payment of `period` (`IPMT`); see [`balance`](#balance).
//
//kavun:fn charge_part slot=20 pure (rate, period, n, pv, fv, timing, scale, mode) -> decimal
func finChargePart(_ core.VM, args []core.Value) (core.Value, error) {
	return finSchedule("charge_part", args, fin128.ChargePart)
}

// Answer the principal part of the payment of `period` (`PPMT`); see [`balance`](#balance).
//
//kavun:fn principal_part slot=21 pure (rate, period, n, pv, fv, timing, scale, mode) -> decimal
func finPrincipalPart(_ core.VM, args []core.Value) (core.Value, error) {
	return finSchedule("principal_part", args, fin128.PrincipalPart)
}

// ---------------------------------------------------------------------------------------------------------------
// cashflows

// Answer the net present value of the cashflows `cf`, `cf[0]` at t = 0 and not discounted.
//
// `cf` is an array of `decimal|int`, one per period.
//
// ```go
// fin := import("fin")
// fin.npv(0.1d, [-1000, 300, 400, 500], 8, "half_even")                    // => -21.03681443d
// ```
//
//kavun:fn npv slot=22 pure (rate, cf, scale, mode) -> decimal
func finNPV(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("npv", args)
	rate, cf, out := a.dec(0, "rate"), a.flows(1, "cf"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.NPV(rate, cf, out)
	return finResult(a.name, d, err)
}

// Solve for the internal rate of return of the cashflows `cf`.
//
// ```go
// fin := import("fin")
// fin.irr([-1000, 300, 400, 500], 10, "half_even")                         // => 0.0889633947d
// ```
//
//kavun:fn irr slot=23 pure (cf, scale, mode) | (cf, solver, scale, mode) -> decimal
func finIRR(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("irr", args, 3, 4)
	cf := a.flows(0, "cf")
	s := a.solver(1, len(args) == 4)
	out := a.rounding(len(args) - 2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.IRR(cf, s, out)
	return finResult(a.name, d, err)
}

// Answer the modified internal rate of return of the cashflows `cf`.
//
//kavun:fn mirr slot=24 pure (cf, finance_rate, reinvest_rate, scale, mode) -> decimal
func finMIRR(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("mirr", args)
	cf, finance, reinvest, out := a.flows(0, "cf"), a.dec(1, "finance_rate"), a.dec(2, "reinvest_rate"), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.MIRR(cf, finance, reinvest, out)
	return finResult(a.name, d, err)
}

// ---------------------------------------------------------------------------------------------------------------
// depreciation

// Answer the straight-line depreciation per period (`SLN`).
//
// ```go
// fin := import("fin")
// fin.straight_line(10000, 1000, 9, 2, "half_even")                        // => 1000.00d — a year
// ```
//
//kavun:fn straight_line slot=25 pure (cost, salvage, life, scale, mode) -> decimal
func finStraightLine(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("straight_line", args)
	cost, salvage, life, out := a.dec(0, "cost"), a.dec(1, "salvage"), a.int(2, "life"), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.StraightLine(cost, salvage, life, out)
	return finResult(a.name, d, err)
}

// Answer the sum-of-years'-digits depreciation of `period` (`SYD`).
//
//kavun:fn sum_of_digits slot=26 pure (cost, salvage, life, period, scale, mode) -> decimal
func finSumOfDigits(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("sum_of_digits", args)
	cost, salvage, life, period, out := a.dec(0, "cost"), a.dec(1, "salvage"), a.int(2, "life"), a.int(3, "period"), a.rounding(4)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.SumOfDigits(cost, salvage, life, period, out)
	return finResult(a.name, d, err)
}

// Answer one period's declining-balance depreciation of `book`.
//
//kavun:fn declining_balance slot=27 pure (book, salvage, rate, scale, mode) -> decimal
func finDecliningBalance(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("declining_balance", args)
	book, salvage, rate, out := a.dec(0, "book"), a.dec(1, "salvage"), a.dec(2, "rate"), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.DecliningBalance(book, salvage, rate, out)
	return finResult(a.name, d, err)
}

// Answer the declining-balance rate for a `factor` over `life`.
//
//kavun:fn declining_rate_from_factor slot=28 pure (factor, life, scale, mode) -> decimal
func finDecliningRateFromFactor(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("declining_rate_from_factor", args)
	factor, life, out := a.dec(0, "factor"), a.int(1, "life"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.DecliningRateFromFactor(factor, life, out)
	return finResult(a.name, d, err)
}

// Answer the declining-balance rate that reaches `salvage` from `cost` over `life`.
//
//kavun:fn declining_rate_from_salvage slot=29 pure (cost, salvage, life, scale, mode) -> decimal
func finDecliningRateFromSalvage(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("declining_rate_from_salvage", args)
	cost, salvage, life, out := a.dec(0, "cost"), a.dec(1, "salvage"), a.int(2, "life"), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.DecliningRateFromSalvage(cost, salvage, life, out)
	return finResult(a.name, d, err)
}

// ---------------------------------------------------------------------------------------------------------------
// solver

// Answer the frozen default solver spec `{lo, hi, tolerance, max_iter}`.
//
//kavun:fn default_solver slot=30 pure () -> record
func finDefaultSolver(_ core.VM, args []core.Value) (core.Value, error) {
	s := fin128.DefaultSolver()
	return core.NewRecordValue(map[string]core.Value{
		"lo":        core.NewDecimalValue(s.Lo),
		"hi":        core.NewDecimalValue(s.Hi),
		"tolerance": core.IntValue(s.Tolerance),
		"max_iter":  core.IntValue(int64(s.MaxIter)),
	}, false), nil
}

// ---------------------------------------------------------------------------------------------------------------
// fin.year_fraction and day count

// Answer a [`fin.year_fraction`](../types/fin_year_fraction.md) stated directly, or `0/1` with no arguments.
//
//kavun:fn year_fraction slot=31 pure () | (n1, d1[, n2, d2]) -> fin.year_fraction
func finYearFraction(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return NewYearFractionValue(daycount.Zero), nil
	}
	a := newFinArgs("year_fraction", args, 2, 4)
	var t [4]int32
	if a.err == nil {
		for i, pos := range []string{"n1", "d1", "n2", "d2"}[:len(args)] {
			t[i] = a.int32(i, pos)
		}
	}
	if a.err != nil {
		return core.Undefined, a.err
	}
	for _, i := range []int{1, 3} {
		if i < len(args) && (t[i] < 1 || t[i] > MaxYearFractionDenominator) {
			return core.Undefined, errs.NewInvalidValueError(fmt.Sprintf("(fin.year_fraction) denominator must be between 1 and %d", MaxYearFractionDenominator))
		}
	}
	f := daycount.Fraction{N1: t[0], D1: t[1], N2: t[2], D2: t[3]}
	if err := CheckYearFraction(f); err != nil {
		return core.Undefined, errs.NewInvalidValueError("(fin.year_fraction) " + err.Error())
	}
	return NewYearFractionValue(f), nil
}

// Report whether `x` is a `fin.year_fraction`.
//
//kavun:fn is_year_fraction slot=32 pure (x) -> bool
func finIsYearFraction(_ core.VM, args []core.Value) (core.Value, error) {
	return core.BoolValue(args[0].Type == value.FinYearFraction), nil
}

func finBetween(name string, args []core.Value, f func(c daycount.Convention, start, end civil.Date) daycount.Fraction) (core.Value, error) {
	a := newFinArgs(name, args)
	start, end, c := a.date(0, "start"), a.date(1, "end"), a.convention(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	r := f(c, start, end)
	if err := CheckYearFraction(r); err != nil {
		return finRaise(name, fin128.ErrFraction)
	}
	return NewYearFractionValue(r), nil
}

// Answer the year fraction from `start` to `end` under a day-count `convention`.
//
//kavun:fn year_fraction_between slot=33 pure (start, end, convention) -> fin.year_fraction
func finYearFractionBetween(_ core.VM, args []core.Value) (core.Value, error) {
	return finBetween("year_fraction_between", args, daycount.Convention.YearFraction)
}

// Answer the year fraction to `end` as the contract's termination date; see
// [`year_fraction_between`](#year_fraction_between).
//
//kavun:fn year_fraction_between_final slot=34 pure (start, end, convention) -> fin.year_fraction
func finYearFractionBetweenFinal(_ core.VM, args []core.Value) (core.Value, error) {
	return finBetween("year_fraction_between_final", args, daycount.Convention.YearFractionFinal)
}

func finDays(name string, args []core.Value, f func(c daycount.Convention, start, end civil.Date) int32) (core.Value, error) {
	a := newFinArgs(name, args)
	start, end, c := a.date(0, "start"), a.date(1, "end"), a.convention(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	return core.IntValue(int64(f(c, start, end))), nil
}

// Answer the days from `start` to `end` as the `convention` counts them.
//
//kavun:fn days_between slot=35 pure (start, end, convention) -> int
func finDaysBetween(_ core.VM, args []core.Value) (core.Value, error) {
	return finDays("days_between", args, daycount.Convention.Days)
}

// Answer the convention's day count to `end` as the contract's termination date; see [`days_between`](#days_between).
//
//kavun:fn days_between_final slot=36 pure (start, end, convention) -> int
func finDaysBetweenFinal(_ core.VM, args []core.Value) (core.Value, error) {
	return finDays("days_between_final", args, daycount.Convention.DaysFinal)
}

// Answer the year length the `convention` uses on the date `on`.
//
//kavun:fn days_in_year slot=37 pure (on, convention) -> int
func finDaysInYear(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("days_in_year", args)
	on, c := a.date(0, "on"), a.convention(1)
	if a.err != nil {
		return core.Undefined, a.err
	}
	return core.IntValue(int64(c.DaysInYear(on))), nil
}

// Answer the canonical day-count convention names.
//
//kavun:fn conventions slot=38 pure () -> [string]
func finConventions(_ core.VM, args []core.Value) (core.Value, error) {
	names := daycount.Names()
	out := make([]core.Value, len(names))
	for i, n := range names {
		out[i] = core.NewStringValue(n)
	}
	return core.NewArrayValue(out, false), nil
}

// ---------------------------------------------------------------------------------------------------------------
// functions of a year fraction

func finRateFraction(name string, args []core.Value, f func(rate dec128.Dec128, yf daycount.Fraction, out fin128.Rounding) (dec128.Dec128, error)) (core.Value, error) {
	a := newFinArgs(name, args)
	rate, yf, out := a.dec(0, "rate"), a.fraction(1, "f"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := f(rate, yf, out)
	return finResult(name, d, err)
}

// Answer the compound factor over a year fraction `f`.
//
//kavun:fn compound_factor_for slot=39 pure (rate, f, scale, mode) -> decimal
func finCompoundFactorFor(_ core.VM, args []core.Value) (core.Value, error) {
	return finRateFraction("compound_factor_for", args, fin128.CompoundFactorFor)
}

// Answer the discount factor over a year fraction `f`.
//
//kavun:fn discount_factor_for slot=40 pure (rate, f, scale, mode) -> decimal
func finDiscountFactorFor(_ core.VM, args []core.Value) (core.Value, error) {
	return finRateFraction("discount_factor_for", args, fin128.DiscountFactorFor)
}

// Answer the simple interest on `principal` over a year fraction `f`.
//
//kavun:fn accrue_simple slot=41 pure (principal, rate, f, scale, mode) -> decimal
func finAccrueSimple(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("accrue_simple", args)
	p, rate, yf, out := a.dec(0, "principal"), a.dec(1, "rate"), a.fraction(2, "f"), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.AccrueSimple(p, rate, yf, out)
	return finResult(a.name, d, err)
}

// Answer the compound interest on `principal` over a year fraction `f`, compounding `per_year` times a year.
//
//kavun:fn accrue_compound slot=42 pure (principal, rate, f, per_year, scale, mode) -> decimal
func finAccrueCompound(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("accrue_compound", args)
	p, rate, yf, perYear, out := a.dec(0, "principal"), a.dec(1, "rate"), a.fraction(2, "f"), a.int32(3, "per_year"), a.rounding(4)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.AccrueCompound(p, rate, yf, perYear, out)
	return finResult(a.name, d, err)
}

func finDiscount(name, first, second string, args []core.Value, f func(x, y dec128.Dec128, yf daycount.Fraction, out fin128.Rounding) (dec128.Dec128, error)) (core.Value, error) {
	a := newFinArgs(name, args)
	x, y, yf, out := a.dec(0, first), a.dec(1, second), a.fraction(2, "f"), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := f(x, y, yf, out)
	return finResult(name, d, err)
}

// Answer the price of a discount instrument.
//
//kavun:fn discount_price slot=43 pure (redemption, rate, f, scale, mode) -> decimal
func finDiscountPrice(_ core.VM, args []core.Value) (core.Value, error) {
	return finDiscount("discount_price", "redemption", "rate", args, fin128.DiscountPrice)
}

// Answer the discount rate of a discount instrument.
//
//kavun:fn discount_rate slot=44 pure (price, redemption, f, scale, mode) -> decimal
func finDiscountRate(_ core.VM, args []core.Value) (core.Value, error) {
	return finDiscount("discount_rate", "price", "redemption", args, fin128.DiscountRate)
}

// Answer the yield of a discount instrument.
//
//kavun:fn discount_yield slot=45 pure (price, redemption, f, scale, mode) -> decimal
func finDiscountYield(_ core.VM, args []core.Value) (core.Value, error) {
	return finDiscount("discount_yield", "price", "redemption", args, fin128.DiscountYield)
}

// ---------------------------------------------------------------------------------------------------------------
// dated cashflows

// cashflows reads an array of {date, amount} records (or dicts) with exactly those keys, mirroring fin128's
// Cashflow. Order is the caller's: an unsorted stream raises in fin128 (the first date is the discount base, so
// sorting it silently would change the answer).
func (a *finArgs) cashflows(i int, pos string) []fin128.Cashflow {
	if a.err != nil {
		return nil
	}
	v := a.args[i]
	if v.Type != value.Array {
		a.err = errs.NewInvalidArgumentTypeError("fin."+a.name, pos, "array", v.TypeName())
		return nil
	}
	ctx := "fin." + a.name
	elems := (*core.Array)(v.Ptr).Elements
	out := make([]fin128.Cashflow, len(elems))
	for k, e := range elems {
		at := fmt.Sprintf("%s[%d]", pos, k)
		var m map[string]core.Value
		switch e.Type {
		case value.Record:
			m = (*core.Record)(e.Ptr).Elements
		case value.Dict:
			m = (*core.Dict)(e.Ptr).Elements
		default:
			a.err = errs.NewInvalidArgumentTypeError(ctx, at, "record or dict", e.TypeName())
			return nil
		}
		for _, key := range slices.Sorted(maps.Keys(m)) {
			if key != "date" && key != "amount" {
				a.err = errs.NewInvalidValueError(fmt.Sprintf("(%s) %s: unknown key %q, expected: date, amount", ctx, at, key))
				return nil
			}
		}
		dv, okD := m["date"]
		av, okA := m["amount"]
		if !okD || !okA {
			a.err = errs.NewInvalidValueError(fmt.Sprintf("(%s) %s: a cashflow needs both date and amount", ctx, at))
			return nil
		}
		if dv.Type != value.Date {
			a.err = errs.NewInvalidArgumentTypeError(ctx, at+".date", "date", dv.TypeName())
			return nil
		}
		amount, err := core.DecimalOperandArg(ctx, at+".amount", av)
		if err != nil {
			a.err = err
			return nil
		}
		d, _ := dv.AsDate()
		out[k] = fin128.Cashflow{Date: d, Amount: amount}
	}
	return out
}

// Answer the net present value of dated cashflows.
//
//kavun:fn xnpv slot=46 pure (rate, flows, convention, scale, mode) -> decimal
func finXNPV(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("xnpv", args)
	rate, cf, c, out := a.dec(0, "rate"), a.cashflows(1, "flows"), a.convention(2), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.XNPV(rate, cf, c, out)
	return finResult(a.name, d, err)
}

// Solve for the internal rate of return of dated cashflows.
//
//kavun:fn xirr slot=47 pure (flows, convention, scale, mode) | (flows, convention, solver, scale, mode) -> decimal
func finXIRR(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("xirr", args, 4, 5)
	cf, c := a.cashflows(0, "flows"), a.convention(1)
	s := a.solver(2, len(args) == 5)
	out := a.rounding(len(args) - 2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.XIRR(cf, c, s, out)
	return finResult(a.name, d, err)
}

// ---------------------------------------------------------------------------------------------------------------
// rate and charge tables

// The constructors take the table's text form or an array of band records; with no argument they answer the
// type's default, the zero-rate table. tiered_charges with records takes the bounds too: (bands, min, max).

// Build a [`fin.tiered_rates`](../types/fin_tiered_rates.md) from its text form or band records; with no argument, the
// zero-rate table.
//
//kavun:fn tiered_rates slot=48 pure ([table]) -> fin.tiered_rates
func finTieredRates(_ core.VM, args []core.Value) (core.Value, error) {
	switch len(args) {
	case 0:
		return NewTieredRatesValue(TieredRatesDefault), nil
	case 1:
		return NewTieredRatesFrom("fin.tiered_rates", args[0])
	}
	return core.Undefined, core.ErrArgCount
}

// Build a [`fin.tiered_charges`](../types/fin_tiered_charges.md) from its text form, or from band records with `min`
// and `max`; with no argument, the zero table.
//
//kavun:fn tiered_charges slot=49 pure () | (text) | (bands, min, max) -> fin.tiered_charges
func finTieredCharges(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return NewTieredChargesValue(TieredChargesDefault), nil
	}
	return tieredChargesFrom("fin.tiered_charges", args)
}

// Build a [`fin.dated_rates`](../types/fin_dated_rates.md) from its text form or band records; with no argument, the
// zero-rate table.
//
//kavun:fn dated_rates slot=50 pure ([table]) -> fin.dated_rates
func finDatedRates(_ core.VM, args []core.Value) (core.Value, error) {
	switch len(args) {
	case 0:
		return NewDatedRatesValue(DatedRatesDefault), nil
	case 1:
		return NewDatedRatesFrom("fin.dated_rates", args[0])
	}
	return core.Undefined, core.ErrArgCount
}

// Build a [`fin.dated_charges`](../types/fin_dated_charges.md) from its text form or band records; with no argument,
// the zero table.
//
//kavun:fn dated_charges slot=51 pure ([table]) -> fin.dated_charges
func finDatedCharges(_ core.VM, args []core.Value) (core.Value, error) {
	switch len(args) {
	case 0:
		return NewDatedChargesValue(DatedChargesDefault), nil
	case 1:
		return NewDatedChargesFrom("fin.dated_charges", args[0])
	}
	return core.Undefined, core.ErrArgCount
}

// finIsType builds a fin.is_<table>(x) predicate.
func finIsType(t uint8, name string) core.NativeFunc {
	return func(_ core.VM, args []core.Value) (core.Value, error) {
		return core.BoolValue(args[0].Type == t), nil
	}
}

// Report whether `x` is a `fin.tiered_rates`.
//
//kavun:fn is_tiered_rates slot=52 pure (x) -> bool
func finIsTieredRates(vm core.VM, args []core.Value) (core.Value, error) {
	return finIsTieredRatesFn(vm, args)
}

var finIsTieredRatesFn = finIsType(value.FinTieredRates, "is_tiered_rates")

// Report whether `x` is a `fin.tiered_charges`.
//
//kavun:fn is_tiered_charges slot=53 pure (x) -> bool
func finIsTieredCharges(vm core.VM, args []core.Value) (core.Value, error) {
	return finIsTieredChargesFn(vm, args)
}

var finIsTieredChargesFn = finIsType(value.FinTieredCharges, "is_tiered_charges")

// Report whether `x` is a `fin.dated_rates`.
//
//kavun:fn is_dated_rates slot=54 pure (x) -> bool
func finIsDatedRates(vm core.VM, args []core.Value) (core.Value, error) {
	return finIsDatedRatesFn(vm, args)
}

var finIsDatedRatesFn = finIsType(value.FinDatedRates, "is_dated_rates")

// Report whether `x` is a `fin.dated_charges`.
//
//kavun:fn is_dated_charges slot=55 pure (x) -> bool
func finIsDatedCharges(vm core.VM, args []core.Value) (core.Value, error) {
	return finIsDatedChargesFn(vm, args)
}

var finIsDatedChargesFn = finIsType(value.FinDatedCharges, "is_dated_charges")

// ---------------------------------------------------------------------------------------------------------------
// solver with a script callback

// finCallback adapts a callable that accepts one argument (core.CallbackForm: func(x), func(x, ...a), a bound
// member or builtin of pair (0|1, variadic)) into fin128's func(Dec128) Dec128. fin128's callback has no
// error channel, so the first failure — a raise inside the script, or a result that is not decimal|int — is
// stashed, every later evaluation answers NaN without calling the script again (so fin128 stops at once), and
// the caller re-raises the stashed error itself: the script sees its own error, never fin128's ErrBracket or
// ErrNotConverged. A fatal error stays fatal. The stash is per call, so a callback may itself call fin.root.
type finCallback struct {
	vm  core.VM
	ctx string
	fn  core.Value
	err error
	nan dec128.Dec128
	buf [1]core.Value // the argument buffer, reused: the callee never retains or mutates its arguments
}

// newFinCallback validates the callback argument f of fin.root/brackets: a callable that accepts one argument.
func newFinCallback(vm core.VM, ctx string, fn core.Value) (*finCallback, error) {
	if !fn.IsCallable() {
		return nil, errs.NewInvalidArgumentTypeError(ctx, "f", "function", fn.TypeName())
	}
	if _, ok := core.CallbackForm(fn, 1, 1); !ok {
		got := fmt.Sprintf("function of %d arguments", fn.Arity())
		if fn.IsVariadic() {
			got = fmt.Sprintf("function of %d or more arguments", fn.Arity())
		}
		return nil, errs.NewInvalidArgumentTypeError(ctx, "f", "function of one argument", got)
	}
	return &finCallback{vm: vm, ctx: ctx, fn: fn, nan: dec128.NaN(state.DomainError)}, nil
}

func (c *finCallback) eval(x dec128.Dec128) dec128.Dec128 {
	if c.err != nil {
		return c.nan
	}
	c.buf[0] = core.NewDecimalValue(x)
	r, err := c.fn.Call(c.vm, c.buf[:])
	if err != nil {
		c.err = err
		return c.nan
	}
	d, err := core.DecimalOperandArg(c.ctx, "f's result", r)
	if err != nil {
		c.err = err
		return c.nan
	}
	return d
}

// Answer the `x` where the function `f` crosses zero; see [The solver](#the-solver-root-and-brackets).
//
// `f` is any callable that accepts one argument: it gets the `decimal` `x`.
//
//kavun:fn root slot=56 impure (f, scale, mode) | (f, solver, scale, mode) -> decimal
func finRoot(vm core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("root", args, 3, 4)
	if a.err != nil {
		return core.Undefined, a.err
	}
	cb, err := newFinCallback(vm, "fin.root", args[0])
	if err != nil {
		return core.Undefined, err
	}
	s := a.solver(1, len(args) == 4)
	out := a.rounding(len(args) - 2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.Root(cb.eval, s, out)
	if cb.err != nil {
		return core.Undefined, cb.err
	}
	return finResult(a.name, d, err)
}

// Report whether `f` changes sign across the solver bracket; see [The solver](#the-solver-root-and-brackets).
//
// `f` is any callable that accepts one argument: it gets the `decimal` `x`.
//
//kavun:fn brackets slot=57 impure (f[, solver]) -> bool
func finBrackets(vm core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("brackets", args, 1, 2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	cb, err := newFinCallback(vm, "fin.brackets", args[0])
	if err != nil {
		return core.Undefined, err
	}
	s := a.solver(1, len(args) == 2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	ok := fin128.Brackets(cb.eval, s)
	if cb.err != nil {
		return core.Undefined, cb.err
	}
	return core.BoolValue(ok), nil
}
