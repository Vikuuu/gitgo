package gitgo

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// randomOID returns a 40-char hex string, like SecureRandom.hex(20) in Ruby.
func randomOID() string {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		panic("failed to read random bytes: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func thisFileStat(t *testing.T) os.FileInfo {
	// locate this test file so we can grab its FileInfo
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not get current test filename")
	}
	fi, err := os.Stat(thisFile)
	if err != nil {
		t.Fatalf("stat test file: %v", err)
	}
	return fi
}

func TestAddSingleFile(t *testing.T) {
	// create a temporary directory for this test
	tmpDir := t.TempDir()
	rootPath := tmpDir
	gitPath := filepath.Join(rootPath, ".gitgo")
	idx := NewIndex(gitPath)

	oid := randomOID()
	// if err := idx.Add("alice.txt", oid, fi); err != nil {
	// 	t.Fatalf("Add failed: %v", err)
	// }
	idx.Add("alice.txt", oid, thisFileStat(t))

	entries := idx.Entries()
	// if err != nil {
	// 	t.Fatalf("Entries failed: %v", err)
	// }

	// collect just the paths
	var got []string
	for _, e := range entries {
		got = append(got, e.Path)
	}

	expected := []string{"alice.txt"}
	assert.Equal(t, expected, got)
}

func TestReplaceFileWithDir(t *testing.T) {
	index := NewIndex("/tmp")

	index.Add("alice.txt", randomOID(), thisFileStat(t))
	index.Add("bob.txt", randomOID(), thisFileStat(t))

	index.Add("alice.txt/nested.txt", randomOID(), thisFileStat(t))

	expected := []string{"alice.txt/nested.txt", "bob.txt"}
	var got []string
	it := index.keys.Iterator()
	for it.Next() {
		got = append(got, it.Key())
	}

	assert.Equal(t, expected, got)
}

func TestReplaceDirWithFile(t *testing.T) {
	index := NewIndex("/tmp")

	index.Add("alice.txt", randomOID(), thisFileStat(t))
	index.Add("nested/bob.txt", randomOID(), thisFileStat(t))
	index.Add("nested", randomOID(), thisFileStat(t))

	expected := []string{"alice.txt", "nested"}
	var got []string
	it := index.keys.Iterator()
	for it.Next() {
		got = append(got, it.Key())
	}

	assert.Equal(t, expected, got)
}

func TestReplaceNestedDirWithFile(t *testing.T) {
	index := NewIndex("/tmp")

	index.Add("alice.txt", randomOID(), thisFileStat(t))
	index.Add("nested/bob.txt", randomOID(), thisFileStat(t))
	index.Add("nested/inner/claire.txt", randomOID(), thisFileStat(t))
	index.Add("nested", randomOID(), thisFileStat(t))

	expected := []string{"alice.txt", "nested"}
	var got []string
	it := index.keys.Iterator()
	for it.Next() {
		got = append(got, it.Key())
	}

	assert.Equal(t, expected, got)
}
