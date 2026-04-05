package core

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/jokruger/gs/token"
)

type ValueKind uint8

const (
	V_UNDEFINED = ValueKind(0)
	V_BOOL      = ValueKind(1)
	V_CHAR      = ValueKind(2)
	V_FLOAT     = ValueKind(3)
	V_INT       = ValueKind(4)

	V_OBJECT    = ValueKind(253)
	V_ITERATOR  = ValueKind(254)
	V_VALUE_PTR = ValueKind(255)
)

func (k ValueKind) String() string {
	switch k {
	case V_UNDEFINED:
		return "undefined"
	case V_BOOL:
		return "bool"
	case V_CHAR:
		return "char"
	case V_FLOAT:
		return "float"
	case V_INT:
		return "int"
	case V_OBJECT:
		return "object"
	case V_ITERATOR:
		return "iterator"
	case V_VALUE_PTR:
		return "value-pointer"
	default:
		return fmt.Sprintf("unknown(%d)", k)
	}
}

type Value struct {
	kind ValueKind
	data uint64
	ptr  any
}

func NewUndefined() Value {
	return Value{kind: V_UNDEFINED}
}

func NewBool(b bool) Value {
	v := Value{kind: V_BOOL}
	if b {
		v.data = 1
	}
	return v
}

func NewChar(c rune) Value {
	return Value{kind: V_CHAR, data: uint64(c)}
}

func NewFloat(f float64) Value {
	return Value{kind: V_FLOAT, data: math.Float64bits(f)}
}

func NewInt(i int64) Value {
	return Value{kind: V_INT, data: uint64(i)}
}

func NewObject(o Object) Value {
	if o == nil {
		return NewUndefined()
	}
	return Value{kind: V_OBJECT, ptr: o}
}

func NewIterator(i Iterator) Value {
	return Value{kind: V_ITERATOR, ptr: i}
}

func NewValuePtr(o *Value) Value {
	return Value{kind: V_VALUE_PTR, ptr: o}
}

func (v *Value) Set(val Value) {
	v.data = val.data
	v.ptr = val.ptr
	v.kind = val.kind
}

func (v *Value) Kind() ValueKind {
	return v.kind
}

func (v *Value) SetKind(k ValueKind) {
	v.kind = k
}

func (v *Value) Object() Object {
	return v.ptr.(Object)
}

func (v *Value) SetObject(o Object) {
	v.ptr = o
}

func (v *Value) Iterator() Iterator {
	return v.ptr.(Iterator)
}

func (v *Value) SetIterator(i Iterator) {
	v.ptr = i
}

func (v *Value) ValuePtr() *Value {
	return v.ptr.(*Value)
}

func (v *Value) SetValuePtr(ptr *Value) {
	v.ptr = ptr
}

func (v *Value) Int() int64 {
	return int64(v.data)
}

func (v *Value) SetInt(i int64) {
	v.data = uint64(i)
}

func (v *Value) Float() float64 {
	return math.Float64frombits(v.data)
}

func (v *Value) SetFloat(f float64) {
	v.data = math.Float64bits(f)
}

func (v *Value) Char() rune {
	return rune(v.data)
}

func (v *Value) SetChar(c rune) {
	v.data = uint64(c)
}

func (v *Value) Bool() bool {
	return v.data != 0
}

func (v *Value) SetBool(b bool) {
	if b {
		v.data = 1
	} else {
		v.data = 0
	}
}

// must be value receiver because core.Value is used in maps (which require serialization)
func (v Value) GobEncode() ([]byte, error) {
	switch v.kind {
	case V_UNDEFINED:
		return []byte{uint8(V_UNDEFINED)}, nil

	case V_BOOL:
		if v.Bool() {
			return []byte{uint8(V_BOOL), 1}, nil
		}
		return []byte{uint8(V_BOOL), 0}, nil

	case V_CHAR:
		r := v.Char()
		b := make([]byte, 5)
		b[0] = uint8(V_CHAR)
		binary.BigEndian.PutUint32(b[1:], uint32(int32(r)))
		return b, nil

	case V_FLOAT:
		b := make([]byte, 9)
		b[0] = uint8(V_FLOAT)
		binary.BigEndian.PutUint64(b[1:], v.data)
		return b, nil

	case V_INT:
		b := make([]byte, 9)
		b[0] = uint8(V_INT)
		binary.BigEndian.PutUint64(b[1:], v.data)
		return b, nil

	case V_OBJECT:
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		obj := v.ptr.(Object)
		if err := enc.Encode(&obj); err != nil {
			return nil, err
		}
		return append([]byte{uint8(V_OBJECT)}, buf.Bytes()...), nil

	default:
		panic(fmt.Sprintf("unexpected use of %s with GobEncode()", v.kind.String()))
	}
}

