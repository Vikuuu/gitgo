package gitgo

import (
	"log"
	"os"
	"path/filepath"
)

var (
	ROOTPATH string
	GITPATH  string
	DBPATH   string
)

func InitGlobals() {
	var err error
	ROOTPATH, err = os.Getwd()
	if err != nil {
		log.Fatalf("Err getting pwd: %s", err)
		os.Exit(1)
	}
	GITPATH = filepath.Join(ROOTPATH, ".gitgo")
	DBPATH = filepath.Join(GITPATH, "objects")
}

var GitFolders = []string{"objects", "refs"}
