package db

import "regexp"

// Pragmatic email shape check. Deliberately loose — the previous pattern
// required 3+ characters in both the local part and the domain label,
// rejecting real addresses like jo@gmail.com or me@x.io.
var emailRegexp = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9][A-Za-z0-9.\-]*\.[A-Za-z]{2,}$`)

const (
	MAX_NAME_LEN   = 48
	MAX_EMAIL_LEN  = 72
	MIN_SECRET_LEN = 8
	MAX_SECRET_LEN = 32
)
