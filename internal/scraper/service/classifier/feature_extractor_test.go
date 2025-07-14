package classifier

import (
	"testing"
)

func TestFeatureExtractor_ExtractFeatures_HTML(t *testing.T) {
	extractor := NewPageFeatureExtractor()
	html := `<div>"Quote one."</div><div>Some text.</div><div>"Quote two."</div>`
	blocks, stats, err := extractor.ExtractFeatures(html)
	if err != nil {
		t.Fatalf("ExtractFeatures error: %v", err)
	}
	if len(blocks) != 3 {
		t.Errorf("Expected 3 blocks, got %d", len(blocks))
	}
	if stats.TextCharCount <= 0 {
		t.Errorf("Expected positive TextCharCount, got %d", stats.TextCharCount)
	}
}

func TestFeatureExtractor_ExtractFeatures_Empty(t *testing.T) {
	extractor := NewPageFeatureExtractor()
	html := ""
	blocks, stats, err := extractor.ExtractFeatures(html)
	if err != nil {
		t.Fatalf("ExtractFeatures error: %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("Expected 0 blocks, got %d", len(blocks))
	}
	if stats.TextCharCount != 0 {
		t.Errorf("Expected 0 TextCharCount, got %d", stats.TextCharCount)
	}
}
