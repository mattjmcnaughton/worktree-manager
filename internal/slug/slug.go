// Package slug normalizes user-provided task identifiers.
package slug

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const MaxLength = 60

var validSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Normalize converts free text, ticket IDs, or `gh#NN` refs into a valid slug.
// Behavior: lowercase, replace any non-alphanumeric run with "-", trim leading
// and trailing "-", cap at MaxLength.
func Normalize(input string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(input))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > MaxLength {
		out = strings.TrimRight(out[:MaxLength], "-")
	}
	if out == "" {
		return "", fmt.Errorf("input %q normalizes to an empty slug", input)
	}
	if err := Validate(out); err != nil {
		return "", err
	}
	return out, nil
}

// Validate enforces the canonical slug shape.
func Validate(s string) error {
	if s == "" {
		return errors.New("slug is empty")
	}
	if len(s) > MaxLength {
		return fmt.Errorf("slug %q exceeds max length of %d", s, MaxLength)
	}
	if !validSlug.MatchString(s) {
		return fmt.Errorf("slug %q must match ^[a-z0-9][a-z0-9-]*$", s)
	}
	return nil
}
