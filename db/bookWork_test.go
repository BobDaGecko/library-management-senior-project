package db

import (
	"slices"
	"testing"
)

func TestAvailableCopiesTotal(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	book := BookWork{
		ID:    "test-book-id",
		Title: "Test Book",
	}
	tx.Save(&book)

	copy1 := BookCopy{
		BookWorkID: book.ID,
		Format:     BookFmtPaperback,
		Status:     CopyStatusPublic,
	}
	copy2 := BookCopy{
		BookWorkID: book.ID,
		Format:     BookFmtPaperback,
		Status:     CopyStatusPublic,
	}
	tx.Save(&copy1)
	tx.Save(&copy2)

	counts, err := book.AvailableCopies(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := counts[BookFmtPaperback]
	if !ok {
		t.Fatalf("expected BookFmtPaperback in results")
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if result.Available != 2 {
		t.Fatalf("expected available 2, got %d", result.Available)
	}
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

	copy1 := BookCopy{
		BookWorkID: book.ID,
		Format:     BookFmtPaperback,
		Status:     CopyStatusPublic,
	}
	copy2 := BookCopy{
		BookWorkID: book.ID,
		Format:     BookFmtPaperback,
		Status:     CopyStatusPublic,
	}
	tx.Save(&copy1)
	tx.Save(&copy2)

	loan := Loan{
		BookCopyID:   copy1.ID,
		UserID:       user.ID,
		DateCheckout: NilTime,
		DateReturned: NilTime,
	}
	tx.Save(&loan)

	counts, err := book.AvailableCopies(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := counts[BookFmtPaperback]
	if !ok {
		t.Fatalf("expected BookFmtPaperback in results")
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if result.Available != 1 {
		t.Fatalf("expected available 1, got %d", result.Available)
	}
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
