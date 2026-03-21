package value

var (
	// TrueValue is the singleton instance representing the boolean value true.
	TrueValue *Bool = &Bool{value: true}

	// FalseValue is the singleton instance representing the boolean value false.
	FalseValue *Bool = &Bool{value: false}

	// UndefinedValue is the singleton instance representing the undefined value.
	UndefinedValue *Undefined = &Undefined{}
)
