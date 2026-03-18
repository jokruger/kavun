package stdlib

import (
	"math/rand"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

var randModule = map[string]core.Object{
	"int":        value.NewBuiltinFunction("int", randInt63, 0, false),
	"float":      value.NewBuiltinFunction("float", randFloat64, 0, false),
	"intn":       value.NewBuiltinFunction("intn", randInt63n, 1, false),
	"exp_float":  value.NewBuiltinFunction("exp_float", randExpFloat64, 0, false),
	"norm_float": value.NewBuiltinFunction("norm_float", randNormFloat64, 0, false),
	"perm":       value.NewBuiltinFunction("perm", randPerm, 1, false),
	"seed":       value.NewBuiltinFunction("seed", randSeed, 1, false),
	"read":       value.NewBuiltinFunction("read", randRead, 1, false),
	"rand":       value.NewBuiltinFunction("rand", randFunc, 1, false),
}

func randPerm(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("rand.perm", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("rand.perm", "first", "int(compatible)", args[0])
	}
	res := rand.Perm(int(i1))
	arr := make([]core.Object, 0, len(res))
	for _, v := range res {
		arr = append(arr, value.NewInt(int64(v)))
	}
	return value.NewArray(arr, false), nil
}

func randNormFloat64(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.WrongNumArguments("rand.norm_float", "0", len(args))
	}
	return value.NewFloat(rand.NormFloat64()), nil
}

func randExpFloat64(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.WrongNumArguments("rand.exp_float", "0", len(args))
	}
	return value.NewFloat(rand.ExpFloat64()), nil
}

func randFloat64(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.WrongNumArguments("rand.float", "0", len(args))
	}
	return value.NewFloat(rand.Float64()), nil
}

func randSeed(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("rand.seed", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("rand.seed", "first", "int(compatible)", args[0])
	}
	rand.Seed(i1)
	return value.UndefinedValue, nil
}

func randInt63n(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("rand.intn", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("rand.intn", "first", "int(compatible)", args[0])
	}
	return value.NewInt(rand.Int63n(i1)), nil
}

func randRead(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("rand.read", "1", len(args))
	}
	y1, ok := args[0].(*value.Bytes)
	if !ok {
		return nil, core.InvalidArgumentType("rand.read", "first", "bytes", args[0])
	}
	res, err := rand.Read(y1.Value())
	if err != nil {
		ret = wrapError(err)
		return
	}
	return value.NewInt(int64(res)), nil
}

func randFunc(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("rand.rand", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("rand.rand", "first", "int(compatible)", args[0])
	}
	src := rand.NewSource(i1)
	return randRand(rand.New(src)), nil
}

func randInt63(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.WrongNumArguments("rand.int", "0", len(args))
	}
	return value.NewInt(rand.Int63()), nil
}

func randRand(r *rand.Rand) *value.Map {
	rInt63 := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.WrongNumArguments("rand.rand.int", "0", len(args))
		}
		return value.NewInt(r.Int63()), nil
	}

	rRead := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, core.WrongNumArguments("rand.rand.read", "1", len(args))
		}
		y1, ok := args[0].(*value.Bytes)
		if !ok {
			return nil, core.InvalidArgumentType("rand.rand.read", "first", "bytes", args[0])
		}
		res, err := r.Read(y1.Value())
		if err != nil {
			ret = wrapError(err)
			return
		}
		return value.NewInt(int64(res)), nil
	}

	rInt63n := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, core.WrongNumArguments("rand.rand.intn", "1", len(args))
		}

		i1, ok := args[0].AsInt()
		if !ok {
			return nil, core.InvalidArgumentType("rand.rand.intn", "first", "int(compatible)", args[0])
		}
		return value.NewInt(r.Int63n(i1)), nil
	}

	rSeed := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, core.WrongNumArguments("rand.rand.seed", "1", len(args))
		}

		i1, ok := args[0].AsInt()
		if !ok {
			return nil, core.InvalidArgumentType("rand.rand.seed", "first", "int(compatible)", args[0])
		}
		r.Seed(i1)
		return value.UndefinedValue, nil
	}

	rFloat64 := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.WrongNumArguments("rand.rand.float", "0", len(args))
		}
		return value.NewFloat(r.Float64()), nil
	}

	rExpFloat64 := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.WrongNumArguments("rand.rand.exp_float", "0", len(args))
		}
		return value.NewFloat(r.ExpFloat64()), nil
	}

	rNormFloat64 := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 0 {
			return nil, core.WrongNumArguments("rand.rand.norm_float", "0", len(args))
		}
		return value.NewFloat(r.NormFloat64()), nil
	}

	rPerm := func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, core.WrongNumArguments("rand.rand.perm", "1", len(args))
		}
		i1, ok := args[0].AsInt()
		if !ok {
			return nil, core.InvalidArgumentType("rand.rand.perm", "first", "int(compatible)", args[0])
		}
		res := r.Perm(int(i1))
		arr := make([]core.Object, 0, len(res))
		for _, v := range res {
			arr = append(arr, value.NewInt(int64(v)))
		}
		return value.NewArray(arr, false), nil
	}

	return value.NewMap(map[string]core.Object{
		"int":        value.NewBuiltinFunction("int", rInt63, 0, false),
		"float":      value.NewBuiltinFunction("float", rFloat64, 0, false),
		"intn":       value.NewBuiltinFunction("intn", rInt63n, 1, false),
		"exp_float":  value.NewBuiltinFunction("exp_float", rExpFloat64, 0, false),
		"norm_float": value.NewBuiltinFunction("norm_float", rNormFloat64, 0, false),
		"perm":       value.NewBuiltinFunction("perm", rPerm, 1, false),
		"seed":       value.NewBuiltinFunction("seed", rSeed, 1, false),
		"read":       value.NewBuiltinFunction("read", rRead, 1, false),
	}, true)
}
