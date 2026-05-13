package db

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestSetFirstNameValid(t *testing.T) {
	user := User{}
	err := user.SetFirstName("John")

	assert.NilError(t, err)
}

func TestSetFirstNameEmpty(t *testing.T) {
	user := User{}
	err := user.SetFirstName("")

	assert.ErrorContains(t, err, "first name cannot be blank")
}

func TestSetFirstNameTooLong(t *testing.T) {
	user := User{}
	long := string(make([]byte, MAX_NAME_LEN+1))
	err := user.SetFirstName(long)

	assert.ErrorContains(t, err, "first name cannot exceed")
}

func TestSetLastNameValid(t *testing.T) {
	user := User{}
	err := user.SetLastName("Doe")

	assert.NilError(t, err)
}

func TestSetLastNameEmpty(t *testing.T) {
	user := User{}
	err := user.SetLastName("")

	assert.ErrorContains(t, err, "last name cannot be blank")
}

func TestSetLastNameTooLong(t *testing.T) {
	user := User{}
	long := string(make([]byte, MAX_NAME_LEN+1))
	err := user.SetLastName(long)

	assert.ErrorContains(t, err, "last name cannot exceed")
}

func TestSecretStrengthValid(t *testing.T) {
	err := TestSecretStrength("Password1!")
	assert.NilError(t, err)
}

func TestSecretStrengthMissingUpper(t *testing.T) {
	err := TestSecretStrength("password1!")
	assert.ErrorContains(t, err, "an uppercase letter")
}

func TestSecretStrengthMissingLower(t *testing.T) {
	err := TestSecretStrength("PASSWORD1!")
	assert.ErrorContains(t, err, "a lowercase letter")
}

func TestSecretStrengthMissingDigit(t *testing.T) {
	err := TestSecretStrength("Password!")
	assert.ErrorContains(t, err, "a number")
}

func TestSecretStrengthMissingSymbol(t *testing.T) {
	err := TestSecretStrength("Password1")
	assert.ErrorContains(t, err, "a symbol")
}

func TestSecretStrengthTooShort(t *testing.T) {
	err := TestSecretStrength("Pass1!")
	assert.ErrorContains(t, err, "secret too short")
}

func TestSecretStrengthTooLong(t *testing.T) {
	err := TestSecretStrength("SomeReallyObscenelyLongPassword1!")
	assert.ErrorContains(t, err, "secret cannot exceed")
}

func TestValidateJWT(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	user := User{
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
	}
	tx.Save(&user)

	entry := user.IssueJWT()

	partial, _, err := ValidateJWT(entry.Token)
	assert.NilError(t, err)
	assert.Equal(t, partial.ID, user.ID.String())
}

func TestValidateJWTInvalid(t *testing.T) {
	_, _, err := ValidateJWT("invalid.jwt.token")
	assert.ErrorContains(t, err, "signature mismatch")
}

func TestValidateJWTMalformed(t *testing.T) {
	_, _, err := ValidateJWT("malformed.jwt-token")
	assert.ErrorContains(t, err, "malformed jwt")
}
