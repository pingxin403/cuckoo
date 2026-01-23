# Sensitive Word Filter

A high-performance content filtering service using the Aho-Corasick automaton algorithm for O(n) complexity sensitive word detection.

## Features

- **Fast Detection**: O(n) complexity using AC automaton (single pass through content)
- **Multi-Language Support**: Handles English, Chinese, Japanese, and other Unicode languages
- **Flexible Actions**: Block, replace, or audit detected sensitive words
- **Unicode Normalization**: Handles different Unicode representations of the same character
- **Case Sensitivity**: Configurable case-sensitive or case-insensitive matching
- **Runtime Updates**: Add or remove words from the filter without restart
- **File-Based Configuration**: Load word lists from text files

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "github.com/pingxin403/cuckoo/apps/im-service/filter"
)

func main() {
    // Create filter with word list
    f := filter.NewSensitiveWordFilter([]string{"badword", "spam"}, filter.FilterConfig{
        Action:        filter.ActionReplace,
        CaseSensitive: false,
        Enabled:       true,
    })

    // Filter content
    result := f.Filter("This is a badword message")
    fmt.Println(result.Filtered)  // "This is a ******* message"
    fmt.Println(result.ContainsSensitiveWords)  // true
}
```

### Load from File

```go
// Load word list from file (one word per line)
f, err := filter.NewSensitiveWordFilterFromFile("sensitive_words.txt", filter.FilterConfig{
    Action:        filter.ActionBlock,
    CaseSensitive: false,
    Enabled:       true,
})
if err != nil {
    log.Fatal(err)
}

result := f.Filter("Check this content")
if result.ContainsSensitiveWords {
    // Handle blocked content
}
```

### Filter Actions

```go
// Block: Reject content containing sensitive words
config := filter.FilterConfig{
    Action:  filter.ActionBlock,
    Enabled: true,
}

// Replace: Replace sensitive words with asterisks
config := filter.FilterConfig{
    Action:  filter.ActionReplace,
    Enabled: true,
}

// Audit: Log but allow content (for monitoring)
config := filter.FilterConfig{
    Action:  filter.ActionAudit,
    Enabled: true,
}
```

### Multi-Language Support

```go
// English words
enWords := []string{"spam", "scam"}
enFilter := filter.NewSensitiveWordFilter(enWords, filter.FilterConfig{
    Action:  filter.ActionReplace,
    Enabled: true,
})

// Chinese words
zhWords := []string{"垃圾", "骗子"}
zhFilter := filter.NewSensitiveWordFilter(zhWords, filter.FilterConfig{
    Action:  filter.ActionReplace,
    Enabled: true,
})

// Use appropriate filter based on language
result := enFilter.Filter("This is spam")
```

### Runtime Updates

```go
f := filter.NewSensitiveWordFilter([]string{"word1"}, filter.FilterConfig{
    Action:  filter.ActionReplace,
    Enabled: true,
})

// Add new words
f.UpdateWordList([]string{"word2", "word3"})

// Filter will now detect all three words
result := f.Filter("Check word2 and word3")
```

## Configuration

```go
type FilterConfig struct {
    Action        FilterAction  // Block, Replace, or Audit
    CaseSensitive bool          // Case-sensitive matching
    Enabled       bool          // Enable/disable filter
}

type FilterAction string

const (
    ActionBlock   FilterAction = "block"    // Reject content
    ActionReplace FilterAction = "replace"  // Replace with asterisks
    ActionAudit   FilterAction = "audit"    // Log but allow
)
```

## Performance

- **Complexity**: O(n) where n is the content length
- **Memory**: ~1MB for 10,000 words
- **Throughput**: Tested with 100+ iterations in property-based tests
- **Scalability**: Handles messages up to 10,000 characters efficiently

## Testing

```bash
# Run unit tests
go test ./filter/... -v

# Run property-based tests (100 iterations each)
go test ./filter/... -tags=property -v

# Run with coverage
go test ./filter/... -cover
```

## Integration with IM Service

The filter is designed to integrate with the IM message routing pipeline:

1. **Pre-Encryption**: Filter content before encryption (Requirement 17.5)
2. **Message Router**: Apply filter in Message Router before delivery
3. **Configurable**: Different actions per message type or user group
4. **Monitoring**: Track filter statistics for compliance

```go
// Example integration in message router
func (r *MessageRouter) RouteMessage(msg *Message) error {
    // Apply sensitive word filter
    result := r.filter.Filter(msg.Content)
    
    if result.ContainsSensitiveWords {
        switch r.filterConfig.Action {
        case filter.ActionBlock:
            return errors.New("message contains sensitive words")
        case filter.ActionReplace:
            msg.Content = result.Filtered
        case filter.ActionAudit:
            log.Warn("sensitive words detected", "msg_id", msg.ID)
        }
    }
    
    // Continue with routing...
    return r.route(msg)
}
```

## Privacy Considerations

- **Server-Side Filtering**: Requires access to plaintext content
- **Not E2EE Compatible**: Cannot filter end-to-end encrypted messages
- **GDPR Compliance**: Justified as legitimate interest (Article 6(1)(f)) for platform safety
- **Future E2EE Mode**: Client-side filtering only when E2EE is enabled

## Word List Management

Word lists should be stored in text files with one word per line:

```
# sensitive_words_en.txt
spam
scam
fraud
phishing

# sensitive_words_zh.txt
垃圾
骗子
诈骗
```

Load multiple language files as needed:

```go
enFilter, _ := filter.NewSensitiveWordFilterFromFile("sensitive_words_en.txt", config)
zhFilter, _ := filter.NewSensitiveWordFilterFromFile("sensitive_words_zh.txt", config)
```

## Requirements Validated

- **Requirement 11.4**: Sensitive word filtering with configurable word list
- **Requirement 11.8**: Block or replace actions based on configuration
- **Requirement 17.4**: O(n) complexity using AC automaton
- **Requirement 17.5**: Filtering before encryption
- **Property 11**: Filtering consistency across all inputs

## References

- [Aho-Corasick Algorithm](https://en.wikipedia.org/wiki/Aho%E2%80%93Corasick_algorithm)
- [IM Chat System Design Document](.kiro/specs/im-chat-system/design.md)
- [Requirements Document](.kiro/specs/im-chat-system/requirements.md)
