package commands

import (
	"fmt"

	"app/internal/scraper/cli"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a scraping target",
	Long: `Remove a scraping target from the database.

Examples:
  scraper-cli remove --id 1
  scraper-cli remove --id 1 --force`,
	RunE: runRemoveTarget,
}

func init() {
	removeCmd.Flags().Int64P("id", "i", 0, "Target ID to remove (required)")
	removeCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")
	if err := removeCmd.MarkFlagRequired("id"); err != nil {
		panic(err)
	}
}

func runRemoveTarget(cmd *cobra.Command, args []string) error {
	targetID, _ := cmd.Flags().GetInt64("id")
	force, _ := cmd.Flags().GetBool("force")

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

	return manager.RemoveTarget(targetID, force)
}
