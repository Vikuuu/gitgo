package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Vikuuu/gitgo"
)

func cmdInitHandler(initPath string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if len(initPath) > 0 {
		wd = filepath.Join(wd, initPath)
	}
	gitPath := filepath.Join(wd, ".gitgo")

	for _, folder := range gitgo.GitFolders {
		err = os.MkdirAll(filepath.Join(gitPath, folder), 0755)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stdout, "Initialized empty Gitgo repository in %s", gitPath)
	return nil
}

func cmdCommitHandler(commit string) error {
	// Get all the files in the working directory
	allFiles, err := os.ReadDir(gitgo.ROOTPATH)
	if err != nil {
		return fmt.Errorf("Error reading Dir: %s", err)
	}
	workFiles := gitgo.RemoveIgnoreFiles(
		allFiles,
		gitgo.GITGO_IGNORE,
	) // Remove the files or Dir that are in ignore.

	var entries []gitgo.Entries
	for _, file := range workFiles {
		if file.IsDir() {
			continue
		}
		data, err := os.ReadFile(file.Name())
		if err != nil {
			return fmt.Errorf("Error reading file: %s\n%s", file.Name(), err)
		}

		blobSHA, err := gitgo.StoreBlobObject(data)

		entries = append(entries, gitgo.Entries{
			Path: file.Name(),
			OID:  blobSHA,
		})
	}

	// create the tree entry.
	treeEntry := gitgo.CreateTreeEntry(entries)
	// store the tree data in the .gitgo/objects
	treeHash, err := gitgo.StoreTreeObject(treeEntry)
	if err != nil {
		return err
	}

	name := os.Getenv("GITGO_AUTHOR_NAME")
	email := os.Getenv("GITGO_AUTHOR_EMAIL")
	authorData := gitgo.Author{
		Name:      name,
		Email:     email,
		Timestamp: time.Now(),
	}
	author := authorData.New()
	message := gitgo.ReadStdinMsg()

	commitData := gitgo.Commit{
		TreeOID: treeHash,
		Author:  author,
		Message: message,
	}.New()
	cHash, err := gitgo.StoreCommitObject(commitData)
	if err != nil {
		return err
	}

	HeadFile, err := os.OpenFile(
		filepath.Join(gitgo.GITPATH, "HEAD"),
		os.O_WRONLY|os.O_CREATE,
		0644,
	)
	if err != nil {
		return fmt.Errorf("Err creating HEAD file: %s", err)
	}
	defer HeadFile.Close()

	_, err = HeadFile.WriteString(cHash)
	if err != nil {
		return fmt.Errorf("Err writing to HEAD file: %s", err)
	}
	fmt.Printf("root-commit  %s", cHash)

	return nil
}
