package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"voxelprismatic/library-management-senior-project/db"
)

// copyPool tracks Public copies still available to be actively loaned, so we
// never double-loan the same physical copy. Iteration order is seeded for
// reproducibility.
type copyPool struct {
	byBook  map[string][]db.BookCopy
	bookIDs []string // books that (may) still have stock
}

func newCopyPool(books []db.BookWork, copiesByBook map[string][]db.BookCopy) *copyPool {
	p := &copyPool{byBook: map[string][]db.BookCopy{}}
	for _, b := range books { // stable order
		var pub []db.BookCopy
		for _, c := range copiesByBook[b.ID] {
			if c.Status == db.CopyStatusPublic {
				pub = append(pub, c)
			}
		}
		if len(pub) > 0 {
			p.byBook[b.ID] = pub
			p.bookIDs = append(p.bookIDs, b.ID)
		}
	}
	return p
}

// takeFromBook pops one available copy from a specific book.
func (p *copyPool) takeFromBook(id string) (db.BookCopy, bool) {
	cs := p.byBook[id]
	if len(cs) == 0 {
		return db.BookCopy{}, false
	}
	c := cs[len(cs)-1]
	p.byBook[id] = cs[:len(cs)-1]
	return c, true
}

// takeRandom pops one available copy from a random book that still has stock.
func (p *copyPool) takeRandom() (db.BookCopy, bool) {
	for len(p.bookIDs) > 0 {
		i := rng.Intn(len(p.bookIDs))
		id := p.bookIDs[i]
		if c, ok := p.takeFromBook(id); ok {
			return c, true
		}
		// exhausted: swap-remove this book id
		p.bookIDs[i] = p.bookIDs[len(p.bookIDs)-1]
		p.bookIDs = p.bookIDs[:len(p.bookIDs)-1]
	}
	return db.BookCopy{}, false
}

