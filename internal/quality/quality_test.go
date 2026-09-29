package quality

import "testing"

func TestNormalizeTextKeepsPTBRLetters(t *testing.T) {
	got := normalizeText("Conteúdo jurídico: ação, revisão e intenção.")
	want := "conteúdo jurídico ação revisão e intenção"
	if got != want {
		t.Fatalf("normalizeText() = %q, want %q", got, want)
	}
}
