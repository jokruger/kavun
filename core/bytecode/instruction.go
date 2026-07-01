package bytecode

import (
	"fmt"
	"strings"
)

type Instruction struct {
	Op  Opcode
	Op1 uint8
	Op2 uint16
	Op3 uint32
}

func (i Instruction) String() string {
	return fmt.Sprintf("%s(%d, %d, %d)", i.Op.String(), i.Op1, i.Op2, i.Op3)
}

func (i Instruction) EncodeBinary() ([]byte, error) {
	var b [8]byte
	b[0] = byte(i.Op)
	b[1] = i.Op1
	b[2] = byte(i.Op2 >> 8)
	b[3] = byte(i.Op2)
	b[4] = byte(i.Op3 >> 24)
	b[5] = byte(i.Op3 >> 16)
	b[6] = byte(i.Op3 >> 8)
	b[7] = byte(i.Op3)
	return b[:], nil
}

func (i *Instruction) DecodeBinary(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("binary decoding failed: expected at least 8 bytes for instruction, got %d", len(data))
	}
	i.Op = Opcode(data[0])
	i.Op1 = data[1]
	i.Op2 = uint16(data[2])<<8 | uint16(data[3])
	i.Op3 = uint32(data[4])<<24 | uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7])
	return nil
}

func (i Instruction) GobEncode() ([]byte, error) {
	return i.EncodeBinary()
}

func (i *Instruction) GobDecode(data []byte) error {
	return i.DecodeBinary(data)
}

type Instructions []Instruction

func (is Instructions) String() string {
	var sb strings.Builder
	for i, inst := range is {
		fmt.Fprintf(&sb, "%04d: %s\n", i, inst.String())
	}
	return sb.String()
}

func (is Instructions) Equal(other Instructions) bool {
	if len(is) != len(other) {
		return false
	}
	for i := range is {
		if is[i] != other[i] {
			return false
		}
	}
	return true
}

func (is Instructions) EncodeBinary() ([]byte, error) {
	var b []byte
	for _, inst := range is {
		eb, err := inst.EncodeBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to encode instruction: %w", err)
		}
		b = append(b, eb...)
	}
	return b, nil
}

func (is *Instructions) DecodeBinary(data []byte) error {
	if len(data)%8 != 0 {
		return fmt.Errorf("binary decoding failed: expected data length to be a multiple of 8, got %d", len(data))
	}
	count := len(data) / 8
	*is = make(Instructions, count)
	for i := 0; i < count; i++ {
		offset := i * 8
		if err := (*is)[i].DecodeBinary(data[offset : offset+8]); err != nil {
			return fmt.Errorf("failed to decode instruction at index %d: %w", i, err)
		}
	}
	return nil
}

func (is Instructions) GobEncode() ([]byte, error) {
	return is.EncodeBinary()
}

func (is *Instructions) GobDecode(data []byte) error {
	return is.DecodeBinary(data)
}
