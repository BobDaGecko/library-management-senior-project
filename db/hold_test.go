package db

import (
	"testing"
	"time"
)

func TestHoldStatusCompleted(t *testing.T) {
	hold := Hold{
		FulfilledDate: time.Now(),
	}

	status, err := hold.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != HoldCompleted {
		t.Fatalf("expected %s, got %s", HoldCompleted, status)
	}
}

func TestHoldStatusCancelled(t *testing.T) {
	hold := Hold{
		CancelledDate: time.Now(),
	}

	status, err := hold.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != HoldCancelled {
		t.Fatalf("expected %s, got %s", HoldCancelled, status)
	}
}

func TestHoldStatusQueued(t *testing.T) {
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

	hold := Hold{
		User:   user,
		UserID: user.ID,
	}

	status, err := hold.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != HoldQueued {
		t.Fatalf("expected %s, got %s", HoldQueued, status)
	}
}

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
		t.Fatalf("unexpected error: %v", err)
	}
	if status != HoldRevoked {
		t.Fatalf("expected %s, got %s", HoldRevoked, status)
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
		t.Fatalf("unexpected error: %v", err)
	}
	if status != HoldPostponed {
		t.Fatalf("expected %s, got %s", HoldPostponed, status)
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
		t.Fatalf("unexpected error: %v", err)
	}
	if status != HoldPostponed {
		t.Fatalf("expected %s, got %s", HoldPostponed, status)
	}
}
