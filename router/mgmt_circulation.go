package router

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

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

func handleCheckoutPost(p *fail.RoutingParams) {
	if fail.Form(p) {
		return
	}

	userIDStr := p.Req.FormValue("user_id")
	copyIDStr := p.Req.FormValue("copy_id")

	var copy db.BookCopy
	db.Db().Preload("BookWork").Where("id = ?", copyIDStr).First(&copy)
	if copy.ID.IsEmpty() {
		http.Error(p.W, "copy not found", http.StatusBadRequest)
		return
	}

	loanStatus, err := copy.LoanStatus()
	if err != nil || loanStatus != db.CopyLoanAvailable {
		var user db.User
		db.Db().Where("id = ?", userIDStr).First(&user)
		loans, _ := user.CheckedOut()
		fail.Render(p, pages.CheckoutErrorState(user, len(loans), "This copy is no longer available. Please select another."))
		return
	}

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

	loan := db.Loan{
		BookCopyID:        copy.ID,
		UserID:            userUUID,
		DateCheckout:      time.Now(),
		OutgoingCondition: copy.Condition,
	}
	if res := db.Db().Create(&loan); res.Error != nil {
		http.Error(p.W, res.Error.Error(), http.StatusInternalServerError)
		return
	}

	db.Db().Model(&db.BookCopy{}).Where("id = ?", copy.ID).Update("status", db.CopyStatusPendingReturn)

	// Fulfill open hold if one exists for this patron + book work
	db.Db().Model(&db.Hold{}).
		Where("book_work_id = ? AND user_id = ? AND fulfilled_date = ? AND cancelled_date = ?",
			copy.BookWorkID, userUUID, db.NilTime, db.NilTime).
		Update("fulfilled_date", time.Now())

	dueDate := time.Now().Add(db.LOAN_DURATION)
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

	condition := db.ConditionGood
	if v, err := strconv.Atoi(conditionStr); err == nil {
		condition = db.ConditionFlag(v)
	}

	db.Db().Model(&db.Loan{}).Where("id = ?", loan.ID).Updates(map[string]interface{}{
		"date_returned":      time.Now(),
		"incoming_condition": condition,
	})

	newStatus := db.ReturnCopyStatus(condition)
	db.Db().Model(&db.BookCopy{}).Where("id = ?", loan.BookCopyID).Update("status", newStatus)

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
			db.Db().Model(&db.Hold{}).Where("id = ?", holdIDStr).Update("cancelled_date", time.Now())
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

			txn := db.Transaction{
				UserID:     fine.UserID,
				AmountPaid: fine.AmountRemaining,
				Date:       time.Now(),
			}
			db.Db().Create(&txn)
			db.Db().Model(&db.Fine{}).Where("id = ?", fine.ID).Update("amount_remaining", float32(0))
			fail.Render(p, pages.FinePaidBadge())
			return
		}
	}

	fail.Render(p, pages.MgmtFines(p))
}
