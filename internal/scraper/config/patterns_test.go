package config

import (
	"testing"
)

func TestDefaultPatterns(t *testing.T) {
	patterns := DefaultPatterns()

	if len(patterns.SitemapPatterns) == 0 {
		t.Error("Expected default sitemap patterns, got none")
	}

	if len(patterns.URLPatterns) == 0 {
		t.Error("Expected default URL patterns, got none")
	}

	// Test that patterns are valid regex
	_, err := CompilePatterns(patterns.SitemapPatterns)
	if err != nil {
		t.Errorf("Default sitemap patterns invalid: %v", err)
	}

	_, err = CompilePatterns(patterns.URLPatterns)
	if err != nil {
		t.Errorf("Default URL patterns invalid: %v", err)
	}
}

func TestCompilePatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		wantErr  bool
	}{
		{
			name:     "valid patterns",
			patterns: []string{`sitemap-\d+\.xml$`, `post.*\.xml$`},
			wantErr:  false,
		},
		{
			name:     "invalid regex",
			patterns: []string{`[invalid`},
			wantErr:  true,
		},
		{
			name:     "empty patterns",
			patterns: []string{},
			wantErr:  false,
		},
		{
			name:     "mixed valid/invalid",
			patterns: []string{`valid\.xml$`, `[invalid`},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled, err := CompilePatterns(tt.patterns)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.wantErr && len(compiled) != len(tt.patterns) {
				t.Errorf("Expected %d compiled patterns, got %d", len(tt.patterns), len(compiled))
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "simple domain",
			url:      "https://example.com/path",
			expected: "example.com",
		},
		{
			name:     "subdomain",
			url:      "https://blog.example.com/",
			expected: "blog.example.com",
		},
		{
			name:     "port number",
			url:      "http://localhost:8080/test",
			expected: "localhost",
		},
		{
			name:     "invalid URL",
			url:      "not-a-url",
			expected: "",
		},
		{
			name:     "case insensitive",
			url:      "https://EXAMPLE.COM/",
			expected: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractDomain(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPatternConfigJSON(t *testing.T) {
	config := PatternConfig{
		SitemapPatterns: []string{"pattern1", "pattern2"},
		URLPatterns:     []string{"url1", "url2"},
	}

	// Test ToJSON
	jsonStr, err := config.ToJSON()
	if err != nil {
		t.Errorf("ToJSON failed: %v", err)
	}

	// Test FromJSON
	parsed, err := FromJSON(jsonStr)
	if err != nil {
		t.Errorf("FromJSON failed: %v", err)
	}

	if len(parsed.SitemapPatterns) != 2 || parsed.SitemapPatterns[0] != "pattern1" {
		t.Error("SitemapPatterns not preserved in JSON round-trip")
	}

	if len(parsed.URLPatterns) != 2 || parsed.URLPatterns[0] != "url1" {
		t.Error("URLPatterns not preserved in JSON round-trip")
	}
}
