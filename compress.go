package gitgo

import (
	"bytes"
	"compress/zlib"
	"slices"
)

func getCompressBuf(prefix, data []byte) bytes.Buffer {
	var buf bytes.Buffer
	prefix = append(prefix, byte(0))
	w := zlib.NewWriter(&buf)
	w.Write(slices.Concat(prefix, data))
	w.Close()
	return buf
}
