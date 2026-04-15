// Package cardid mints stable 10-char alphanumeric identifiers for cards.
package cardid

import (
	"crypto/rand"
	"encoding/binary"
)

// Alphabet is the character set IDs are drawn from (62 chars).
const Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// NewID returns a fresh card ID. Exposed as a variable so tests can inject
// a deterministic generator.
var NewID = defaultNewID

func defaultNewID() string {
	var raw [40]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic("cardid: crypto/rand failed: " + err.Error())
	}
	var b [10]byte
	for i := 0; i < 10; i++ {
		n := binary.BigEndian.Uint32(raw[i*4 : i*4+4])
		b[i] = Alphabet[int(n)%len(Alphabet)]
	}
	return string(b[:])
}
