package gitgo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrLockDenied = errors.New("Lock Denied")

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
	lockfile := lockInitialize(r.headPath)

	if lock, _ := lockfile.holdForUpdate(); !lock {
		return fmt.Errorf("Err: %s\nCould not aquire lock on file: %s", ErrLockDenied, r.headPath)
	}

	oid = append(oid, '\n')
	lockfile.write(oid)
	lockfile.commit()

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
