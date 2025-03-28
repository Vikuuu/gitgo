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

	fmt.Fprintf(os.Stdout, "Initialized empty Gitgo repository in %s\n", gitPath)
	return nil
}

func cmdCommitHandler(_ string) error {
	rootPath := gitgo.ROOTPATH
	// storing all the blobs first
	entries, err := gitgo.StoreOnDisk(rootPath)
	if err != nil {
		return err
	}
	// build merkel tree, and store all the subdirectories
	// tree file
	tree := gitgo.BuildTree(entries)
	e, err := gitgo.TraverseTree(tree)
	if err != nil {
		return err
	}
	// now store the root tree
	rootTree := gitgo.TreeBlob{Data: gitgo.CreateTreeEntry(e)}.Init()
	treeHash, err := rootTree.Store()
	if err != nil {
		return err
	}

	// storing commit object
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
	fmt.Fprintf(os.Stdout, "%s %s %s\n", is_root, cHash, gitgo.FirstLine(message))

	return nil
}

func cmdCatFileHandler(hash string) error {
	folderPath, filePath := hash[:2], hash[2:]
	b, err := os.ReadFile(filepath.Join(gitgo.ROOTPATH, ".gitgo/objects", folderPath, filePath))
	if err != nil {
		return err
	}

	data, err := gitgo.GetDecompress(b)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, string(data))
	return nil
}
