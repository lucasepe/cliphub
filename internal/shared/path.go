package shared

import (
	"path/filepath"
	"strings"
)

// PathWithSuffix returns inputPath's filename stem plus suffix, preserving or replacing its extension.
func PathWithSuffix(inputPath, suffix, replacementExt string) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	originalExt := filepath.Ext(base)
	ext := originalExt
	if replacementExt != "" {
		ext = replacementExt
	}
	name := strings.TrimSuffix(base, originalExt)
	return filepath.Join(dir, name+suffix+ext)
}
