package stdlib

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/fin128"
	"github.com/jokruger/fin128/civil"

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
		},
	)
}

// finRaise is the module's single translation point from a fin128/dec128 error to a Kavun error.
func finRaise(name string, err error) (core.Value, error) {
	ctx := "fin." + name
	switch {
	case errors.Is(err, state.DivisionByZero.Error()):
		return core.Undefined, errs.NewDivisionByZeroError()
	case errors.Is(err, fin128.ErrSyntax):
		return core.Undefined, errs.NewConversionError("string", ctx, strings.TrimPrefix(err.Error(), "fin128: "))
	case errors.Is(err, fin128.ErrRoundingUnset), errors.Is(err, fin128.ErrNotBuilt), errors.Is(err, fin128.ErrTiming),
		errors.Is(err, fin128.ErrRule), errors.Is(err, fin128.ErrSolverSpec):
		// the binding validates all of these before calling fin128: reaching one is a defect, not a script fault
		return core.Undefined, errs.NewInternalError(fmt.Sprintf("(%s) unexpected %v", ctx, err))
	}
	return core.Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", ctx, strings.TrimPrefix(err.Error(), "fin128: ")))
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
