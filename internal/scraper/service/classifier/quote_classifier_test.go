package classifier

import (
	"testing"
)

func TestQuoteClassifierDecision_Marshal(t *testing.T) {
	features := map[string]interface{}{
		"text_char_count": 3100,
		"num_text_blocks": 14,
		"avg_quote_score": 0.82,
	}
	selectors := []string{".quote-block", "#main .quote"}
	decision := NewQuoteClassifierDecision(
		"https://example.com/page",
		features,
		true,
		selectors,
		0.92,
		DecisionQuoteStructure,
	)
	jsonStr, err := decision.MarshalDecision()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if len(jsonStr) == 0 {
		t.Error("empty json output")
	}
}
