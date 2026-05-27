package db

import "fmt"

type BookFmtFlag int

const (
	BookFmtPaperback     BookFmtFlag = 1 << iota // Physical paperback book
	BookFmtHardCover                             // Physical hard-cover book
	BookFmtPhysicalAudio                         // E.g. a physical MP3 player with the book preloaded
	BookFmtDigitalBook                           // E.g. Kindle
	BookFmtDigitalAudio                          // E.g. Audible
)

func (f BookFmtFlag) String() string {
	switch f {
	case BookFmtPaperback:
		return "BookFmtPaperback"
	case BookFmtHardCover:
		return "BookFmtHardCover"
	case BookFmtPhysicalAudio:
		return "BookFmtPhysicalAudio"
	case BookFmtDigitalBook:
		return "BookFmtDigitalBook"
	case BookFmtDigitalAudio:
		return "BookFmtDigitalAudio"
	default:
		return fmt.Sprintf("BookFmtFlag(%d)", f)
	}
}

// DisplayName returns a human-friendly lowercase label for use in UI sections/headers.
func (f BookFmtFlag) DisplayName() string {
	switch f {
	case BookFmtPaperback:
		return "paperback"
	case BookFmtHardCover:
		return "hardcover"
	case BookFmtPhysicalAudio:
		return "physical audio"
	case BookFmtDigitalBook:
		return "digital book"
	case BookFmtDigitalAudio:
		return "audiobook"
	default:
		return "unknown format"
	}
}

type ConditionFlag int

const (
	ConditionMint ConditionFlag = iota // New from the factory
	ConditionGood                      // No major wear, but some pages are bent
	ConditionFair                      // Light wear on corners, crease marks, but no torn pages
	ConditionPoor                      // Some tears, annotations, etc
	ConditionDead                      // Missing pages
	ConditionLost
)

func (f ConditionFlag) String() string {
	switch f {
	case ConditionMint:
		return "ConditionMint"
	case ConditionGood:
		return "ConditionGood"
	case ConditionFair:
		return "ConditionFair"
	case ConditionPoor:
		return "ConditionPoor"
	case ConditionDead:
		return "ConditionDead"
	case ConditionLost:
		return "ConditionLost"
	default:
		return fmt.Sprintf("ConditionFlag(%d)", f)
	}
}

// DisplayName returns human-friendly condition for copy cards.
func (f ConditionFlag) DisplayName() string {
	switch f {
	case ConditionMint:
		return "mint"
	case ConditionGood:
		return "good"
	case ConditionFair:
		return "fair"
	case ConditionPoor:
		return "poor"
	case ConditionDead:
		return "damaged"
	case ConditionLost:
		return "lost"
	default:
		return "unknown"
	}
}

type CopyStatusFlag int

const (
	CopyStatusPublic        CopyStatusFlag = iota // Open to the public
	CopyStatusPendingReturn                       // Book is checked out, waiting to be removed
	CopyStatusPendingAction                       // Book is returned, but waiting for action (repair or discard)
	CopyStatusRepairing                           // Book is being repaired (rebound, etc.)
	CopyStatusDiscarded                           // Book is discarded, possibly replaced
)

func (f CopyStatusFlag) String() string {
	switch f {
	case CopyStatusPublic:
		return "CopyStatusPublic"
	case CopyStatusPendingReturn:
		return "CopyStatusPendingReturn"
	case CopyStatusPendingAction:
		return "CopyStatusPendingAction"
	case CopyStatusRepairing:
		return "CopyStatusRepairing"
	case CopyStatusDiscarded:
		return "CopyStatusDiscarded"
	default:
		return fmt.Sprintf("CopyStatusFlag(%d)", f)
	}
}

// DisplayName returns human-friendly circulation status.
func (f CopyStatusFlag) DisplayName() string {
	switch f {
	case CopyStatusPublic:
		return "in circulation"
	case CopyStatusPendingReturn:
		return "pending return"
	case CopyStatusPendingAction:
		return "pending action"
	case CopyStatusRepairing:
		return "out for repair"
	case CopyStatusDiscarded:
		return "discarded"
	default:
		return "unknown"
	}
}

type CopyLoanFlag int

const (
	CopyLoanAvailable   CopyLoanFlag = 1 << iota // Open to the public
	CopyLoanUnavailable                          // Book is checked out
	CopyLoanOverdue                              // Book is checked out and overdue
	CopyLoanWithdrawn                            // Book is withdrawn from circulation until repairs are complete
)

