package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Vikuuu/gitgo"
)

var gitgoFolders []string

func cmdInitHandler(cmd command) int {
	gitgoFolders = []string{"objects", "refs"}

	initPath := cmd.repo.Path
	var arg string = ""
	if len(cmd.args) > 0 {
		arg = cmd.args[0]
	}

	if arg != "" {
		initPath = filepath.Join(initPath, arg)
	}
	gitPath := filepath.Join(initPath, ".gitgo")

	for _, folder := range gitgoFolders {
		err := os.MkdirAll(filepath.Join(gitPath, folder), 0755)
		if err != nil {
			fmt.Fprintf(cmd.stderr, "error: %v\n", err)
			return 1
		}
	}

	fmt.Fprintf(cmd.stdout, "Initialized empty Gitgo repository in %s\n", gitPath)
	return 0
}

func cmdCommitHandler(cmd command) int {
	// The command `commit` reads the index file and then
	// creates and merkel tree, and store all the sub-trees on the way
	// The blob files are already being stored during the
	// `add` command
	index := gitgo.NewIndex(cmd.repo.Path, cmd.repo.GitPath)
	index.Load()
	entries := index.Entries()
	tree := gitgo.BuildTree(entries)
	e, err := gitgo.TraverseTree(tree, cmd.repo.Database)
	if err != nil {
		fmt.Fprintf(cmd.stderr, "error: %v\n", err)
	}
	// now store the root tree
	rootTree := gitgo.TreeBlob{
		Data:   gitgo.CreateTreeEntry(e),
		DBPath: cmd.repo.Database,
	}.Init()
	treeHash, err := rootTree.Store()
	if err != nil {
		fmt.Fprintf(cmd.stderr, "error: %v\n", err)
		return 1
	}

	// storing commit object
	author := gitgo.Author{
		Name:      cmd.env["name"],
		Email:     cmd.env["email"],
		Timestamp: time.Now(),
	}.New()
	message := gitgo.ReadStdinMsg(cmd.stdin)
	refs := gitgo.RefInitialize(cmd.repo.Refs)
	parent := refs.ReadHead()

	is_root := ""
	if parent == "" {
		is_root = "(root-commit) "
	}

	commitData := gitgo.Commit{
		Parent:  parent,
		TreeOID: treeHash,
		Author:  author,
		Message: message,
		DBPath:  cmd.repo.Database,
	}.New()
	cHash, err := commitData.Store()
	if err != nil {
		fmt.Fprintf(cmd.stderr, "error: %v\n", err)
		return 1
	}
	refs.UpdateHead([]byte(cHash))
	fmt.Fprintf(cmd.stdout, "%s %s %s\n", is_root, cHash, gitgo.FirstLine(message))

	// clear the file after the commit is done
	err = os.Remove(cmd.repo.Index)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}

		fmt.Fprintf(cmd.stderr, "error: %v\n", err)
		return 1
	}

	return 0
}

func cmdCatFileHandler(cmd command) int {
	if len(cmd.args) != 1 {
		fmt.Fprintln(cmd.stderr, "error: no file hash provided")
		return 1
	}
	hash := cmd.args[0]

	folderPath, filePath := hash[:2], hash[2:]
	b, err := os.ReadFile(filepath.Join(cmd.repo.Database, folderPath, filePath))
	if err != nil {
		fmt.Fprintf(cmd.stderr, "error: %v\n", err)
		return 1
	}

	data, err := gitgo.GetDecompress(b)
	if err != nil {
		fmt.Fprintf(cmd.stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintln(cmd.stdout, string(data))
	return 0
}

func cmdAddHandler(cmd command) int {
	// gitgo.InitGlobals()
	// index := gitgo.NewIndex()
	_, index, err := gitgo.IndexHoldForUpdate(cmd.repo.Path, cmd.repo.GitPath)
	if err != nil {
		fmt.Fprintf(cmd.stderr, `
Fatal: %v

Another gitgo process seems to be running in this repository.
Please make sure all processes are terminated then try again.
If it still fails, a gitgo process may have crashed in this
repository earlier: remove the file manually to continue.`, err)
		return 1
	}
	var filePaths []string

	// add all the paths to a slice first
	for _, path := range cmd.args {
		absPath := filepath.Join(cmd.pwd, path)
		expandPaths, err := gitgo.ListFiles(absPath, cmd.repo.Path)
		if err != nil {
			index.Release()
			fmt.Fprintf(cmd.stderr, "error: %v\n", err)
			return 1
		}
		filePaths = append(filePaths, expandPaths...)
	}

	for _, p := range filePaths {
		ap := filepath.Join(cmd.pwd, p)
		data, err := os.ReadFile(ap)
		if err != nil {
			if os.IsPermission(err) {
				fmt.Fprintf(cmd.stderr, "%v '%s'\nfatal: adding files failed", os.ErrPermission, p)
				return 1
			}
			fmt.Fprintf(cmd.stderr, "error: %v\n", err)
			return 1
		}
		stat, err := os.Stat(ap)
		if err != nil {
			fmt.Fprintf(cmd.stderr, "error: %v\n", err)
			return 1
		}

		blob := gitgo.Blob{
			Data:   data,
			DBPath: cmd.repo.Database,
		}.Init()
		hash, err := blob.Store()
		if err != nil {
			fmt.Fprintf(cmd.stderr, "error: %v\n", err)
			return 1
		}

		index.Add(p, hash, stat)
	}

	res, err := index.WriteUpdate()
	if err != nil {
		fmt.Fprintf(cmd.stderr, "error: %v\n", err)
		return 1
	}

	if res {
		fmt.Fprintln(cmd.stdout, "Written data to Index file")
	}

	return 0
}
