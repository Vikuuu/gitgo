package gitgo

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

func Compress(data []byte) []byte {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)

	if _, err := w.Write(data); err != nil {
		panic("error compressing: " + err.Error())
	}
	if err := w.Close(); err != nil {
		panic("error closing zlib comp: " + err.Error())
	}

	return b.Bytes()
}

func Decompress(data []byte) ([]byte, error) {
	var d bytes.Buffer
	r, err := zlib.NewReader(&d)
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
