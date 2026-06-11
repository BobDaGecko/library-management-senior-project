package router

import (
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
	"voxelprismatic/library-management-senior-project/db"
	"voxelprismatic/library-management-senior-project/router/fail"
	"voxelprismatic/library-management-senior-project/web/pages"
)

// BookRouter handles the public /book tree (distinct from /management/books):
//
//	GET  /book/:id              -> BookDetailsPage
//	POST /book/:id/hold         -> HandleBookHold
//	POST /book/:id/hold/cancel  -> HandleBookCancelHold
func BookRouter(p *fail.RoutingParams) {
	id := ""
	if fail.Remainder(p) {
		id = p.Pop()
	}
	if id == "" {
		fail.Redirect(p)
		return
	}

	switch p.Pop() {
	case "hold":
		switch p.Pop() {
		case "cancel":
			HandleBookCancelHold(p, id)
		case "":
			if p.Req.Method != http.MethodPost {
				http.Redirect(p.W, p.Req, "/book/"+id, http.StatusSeeOther)
				return
			}
			HandleBookHold(p, id)
		default:
			http.Error(p.W, "Not Found", http.StatusNotFound)
		}
	case "":
		book, copiesMap, err := GetBookAndCopies(id)
		if err != nil {
			http.Error(p.W, "Book not found", http.StatusNotFound)
			return
		}
		var activeHold *pages.PatronHoldView
		if p.User != nil {
			if uid, err := db.ParseShort(p.User.ID); err == nil {
				var hold pages.PatronHoldView
				db.Db().Raw(`
					SELECT h.id as hold_id, h.book_work_id as book_work_raw_id, h.format,
					       h.requested_date, COALESCE(bw.title, '') as book_title
					FROM holds h
					LEFT JOIN book_works bw ON bw.id = h.book_work_id
					WHERE h.book_work_id = ? AND h.user_id = ? AND h.fulfilled_date = ? AND h.cancelled_date = ?
					  AND h.deleted_at IS NULL
					ORDER BY h.requested_date ASC
					LIMIT 1
				`, id, uid, db.NilTime, db.NilTime).Scan(&hold)
				if hold.HoldID != "" {
					activeHold = &hold
				}
			}
		}
		fail.Render(p, pages.BookDetailsPage(p, id, book, copiesMap, activeHold))
	default:
		http.Error(p.W, "Not Found", http.StatusNotFound)
	}
}

// HandleBookHold processes POST /book/:id/hold — places a hold for the logged-in patron.
func HandleBookHold(p *fail.RoutingParams, bookID string) {
	if p.User == nil {
		p.W.Header().Set("HX-Redirect", "/user/login")
		p.W.WriteHeader(http.StatusOK)
		return
	}

	userUUID, err := db.ParseShort(p.User.ID)
	if err != nil {
		http.Error(p.W, "Invalid session", http.StatusUnauthorized)
		return
	}

	if fail.Form(p) {
		return
	}
	formData := p.Form()

	fmtInt := 0
	fmt.Sscanf(formData["format"], "%d", &fmtInt)
	format := db.BookFmtFlag(fmtInt)

	// Duplicate check — use Count to avoid scanning book_work_id (raw Google Books string) into SqlUUID.
	var dupCount int64
	db.Db().Table("holds").
		Where("book_work_id = ? AND user_id = ? AND fulfilled_date = ? AND cancelled_date = ? AND deleted_at IS NULL",
			bookID, userUUID, db.NilTime, db.NilTime).
		Count(&dupCount)
	if dupCount > 0 {
		fail.Render(p, pages.HoldFragment("Hold already active", "", "ctag-teal"))
		return
	}

	// Create hold then patch book_work_id with the raw Google Books string
	// (bypasses SqlUUID.Value) — atomically, so a failed patch can't leave an
	// orphan hold with a zero book ID counted against the patron.
	hold := db.Hold{
		UserID:        userUUID,
		Format:        format,
		RequestedDate: time.Now(),
	}
	err := db.Db().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&hold).Error; err != nil {
			return err
		}
		return tx.Model(&db.Hold{}).
			Where("id = ?", hold.ID).
			Update("book_work_id", bookID).Error
	})
	if err != nil {
		http.Error(p.W, "Failed to place hold", http.StatusInternalServerError)
		return
	}

	fullUser, err := p.User.Fetch()
	if err == nil {
		if hasOverdue, _ := fullUser.HasOverdueBooks(); hasOverdue {
			fail.Render(p, pages.HoldFragment("Hold placed — currently postponed", "You have overdue items", "ctag-peach"))
			return
		}
		if loans, _ := fullUser.CheckedOut(); len(loans) >= db.LOAN_LIMIT {
			fail.Render(p, pages.HoldFragment("Hold placed — currently postponed", "Loan limit reached", "ctag-peach"))
			return
		}
	}

	fail.Render(p, pages.HoldFragment("Hold placed", hold.RequestedDate.Format("Jan 2, 2006"), "ctag-sky"))
}

// HandleBookCancelHold processes POST /book/:id/hold/cancel — cancels the patron's active hold
// and returns PlaceHoldFormFragment to restore the action area via HTMX swap.
func HandleBookCancelHold(p *fail.RoutingParams, bookID string) {
	if p.User == nil {
		p.W.Header().Set("HX-Redirect", "/user/login")
		p.W.WriteHeader(http.StatusOK)
		return
	}
	if p.Req.Method != http.MethodPost {
		http.Redirect(p.W, p.Req, "/book/"+bookID, http.StatusSeeOther)
		return
	}
	userUUID, err := db.ParseShort(p.User.ID)
	if err != nil {
		http.Error(p.W, "Invalid session", http.StatusUnauthorized)
		return
	}
	if fail.Form(p) {
		return
	}
	holdID := p.Form()["hold_id"]

	result := db.Db().Model(&db.Hold{}).
		Where("id = ? AND user_id = ?", holdID, userUUID).
		Update("cancelled_date", time.Now())
	if result.Error != nil {
		http.Error(p.W, "Failed to cancel hold", http.StatusInternalServerError)
		return
	}
	if result.RowsAffected == 0 {
		http.Error(p.W, "Hold not found or not authorized", http.StatusForbidden)
		return
	}

	_, copiesMap, err := GetBookAndCopies(bookID)
	if err != nil {
		http.Error(p.W, "Book not found", http.StatusNotFound)
		return
	}
	formats := pages.GetAvailableFormats(copiesMap)
	fail.Render(p, pages.PlaceHoldFormFragment(bookID, formats))
}
