package shared

import "testing"

func TestShellQuoteLeavesSimpleArgumentsAlone(t *testing.T) {
	got := ShellQuote("input.mp4")
	if got != "input.mp4" {
		t.Fatalf("ShellQuote() = %q, want input.mp4", got)
	}
}

func TestShellQuoteQuotesSpaces(t *testing.T) {
	got := ShellQuote("two words.mp4")
	want := "'two words.mp4'"
	if got != want {
		t.Fatalf("ShellQuote() = %q, want %q", got, want)
	}
}

func TestShellQuoteLeavesPlaceholderAlone(t *testing.T) {
	got := ShellQuote("<cliphub-pass-001.mp4>")
	if got != "'<cliphub-pass-001.mp4>'" {
		t.Fatalf("ShellQuote() = %q", got)
	}
}

func TestShellQuoteQuotesFilterLabels(t *testing.T) {
	got := ShellQuote("[0:v:0]reverse[vout]")
	want := "'[0:v:0]reverse[vout]'"
	if got != want {
		t.Fatalf("ShellQuote() = %q, want %q", got, want)
	}
}
