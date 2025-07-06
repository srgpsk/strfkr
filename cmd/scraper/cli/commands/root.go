package commands

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "scraper-cli",
	Short: "Web scraper CLI tool",
	Long:  "Command line interface for managing web scraping targets and operations",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(showCmd)
}
