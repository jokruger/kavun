package alloc

type ArenaOption func(*ArenaOptions)

type ArenaOptions struct {
	decimals int
	times    int

	builtinFunctions  int
	compiledFunctions int

	errorValues    int
	stringValues   int
	runesValues    int
	bytesValues    int
	arrayValues    int
	mapValues      int
	intRangeValues int

	runesIterators    int
	bytesIterators    int
	arrayIterators    int
	mapIterators      int
	intRangeIterators int
}

func WithDecimals(decimals int) ArenaOption {
	return func(o *ArenaOptions) {
		if decimals >= 0 {
			o.decimals = decimals
		}
	}
}

func WithTimes(times int) ArenaOption {
	return func(o *ArenaOptions) {
		if times >= 0 {
			o.times = times
		}
	}
}

func WithBuiltinFunctions(builtinFunctions int) ArenaOption {
	return func(o *ArenaOptions) {
		if builtinFunctions >= 0 {
			o.builtinFunctions = builtinFunctions
		}
	}
}

func WithCompiledFunctions(compiledFunctions int) ArenaOption {
	return func(o *ArenaOptions) {
		if compiledFunctions >= 0 {
			o.compiledFunctions = compiledFunctions
		}
	}
}

func WithErrorValues(errorValues int) ArenaOption {
	return func(o *ArenaOptions) {
		if errorValues >= 0 {
			o.errorValues = errorValues
		}
	}
}

func WithStringValues(stringValues int) ArenaOption {
	return func(o *ArenaOptions) {
		if stringValues >= 0 {
			o.stringValues = stringValues
		}
	}
}

func WithRunesValues(runesValues int) ArenaOption {
	return func(o *ArenaOptions) {
		if runesValues >= 0 {
			o.runesValues = runesValues
		}
	}
}

func WithBytesValues(bytesValues int) ArenaOption {
	return func(o *ArenaOptions) {
		if bytesValues >= 0 {
			o.bytesValues = bytesValues
		}
	}
}

func WithArrayValues(arrayValues int) ArenaOption {
	return func(o *ArenaOptions) {
		if arrayValues >= 0 {
			o.arrayValues = arrayValues
		}
	}
}

func WithMapValues(mapValues int) ArenaOption {
	return func(o *ArenaOptions) {
		if mapValues >= 0 {
			o.mapValues = mapValues
		}
	}
}

func WithIntRangeValues(intRangeValues int) ArenaOption {
	return func(o *ArenaOptions) {
		if intRangeValues >= 0 {
			o.intRangeValues = intRangeValues
		}
	}
}

func WithRunesIterators(runesIterators int) ArenaOption {
	return func(o *ArenaOptions) {
		if runesIterators >= 0 {
			o.runesIterators = runesIterators
		}
	}
}

func WithBytesIterators(bytesIterators int) ArenaOption {
	return func(o *ArenaOptions) {
		if bytesIterators >= 0 {
			o.bytesIterators = bytesIterators
		}
	}
}

func WithArrayIterators(arrayIterators int) ArenaOption {
	return func(o *ArenaOptions) {
		if arrayIterators >= 0 {
			o.arrayIterators = arrayIterators
		}
	}
}

func WithMapIterators(mapIterators int) ArenaOption {
	return func(o *ArenaOptions) {
		if mapIterators >= 0 {
			o.mapIterators = mapIterators
		}
	}
}

func WithIntRangeIterators(intRangeIterators int) ArenaOption {
	return func(o *ArenaOptions) {
		if intRangeIterators >= 0 {
			o.intRangeIterators = intRangeIterators
		}
	}
}
