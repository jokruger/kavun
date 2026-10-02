package stdlib

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
	"github.com/jokruger/kavun/core/module"
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

func init() {
	InitModule("fin", module.Fin,
		map[string]core.Value{},
		map[uint64]*core.BuiltinFunction{
			// units (exact, no rounding)
			0: core.NewBuiltinFunction("to_percent", finToPercent, 1, false, true),
			1: core.NewBuiltinFunction("from_percent", finFromPercent, 1, false, true),
			2: core.NewBuiltinFunction("to_basis_points", finToBasisPoints, 1, false, true),
			3: core.NewBuiltinFunction("from_basis_points", finFromBasisPoints, 1, false, true),
			// rounding / exact
			4: core.NewBuiltinFunction("apply_rate", finApplyRate, 4, false, true),
			5: core.NewBuiltinFunction("exact_product", finExactProduct, 2, false, true),
			6: core.NewBuiltinFunction("mul_div_round", finMulDivRound, 6, false, true),
			// rate conversion
			7: core.NewBuiltinFunction("nominal_to_effective", finNominalToEffective, 4, false, true),
			8: core.NewBuiltinFunction("effective_to_nominal", finEffectiveToNominal, 4, false, true),
			9: core.NewBuiltinFunction("per_year", finPerYear, 1, false, true),
			// factors
			10: core.NewBuiltinFunction("compound_factor", finCompoundFactor, 4, false, true),
			11: core.NewBuiltinFunction("discount_factor", finDiscountFactor, 4, false, true),
			// annuity / TVM
			12: core.NewBuiltinFunction("annuity_factor_fv", finAnnuityFactorFV, 5, false, true),
			13: core.NewBuiltinFunction("annuity_factor_pv", finAnnuityFactorPV, 5, false, true),
			14: core.NewBuiltinFunction("payment", finPayment, 7, false, true),
			15: core.NewBuiltinFunction("present_value", finPresentValue, 7, false, true),
			16: core.NewBuiltinFunction("future_value", finFutureValue, 7, false, true),
			17: core.NewBuiltinFunction("periods", finPeriods, 7, false, true),
			18: core.NewBuiltinFunction("rate", finRate, 7, true, true),
			// schedule
			19: core.NewBuiltinFunction("balance", finBalance, 8, false, true),
			20: core.NewBuiltinFunction("charge_part", finChargePart, 8, false, true),
			21: core.NewBuiltinFunction("principal_part", finPrincipalPart, 8, false, true),
			// cashflows
			22: core.NewBuiltinFunction("npv", finNPV, 4, false, true),
			23: core.NewBuiltinFunction("irr", finIRR, 3, true, true),
			24: core.NewBuiltinFunction("mirr", finMIRR, 5, false, true),
			// depreciation
			25: core.NewBuiltinFunction("straight_line", finStraightLine, 5, false, true),
			26: core.NewBuiltinFunction("sum_of_digits", finSumOfDigits, 6, false, true),
			27: core.NewBuiltinFunction("declining_balance", finDecliningBalance, 5, false, true),
			28: core.NewBuiltinFunction("declining_rate_from_factor", finDecliningRateFromFactor, 4, false, true),
			29: core.NewBuiltinFunction("declining_rate_from_salvage", finDecliningRateFromSalvage, 5, false, true),
			// solver
			30: core.NewBuiltinFunction("default_solver", finDefaultSolver, 0, false, true),
			// fin.year_fraction and day count
			31: core.NewBuiltinFunction("year_fraction", finYearFraction, 0, true, true),
			32: core.NewBuiltinFunction("is_year_fraction", finIsYearFraction, 1, false, true),
			33: core.NewBuiltinFunction("year_fraction_between", finYearFractionBetween, 3, false, true),
			34: core.NewBuiltinFunction("year_fraction_between_final", finYearFractionBetweenFinal, 3, false, true),
			35: core.NewBuiltinFunction("days_between", finDaysBetween, 3, false, true),
			36: core.NewBuiltinFunction("days_between_final", finDaysBetweenFinal, 3, false, true),
			37: core.NewBuiltinFunction("days_in_year", finDaysInYear, 2, false, true),
			38: core.NewBuiltinFunction("conventions", finConventions, 0, false, true),
			// functions of a year fraction
			39: core.NewBuiltinFunction("compound_factor_for", finCompoundFactorFor, 4, false, true),
			40: core.NewBuiltinFunction("discount_factor_for", finDiscountFactorFor, 4, false, true),
			41: core.NewBuiltinFunction("accrue_simple", finAccrueSimple, 5, false, true),
			42: core.NewBuiltinFunction("accrue_compound", finAccrueCompound, 6, false, true),
			43: core.NewBuiltinFunction("discount_price", finDiscountPrice, 5, false, true),
			44: core.NewBuiltinFunction("discount_rate", finDiscountRate, 5, false, true),
			45: core.NewBuiltinFunction("discount_yield", finDiscountYield, 5, false, true),
			// dated cashflows
			46: core.NewBuiltinFunction("xnpv", finXNPV, 5, false, true),
			47: core.NewBuiltinFunction("xirr", finXIRR, 4, true, true),
			// rate and charge tables
			48: core.NewBuiltinFunction("tiered_rates", finTieredRates, 0, true, true),
			49: core.NewBuiltinFunction("tiered_charges", finTieredCharges, 0, true, true),
			50: core.NewBuiltinFunction("dated_rates", finDatedRates, 0, true, true),
			51: core.NewBuiltinFunction("dated_charges", finDatedCharges, 0, true, true),
			52: core.NewBuiltinFunction("is_tiered_rates", finIsType(value.FinTieredRates, "is_tiered_rates"), 1, false, true),
			53: core.NewBuiltinFunction("is_tiered_charges", finIsType(value.FinTieredCharges, "is_tiered_charges"), 1, false, true),
			54: core.NewBuiltinFunction("is_dated_rates", finIsType(value.FinDatedRates, "is_dated_rates"), 1, false, true),
			55: core.NewBuiltinFunction("is_dated_charges", finIsType(value.FinDatedCharges, "is_dated_charges"), 1, false, true),
			// solver with a script callback — CALLABLE-DEPENDENT, so registered impure: the optimizer cannot prove a
			// callback pure, and must never fold a call that runs one
			56: core.NewBuiltinFunction("root", finRoot, 3, true, false),
			57: core.NewBuiltinFunction("brackets", finBrackets, 1, true, false),
		},
	)
}

