package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Vikuuu/gitgo"
)

func main() {
	init := flag.String("init", "", "Create .gitgo files in directory")
	commit := flag.String("commit", "", "Commit file")
	// flag.Bool("help", false, "Help Message")
	flag.Parse()

	if len(os.Args) < 2 {
		flag.Usage()
		return
	}

	// Initialize global vars
	gitgo.InitGlobals()

	switch os.Args[1] {
	case "init":
		err := cmdInitHandler(*init)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s", err)
			os.Exit(1)
		}
	case "commit":
		err := cmdCommitHandler(*commit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s", err)
			os.Exit(1)
		}
	default:
		flag.Usage()
	}
}
