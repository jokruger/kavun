package stdlib

import (
	"regexp"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

func makeTextRegexp(re *regexp.Regexp) *value.Map {
	return value.NewMap(map[string]core.Object{
		/*
			// match(text) => bool
			"match": &value.BuiltinFunction{
				Value: func(args ...core.Object) (
					ret core.Object,
					err error,
				) {
					if len(args) != 1 {
						err = gse.ErrWrongNumArguments
						return
					}

					s1, ok := args[0].AsString()
					if !ok {
						err = gse.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "string(compatible)",
							Found:    args[0].TypeName(),
						}
						return
					}

					if re.MatchString(s1) {
						ret = value.TrueValue
					} else {
						ret = value.FalseValue
					}

					return
				},
			},

			// find(text) 			=> array(array({text:,begin:,end:}))/undefined
			// find(text, maxCount) => array(array({text:,begin:,end:}))/undefined
			"find": &value.BuiltinFunction{
				Value: func(args ...core.Object) (
					ret core.Object,
					err error,
				) {
					numArgs := len(args)
					if numArgs != 1 && numArgs != 2 {
						err = gse.ErrWrongNumArguments
						return
					}

					s1, ok := args[0].AsString()
					if !ok {
						err = gse.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "string(compatible)",
							Found:    args[0].TypeName(),
						}
						return
					}

					if numArgs == 1 {
						m := re.FindStringSubmatchIndex(s1)
						if m == nil {
							ret = value.UndefinedValue
							return
						}

						arr := &value.Array{}
						for i := 0; i < len(m); i += 2 {
							arr.Value = append(arr.Value,
								&value.ImmutableMap{
									Value: map[string]core.Object{
										"text": &value.String{
											Value: s1[m[i]:m[i+1]],
										},
										"begin": &value.Int{
											Value: int64(m[i]),
										},
										"end": &value.Int{
											Value: int64(m[i+1]),
										},
									}})
						}

						ret = &value.Array{Value: []core.Object{arr}}

						return
					}

					i2, ok := args[1].AsInt()
					if !ok {
						err = gse.ErrInvalidArgumentType{
							Name:     "second",
							Expected: "int(compatible)",
							Found:    args[1].TypeName(),
						}
						return
					}
					m := re.FindAllStringSubmatchIndex(s1, int(i2))
					if m == nil {
						ret = value.UndefinedValue
						return
					}

					arr := &value.Array{}
					for _, m := range m {
						subMatch := &value.Array{}
						for i := 0; i < len(m); i += 2 {
							subMatch.Value = append(subMatch.Value,
								&value.ImmutableMap{
									Value: map[string]core.Object{
										"text": &value.String{
											Value: s1[m[i]:m[i+1]],
										},
										"begin": &value.Int{
											Value: int64(m[i]),
										},
										"end": &value.Int{
											Value: int64(m[i+1]),
										},
									}})
						}

						arr.Value = append(arr.Value, subMatch)
					}

					ret = arr

					return
				},
			},

			// replace(src, repl) => string
			"replace": &value.BuiltinFunction{
				Value: func(args ...core.Object) (
					ret core.Object,
					err error,
				) {
					if len(args) != 2 {
						err = gse.ErrWrongNumArguments
						return
					}

					s1, ok := args[0].AsString()
					if !ok {
						err = gse.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "string(compatible)",
							Found:    args[0].TypeName(),
						}
						return
					}

					s2, ok := args[1].AsString()
					if !ok {
						err = gse.ErrInvalidArgumentType{
							Name:     "second",
							Expected: "string(compatible)",
							Found:    args[1].TypeName(),
						}
						return
					}

					s, ok := doTextRegexpReplace(re, s1, s2)
					if !ok {
						return nil, gse.ErrStringLimit
					}

					ret = value.NewString(s)

					return
				},
			},

			// split(text) 			 => array(string)
			// split(text, maxCount) => array(string)
			"split": &value.BuiltinFunction{
				Value: func(args ...core.Object) (
					ret core.Object,
					err error,
				) {
					numArgs := len(args)
					if numArgs != 1 && numArgs != 2 {
						err = gse.ErrWrongNumArguments
						return
					}

					s1, ok := args[0].AsString()
					if !ok {
						err = gse.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "string(compatible)",
							Found:    args[0].TypeName(),
						}
						return
					}

					var i2 = -1
					if numArgs > 1 {
						var i2t int64
						i2t, ok = args[1].AsInt()
						i2 = int(i2t)
						if !ok {
							err = gse.ErrInvalidArgumentType{
								Name:     "second",
								Expected: "int(compatible)",
								Found:    args[1].TypeName(),
							}
							return
						}
					}

					arr := &value.Array{}
					for _, s := range re.Split(s1, i2) {
						arr.Value = append(arr.Value,
							value.NewString(s))
					}

					ret = arr

					return
				},
			},
		*/
	}, true)
}

// Size-limit checking implementation of regexp.ReplaceAllString.
func doTextRegexpReplace(re *regexp.Regexp, src, repl string) (string, bool) {
	idx := 0
	out := ""
	for _, m := range re.FindAllStringSubmatchIndex(src, -1) {
		var exp []byte
		exp = re.ExpandString(exp, repl, src, m)
		if len(out)+m[0]-idx+len(exp) > core.MaxStringLen {
			return "", false
		}
		out += src[idx:m[0]] + string(exp)
		idx = m[1]
	}
	if idx < len(src) {
		if len(out)+len(src)-idx > core.MaxStringLen {
			return "", false
		}
		out += src[idx:]
	}
	return out, true
}
