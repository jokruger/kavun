package vm

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
)

// Signer authenticates encoded bytecode for one application: Bytecode.Encode tags what it writes with Sign, and
// Bytecode.Decode refuses anything Verify does not accept. The tag covers everything after the magic and the tag
// itself: the versions, the imports, the code. Key custody, rotation and any key id belong to the implementation
// (a key id can travel inside its tag). HMACSigner is the shipped implementation; an embedder that builds in one
// place and runs in another can bring an asymmetric one (Ed25519: sign with the private key at build, verify with
// the public key at run).
type Signer interface {
	Sign(data []byte) (tag []byte, err error)
	Verify(data, tag []byte) error // nil when tag authenticates data
}

// MinHMACKeyLen is the shortest key HMACSigner accepts: the HMAC-SHA256 output size.
const MinHMACKeyLen = sha256.Size

// HMACSigner answers a Signer computing HMAC-SHA256 with key; Verify compares in constant time. A key shorter than
// MinHMACKeyLen makes Sign and Verify fail. The key is copied.
func HMACSigner(key []byte) Signer {
	return hmacSigner{key: append([]byte(nil), key...)}
}

type hmacSigner struct {
	key []byte
}

func (s hmacSigner) mac(data []byte) ([]byte, error) {
	if len(s.key) < MinHMACKeyLen {
		return nil, fmt.Errorf("HMAC key too short: %d bytes (min: %d)", len(s.key), MinHMACKeyLen)
	}
	m := hmac.New(sha256.New, s.key)
	m.Write(data)
	return m.Sum(nil), nil
}

func (s hmacSigner) Sign(data []byte) ([]byte, error) {
	return s.mac(data)
}

func (s hmacSigner) Verify(data, tag []byte) error {
	want, err := s.mac(data)
	if err != nil {
		return err
	}
	if !hmac.Equal(want, tag) {
		return errors.New("HMAC mismatch")
	}
	return nil
}
