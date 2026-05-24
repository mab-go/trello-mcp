package version

import "testing"

func TestShortCommit(t *testing.T) {
	got := ShortCommit()
	if got != "000000000000" {
		t.Errorf("ShortCommit() = %q, want %q", got, "000000000000")
	}
	if len(got) != 12 {
		t.Errorf("len = %d, want 12", len(got))
	}
}

func TestShortCommitShortValue(t *testing.T) {
	orig := Commit
	Commit = "abc"
	defer func() { Commit = orig }()

	got := ShortCommit()
	if got != "abc" {
		t.Errorf("ShortCommit() = %q, want %q", got, "abc")
	}
}
