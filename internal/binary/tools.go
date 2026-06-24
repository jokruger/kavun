package binary

import (
	"encoding/binary"
	"fmt"
)

func AppendUint64(b []byte, v uint64) []byte {
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], v)
	return append(b, tmp[:]...)
}

func AppendBytes(b []byte, payload []byte) []byte {
	b = AppendUint64(b, uint64(len(payload)))
	return append(b, payload...)
}

func ReadUint64(data []byte, offset *int, field string) (uint64, error) {
	if len(data)-*offset < 8 {
		return 0, fmt.Errorf("%s: expected 8 bytes, got %d", field, len(data)-*offset)
	}
	v := binary.LittleEndian.Uint64(data[*offset : *offset+8])
	*offset += 8
	return v, nil
}

func ReadBytes(data []byte, offset *int, field string) ([]byte, error) {
	l, err := ReadUint64(data, offset, field+" (length)")
	if err != nil {
		return nil, err
	}
	remaining := len(data) - *offset
	if l > uint64(remaining) {
		return nil, fmt.Errorf("%s: declared length %d exceeds remaining data %d", field, l, remaining)
	}
	end := *offset + int(l)
	b := data[*offset:end]
	*offset = end
	return b, nil
}
