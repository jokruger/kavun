package stdlib

import (
	"math/rand"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
)

var randModule = map[string]core.Value{
	"int":        core.NewBuiltinFunctionValue("int", randInt63, 0, false),
	"float":      core.NewBuiltinFunctionValue("float", randFloat64, 0, false),
	"int_n":      core.NewBuiltinFunctionValue("int_n", randInt63n, 1, false),
	"exp_float":  core.NewBuiltinFunctionValue("exp_float", randExpFloat64, 0, false),
	"norm_float": core.NewBuiltinFunctionValue("norm_float", randNormFloat64, 0, false),
	"perm":       core.NewBuiltinFunctionValue("perm", randPerm, 1, false),
	"seed":       core.NewBuiltinFunctionValue("seed", randSeed, 1, false),
	"read":       core.NewBuiltinFunctionValue("read", randRead, 1, false),
	"rand":       core.NewBuiltinFunctionValue("rand", randFunc, 1, false),
}

func randPerm(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.perm", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("rand.perm", "first", "int(compatible)", args[0].TypeName())
	}
	res := rand.Perm(int(i1))
	arr := make([]core.Value, 0, len(res))
	alloc := vm.Allocator()
	for _, v := range res {
		arr = append(arr, core.IntValue(int64(v)))
	}
	return alloc.NewArrayValue(arr, false)
}

func randNormFloat64(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.norm_float", "0", len(args))
	}
	return core.FloatValue(rand.NormFloat64()), nil
}

func randExpFloat64(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.exp_float", "0", len(args))
	}
	return core.FloatValue(rand.ExpFloat64()), nil
}

func randFloat64(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.float", "0", len(args))
	}
	return core.FloatValue(rand.Float64()), nil
}

func randSeed(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.seed", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("rand.seed", "first", "int(compatible)", args[0].TypeName())
	}
	rand.Seed(i1)
	return core.Undefined, nil
}

func randInt63n(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.int_n", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("rand.int_n", "first", "int(compatible)", args[0].TypeName())
	}
	return core.IntValue(rand.Int63n(i1)), nil
}

func randRead(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.read", "1", len(args))
	}
	y1, ok := args[0].AsBytes()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("rand.read", "first", "bytes", args[0].TypeName())
	}
	res, err := rand.Read(y1)
	if err != nil {
		return wrapError(vm, err)
	}
	return core.IntValue(int64(res)), nil
}

func randFunc(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("rand.rand", "first", "int(compatible)", args[0].TypeName())
	}
	src := rand.NewSource(i1)
	return randRand(vm, rand.New(src))
}

func randInt63(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rand.int", "0", len(args))
	}
	return core.IntValue(rand.Int63()), nil
}

func randRand(vm core.VM, r *rand.Rand) (core.Value, error) {
	alloc := vm.Allocator()

	rInt63, err := alloc.NewBuiltinFunctionValue("int", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.int", "0", len(args))
		}
		return core.IntValue(r.Int63()), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	rFloat64, err := alloc.NewBuiltinFunctionValue("float", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.float", "0", len(args))
		}
		return core.FloatValue(r.Float64()), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	rInt63n, err := alloc.NewBuiltinFunctionValue("int_n", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.int_n", "1", len(args))
		}

		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("rand.rand.int_n", "first", "int(compatible)", args[0].TypeName())
		}
		return core.IntValue(r.Int63n(i1)), nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	rExpFloat64, err := alloc.NewBuiltinFunctionValue("exp_float", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.exp_float", "0", len(args))
		}
		return core.FloatValue(r.ExpFloat64()), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	rNormFloat64, err := alloc.NewBuiltinFunctionValue("norm_float", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 0 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.norm_float", "0", len(args))
		}
		return core.FloatValue(r.NormFloat64()), nil
	}, 0, false)
	if err != nil {
		return core.Undefined, err
	}

	rPerm, err := alloc.NewBuiltinFunctionValue("perm", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.perm", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("rand.rand.perm", "first", "int(compatible)", args[0].TypeName())
		}
		res := r.Perm(int(i1))
		arr := make([]core.Value, 0, len(res))
		alloc := vm.Allocator()
		for _, v := range res {
			arr = append(arr, core.IntValue(int64(v)))
		}
		return alloc.NewArrayValue(arr, false)
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	rSeed, err := alloc.NewBuiltinFunctionValue("seed", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.seed", "1", len(args))
		}

		i1, ok := args[0].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("rand.rand.seed", "first", "int(compatible)", args[0].TypeName())
		}
		r.Seed(i1)
		return core.Undefined, nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	rRead, err := alloc.NewBuiltinFunctionValue("read", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("rand.rand.read", "1", len(args))
		}
		y1, ok := args[0].AsBytes()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("rand.rand.read", "first", "bytes", args[0].TypeName())
		}
		res, err := r.Read(y1)
		if err != nil {
			return wrapError(vm, err)
		}
		return core.IntValue(int64(res)), nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	m, err := vm.Allocator().NewRecordValue(map[string]core.Value{
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