func (v *Value) GobDecode(data []byte) error {
	if len(data) < 1 {
		return NewDecodeBinarySizeError(v.TypeName(), 2, len(data))
	}
	v.kind = ValueKind(data[0])

	switch v.kind {
	case V_UNDEFINED:
		v.data = 0
		v.ptr = nil
		return nil

	case V_BOOL:
		if len(data) < 2 {
			return NewDecodeBinarySizeError(v.TypeName(), 2, len(data))
		}
		v.data = uint64(data[1])
		v.ptr = nil
		return nil

	case V_CHAR:
		if len(data) < 5 {
			return NewDecodeBinarySizeError(v.TypeName(), 5, len(data))
		}
		v.data = uint64(binary.BigEndian.Uint32(data[1:5]))
		v.ptr = nil
		return nil

	case V_FLOAT:
		if len(data) < 9 {
			return NewDecodeBinarySizeError(v.TypeName(), 9, len(data))
		}
		v.data = binary.BigEndian.Uint64(data[1:9])
		v.ptr = nil
		return nil

	case V_INT:
		if len(data) < 9 {
			return NewDecodeBinarySizeError(v.TypeName(), 9, len(data))
		}
		v.data = binary.BigEndian.Uint64(data[1:9])
		v.ptr = nil
		return nil

	case V_OBJECT:
		var o Object
		buf := bytes.NewBuffer(data[1:])
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(&o); err != nil {
			return err
		}
		v.data = 0
		v.ptr = o
		return nil

	default:
		panic(fmt.Sprintf("unexpected use of %s with GobDecode()", v.kind.String()))
	}
}

func (v *Value) Next() bool {
	switch v.kind {
	case V_UNDEFINED:
		return false
	case V_ITERATOR:
		return v.ptr.(Iterator).Next()
	default:
		panic(fmt.Sprintf("unexpected use of %s with Next()", v.kind.String()))
	}
}

func (v *Value) Key(alloc Allocator) Value {
	switch v.kind {
	case V_UNDEFINED:
		return NewUndefined()
	case V_ITERATOR:
		return v.ptr.(Iterator).Key(alloc)
	default:
		panic(fmt.Sprintf("unexpected use of %s with Key()", v.kind.String()))
	}
}

func (v *Value) Value(alloc Allocator) Value {
	switch v.kind {
	case V_UNDEFINED:
		return NewUndefined()
	case V_ITERATOR:
		return v.ptr.(Iterator).Value(alloc)
	default:
		panic(fmt.Sprintf("unexpected use of %s with Value()", v.kind.String()))
	}
}

func (v *Value) TypeName() string {
	switch v.kind {
	case V_UNDEFINED:
		return "undefined"
	case V_BOOL:
		return "bool"
	case V_CHAR:
		return "char"
	case V_FLOAT:
		return "float"
	case V_INT:
		return "int"
	case V_OBJECT:
		return v.ptr.(Object).TypeName()
	case V_ITERATOR:
		return v.ptr.(Iterator).TypeName()
	default:
		panic(fmt.Sprintf("unexpected use of %s with TypeName()", v.kind.String()))
	}
}

func (v *Value) String() string {
	switch v.kind {
	case V_UNDEFINED:
		return "undefined"
	case V_BOOL:
		if v.Bool() {
			return "true"
		}
		return "false"
	case V_CHAR:
		return fmt.Sprintf("%q", v.Char())
	case V_FLOAT:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case V_INT:
		return strconv.FormatInt(v.Int(), 10)
	case V_OBJECT:
		return v.ptr.(Object).String()
	case V_ITERATOR:
		return v.ptr.(Iterator).String()
	default:
		panic(fmt.Sprintf("unexpected use of %s with String()", v.kind.String()))
	}
}

