package classifier

import (
	"testing"
)

func TestProcessableDecision_HighQuoteScore(t *testing.T) {
	stats := PatternStats{
		TextCharCount:   1000,
		BlockLens:       []int{120, 130, 140},
		QuoteScores:     []float64{0.8, 0.9, 0.85},
		SelectorCount:   map[string]int{"p": 3},
		ColorizedBlocks: false,
	}
	d := processableDecision(stats, 3, 1.0, 0.85, false)
	if !d.Processable || d.Reason != DecisionQuoteStructure {
		t.Errorf("Expected processable quote structure, got %v", d)
	}
}

func TestProcessableDecision_ShortMainText(t *testing.T) {
	stats := PatternStats{
		TextCharCount: 100,
		BlockLens:     []int{100},
		QuoteScores:   []float64{0.2},
		SelectorCount: map[string]int{"p": 1},
	}
	d := processableDecision(stats, 1, 1.0, 0.2, false)
	if d.Processable || d.Reason != DecisionShortMainText {
		t.Errorf("Expected non-processable short main text, got %v", d)
	}
}

func TestProcessableDecision_SingleAuthorBias(t *testing.T) {
	stats := PatternStats{
		TextCharCount:   1000,
		BlockLens:       []int{120, 130, 140},
		QuoteScores:     []float64{0.8, 0.9, 0.85},
		SelectorCount:   map[string]int{"p": 3},
		ColorizedBlocks: false,
	}
	d := processableDecision(stats, 3, 1.0, 0.85, true)
	if d.Processable || d.Reason != DecisionSingleAuthorBias {
		t.Errorf("Expected non-processable single author bias, got %v", d)
	}
}
