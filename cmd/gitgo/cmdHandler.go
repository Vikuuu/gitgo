package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

	fmt.Fprintf(os.Stdout, "Initialized empty Gitgo repository in %s\n", gitPath)
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

		blob := gitgo.BlobInitialize(data)

		fileMode, err := gitgo.FileMode(file)
		fmt.Printf("File: %s Mode: %o", file.Name(), fileMode)
		if err != nil {
			return err
		}

		blobHash, err := blob.Store()
		// blobSHA, err := gitgo.StoreBlobObject(data)

		entry := gitgo.Entries{
			Path: file.Name(),
			OID:  blobHash,
			Stat: strconv.FormatUint(uint64(fileMode), 8),
		}
		// entry.Mode(fileMode)
		entries = append(entries, entry)
	}

	tree := gitgo.TreeInitialize(gitgo.CreateTreeEntry(entries))
	// create the tree entry.
	// treeEntry :=
	// store the tree data in the .gitgo/objects
	treeHash, err := tree.Store()
	if err != nil {
		return err
	}

	name := os.Getenv("GITGO_AUTHOR_NAME")
	email := os.Getenv("GITGO_AUTHOR_EMAIL")
	author := gitgo.Author{
		Name:      name,
		Email:     email,
		Timestamp: time.Now(),
	}.New()
	message := gitgo.ReadStdinMsg()
	refs := gitgo.RefInitialize(gitgo.GITPATH)
	parent := refs.Read_head()
	// now check this is first commit or not
	is_root := ""
	if parent == "" {
		is_root = "(root-commit) "
	}

	commitData := gitgo.Commit{
		Parent:  parent,
		TreeOID: treeHash,
		Author:  author,
		Message: message,
	}.New()
	cHash, err := commitData.Store()
	if err != nil {
		return err
	}
	refs.UpdateHead([]byte(cHash))

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

	fmt.Fprintf(os.Stdout, "%s %s %s\n", is_root, cHash, gitgo.FirstLine(message))

	return nil
}
