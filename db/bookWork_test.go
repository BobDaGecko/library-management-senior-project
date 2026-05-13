package db

import (
	"slices"
	"testing"

	"gotest.tools/v3/assert"
)

func TestAvailableCopiesTotal(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	book := BookWork{
		ID:    "test-book-id",
		Title: "Test Book",
	}
	tx.Save(&book)

	copies := 2
	for i := 0; i < copies; i++ {
		tx.Save(&BookCopy{
			BookWorkID: book.ID,
			Format:     BookFmtPaperback,
			Status:     CopyStatusPublic,
		})
	}

	counts, err := book.AvailableCopies(true)
	assert.NilError(t, err)

	result, ok := counts[BookFmtPaperback]
	if !ok {
		t.Fatalf("expected BookFmtPaperback in results")
	}

	assert.Equal(t, result.Total, copies, "total copies")
	assert.Equal(t, result.Available, copies, "available copies")
}

func TestAvailableCopiesWithLoan(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	user := User{
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
	}
	tx.Save(&user)

	book := BookWork{
		ID:    "test-book-id-2",
		Title: "Test Book 2",
	}
	tx.Save(&book)

	primaryCopy := BookCopy{
		BookWorkID: book.ID,
		Format:     BookFmtPaperback,
		Status:     CopyStatusPublic,
	}
	tx.Save(&primaryCopy)

	extraCopies := 2
	for i := 0; i < extraCopies; i++ {
		tx.Save(&BookCopy{
			BookWorkID: book.ID,
			Format:     BookFmtPaperback,
			Status:     CopyStatusPublic,
		})
	}

	loan := Loan{
		BookCopyID:   primaryCopy.ID,
		UserID:       user.ID,
		DateCheckout: NilTime,
		DateReturned: NilTime,
	}
	tx.Save(&loan)

	counts, err := book.AvailableCopies(true)
	assert.NilError(t, err)

	result, ok := counts[BookFmtPaperback]
	if !ok {
		t.Fatalf("expected BookFmtPaperback in results")
	}

	assert.Equal(t, result.Total, extraCopies+1, "total copies")
	assert.Equal(t, result.Available, extraCopies, "available copies")
}

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
			t.Fatalf("expected tag %q to be present, got %v", e, tags)
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
			t.Fatalf("duplicate tag found: %q", tag)
		}
		seen[tag] = true
	}
}
