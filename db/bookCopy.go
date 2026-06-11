package db

import (
	"time"
)

var _ = Migrate(BookCopy{}, RepairLog{})

// An individual copy of a book
type BookCopy struct {
	BaseModel
	BookWork   BookWork
	BookWorkID string
	Barcode    string        // TO-DO: Replace with deterministic function
	Condition  ConditionFlag // TO-DO: Replace with function to derive this based on last return and repair dates
	Format     BookFmtFlag   // Hard-cover, paperback, etc.
	Status     CopyStatusFlag
}

// Repair log for individual copies of a book for audit purposes
type RepairLog struct {
	BaseModel
	BookCopyID     SqlUUID `gorm:"type:text"`
	Date           time.Time
	IncomingStatus CopyStatusFlag
	OutgoingStatus CopyStatusFlag
	TechnicianName string
}

type FormatsMap[T any] map[BookFmtFlag]T
type CopyList []BookCopy

func (arr CopyList) MapFormats() FormatsMap[CopyList] {
	ret := FormatsMap[CopyList]{}
	for _, e := range arr {
		_, exists := ret[e.Format]
		if !exists {
			ret[e.Format] = CopyList{e}
		} else {
			ret[e.Format] = append(ret[e.Format], e)
		}

	}
	return ret
}

func (c BookCopy) LoanHistory() ([]Loan, error) {
	db := Db()
	ret := []Loan{}
	status := db.Model(&Loan{}).
		Where(&Loan{
			BookCopyID: c.ID,
		}).
		Order("date_checkout DESC").
		Preload("User").
		Preload("BookCopy").
		Find(&ret)
	return ret, status.Error
}

// LoanStatus derives the circulation state of this copy. A copy that is
// checked out (CopyStatusPendingReturn) reports Unavailable or Overdue based
// on its active loan; only repair/discard states report Withdrawn.
func (c BookCopy) LoanStatus() (CopyLoanFlag, error) {
	switch c.Status {
	case CopyStatusPublic, CopyStatusPendingReturn:
		// fall through to the loan check below
	default:
		return CopyLoanWithdrawn, nil
	}

	// Only the newest loan matters here — never load the whole history.
	var active []Loan
	err := Db().Model(&Loan{}).
		Select("date_checkout", "date_returned").
		Where("book_copy_id = ?", c.ID).
		Where("date_returned = ?", NilTime).
		Order("date_checkout DESC").
		Limit(1).
		Find(&active).Error
	if err != nil {
		return CopyLoanWithdrawn, err
	}

	if len(active) > 0 {
		return active[0].Status().ToCopyStatus(), nil
	}

	if c.Status == CopyStatusPendingReturn {
		// Marked checked-out but no active loan (e.g. just returned and not
		// yet reshelved) — not available for a new checkout.
		return CopyLoanUnavailable, nil
	}
	return CopyLoanAvailable, nil
}

// HistoryEntry represents a unified "returned to library" event for a physical copy.
// It merges user check-in returns (from Loans, with User) and repair completions
// (from RepairLogs, with Technician; condition is assumed Good/Mint).
type HistoryEntry struct {
	UserID     SqlUUID       `gorm:"column:user_id"`
	User       User          `gorm:"-"`
	Technician string        `gorm:"column:technician"`
	Condition  ConditionFlag `gorm:"column:condition"`
	Date       time.Time     `gorm:"column:date"`
}

// History returns an intertwined, reverse-chronological (newest first) list of
// return-to-library events for this specific copy: loan returns + repair logs.
// It uses a single UNION SQL query for correct date ordering + limit/offset pagination.
// A separate count subquery gives the total for pagination UI.
// Users for loan entries are bulk-loaded after the main scan.
func (c BookCopy) History(page, perPage int) ([]HistoryEntry, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 25
	}
	offset := (page - 1) * perPage

	dbh := Db()
	nilT := NilTime

	// Main data: UNION for perfect interleave of the two sources, ordered + paged
	var entries []HistoryEntry
	dataSQL := `
		SELECT
			l.user_id AS user_id,
			'' AS technician,
			l.incoming_condition AS condition,
			l.date_returned AS date
		FROM loans l
		WHERE l.book_copy_id = ? AND l.date_returned != ?

		UNION ALL

		SELECT
			'' AS user_id,
			r.technician_name AS technician,
			? AS condition,
			r.date AS date
		FROM repair_logs r
		WHERE r.book_copy_id = ?

		ORDER BY date DESC
		LIMIT ? OFFSET ?
	`
	// Use ConditionGood for repairs (per spec: technician present => condition Good)
	err := dbh.Raw(dataSQL,
		c.ID, nilT,
		ConditionGood,
		c.ID,
		perPage, offset,
	).Scan(&entries).Error
	if err != nil {
		return nil, 0, err
	}

	// Bulk load Users for any loan-return entries (smart GORM select)
	userIDs := make([]SqlUUID, 0, len(entries))
	for i := range entries {
		if !entries[i].UserID.IsEmpty() {
			userIDs = append(userIDs, entries[i].UserID)
		}
	}
	if len(userIDs) > 0 {
		var users []User
		if err := dbh.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, 0, err
		}
		umap := make(map[SqlUUID]User, len(users))
		for _, u := range users {
			umap[u.ID] = u
		}
		for i := range entries {
			if u, ok := umap[entries[i].UserID]; ok {
				entries[i].User = u
			}
		}
	}

	// Total count via UNION subquery (for pagination)
	var total int64
	countSQL := `
		SELECT COUNT(*) FROM (
			SELECT l.id FROM loans l
			WHERE l.book_copy_id = ? AND l.date_returned != ?

			UNION ALL

			SELECT r.id FROM repair_logs r
			WHERE r.book_copy_id = ?
		) AS h
	`
	err = dbh.Raw(countSQL, c.ID, nilT, c.ID).Scan(&total).Error
	if err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}
