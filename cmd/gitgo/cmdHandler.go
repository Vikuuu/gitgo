package main

import (
	"fmt"
	"os"
	"path/filepath"
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
