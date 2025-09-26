package escape_markdown

var (
	escapeSet = map[rune]struct{}{
		'*': {}, // Asterisk for bold/italic
		'_': {}, // Underscore for bold/italic
		'`': {}, // Backtick for inline code
		'~': {}, // Tilde for strikethrough
		'#': {}, // Hash for headers
		'-': {}, // Hyphen for list markers
		'!': {}, // Exclamation for emphasis (less common, but included)
	}
)

// EscapeMarkdown ...
func EscapeMarkdown(s string) string {
	var result []rune
	for _, r := range s {
		if _, ok := escapeSet[r]; ok {
			result = append(result, '\\', r)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
