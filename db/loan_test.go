package db

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestLoanFlags(t *testing.T) {
	assert.Equal(t, LoanStatusReturned.ToCopyStatus(), CopyLoanAvailable)
	assert.Equal(t, LoanStatusCheckedOut.ToCopyStatus(), CopyLoanUnavailable)
	assert.Equal(t, LoanStatusOverdue.ToCopyStatus(), CopyLoanOverdue)
}

func TestLoanStatusCheckedOut(t *testing.T) {
	loan := Loan{
		DateCheckout: time.Now().Add(-DAY),
		DateReturned: NilTime,
	}

	assert.Equal(t, loan.Status(), LoanStatusCheckedOut)
}

func TestLoanStatusOverdue(t *testing.T) {
	loan := Loan{
		DateCheckout: time.Now().Add(-LOAN_DURATION + 1),
		DateReturned: NilTime,
	}

	assert.Equal(t, loan.Status(), LoanStatusOverdue)
}

func TestLoanStatusReturned(t *testing.T) {
	loan := Loan{
		DateCheckout: time.Now().Add(-WEEK),
		DateReturned: time.Now(),
	}

	assert.Equal(t, loan.Status(), LoanStatusReturned)
}

func TestLoanReturn(t *testing.T) {
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

	err := loan.Return()
	assert.NilError(t, err)
	if err != nil {
		t.Fatalf("loan return: unexpected error: %v", err)
	}

	assert.Assert(t, !loan.DateReturned.IsZero())
}
