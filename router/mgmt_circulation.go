package router

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"
	"voxelprismatic/library-management-senior-project/db"
	"voxelprismatic/library-management-senior-project/router/fail"
	"voxelprismatic/library-management-senior-project/web/pages"
)

// ── Checkout ──────────────────────────────────────────────────────────────────

func HandleManagementCheckout(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	if p.Req.Method == http.MethodPost {
		handleCheckoutPost(p)
		return
	}

	switch p.Param("step") {
	case "patron":
		users := searchPatrons(p.Param("q"))
		fail.Render(p, pages.CheckoutPatronResults(users))
	case "patron_selected":
		var user db.User
		db.Db().Where("id = ?", p.Param("user_id")).First(&user)
		loans, _ := user.CheckedOut()
		fail.Render(p, pages.CheckoutPatronSelected(user, len(loans)))
	case "copy":
		copies := searchAvailableCopies(p.Param("q"))
		fail.Render(p, pages.CheckoutCopyResults(copies, p.Param("user_id")))
	case "copy_selected":
		var user db.User
		db.Db().Where("id = ?", p.Param("user_id")).First(&user)
		var copy db.BookCopy
		db.Db().Preload("BookWork").Where("id = ?", p.Param("copy_id")).First(&copy)
		fail.Render(p, pages.CheckoutConfirmCard(user, copy, p.Param("user_id"), p.Param("copy_id")))
	default:
		fail.Render(p, pages.MgmtCheckout(p))
	}
}

var errCopyTaken = errors.New("copy no longer available")

