package stdlib

import (
	"math/rand"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func init() {
	// 9..127 reserved
	InitModule("rand", core.BI_MOD_RAND, nil, nil, map[uint64]*core.BuiltinFunction{
		0: core.NewBuiltinFunction("int", randInt63, 0, false),
		1: core.NewBuiltinFunction("float", randFloat64, 0, false),
		2: core.NewBuiltinFunction("int_n", randInt63n, 1, false),
		3: core.NewBuiltinFunction("exp_float", randExpFloat64, 0, false),
		4: core.NewBuiltinFunction("norm_float", randNormFloat64, 0, false),
		5: core.NewBuiltinFunction("perm", randPerm, 1, false),
		6: core.NewBuiltinFunction("seed", randSeed, 1, false),
		7: core.NewBuiltinFunction("read", randRead, 1, false),
		8: core.NewBuiltinFunction("rand", randFunc, 1, false),
	})
}

func randPerm(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.perm", "1", len(args))
	}
	i1, ok := args[0].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("rand.perm", "first", "int(compatible)", args[0].TypeName(a))
	}
	res := rand.Perm(int(i1))
	arr := a.NewArray(len(res), false)
	for _, v := range res {
		arr = append(arr, core.IntValue(int64(v)))
	}
	return a.NewArrayValue(arr, false)
}

func randNormFloat64(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.norm_float", "0", len(args))
	}
	return core.FloatValue(rand.NormFloat64()), nil
}

func randExpFloat64(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.exp_float", "0", len(args))
	}
	return core.FloatValue(rand.ExpFloat64()), nil
}

func randFloat64(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.float", "0", len(args))
	}
	return core.FloatValue(rand.Float64()), nil
}

func randSeed(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.seed", "1", len(args))
	}

	i1, ok := args[0].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("rand.seed", "first", "int(compatible)", args[0].TypeName(a))
	}
	rand.Seed(i1)
	return core.Undefined, nil
}

func randInt63n(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.int_n", "1", len(args))
	}

	i1, ok := args[0].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("rand.int_n", "first", "int(compatible)", args[0].TypeName(a))
	}
	return core.IntValue(rand.Int63n(i1)), nil
}

func randRead(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.read", "1", len(args))
	}
	y1, ok := args[0].AsBytes(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("rand.read", "first", "bytes", args[0].TypeName(a))
	}
	res, err := rand.Read(y1)
	if err != nil {
		return wrapError(a, err)
	}
	return core.IntValue(int64(res)), nil
}

func randFunc(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand", "1", len(args))
	}
	i1, ok := args[0].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("rand.rand", "first", "int(compatible)", args[0].TypeName(a))
	}
	src := rand.NewSource(i1)
	return randRand(a, vm, rand.New(src))
}

func randInt63(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.int", "0", len(args))
	}
	return core.IntValue(rand.Int63()), nil
}

func randRand(a *core.Arena, vm core.VM, r *rand.Rand) (core.Value, error) {
	rInt63, err := a.NewBuiltinClosureValue("int", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.int", "0", len(args))
		}
		return core.IntValue(r.Int63()), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	rFloat64, err := a.NewBuiltinClosureValue("float", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.float", "0", len(args))
		}
		return core.FloatValue(r.Float64()), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	rInt63n, err := a.NewBuiltinClosureValue("int_n", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.int_n", "1", len(args))
		}

		i1, ok := args[0].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("rand.rand.int_n", "first", "int(compatible)", args[0].TypeName(a))
		}
		return core.IntValue(r.Int63n(i1)), nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	rExpFloat64, err := a.NewBuiltinClosureValue("exp_float", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.exp_float", "0", len(args))
		}
		return core.FloatValue(r.ExpFloat64()), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	rNormFloat64, err := a.NewBuiltinClosureValue("norm_float", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.norm_float", "0", len(args))
		}
		return core.FloatValue(r.NormFloat64()), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	rPerm, err := a.NewBuiltinClosureValue("perm", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.perm", "1", len(args))
		}
		i1, ok := args[0].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("rand.rand.perm", "first", "int(compatible)", args[0].TypeName(a))
		}
		res := r.Perm(int(i1))
		arr := a.NewArray(len(res), false)
		for _, v := range res {
			arr = append(arr, core.IntValue(int64(v)))
		}
		return a.NewArrayValue(arr, false)
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	rSeed, err := a.NewBuiltinClosureValue("seed", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.seed", "1", len(args))
		}

		i1, ok := args[0].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("rand.rand.seed", "first", "int(compatible)", args[0].TypeName(a))
		}
		r.Seed(i1)
		return core.Undefined, nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	rRead, err := a.NewBuiltinClosureValue("read", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.read", "1", len(args))
		}
		y1, ok := args[0].AsBytes(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("rand.rand.read", "first", "bytes", args[0].TypeName(a))
		}
		res, err := r.Read(y1)
		if err != nil {
			return wrapError(a, err)
		}
		return core.IntValue(int64(res)), nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	m, err := a.NewRecordValue(map[string]core.Value{
		"int":        rInt63,
		"float":      rFloat64,
		"int_n":      rInt63n,
		"exp_float":  rExpFloat64,
		"norm_float": rNormFloat64,
		"perm":       rPerm,
		"seed":       rSeed,
		"read":       rRead,
	}, true)
	if err != nil {
		return core.Undefined, err
	}

	return m, nil
}
