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
	keys     *datastr.SortedSet
	lockfile *lockFile
	changed  bool
	parents  map[string]*datastr.Set
}

func NewIndex() *Index {
	return &Index{
		entries:  make(map[string]IndexEntry),
		keys:     datastr.NewSortedSet(),
		lockfile: lockInitialize(filepath.Join(GITPATH, "index")),
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
			Stat: strconv.Itoa(entry.Mode),
		})
	}
	return e
}

func IndexHoldForUpdate() (bool, *Index, error) {
	index := NewIndex()
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
	fileReader, err := os.Open(filepath.Join(GITPATH, "index"))
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

	// TODO: Remove the comments
	// I do not need to do this because I am writing
	// it in the function that called this function
	// _, err = h.Write(data)
	// if err != nil {
	// 	return nil, err
	// }
	// if n != hn {
	// 	return nil, errors.New("read and write not equal")
	// }
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
	log.Printf(
		"storeEntryByte: raw entry len=%d, entry bytes (40–60): % x",
		len(entry),
		entry[40:60],
	)

	log.Printf("raw tail from offset62: % x", entry[62:])

	fNameInEntry := entry[62:]
	log.Printf("slicing filename: entry[62:%d] ⇒ len=%d", len(entry), len(fNameInEntry))

	log.Printf("storeEntryByte: fNameInEntry bytes: % x len: %d", fNameInEntry, len(fNameInEntry))
	log.Printf("storeEntryByte: fNameInEntry as string (raw): %q", fNameInEntry)

	oidInEntry := entry[40:60]
	nullIdx := bytes.IndexByte(fNameInEntry, byte(0))
	log.Printf(
		"computed nullIdx=%d; (nullIdx+1) mod ENTRYBLOCK = %d",
		nullIdx,
		(nullIdx+1)%ENTRYBLOCK,
	)

	log.Printf("storeEntryByte: nullIdx=%d, fNameInEntry length=%d", nullIdx, len(fNameInEntry))

	// for i, v := range entry {
	// 	log.Printf("%d element of entry: %s\n", i, string(v))
	// }
	log.Printf("Null Index: %d\n", nullIdx)
	fileName := ""
	if nullIdx != -1 {
		fileName = string(fNameInEntry[:nullIdx])
	} else {
		fileName = string(fNameInEntry[:])
	}
	log.Printf("storeEntryByte: about to Stat() fileName=%q", fileName)
	log.Printf("final filename string=%q (len %d)", fileName, len(fileName))

	stat, err := os.Stat(fileName)
	if err != nil {
		log.Fatalln(err)
	}

	i.Add(fileName, hex.EncodeToString(oidInEntry), stat)
}

func verifyChecksum(f io.Reader, h *bytes.Buffer) {
	checksum := make([]byte, 20)
	c := bytes.NewBuffer(checksum)
	_, err := f.Read(c.AvailableBuffer())
	if err != nil {
		log.Fatalln(err)
	}
	// if n != 20 {
	// 	log.Fatalln("n not equals to 20 in verifychecksum: n = ", n)
	// }
	currChecksum := sha1.Sum(h.Bytes())
	currC := bytes.NewBuffer(currChecksum[:])
	if bytes.Equal(c.Bytes(), currC.Bytes()) {
		log.Fatalln("checksums not equal")
	}
}

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
	// b, err := i.lockfile.holdForUpdate()
	// if err != nil {
	// 	return false, err
	// }
	// if !b {
	// 	return false, nil
	// }
	if !i.changed {
		return false, i.lockfile.rollback()
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

func (i *Index) Release() error { return i.lockfile.rollback() }
