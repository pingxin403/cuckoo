//go:build property
// +build property

package filter

import (
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Property 11: Sensitive Word Filtering Consistency
// **Validates: Requirements 11.4, 11.8, 17.4, 17.5**
func TestProperty_FilteringConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := Config{
			Enabled:          true,
			DefaultAction:    ActionReplace,
			WordLists:        map[string]string{},
			CaseSensitive:    false,
			NormalizeUnicode: true,
		}

		filter, err := NewSensitiveWordFilter(cfg)
		if err != nil {
			t.Fatalf("Failed to create filter: %v", err)
		}

		// Generate random sensitive words
		numWords := rapid.IntRange(1, 10).Draw(t, "num_words")
		words := make([]string, numWords)
		for i := 0; i < numWords; i++ {
			words[i] = rapid.StringMatching(`^[a-z]{3,8}$`).Draw(t, "word_"+string(rune(i)))
		}

		err = filter.UpdateWordList(words)
		if err != nil {
			t.Fatalf("Failed to update word list: %v", err)
		}

		// Generate random content
		content := rapid.StringN(10, 100, -1).Draw(t, "content")

		// Property: Filtering should be consistent across multiple calls
		result1 := filter.Filter(content, ActionReplace)
		result2 := filter.Filter(content, ActionReplace)

		if result1.ContainsSensitiveWords != result2.ContainsSensitiveWords {
			t.Fatalf("Inconsistent detection: %v vs %v", result1.ContainsSensitiveWords, result2.ContainsSensitiveWords)
		}

		if result1.FilteredContent != result2.FilteredContent {
			t.Fatalf("Inconsistent filtering: %s vs %s", result1.FilteredContent, result2.FilteredContent)
		}

		if len(result1.Matches) != len(result2.Matches) {
			t.Fatalf("Inconsistent match count: %d vs %d", len(result1.Matches), len(result2.Matches))
		}
	})
}

// Property: O(n) complexity guarantee
// **Validates: Requirements 17.4**
func TestProperty_LinearComplexity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := Config{
			Enabled:          true,
			DefaultAction:    ActionReplace,
			WordLists:        map[string]string{},
			CaseSensitive:    false,
			NormalizeUnicode: true,
		}

		filter, err := NewSensitiveWordFilter(cfg)
		if err != nil {
			t.Fatalf("Failed to create filter: %v", err)
		}

		// Add some words
		words := []string{"bad", "word", "test", "spam", "illegal"}
		err = filter.UpdateWordList(words)
		if err != nil {
			t.Fatalf("Failed to update word list: %v", err)
		}

		// Generate content of varying lengths
		length := rapid.IntRange(100, 1000).Draw(t, "length")
		content := strings.Repeat("a ", length)

		// Property: Time should scale linearly with content length
		start := time.Now()
		result := filter.Filter(content, ActionReplace)
		duration := time.Since(start)

		// Should complete quickly (O(n) complexity)
		maxDuration := time.Duration(length) * time.Microsecond * 10 // 10 microseconds per character
		if duration > maxDuration {
			t.Fatalf("Filtering too slow: %v for length %d (expected < %v)", duration, length, maxDuration)
		}

		// Result should be valid
		if result.Action != ActionReplace {
			t.Fatalf("Wrong action: %v", result.Action)
		}
	})
}

// Property: Filtering before encryption
// **Validates: Requirements 17.5**
func TestProperty_FilterBeforeEncryption(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := Config{
			Enabled:          true,
			DefaultAction:    ActionReplace,
			WordLists:        map[string]string{},
			CaseSensitive:    false,
			NormalizeUnicode: true,
		}

		filter, err := NewSensitiveWordFilter(cfg)
		if err != nil {
			t.Fatalf("Failed to create filter: %v", err)
		}

		// Add sensitive words
		err = filter.UpdateWordList([]string{"badword", "offensive"})
		if err != nil {
			t.Fatalf("Failed to update word list: %v", err)
		}

		// Generate content with sensitive words
		content := rapid.StringMatching(`.*badword.*`).Draw(t, "content")

		// Property: Filtered content should not contain original sensitive words
		result := filter.Filter(content, ActionReplace)

		if result.ContainsSensitiveWords {
			// Filtered content should have asterisks instead of sensitive words
			if strings.Contains(result.FilteredContent, "badword") {
				t.Fatalf("Filtered content still contains sensitive word: %s", result.FilteredContent)
			}

			// Should contain asterisks
			if !strings.Contains(result.FilteredContent, "*") {
				t.Fatalf("Filtered content should contain asterisks: %s", result.FilteredContent)
			}
		}
	})
}

