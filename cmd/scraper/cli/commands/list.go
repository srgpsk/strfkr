package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"app/internal/scraper/cli"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all scraping targets",
	Long: `List all configured scraping targets.

Examples:
  scraper-cli list
  scraper-cli list --active
  scraper-cli list --format json`,
	RunE: runListTargets,
}

func init() {
	listCmd.Flags().BoolP("active", "a", false, "Show only active targets")
	listCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
}

func runListTargets(cmd *cobra.Command, args []string) error {
	activeOnly, _ := cmd.Flags().GetBool("active")
	format, _ := cmd.Flags().GetString("format")

	manager, err := cli.NewTargetManager()
	if err != nil {
		return fmt.Errorf("failed to initialize target manager: %w", err)
	}
	defer func() {
		err := manager.Close()
		if err != nil {
			fmt.Printf("failed to close manager: %v\n", err)
		}
	}()

	targets, err := manager.ListTargets(activeOnly)
	if err != nil {
		return fmt.Errorf("failed to list targets: %w", err)
	}

	if len(targets) == 0 {
		fmt.Println("No targets found.")
		return nil
	}

	switch format {
	case "json":
		return manager.PrintTargetsJSON(targets)
	case "table":
		return printTargetsTable(targets)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func printTargetsTable(targets []cli.TargetInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tURL\tSitemap\tActive\tLast Updated"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "---\t---\t---\t---\t---"); err != nil {
		return err
	}

	for _, target := range targets {
		active := "Yes"
		if !target.Active {
			active = "No"
		}

		sitemap := target.SitemapURL
		if sitemap == "" {
			sitemap = "Auto-discover"
		}

		if _, err := fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			target.ID,
			target.WebsiteURL,
			sitemap,
			active,
			target.UpdatedAt.Format("2006-01-02 15:04"),
		); err != nil {
			return err
		}
	}

	return w.Flush()
}
