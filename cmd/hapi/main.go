package main

import (
	"fmt"
	"os"

	"github.com/ibagur/cli-hdx/internal/cli"
	"github.com/ibagur/cli-hdx/internal/output"
)

func main() {
	root := cli.NewRootCommand(cli.Options{Stdout: os.Stdout, Stderr: os.Stderr})
	if err := root.Execute(); err != nil {
		if writeErr := output.WriteJSON(os.Stdout, cli.ErrorEnvelope(err)); writeErr != nil {
			fmt.Fprintln(os.Stderr, writeErr)
		}
		os.Exit(cli.ExitCode(err))
	}
}