// finRaise translates a fin128/dec128 error through core.FinError, the single translation point shared with the
// fin types' members.
func finRaise(name string, err error) (core.Value, error) {
	return core.Undefined, core.FinError("fin."+name, err)
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

func newFinArgs(name string, args []core.Value, counts ...int) *finArgs {
	a := &finArgs{name: name, args: args}
	for _, c := range counts {
		if len(args) == c {
			return a
		}
	}
	want := fmt.Sprint(counts[0])
	if len(counts) > 1 {
		want = fmt.Sprintf("%d or %d", counts[0], counts[1])
	}
	a.err = errs.NewWrongNumArgumentsError("fin."+name, want, len(args))
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
	return core.FinYearFractionOf(v)
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

// convention reads a day-count convention by name (core.FinConventionArg: market spellings and aliases).
func (a *finArgs) convention(i int) daycount.Convention {
	if a.err != nil {
		return daycount.Convention{}
	}
	c, err := core.FinConventionArg("fin."+a.name, "convention", a.args[i])
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
	a := newFinArgs(name, args, 1)
	v := a.dec(0, "value")
	if a.err != nil {
		return core.Undefined, a.err
	}
	return core.DecimalResult("fin."+name, f(v))
}

func finToPercent(_ core.VM, args []core.Value) (core.Value, error) {
	return finUnit("to_percent", args, fin128.ToPercent)
}

func finFromPercent(_ core.VM, args []core.Value) (core.Value, error) {
	return finUnit("from_percent", args, fin128.FromPercent)
}

func finToBasisPoints(_ core.VM, args []core.Value) (core.Value, error) {
	return finUnit("to_basis_points", args, fin128.ToBasisPoints)
}

func finFromBasisPoints(_ core.VM, args []core.Value) (core.Value, error) {
	return finUnit("from_basis_points", args, fin128.FromBasisPoints)
}

// ---------------------------------------------------------------------------------------------------------------
// rounding / exact

func finApplyRate(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("apply_rate", args, 4)
	amount, rate, out := a.dec(0, "amount"), a.dec(1, "rate"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.ApplyRate(amount, rate, out)
	return finResult(a.name, d, err)
}

func finExactProduct(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("exact_product", args, 2)
	x, y := a.dec(0, "a"), a.dec(1, "b")
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.ExactProduct(x, y)
	return finResult(a.name, d, err)
}

func finMulDivRound(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("mul_div_round", args, 6)
	x, y, num, den, out := a.dec(0, "a"), a.dec(1, "b"), a.int(2, "num"), a.int(3, "den"), a.rounding(4)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.MulDivRound(x, y, num, den, out)
	return finResult(a.name, d, err)
}

// ---------------------------------------------------------------------------------------------------------------
// rate conversion

func finNominalToEffective(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("nominal_to_effective", args, 4)
	r, m, out := a.dec(0, "nominal"), a.int32(1, "m"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.NominalToEffective(r, m, out)
	return finResult(a.name, d, err)
}

func finEffectiveToNominal(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("effective_to_nominal", args, 4)
	r, m, out := a.dec(0, "effective"), a.int32(1, "m"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.EffectiveToNominal(r, m, out)
	return finResult(a.name, d, err)
}

func finPerYear(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("per_year", args, 1)
	f := a.frequency(0)
	if a.err != nil {
		return core.Undefined, a.err
	}
	return core.IntValue(int64(f.PerYear())), nil
}

// ---------------------------------------------------------------------------------------------------------------
// factors

func finCompoundFactor(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("compound_factor", args, 4)
	rate, n, out := a.dec(0, "rate"), a.int(1, "n"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.CompoundFactor(rate, n, out)
	return finResult(a.name, d, err)
}

func finDiscountFactor(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("discount_factor", args, 4)
	rate, n, out := a.dec(0, "rate"), a.int(1, "n"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.DiscountFactor(rate, n, out)
	return finResult(a.name, d, err)
}

// ---------------------------------------------------------------------------------------------------------------
// annuity / TVM

func finAnnuityFactorFV(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("annuity_factor_fv", args, 5)
	rate, n, t, out := a.dec(0, "rate"), a.int(1, "n"), a.timing(2), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.AnnuityFactorFV(rate, n, t, out)
	return finResult(a.name, d, err)
}

func finAnnuityFactorPV(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("annuity_factor_pv", args, 5)
	rate, n, t, out := a.dec(0, "rate"), a.int(1, "n"), a.timing(2), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.AnnuityFactorPV(rate, n, t, out)
	return finResult(a.name, d, err)
}

func finPayment(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("payment", args, 7)
	rate, n, pv, fv, t, out := a.dec(0, "rate"), a.int(1, "n"), a.dec(2, "pv"), a.dec(3, "fv"), a.timing(4), a.rounding(5)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.Payment(rate, n, pv, fv, t, out)
	return finResult(a.name, d, err)
}

func finPresentValue(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("present_value", args, 7)
	rate, n, pmt, fv, t, out := a.dec(0, "rate"), a.int(1, "n"), a.dec(2, "pmt"), a.dec(3, "fv"), a.timing(4), a.rounding(5)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.PresentValue(rate, n, pmt, fv, t, out)
	return finResult(a.name, d, err)
}

func finFutureValue(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("future_value", args, 7)
	rate, n, pmt, pv, t, out := a.dec(0, "rate"), a.int(1, "n"), a.dec(2, "pmt"), a.dec(3, "pv"), a.timing(4), a.rounding(5)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.FutureValue(rate, n, pmt, pv, t, out)
	return finResult(a.name, d, err)
}

// finPeriods answers [whole, remainder]: the whole periods of pmt needed to move pv to fv, and the fraction of one
// further period — a two-part decomposition, destructured positionally like quo_rem.
func finPeriods(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("periods", args, 7)
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

// finRate solves for the periodic rate: rate(n, pmt, pv, fv, timing, [solver,] scale, mode).
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
	a := newFinArgs(name, args, 8)
	rate, period, n, pv, fv, t, out := a.dec(0, "rate"), a.int(1, "period"), a.int(2, "n"), a.dec(3, "pv"), a.dec(4, "fv"), a.timing(5), a.rounding(6)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := f(rate, period, n, pv, fv, t, out)
	return finResult(name, d, err)
}

func finBalance(_ core.VM, args []core.Value) (core.Value, error) {
	return finSchedule("balance", args, fin128.Balance)
}

func finChargePart(_ core.VM, args []core.Value) (core.Value, error) {
	return finSchedule("charge_part", args, fin128.ChargePart)
}

func finPrincipalPart(_ core.VM, args []core.Value) (core.Value, error) {
	return finSchedule("principal_part", args, fin128.PrincipalPart)
}

// ---------------------------------------------------------------------------------------------------------------
// cashflows

// finNPV is the net present value of equally spaced flows; cf[0] sits at t = 0 and is NOT discounted (unlike a
// spreadsheet's NPV — fin128 follows the definition and numpy-financial).
func finNPV(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("npv", args, 4)
	rate, cf, out := a.dec(0, "rate"), a.flows(1, "cf"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.NPV(rate, cf, out)
	return finResult(a.name, d, err)
}

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

func finMIRR(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("mirr", args, 5)
	cf, finance, reinvest, out := a.flows(0, "cf"), a.dec(1, "finance_rate"), a.dec(2, "reinvest_rate"), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.MIRR(cf, finance, reinvest, out)
	return finResult(a.name, d, err)
}

// ---------------------------------------------------------------------------------------------------------------
// depreciation

func finStraightLine(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("straight_line", args, 5)
	cost, salvage, life, out := a.dec(0, "cost"), a.dec(1, "salvage"), a.int(2, "life"), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.StraightLine(cost, salvage, life, out)
	return finResult(a.name, d, err)
}

func finSumOfDigits(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("sum_of_digits", args, 6)
	cost, salvage, life, period, out := a.dec(0, "cost"), a.dec(1, "salvage"), a.int(2, "life"), a.int(3, "period"), a.rounding(4)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.SumOfDigits(cost, salvage, life, period, out)
	return finResult(a.name, d, err)
}

func finDecliningBalance(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("declining_balance", args, 5)
	book, salvage, rate, out := a.dec(0, "book"), a.dec(1, "salvage"), a.dec(2, "rate"), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.DecliningBalance(book, salvage, rate, out)
	return finResult(a.name, d, err)
}

func finDecliningRateFromFactor(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("declining_rate_from_factor", args, 4)
	factor, life, out := a.dec(0, "factor"), a.int(1, "life"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.DecliningRateFromFactor(factor, life, out)
	return finResult(a.name, d, err)
}

func finDecliningRateFromSalvage(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("declining_rate_from_salvage", args, 5)
	cost, salvage, life, out := a.dec(0, "cost"), a.dec(1, "salvage"), a.int(2, "life"), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.DecliningRateFromSalvage(cost, salvage, life, out)
	return finResult(a.name, d, err)
}

// ---------------------------------------------------------------------------------------------------------------
// solver

// finDefaultSolver answers the frozen defaults the solver-taking functions lay a partial spec over.
func finDefaultSolver(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fin.default_solver", "0", len(args))
	}
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

// finYearFraction is the constructor: a fraction stated directly — fin.year_fraction(n1, d1[, n2, d2]), the
// counted days over the year length — or the default 0/1 with no arguments. The bounds are the type's invariant:
// numerators int32, denominators 1…65535.
func finYearFraction(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.NewFinYearFractionValue(daycount.Zero), nil
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
		if i < len(args) && (t[i] < 1 || t[i] > core.MaxFinYearFractionDenominator) {
			return core.Undefined, errs.NewInvalidValueError(fmt.Sprintf("(fin.year_fraction) denominator must be between 1 and %d", core.MaxFinYearFractionDenominator))
		}
	}
	f := daycount.Fraction{N1: t[0], D1: t[1], N2: t[2], D2: t[3]}
	if err := core.CheckFinYearFraction(f); err != nil {
		return core.Undefined, errs.NewInvalidValueError("(fin.year_fraction) " + err.Error())
	}
	return core.NewFinYearFractionValue(f), nil
}

func finIsYearFraction(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fin.is_year_fraction", "1", len(args))
	}
	return core.BoolValue(args[0].Type == value.FinYearFraction), nil
}

func finBetween(name string, args []core.Value, f func(c daycount.Convention, start, end civil.Date) daycount.Fraction) (core.Value, error) {
	a := newFinArgs(name, args, 3)
	start, end, c := a.date(0, "start"), a.date(1, "end"), a.convention(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	r := f(c, start, end)
	if err := core.CheckFinYearFraction(r); err != nil {
		return finRaise(name, fin128.ErrFraction)
	}
	return core.NewFinYearFractionValue(r), nil
}

// finYearFractionBetween is the year fraction a convention measures from start to end (fin128 YearFraction).
func finYearFractionBetween(_ core.VM, args []core.Value) (core.Value, error) {
	return finBetween("year_fraction_between", args, daycount.Convention.YearFraction)
}

// finYearFractionBetweenFinal is year_fraction_between with end taken as the contract's termination date
// (only 30E-ISDA/360 distinguishes the two, at a February-end termination).
func finYearFractionBetweenFinal(_ core.VM, args []core.Value) (core.Value, error) {
	return finBetween("year_fraction_between_final", args, daycount.Convention.YearFractionFinal)
}

func finDays(name string, args []core.Value, f func(c daycount.Convention, start, end civil.Date) int32) (core.Value, error) {
	a := newFinArgs(name, args, 3)
	start, end, c := a.date(0, "start"), a.date(1, "end"), a.convention(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	return core.IntValue(int64(f(c, start, end))), nil
}

// finDaysBetween is the days the convention COUNTS from start to end (30 for a February under 30E/360) — the
// calendar's answer is d2 - d1.
func finDaysBetween(_ core.VM, args []core.Value) (core.Value, error) {
	return finDays("days_between", args, daycount.Convention.Days)
}

func finDaysBetweenFinal(_ core.VM, args []core.Value) (core.Value, error) {
	return finDays("days_between_final", args, daycount.Convention.DaysFinal)
}

// finDaysInYear is the year length the convention takes on's year to have (360 under ACT/360 even in a leap
// year) — the calendar's answer is times.days_in_year(y).
func finDaysInYear(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("days_in_year", args, 2)
	on, c := a.date(0, "on"), a.convention(1)
	if a.err != nil {
		return core.Undefined, a.err
	}
	return core.IntValue(int64(c.DaysInYear(on))), nil
}

// finConventions answers the canonical name of every convention, sorted.
func finConventions(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("fin.conventions", "0", len(args))
	}
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
	a := newFinArgs(name, args, 4)
	rate, yf, out := a.dec(0, "rate"), a.fraction(1, "f"), a.rounding(2)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := f(rate, yf, out)
	return finResult(name, d, err)
}

func finCompoundFactorFor(_ core.VM, args []core.Value) (core.Value, error) {
	return finRateFraction("compound_factor_for", args, fin128.CompoundFactorFor)
}

func finDiscountFactorFor(_ core.VM, args []core.Value) (core.Value, error) {
	return finRateFraction("discount_factor_for", args, fin128.DiscountFactorFor)
}

// finAccrueSimple is principal × annual rate × f, rounded once — exact.
func finAccrueSimple(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("accrue_simple", args, 5)
	p, rate, yf, out := a.dec(0, "principal"), a.dec(1, "rate"), a.fraction(2, "f"), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.AccrueSimple(p, rate, yf, out)
	return finResult(a.name, d, err)
}

// finAccrueCompound is the interest on principal at a nominal annual rate compounded per_year times a year, over f.
func finAccrueCompound(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("accrue_compound", args, 6)
	p, rate, yf, perYear, out := a.dec(0, "principal"), a.dec(1, "rate"), a.fraction(2, "f"), a.int32(3, "per_year"), a.rounding(4)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.AccrueCompound(p, rate, yf, perYear, out)
	return finResult(a.name, d, err)
}

func finDiscount(name, first, second string, args []core.Value, f func(x, y dec128.Dec128, yf daycount.Fraction, out fin128.Rounding) (dec128.Dec128, error)) (core.Value, error) {
	a := newFinArgs(name, args, 5)
	x, y, yf, out := a.dec(0, first), a.dec(1, second), a.fraction(2, "f"), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := f(x, y, yf, out)
	return finResult(name, d, err)
}

func finDiscountPrice(_ core.VM, args []core.Value) (core.Value, error) {
	return finDiscount("discount_price", "redemption", "rate", args, fin128.DiscountPrice)
}

func finDiscountRate(_ core.VM, args []core.Value) (core.Value, error) {
	return finDiscount("discount_rate", "price", "redemption", args, fin128.DiscountRate)
}

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

// finXNPV is the net present value of dated flows at an annual effective rate, each discounted by the year
// fraction from the first flow's date under the stated convention.
func finXNPV(_ core.VM, args []core.Value) (core.Value, error) {
	a := newFinArgs("xnpv", args, 5)
	rate, cf, c, out := a.dec(0, "rate"), a.cashflows(1, "flows"), a.convention(2), a.rounding(3)
	if a.err != nil {
		return core.Undefined, a.err
	}
	d, err := fin128.XNPV(rate, cf, c, out)
	return finResult(a.name, d, err)
}

// finXIRR is the annual effective rate at which xnpv of the flows is zero: xirr(flows, convention, [solver,] s, m).
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

func finTieredRates(_ core.VM, args []core.Value) (core.Value, error) {
	switch len(args) {
	case 0:
		return core.NewFinTieredRatesValue(core.FinTieredRatesDefault), nil
	case 1:
		return core.NewFinTieredRatesFrom("fin.tiered_rates", args[0])
	}
	return core.Undefined, errs.NewWrongNumArgumentsError("fin.tiered_rates", "0 or 1", len(args))
}

func finTieredCharges(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.NewFinTieredChargesValue(core.FinTieredChargesDefault), nil
	}
	return core.NewFinTieredChargesFrom("fin.tiered_charges", args)
}

func finDatedRates(_ core.VM, args []core.Value) (core.Value, error) {
	switch len(args) {
	case 0:
		return core.NewFinDatedRatesValue(core.FinDatedRatesDefault), nil
	case 1:
		return core.NewFinDatedRatesFrom("fin.dated_rates", args[0])
	}
	return core.Undefined, errs.NewWrongNumArgumentsError("fin.dated_rates", "0 or 1", len(args))
}

func finDatedCharges(_ core.VM, args []core.Value) (core.Value, error) {
	switch len(args) {
	case 0:
		return core.NewFinDatedChargesValue(core.FinDatedChargesDefault), nil
	case 1:
		return core.NewFinDatedChargesFrom("fin.dated_charges", args[0])
	}
	return core.Undefined, errs.NewWrongNumArgumentsError("fin.dated_charges", "0 or 1", len(args))
}

// finIsType builds a fin.is_<table>(x) predicate.
func finIsType(t uint8, name string) core.NativeFunc {
	return func(_ core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("fin."+name, "1", len(args))
		}
		return core.BoolValue(args[0].Type == t), nil
	}
}

// ---------------------------------------------------------------------------------------------------------------
// solver with a script callback

// finCallback adapts a script function of one decimal into fin128's func(Dec128) Dec128. fin128's callback has no
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
}

func newFinCallback(vm core.VM, ctx string, fn core.Value) (*finCallback, error) {
	if !fn.IsCallable() {
		return nil, errs.NewInvalidArgumentTypeError(ctx, "f", "function", fn.TypeName())
	}
	if fn.IsVariadic() || fn.Arity() != 1 {
		got := fmt.Sprintf("function of %d arguments", fn.Arity())
		if fn.IsVariadic() {
			got = "variadic function"
		}
		return nil, errs.NewInvalidArgumentTypeError(ctx, "f", "function of one argument", got)
	}
	return &finCallback{vm: vm, ctx: ctx, fn: fn, nan: dec128.NaN(state.DomainError)}, nil
}

func (c *finCallback) eval(x dec128.Dec128) dec128.Dec128 {
	if c.err != nil {
		return c.nan
	}
	r, err := c.fn.Call(c.vm, []core.Value{core.NewDecimalValue(x)})
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

// finRoot finds x where f(x) = 0 by bisection over the solver bracket: root(f, [solver,] scale, mode).
//
// CALLABLE-DEPENDENT: as pure as f.
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

// finBrackets reports whether f changes sign across the solver bracket — whether root has a root to find there:
// brackets(f[, solver]). It tells a bracket that merely missed (widen hi) from a function with no root at all.
//
// CALLABLE-DEPENDENT: as pure as f.
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
