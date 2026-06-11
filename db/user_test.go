package db

import (
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
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
	assert.NilError(t, err)
	assert.Equal(t, len(checkedOut), 1)
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
	assert.NilError(t, err)
	assert.Equal(t, len(checkedOut), 0)
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
	assert.NilError(t, err)
	assert.Assert(t, hasOverdue)
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
	assert.NilError(t, err)
	assert.Assert(t, !hasOverdue)
}

func TestSecretBcrypt(t *testing.T) {
	u := User{Email: "bcrypt@example.com"}
	err := u.SetSecret("Password1!", "Password1!")
	assert.NilError(t, err)
	assert.Assert(t, strings.HasPrefix(u.Secret, "$2"), "new hashes must be bcrypt")
	assert.Assert(t, u.TestSecret("Password1!"))
	assert.Assert(t, !u.TestSecret("WrongPass1!"))
}

func TestSecretLegacyUpgrade(t *testing.T) {
	u := User{Email: "legacy@example.com"}
	u.Secret = u.legacyHashSecret("Password1!")
	assert.Assert(t, !strings.HasPrefix(u.Secret, "$2"))

	// A wrong password must not match nor alter the stored hash.
	assert.Assert(t, !u.TestSecret("WrongPass1!"))
	assert.Assert(t, !strings.HasPrefix(u.Secret, "$2"))

	// The correct password matches and upgrades the hash to bcrypt in memory.
	assert.Assert(t, u.TestSecret("Password1!"))
	assert.Assert(t, strings.HasPrefix(u.Secret, "$2"), "legacy verify must upgrade to bcrypt")
	assert.Assert(t, u.TestSecret("Password1!"))
}

func TestSetSecretMismatchAndWeak(t *testing.T) {
	u := User{Email: "weak@example.com"}
	err := u.SetSecret("Password1!", "Different1!")
	assert.ErrorContains(t, err, "passwords must match")

	// Weak passwords must surface the real strength error, not "must match".
	err = u.SetSecret("password1!", "password1!")
	assert.ErrorContains(t, err, "an uppercase letter")
}
