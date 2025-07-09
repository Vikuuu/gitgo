package gitgo

import (
	"errors"
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

	if _, err := lockfile.holdForUpdate(); err != nil {
		return err
	}

	oid = append(oid, '\n')
	lockfile.write(oid)
	lockfile.commit()

	return nil
}

func (r ref) HeadPath() string {
	return r.headPath
}

func (r ref) ReadHead() string {
	_, err := os.Stat(r.headPath)
	if os.IsNotExist(err) {
		return ""
	}

	content, _ := os.ReadFile(r.headPath)
	return string(content)
}
