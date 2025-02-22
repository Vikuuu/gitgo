package gitgo

import (
	"errors"
	"log"
	"os"
	"sync"
)

var (
	ErrMissingParent = errors.New("Missing Parent")
	ErrNoPermission  = errors.New("No Permission")
	ErrStaleLock     = errors.New("Stale Lock")
)

type lockFile struct {
	FilePath string
	LockPath string
	Lock     *os.File
	mu       sync.Mutex
}

func lockInitialize(path string) *lockFile {
	lockPath := path + ".lock"
	return &lockFile{
		FilePath: path,
		LockPath: lockPath,
	}
}

func (l *lockFile) holdForUpdate() (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.Lock != nil {
		return true, nil // lock already aquired
	}

	file, err := os.OpenFile(l.LockPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		} else if os.IsNotExist(err) {
			return false, ErrMissingParent
		} else if os.IsPermission(err) {
			return false, ErrNoPermission
		}
		return false, err
	}

	l.Lock = file
	return true, nil
}

func (l *lockFile) write(data []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.errOnStaleLock()
	_, err := l.Lock.Write(data)
	if err != nil {
		log.Fatalf("Write error: %s\n", err)
	}
}

func (l *lockFile) commit() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.errOnStaleLock()

	err := l.Lock.Close()
	if err != nil {
		log.Fatalf("Err closing file: %s\n", err)
	}

	err = os.Rename(l.LockPath, l.FilePath)
	if err != nil {
		log.Fatalf("Err renaming file: %s\n", err)
	}

	l.Lock = nil
}

func (l *lockFile) errOnStaleLock() {
	if l.Lock == nil {
		log.Fatalf("Err: %s\nNot holding lock on file: %s", ErrStaleLock, l.LockPath)
	}
}