func seedCirculation(c *seedCast, books []db.BookWork, copiesByBook map[string][]db.BookCopy, zeroAvailable map[string]bool) {
	fmt.Println("Circulation: loans, holds, fines, transactions…")
	pool := newCopyPool(books, copiesByBook)

	var loanCount, holdCount, fineCount, txnCount int

	// ── 1. Zero-available hooks: loan out every Public copy of the chosen
	//      titles to random patrons (~30% overdue) so they read as 0-available.
	patrons := append([]db.User{}, c.filler...)
	patrons = append(patrons, c.personas["active"], c.personas["maxedout"])
	var zeroBookIDs []string
	for _, b := range books {
		if !zeroAvailable[b.ID] {
			continue
		}
		zeroBookIDs = append(zeroBookIDs, b.ID)
		for {
			copy, ok := pool.takeFromBook(b.ID)
			if !ok {
				break
			}
			u := patrons[rng.Intn(len(patrons))]
			checkout := daysAgo(rng.Intn(25) + 1)
			loanCount += loanActive(u, copy, checkout)
		}
	}

	// ── 2. Personas ────────────────────────────────────────────────────────
	// active: 3 current (on-time) loans
	for i := 0; i < 3; i++ {
		loanCount += loanRandomActive(c.personas["active"], pool, daysAgo(rng.Intn(10)+1))
	}

	// maxedout: 7/8 current loans
	for i := 0; i < 7; i++ {
		loanCount += loanRandomActive(c.personas["maxedout"], pool, daysAgo(rng.Intn(12)+1))
	}

	// overdue: 2 overdue loans + 1 outstanding late fine on one of them
	var overdueLoan db.Loan
	for i := 0; i < 2; i++ {
		l, ok := loanRandomOverdue(c.personas["overdue"], pool)
		if ok {
			overdueLoan = l
			loanCount++
		}
	}
	if !overdueLoan.ID.IsEmpty() {
		createFine(c.personas["overdue"], overdueLoan.ID, db.FineReasonLate, 6.50, daysAgo(3), 6.50)
		fineCount++
	}

	// paid: returned-history loan that incurred a late fine, since paid off
	if l, ok := loanReturnedLate(c.personas["paid"], pool); ok {
		loanCount++
		f := createFine(c.personas["paid"], l.ID, db.FineReasonLate, 4.00, daysAgo(20), 4.00)
		fineCount++
		payFine(c.personas["paid"], f, daysAgo(12))
		txnCount++
	}
	// a couple more clean returned loans for history depth
	for i := 0; i < 2; i++ {
		if loanReturnedOnTime(c.personas["paid"], pool) {
			loanCount++
		}
	}

	// holds: 4 queued holds on zero-available titles
	holdCount += giveHolds(c.personas["holds"], zeroBookIDs, copiesByBook, 4)

	// limited: 2 loans + an outstanding fine (justifies the limited status)
	for i := 0; i < 2; i++ {
		loanCount += loanRandomActive(c.personas["limited"], pool, daysAgo(rng.Intn(10)+1))
	}
	if l, ok := loanRandomOverdue(c.personas["limited"], pool); ok {
		loanCount++
		createFine(c.personas["limited"], l.ID, db.FineReasonLate, 8.25, daysAgo(2), 8.25)
		fineCount++
	}

	// locked: an outstanding lost-book fine (justifies the lock)
	if l, ok := loanReturnedLate(c.personas["locked"], pool); ok {
		loanCount++
		createFine(c.personas["locked"], l.ID, db.FineReasonLost, 24.99, daysAgo(30), 24.99)
		fineCount++
	}

	// ── 3. Collaborators: light activity, NO fines ──────────────────────────
	for _, u := range c.collaborators {
		n := rng.Intn(3) + 1 // 1-3 loans
		for i := 0; i < n; i++ {
			loanCount += loanRandomActive(u, pool, daysAgo(rng.Intn(12)+1))
		}
		if rng.Intn(100) < 30 && len(zeroBookIDs) > 0 {
			holdCount += giveHolds(u, zeroBookIDs, copiesByBook, 1)
		}
	}

	// ── 4. Filler: spread loans/returns/holds/fines for aggregate richness ──
	for _, u := range c.filler {
		switch r := rng.Intn(100); {
		case r < 40: // current loans
			for i := 0; i < rng.Intn(3)+1; i++ {
				loanCount += loanRandomActive(u, pool, daysAgo(rng.Intn(12)+1))
			}
		case r < 60: // overdue + fine
			if l, ok := loanRandomOverdue(u, pool); ok {
				loanCount++
				amt := float32(rng.Intn(8)) + 2.5
				createFine(u, l.ID, db.FineReasonLate, amt, daysAgo(rng.Intn(5)+1), amt)
				fineCount++
			}
		case r < 80: // returned history, some with paid fines
			if loanReturnedOnTime(u, pool) {
				loanCount++
			}
			if rng.Intn(100) < 40 {
				if l, ok := loanReturnedLate(u, pool); ok {
					loanCount++
					f := createFine(u, l.ID, db.FineReasonLate, 3.50, daysAgo(25), 3.50)
					fineCount++
					payFine(u, f, daysAgo(15))
					txnCount++
				}
			}
		default: // holds
			if len(zeroBookIDs) > 0 {
				holdCount += giveHolds(u, zeroBookIDs, copiesByBook, rng.Intn(2)+1)
			}
		}
	}

	fmt.Printf("  \x1b[32m✓ %d loans, %d holds, %d fines, %d transactions\x1b[0m\n",
		loanCount, holdCount, fineCount, txnCount)
}

// ── loan helpers ───────────────────────────────────────────────────────────

func loanActive(u db.User, copy db.BookCopy, checkout time.Time) int {
	loan := db.Loan{
		BookCopyID:        copy.ID,
		UserID:            u.ID,
		DateCheckout:      checkout,
		OutgoingCondition: copy.Condition,
	}
	if err := db.Db().Create(&loan).Error; err != nil {
		fmt.Printf("  \x1b[33mwarn: loan create: %v\x1b[0m\n", err)
		return 0
	}
	db.Db().Model(&db.BookCopy{}).Where("id = ?", copy.ID).Update("status", db.CopyStatusPendingReturn)
	return 1
}

func loanRandomActive(u db.User, pool *copyPool, checkout time.Time) int {
	copy, ok := pool.takeRandom()
	if !ok {
		return 0
	}
	return loanActive(u, copy, checkout)
}

func loanRandomOverdue(u db.User, pool *copyPool) (db.Loan, bool) {
	copy, ok := pool.takeRandom()
	if !ok {
		return db.Loan{}, false
	}
	checkout := daysAgo(int(db.LOAN_DURATION/db.DAY) + rng.Intn(10) + 3) // safely past due
	loan := db.Loan{
		BookCopyID:        copy.ID,
		UserID:            u.ID,
		DateCheckout:      checkout,
		OutgoingCondition: copy.Condition,
	}
	if err := db.Db().Create(&loan).Error; err != nil {
		return db.Loan{}, false
	}
	db.Db().Model(&db.BookCopy{}).Where("id = ?", copy.ID).Update("status", db.CopyStatusPendingReturn)
	return loan, true
}

