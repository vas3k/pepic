package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/vas3k/pepic/pepic/config"
)

func Execute() error {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Usage: pepic serve [--config path]")
		return nil
	}
	commandFirst := args[0] == "serve"
	if commandFirst {
		args = args[1:]
	}
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	filename := flags.String("config", "/etc/pepic/config.yml", "configuration file")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if commandFirst && flags.NArg() != 0 || !commandFirst && (flags.NArg() != 1 || flags.Arg(0) != "serve") {
		return fmt.Errorf("expected serve command, got %v", flags.Args())
	}

	if err := config.Load(*filename); err != nil {
		return err
	}
	return serve()
}
