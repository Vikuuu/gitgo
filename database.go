package gitgo

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type BlobType int

const (
	TypeFile BlobType = iota
	TypeTree
	TypeCommit
)

var G_ignore = map[string]bool{
	".":      true,
	"..":     true,
	".gitgo": true,
}

type Database struct {
	BlobData []byte
	DbPath   string
	FilePath string
	Object   map[string]string
}

func NewDatabase(dbPath string) *Database {
	return &Database{
		DbPath: dbPath,
		Object: make(map[string]string),
	}
}

func (d *Database) Data(blobType BlobType, data []byte) {
	var buf bytes.Buffer
	buf.Write(GetPrefix(blobType, len(data)))
	buf.WriteByte(byte(0))
	buf.Write(data)

	d.BlobData = buf.Bytes()
}

func (d *Database) Store() (string, error) {
	sha := Hash(d.BlobData)
	oid := hex.EncodeToString(sha)
	return oid, d.Write(oid)
}

func (d *Database) Write(oid string) error {
	compressData := Compress(d.BlobData)
	d.objectPath(oid)
	folderPath := filepath.Join(d.DbPath, oid[:2])

	err := StoreObject(compressData, folderPath, d.FilePath)
	if err != nil {
		return err
	}

	return nil
}

func (d *Database) objectPath(oid string) {
	d.FilePath = filepath.Join(d.DbPath, oid[:2], oid[2:])
}

func (d *Database) Load(oid string) {
	d.Object[oid] = d.ReadObject(oid)
}

func (d *Database) ReadObject(oid string) string {
	return ""
}

func AuthorData(name, email string, t time.Time) string {
	utcOffset := getUTCOffset(t)
	return fmt.Sprintf("%s <%s> %d %s", name, email, t.Unix(), utcOffset)
}

func CommitData(parent, treeOID, author, message string) []byte {
	data := bytes.Buffer{}
	data.WriteString(fmt.Sprintf("tree %s\n", treeOID))
	if parent != "" {
		data.WriteString(fmt.Sprintf("parent %s\n", parent))
	}
	data.WriteString(fmt.Sprintf("author %s\n", author))
	data.WriteString(fmt.Sprintf("comitter %s\n", author))
	data.WriteString("\n")
	data.WriteString(message)

	return data.Bytes()
}

func ReadStdinMsg(file *os.File) string {
	msg, _ := io.ReadAll(file)
	return string(msg)
}

func GetPrefix(blob BlobType, size int) []byte {
	res := ""
	switch blob {
	case TypeFile:
		res = fmt.Sprintf(`blob %d`, size)
	case TypeTree:
		res = fmt.Sprintf(`tree %d`, size)
	case TypeCommit:
		res = fmt.Sprintf(`commit %d`, size)
	default:
		panic("undefined blob type")
	}

	return []byte(res)
}

func BlobData(data []byte) []byte {
	prefix := GetPrefix(TypeFile, len(data))
	data = append(data, []byte(prefix)...)
	return data
}

func StoreObject(data []byte, folderPath, filePath string) error {
	err := os.MkdirAll(folderPath, 0755)
	if err != nil {
		return err
	}

	// if the file exists exit
	_, err = os.Stat(filePath)
	if os.IsExist(err) {
		return nil
	}

	// Create a temp file for writing
	tName := generateGitTempFileName(".temp-obj-")
	tempPath := filepath.Join(folderPath, tName)
	tf, err := os.OpenFile(
		tempPath,
		os.O_RDWR|os.O_CREATE|os.O_EXCL,
		0644,
	)
	if err != nil {
		return fmt.Errorf("creating temp file: %s", err)
	}
	defer tf.Close()

	// Write to temp file
	_, err = tf.Write(data)
	if err != nil {
		return fmt.Errorf("writing to temp file: %s", err)
	}

	// rename the file
	os.Rename(tempPath, filePath)
	return nil
}

func FileMode(file string) (uint32, error) {
	f, err := os.Stat(file)
	if err != nil {
		return 0, err
	}
	// getting the stat from the underlying syscall
	// it will only work for unix like operating systems
	stat := f.Sys().(*syscall.Stat_t)
	return stat.Mode, nil
}
