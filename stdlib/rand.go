package stdlib

import (
	"math/rand"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

var randModule = map[string]core.Object{
	"int": &value.BuiltinFunction{
		Name:  "int",
		Value: randInt63,
	},
	"float": &value.BuiltinFunction{
		Name:  "float",
		Value: FuncARF(rand.Float64),
	},
	"intn": &value.BuiltinFunction{
		Name:  "intn",
		Value: FuncAI64RI64(rand.Int63n),
	},
	"exp_float": &value.BuiltinFunction{
		Name:  "exp_float",
		Value: FuncARF(rand.ExpFloat64),
	},
	"norm_float": &value.BuiltinFunction{
		Name:  "norm_float",
		Value: FuncARF(rand.NormFloat64),
	},
	"perm": &value.BuiltinFunction{
		Name:  "perm",
		Value: FuncAIRIs(rand.Perm),
	},
	"seed": &value.BuiltinFunction{
		Name:  "seed",
		Value: FuncAI64R(rand.Seed),
	},
	"read": &value.BuiltinFunction{
		Name:  "read",
		Value: randRead,
	},
	"rand": &value.BuiltinFunction{
		Name:  "rand",
		Value: randFunc,
	},
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
	return &value.Int{Value: int64(res)}, nil
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

func randRand(r *rand.Rand) *value.ImmutableMap {
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
		return &value.Int{Value: int64(res)}, nil
	}

	return &value.ImmutableMap{
		Value: map[string]core.Object{
			"int": &value.BuiltinFunction{
				Name:  "int",
				Value: rInt63,
			},
			"float": &value.BuiltinFunction{
				Name:  "float",
				Value: FuncARF(r.Float64),
			},
			"intn": &value.BuiltinFunction{
				Name:  "intn",
				Value: FuncAI64RI64(r.Int63n),
			},
			"exp_float": &value.BuiltinFunction{
				Name:  "exp_float",
				Value: FuncARF(r.ExpFloat64),
			},
			"norm_float": &value.BuiltinFunction{
				Name:  "norm_float",
				Value: FuncARF(r.NormFloat64),
			},
			"perm": &value.BuiltinFunction{
				Name:  "perm",
				Value: FuncAIRIs(r.Perm),
			},
			"seed": &value.BuiltinFunction{
				Name:  "seed",
				Value: FuncAI64R(r.Seed),
			},
			"read": &value.BuiltinFunction{
				Name:  "read",
				Value: rRead,
			},
		},
	}
}
