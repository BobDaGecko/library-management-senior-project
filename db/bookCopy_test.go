package db

import (
	"testing"
	"time"
)

func TestBookCopyLoanStatusWithdrawn(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	book := BookWork{
		ID:    "test-book-id",
		Title: "Test Book",
	}
	tx.Save(&book)

	c := BookCopy{
		BookWorkID: book.ID,
		Format:     BookFmtPaperback,
		Status:     CopyStatusRepairing,
	}
	tx.Save(&c)

	status, err := c.LoanStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != CopyLoanWithdrawn {
		t.Fatalf("expected %s, got %s", CopyLoanWithdrawn, status)
	}
}

func TestBookCopyLoanStatusAvailable(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

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

	status, err := c.LoanStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != CopyLoanAvailable {
		t.Fatalf("expected %s, got %s", CopyLoanAvailable, status)
	}
}

func TestBookCopyLoanStatusLatest(t *testing.T) {
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

	status, err := c.LoanStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != CopyLoanUnvailable {
		t.Fatalf("expected %s, got %s", CopyLoanUnvailable, status)
	}
}
