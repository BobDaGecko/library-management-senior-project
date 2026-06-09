package pages

import (
	"testing"

	"gotest.tools/v3/assert"
	"voxelprismatic/library-management-senior-project/db"
)

func TestSearchBooksEmptyQuery(t *testing.T) {
	tx := db.TestDb()
	defer tx.Rollback()

	tx.Save(&db.BookWork{ID: "search-empty-1", Title: "Alpha Book"})
	tx.Save(&db.BookWork{ID: "search-empty-2", Title: "Beta Book"})

	results := SearchBooks(SearchFilter{})
	assert.Assert(t, len(results) >= 2, "expected at least 2 results for empty query, got %d", len(results))
}

func TestSearchBooksByTitle(t *testing.T) {
	tx := db.TestDb()
	defer tx.Rollback()

	tx.Save(&db.BookWork{ID: "search-title-1", Title: "The Great Gatsby"})
	tx.Save(&db.BookWork{ID: "search-title-2", Title: "To Kill a Mockingbird"})

	results := SearchBooks(SearchFilter{Q: "gatsby"})
	assert.Equal(t, len(results), 1, "expected 1 result for 'gatsby'")
	assert.Equal(t, results[0].ID, "search-title-1")
}

func TestSearchBooksByAuthor(t *testing.T) {
	tx := db.TestDb()
	defer tx.Rollback()

	tx.Save(&db.BookWork{
		ID:      "search-author-1",
		Title:   "Neuromancer",
		Authors: db.SqlStringList{"William Gibson"},
	})
	tx.Save(&db.BookWork{
		ID:      "search-author-2",
		Title:   "Dune",
		Authors: db.SqlStringList{"Frank Herbert"},
	})

	results := SearchBooks(SearchFilter{Q: "gibson"})
	assert.Equal(t, len(results), 1, "expected 1 result for author 'gibson'")
	assert.Equal(t, results[0].ID, "search-author-1")
}

func TestSearchBooksFormatFilter(t *testing.T) {
	tx := db.TestDb()
	defer tx.Rollback()

	tx.Save(&db.BookWork{ID: "search-fmt-1", Title: "Paperback Book"})
	tx.Save(&db.BookWork{ID: "search-fmt-2", Title: "Hardcover Book"})
	tx.Save(&db.BookCopy{
		BookWorkID: "search-fmt-1",
		Format:     db.BookFmtPaperback,
		Status:     db.CopyStatusPublic,
	})
	tx.Save(&db.BookCopy{
		BookWorkID: "search-fmt-2",
		Format:     db.BookFmtHardCover,
		Status:     db.CopyStatusPublic,
	})

	results := SearchBooks(SearchFilter{Format: db.BookFmtPaperback})
	found := false
	for _, r := range results {
		if r.ID == "search-fmt-1" {
			found = true
		}
		if r.ID == "search-fmt-2" {
			t.Errorf("hardcover book should not appear in paperback filter")
		}
	}
	assert.Assert(t, found, "paperback book should appear in results")
}

func TestSearchBooksGenreFilter(t *testing.T) {
	tx := db.TestDb()
	defer tx.Rollback()

	tx.Save(&db.BookWork{
		ID:         "search-genre-1",
		Title:      "A Science Fiction Novel",
		Categories: db.SqlStringList{"Science Fiction / Space Opera"},
	})
	tx.Save(&db.BookWork{
		ID:         "search-genre-2",
		Title:      "A Mystery Novel",
		Categories: db.SqlStringList{"Fiction / Mystery"},
	})

	results := SearchBooks(SearchFilter{Genre: "science fiction"})
	assert.Equal(t, len(results), 1, "expected 1 sci-fi result")
	assert.Equal(t, results[0].ID, "search-genre-1")
}

func TestSearchBooksSortTitle(t *testing.T) {
	tx := db.TestDb()
	defer tx.Rollback()

	tx.Save(&db.BookWork{ID: "search-sort-1", Title: "Zephyr Tales"})
	tx.Save(&db.BookWork{ID: "search-sort-2", Title: "Apple Stories"})
	tx.Save(&db.BookWork{ID: "search-sort-3", Title: "Middle Ground"})

	results := SearchBooks(SearchFilter{Sort: "title"})
	assert.Assert(t, len(results) >= 3, "expected at least 3 results")

	// Verify alphabetical ordering among our seeded books
	positions := map[string]int{}
	for i, r := range results {
		positions[r.ID] = i
	}
	applePos, appleOk := positions["search-sort-2"]
	middlePos, middleOk := positions["search-sort-3"]
	zephyrPos, zephyrOk := positions["search-sort-1"]

	assert.Assert(t, appleOk && middleOk && zephyrOk, "all seeded books should be present")
	assert.Assert(t, applePos < middlePos, "Apple should come before Middle (title sort)")
	assert.Assert(t, middlePos < zephyrPos, "Middle should come before Zephyr (title sort)")
}

func TestSearchBooksAvailabilityFilter(t *testing.T) {
	tx := db.TestDb()
	defer tx.Rollback()

	// A book with an available copy (no active loan)
	tx.Save(&db.BookWork{ID: "search-avail-1", Title: "Available Book"})
	tx.Save(&db.BookCopy{
		BookWorkID: "search-avail-1",
		Format:     db.BookFmtPaperback,
		Status:     db.CopyStatusPublic,
	})

	// A book with all copies checked out (active loan, date_returned = NilTime)
	tx.Save(&db.BookWork{ID: "search-avail-2", Title: "Checked Out Book"})
	checkedOutCopy := db.BookCopy{
		BookWorkID: "search-avail-2",
		Format:     db.BookFmtPaperback,
		Status:     db.CopyStatusPublic,
	}
	tx.Save(&checkedOutCopy)
	checkedOutUser := db.User{FirstName: "Test", LastName: "User", Email: "avail-test@example.com"}
	tx.Save(&checkedOutUser)
	tx.Save(&db.Loan{
		BookCopyID:   checkedOutCopy.ID,
		UserID:       checkedOutUser.ID,
		DateCheckout: db.NilTime,
		DateReturned: db.NilTime,
	})

	results := SearchBooks(SearchFilter{Available: true})
	foundAvail := false
	for _, r := range results {
		if r.ID == "search-avail-1" {
			foundAvail = true
		}
		if r.ID == "search-avail-2" {
			t.Errorf("fully checked-out book should not appear in availability filter")
		}
	}
	assert.Assert(t, foundAvail, "book with available copy should appear in results")
}
