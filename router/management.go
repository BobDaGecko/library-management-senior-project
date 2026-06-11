package router

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"
	"voxelprismatic/library-management-senior-project/db"
	"voxelprismatic/library-management-senior-project/router/fail"
	"voxelprismatic/library-management-senior-project/web/pages"
)

func ManagementRouter(p *fail.RoutingParams) {
	if fail.Auth(p, db.UserRoleLibrarian) {
		return
	}

	switch p.Pop() {
	case "books":
		ManagementBooksRouter(p)
	case "users":
		HandleManagementUsers(p)
	case "checkout":
		HandleManagementCheckout(p)
	case "return":
		HandleManagementReturn(p)
	case "holds":
		HandleManagementHolds(p)
	case "fines":
		HandleManagementFines(p)
	case "transactions":
		HandleManagementTransactions(p)
	case "overdue":
		HandleManagementOverdue(p)
	default:
		fail.Render(p, pages.ManagementHome(p))
	}
}

// Note: Book-specific routers and handlers have been moved to mgmt_book.go
// for better organization (ManagementBooksRouter, HandleManagementBooksAdd, HandleManagementBook, etc.)

func HandleManagementUsers(p *fail.RoutingParams) {
	if fail.Auth(p, db.UserRoleLibrarian) {
		return
	}

	// Support for user detail subpage: /management/users/{id}
	if fail.Remainder(p) {
		userIDStr := p.Pop()
		if userIDStr == "" {
			// Trailing slash (/management/users/) → canonical redirect,
			// not a detail page for an empty ID.
			fail.Redirect(p)
			return
		}
		HandleManagementUserDetail(p, userIDStr)
		return
	}

	if fail.Done(p) {
		return
	}

	if p.Req.Method == http.MethodPost {
		action := p.Req.FormValue("action")
		if action == "update_role" {
			userID := p.Req.FormValue("user_id")
			roleStr := p.Req.FormValue("role")
			roleVal, err := strconv.Atoi(roleStr)
			if err != nil {
				http.Error(p.W, "invalid role", http.StatusBadRequest)
				return
			}
			newRole := db.UserRoleFlag(roleVal)
			if newRole != db.UserRolePublic && newRole != db.UserRoleLibrarian && newRole != db.UserRoleAdmin {
				http.Error(p.W, "invalid role value", http.StatusBadRequest)
				return
			}

			var target db.User
			if err := db.Db().Where("id = ?", userID).First(&target).Error; err != nil {
				http.Error(p.W, "user not found", http.StatusNotFound)
				return
			}

			// Librarians may not grant admin, touch an admin's account, or
			// change their own role — those require a (different) admin.
			isAdmin := p.User.Roles >= db.UserRoleAdmin
			if (newRole == db.UserRoleAdmin || target.Roles >= db.UserRoleAdmin) && !isAdmin {
				http.Error(p.W, "only an admin can assign or modify the admin role", http.StatusForbidden)
				return
			}
			if target.ID.Short() == p.User.ID {
				http.Error(p.W, "you cannot change your own role", http.StatusForbidden)
				return
			}

			res := db.Db().Model(&db.User{}).Where("id = ?", target.ID).Update("roles", newRole)
			if res.Error != nil {
				http.Error(p.W, res.Error.Error(), http.StatusInternalServerError)
				return
			}
			fail.Render(p, pages.UserRoleSelect(userID, newRole))
			return
		}
	}

	page := parsePageParam(p)
	if p.Req.Header.Get("HX-Request") == "true" {
		fail.Render(p, pages.MgmtUsersTable(page))
		return
	}
	fail.Render(p, pages.MgmtUsers(p, page))
}

func HandleManagementUserDetail(p *fail.RoutingParams, userID string) {
	if fail.Done(p) {
		return
	}
	tab := p.Param("tab")
	if tab == "" {
		tab = "loans"
	}
	fail.Render(p, pages.UserDetail(p, userID, tab))
}

func HandleManagementOverdue(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	if p.Req.Method == http.MethodPost {
		action := p.Req.FormValue("action")
		if action == "waive" {
			loanID, err := db.FromString(p.Req.FormValue("loan_id"))
			if err != nil || loanID.IsEmpty() {
				http.Error(p.W, "invalid loan_id", http.StatusBadRequest)
				return
			}

			var loan db.Loan
			if err := db.Db().Where("id = ?", loanID).First(&loan).Error; err != nil {
				http.Error(p.W, "loan not found", http.StatusNotFound)
				return
			}

			// p.User.ID is the 22-char Short() form — never uuid.Parse it.
			waivedBy := db.SqlUUID{}
			if u, err := db.ParseShort(p.User.ID); err == nil {
				waivedBy = u
			}
			const waivedReason = "Waived by librarian via overdue page"

			var fine db.Fine
			err = db.Db().Where("loan_id = ?", loanID).Order("created_at desc").First(&fine).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				// No fine exists yet — record the waive as an audit entry
				// with nothing owed, instead of minting a new debt.
				fine = db.Fine{
					UserID:          loan.UserID,
					LoanID:          loanID,
					IssueReason:     db.FineReasonLate,
					IssueDate:       time.Now(),
					AmountIssued:    5.0,
					AmountRemaining: 0,
					AmountWaived:    5.0,
					WaivedReason:    waivedReason,
					WaivedBy:        waivedBy,
				}
				if err := db.Db().Create(&fine).Error; err != nil {
					http.Error(p.W, "failed to record waive", http.StatusInternalServerError)
					return
				}
			case err != nil:
				http.Error(p.W, "failed to look up fine", http.StatusInternalServerError)
				return
			case fine.AmountRemaining > 0:
				res := db.Db().Model(&db.Fine{}).Where("id = ?", fine.ID).Updates(map[string]any{
					"amount_waived":    fine.AmountWaived + fine.AmountRemaining,
					"amount_remaining": float32(0),
					"waived_reason":    waivedReason,
					"waived_by":        waivedBy,
				})
				if res.Error != nil {
					http.Error(p.W, "failed to waive fine", http.StatusInternalServerError)
					return
				}
			}

			fail.Render(p, pages.WaiveButtonDisabled())
			return
		}
	}

	fail.Render(p, pages.MgmtOverdue(p))
}

func HandleManagementTransactions(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	page := parsePageParam(p)
	if p.Req.Header.Get("HX-Request") == "true" {
		fail.Render(p, pages.MgmtTransactionsTable(page))
		return
	}
	fail.Render(p, pages.MgmtTransactions(p, page))
}

func parsePageParam(p *fail.RoutingParams) int {
	pageStr := p.Param("page")
	if pageStr == "" {
		return 1
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 1
	}
	return page
}
