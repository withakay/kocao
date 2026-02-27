package namegen

import (
	"regexp"
	"testing"
)

var rfc1123Pattern = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)

func TestGenerate_ReturnsNonEmpty(t *testing.T) {
	name := Generate()
	if name == "" {
		t.Fatal("Generate() returned empty string")
	}
}

func TestGenerate_FormatIsAdjectiveHyphenNoun(t *testing.T) {
	for range 100 {
		name := Generate()
		parts := splitOnce(name, '-')
		if len(parts) != 2 {
			t.Fatalf("expected adjective-noun format, got %q", name)
		}
		if parts[0] == "" || parts[1] == "" {
			t.Fatalf("empty part in %q", name)
		}
	}
}

func TestGenerate_RFC1123Compliant(t *testing.T) {
	for range 200 {
		name := Generate()
		if len(name) > 63 {
			t.Fatalf("name exceeds 63 chars: %q (len=%d)", name, len(name))
		}
		if !rfc1123Pattern.MatchString(name) {
			t.Fatalf("name is not RFC 1123 compliant: %q", name)
		}
	}
}

func TestGenerate_LowercaseOnly(t *testing.T) {
	for range 100 {
		name := Generate()
		for _, c := range name {
			if c >= 'A' && c <= 'Z' {
				t.Fatalf("name contains uppercase: %q", name)
			}
		}
	}
}

func TestGenerate_Randomness(t *testing.T) {
	seen := make(map[string]bool)
	for range 50 {
		seen[Generate()] = true
	}
	// With ~100 adjectives * ~230 nouns = ~23000 combos,
	// 50 draws should produce at least 40 unique names.
	if len(seen) < 40 {
		t.Fatalf("expected at least 40 unique names from 50 draws, got %d", len(seen))
	}
}

func TestGenerateUnique_ReturnsUniqueName(t *testing.T) {
	taken := map[string]bool{"bold-tiger": true, "calm-river": true}
	existing := func(name string) bool { return taken[name] }

	name, err := GenerateUnique(existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if taken[name] {
		t.Fatalf("returned a taken name: %q", name)
	}
	if name == "" {
		t.Fatal("returned empty string")
	}
}

func TestGenerateUnique_RetriesOnCollision(t *testing.T) {
	attempts := 0
	existing := func(name string) bool {
		attempts++
		// Reject the first 5 attempts, accept the 6th
		return attempts <= 5
	}

	name, err := GenerateUnique(existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name == "" {
		t.Fatal("returned empty string")
	}
	if attempts < 6 {
		t.Fatalf("expected at least 6 attempts, got %d", attempts)
	}
}

func TestGenerateUnique_ErrorAfterMaxRetries(t *testing.T) {
	// Always reject
	existing := func(string) bool { return true }

	_, err := GenerateUnique(existing)
	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}
}

func TestGenerateUnique_NilExistingFunc(t *testing.T) {
	name, err := GenerateUnique(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name == "" {
		t.Fatal("returned empty string")
	}
}

// splitOnce splits s at the first occurrence of sep.
func splitOnce(s string, sep byte) []string {
	for i := range len(s) {
		if s[i] == sep {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func TestWords_NonEmpty(t *testing.T) {
	if len(adjectives) == 0 {
		t.Fatal("adjectives list is empty")
	}
	if len(nouns) == 0 {
		t.Fatal("nouns list is empty")
	}
}

func TestWords_AllLowercaseAlpha(t *testing.T) {
	alphaPattern := regexp.MustCompile(`^[a-z]+$`)
	for _, w := range adjectives {
		if !alphaPattern.MatchString(w) {
			t.Fatalf("adjective not lowercase alpha: %q", w)
		}
	}
	for _, w := range nouns {
		if !alphaPattern.MatchString(w) {
			t.Fatalf("noun not lowercase alpha: %q", w)
		}
	}
}

func TestWords_MaxLengthFitsRFC1123(t *testing.T) {
	maxAdj := 0
	for _, w := range adjectives {
		if len(w) > maxAdj {
			maxAdj = len(w)
		}
	}
	maxNoun := 0
	for _, w := range nouns {
		if len(w) > maxNoun {
			maxNoun = len(w)
		}
	}
	// adjective + "-" + noun <= 63
	if maxAdj+1+maxNoun > 63 {
		t.Fatalf("worst-case name length %d exceeds 63 (adj=%d, noun=%d)", maxAdj+1+maxNoun, maxAdj, maxNoun)
	}
}
