package filter

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// FilterAction defines the action to take when sensitive words are detected
type FilterAction string

const (
	// ActionBlock blocks the entire message
	ActionBlock FilterAction = "block"
	// ActionReplace replaces sensitive words with asterisks
	ActionReplace FilterAction = "replace"
	// ActionAudit logs the sensitive words but allows the message
	ActionAudit FilterAction = "audit"
)

// Match represents a detected sensitive word match
type Match struct {
	Word     string // The matched sensitive word
	Position int    // Start position in the original text
	Length   int    // Length of the match
}

// FilterResult contains the result of filtering
type FilterResult struct {
	ContainsSensitiveWords bool
	FilteredContent        string
	Matches                []Match
	Action                 FilterAction
}

// Config holds configuration for SensitiveWordFilter
type Config struct {
	Enabled          bool
	DefaultAction    FilterAction
	WordLists        map[string]string // language -> file path
	CaseSensitive    bool
	NormalizeUnicode bool
}

// ACNode represents a node in the Aho-Corasick automaton
type ACNode struct {
	children map[rune]*ACNode
	fail     *ACNode
	output   []string // Words that end at this node
}

// SensitiveWordFilter provides O(n) sensitive word filtering using Aho-Corasick automaton
// Requirements: 11.4, 17.4, 17.5
type SensitiveWordFilter struct {
	root             *ACNode
	config           Config
	mu               sync.RWMutex
	caseSensitive    bool
	normalizeUnicode bool
}

// NewSensitiveWordFilter creates a new sensitive word filter
// Requirements: 11.4
func NewSensitiveWordFilter(cfg Config) (*SensitiveWordFilter, error) {
	filter := &SensitiveWordFilter{
		root:             &ACNode{children: make(map[rune]*ACNode)},
		config:           cfg,
		caseSensitive:    cfg.CaseSensitive,
		normalizeUnicode: cfg.NormalizeUnicode,
	}

	// Load word lists from configuration files
	for lang, filePath := range cfg.WordLists {
		if err := filter.LoadWordList(filePath, lang); err != nil {
			return nil, fmt.Errorf("failed to load word list for %s: %w", lang, err)
		}
	}

	// Build failure links for AC automaton
	filter.buildFailureLinks()

	return filter, nil
}

// LoadWordList loads sensitive words from a file
// Requirements: 11.4
func (f *SensitiveWordFilter) LoadWordList(filePath string, lang string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open word list file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word == "" || strings.HasPrefix(word, "#") {
			continue // Skip empty lines and comments
		}

		// Normalize the word
		word = f.normalizeWord(word)

		// Add word to trie
		f.addWord(word)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading word list file %s: %w", filePath, err)
	}

	return nil
}

// normalizeWord normalizes a word based on configuration
func (f *SensitiveWordFilter) normalizeWord(word string) string {
	// Unicode normalization
	if f.normalizeUnicode {
		word = norm.NFC.String(word)
	}

	// Case normalization
	if !f.caseSensitive {
		word = strings.ToLower(word)
	}

	return word
}

// addWord adds a word to the AC automaton trie
func (f *SensitiveWordFilter) addWord(word string) {
	node := f.root
	runes := []rune(word)

	for _, r := range runes {
		if node.children[r] == nil {
			node.children[r] = &ACNode{children: make(map[rune]*ACNode)}
		}
		node = node.children[r]
	}

	// Mark this node as an output node
	if node.output == nil {
		node.output = []string{}
	}
	node.output = append(node.output, word)
}

