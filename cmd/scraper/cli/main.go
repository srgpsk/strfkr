package main

import (
	"app/cmd/scraper/cli/commands"
	"fmt"
	"os"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
