package router

import (
	"fmt"
	"net/http"
	"strings"

	"voxelprismatic/library-management-senior-project/db"
	"voxelprismatic/library-management-senior-project/router/fail"
	"voxelprismatic/library-management-senior-project/web/pages"
	"voxelprismatic/library-management-senior-project/web/user"
)

// UserRouter handles the /user tree:
//
//	/user/login     -> public_login page (LoginPage)
//	/user/register  -> public_register page (RegisterPage)
//	/user           -> user_account page (AccountPage)
//	/user/loans     -> user_loans
//	/user/holds     -> user_holds
//	/user/fines     -> user_fines
//	/user/saved     -> user_saved (included for completeness with account dashboard).
func UserRouter(p *fail.RoutingParams) {
	switch p.Pop() {
	case "login":
		HandleUserLogin(p)
	case "register":
		HandleUserRegister(p)
	case "loans":
		HandleUserLoans(p)
	case "holds":
		HandleUserHolds(p)
	case "fines":
		HandleUserFines(p)
	case "saved":
		HandleUserSaved(p)
	case "":
		// /user exact -> account dashboard
		HandleUserAccount(p)
	default:
		fail.Redirect(p)
	}
}

func HandleUserLogin(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	switch p.Req.Method {
	case http.MethodGet:
		fail.Render(p, user.Login())
	case http.MethodPost:
		HandleUserLoginPost(p)
	default:
		http.Error(p.W, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleUserLoginPost(p *fail.RoutingParams) {
	if fail.Form(p) {
		return
	}

	formData := p.Form()
	email := strings.TrimSpace(formData["email"])
	secret := formData["password"]

	userObj := db.User{Email: email}
	if email == "" {
		fail.Render(p, pages.LoginPage())
		return
	} else if err := db.Db().Where(&userObj).First(&userObj).Error; err != nil {
		fail.Render(p, pages.LoginPage())
		return
	}

	if !userObj.TestSecret(secret) {
		fail.Render(p, pages.LoginPage())
		return
	}

	jwt := userObj.IssueJWT()
	p.W.Header().Set("Set-Cookie", fmt.Sprintf("tok=%s; path=/", jwt.Token))
	http.Redirect(p.W, p.Req, "/user", http.StatusSeeOther)
}

func HandleUserRegister(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	switch p.Req.Method {
	case http.MethodGet:
		fail.Render(p, user.Register())
	case http.MethodPost:
		HandleUserRegisterPost(p)
	default:
		http.Error(p.W, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleUserRegisterPost(p *fail.RoutingParams) {
	if fail.Form(p) {
		return
	}

	form := p.Form()
	fullName := strings.TrimSpace(form["name"])
	email := strings.TrimSpace(form["email"])
	secret := form["password"]
	secret2 := form["confirm_password"]

	// crude name split (new templates use single "name"; db model uses first+last)
	first, last := "", ""
	if fullName != "" {
		parts := strings.SplitN(fullName, " ", 2)
		first = parts[0]
		if len(parts) > 1 {
			last = parts[1]
		}
	}

	userObj := db.User{}
	if err := userObj.SetFirstName(first); err != nil {
		fail.Render(p, pages.RegisterPage())
		return
	}
	if err := userObj.SetLastName(last); err != nil {
		fail.Render(p, pages.RegisterPage())
		return
	}
	if err := userObj.SetEmail(email); err != nil {
		fail.Render(p, pages.RegisterPage())
		return
	}
	if err := db.TestSecretStrength(secret); err != nil {
		fail.Render(p, pages.RegisterPage())
		return
	}
	if err := userObj.SetSecret(secret, secret2); err != nil {
		fail.Render(p, pages.RegisterPage())
		return
	}

	jwt := userObj.IssueJWT()
	p.W.Header().Set("Set-Cookie", fmt.Sprintf("tok=%s; path=/", jwt.Token))
	http.Redirect(p.W, p.Req, "/user", http.StatusSeeOther)
}

// Account (and subs) require a logged-in user.

func HandleUserAccount(p *fail.RoutingParams) {
	if p.User == nil {
		http.Redirect(p.W, p.Req, "/user/login", http.StatusSeeOther)
		return
	}
	if fail.Done(p) {
		fail.Render(p, pages.AccountPage())
		return
	}
	fail.Redirect(p)
}

func HandleUserLoans(p *fail.RoutingParams) {
	if p.User == nil {
		http.Redirect(p.W, p.Req, "/user/login", http.StatusSeeOther)
		return
	}
	if fail.Done(p) {
		return
	}
	fail.Render(p, pages.LoansPage())
}

func HandleUserHolds(p *fail.RoutingParams) {
	if p.User == nil {
		http.Redirect(p.W, p.Req, "/user/login", http.StatusSeeOther)
		return
	}
	if fail.Done(p) {
		return
	}
	fail.Render(p, pages.HoldsPage())
}

func HandleUserFines(p *fail.RoutingParams) {
	if p.User == nil {
		http.Redirect(p.W, p.Req, "/user/login", http.StatusSeeOther)
		return
	}
	if fail.Done(p) {
		return
	}
	fail.Render(p, pages.FinesPage())
}

func HandleUserSaved(p *fail.RoutingParams) {
	if p.User == nil {
		http.Redirect(p.W, p.Req, "/user/login", http.StatusSeeOther)
		return
	}
	if fail.Done(p) {
		return
	}
	fail.Render(p, pages.SavedPage())
}
