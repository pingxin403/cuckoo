package filter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test word detection and replacement
func TestSensitiveWordFilter_BasicDetection(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	// Add test words
	err = filter.UpdateWordList([]string{"badword", "offensive", "spam"})
	require.NoError(t, err)

	// Test detection
	result := filter.Filter("This contains badword in it", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)
	assert.Equal(t, "This contains ******* in it", result.FilteredContent)
	assert.Len(t, result.Matches, 1)
	assert.Equal(t, "badword", result.Matches[0].Word)
}

// Test multiple word detection
func TestSensitiveWordFilter_MultipleWords(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	err = filter.UpdateWordList([]string{"bad", "word", "test"})
	require.NoError(t, err)

	result := filter.Filter("bad word test", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)
	assert.Equal(t, "*** **** ****", result.FilteredContent)
	assert.Len(t, result.Matches, 3)
}

// Test overlapping words
func TestSensitiveWordFilter_OverlappingWords(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	// Add overlapping words
	err = filter.UpdateWordList([]string{"ass", "assassin"})
	require.NoError(t, err)

	result := filter.Filter("The assassin was caught", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)
	// Should detect both "ass" and "assassin"
	assert.GreaterOrEqual(t, len(result.Matches), 1)
}

// Test Unicode normalization
func TestSensitiveWordFilter_UnicodeNormalization(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	// Add word with accented characters
	err = filter.UpdateWordList([]string{"café"})
	require.NoError(t, err)

	// Test with different Unicode representations
	result := filter.Filter("I love café", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)
}

// Test case sensitivity
func TestSensitiveWordFilter_CaseSensitivity(t *testing.T) {
	// Case insensitive
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	err = filter.UpdateWordList([]string{"badword"})
	require.NoError(t, err)

	result := filter.Filter("BADWORD BadWord badword", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)
	assert.Len(t, result.Matches, 3)

	// Case sensitive
	cfg.CaseSensitive = true
	filter, err = NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	err = filter.UpdateWordList([]string{"badword"})
	require.NoError(t, err)

	result = filter.Filter("BADWORD BadWord badword", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)
	assert.Len(t, result.Matches, 1) // Only exact case match
}

// Test block action
func TestSensitiveWordFilter_BlockAction(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionBlock,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	err = filter.UpdateWordList([]string{"badword"})
	require.NoError(t, err)

	result := filter.Filter("This contains badword", ActionBlock)
	assert.True(t, result.ContainsSensitiveWords)
	assert.Equal(t, "", result.FilteredContent) // Entire message blocked
	assert.Equal(t, ActionBlock, result.Action)
}

// Test replace action
func TestSensitiveWordFilter_ReplaceAction(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	err = filter.UpdateWordList([]string{"badword"})
	require.NoError(t, err)

	result := filter.Filter("This contains badword", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)
	assert.Equal(t, "This contains *******", result.FilteredContent)
	assert.Equal(t, ActionReplace, result.Action)
}

// Test audit action
func TestSensitiveWordFilter_AuditAction(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionAudit,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	err = filter.UpdateWordList([]string{"badword"})
	require.NoError(t, err)

	result := filter.Filter("This contains badword", ActionAudit)
	assert.True(t, result.ContainsSensitiveWords)
	assert.Equal(t, "This contains badword", result.FilteredContent) // Content unchanged
	assert.Len(t, result.Matches, 1)                                 // But matches are recorded
	assert.Equal(t, ActionAudit, result.Action)
}

