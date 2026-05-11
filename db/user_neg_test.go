package db

import (
	"testing"
	"time"
)

func TestUserCheckedOutNone(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	user := User{
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
	}
	tx.Save(&user)

	checkedOut, err := user.CheckedOut()
	if err != nil {
		t.Fatalf("user checked out none: unexpected error: %v", err)
	}
	if len(checkedOut) != 0 {
		t.Fatalf("user checked out none: expected 0 loans, got %d", len(checkedOut))
	}
}

func TestUserNoOverdueBooks(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	user := User{
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
	}
	tx.Save(&user)

	book := BookWork{
		ID:    "test-book-id",
		Title: "Test Book",
	}
	tx.Save(&book)

	c := BookCopy{
		BookWorkID: book.ID,
		Format:     BookFmtPaperback,
		Status:     CopyStatusPublic,
	}
	tx.Save(&c)

	loan := Loan{
		BookCopyID:   c.ID,
		UserID:       user.ID,
		DateCheckout: time.Now().Add(-DAY),
		DateReturned: NilTime,
	}
	tx.Save(&loan)

	hasOverdue, err := user.HasOverdueBooks()
	if err != nil {
		t.Fatalf("user no overdue books: unexpected error: %v", err)
	}
	if hasOverdue {
		t.Fatalf("user no overdue books: expected false, got true")
	}
}
