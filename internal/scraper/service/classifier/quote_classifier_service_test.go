package classifier

import (
	"testing"
)

func TestQuotePageClassifierService_ClassifyPage_Processable(t *testing.T) {
	service := NewQuotePageClassifierService()
	// Use multiple divs with highly quote-like text and different authors to maximize score and avoid single author bias
	quoteBlocks := []string{
		`"This is a proper quote. – Author1"`,
		`"Another insightful quote! – Author2"`,
		`"Yet another quote? – Author3"`,
		`"A fourth quote. – Author4"`,
		`"Fifth quote here. – Author5"`,
	}
	blocks := ""
	for _, qb := range quoteBlocks {
		blocks += `<div>` + qb + `</div>`
	}
	// Add enough filler to exceed minTextCharCount
	filler := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. "
	for i := 0; i < 10; i++ {
		blocks += `<div>` + filler + `</div>`
	}
	html := blocks
	result, err := service.ClassifyPage("http://example.com/quotes", html)
	if err != nil {
		t.Fatalf("ClassifyPage error: %v", err)
	}
	if !result.Decision.Processable {
		t.Errorf("Expected processable, got false")
	}
	if result.Decision.DecisionReason != DecisionQuoteStructure && result.Decision.DecisionReason != DecisionStructuredDiverse {
		t.Errorf("Expected reason %s or %s, got %s", DecisionQuoteStructure, DecisionStructuredDiverse, result.Decision.DecisionReason)
	}
}

func TestQuotePageClassifierService_ClassifyPage_NonProcessable(t *testing.T) {
	service := NewQuotePageClassifierService()
	html := `<div>No quotes here.</div>`
	result, err := service.ClassifyPage("http://example.com/noquotes", html)
	if err != nil {
		t.Fatalf("ClassifyPage error: %v", err)
	}
	if result.Decision.Processable {
		t.Errorf("Expected non-processable, got true")
	}
}

func TestQuotePageClassifierService_FeatureExtraction(t *testing.T) {
	service := NewQuotePageClassifierService()
	html := `<div>"Quote one."</div><div>Some text.</div><div>"Quote two."</div>`
	result, err := service.ClassifyPage("http://example.com/quotes", html)
	if err != nil {
		t.Fatalf("ClassifyPage error: %v", err)
	}
	if result.Features["num_text_blocks"] == nil || result.Features["text_char_count"].(int) <= 0 {
		t.Errorf("Expected valid feature extraction, got %+v", result.Features)
	}
}

func TestQuotePageClassifierService_EdgeCases(t *testing.T) {
	service := NewQuotePageClassifierService()
	// Empty HTML
	result, err := service.ClassifyPage("http://example.com/empty", "")
	if err != nil {
		t.Fatalf("ClassifyPage error: %v", err)
	}
	if result.Decision.Processable {
		t.Errorf("Expected non-processable for empty HTML, got true")
	}
	// Only dialog
	html := `<div>– Dialog line one.</div><div>– Dialog line two.</div>`
	result, err = service.ClassifyPage("http://example.com/dialog", html)
	if err != nil {
		t.Fatalf("ClassifyPage error: %v", err)
	}
	if result.Decision.Processable {
		t.Errorf("Expected non-processable for dialog, got true")
	}
}
