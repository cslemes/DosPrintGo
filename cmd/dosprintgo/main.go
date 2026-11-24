package main

import (
	"os"

	"dosprintgo/cmd/dosprintgo/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