// Test disabled filter
func TestSensitiveWordFilter_Disabled(t *testing.T) {
	cfg := Config{
		Enabled:          false,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	err = filter.UpdateWordList([]string{"badword"})
	require.NoError(t, err)

	result := filter.Filter("This contains badword", ActionReplace)
	assert.False(t, result.ContainsSensitiveWords)
	assert.Equal(t, "This contains badword", result.FilteredContent)
	assert.Len(t, result.Matches, 0)
}

// Test loading word list from file
func TestSensitiveWordFilter_LoadFromFile(t *testing.T) {
	// Create temporary word list file
	tmpDir := t.TempDir()
	wordListPath := filepath.Join(tmpDir, "words.txt")

	content := `# Test word list
badword
offensive
spam
# Another comment
illegal
`
	err := os.WriteFile(wordListPath, []byte(content), 0600)
	require.NoError(t, err)

	cfg := Config{
		Enabled:       true,
		DefaultAction: ActionReplace,
		WordLists: map[string]string{
			"en": wordListPath,
		},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	// Test that words were loaded
	result := filter.Filter("This is badword and spam", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)
	assert.Len(t, result.Matches, 2)
}

// Test empty content
func TestSensitiveWordFilter_EmptyContent(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	err = filter.UpdateWordList([]string{"badword"})
	require.NoError(t, err)

	result := filter.Filter("", ActionReplace)
	assert.False(t, result.ContainsSensitiveWords)
	assert.Equal(t, "", result.FilteredContent)
	assert.Len(t, result.Matches, 0)
}

// Test no sensitive words
func TestSensitiveWordFilter_NoSensitiveWords(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	err = filter.UpdateWordList([]string{"badword"})
	require.NoError(t, err)

	result := filter.Filter("This is a clean message", ActionReplace)
	assert.False(t, result.ContainsSensitiveWords)
	assert.Equal(t, "This is a clean message", result.FilteredContent)
	assert.Len(t, result.Matches, 0)
}

// Test performance (O(n) complexity)
func TestSensitiveWordFilter_Performance(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	// Add many words
	words := []string{}
	for i := 0; i < 1000; i++ {
		words = append(words, "word"+string(rune(i)))
	}
	err = filter.UpdateWordList(words)
	require.NoError(t, err)

	// Test with long content
	longContent := ""
	for i := 0; i < 10000; i++ {
		longContent += "a "
	}

	start := time.Now()
	result := filter.Filter(longContent, ActionReplace)
	duration := time.Since(start)

	// Should complete quickly (O(n) complexity)
	assert.Less(t, duration.Milliseconds(), int64(100), "Filtering should be fast (O(n))")
	assert.False(t, result.ContainsSensitiveWords)
}

// Test word boundaries
func TestSensitiveWordFilter_WordBoundaries(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	err = filter.UpdateWordList([]string{"ass"})
	require.NoError(t, err)

	// "ass" should match in "assassin" (substring match)
	result := filter.Filter("The assassin was caught", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)
	assert.Contains(t, result.FilteredContent, "*")
}

// Test multiple languages
func TestSensitiveWordFilter_MultipleLanguages(t *testing.T) {
	cfg := Config{
		Enabled:       true,
		DefaultAction: ActionReplace,
		WordLists: map[string]string{
			"en": "testdata/sensitive_words_en.txt",
			"zh": "testdata/sensitive_words_zh.txt",
		},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	// Test English word
	result := filter.Filter("This is badword content", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)

	// Test Chinese word
	result = filter.Filter("这是敏感词内容", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)
}

// Test UpdateWordList
func TestSensitiveWordFilter_UpdateWordList(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	// Initial word list
	err = filter.UpdateWordList([]string{"badword"})
	require.NoError(t, err)

	result := filter.Filter("badword test", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)

	// Update word list
	err = filter.UpdateWordList([]string{"newword"})
	require.NoError(t, err)

	// Old word should not match
	result = filter.Filter("badword test", ActionReplace)
	assert.False(t, result.ContainsSensitiveWords)

	// New word should match
	result = filter.Filter("newword test", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)
}

// Test GetConfig
func TestSensitiveWordFilter_GetConfig(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	retrievedCfg := filter.GetConfig()
	assert.Equal(t, cfg.Enabled, retrievedCfg.Enabled)
	assert.Equal(t, cfg.DefaultAction, retrievedCfg.DefaultAction)
	assert.Equal(t, cfg.CaseSensitive, retrievedCfg.CaseSensitive)
	assert.Equal(t, cfg.NormalizeUnicode, retrievedCfg.NormalizeUnicode)
}

// Test whitespace preservation
func TestSensitiveWordFilter_WhitespacePreservation(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		DefaultAction:    ActionReplace,
		WordLists:        map[string]string{},
		CaseSensitive:    false,
		NormalizeUnicode: true,
	}

	filter, err := NewSensitiveWordFilter(cfg)
	require.NoError(t, err)

	err = filter.UpdateWordList([]string{"bad word"})
	require.NoError(t, err)

	result := filter.Filter("This is bad word content", ActionReplace)
	assert.True(t, result.ContainsSensitiveWords)
	// Whitespace should be preserved
	assert.Contains(t, result.FilteredContent, " ")
}
