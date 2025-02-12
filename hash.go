package gitgo

import (
	"crypto/sha1"
	"io"
)

func getHash(prefix, data string) []byte {
	h := sha1.New()
	p := append([]byte(prefix), byte(0))
	io.WriteString(h, string(p))
	io.WriteString(h, data)
	shaCode := h.Sum(nil)
	return shaCode
}
