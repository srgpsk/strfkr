package classifier

import (
	"encoding/json"
	"math"
	"strings"
	"time"
)

// Quote block length thresholds
const (
	minQuoteLength    = 30
	maxQuoteLength    = 300
	introParagraphMax = 120
	shallowDepthMax   = 4
	dialogPrefix      = "–"
)

// DecisionReason enumerates possible classifier outcomes
// See design doc for full list
const (
	DecisionQuoteStructure      = "QUOTE_STRUCTURE"
	DecisionLongNarrative       = "LONG_NARRATIVE"
	DecisionSingleAuthorBias    = "SINGLE_AUTHOR_BIAS"
	DecisionTooFewBlocks        = "TOO_FEW_BLOCKS"
	DecisionLowQuoteScore       = "LOW_QUOTE_SCORE"
	DecisionDominantSelectorLow = "DOMINANT_SELECTOR_LOW"
	DecisionShortMainText       = "SHORT_MAIN_TEXT"
	DecisionOneLongParagraph    = "ONE_LONG_PARAGRAPH"
	DecisionDialogPattern       = "DIALOG_PATTERN"
	DecisionStructuredDiverse   = "STRUCTURED_DIVERSE"
	DecisionBirthdayMessages    = "BIRTHDAY_MESSAGES"
	DecisionEdgeCase            = "EDGE_CASE"
)

// QuoteClassifierDecision holds the classifier output for a page
// This is stored as JSON in the DB
// selectors: always present, max 2 CSS selectors, empty for non-quotes pages
// confidence: classifier's confidence score
// decision_reason: enum value
// classified_at: timestamp
// processable: true/false
// features: extracted features
// url: page url
// (see design doc for full feature list)
type QuoteClassifierDecision struct {
	URL      string                 `json:"url"`
	Features map[string]interface{} `json:"features"`
	Decision QuoteDecision          `json:"decision"`
}

type QuoteDecision struct {
	Processable    bool     `json:"processable"`
	Selectors      []string `json:"selectors"`
	Confidence     float64  `json:"confidence"`
	DecisionReason string   `json:"decision_reason"`
	ClassifiedAt   string   `json:"classified_at"`
}

