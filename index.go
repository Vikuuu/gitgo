package gitgo

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vikuuu/gitgo/internal/datastr"
)

type Index struct {
	entries  map[string]IndexEntry
	keys     *datastr.SortedSet
	lockfile *lockFile
}

func NewIndex() *Index {
	return &Index{
		entries:  make(map[string]IndexEntry),
		keys:     datastr.NewSortedSet(),
		lockfile: lockInitialize(filepath.Join(GITPATH, "index")),
	}
}

func (i *Index) Add(path, oid string, stat os.FileInfo) {
	entry := NewIndexEntry(path, oid, stat)
	i.keys.Add(path)
	i.entries[path] = *entry
}

func (i *Index) WriteUpdate() (bool, error) {
	b, err := i.lockfile.holdForUpdate()
	if err != nil {
		return false, err
	}
	if !b {
		return false, nil
	}

	buf := new(bytes.Buffer) // makes a new buffer and returns its pointer
	writeHeader(buf, len(i.entries))
	it := i.keys.Iterator()
	for it.Next() {
		path := it.Key()
		entry := i.entries[path]
		data, err := writeIndexEntry(entry)
		if err != nil {
			return true, err
		}
		buf.Write(data)
	}

	// getting the hash of the whole content in the
	// index file
	content := buf.Bytes()
	bufHash := sha1.Sum(content)
	buf.Write(bufHash[:])

	i.lockfile.write(buf.Bytes())
	i.lockfile.commit()
	return true, nil
}

func writeHeader(buf *bytes.Buffer, entryLen int) error {
	_, err := buf.Write([]byte("DIRC"))
	if err != nil {
		return fmt.Errorf("writing index header: %s", err)
	}
	b := new(bytes.Buffer)
	versionNum := uint32(2)
	entriesNum := uint32(entryLen)
	binary.Write(b, binary.BigEndian, versionNum)
	binary.Write(b, binary.BigEndian, entriesNum)

	_, err = buf.Write(b.Bytes())
	if err != nil {
		return fmt.Errorf("writing index header: %s", err)
	}
	return nil
}

func writeIndexEntry(entry IndexEntry) ([]byte, error) {
	b := new(bytes.Buffer)

	err := binary.Write(b, binary.BigEndian, uint32(entry.Ctime))
	if err != nil {
		return nil, fmt.Errorf("writing ctime: %s", err)
	}
	err = binary.Write(b, binary.BigEndian, uint32(entry.CtimeNsec))
	if err != nil {
		return nil, fmt.Errorf("writing ctime nsec: %s", err)
	}
	err = binary.Write(b, binary.BigEndian, uint32(entry.Mtime))
	if err != nil {
		return nil, fmt.Errorf("writing mtime: %s", err)
	}
	err = binary.Write(b, binary.BigEndian, uint32(entry.MtimeNsec))
	if err != nil {
		return nil, fmt.Errorf("writing mtime nsec: %s", err)
	}
	err = binary.Write(b, binary.BigEndian, uint32(entry.Dev))
	if err != nil {
		return nil, fmt.Errorf("writing dev: %s", err)
	}
	err = binary.Write(b, binary.BigEndian, uint32(entry.Ino))
	if err != nil {
		return nil, fmt.Errorf("writing ino: %s", err)
	}
	err = binary.Write(b, binary.BigEndian, uint32(entry.Mode))
	if err != nil {
		return nil, fmt.Errorf("writing mode: %s", err)
	}
	err = binary.Write(b, binary.BigEndian, uint32(entry.Uid))
	if err != nil {
		return nil, fmt.Errorf("writing uid: %s", err)
	}
	err = binary.Write(b, binary.BigEndian, uint32(entry.Gid))
	if err != nil {
		return nil, fmt.Errorf("writing gid: %s", err)
	}
	err = binary.Write(b, binary.BigEndian, uint32(entry.Size))
	if err != nil {
		return nil, fmt.Errorf("writing size: %s", err)
	}
	oid, err := hex.DecodeString(entry.Oid)
	if err != nil {
		return nil, fmt.Errorf("decoding string oid: %s", err)
	}
	err = binary.Write(b, binary.BigEndian, oid)
	if err != nil {
		return nil, fmt.Errorf("writing oid: %s", err)
	}
	err = binary.Write(b, binary.BigEndian, uint16(entry.Flags))
	if err != nil {
		return nil, fmt.Errorf("writing flag: %s", err)
	}
	_, err = b.Write([]byte(entry.Path))
	if err != nil {
		return nil, fmt.Errorf("writing entry path: %s", err)
	}
	err = b.WriteByte(0)
	if err != nil {
		return nil, fmt.Errorf("writing null byte after entry: %s", err)
	}
	missing := (8 - (b.Len() % 8)) % 8
	for range missing {
		b.WriteByte(0)
	}
	return b.Bytes(), nil
}
