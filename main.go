package main

import (
	"errors"
	"fmt"
	"os"
)

type Runner interface {
	Name() string
	Run(args []string) error
}

func main() {
	if err := executeSubcommand(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func executeSubcommand(args []string) error {
	cmds := []Runner{
		NewCreateCommand(),
		NewCopyCommand(),
		NewListCommand(),
	}
	if len(args) < 1 {
		return errors.New("specify a subcommand: 'create', 'ls', or 'cp'")
	}
	subcommand := os.Args[1]
	for _, cmd := range cmds {
		if cmd.Name() == subcommand {
			return cmd.Run(os.Args[2:])
		}
	}
	return fmt.Errorf("unknown subcommand: %s", subcommand)
}
