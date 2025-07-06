package commands

import (
	"fmt"

	"app/internal/scraper/cli"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate sitemap and extraction patterns",
	Long: `Validate that a sitemap is accessible and contains URLs that match extraction patterns.

Examples:
  scraper-cli validate --url https://example.com --sitemap https://example.com/sitemap.xml
  scraper-cli validate --id 1
  scraper-cli validate --url https://example.com --auto-discover`,
	RunE: runValidateTarget,
}

func init() {
	validateCmd.Flags().StringP("url", "u", "", "Website URL to validate")
	validateCmd.Flags().StringP("sitemap", "s", "", "Sitemap URL to validate")
	validateCmd.Flags().Int64P("id", "i", 0, "Target ID to validate")
	validateCmd.Flags().BoolP("auto-discover", "a", false, "Auto-discover sitemap")
	validateCmd.Flags().IntP("limit", "l", 10, "Limit number of URLs to preview")
}

func runValidateTarget(cmd *cobra.Command, args []string) error {
	websiteURL, _ := cmd.Flags().GetString("url")
	sitemapURL, _ := cmd.Flags().GetString("sitemap")
	targetID, _ := cmd.Flags().GetInt64("id")
	autoDiscover, _ := cmd.Flags().GetBool("auto-discover")
	limit, _ := cmd.Flags().GetInt("limit")

	manager, err := cli.NewTargetManager()
	if err != nil {
		return fmt.Errorf("failed to initialize target manager: %w", err)
	}
	defer func() {
		if err := manager.Close(); err != nil {
			fmt.Printf("failed to close manager: %v\n", err)
		}
	}()

	if targetID > 0 {
		// Validate existing target by showing its details
		return manager.ShowTarget(targetID)
	}

	if websiteURL == "" {
		return fmt.Errorf("either --id or --url must be specified")
	}

	// Validate new target configuration
	fmt.Printf("Validating configuration for: %s\n", websiteURL)

	if sitemapURL == "" && autoDiscover {
		fmt.Println("Would auto-discover sitemap...")
		// TODO: Implement auto-discovery validation
	}

	if sitemapURL != "" {
		fmt.Printf("Validating sitemap: %s\n", sitemapURL)
		// TODO: Implement sitemap validation
	}

	fmt.Printf("Preview limit: %d URLs\n", limit)
	return nil
}
