package alloc

type ArenaOption func(*ArenaOptions)

type ArenaOptions struct {
	decimals int
	times    int

	bytesNum int
	bytesCap int

	runesNum int
	runesCap int

	arraysNum int
	arraysCap int

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

func WithDecimals(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.decimals = n
		}
	}
}

func WithTimes(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.times = n
		}
	}
}

func WithByteSlices(num int, cap int) ArenaOption {
	return func(o *ArenaOptions) {
		if num >= 0 {
			o.bytesNum = num
		}
		if cap >= 0 {
			o.bytesCap = cap
		}
	}
}

func WithRuneSlices(num int, cap int) ArenaOption {
	return func(o *ArenaOptions) {
		if num >= 0 {
			o.runesNum = num
		}
		if cap >= 0 {
			o.runesCap = cap
		}
	}
}

func WithArraySlices(num int, cap int) ArenaOption {
	return func(o *ArenaOptions) {
		if num >= 0 {
			o.arraysNum = num
		}
		if cap >= 0 {
			o.arraysCap = cap
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

func WithCompiledFunctions(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.compiledFunctions = n
		}
	}
}

func WithErrorValues(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.errorValues = n
		}
	}
}

func WithStringValues(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.stringValues = n
		}
	}
}

func WithRunesValues(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.runesValues = n
		}
	}
}

func WithBytesValues(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.bytesValues = n
		}
	}
}

func WithArrayValues(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.arrayValues = n
		}
	}
}

func WithMapValues(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.mapValues = n
		}
	}
}

func WithIntRangeValues(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.intRangeValues = n
		}
	}
}

func WithRunesIterators(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.runesIterators = n
		}
	}
}

func WithBytesIterators(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.bytesIterators = n
		}
	}
}

func WithArrayIterators(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.arrayIterators = n
		}
	}
}

func WithMapIterators(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.mapIterators = n
		}
	}
}

func WithIntRangeIterators(n int) ArenaOption {
	return func(o *ArenaOptions) {
		if n >= 0 {
			o.intRangeIterators = n
		}
	}
}
