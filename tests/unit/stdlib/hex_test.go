package stdlib

import "testing"

var hexBytes1 = []byte{
	0x06, 0xAC, 0x76, 0x1B, 0x1D, 0x6A, 0xFA, 0x9D, 0xB1, 0xA0,
}

const hex1 = "06ac761b1d6afa9db1a0"

func TestHex(t *testing.T) {
	module(t, `hex`).call(rta, "encode", hexBytes1).expect(rta, hex1)
	module(t, `hex`).call(rta, "decode", hex1).expect(rta, hexBytes1)
}
