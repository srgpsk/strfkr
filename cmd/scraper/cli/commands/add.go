package commands

import (
	"fmt"
	"net/url"
	"strings"

	"app/internal/scraper/cli"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new scraping target",
	Long: `Add a new website target for scraping.

Examples:
  scraper-cli add --url https://example.com --sitemap https://example.com/sitemap.xml
  scraper-cli add --url https://example.com --auto-discover
  scraper-cli add --url https://example.com --auto-discover --no-validate`,
	RunE: runAddTarget,
}

func init() {
	addCmd.Flags().StringP("url", "u", "", "Website URL (required)")
	addCmd.Flags().StringP("sitemap", "s", "", "Sitemap URL (optional)")
	addCmd.Flags().BoolP("auto-discover", "a", false, "Auto-discover sitemap")
	addCmd.Flags().BoolP("validate", "v", true, "Validate sitemap before adding")
	addCmd.Flags().StringP("user-agent", "", "ScraperBot/1.0", "User agent for requests")
	if err := addCmd.MarkFlagRequired("url"); err != nil {
		panic(err)
	}
}

func runAddTarget(cmd *cobra.Command, args []string) error {
	websiteURL, _ := cmd.Flags().GetString("url")
	sitemapURL, _ := cmd.Flags().GetString("sitemap")
	autoDiscover, _ := cmd.Flags().GetBool("auto-discover")
	validate, _ := cmd.Flags().GetBool("validate")
	userAgent, _ := cmd.Flags().GetString("user-agent")

	// Ensure URL has proper scheme
	if !strings.HasPrefix(websiteURL, "http://") && !strings.HasPrefix(websiteURL, "https://") {
		websiteURL = "https://" + websiteURL
	}

	// Validate URL format
	if _, err := url.ParseRequestURI(websiteURL); err != nil {
		return fmt.Errorf("invalid website URL: %w", err)
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

	return manager.AddTarget(websiteURL, sitemapURL, userAgent, autoDiscover, validate)
}
