package gitgo

import "crypto/sha1"

func Hash(data []byte) []byte {
	h := sha1.New()
	h.Write(data)
	return h.Sum(nil)
}
