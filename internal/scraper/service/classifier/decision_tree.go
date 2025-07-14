package classifier

// Decision tree configuration constants
const (
	minTextCharCount     = 500
	minLongParagraphLen  = 400
	minNumBlocks         = 3
	minDominantSelector  = 0.7
	highQuoteScore       = 0.7
	structuredQuoteScore = 0.5
	colorizedSelector    = "[style*=color]"
	maxPageSelectors     = 2 // Maximum selectors to use per page
)

// DecisionStats struct definition
type DecisionStats struct {
	Reason      string
	Processable bool
	Selectors   []string
	Confidence  float64
}

type DecisionContext struct {
	Stats                 PatternStats
	NumTextBlocks         int
	DominantSelector      string
	DominantSelectorRatio float64
	AvgQuoteScore         float64
	SingleAuthorBias      bool
}

type DecisionRule func(ctx DecisionContext) *DecisionStats

var decisionRules = []DecisionRule{
	ruleShortMainText,
	ruleOneLongParagraph,
	ruleTooFewBlocks,
	ruleLowDominantSelector,
	ruleHighQuoteScore,
	ruleStructuredDiverseContent,
	ruleSingleAuthorBias,
	ruleDialogPattern,
	ruleLowQuoteScore,
}

// Selector helper
func getSelectors(stats PatternStats, dominantSelector string) []string {
	selectors := []string{dominantSelector}
	if stats.ColorizedBlocks {
		selectors = append(selectors, colorizedSelector)
	}
	if len(selectors) > maxPageSelectors {
		selectors = selectors[:maxPageSelectors]
	}
	return selectors
}

// Decision tree logic, returns DecisionStats
func processableDecision(stats PatternStats, numTextBlocks int, dominantSelectorRatio float64, avgQuoteScore float64, singleAuthorBias bool) DecisionStats {
	dominantSelector := ""
	dominantSelectorCount := 0
	for sel, cnt := range stats.SelectorCount {
		if cnt > dominantSelectorCount {
			dominantSelector = sel
			dominantSelectorCount = cnt
		}
	}
	ctx := DecisionContext{
		Stats:                 stats,
		NumTextBlocks:         numTextBlocks,
		DominantSelector:      dominantSelector,
		DominantSelectorRatio: dominantSelectorRatio,
		AvgQuoteScore:         avgQuoteScore,
		SingleAuthorBias:      singleAuthorBias,
	}
	for _, rule := range decisionRules {
		if result := rule(ctx); result != nil {
			return *result
		}
	}
	return DecisionStats{Reason: DecisionEdgeCase, Processable: false, Selectors: nil, Confidence: 0.0}
}

// Rule implementations
func ruleShortMainText(ctx DecisionContext) *DecisionStats {
	if ctx.Stats.TextCharCount < minTextCharCount {
		return &DecisionStats{Reason: DecisionShortMainText, Processable: false, Selectors: nil, Confidence: 0.1}
	}
	return nil
}

func ruleOneLongParagraph(ctx DecisionContext) *DecisionStats {
	if len(ctx.Stats.BlockLens) == 1 && ctx.Stats.LongestBlockLen > minLongParagraphLen {
		return &DecisionStats{Reason: DecisionOneLongParagraph, Processable: false, Selectors: nil, Confidence: 0.2}
	}
	return nil
}

func ruleTooFewBlocks(ctx DecisionContext) *DecisionStats {
	if len(ctx.Stats.BlockLens) < minNumBlocks {
		return &DecisionStats{Reason: DecisionTooFewBlocks, Processable: false, Selectors: nil, Confidence: 0.2}
	}
	return nil
}

func ruleLowDominantSelector(ctx DecisionContext) *DecisionStats {
	if ctx.DominantSelectorRatio < minDominantSelector {
		return &DecisionStats{Reason: DecisionDominantSelectorLow, Processable: false, Selectors: nil, Confidence: 0.3}
	}
	return nil
}

func ruleHighQuoteScore(ctx DecisionContext) *DecisionStats {
	// If single author bias, always unprocessable
	if ctx.SingleAuthorBias {
		return &DecisionStats{Reason: DecisionSingleAuthorBias, Processable: false, Selectors: nil, Confidence: ctx.AvgQuoteScore}
	}
	if ctx.AvgQuoteScore >= highQuoteScore {
		selectors := getSelectors(ctx.Stats, ctx.DominantSelector)
		return &DecisionStats{Reason: DecisionQuoteStructure, Processable: true, Selectors: selectors, Confidence: ctx.AvgQuoteScore}
	}
	return nil
}

func ruleStructuredDiverseContent(ctx DecisionContext) *DecisionStats {
	if ctx.AvgQuoteScore >= structuredQuoteScore && !ctx.Stats.DialogPattern {
		selectors := getSelectors(ctx.Stats, ctx.DominantSelector)
		return &DecisionStats{Reason: DecisionStructuredDiverse, Processable: true, Selectors: selectors, Confidence: ctx.AvgQuoteScore}
	}
	return nil
}

func ruleSingleAuthorBias(ctx DecisionContext) *DecisionStats {
	if ctx.SingleAuthorBias {
		return &DecisionStats{Reason: DecisionSingleAuthorBias, Processable: false, Selectors: nil, Confidence: 0.4}
	}
	return nil
}

func ruleDialogPattern(ctx DecisionContext) *DecisionStats {
	if ctx.Stats.DialogPattern {
		return &DecisionStats{Reason: DecisionDialogPattern, Processable: false, Selectors: nil, Confidence: 0.3}
	}
	return nil
}

func ruleLowQuoteScore(ctx DecisionContext) *DecisionStats {
	if ctx.AvgQuoteScore < structuredQuoteScore {
		return &DecisionStats{Reason: DecisionLowQuoteScore, Processable: false, Selectors: nil, Confidence: ctx.AvgQuoteScore}
	}
	return nil
}
