package db

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
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
	assert.NilError(t, err)
	assert.Equal(t, status, CopyLoanWithdrawn)
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
	assert.NilError(t, err)
	assert.Equal(t, status, CopyLoanAvailable)
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
	assert.NilError(t, err)
	assert.Equal(t, status, CopyLoanUnavailable)
}
