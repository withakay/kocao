// Package namegen generates human-readable names in adjective-noun format
// (e.g. "elegant-galileo"). Names are RFC 1123 label compliant: lowercase
// alphanumeric and hyphens, at most 63 characters.
package namegen

import (
	"errors"
	"math/rand/v2"
)

const maxRetries = 100

// Generate returns a random adjective-noun name like "bold-tiger".
// The result is always RFC 1123 compliant (lowercase, max 63 chars).
func Generate() string {
	adj := adjectives[rand.IntN(len(adjectives))]
	noun := nouns[rand.IntN(len(nouns))]
	return adj + "-" + noun
}

// GenerateUnique generates a name that does not collide with any existing name.
// The existing function returns true if the name is already taken.
// If existing is nil, no collision check is performed.
// Returns an error if a unique name cannot be found within 100 attempts.
func GenerateUnique(existing func(string) bool) (string, error) {
	for range maxRetries {
		name := Generate()
		if existing == nil || !existing(name) {
			return name, nil
		}
	}
	return "", errors.New("namegen: failed to generate unique name after 100 attempts")
}
