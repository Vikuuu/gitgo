package gitgo

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var GITGO_IGNORE = []string{".", "..", ".gitgo"}

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
}

func (a Author) New() string {
	unixTimeStamp := a.Timestamp.Unix()
	utcOffset := getUTCOffset(a.Timestamp)
	return fmt.Sprintf("%s <%s> %d %s", a.Name, a.Email, unixTimeStamp, utcOffset)
}

func (c Commit) New() string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("tree %s", c.TreeOID))
	if c.Parent != "" {
		lines = append(lines, fmt.Sprintf("parent %s", c.Parent))
	}
	lines = append(lines, fmt.Sprintf("author %s", c.Author))
	lines = append(lines, fmt.Sprintf("comitter %s", c.Author))
	lines = append(lines, "")
	lines = append(lines, c.Message)
	return strings.Join(lines, "\n")
}

func (c Commit) Type() string {
	return "commit"
}

func ReadStdinMsg() string {
	msg, _ := io.ReadAll(os.Stdin)
	return string(msg)
}

type Entries struct {
	Path string
	OID  []byte
}

func StoreTreeObject(treeEntry bytes.Buffer) (string, error) {
	treePrefix := fmt.Sprintf(`tree %d`, treeEntry.Len())
	treeSHA := getHash(treePrefix, treeEntry.String())
	hexTreeSha := hex.EncodeToString(treeSHA)
	// fmt.Printf("Tree: %s", hexTreeSha)
	tree := getCompressBuf([]byte(treePrefix), treeEntry.Bytes())
	folderPath := filepath.Join(DBPATH, hexTreeSha[:2])
	permPath := filepath.Join(DBPATH, hexTreeSha[:2], hexTreeSha[2:])
	err := StoreObject(tree, treePrefix, folderPath, permPath)
	if err != nil {
		return "", err
	}
	return hexTreeSha, nil
}

func StoreBlobObject(blobData []byte) ([]byte, error) {
	blobPrefix := fmt.Sprintf(`blob %d`, len(blobData))

	// getting the SHA-1
	blobSHA := getHash(blobPrefix, string(blobData)) // []byte
	blob := getCompressBuf([]byte(blobPrefix), blobData)
	hexBlobSha := hex.EncodeToString(blobSHA)
	folderPath := filepath.Join(DBPATH, hexBlobSha[:2])
	permPath := filepath.Join(DBPATH, hexBlobSha[:2], hexBlobSha[2:])
	err := StoreObject(blob, blobPrefix, folderPath, permPath)
	if err != nil {
		return nil, err
	}

	return blobSHA, nil
}

func StoreCommitObject(commitData string) (string, error) {
	commitPrefix := fmt.Sprintf(`commit %d`, len(commitData))
	commitHash := getHash(commitPrefix, commitData)
	commit := getCompressBuf([]byte(commitPrefix), []byte(commitData))
	hexCommitHash := hex.EncodeToString(commitHash)
	folderPath := filepath.Join(DBPATH, hexCommitHash[:2])
	permPath := filepath.Join(DBPATH, hexCommitHash[:2], hexCommitHash[2:])
	err := StoreObject(commit, commitPrefix, folderPath, permPath)
	if err != nil {
		return "", err
	}

	return hexCommitHash, nil
}

func CreateTreeEntry(entries []Entries) bytes.Buffer {
	var buf bytes.Buffer
	for _, entry := range entries {
		input := fmt.Sprintf("100644 %s", entry.Path)
		buf.WriteString(input)
		buf.WriteByte(0)
		buf.Write(entry.OID)
		// buf.WriteString("\n")
	}
	return buf
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
		return fmt.Errorf("Err creating temp file: %s", err)
	}
	defer tf.Close()

	// Write to temp file
	_, err = tf.Write(data.Bytes())
	if err != nil {
		return fmt.Errorf("Err writing to temp file: %s", err)
	}

	// rename the file
	os.Rename(tempPath, PermPath)
	return nil
}
