package safe_tg

import (
	"strings"
)

// Stupid (but works) escape: just prepend \ to every special symbol.
func EscapeStupid(text string) string {
	// All characters that need escaping in Telegram Markdown.
	special := `\\*_[]()~>#+-=|{}.!` // note: backslash included once

	var out strings.Builder
	for _, r := range text {
		if strings.ContainsRune(special, r) {
			out.WriteRune('\\')
		}
		out.WriteRune(r)
	}
	return out.String()
}
