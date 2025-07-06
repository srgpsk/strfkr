package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// PatternConfig holds regex patterns for a target
type PatternConfig struct {
	SitemapPatterns []string `json:"sitemap_patterns"`
	URLPatterns     []string `json:"url_patterns"`
}

// CompilePatterns compiles regex patterns
func CompilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
		compiled = append(compiled, regex)
	}
	return compiled, nil
}

// DefaultPatterns provides fallback patterns when none are configured
func DefaultPatterns() PatternConfig {
	return PatternConfig{
		SitemapPatterns: []string{
			`sitemap-\d+\.xml$`,          // sitemap-1.xml, sitemap-2.xml
			`post-sitemap[^/]*\.xml$`,    // post-sitemap.xml, post-sitemap-1.xml
			`posts?[-_]sitemap.*\.xml$`,  // posts-sitemap.xml variations
			`content[-_]sitemap.*\.xml$`, // content-sitemap.xml variations
		},
		URLPatterns: []string{
			`/[^/]+/$`, // Simple path patterns like /quote-text/
		},
	}
}

// ToJSON converts patterns to JSON string for database storage
func (p PatternConfig) ToJSON() (string, error) {
	data, err := json.Marshal(p)
	return string(data), err
}

// FromJSON parses patterns from JSON string
func FromJSON(jsonStr string) (PatternConfig, error) {
	var config PatternConfig
	err := json.Unmarshal([]byte(jsonStr), &config)
	return config, err
}

// ExtractDomain extracts domain from URL for pattern lookup
func ExtractDomain(websiteURL string) string {
	parsed, err := url.Parse(websiteURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}