func (f CopyLoanFlag) String() string {
	switch f {
	case CopyLoanAvailable:
		return "CopyLoanAvailable"
	case CopyLoanUnavailable:
		return "CopyLoanUnvailable"
	case CopyLoanOverdue:
		return "CopyLoanOverdue"
	case CopyLoanWithdrawn:
		return "CopyLoanWithdrawn"
	default:
		return fmt.Sprintf("CopyLoanFlag(%d)", f)
	}
}

// DisplayName returns human-friendly checkout/loan status matching UI spec (ready, checked out, etc).
func (f CopyLoanFlag) DisplayName() string {
	switch f {
	case CopyLoanAvailable:
		return "ready"
	case CopyLoanUnavailable:
		return "checked out"
	case CopyLoanOverdue:
		return "overdue"
	case CopyLoanWithdrawn:
		return "withheld"
	default:
		return "unknown"
	}
}

type FineReasonFlag int

const (
	FineReasonLate    FineReasonFlag = iota // Did not return the book on time
	FineReasonLost                          // Lost the book; fee for replacement
	FineReasonDamaged                       // Book was received damaged, eg torn pages
)

func (f FineReasonFlag) String() string {
	switch f {
	case FineReasonLate:
		return "FineReasonLate"
	case FineReasonLost:
		return "FineReasonLost"
	case FineReasonDamaged:
		return "FineReasonDamaged"
	default:
		return fmt.Sprintf("FineReasonFlag(%d)", f)
	}
}

type HoldStatusFlag int

const (
	HoldQueued    HoldStatusFlag = 1 << iota // User in queue
	HoldCancelled                            // User canceled hold
	HoldPostponed                            // User have outstanding charges and cannot check out books right now
	HoldCompleted                            // User has checked out the book
	HoldRevoked                              // User account has been deleted
)

func (f HoldStatusFlag) String() string {
	switch f {
	case HoldQueued:
		return "HoldQueued"
	case HoldCancelled:
		return "HoldCancelled"
	case HoldPostponed:
		return "HoldPostponed"
	case HoldCompleted:
		return "HoldCompleted"
	case HoldRevoked:
		return "HoldRevoked"
	default:
		return fmt.Sprintf("HoldStatus(%d)", f)
	}
}

type LoanStatusFlag int

const (
	LoanStatusReturned   LoanStatusFlag = 1 << iota // Book has been returned
	LoanStatusCheckedOut                            // Book is currently checked out
	LoanStatusOverdue                               // Book is checked out, but overdue
)

func (s LoanStatusFlag) String() string {
	switch s {
	case LoanStatusReturned:
		return "LoanStatusReturned"
	case LoanStatusCheckedOut:
		return "LoanStatusCheckedOut"
	case LoanStatusOverdue:
		return "LoanStatusOverdue"
	default:
		return fmt.Sprintf("LoanStatusFlag(%d)", s)
	}
}

type UserRoleFlag int

const (
	UserRoleNone      UserRoleFlag = 0         // Logged out user
	UserRolePublic    UserRoleFlag = 1 << iota // General public user
	UserRoleLibrarian                          // Librarian
	UserRoleAdmin                              // Administrator
)

func (f UserRoleFlag) String() string {
	switch f {
	case UserRoleNone:
		return "UserRoleNone"
	case UserRolePublic:
		return "UserRolePublic"
	case UserRoleLibrarian:
		return "UserRoleLibrarian"
	case UserRoleAdmin:
		return "UserRoleAdmin"
	default:
		return fmt.Sprintf("UserRoleFlag(%d)", f)
	}
}

type UserStatusFlag int

const (
	UserStatusActive  UserStatusFlag = 1 << iota // No outstanding issues
	UserStatusLimited                            // User has hit loan limit
	UserStatusLocked                             // TO-DO: Check if user has outstanding fees and remove this redundant lock
	UserStatusDeleted                            // For audit purposes; we may choose to anonymize any data
)

func (f UserStatusFlag) String() string {
	switch f {
	case UserStatusActive:
		return "UserStatusActive"
	case UserStatusLimited:
		return "UserStatusLimited"
	case UserStatusLocked:
		return "UserStatusLocked"
	case UserStatusDeleted:
		return "UserStatusDeleted"
	default:
		return fmt.Sprintf("UserStatusFlag(%d)", f)
	}
}