func handleCheckoutPost(p *fail.RoutingParams) {
	if fail.Form(p) {
		return
	}

	userIDStr := p.Req.FormValue("user_id")
	copyIDStr := p.Req.FormValue("copy_id")

	var user db.User
	db.Db().Where("id = ?", userIDStr).First(&user)
	if user.ID.IsEmpty() {
		http.Error(p.W, "user not found", http.StatusBadRequest)
		return
	}

	userUUID, err := db.FromString(userIDStr)
	if err != nil {
		http.Error(p.W, "invalid user_id", http.StatusBadRequest)
		return
	}

	renderError := func(msg string) {
		loans, _ := user.CheckedOut()
		fail.Render(p, pages.CheckoutErrorState(user, len(loans), msg))
	}

	if user.Status == db.UserStatusLocked || user.Status == db.UserStatusDeleted {
		renderError("This account is " + user.Status.DisplayName() + " and cannot check out books.")
		return
	}

	var activeLoans int64
	if err := db.Db().Model(&db.Loan{}).
		Where("user_id = ? AND date_returned = ?", userUUID, db.NilTime).
		Count(&activeLoans).Error; err != nil {
		http.Error(p.W, "failed to check loan limit", http.StatusInternalServerError)
		return
	}
	if activeLoans >= db.LOAN_LIMIT {
		renderError(fmt.Sprintf("Patron has reached the loan limit (%d books).", db.LOAN_LIMIT))
		return
	}

	var copy db.BookCopy
	db.Db().Preload("BookWork").Where("id = ?", copyIDStr).First(&copy)
	if copy.ID.IsEmpty() {
		http.Error(p.W, "copy not found", http.StatusBadRequest)
		return
	}

	loanStatus, err := copy.LoanStatus()
	if err != nil || loanStatus != db.CopyLoanAvailable {
		renderError("This copy is no longer available. Please select another.")
		return
	}

	loan := db.Loan{
		BookCopyID:        copy.ID,
		UserID:            userUUID,
		DateCheckout:      time.Now(),
		OutgoingCondition: copy.Condition,
	}

	// One atomic unit: claim the copy, create the loan, fulfill the oldest
	// matching hold. The conditional status update doubles as the race guard —
	// a concurrent checkout of the same copy affects zero rows and aborts.
	err = db.Db().Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&db.BookCopy{}).
			Where("id = ? AND status = ?", copy.ID, db.CopyStatusPublic).
			Update("status", db.CopyStatusPendingReturn)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errCopyTaken
		}

		if err := tx.Create(&loan).Error; err != nil {
			return err
		}

		// Fulfill only the oldest open hold for this patron + work, not all of them.
		return tx.Exec(`
			UPDATE holds SET fulfilled_date = ? WHERE id IN (
				SELECT id FROM holds
				WHERE book_work_id = ? AND user_id = ?
				  AND fulfilled_date = ? AND cancelled_date = ? AND deleted_at IS NULL
				ORDER BY requested_date ASC LIMIT 1
			)`, time.Now(), copy.BookWorkID, userUUID, db.NilTime, db.NilTime).Error
	})
	if errors.Is(err, errCopyTaken) {
		renderError("This copy is no longer available. Please select another.")
		return
	}
	if err != nil {
		http.Error(p.W, "checkout failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	dueDate := loan.DateCheckout.Add(db.LOAN_DURATION)
	fail.Render(p, pages.CheckoutSuccess(
		fmt.Sprintf("%s %s", user.FirstName, user.LastName),
		copy.BookWork.Title,
		dueDate,
	))
}

func searchPatrons(q string) []db.User {
	var users []db.User
	if q == "" {
		return users
	}
	like := "%" + q + "%"
	db.Db().
		Where("email LIKE ? OR (first_name || ' ' || last_name) LIKE ?", like, like).
		Order("last_name asc, first_name asc").
		Limit(10).
		Find(&users)
	return users
}

func searchAvailableCopies(q string) []db.BookCopy {
	if q == "" {
		return nil
	}
	like := "%" + q + "%"
	var copies []db.BookCopy
	db.Db().
		Joins("JOIN book_works ON book_copies.book_work_id = book_works.id").
		Where("book_copies.status = ?", db.CopyStatusPublic).
		Where("book_works.title LIKE ? OR book_works.authors LIKE ?", like, like).
		Preload("BookWork").
		Limit(50).
		Find(&copies)

	var available []db.BookCopy
	for _, c := range copies {
		if s, err := c.LoanStatus(); err == nil && s == db.CopyLoanAvailable {
			available = append(available, c)
		}
	}
	return available
}

// ── Return ────────────────────────────────────────────────────────────────────

func HandleManagementReturn(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	if p.Req.Method == http.MethodPost {
		handleReturnPost(p)
		return
	}

	switch p.Param("step") {
	case "loan":
		loans := searchActiveLoans(p.Param("q"))
		fail.Render(p, pages.ReturnLoanResults(loans))
	case "loan_selected":
		var loan db.Loan
		db.Db().Preload("BookCopy.BookWork").Preload("User").Where("id = ?", p.Param("loan_id")).First(&loan)
		fail.Render(p, pages.ReturnConfirmCard(loan, p.Param("loan_id")))
	default:
		fail.Render(p, pages.MgmtReturn(p))
	}
}

func handleReturnPost(p *fail.RoutingParams) {
	if fail.Form(p) {
		return
	}

	loanIDStr := p.Req.FormValue("loan_id")
	conditionStr := p.Req.FormValue("condition")

	var loan db.Loan
	db.Db().Preload("BookCopy.BookWork").Preload("User").Where("id = ?", loanIDStr).First(&loan)
	if loan.ID.IsEmpty() {
		http.Error(p.W, "loan not found", http.StatusBadRequest)
		return
	}
	if !loan.DateReturned.IsZero() {
		// Replay guard: a double-submit must not re-stamp the return date or
		// reset a copy that has since moved to repair/discard.
		http.Error(p.W, "loan already returned", http.StatusConflict)
		return
	}

	v, err := strconv.Atoi(conditionStr)
	if err != nil || v < int(db.ConditionMint) || v > int(db.ConditionLost) {
		http.Error(p.W, "invalid condition", http.StatusBadRequest)
		return
	}
	condition := db.ConditionFlag(v)

	newStatus := db.ReturnCopyStatus(condition)
	err = db.Db().Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&db.Loan{}).
			Where("id = ? AND date_returned = ?", loan.ID, db.NilTime).
			Updates(map[string]any{
				"date_returned":      time.Now(),
				"incoming_condition": condition,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("loan already returned")
		}
		return tx.Model(&db.BookCopy{}).
			Where("id = ?", loan.BookCopyID).
			Update("status", newStatus).Error
	})
	if err != nil {
		http.Error(p.W, "return failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userName := fmt.Sprintf("%s %s", loan.User.FirstName, loan.User.LastName)
	bookTitle := loan.BookCopy.BookWork.Title

	if newStatus == db.CopyStatusPublic {
		fail.Render(p, pages.ReturnSuccess(userName, bookTitle))
	} else {
		fail.Render(p, pages.ReturnSuccessFlagged(userName, bookTitle))
	}
}

func searchActiveLoans(q string) []db.Loan {
	if q == "" {
		return nil
	}
	like := "%" + q + "%"
	var loans []db.Loan
	db.Db().
		Joins("JOIN book_copies ON loans.book_copy_id = book_copies.id").
		Joins("JOIN book_works ON book_copies.book_work_id = book_works.id").
		Where("loans.date_returned = ?", db.NilTime).
		Where("book_works.title LIKE ? OR book_works.authors LIKE ?", like, like).
		Preload("BookCopy.BookWork").
		Preload("User").
		Limit(50).
		Find(&loans)
	return loans
}

// ── Holds ─────────────────────────────────────────────────────────────────────

func HandleManagementHolds(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	if p.Req.Method == http.MethodPost {
		if fail.Form(p) {
			return
		}
		if p.Req.FormValue("action") == "cancel" {
			holdIDStr := p.Req.FormValue("hold_id")
			res := db.Db().Model(&db.Hold{}).
				Where("id = ? AND fulfilled_date = ? AND cancelled_date = ?", holdIDStr, db.NilTime, db.NilTime).
				Update("cancelled_date", time.Now())
			if res.Error != nil {
				http.Error(p.W, "failed to cancel hold", http.StatusInternalServerError)
				return
			}
			if res.RowsAffected == 0 {
				http.Error(p.W, "hold not found or already resolved", http.StatusConflict)
				return
			}
			fail.Render(p, pages.HoldCancelledBadge())
			return
		}
	}

	fail.Render(p, pages.MgmtHolds(p))
}

// ── Fines ─────────────────────────────────────────────────────────────────────

func HandleManagementFines(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	if p.Req.Method == http.MethodPost {
		if fail.Form(p) {
			return
		}
		if p.Req.FormValue("action") == "mark_paid" {
			fineIDStr := p.Req.FormValue("fine_id")

			var fine db.Fine
			db.Db().Where("id = ?", fineIDStr).First(&fine)
			if fine.ID.IsEmpty() {
				http.Error(p.W, "fine not found", http.StatusBadRequest)
				return
			}
			if fine.AmountRemaining <= 0 {
				// Already settled (double-submit) — idempotent success.
				fail.Render(p, pages.FinePaidBadge())
				return
			}

			// Fine.UserID is not reliably populated — derive the payer
			// through the loan, falling back to whatever the fine has.
			payerID := fine.UserID
			var loan db.Loan
			if err := db.Db().Where("id = ?", fine.LoanID).First(&loan).Error; err == nil {
				payerID = loan.UserID
			}

			txn := db.Transaction{
				UserID:     payerID,
				AmountPaid: fine.AmountRemaining,
				Date:       time.Now(),
			}
			err := db.Db().Transaction(func(tx *gorm.DB) error {
				if err := tx.Create(&txn).Error; err != nil {
					return err
				}
				return tx.Model(&db.Fine{}).Where("id = ?", fine.ID).Updates(map[string]any{
					"amount_remaining": float32(0),
					"user_id":          payerID,
				}).Error
			})
			if err != nil {
				http.Error(p.W, "failed to record payment: "+err.Error(), http.StatusInternalServerError)
				return
			}
			fail.Render(p, pages.FinePaidBadge())
			return
		}
	}

	fail.Render(p, pages.MgmtFines(p))
}
