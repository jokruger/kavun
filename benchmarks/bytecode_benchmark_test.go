package benchmarks

import (
	"encoding/binary"
	"testing"
	"unsafe"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
)

const (
	OpIndex = byte(0)
	OpInt1  = byte(1)
	OpInt2  = byte(2)
	OpInt4  = byte(3)
	OpInt8  = byte(4)
)

func runBytecode(bytecode []byte, static []core.Value) core.Value {
	r := core.IntValue(0)
	ip := -1
	n := len(bytecode)
	var err error
	var v core.Value
	for {
		ip++
		if ip >= n {
			break
		}
		switch bytecode[ip] {
		case OpIndex:
			ip++
			v = static[bytecode[ip]]
		case OpInt1:
			ip++
			v = core.Value{Type: value.Int, Immutable: true, Data: uint64(bytecode[ip])}
		case OpInt2:
			ip++
			i := binary.LittleEndian.Uint16(bytecode[ip:])
			//i := uint16(bytecode[ip]) | uint16(bytecode[ip+1])<<8
			v = core.Value{Type: value.Int, Immutable: true, Data: uint64(i)}
			ip += 1
		case OpInt4:
			ip++
			//i := binary.LittleEndian.Uint32(bytecode[ip:])
			i := uint32(bytecode[ip]) | uint32(bytecode[ip+1])<<8 | uint32(bytecode[ip+2])<<16 | uint32(bytecode[ip+3])<<24
			v = core.Value{Type: value.Int, Immutable: true, Data: uint64(i)}
			ip += 3
		case OpInt8:
			ip++
			i := binary.LittleEndian.Uint64(bytecode[ip:])
			//i := uint64(bytecode[ip]) | uint64(bytecode[ip+1])<<8 | uint64(bytecode[ip+2])<<16 | uint64(bytecode[ip+3])<<24 | uint64(bytecode[ip+4])<<32 | uint64(bytecode[ip+5])<<40 | uint64(bytecode[ip+6])<<48 | uint64(bytecode[ip+7])<<56
			v = core.Value{Type: value.Int, Immutable: true, Data: i}
			ip += 7
		default:
			panic("unknown opcode")
		}
		r, err = r.BinaryOp(token.Add, v)
		if err != nil {
			panic(err)
		}
	}
	return r
}

// runBytecodeUnsafe assumes the bytecode is well-formed (produced by a trusted
// compiler) and uses unsafe to elide all bounds checks. Multi-byte payloads
// are decoded via binary.LittleEndian so the result is endian-portable: on
// little-endian targets the SSA backend lowers each call to a single MOVx;
// on big-endian targets it becomes MOVx + BSWAPx. The (*[N]byte)(...)[:]
// dance gives the helper a slice with a compile-time-constant length so the
// inner `_ = b[N-1]` bounds check is also elided.
func runBytecodeUnsafe(bytecode []byte, static []core.Value) core.Value {
	r := core.IntValue(0)
	n := len(bytecode)
	base := unsafe.Pointer(unsafe.SliceData(bytecode))
	// Pull static base out of the slice header too, so the OpIndex path skips
	// the slice bounds check.
	sbase := unsafe.Pointer(unsafe.SliceData(static))
	const vsz = unsafe.Sizeof(core.Value{})

	ip := -1
	var err error
	var v core.Value
	for {
		ip++
		if ip >= n {
			break
		}
		op := *(*byte)(unsafe.Add(base, ip))
		switch op {
		case OpIndex:
			idx := *(*byte)(unsafe.Add(base, ip+1))
			v = *(*core.Value)(unsafe.Add(sbase, uintptr(idx)*vsz))
			ip += 1
		case OpInt1:
			b := *(*byte)(unsafe.Add(base, ip+1))
			v = core.Value{Type: value.Int, Immutable: true, Data: uint64(b)}
			ip += 1
		case OpInt2:
			i := binary.LittleEndian.Uint16((*[2]byte)(unsafe.Add(base, ip+1))[:])
			v = core.Value{Type: value.Int, Immutable: true, Data: uint64(i)}
			ip += 2
		case OpInt4:
			i := binary.LittleEndian.Uint32((*[4]byte)(unsafe.Add(base, ip+1))[:])
			v = core.Value{Type: value.Int, Immutable: true, Data: uint64(i)}
			ip += 4
		case OpInt8:
			i := binary.LittleEndian.Uint64((*[8]byte)(unsafe.Add(base, ip+1))[:])
			v = core.Value{Type: value.Int, Immutable: true, Data: i}
			ip += 8
		default:
			panic("unknown opcode")
		}
		r, err = r.BinaryOp(token.Add, v)
		if err != nil {
			panic(err)
		}
	}
	return r
}

