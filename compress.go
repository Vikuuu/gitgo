package gitgo

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
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

func GetDecompress(data []byte) ([]byte, error) {
	var d bytes.Buffer
	b := bytes.NewReader(data)
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	io.Copy(&d, r)
	r.Close()
	res := d.Bytes()
	idx := bytes.IndexByte(res, byte(0))
	if idx == -1 {
		return nil, fmt.Errorf("no null byte in blob")
	}
	return res[idx:], nil
}
