package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
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
	if p.SubPtr < len(p.FullPath) {
		userIDStr := p.Pop()
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
			res := db.Db().Model(&db.User{}).Where("id = ?", userID).Update("roles", newRole)
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
			loanIDStr := p.Req.FormValue("loan_id")
			if loanIDStr == "" {
				http.Error(p.W, "missing loan_id", http.StatusBadRequest)
				return
			}

			var fine db.Fine
			db.Db().Where("loan_id = ?", loanIDStr).Order("created_at desc").First(&fine)

			if fine.ID.IsEmpty() {
				fine = db.Fine{
					IssueReason:     db.FineReasonLate,
					IssueDate:       time.Now(),
					AmountIssued:    5.0,
					AmountRemaining: 5.0,
				}
				if u, err := uuid.Parse(loanIDStr); err == nil {
					fine.LoanID = db.SqlUUID{UUID: u}
				}
				db.Db().Create(&fine)
			}

			if fine.AmountWaived == 0 {
				fine.AmountWaived = fine.AmountIssued
				fine.WaivedReason = "Waived by librarian via overdue page"
				if p.User != nil {
					if u, err := uuid.Parse(p.User.ID); err == nil {
						fine.WaivedBy = db.SqlUUID{UUID: u}
					}
				}
				db.Db().Save(&fine)
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