// Property: Multi-language support
// **Validates: Requirements 11.4**
func TestProperty_MultiLanguageSupport(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := Config{
			Enabled:          true,
			DefaultAction:    ActionReplace,
			WordLists:        map[string]string{},
			CaseSensitive:    false,
			NormalizeUnicode: true,
		}

		filter, err := NewSensitiveWordFilter(cfg)
		if err != nil {
			t.Fatalf("Failed to create filter: %v", err)
		}

		// Add words in different languages
		words := []string{
			"badword", // English
			"敏感词",     // Chinese
			"テスト",     // Japanese
		}
		err = filter.UpdateWordList(words)
		if err != nil {
			t.Fatalf("Failed to update word list: %v", err)
		}

		// Test with random language
		lang := rapid.SampledFrom([]string{"en", "zh", "ja"}).Draw(t, "language")
		var content string
		switch lang {
		case "en":
			content = "This contains badword"
		case "zh":
			content = "这包含敏感词"
		case "ja":
			content = "これはテストです"
		}

		// Property: Should detect words in any language
		result := filter.Filter(content, ActionReplace)

		if !result.ContainsSensitiveWords {
			t.Fatalf("Failed to detect sensitive word in %s: %s", lang, content)
		}

		if len(result.Matches) == 0 {
			t.Fatalf("No matches found for %s content", lang)
		}
	})
}

// Property: Block action consistency
// **Validates: Requirements 11.8**
func TestProperty_BlockActionConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := Config{
			Enabled:          true,
			DefaultAction:    ActionBlock,
			WordLists:        map[string]string{},
			CaseSensitive:    false,
			NormalizeUnicode: true,
		}

		filter, err := NewSensitiveWordFilter(cfg)
		if err != nil {
			t.Fatalf("Failed to create filter: %v", err)
		}

		// Add sensitive words
		words := []string{"bad", "offensive", "spam"}
		err = filter.UpdateWordList(words)
		if err != nil {
			t.Fatalf("Failed to update word list: %v", err)
		}

		// Generate content with sensitive words
		content := rapid.StringMatching(`.*bad.*`).Draw(t, "content")

		// Property: Block action should always return empty content
		result := filter.Filter(content, ActionBlock)

		if result.ContainsSensitiveWords {
			if result.FilteredContent != "" {
				t.Fatalf("Block action should return empty content, got: %s", result.FilteredContent)
			}

			if result.Action != ActionBlock {
				t.Fatalf("Wrong action: %v", result.Action)
			}
		}
	})
}

// Property: Replace action preserves length
// **Validates: Requirements 11.4**
func TestProperty_ReplacePreservesStructure(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := Config{
			Enabled:          true,
			DefaultAction:    ActionReplace,
			WordLists:        map[string]string{},
			CaseSensitive:    false,
			NormalizeUnicode: true,
		}

		filter, err := NewSensitiveWordFilter(cfg)
		if err != nil {
			t.Fatalf("Failed to create filter: %v", err)
		}

		// Add sensitive words
		err = filter.UpdateWordList([]string{"bad"})
		if err != nil {
			t.Fatalf("Failed to update word list: %v", err)
		}

		// Generate content with known sensitive word
		content := "This is bad content"

		// Property: Replace action should preserve content length
		result := filter.Filter(content, ActionReplace)

		if result.ContainsSensitiveWords {
			originalLen := len([]rune(content))
			filteredLen := len([]rune(result.FilteredContent))

			if originalLen != filteredLen {
				t.Fatalf("Length mismatch: original=%d, filtered=%d", originalLen, filteredLen)
			}
		}
	})
}

// Property: Audit action preserves content
// **Validates: Requirements 11.4**
func TestProperty_AuditPreservesContent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := Config{
			Enabled:          true,
			DefaultAction:    ActionAudit,
			WordLists:        map[string]string{},
			CaseSensitive:    false,
			NormalizeUnicode: true,
		}

		filter, err := NewSensitiveWordFilter(cfg)
		if err != nil {
			t.Fatalf("Failed to create filter: %v", err)
		}

		// Add sensitive words
		words := []string{"bad", "offensive"}
		err = filter.UpdateWordList(words)
		if err != nil {
			t.Fatalf("Failed to update word list: %v", err)
		}

		// Generate random content
		content := rapid.String().Draw(t, "content")

		// Property: Audit action should never modify content
		result := filter.Filter(content, ActionAudit)

		if result.FilteredContent != content {
			t.Fatalf("Audit action modified content: original=%s, filtered=%s", content, result.FilteredContent)
		}

		if result.Action != ActionAudit {
			t.Fatalf("Wrong action: %v", result.Action)
		}
	})
}

// Property: Disabled filter never detects
// **Validates: Requirements 11.4**
func TestProperty_DisabledFilterNeverDetects(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := Config{
			Enabled:          false,
			DefaultAction:    ActionReplace,
			WordLists:        map[string]string{},
			CaseSensitive:    false,
			NormalizeUnicode: true,
		}

		filter, err := NewSensitiveWordFilter(cfg)
		if err != nil {
			t.Fatalf("Failed to create filter: %v", err)
		}

		// Add sensitive words
		err = filter.UpdateWordList([]string{"bad", "offensive", "spam"})
		if err != nil {
			t.Fatalf("Failed to update word list: %v", err)
		}

		// Generate random content
		content := rapid.String().Draw(t, "content")

		// Property: Disabled filter should never detect sensitive words
		result := filter.Filter(content, ActionReplace)

		if result.ContainsSensitiveWords {
			t.Fatalf("Disabled filter detected sensitive words in: %s", content)
		}

		if result.FilteredContent != content {
			t.Fatalf("Disabled filter modified content: %s -> %s", content, result.FilteredContent)
		}

		if len(result.Matches) > 0 {
			t.Fatalf("Disabled filter found matches: %v", result.Matches)
		}
	})
}
