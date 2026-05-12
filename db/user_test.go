package db

import (
	"testing"
	"time"
)

func TestUserCheckedOut(t *testing.T) {
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

	checkedOut, err := user.CheckedOut()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(checkedOut) != 1 {
		t.Fatalf("expected 1 loan, got %d", len(checkedOut))
	}
}

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
		t.Fatalf("unexpected error: %v", err)
	}
	if len(checkedOut) != 0 {
		t.Fatalf("expected 0 loans, got %d", len(checkedOut))
	}
}

func TestUserHasOverdueBooks(t *testing.T) {
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
		DateCheckout: time.Now().Add(-LOAN_DURATION * 2),
		DateReturned: NilTime,
	}
	tx.Save(&loan)

	hasOverdue, err := user.HasOverdueBooks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasOverdue {
		t.Fatalf("expected true, got false")
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
		t.Fatalf("unexpected error: %v", err)
	}
	if hasOverdue {
		t.Fatalf("expected false, got true")
	}
}
