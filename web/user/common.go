package user

import "unicode"

type FormEntry struct {
	ID               string
	Type             string
	Label            string
	Ghost            string
	ShowPwdChecklist bool // show strength checklist below this field
}

// PwdReq represents one password strength requirement and whether it is met.
type PwdReq struct {
	Label string
	Met   bool
}

// CheckPwdReqs mirrors db.TestSecretStrength so the UI matches server
// validation exactly: 8+ chars, uppercase, lowercase, digit, symbol.
func CheckPwdReqs(password string) []PwdReq {
	var upper, lower, digit, sym bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsSymbol(r), unicode.IsPunct(r):
			sym = true
		}
	}
	return []PwdReq{
		{"8+ chars", len(password) >= 8},
		{"Uppercase", upper},
		{"Lowercase", lower},
		{"Number", digit},
		{"Symbol", sym},
	}
}

// PwdReqClass returns the CSS modifier class for a checklist chip.
func PwdReqClass(met bool, password string) string {
	if password == "" {
		return "pwd-req-neutral"
	}
	if met {
		return "pwd-req-pass"
	}
	return "pwd-req-fail"
}

// PwdReqIcon returns the Material Icon name for a checklist chip state.
func PwdReqIcon(met bool, password string) string {
	if password == "" {
		return "radio_button_unchecked"
	}
	if met {
		return "check"
	}
	return "close"
}

var (
	EntryFirstName   = FormEntry{ID: "firstName",    Type: "text",     Label: "First Name",       Ghost: "John"}
	EntryLastName    = FormEntry{ID: "lastName",     Type: "text",     Label: "Last Name",        Ghost: "Smith"}
	EntryEmailAddr   = FormEntry{ID: "emailAddr",    Type: "email",    Label: "Email Address",    Ghost: "jsmith27@depaul.edu"}
	EntrySecretAgain = FormEntry{ID: "secret_again", Type: "password", Label: "Re-type Password", Ghost: "Password1!"}

	RegisterFormEntries = []FormEntry{
		EntryFirstName,
		EntryLastName,
		EntryEmailAddr,
		{ID: "secret", Type: "password", Label: "Password", Ghost: "Password1!", ShowPwdChecklist: true},
		EntrySecretAgain,
	}

	LoginFormEntries = []FormEntry{
		EntryEmailAddr,
		{ID: "secret", Type: "password", Label: "Password", Ghost: "Password1!"},
	}
)
