package gitgo

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var GITGO_IGNORE = []string{".", "..", ".gitgo"}

var g_ignore = map[string]bool{
	".":      true,
	"..":     true,
	".gitgo": true,
}

type Author struct {
	Name      string
	Email     string
	Timestamp time.Time
}

type Commit struct {
	Parent  string
	TreeOID string
	Author  string
	Message string
	Data    string
	Prefix  string
}

func (a Author) New() string {
	unixTimeStamp := a.Timestamp.Unix()
	utcOffset := getUTCOffset(a.Timestamp)
	return fmt.Sprintf("%s <%s> %d %s", a.Name, a.Email, unixTimeStamp, utcOffset)
}

func (c Commit) New() *Commit {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("tree %s", c.TreeOID))
	if c.Parent != "" {
		lines = append(lines, fmt.Sprintf("parent %s", c.Parent))
	}
	lines = append(lines, fmt.Sprintf("author %s", c.Author))
	lines = append(lines, fmt.Sprintf("comitter %s", c.Author))
	lines = append(lines, "")
	lines = append(lines, c.Message)
	c.Data = strings.Join(lines, "\n")
	c.Prefix = fmt.Sprintf(`commit %d`, len(c.Data))
	return &c
}

func (c Commit) Type() string {
	return "commit"
}

func ReadStdinMsg() string {
	msg, _ := io.ReadAll(os.Stdin)
	return string(msg)
}

type Blob struct {
	Prefix string
	Data   []byte
}

type TreeBlob struct {
	Prefix string
	Data   bytes.Buffer
}

func (b Blob) Init() *Blob {
	prefix := fmt.Sprintf(`blob %d`, len(b.Data))
	b.Prefix = prefix
	return &b
}

func (t TreeBlob) Init() *TreeBlob {
	prefix := fmt.Sprintf(`tree %d`, t.Data.Len())
	t.Prefix = prefix
	return &t
}

func (b *Blob) Store() (string, error) {
	return StoreBlobObject(b.Data, b.Prefix)
}

func (t *TreeBlob) Store() (string, error) {
	return StoreTreeObject(t.Data, t.Prefix)
}

func (c *Commit) Store() (string, error) {
	return StoreCommitObject(c.Data, c.Prefix)
}

func StoreTreeObject(treeEntry bytes.Buffer, prefix string) (string, error) {
	// treePrefix := fmt.Sprintf(`tree %d`, treeEntry.Len())
	treeSHA := getHash(prefix, treeEntry.String())
	hexTreeSha := hex.EncodeToString(treeSHA)
	// fmt.Printf("Tree: %s", hexTreeSha)
	tree := getCompressBuf([]byte(prefix), treeEntry.Bytes())
	folderPath := filepath.Join(DBPATH, hexTreeSha[:2])
	permPath := filepath.Join(DBPATH, hexTreeSha[:2], hexTreeSha[2:])
	err := StoreObject(tree, prefix, folderPath, permPath)
	if err != nil {
		return "", err
	}
	return hexTreeSha, nil
}

func StoreBlobObject(blobData []byte, prefix string) (string, error) {
	// blobPrefix := fmt.Sprintf(`blob %d`, len(blobData))

	// getting the SHA-1
	blobSHA := getHash(prefix, string(blobData)) // []byte
	blob := getCompressBuf([]byte(prefix), blobData)
	hexBlobSha := hex.EncodeToString(blobSHA)
	folderPath := filepath.Join(DBPATH, hexBlobSha[:2])
	permPath := filepath.Join(DBPATH, hexBlobSha[:2], hexBlobSha[2:])
	err := StoreObject(blob, prefix, folderPath, permPath)
	if err != nil {
		return "", err
	}

	return hexBlobSha, nil
}

func StoreCommitObject(commitData, prefix string) (string, error) {
	// commitPrefix := fmt.Sprintf(`commit %d`, len(commitData))
	commitHash := getHash(prefix, commitData)
	commit := getCompressBuf([]byte(prefix), []byte(commitData))
	hexCommitHash := hex.EncodeToString(commitHash)
	folderPath := filepath.Join(DBPATH, hexCommitHash[:2])
	permPath := filepath.Join(DBPATH, hexCommitHash[:2], hexCommitHash[2:])
	err := StoreObject(commit, prefix, folderPath, permPath)
	if err != nil {
		return "", err
	}

	return hexCommitHash, nil
}

func StoreObject(
	data bytes.Buffer,
	prefix, folderPath, PermPath string,
) error {
	err := os.MkdirAll(folderPath, 0755)
	if err != nil {
		return err
	}

	// if the file exists exit
	_, err = os.Stat(PermPath)
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
	_, err = tf.Write(data.Bytes())
	if err != nil {
		return fmt.Errorf("writing to temp file: %s", err)
	}

	// rename the file
	os.Rename(tempPath, PermPath)
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

func StoreOnDisk(path string) ([]Entries, error) {
	files, err := ListFiles(path)
	if err != nil {
		return nil, err
	}
	var entries []Entries
	for _, f := range files {
		err = blobStore(f, &entries)
		if err != nil {
			return nil, fmt.Errorf("from storeOndisk %s", err)
		}
	}

	return entries, nil
}

func blobStore(f string, entries *[]Entries) error {
	fp := filepath.Join(ROOTPATH, f)
	data, err := os.ReadFile(fp)
	if err != nil {
		return err
	}
	blob := Blob{Data: data}.Init()
	fileMode, err := FileMode(fp)
	if err != nil {
		return err
	}

	hash, err := blob.Store()
	entry := Entries{
		Path: f,
		OID:  hash,
		Stat: strconv.FormatUint(uint64(fileMode), 8),
	}

	*entries = append(*entries, entry)
	return nil
}
