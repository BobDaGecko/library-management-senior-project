package db

import (
	"testing"
	"time"
)

func TestHoldStatusRevoked(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	user := User{
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
		Status:    UserStatusDeleted,
	}
	tx.Save(&user)

	hold := Hold{
		User:   user,
		UserID: user.ID,
	}

	status, err := hold.Status()
	if err != nil {
		t.Fatalf("hold revoked: unexpected error: %v", err)
	}
	if status != HoldRevoked {
		t.Fatalf("hold revoked: expected %s, got %s", HoldRevoked, status)
	}
}

func TestHoldStatusPostponedOverdue(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	user := User{
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
		Status:    UserStatusActive,
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

	hold := Hold{
		User:   user,
		UserID: user.ID,
	}

	status, err := hold.Status()
	if err != nil {
		t.Fatalf("hold postponed overdue: unexpected error: %v", err)
	}
	if status != HoldPostponed {
		t.Fatalf("hold postponed overdue: expected %s, got %s", HoldPostponed, status)
	}
}

func TestHoldStatusPostponedTooManyLoans(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	user := User{
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
		Status:    UserStatusActive,
	}
	tx.Save(&user)

	book := BookWork{
		ID:    "test-book-id",
		Title: "Test Book",
	}
	tx.Save(&book)

	for i := 0; i < LOAN_LIMIT; i++ {
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
	}

	hold := Hold{
		User:   user,
		UserID: user.ID,
	}

	status, err := hold.Status()
	if err != nil {
		t.Fatalf("hold postponed too many loans: unexpected error: %v", err)
	}
	if status != HoldPostponed {
		t.Fatalf("hold postponed too many loans: expected %s, got %s", HoldPostponed, status)
	}
}
