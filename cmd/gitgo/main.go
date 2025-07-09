package main

import (
	"fmt"
	"os"

	"github.com/Vikuuu/gitgo"
)

func main() {
	cmds := &commands{
		registeredCmds: make(map[string]commandInfo),
	}
	cmds.initializeCommands()

	if len(os.Args) < 2 {
		fmt.Println("Usage: gitgo <command> [args...]")
	}

	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]

	env := GetGitgoVar()

	cmd := command{
		name:   cmdName,
		args:   cmdArgs,
		env:    env,
		pwd:    os.Getenv("PWD"),
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
		repo:   gitgo.NewRepository(os.Getenv("PWD")),
	}

	exitCode, err := cmds.run(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(exitCode)
	}
}

func (c *commands) initializeCommands() {
	c.register("help", func(cmd command) int {
		handlerHelp(c, cmd)
		return 0
	}, "help", "Displays all available commands and their usage")

	c.register("commit", cmdCommitHandler, "commit", "Commits the files in staging area")
	c.register("init", cmdInitHandler, "init", "Initialize gitgo repository in the directory.")
	c.register("add", cmdAddHandler, "add", "Add files to staging area.")
	c.register("cat-file", cmdCatFileHandler, "cat-file", "Get the blob content.")
}

func GetGitgoVar() map[string]string {
	env := make(map[string]string)
	env["name"] = os.Getenv("GITGO_AUTHOR_NAME")
	env["email"] = os.Getenv("GITGO_AUTHOR_EMAIL")

	return env
}
