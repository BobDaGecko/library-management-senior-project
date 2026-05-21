package router

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
	default:
		fail.Render(p, pages.ManagementHome(p))
	}
}

func ManagementBooksRouter(p *fail.RoutingParams) {
	if fail.Auth(p, db.UserRoleLibrarian) {
		return
	}

	switch p.Pop() {
	case "add":
		HandleManagementBooksAdd(p)
	default:
		fail.Redirect(p)
	}
}

func HandleManagementUsers(p *fail.RoutingParams) {
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
			// allow only standard assignable roles
			if newRole != db.UserRolePublic && newRole != db.UserRoleLibrarian && newRole != db.UserRoleAdmin {
				http.Error(p.W, "invalid role value", http.StatusBadRequest)
				return
			}
			res := db.Db().Model(&db.User{}).Where("id = ?", userID).Update("roles", newRole)
			if res.Error != nil {
				http.Error(p.W, res.Error.Error(), http.StatusInternalServerError)
				return
			}
			// return updated select (swaps in place via HTMX)
			fail.Render(p, pages.UserRoleSelect(userID, newRole))
			return
		}
		// unknown post - fallthrough or error
	}

	// GET list (or default)
	page := parsePageParam(p)
	if p.Req.Header.Get("HX-Request") == "true" {
		fail.Render(p, pages.MgmtUsersTable(page))
		return
	}
	fail.Render(p, pages.MgmtUsers(p, page))
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

func HandleManagementBooksAdd(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	switch p.Req.Method {
	case http.MethodGet:
		query := p.Req.URL.Query().Get("q")
		fail.Render(p, pages.BookMgmtSearchFull(query))
	case http.MethodPost:
		if fail.Form(p) {
			return
		}
		query := p.Req.Form.Get("q")
		p.W.Header().Set(
			"hx-replace-url",
			fmt.Sprintf("%s?q=%s", p.Req.URL.String(), url.QueryEscape(query)),
		)
		fail.Render(p, pages.BookMgmtSearchGrid(query))

	}
}