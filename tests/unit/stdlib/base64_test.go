package stdlib

import "testing"

var base64Bytes1 = []byte{
	0x06, 0xAC, 0x76, 0x1B, 0x1D, 0x6A, 0xFA, 0x9D, 0xB1, 0xA0,
}

const (
	base64Std    = "Bqx2Gx1q+p2xoA=="
	base64URL    = "Bqx2Gx1q-p2xoA=="
	base64RawStd = "Bqx2Gx1q+p2xoA"
	base64RawURL = "Bqx2Gx1q-p2xoA"
)

func TestBase64(t *testing.T) {
	module(t, `base64`).call(rta, "encode", base64Bytes1).expect(rta, base64Std)
	module(t, `base64`).call(rta, "decode", base64Std).expect(rta, base64Bytes1)
	module(t, `base64`).call(rta, "url_encode", base64Bytes1).expect(rta, base64URL)
	module(t, `base64`).call(rta, "url_decode", base64URL).expect(rta, base64Bytes1)
	module(t, `base64`).call(rta, "raw_encode", base64Bytes1).expect(rta, base64RawStd)
	module(t, `base64`).call(rta, "raw_decode", base64RawStd).expect(rta, base64Bytes1)
	module(t, `base64`).call(rta, "raw_url_encode", base64Bytes1).expect(rta, base64RawURL)
	module(t, `base64`).call(rta, "raw_url_decode", base64RawURL).expect(rta, base64Bytes1)
}