func (v *Value) Interface() any {
	switch v.kind {
	case V_UNDEFINED:
		return nil
	case V_BOOL:
		return v.Bool()
	case V_CHAR:
		return v.Char()
	case V_FLOAT:
		return v.Float()
	case V_INT:
		return v.Int()
	case V_OBJECT:
		return v.ptr.(Object).Interface()
	default:
		panic(fmt.Sprintf("unexpected use of %s with Interface()", v.kind.String()))
	}
}

func (v *Value) Arity() int {
	switch v.kind {
	case V_OBJECT:
		return v.ptr.(Object).Arity()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with Arity()", v.kind.String()))
	default:
		return 0
	}
}

func (v *Value) IsObject() bool {
	return v.kind == V_OBJECT
}

func (v *Value) IsIterator() bool {
	return v.kind == V_ITERATOR
}

func (v *Value) IsValuePtr() bool {
	return v.kind == V_VALUE_PTR
}

func (v *Value) IsUndefined() bool {
	return v.kind == V_UNDEFINED
}

func (v *Value) IsInt() bool {
	return v.kind == V_INT
}

func (v *Value) IsFloat() bool {
	return v.kind == V_FLOAT
}

func (v *Value) IsBool() bool {
	return v.kind == V_BOOL
}

func (v *Value) IsChar() bool {
	return v.kind == V_CHAR
}

func (v *Value) IsString() bool {
	if v.kind == V_OBJECT {
		return v.ptr.(Object).IsString()
	}
	return false
}

func (v *Value) IsBytes() bool {
	if v.kind == V_OBJECT {
		return v.ptr.(Object).IsBytes()
	}
	return false
}

func (v *Value) IsTime() bool {
	if v.kind == V_OBJECT {
		return v.ptr.(Object).IsTime()
	}
	return false
}

func (v *Value) IsArray() bool {
	if v.kind == V_OBJECT {
		return v.ptr.(Object).IsArray()
	}
	return false
}

func (v *Value) IsError() bool {
	if v.kind == V_OBJECT {
		return v.ptr.(Object).IsError()
	}
	return false
}

func (v *Value) IsMap() bool {
	if v.kind == V_OBJECT {
		return v.ptr.(Object).IsMap()
	}
	return false
}

func (v *Value) IsRecord() bool {
	if v.kind == V_OBJECT {
		return v.ptr.(Object).IsRecord()
	}
	return false
}

func (v *Value) IsCompiledFunction() bool {
	if v.kind == V_OBJECT {
		return v.ptr.(Object).IsCompiledFunction()
	}
	return false
}

func (v *Value) IsBuiltinFunction() bool {
	if v.kind == V_OBJECT {
		return v.ptr.(Object).IsBuiltinFunction()
	}
	return false
}

func (v *Value) IsTrue() bool {
	switch v.kind {
	case V_BOOL:
		return v.data != 0
	case V_CHAR:
		return v.data != 0
	case V_FLOAT:
		return !math.IsNaN(v.Float())
	case V_INT:
		return v.data != 0
	case V_OBJECT:
		return v.ptr.(Object).IsTrue()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with IsTrue()", v.kind.String()))
	default:
		return false
	}
}

func (v *Value) IsFalse() bool {
	switch v.kind {
	case V_BOOL:
		return v.data == 0
	case V_CHAR:
		return v.data == 0
	case V_FLOAT:
		return math.IsNaN(v.Float())
	case V_INT:
		return v.data == 0
	case V_OBJECT:
		return v.ptr.(Object).IsFalse()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with IsFalse()", v.kind.String()))
	default:
		return true
	}
}

func (v *Value) IsIterable() bool {
	switch v.kind {
	case V_UNDEFINED:
		return true
	case V_OBJECT:
		return v.ptr.(Object).IsIterable()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with IsIterable()", v.kind.String()))
	default:
		return false
	}
}

func (v *Value) IsCallable() bool {
	switch v.kind {
	case V_OBJECT:
		return v.ptr.(Object).IsCallable()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with IsCallable()", v.kind.String()))
	default:
		return false
	}
}

func (v *Value) IsVariadic() bool {
	switch v.kind {
	case V_OBJECT:
		return v.ptr.(Object).IsVariadic()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with IsVariadic()", v.kind.String()))
	default:
		return false
	}
}

func (v *Value) IsImmutable() bool {
	switch v.kind {
	case V_OBJECT:
		return v.ptr.(Object).IsImmutable()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with IsImmutable()", v.kind.String()))
	default:
		return true
	}
}

