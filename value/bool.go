package value

import "github.com/jokruger/gs/core"

type Bool struct {
	ObjectImpl
	value bool
}

func (o *Bool) TypeName() string {
	return "bool"
}

func (o *Bool) String() string {
	if o.value {
		return TrueString
	}
	return FalseString
}

func (o *Bool) Copy() core.Object {
	return o
}

func (o *Bool) IsFalsy() bool {
	return !o.value
}

func (o *Bool) Equals(x core.Object) bool {
	return o == x
}

func (o *Bool) GobDecode(b []byte) (err error) {
	o.value = b[0] == 1
	return
}

func (o *Bool) GobEncode() (b []byte, err error) {
	if o.value {
		b = []byte{1}
	} else {
		b = []byte{0}
	}
	return
}

func (o *Bool) ToString() (string, bool) {
	return o.String(), true
}

func (o *Bool) ToInt() (int, bool) {
	if o == TrueValue {
		return 1, true
	}
	return 0, true
}

func (o *Bool) ToInt64() (int64, bool) {
	if o == TrueValue {
		return 1, true
	}
	return 0, true
}

func (o *Bool) ToBool() (bool, bool) {
	return o.value, true
}

func (o *Bool) ToInterface() any {
	return o.value
}
