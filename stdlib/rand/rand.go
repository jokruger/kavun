package rand

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// The rand module is pseudo-random numbers as a VALUE: rand.generator(seed) answers a generator whose draws are its
// own state, so the same seed gives the same sequence on every build and nothing is shared between scripts. It is
// pure — it reads nothing from the world; entropy comes from os.rand_generator(), which answers the same type.
func init() {
	m := core.NewModule(value.ModuleRand, "rand", true)
	m.Type(value.RandGenerator, TypeGenerator)
	m.PureFn(members.Generator, 0, randGenerator, 1, core.Fixed)
	// 1..127 reserved
	if err := m.Register(); err != nil {
		panic(err)
	}
}

// rand.generator(seed): a seeded generator (PCG); the seed is required — a pure module cannot pick one itself.
func randGenerator(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.generator", "1", len(args))
	}
	if args[0].Type != value.Int {
		return core.Undefined, errs.NewInvalidArgumentTypeError("rand.generator", "first", "int", args[0].TypeName())
	}
	return NewSeededValue(int64(args[0].Data)), nil
}
