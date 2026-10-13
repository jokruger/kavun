package num

import (
	"math/big"

	"github.com/jokruger/dec128"
)

// DecimalToExactRat converts d to its true exact value as ±coefficient/10^scale — never through a
// lossy intermediate (no ToString()/parse, no float64) — for use in exact cross-type comparisons
// against float, where dec128's own fixed ~34-digit precision can't hold an arbitrary float64's
// exact decimal expansion but math/big.Rat, being arbitrary-precision, always can. Caller must
// exclude NaN first — a NaN decimal has no coefficient/scale worth converting.
func DecimalToExactRat(d *dec128.Dec128) *big.Rat {
	coef := d.Coefficient().BigInt()
	if d.IsNegative() {
		coef = new(big.Int).Neg(coef)
	}
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d.Scale())), nil)
	return new(big.Rat).SetFrac(coef, denom)
}
