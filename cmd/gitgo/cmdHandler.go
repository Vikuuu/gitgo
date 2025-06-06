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
	// rootPath := gitgo.ROOTPATH
	// storing all the blobs first
	// entries, err := gitgo.StoreOnDisk(rootPath)
	// if err != nil {
	// 	return err
	// }
	// build merkel tree, and store all the subdirectories
	// tree file
	index := gitgo.NewIndex()
	index.Load()
	tree := gitgo.BuildTree(index.Entries())
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

	// clear the file after the commit is done
	err = os.Remove(filepath.Join(gitgo.GITPATH, "index"))
	if err != nil {
		return err
	}

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

func cmdAddHandler(args []string) error {
	// index := gitgo.NewIndex()
	_, index := gitgo.IndexHoldForUpdate()
	var filePaths []string

	// add all the paths to a slice first
	for _, path := range args {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		expandPaths, err := gitgo.ListFiles(absPath)
		if err != nil {
			index.Release()
			return err
		}
		filePaths = append(filePaths, expandPaths...)
	}

	for _, p := range filePaths {
		ap, err := filepath.Abs(p)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(ap)
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("%w '%s'\nFatal: adding files failed", os.ErrPermission, p)
			}
			return err
		}
		stat, err := os.Stat(ap)
		if err != nil {
			return err
		}

		blob := gitgo.Blob{Data: data}.Init()
		hash, err := blob.Store()
		if err != nil {
			return err
		}

		index.Add(p, hash, stat)
	}

	res, err := index.WriteUpdate()
	if err != nil {
		return err
	}

	if res {
		fmt.Println("Written data to Index file")
	}

	return nil
}
