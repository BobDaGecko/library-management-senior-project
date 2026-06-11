package db

import (
	"time"
)

const (
	DAY           time.Duration = time.Hour * 24
	WEEK                        = DAY * 7
	LOAN_DURATION               = WEEK * 2 // Maximum amount of time before a book is considered overdue
	LOAN_LIMIT                  = 8        // Maximum amount of books that can be checked out at once
)

var _ = Migrate(Loan{})

// Loan record, all checked-out books are like this
// Note: When a continuing hold is satisfied, the previous loan should be
// marked as returned and a new loan should be issued.
type Loan struct {
	BaseModel
	BookCopy          BookCopy
	BookCopyID        SqlUUID `gorm:"type:text"`
	User              User
	UserID            SqlUUID `gorm:"type:text"`
	DateCheckout      time.Time
	DateReturned      time.Time
	OutgoingCondition ConditionFlag
	IncomingCondition ConditionFlag
}

func (s LoanStatusFlag) ToCopyStatus() CopyLoanFlag {
	switch s {
	case LoanStatusReturned:
		return CopyLoanAvailable
	case LoanStatusCheckedOut:
		return CopyLoanUnavailable
	case LoanStatusOverdue:
		return CopyLoanOverdue
	default:
		panic("unreachable")
	}
}

func (l Loan) Status() LoanStatusFlag {
	if !l.DateReturned.IsZero() {
		return LoanStatusReturned
	}
	if l.DateCheckout.Add(LOAN_DURATION).Before(time.Now()) {
		return LoanStatusOverdue
	}
	return LoanStatusCheckedOut
}

// ReturnCopyStatus maps a returned book's condition to the appropriate copy status.
// Poor/Dead/Lost conditions require librarian inspection before re-circulation.
func ReturnCopyStatus(c ConditionFlag) CopyStatusFlag {
	switch c {
	case ConditionPoor, ConditionDead, ConditionLost:
		return CopyStatusPendingAction
	default:
		return CopyStatusPublic
	}
}

// Marks a book as returned. Uses a targeted update — Save() on a loan with
// preloaded associations would cascade stale BookCopy/BookWork/User writes.
func (c *Loan) Return() error {
	c.DateReturned = time.Now()
	return Db().Model(&Loan{}).
		Where("id = ?", c.ID).
		Update("date_returned", c.DateReturned).Error
}