func BenchmarkBytecodeIntEncoding(b *testing.B) {
	var bcIndex []byte
	var static []core.Value

	var bcInt1 []byte
	var bcInt2 []byte
	var bcInt4 []byte
	var bcInt8 []byte

	for i := 0; i < 100; i++ {
		bcIndex = append(bcIndex, OpIndex, byte(i))
		static = append(static, core.IntValue(int64(i)))

		bcInt1 = append(bcInt1, OpInt1, byte(i))

		bcInt2 = append(bcInt2, OpInt2, 0, 0)
		binary.LittleEndian.PutUint16(bcInt2[len(bcInt2)-2:], uint16(i))

		bcInt4 = append(bcInt4, OpInt4, 0, 0, 0, 0)
		binary.LittleEndian.PutUint32(bcInt4[len(bcInt4)-4:], uint32(i))

		bcInt8 = append(bcInt8, OpInt8, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.LittleEndian.PutUint64(bcInt8[len(bcInt8)-8:], uint64(i))
	}

	b.Run("Index", func(b *testing.B) {
		var r core.Value
		for i := 0; i < b.N; i++ {
			r = runBytecode(bcIndex, static)
		}
		//b.Logf("result: %s", r.String())
		_ = r
	})

	b.Run("Int1", func(b *testing.B) {
		var r core.Value
		for i := 0; i < b.N; i++ {
			r = runBytecode(bcInt1, nil)
		}
		//b.Logf("result: %s", r.String())
		_ = r
	})

	b.Run("Int2", func(b *testing.B) {
		var r core.Value
		for i := 0; i < b.N; i++ {
			r = runBytecode(bcInt2, nil)
		}
		//b.Logf("result: %s", r.String())
		_ = r
	})

	b.Run("Int4", func(b *testing.B) {
		var r core.Value
		for i := 0; i < b.N; i++ {
			r = runBytecode(bcInt4, nil)
		}
		//b.Logf("result: %s", r.String())
		_ = r
	})

	b.Run("Int8", func(b *testing.B) {
		var r core.Value
		for i := 0; i < b.N; i++ {
			r = runBytecode(bcInt8, nil)
		}
		//b.Logf("result: %s", r.String())
		_ = r
	})

	b.Run("Unsafe/Index", func(b *testing.B) {
		var r core.Value
		for i := 0; i < b.N; i++ {
			r = runBytecodeUnsafe(bcIndex, static)
		}
		_ = r
	})
	b.Run("Unsafe/Int1", func(b *testing.B) {
		var r core.Value
		for i := 0; i < b.N; i++ {
			r = runBytecodeUnsafe(bcInt1, nil)
		}
		_ = r
	})
	b.Run("Unsafe/Int2", func(b *testing.B) {
		var r core.Value
		for i := 0; i < b.N; i++ {
			r = runBytecodeUnsafe(bcInt2, nil)
		}
		_ = r
	})
	b.Run("Unsafe/Int4", func(b *testing.B) {
		var r core.Value
		for i := 0; i < b.N; i++ {
			r = runBytecodeUnsafe(bcInt4, nil)
		}
		_ = r
	})
	b.Run("Unsafe/Int8", func(b *testing.B) {
		var r core.Value
		for i := 0; i < b.N; i++ {
			r = runBytecodeUnsafe(bcInt8, nil)
		}
		_ = r
	})
}
