package db

import (
	"testing"
)

func TestSetFirstNameValid(t *testing.T) {
	user := User{}
	err := user.SetFirstName("John")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetFirstNameEmpty(t *testing.T) {
	user := User{}
	err := user.SetFirstName("")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSetFirstNameTooLong(t *testing.T) {
	user := User{}
	long := string(make([]byte, MAX_NAME_LEN+1))
	err := user.SetFirstName(long)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSetLastNameValid(t *testing.T) {
	user := User{}
	err := user.SetLastName("Doe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetLastNameEmpty(t *testing.T) {
	user := User{}
	err := user.SetLastName("")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSetLastNameTooLong(t *testing.T) {
	user := User{}
	long := string(make([]byte, MAX_NAME_LEN+1))
	err := user.SetLastName(long)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSecretStrengthValid(t *testing.T) {
	err := TestSecretStrength("Password1!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSecretStrengthMissingUpper(t *testing.T) {
	err := TestSecretStrength("password1!")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSecretStrengthMissingLower(t *testing.T) {
	err := TestSecretStrength("PASSWORD1!")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSecretStrengthMissingDigit(t *testing.T) {
	err := TestSecretStrength("Password!")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSecretStrengthMissingSymbol(t *testing.T) {
	err := TestSecretStrength("Password1")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSecretStrengthTooShort(t *testing.T) {
	err := TestSecretStrength("Pass1!")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSecretStrengthTooLong(t *testing.T) {
	long := "SomeReallyObscenelyLongPassword1!"
	err := TestSecretStrength(long)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if partial.ID != user.ID.String() {
		t.Fatalf("expected user ID %s, got %s", user.ID.String(), partial.ID)
	}
}

func TestValidateJWTInvalid(t *testing.T) {
	_, _, err := ValidateJWT("invalid.jwt.token")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
