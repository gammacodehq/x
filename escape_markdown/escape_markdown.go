package escape_markdown

func EscapeMarkdown(s string) string {
	escapeMap := map[rune]bool{
		'*': true, // Asterisk for bold/italic
		'_': true, // Underscore for bold/italic
		'`': true, // Backtick for inline code
		'~': true, // Tilde for strikethrough
		'#': true, // Hash for headers
		'-': true, // Hyphen for list markers
		'!': true, // Exclamation for emphasis (less common, but included)
	}
	var result []rune
	for _, r := range s {
		if escapeMap[r] {
			result = append(result, '\\', r)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
