package commands

import (
	"fmt"

	"app/internal/scraper/cli"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the web scraper",
	Long: `Run the web scraper to process targets and download pages.

Examples:
  scraper-cli run
  scraper-cli run --target-id 1
  scraper-cli run --progress --verbose
  scraper-cli run --dry-run`,
	RunE: runScraper,
}

func init() {
	runCmd.Flags().Int64P("target-id", "t", 0, "Run for specific target ID (0 = all targets)")
	runCmd.Flags().BoolP("progress", "p", true, "Show progress bar")
	runCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	runCmd.Flags().BoolP("dry-run", "d", false, "Dry run (no actual crawling)")
	runCmd.Flags().IntP("workers", "w", 3, "Number of worker threads")
	runCmd.Flags().IntP("batch-size", "b", 10, "Batch size for URL processing")
}

func runScraper(cmd *cobra.Command, args []string) error {
	targetID, _ := cmd.Flags().GetInt64("target-id")
	progress, _ := cmd.Flags().GetBool("progress")
	verbose, _ := cmd.Flags().GetBool("verbose")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	workers, _ := cmd.Flags().GetInt("workers")
	batchSize, _ := cmd.Flags().GetInt("batch-size")

	if workers < 1 || workers > 20 {
		return fmt.Errorf("workers must be between 1 and 20")
	}

	if batchSize < 1 || batchSize > 100 {
		return fmt.Errorf("batch-size must be between 1 and 100")
	}

	runner, err := cli.NewScraperRunner(workers, batchSize)
	if err != nil {
		return fmt.Errorf("failed to initialize scraper runner: %w", err)
	}
	defer func() {
		err := runner.Close()
		if err != nil {
			fmt.Printf("failed to close runner: %v\n", err)
		}
	}()

	return runner.Run(targetID, progress, verbose, dryRun)
}
