package shared

import (
	"fmt"
	"strings"
	"unicode"
)

// PrintCommand writes a shell-ready command line without executing it.
func PrintCommand(name string, args []string) {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, name)
	for _, arg := range args {
		parts = append(parts, ShellQuote(arg))
	}
	fmt.Println(strings.Join(parts, " "))
}

// ShellQuote returns value unchanged when it is shell-safe, otherwise single-quotes it.
func ShellQuote(value string) string {
	if value == "" {
		return "''"
	}
	for _, r := range value {
		if shellNeedsQuote(r) {
			return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
		}
	}
	return value
}

func shellNeedsQuote(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune(`"'$&;()<>[]{}*?!|\`, r)
}