func (v *Value) AsString() (string, bool) {
	switch v.kind {
	case V_BOOL:
		if v.Bool() {
			return "true", true
		}
		return "false", true
	case V_CHAR:
		return string(v.Char()), true
	case V_FLOAT:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64), true
	case V_INT:
		return strconv.FormatInt(v.Int(), 10), true
	case V_OBJECT:
		return v.ptr.(Object).AsString()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with AsString()", v.kind.String()))
	default:
		return "", false
	}
}

func (v *Value) AsInt() (int64, bool) {
	switch v.kind {
	case V_BOOL:
		if v.Bool() {
			return 1, true
		}
		return 0, true
	case V_CHAR:
		return int64(v.Char()), true
	case V_FLOAT:
		return int64(v.Float()), true
	case V_INT:
		return v.Int(), true
	case V_OBJECT:
		return v.ptr.(Object).AsInt()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with AsInt()", v.kind.String()))
	default:
		return 0, false
	}
}

func (v *Value) AsFloat() (float64, bool) {
	switch v.kind {
	case V_FLOAT:
		return v.Float(), true
	case V_INT:
		return float64(v.Int()), true
	case V_OBJECT:
		return v.ptr.(Object).AsFloat()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with AsFloat()", v.kind.String()))
	default:
		return 0, false
	}
}

func (v *Value) AsBool() (bool, bool) {
	switch v.kind {
	case V_UNDEFINED:
		return false, true
	case V_BOOL:
		return v.Bool(), true
	case V_CHAR:
		return v.data != 0, true
	case V_FLOAT:
		return !math.IsNaN(v.Float()), true
	case V_INT:
		return v.data != 0, true
	case V_OBJECT:
		return v.ptr.(Object).AsBool()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with AsBool()", v.kind.String()))
	default:
		return false, false
	}
}

func (v *Value) AsChar() (rune, bool) {
	switch v.kind {
	case V_CHAR:
		return v.Char(), true
	case V_INT:
		return rune(v.Int()), true
	case V_OBJECT:
		return v.ptr.(Object).AsChar()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with AsChar()", v.kind.String()))
	default:
		return 0, false
	}
}

func (v *Value) AsBytes() ([]byte, bool) {
	switch v.kind {
	case V_OBJECT:
		return v.ptr.(Object).AsBytes()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with AsBytes()", v.kind.String()))
	default:
		return nil, false
	}
}

func (v *Value) AsTime() (time.Time, bool) {
	switch v.kind {
	case V_INT:
		return time.Unix(v.Int(), 0), true
	case V_OBJECT:
		return v.ptr.(Object).AsTime()
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with AsTime()", v.kind.String()))
	default:
		return time.Time{}, false
	}
}

func (v *Value) BinaryOp(vm VM, op token.Token, rhs Value) (Value, error) {
	switch v.kind {
	case V_CHAR:
		return v.charBinaryOp(vm, op, rhs)
	case V_FLOAT:
		return v.floatBinaryOp(vm, op, rhs)
	case V_INT:
		return v.intBinaryOp(vm, op, rhs)
	case V_OBJECT:
		return v.ptr.(Object).BinaryOp(vm, op, rhs)
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with BinaryOp()", v.kind.String()))
	default:
		return NewUndefined(), NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}
}

