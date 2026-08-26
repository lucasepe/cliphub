package captions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultEmojiCacheDirUsesEnvironment(t *testing.T) {
	want := filepath.Join(t.TempDir(), "emoji-cache")
	t.Setenv(emojiCacheEnv, want)

	got, err := defaultEmojiCacheDir()
	if err != nil {
		t.Fatalf("defaultEmojiCacheDir() error = %v", err)
	}
	if got != want {
		t.Fatalf("defaultEmojiCacheDir() = %q, want %q", got, want)
	}
}

func TestDefaultEmojiCacheDirUsesOSCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(emojiCacheEnv, "")

	got, err := defaultEmojiCacheDir()
	if err != nil {
		t.Fatalf("defaultEmojiCacheDir() error = %v", err)
	}
	if got == "" {
		t.Fatal("defaultEmojiCacheDir() = empty")
	}
	if got == home {
		t.Fatal("defaultEmojiCacheDir() should create a cache subdirectory")
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("stat cache dir %q: %v", got, err)
	}
}

func TestEmojiURLUsesTwemojiFilename(t *testing.T) {
	got := emojiURL("🏍️")
	want := twemojiBaseURL + "/1f3cd.png"
	if got != want {
		t.Fatalf("emojiURL() = %q, want %q", got, want)
	}
}
