package db

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestHoldStatusCompleted(t *testing.T) {
	hold := Hold{
		FulfilledDate: time.Now(),
	}

	status, err := hold.Status()
	assert.NilError(t, err)
	assert.Equal(t, status, HoldCompleted)
}

func TestHoldStatusCancelled(t *testing.T) {
	hold := Hold{
		CancelledDate: time.Now(),
	}

	status, err := hold.Status()
	assert.NilError(t, err)
	assert.Equal(t, status, HoldCancelled)
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
	assert.NilError(t, err)
	assert.Equal(t, status, HoldQueued)
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
	assert.NilError(t, err)
	assert.Equal(t, status, HoldRevoked)
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
	assert.NilError(t, err)
	assert.Equal(t, status, HoldPostponed)
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
	assert.NilError(t, err)
	assert.Equal(t, status, HoldPostponed)
}

func TestCancelHoldSetsCancelledDate(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	user := User{
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
		Status:    UserStatusActive,
	}
	tx.Save(&user)

	hold := Hold{
		UserID:        user.ID,
		RequestedDate: time.Now(),
	}
	tx.Create(&hold)

	tx.Model(&Hold{}).Where("id = ?", hold.ID).Update("cancelled_date", time.Now())

	var updated Hold
	tx.Where("id = ?", hold.ID).First(&updated)
	assert.Assert(t, !updated.CancelledDate.IsZero(), "CancelledDate should be set after cancel")
}

func TestCancelledHoldExcludedFromOpenQuery(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	user := User{
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
		Status:    UserStatusActive,
	}
	tx.Save(&user)

	hold := Hold{
		UserID:        user.ID,
		RequestedDate: time.Now(),
	}
	tx.Create(&hold)

	tx.Model(&Hold{}).Where("id = ?", hold.ID).Update("cancelled_date", time.Now())

	var openHolds []Hold
	tx.Where("fulfilled_date = ? AND cancelled_date = ?", NilTime, NilTime).Find(&openHolds)

	for _, h := range openHolds {
		if h.ID == hold.ID {
			t.Error("cancelled hold should not appear in open holds query")
		}
	}
}