func loanReturnedOnTime(u db.User, pool *copyPool) bool {
	copy, ok := pool.takeRandom()
	if !ok {
		return false
	}
	checkout := daysAgo(rng.Intn(60) + 20)
	returned := checkout.Add(time.Duration(rng.Intn(13)+1) * db.DAY)
	inCond := weightedCondition()
	loan := db.Loan{
		BookCopyID:        copy.ID,
		UserID:            u.ID,
		DateCheckout:      checkout,
		DateReturned:      returned,
		OutgoingCondition: copy.Condition,
		IncomingCondition: inCond,
	}
	if err := db.Db().Create(&loan).Error; err != nil {
		return false
	}
	db.Db().Model(&db.BookCopy{}).Where("id = ?", copy.ID).Update("status", db.ReturnCopyStatus(inCond))
	return true
}

func loanReturnedLate(u db.User, pool *copyPool) (db.Loan, bool) {
	copy, ok := pool.takeRandom()
	if !ok {
		return db.Loan{}, false
	}
	checkout := daysAgo(rng.Intn(40) + 40)
	returned := checkout.Add(db.LOAN_DURATION + time.Duration(rng.Intn(20)+3)*db.DAY) // returned late
	loan := db.Loan{
		BookCopyID:        copy.ID,
		UserID:            u.ID,
		DateCheckout:      checkout,
		DateReturned:      returned,
		OutgoingCondition: copy.Condition,
		IncomingCondition: db.ConditionGood,
	}
	if err := db.Db().Create(&loan).Error; err != nil {
		return db.Loan{}, false
	}
	db.Db().Model(&db.BookCopy{}).Where("id = ?", copy.ID).Update("status", db.CopyStatusPublic)
	return loan, true
}

// ── fine / transaction helpers ───────────────────────────────────────────────

func createFine(u db.User, loanID db.SqlUUID, reason db.FineReasonFlag, issued float32, date time.Time, remaining float32) db.Fine {
	f := db.Fine{
		UserID:          u.ID,
		LoanID:          loanID,
		IssueReason:     reason,
		IssueDate:       date,
		AmountIssued:    issued,
		AmountRemaining: remaining,
	}
	if err := db.Db().Create(&f).Error; err != nil {
		fmt.Printf("  \x1b[33mwarn: fine create: %v\x1b[0m\n", err)
	}
	return f
}

func payFine(u db.User, f db.Fine, date time.Time) {
	txn := db.Transaction{UserID: u.ID, AmountPaid: f.AmountRemaining, Date: date}
	db.Db().Create(&txn)
	db.Db().Model(&db.Fine{}).Where("id = ?", f.ID).Update("amount_remaining", float32(0))
}

// ── hold helper (raw SQL — book_work_id stores a Google Books string id) ─────

func giveHolds(u db.User, bookIDs []string, copiesByBook map[string][]db.BookCopy, n int) int {
	if len(bookIDs) == 0 {
		return 0
	}
	made := 0
	used := map[string]bool{}
	for attempts := 0; made < n && attempts < n*4; attempts++ {
		bid := bookIDs[rng.Intn(len(bookIDs))]
		if used[bid] {
			continue
		}
		used[bid] = true
		insertHold(u, bid, pickHoldFormat(copiesByBook[bid]), daysAgo(rng.Intn(14)+1))
		made++
	}
	return made
}

func insertHold(u db.User, bookWorkID string, format db.BookFmtFlag, requested time.Time) {
	now := time.Now()
	err := db.Db().Exec(
		`INSERT INTO holds (id, created_at, updated_at, book_work_id, user_id, format, requested_date, fulfilled_date, cancelled_date)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), now, now, bookWorkID, u.ID.String(), int(format), requested, db.NilTime, db.NilTime,
	).Error
	if err != nil {
		fmt.Printf("  \x1b[33mwarn: hold insert: %v\x1b[0m\n", err)
	}
}

func pickHoldFormat(copies []db.BookCopy) db.BookFmtFlag {
	if len(copies) > 0 {
		return copies[rng.Intn(len(copies))].Format
	}
	return db.BookFmtPaperback
}

// daysAgo returns a timestamp n days before now.
func daysAgo(n int) time.Time {
	return time.Now().Add(-time.Duration(n) * db.DAY)
}