// buildFailureLinks builds failure links for the AC automaton using BFS
// Requirements: 17.4
func (f *SensitiveWordFilter) buildFailureLinks() {
	queue := []*ACNode{}

	// Initialize failure links for depth-1 nodes
	for _, child := range f.root.children {
		child.fail = f.root
		queue = append(queue, child)
	}

	// BFS to build failure links
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for r, child := range current.children {
			queue = append(queue, child)

			// Find failure link
			failNode := current.fail
			for failNode != nil && failNode.children[r] == nil {
				if failNode == f.root {
					break
				}
				failNode = failNode.fail
			}

			if failNode == nil || failNode == f.root {
				if f.root.children[r] != nil && f.root.children[r] != child {
					child.fail = f.root.children[r]
				} else {
					child.fail = f.root
				}
			} else {
				child.fail = failNode.children[r]
			}

			// Merge output from failure link
			if child.fail != nil && child.fail.output != nil {
				if child.output == nil {
					child.output = []string{}
				}
				child.output = append(child.output, child.fail.output...)
			}
		}
	}
}

// Filter filters the content for sensitive words
// Returns FilterResult with detected matches and filtered content
// Requirements: 11.4, 17.4, 17.5 - O(n) complexity using AC automaton
func (f *SensitiveWordFilter) Filter(content string, action FilterAction) FilterResult {
	if !f.config.Enabled {
		return FilterResult{
			ContainsSensitiveWords: false,
			FilteredContent:        content,
			Matches:                []Match{},
			Action:                 action,
		}
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// Normalize content
	normalizedContent := f.normalizeWord(content)
	runes := []rune(normalizedContent)

	// Find all matches using AC automaton - O(n) complexity
	matches := f.findMatches(runes)

	if len(matches) == 0 {
		return FilterResult{
			ContainsSensitiveWords: false,
			FilteredContent:        content,
			Matches:                []Match{},
			Action:                 action,
		}
	}

	// Apply action
	var filteredContent string
	switch action {
	case ActionBlock:
		filteredContent = "" // Block entire message
	case ActionReplace:
		filteredContent = f.replaceMatches(content, matches)
	case ActionAudit:
		filteredContent = content // Allow message but log matches
	default:
		filteredContent = content
	}

	return FilterResult{
		ContainsSensitiveWords: true,
		FilteredContent:        filteredContent,
		Matches:                matches,
		Action:                 action,
	}
}

// findMatches finds all sensitive word matches using AC automaton
// O(n) complexity where n is the length of the text
// Requirements: 17.4
func (f *SensitiveWordFilter) findMatches(runes []rune) []Match {
	matches := []Match{}
	node := f.root

	for i, r := range runes {
		// Follow failure links until we find a match or reach root
		for node != f.root && node.children[r] == nil {
			node = node.fail
		}

		// Move to next node if possible
		if node.children[r] != nil {
			node = node.children[r]
		}

		// Check for matches at current position
		if node.output != nil {
			for _, word := range node.output {
				wordLen := len([]rune(word))
				position := i - wordLen + 1
				matches = append(matches, Match{
					Word:     word,
					Position: position,
					Length:   wordLen,
				})
			}
		}
	}

	return matches
}

// replaceMatches replaces sensitive words with asterisks
// Requirements: 11.4
func (f *SensitiveWordFilter) replaceMatches(content string, matches []Match) string {
	if len(matches) == 0 {
		return content
	}

	runes := []rune(content)
	replaced := make([]rune, len(runes))
	copy(replaced, runes)

	// Mark positions to replace
	toReplace := make(map[int]bool)
	for _, match := range matches {
		for i := match.Position; i < match.Position+match.Length && i < len(runes); i++ {
			toReplace[i] = true
		}
	}

	// Replace marked positions with asterisks
	for i := range replaced {
		if toReplace[i] {
			// Preserve whitespace
			if !unicode.IsSpace(replaced[i]) {
				replaced[i] = '*'
			}
		}
	}

	return string(replaced)
}

// UpdateWordList updates the word list at runtime
// Requirements: 11.4
func (f *SensitiveWordFilter) UpdateWordList(words []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Rebuild trie with new words
	f.root = &ACNode{children: make(map[rune]*ACNode)}

	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		// Normalize and add word
		word = f.normalizeWord(word)
		f.addWord(word)
	}

	// Rebuild failure links
	f.buildFailureLinks()

	return nil
}

// GetConfig returns the current configuration
func (f *SensitiveWordFilter) GetConfig() Config {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.config
}
