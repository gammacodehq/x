package safe_tg

import (
 "strings"
)

// SplitMessage splits the provided text into chunks that can safely be sent to
// Telegram.  Each chunk will be at most maxLength bytes long.  The split is
// performed on newline or space boundaries, and it guarantees that an opening
// ```` and closing ```` sequence is never split between chunks.  If an
// ```` starts in one chunk and does not finish, the chunk will receive the
// closing ```` and the next chunk will get the opening ````.
//
// The behaviour matches the original python implementation.
func SplitMessage(text string, maxLength int) []string {
 if len(text) <= maxLength {
  return []string{text}
 }

 messages := []string{}

 for len(text) > 0 {
  // Look for a newline before maxLength.
  lastNewLine := strings.LastIndex(text[:maxLength], "\n")
  if lastNewLine <= 0 {
   // No suitable newline: look for a space.
   lastSpace := strings.LastIndex(text[:maxLength], " ")
   if lastSpace <= 0 {
    lastSpace = maxLength
   }
   messages = append(messages, text[:lastSpace])
   text = text[lastSpace:]
  } else {
   messages = append(messages, text[:lastNewLine])
   // Skip the newline character itself.
   text = text[lastNewLine+1:]
  }

  // If the remaining text fits into one chunk we can finish early.
  if len(text) <= maxLength {
   if len(text) > 0 {
    messages = append(messages, text)
   }
   break
  }
 }

 // Balance triple‑backtick fences across the chunks.
 for i := 0; i < len(messages); i++ {
  count := strings.Count(messages[i], "```")
  if count%2 != 0 {
   // Odd number of fences – add a closing fence.
   messages[i] += "```"
   // If there is a following chunk, prepend an opening fence.
   if i+1 < len(messages) {
    messages[i+1] = "```" + messages[i+1]
   }
  }
 }

 return messages
}


/*
Как использовать

```go
func main() {
 longText := "..."
 chunks := SplitMessage(longText, 4096)
 for i, c := range chunks {
  fmt.Printf("Chunk %d: %s\n", i+1, c)
 }
}
```
*/
