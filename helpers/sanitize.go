package helpers

import "strings"

// SanitizeAssistantMessage normalizes AI replies to avoid leaking markdown artifacts
// into persisted messages or user-visible outputs.
func SanitizeAssistantMessage(msg string) string {
	replacer := strings.NewReplacer("**", "", "***", "", "---", "—", "__", "", "##", "", "```", "")
	cleaned := replacer.Replace(msg)
	return strings.TrimSpace(cleaned)
}
