package gitgo

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Vikuuu/gitgo/internal/datastr"
)

const (
	HEADERSIZE   = 12
	HEADERFORMAT = "a4N2"
	SIGNATURE    = "DIRC"
	VERSION      = 2

	ENTRYFORMAT  = "N10H40Z*"
	ENTRYBLOCK   = 8
	ENTRYMINSIZE = 64
)

type Index struct {
	entries  map[string]IndexEntry
	repoPath string
	path     string
	keys     *datastr.SortedSet
	lockfile *lockFile
	changed  bool
	parents  map[string]*datastr.Set
}

func NewIndex(repoPath, gitPath string) *Index {
	return &Index{
		entries:  make(map[string]IndexEntry),
		keys:     datastr.NewSortedSet(),
		repoPath: repoPath,
		path:     filepath.Join(gitPath, "index"),
		lockfile: lockInitialize(filepath.Join(gitPath, "index")),
		changed:  false,
		parents:  make(map[string]*datastr.Set),
	}
}

func (i *Index) Entries() []Entries {
	e := []Entries{}

	it := i.keys.Iterator()
	for it.Next() {
		path := it.Key()
		entry := i.entries[path]
		e = append(e, Entries{
			Path: path,
			OID:  entry.Oid,
			Stat: strconv.Itoa(int(entry.Mode)),
		})
	}
	return e
}

func IndexHoldForUpdate(repoPath, gitPath string) (bool, *Index, error) {
	index := NewIndex(repoPath, gitPath)
	b, err := index.lockfile.holdForUpdate()
	if err != nil {
		return false, index, err
	}
	if !b {
		return false, index, nil
	}

	// load the index file
	err = index.Load()

	return true, index, nil
}

func (i *Index) Load() error {
	fileReader, err := os.Open(i.path)
	if err != nil {
		return err
	}
	defer fileReader.Close()
	hash := new(bytes.Buffer)
	count := i.readHeader(fileReader, hash)
	i.readEntries(fileReader, count, hash)
	verifyChecksum(fileReader, hash)

	return nil
}

func (i *Index) readHeader(f *os.File, h *bytes.Buffer) int {
	data, err := read(f, HEADERSIZE)
	if err != nil {
		log.Fatalf("%s\n", err)
	}
	signature, version, count := data[:4], data[4:8], data[8:12]
	if string(signature) != SIGNATURE {
		log.Fatalf("signature: expected %s got %s\n", SIGNATURE, signature)
	}
	if binary.BigEndian.Uint32(version) != VERSION {
		log.Fatalf("version: expected %d got %d\n", binary.BigEndian.Uint32(version), VERSION)
	}
	h.Write(data)
	return int(binary.BigEndian.Uint32(count))
}

