package stdlib

import (
	"math/rand"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

var randModule = map[string]core.Object{
	/*
		"int": &value.BuiltinFunction{
			Name:  "int",
			Value: randInt63,
		},
		"float": &value.BuiltinFunction{
			Name:  "float",
			Value: randFloat64,
		},
		"intn": &value.BuiltinFunction{
			Name:  "intn",
			Value: randInt63n,
		},
		"exp_float": &value.BuiltinFunction{
			Name:  "exp_float",
			Value: randExpFloat64,
		},
		"norm_float": &value.BuiltinFunction{
			Name:  "norm_float",
			Value: randNormFloat64,
		},
		"perm": &value.BuiltinFunction{
			Name:  "perm",
			Value: randPerm,
		},
		"seed": &value.BuiltinFunction{
			Name:  "seed",
			Value: randSeed,
		},
		"read": &value.BuiltinFunction{
			Name:  "read",
			Value: randRead,
		},
		"rand": &value.BuiltinFunction{
			Name:  "rand",
			Value: randFunc,
		},
	*/
}

/*
func randPerm(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	res := rand.Perm(int(i1))
	arr := value.NewArray(nil, false)
	for _, v := range res {
		arr.Value = append(arr.Value, &value.Int{Value: int64(v)})
	}
	return arr, nil
}

func randNormFloat64(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Float{Value: rand.NormFloat64()}, nil
}

func randExpFloat64(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Float{Value: rand.ExpFloat64()}, nil
}

func randFloat64(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Float{Value: rand.Float64()}, nil
}

func randSeed(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	rand.Seed(i1)
	return value.UndefinedValue, nil
}

func randInt63n(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Int{Value: rand.Int63n(i1)}, nil
}

func randRead(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	y1, ok := args[0].(*value.Bytes)
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "bytes",
			Found:    args[0].TypeName(),
		}
	}
	res, err := rand.Read(y1.Value)
	if err != nil {
		ret = wrapError(err)
		return
	}
	return value.NewInt(int64(res)), nil
}

func randFunc(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	src := rand.NewSource(i1)
	return randRand(rand.New(src)), nil
}

func randInt63(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Int{Value: rand.Int63()}, nil
}
*/

func randRand(r *rand.Rand) *value.Map {
	/*
		rInt63 := func(args ...core.Object) (ret core.Object, err error) {
			if len(args) != 0 {
				return nil, gse.ErrWrongNumArguments
			}
			return &value.Int{Value: r.Int63()}, nil
		}

		rRead := func(args ...core.Object) (ret core.Object, err error) {
			if len(args) != 1 {
				return nil, gse.ErrWrongNumArguments
			}
			y1, ok := args[0].(*value.Bytes)
			if !ok {
				return nil, gse.ErrInvalidArgumentType{
					Name:     "first",
					Expected: "bytes",
					Found:    args[0].TypeName(),
				}
			}
			res, err := r.Read(y1.Value)
			if err != nil {
				ret = wrapError(err)
				return
			}
			return value.NewInt(int64(res)), nil
		}

		rInt63n := func(args ...core.Object) (ret core.Object, err error) {
			if len(args) != 1 {
				return nil, gse.ErrWrongNumArguments
			}

			i1, ok := args[0].AsInt()
			if !ok {
				return nil, gse.ErrInvalidArgumentType{
					Name:     "first",
					Expected: "int(compatible)",
					Found:    args[0].TypeName(),
				}
			}
			return &value.Int{Value: r.Int63n(i1)}, nil
		}

		rSeed := func(args ...core.Object) (ret core.Object, err error) {
			if len(args) != 1 {
				return nil, gse.ErrWrongNumArguments
			}

			i1, ok := args[0].AsInt()
			if !ok {
				return nil, gse.ErrInvalidArgumentType{
					Name:     "first",
					Expected: "int(compatible)",
					Found:    args[0].TypeName(),
				}
			}
			r.Seed(i1)
			return value.UndefinedValue, nil
		}

		rFloat64 := func(args ...core.Object) (ret core.Object, err error) {
			if len(args) != 0 {
				return nil, gse.ErrWrongNumArguments
			}
			return &value.Float{Value: r.Float64()}, nil
		}

		rExpFloat64 := func(args ...core.Object) (ret core.Object, err error) {
			if len(args) != 0 {
				return nil, gse.ErrWrongNumArguments
			}
			return &value.Float{Value: r.ExpFloat64()}, nil
		}

		rNormFloat64 := func(args ...core.Object) (ret core.Object, err error) {
			if len(args) != 0 {
				return nil, gse.ErrWrongNumArguments
			}
			return &value.Float{Value: r.NormFloat64()}, nil
		}

		rPerm := func(args ...core.Object) (ret core.Object, err error) {
			if len(args) != 1 {
				return nil, gse.ErrWrongNumArguments
			}
			i1, ok := args[0].AsInt()
			if !ok {
				return nil, gse.ErrInvalidArgumentType{
					Name:     "first",
					Expected: "int(compatible)",
					Found:    args[0].TypeName(),
				}
			}
			res := r.Perm(int(i1))
			arr := value.NewArray(nil, false)
			for _, v := range res {
				arr.Value = append(arr.Value, &value.Int{Value: int64(v)})
			}
			return arr, nil
		}
	*/

	return value.NewMap(map[string]core.Object{
		/*
			"int": &value.BuiltinFunction{
				Name:  "int",
				Value: rInt63,
			},
			"float": &value.BuiltinFunction{
				Name:  "float",
				Value: rFloat64,
			},
			"intn": &value.BuiltinFunction{
				Name:  "intn",
				Value: rInt63n,
			},
			"exp_float": &value.BuiltinFunction{
				Name:  "exp_float",
				Value: rExpFloat64,
			},
			"norm_float": &value.BuiltinFunction{
				Name:  "norm_float",
				Value: rNormFloat64,
			},
			"perm": &value.BuiltinFunction{
				Name:  "perm",
				Value: rPerm,
			},
			"seed": &value.BuiltinFunction{
				Name:  "seed",
				Value: rSeed,
			},
			"read": &value.BuiltinFunction{
				Name:  "read",
				Value: rRead,
			},
		*/
	}, true)
}
