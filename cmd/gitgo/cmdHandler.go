package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/Vikuuu/gitgo/internal/utilites"
)

func cmdInitHandler(initPath string) error {
	gitFolders := [2]string{"objects", "refs"}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if len(initPath) > 0 {
		wd = filepath.Join(wd, initPath)
	}
	gitPath := filepath.Join(wd, ".gitgo")

	for _, folder := range gitFolders {
		err = os.MkdirAll(filepath.Join(gitPath, folder), 0755)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stdout, "Initialized empty Gitgo repository in %s", gitPath)
	return nil
}

func cmdCommitHandler() error {
	rootPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Error getting pwd: %s", err)
	}
	gitPath := filepath.Join(rootPath, ".gitgo")
	dbPath := filepath.Join(gitPath, "objects")

	// Get all the files in the working directory
	workFiles, err := os.ReadDir(rootPath)
	if err != nil {
		return fmt.Errorf("Error reading Dir: %s", err)
	}

	for _, file := range workFiles {
		if file.Name() == ".gitgo" {
			continue
		}

		data, err := os.ReadFile(file.Name())
		if err != nil {
			return fmt.Errorf("Error reading file: %s\n%s", file.Name(), err)
		}
		blobPrefix := fmt.Sprintf(`blob %d\0`, len(data))
		// getting the SHA-1
		h := sha1.New()
		io.WriteString(h, blobPrefix)
		io.WriteString(h, string(data))
		// blobSHA := base64.URLEncoding.EncodeToString(h.Sum(nil))
		blobSHA := hex.EncodeToString(h.Sum(nil))

		var blob bytes.Buffer
		w := zlib.NewWriter(&blob)
		w.Write(slices.Concat([]byte(blobPrefix), data))
		w.Close()

		err = os.MkdirAll(filepath.Join(dbPath, string(blobSHA[:2])), 0755)
		if err != nil {
			return fmt.Errorf("Error creating Dir: %s", err)
		}

		// Create a temp file for writing
		tName := utilites.GenerateGitTempFileName(".temp-obj-")
		tempName := filepath.Join(dbPath, string(blobSHA[:2]), tName)
		tf, err := os.OpenFile(
			tempName,
			os.O_RDWR|os.O_CREATE|os.O_EXCL,
			0644,
		)
		defer tf.Close()
		if err != nil {
			return fmt.Errorf("Error creating tempFile: %s", err)
		}

		// Write to the temp file
		_, err = tf.Write(blob.Bytes())
		if err != nil {
			return fmt.Errorf("Error writing to temp file: %s", err)
		}

		// Rename the file
		permName := filepath.Join(dbPath, string(blobSHA[:2]), string(blobSHA[2:]))
		os.Rename(tempName, permName)
	}

	return nil
}