func read(f io.Reader, size int) ([]byte, error) {
	data := make([]byte, size)
	_, err := f.Read(data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (i *Index) readEntries(r io.Reader, count int, h *bytes.Buffer) {
	for range count {
		entry, err := read(r, ENTRYMINSIZE)
		if err != nil {
			log.Fatalln(err)
		}

		for entry[len(entry)-1] != byte(0) {
			e, err := read(r, ENTRYBLOCK)
			if err != nil {
				log.Fatalln(err)
			}

			entry = append(entry, e...)
		}

		i.storeEntryByte(entry)
		_, err = h.Write(entry)
		if err != nil {
			log.Fatalf("err writing entry to hash: %s\n", err)
		}
	}
}

func (i *Index) storeEntryByte(entry []byte) {
	const headerLen = 62
	if len(entry) < headerLen {
		return
	}

	ctime := entry[0:4]
	ctimeN := entry[4:8]
	mtime := entry[8:12]
	mtimeN := entry[12:16]
	dev := entry[16:20]
	ino := entry[20:24]
	mode := entry[24:28]
	uid := entry[28:32]
	gid := entry[32:36]
	size := entry[36:40]
	oidInEntry := entry[40:60]
	flag := entry[60:62]
	fNameInEntry := entry[62:]

	if idx := bytes.IndexByte(fNameInEntry, 0); idx != -1 {
		fNameInEntry = fNameInEntry[:idx]
	}

	mtimeVal := int64(binary.BigEndian.Uint32(mtime))
	mtimeNVal := int64(binary.BigEndian.Uint32(mtimeN))
	ctimeVal := int64(binary.BigEndian.Uint32(ctime))
	ctimeNVal := int64(binary.BigEndian.Uint32(ctimeN))

	devVal := uint64(binary.BigEndian.Uint32(dev))
	inoVal := uint64(binary.BigEndian.Uint32(ino))
	modeVal := uint32(binary.BigEndian.Uint32(mode))
	uidVal := uint32(binary.BigEndian.Uint32(uid))
	gidVal := uint32(binary.BigEndian.Uint32(gid))
	sizeVal := int64(binary.BigEndian.Uint32(size))
	flagVal := uint32(binary.BigEndian.Uint16(flag))

	if len(oidInEntry) != 20 {
		panic("storeEntryByte: oid len != 20 bytes")
	}

	i.add(&IndexEntry{
		Path:      string(fNameInEntry),
		Oid:       hex.EncodeToString(oidInEntry),
		Mtime:     mtimeVal,
		MtimeNsec: mtimeNVal,
		Ctime:     ctimeVal,
		CtimeNsec: ctimeNVal,
		Dev:       devVal,
		Ino:       inoVal,
		Mode:      modeVal,
		Uid:       uidVal,
		Gid:       gidVal,
		Size:      sizeVal,
		Flags:     flagVal,
	})
}

func verifyChecksum(f io.Reader, h *bytes.Buffer) {
	checksum := make([]byte, 20)
	c := bytes.NewBuffer(checksum)
	_, err := f.Read(c.AvailableBuffer())
	if err != nil {
		log.Fatalln(err)
	}

	currChecksum := sha1.Sum(h.Bytes())
	currC := bytes.NewBuffer(currChecksum[:])
	if bytes.Equal(c.Bytes(), currC.Bytes()) {
		log.Fatalln("checksums not equal")
	}
}

func (i *Index) add(entry *IndexEntry) {
	i.discardConflict(entry)
	i.storeEntry(entry)
	i.changed = true
}

// This function is being used in the tests.
func (i *Index) Add(path, oid string, stat os.FileInfo) {
	entry := NewIndexEntry(path, oid, stat)
	i.discardConflict(entry)
	i.storeEntry(entry)
	i.changed = true
}

func (i *Index) discardConflict(e *IndexEntry) {
	var dirPaths []string
	d := filepath.Dir(e.Path)
	dirPaths = append(dirPaths, d)
	for d != "." && d != ".." && d != string(filepath.Separator) {
		d = filepath.Dir(d)
		dirPaths = append(dirPaths, d)
	}

	// Remove files if they are now changed to dir
	for _, dirPath := range dirPaths {
		i.keys.Remove(dirPath)
		delete(i.entries, dirPath)
	}

	// Remove dirs if they are now changed to file
	i.removeChildren(e.Path)
}

func (i *Index) removeEntry(path string) {
	entry, ok := i.entries[path]
	if !ok {
		return
	}
	i.keys.Remove(entry.Path)
	delete(i.entries, entry.Path)

	var dirPaths []string
	d := filepath.Dir(path)
	dirPaths = append(dirPaths, d)
	for d != "." && d != ".." && d != string(filepath.Separator) {
		d = filepath.Dir(d)
		dirPaths = append(dirPaths, d)
	}

	for _, d := range dirPaths {
		dir := d
		i.parents[dir].Remove(entry.Path)
		if i.parents[dir].IsEmpty() {
			delete(i.parents, dir)
		}
	}
}

func (i *Index) removeChildren(p string) {
	pSet, ok := i.parents[p]
	if !ok {
		return
	}
	original := pSet.GetAll()
	children := make([]string, len(original))
	copy(children, original)
	for _, child := range children {
		i.removeEntry(child)
	}
}

func (i *Index) storeEntry(e *IndexEntry) {
	i.keys.Add(e.Path)
	i.entries[e.Path] = *e

	var parents []string
	p := filepath.Dir(e.Path)
	parents = append(parents, p)
	for p != "." && p != ".." && p != string(filepath.Separator) {
		p = filepath.Dir(p)
		parents = append(parents, p)
	}

	for _, p := range parents {
		pSet, ok := i.parents[p]
		if !ok {
			pSet = datastr.NewSet()
			i.parents[p] = pSet
		}
		pSet.Add(e.Path)
	}
}

func (i *Index) WriteUpdate() (bool, error) {
	if !i.changed {
		return false, i.lockfile.rollback()
	}

	buf := new(bytes.Buffer) // Makes a new buffer and returns its pointer
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

	// Getting the hash of the whole content in the
	// index file
	content := buf.Bytes()
	bufHash := sha1.Sum(content)
	buf.Write(bufHash[:])

	i.lockfile.write(buf.Bytes())
	i.lockfile.commit()
	i.changed = false
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

	writeU32 := func(v uint32, what string) error {
		if err := binary.Write(b, binary.BigEndian, v); err != nil {
			return fmt.Errorf("writing %s: %w", what, err)
		}
		return nil
	}
	writeU16 := func(v uint16, what string) error {
		if err := binary.Write(b, binary.BigEndian, v); err != nil {
			return fmt.Errorf("writing %s: %w", what, err)
		}
		return nil
	}

	if err := writeU32(uint32(entry.Ctime), "ctime"); err != nil {
		return nil, err
	}
	if err := writeU32(uint32(entry.CtimeNsec), "ctime nsec"); err != nil {
		return nil, err
	}
	if err := writeU32(uint32(entry.Mtime), "mtime"); err != nil {
		return nil, err
	}
	if err := writeU32(uint32(entry.MtimeNsec), "mtime nsec"); err != nil {
		return nil, err
	}
	if err := writeU32(uint32(entry.Dev), "dev"); err != nil {
		return nil, err
	}
	if err := writeU32(uint32(entry.Ino), "ino"); err != nil {
		return nil, err
	}
	if err := writeU32(uint32(entry.Mode), "mode"); err != nil {
		return nil, err
	}
	if err := writeU32(uint32(entry.Uid), "uid"); err != nil {
		return nil, err
	}
	if err := writeU32(uint32(entry.Gid), "gid"); err != nil {
		return nil, err
	}
	if err := writeU32(uint32(entry.Size), "size"); err != nil {
		return nil, err
	}

	oid, err := hex.DecodeString(entry.Oid)
	if err != nil {
		return nil, fmt.Errorf("decoding string oid: %s", err)
	}
	if len(oid) != 20 {
		return nil, fmt.Errorf("oid must be 20 bytes got %d", len(oid))
	}

	if _, err := b.Write(oid); err != nil {
		return nil, fmt.Errorf("writing oid: %w", err)
	}

	nameLen := len(entry.Path)
	if nameLen > 0xFFF {
		nameLen = 0xFFF
	}
	flagVal := (uint16(entry.Flags) & 0xF000) | uint16(nameLen&0x0FFF)
	if err := writeU16(flagVal, "flag"); err != nil {
		return nil, err
	}

	if _, err := b.Write([]byte(entry.Path)); err != nil {
		return nil, fmt.Errorf("writing entry path: %s", err)
	}
	if err = b.WriteByte(0); err != nil {
		return nil, fmt.Errorf("writing null byte after entry: %s", err)
	}

	missing := (8 - (b.Len() % 8)) % 8
	for range missing {
		if err := b.WriteByte(0); err != nil {
			return nil, fmt.Errorf("writing padding: %w", err)
		}
	}
	return b.Bytes(), nil
}

func (i *Index) Release() error { return i.lockfile.rollback() }

func (i *Index) IsTracked(path string) bool {
	cleanPath := filepath.Clean(path)
	_, pres := i.entries[cleanPath]
	_, pPres := i.parents[cleanPath]
	return pres || pPres
}

func (i *Index) IndexEntries() map[string]IndexEntry {
	return i.entries
}