// NewQuoteClassifierDecision creates a new decision struct
func NewQuoteClassifierDecision(url string, features map[string]interface{}, processable bool, selectors []string, confidence float64, reason string) *QuoteClassifierDecision {
	return &QuoteClassifierDecision{
		URL:      url,
		Features: features,
		Decision: QuoteDecision{
			Processable:    processable,
			Selectors:      selectors,
			Confidence:     confidence,
			DecisionReason: reason,
			ClassifiedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	}
}

// MarshalDecision marshals the decision to JSON for DB storage
func (d *QuoteClassifierDecision) MarshalDecision() (string, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// QuotePageClassifierService provides feature extraction and decision logic
// Implements full feature extraction and classification per design doc

type QuotePageClassifierService struct{}

func NewQuotePageClassifierService() *QuotePageClassifierService {
	return &QuotePageClassifierService{}
}

// PatternStats holds statistics for pattern mining
type PatternStats struct {
	TextCharCount   int
	LongestBlockLen int
	BlockLens       []int
	QuoteScores     []float64
	BlockAuthors    map[string]int
	IntroParagraph  bool
	ColorizedBlocks bool
	SelectorCount   map[string]int
	BlockSelectors  []string
	BlockPaths      map[string]bool
	DialogPattern   bool
}

// Refactored ClassifyPage to use PageFeatureExtractor for HTML traversal and feature extraction
func (s *QuotePageClassifierService) ClassifyPage(url string, htmlStr string) (*QuoteClassifierDecision, error) {
	extractor := NewPageFeatureExtractor()
	_, stats, err := extractor.ExtractFeatures(htmlStr)
	if err != nil {
		return nil, err
	}

	// 3. Decision Tree
	decisionReason, processable, selectors, confidence, features := makeClassificationDecision(stats)

	return NewQuoteClassifierDecision(url, features, processable, selectors, confidence, decisionReason), nil
}

// Heuristic helpers for quote-likeness
func hasExplicitSentenceEnd(text string) bool {
	return strings.HasSuffix(text, ".") || strings.HasSuffix(text, "!") || strings.HasSuffix(text, "?")
}

func startsWithCapital(text string) bool {
	if len(text) == 0 {
		return false
	}
	return strings.ToUpper(string(text[0])) == string(text[0])
}

func isDialogFormat(text string) bool {
	return strings.HasPrefix(text, dialogPrefix)
}

func isShallow(depth int) bool {
	return depth < shallowDepthMax
}

// Pattern mining and heuristics extraction
func buildPatternStats(blocks []textBlock) PatternStats {
	stats := PatternStats{
		BlockAuthors:    map[string]int{},
		SelectorCount:   map[string]int{},
		BlockPaths:      map[string]bool{},
		BlockSelectors:  []string{},
		BlockLens:       []int{},
		QuoteScores:     []float64{},
		IntroParagraph:  false,
		ColorizedBlocks: false,
	}
	for i, b := range blocks {
		stats.TextCharCount += len(b.Text)
		stats.BlockLens = append(stats.BlockLens, len(b.Text))
		if len(b.Text) > stats.LongestBlockLen {
			stats.LongestBlockLen = len(b.Text)
		}
		stats.BlockSelectors = append(stats.BlockSelectors, b.Selector)
		stats.SelectorCount[b.Selector]++
		stats.BlockPaths[b.Path] = true

		qs := computeQuoteScore(b, stats.BlockAuthors)
		stats.QuoteScores = append(stats.QuoteScores, qs)
		if i == 0 && len(b.Text) < introParagraphMax {
			stats.IntroParagraph = true
		}
		if b.Colorized {
			stats.ColorizedBlocks = true
		}
	}
	return stats
}

func computeQuoteScore(b textBlock, blockAuthors map[string]int) float64 {
	qs := 0.0
	if isProperLength(b.Text) {
		qs += 0.3
	}
	if hasExplicitSentenceEnd(b.Text) {
		qs += 0.1
	}
	if startsWithCapital(b.Text) {
		qs += 0.1
	}
	// Penalize dialog format, do not consider as quote
	if isDialogFormat(b.Text) {
		qs -= 0.1
	}
	if isShallow(b.Depth) {
		qs += 0.1
	}
	return qs
}

// Updated makeClassificationDecision to use DecisionStats from decision_tree.go
func makeClassificationDecision(stats PatternStats) (decisionReason string, processable bool, selectors []string, confidence float64, features map[string]interface{}) {
	numTextBlocks := len(stats.BlockLens)
	avgBlockLen := 0
	if numTextBlocks > 0 {
		for _, l := range stats.BlockLens {
			avgBlockLen += l
		}
		avgBlockLen /= numTextBlocks
	}

	shortBlockRatio := 0.0
	for _, l := range stats.BlockLens {
		if l < 300 {
			shortBlockRatio += 1.0
		}
	}
	if numTextBlocks > 0 {
		shortBlockRatio /= float64(numTextBlocks)
	}
	dominantSelectorCount := 0
	for _, cnt := range stats.SelectorCount {
		if cnt > dominantSelectorCount {
			dominantSelectorCount = cnt
		}
	}
	dominantSelectorRatio := 0.0
	if numTextBlocks > 0 {
		dominantSelectorRatio = float64(dominantSelectorCount) / float64(numTextBlocks)
	}
	numDistinctPaths := len(stats.BlockPaths)
	avgQuoteScore := 0.0
	for _, qs := range stats.QuoteScores {
		avgQuoteScore += qs
	}
	if numTextBlocks > 0 {
		avgQuoteScore /= float64(numTextBlocks)
	}
	stddevQuoteScore := 0.0
	if numTextBlocks > 0 {
		mean := avgQuoteScore
		for _, qs := range stats.QuoteScores {
			stddevQuoteScore += (qs - mean) * (qs - mean)
		}
		stddevQuoteScore = math.Sqrt(stddevQuoteScore / float64(numTextBlocks))
	}
	singleAuthorBias := false
	maxAuthor := 0
	for _, cnt := range stats.BlockAuthors {
		if cnt > maxAuthor {
			maxAuthor = cnt
		}
	}
	if numTextBlocks > 0 && maxAuthor > int(0.8*float64(numTextBlocks)) {
		singleAuthorBias = true
	}

	decision := processableDecision(stats, numTextBlocks, dominantSelectorRatio, avgQuoteScore, singleAuthorBias)

	features = map[string]interface{}{
		"text_char_count":               stats.TextCharCount,
		"num_text_blocks":               numTextBlocks,
		"avg_block_length":              avgBlockLen,
		"longest_block_length":          stats.LongestBlockLen,
		"short_block_ratio":             shortBlockRatio,
		"dominant_selector_ratio":       dominantSelectorRatio,
		"num_distinct_paths":            numDistinctPaths,
		"avg_quote_score":               avgQuoteScore,
		"stddev_quote_score":            stddevQuoteScore,
		"single_author_bias":            singleAuthorBias,
		"has_intro_paragraph":           stats.IntroParagraph,
		"page_contains_dialog_patterns": stats.DialogPattern,
		"has_colorized_blocks":          stats.ColorizedBlocks,
	}
	return decision.Reason, decision.Processable, decision.Selectors, decision.Confidence, features
}

func isProperLength(text string) bool {
	return len(text) >= minQuoteLength && len(text) <= maxQuoteLength
}
