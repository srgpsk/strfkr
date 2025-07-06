package commands

import (
	"fmt"

	"app/internal/scraper/cli"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show details of a scraping target",
	Long: `Show detailed information about a specific scraping target.

Examples:
  scraper-cli show --id 1`,
	RunE: runShowTarget,
}

func init() {
	showCmd.Flags().Int64P("id", "i", 0, "Target ID to show (required)")
	if err := showCmd.MarkFlagRequired("id"); err != nil {
		panic(err)
	}
}

func runShowTarget(cmd *cobra.Command, args []string) error {
	targetID, _ := cmd.Flags().GetInt64("id")

	if targetID == 0 {
		return fmt.Errorf("target ID must be specified")
	}

	manager, err := cli.NewTargetManager()
	if err != nil {
		return fmt.Errorf("failed to initialize target manager: %w", err)
	}
	defer func() {
		if err := manager.Close(); err != nil {
			fmt.Printf("failed to close manager: %v\n", err)
		}
	}()

	return manager.ShowTarget(targetID)
}
