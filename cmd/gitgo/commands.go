package main

import (
	"errors"
	"os"

	"github.com/Vikuuu/gitgo"
)

type command struct {
	name   string
	pwd    string
	env    map[string]string
	args   []string
	stdin  *os.File
	stdout *os.File
	stderr *os.File
	repo   gitgo.Repository
}

type commandInfo struct {
	handler     func(command) int
	usage       string
	description string
}

type commands struct {
	registeredCmds map[string]commandInfo
}

func (c *commands) register(name string, handler func(command) int, usage, description string) {
	c.registeredCmds[name] = commandInfo{
		handler:     handler,
		usage:       usage,
		description: description,
	}
}

func (c *commands) run(cmd command) (int, error) {
	ci, ok := c.registeredCmds[cmd.name]
	if !ok {
		return 0, errors.New("command not found")
	}
	exitCode := ci.handler(cmd)
	return exitCode, nil
}