func (v *Value) charBinaryOp(vm VM, op token.Token, rhs Value) (Value, error) {
	alloc := vm.Allocator()

	switch {
	case rhs.IsInt(): // char op int => int
		l := int64(v.Char())
		r := rhs.Int()
		switch op {
		case token.Add:
			return NewInt(l + r), nil
		case token.Sub:
			return NewInt(l - r), nil
		case token.Less:
			return NewBool(l < r), nil
		case token.Greater:
			return NewBool(l > r), nil
		case token.LessEq:
			return NewBool(l <= r), nil
		case token.GreaterEq:
			return NewBool(l >= r), nil
		default:
			return NewUndefined(), NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

	case rhs.IsString(): // char op string => string
		l := string(v.Char())
		r, _ := rhs.ptr.(Object).AsString()
		switch op {
		case token.Add:
			return alloc.NewStringValue(l + r), nil
		default:
			return NewUndefined(), NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

	default:
		// char op any => char
		r, ok := rhs.AsChar()
		if !ok {
			return NewUndefined(), NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

		l := v.Char()
		switch op {
		case token.Add:
			return NewChar(l + r), nil
		case token.Sub:
			return NewChar(l - r), nil
		case token.Less:
			return NewBool(l < r), nil
		case token.Greater:
			return NewBool(l > r), nil
		case token.LessEq:
			return NewBool(l <= r), nil
		case token.GreaterEq:
			return NewBool(l >= r), nil
		default:
			return NewUndefined(), NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}
	}
}

func (v *Value) floatBinaryOp(vm VM, op token.Token, rhs Value) (Value, error) {
	r, ok := rhs.AsFloat()
	if !ok {
		return NewUndefined(), NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	l := v.Float()
	switch op {
	case token.Add:
		return NewFloat(l + r), nil
	case token.Sub:
		return NewFloat(l - r), nil
	case token.Mul:
		return NewFloat(l * r), nil
	case token.Quo:
		return NewFloat(l / r), nil
	case token.Less:
		return NewBool(l < r), nil
	case token.Greater:
		return NewBool(l > r), nil
	case token.LessEq:
		return NewBool(l <= r), nil
	case token.GreaterEq:
		return NewBool(l >= r), nil
	default:
		return NewUndefined(), NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}
}

func (v *Value) intBinaryOp(vm VM, op token.Token, rhs Value) (Value, error) {
	// int op float => float
	if rhs.kind == V_FLOAT {
		l := float64(v.Int())
		r := rhs.Float()
		switch op {
		case token.Add:
			return NewFloat(l + r), nil
		case token.Sub:
			return NewFloat(l - r), nil
		case token.Mul:
			return NewFloat(l * r), nil
		case token.Quo:
			return NewFloat(l / r), nil
		case token.Less:
			return NewBool(l < r), nil
		case token.Greater:
			return NewBool(l > r), nil
		case token.LessEq:
			return NewBool(l <= r), nil
		case token.GreaterEq:
			return NewBool(l >= r), nil
		default:
			return NewUndefined(), NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}
	}

	// int op any => int
	r, ok := rhs.AsInt()
	if !ok {
		return NewUndefined(), NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	l := v.Int()
	switch op {
	case token.Add:
		return NewInt(l + r), nil
	case token.Sub:
		return NewInt(l - r), nil
	case token.Mul:
		return NewInt(l * r), nil
	case token.Quo:
		return NewInt(l / r), nil
	case token.Rem:
		return NewInt(l % r), nil
	case token.And:
		return NewInt(l & r), nil
	case token.Or:
		return NewInt(l | r), nil
	case token.Xor:
		return NewInt(l ^ r), nil
	case token.AndNot:
		return NewInt(l &^ r), nil
	case token.Shl:
		return NewInt(l << uint64(r)), nil
	case token.Shr:
		return NewInt(l >> uint64(r)), nil
	case token.Less:
		return NewBool(l < r), nil
	case token.Greater:
		return NewBool(l > r), nil
	case token.LessEq:
		return NewBool(l <= r), nil
	case token.GreaterEq:
		return NewBool(l >= r), nil
	default:
		return NewUndefined(), NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}
}

func (v *Value) Equals(rhs Value) bool {
	switch v.kind {
	case V_UNDEFINED:
		return rhs.kind == V_UNDEFINED

	case V_BOOL:
		r, ok := rhs.AsBool()
		if !ok {
			return false
		}
		return r == v.Bool()

	case V_CHAR:
		r, ok := rhs.AsChar()
		if !ok {
			return false
		}
		return r == v.Char()

	case V_FLOAT:
		r, ok := rhs.AsFloat()
		if !ok {
			return false
		}
		return r == v.Float()

	case V_INT:
		r, ok := rhs.AsInt()
		if !ok {
			return false
		}
		return r == v.Int()

	case V_OBJECT:
		return v.ptr.(Object).Equals(rhs)

	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with Equals()", v.kind.String()))

	default:
		return false
	}
}

func (v *Value) Copy(alloc Allocator) Value {
	switch v.kind {
	case V_OBJECT:
		return v.ptr.(Object).Copy(alloc)
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with Copy()", v.kind.String()))
	default:
		return *v
	}
}

func (v *Value) Method(vm VM, name string, args ...Value) (Value, error) {
	switch v.kind {
	case V_OBJECT:
		return v.ptr.(Object).Method(vm, name, args...)
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with Method()", v.kind.String()))
	default:
		return NewUndefined(), NewInvalidMethodError(name, v.TypeName())
	}
}

func (v *Value) Access(vm VM, index Value, mode Opcode) (Value, error) {
	switch v.kind {
	case V_UNDEFINED:
		return NewUndefined(), nil
	case V_BOOL:
		return v.boolAccess(vm, index, mode)
	case V_CHAR:
		return v.charAccess(vm, index, mode)
	case V_FLOAT:
		return v.floatAccess(vm, index, mode)
	case V_INT:
		return v.intAccess(vm, index, mode)
	case V_OBJECT:
		return v.ptr.(Object).Access(vm, index, mode)
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with Access()", v.kind.String()))
	default:
		return NewUndefined(), NewNotAccessibleError(v.TypeName())
	}
}

func (v *Value) boolAccess(vm VM, index Value, op Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return NewUndefined(), NewInvalidSelectorError(v.TypeName(), k)
	}

	alloc := vm.Allocator()
	switch k {
	case "bool":
		return NewBool(v.Bool()), nil

	case "int":
		if v.Bool() {
			return NewInt(1), nil
		}
		return NewInt(0), nil

	case "string":
		if v.Bool() {
			return alloc.NewStringValue("true"), nil
		}
		return alloc.NewStringValue("false"), nil

	default:
		return NewUndefined(), NewInvalidSelectorError(v.TypeName(), k)
	}
}

func (v *Value) charAccess(vm VM, index Value, op Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return NewUndefined(), NewInvalidSelectorError(v.TypeName(), k)
	}

	alloc := vm.Allocator()
	switch k {
	case "char":
		return NewChar(v.Char()), nil

	case "bool":
		return NewBool(v.Char() != 0), nil

	case "int":
		return NewInt(int64(v.Char())), nil

	case "string":
		return alloc.NewStringValue(string(v.Char())), nil

	default:
		return NewUndefined(), NewInvalidSelectorError(v.TypeName(), k)
	}
}

func (v *Value) floatAccess(vm VM, index Value, op Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return NewUndefined(), NewInvalidSelectorError(v.TypeName(), k)
	}

	alloc := vm.Allocator()
	switch k {
	case "float":
		return NewFloat(v.Float()), nil

	case "int":
		return NewInt(int64(v.Float())), nil

	case "string":
		return alloc.NewStringValue(strconv.FormatFloat(v.Float(), 'f', -1, 64)), nil

	default:
		return NewUndefined(), NewInvalidSelectorError(v.TypeName(), k)
	}
}

func (v *Value) intAccess(vm VM, index Value, op Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return NewUndefined(), NewInvalidSelectorError(v.TypeName(), k)
	}

	alloc := vm.Allocator()
	switch k {
	case "int":
		return NewInt(v.Int()), nil

	case "float":
		return NewFloat(float64(v.Int())), nil

	case "bool":
		return NewBool(v.Int() != 0), nil

	case "char":
		return NewChar(rune(v.Int())), nil

	case "string":
		return alloc.NewStringValue(strconv.FormatInt(v.Int(), 10)), nil

	case "time":
		return alloc.NewTimeValue(time.Unix(v.Int(), 0)), nil

	default:
		return NewUndefined(), NewInvalidSelectorError(v.TypeName(), k)
	}
}

func (v *Value) Assign(idx, val Value) error {
	switch v.kind {
	case V_OBJECT:
		return v.ptr.(Object).Assign(idx, val)
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with Assign()", v.kind.String()))
	default:
		return NewNotAssignableError(v.TypeName())
	}
}

func (v *Value) Iterate(alloc Allocator) Iterator {
	switch v.kind {
	case V_OBJECT:
		return v.ptr.(Object).Iterate(alloc)
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with Iterate()", v.kind.String()))
	default:
		return nil
	}
}

func (v *Value) Call(vm VM, args ...Value) (Value, error) {
	switch v.kind {
	case V_OBJECT:
		return v.ptr.(Object).Call(vm, args...)
	case V_ITERATOR, V_VALUE_PTR:
		panic(fmt.Sprintf("unexpected use of %s with Call()", v.kind.String()))
	default:
		return NewUndefined(), NewNotCallableError(v.TypeName())
	}
}
