package gitgo

import (
	"os"
	"path/filepath"
)

type ref struct {
	pathname string
	headPath string
}

func RefInitialize(pathname string) ref {
	r := ref{pathname: pathname}
	r.headPath = filepath.Join(r.pathname, "HEAD")
	return r
}

func (r ref) UpdateHead(oid []byte) error {
	flags := os.O_WRONLY | os.O_CREATE
	refFile, err := os.OpenFile(r.head_path(), flags, 0644)
	if err != nil {
		return err
	}
	defer refFile.Close()

	_, err = refFile.Write(oid)
	if err != nil {
		return err
	}

	return nil
}

func (r ref) head_path() string {
	return r.headPath
}

func (r ref) Read_head() string {
	_, err := os.Stat(r.headPath)
	if os.IsNotExist(err) {
		return ""
	}

	content, _ := os.ReadFile(r.headPath)
	return string(content)
}
