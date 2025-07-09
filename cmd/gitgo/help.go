package main

import "fmt"

func handlerHelp(c *commands, _ command) {
	fmt.Print("Welcome, to GITGO a version control system\n\n")
	fmt.Print("Usage: gitgo <commands> [...Args]\n")
	fmt.Print("Avaiable commands: \n\n")

	for name, info := range c.registeredCmds {
		fmt.Printf(" %-10s %-30s %s\n", name, info.usage, info.description)
	}
}
