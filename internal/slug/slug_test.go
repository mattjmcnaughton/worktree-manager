package slug

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"add semantic indexing", "add-semantic-indexing"},
		{"  Add Semantic Indexing  ", "add-semantic-indexing"},
		{"AGE-4", "age-4"},
		{"gh#42", "gh-42"},
		{"foo/bar baz", "foo-bar-baz"},
		{"...weird...title", "weird-title"},
		{"already-good-slug", "already-good-slug"},
		{"WORK_TREE", "work-tree"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := Normalize(tc.in)
			if err != nil {
				t.Fatalf("Normalize(%q): unexpected error %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeErrorsOnEmptyResult(t *testing.T) {
	cases := []string{"", "   ", "---", "###"}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if _, err := Normalize(in); err == nil {
				t.Errorf("expected error for %q", in)
			}
		})
	}
}

func TestNormalizeCapsLengthAt60(t *testing.T) {
	in := "a-very-long-task-description-that-exceeds-the-sixty-character-cap-many-times-over"
	got, err := Normalize(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) > 60 {
		t.Errorf("expected len ≤ 60, got %d (%q)", len(got), got)
	}
}

func TestValidate(t *testing.T) {
	good := []string{"abc", "a1", "feature-x", "gh-42"}
	for _, s := range good {
		if err := Validate(s); err != nil {
			t.Errorf("Validate(%q): unexpected error %v", s, err)
		}
	}

	bad := []string{"", "-abc", "ABC", "foo bar", "a/b", "a..b"}
	for _, s := range bad {
		if err := Validate(s); err == nil {
			t.Errorf("Validate(%q): expected error", s)
		}
	}
}
