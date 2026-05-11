package db

import (
	"slices"
	"testing"
)

func TestBookWorkTagParsing(t *testing.T) {
	book := BookWork{
		ID:    "test-book-id",
		Title: "Test Book",
		Categories: SqlStringList{
			"Fiction / Mystery",
			"Fiction / Detective / Amateur Sleuth",
		},
	}

	tags := book.Tags()

	expected := []string{"Fiction", "Mystery", "Detective", "Amateur Sleuth"}
	for _, e := range expected {
		if !slices.Contains(tags, e) {
			t.Fatalf("tag parsing: expected tag %q to be present, got %v", e, tags)
		}
	}
}

func TestBookWorkTagParsingNoDuplicates(t *testing.T) {
	book := BookWork{
		ID:    "test-book-id",
		Title: "Test Book",
		Categories: SqlStringList{
			"Fiction / Mystery",
			"Fiction / Detective",
		},
	}

	tags := book.Tags()

	seen := map[string]bool{}
	for _, tag := range tags {
		if seen[tag] {
			t.Fatalf("tag parsing: duplicate tag found: %q", tag)
		}
		seen[tag] = true
	}
}
