package router

import (
	"fmt"
	"net/http"
	"strings"
	"time"

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
//	/user/dashboard -> 301 redirect to /user
//	/user/logout    -> clears tok cookie and redirects to /
//	/user/loans     -> user_loans
//	/user/holds     -> user_holds
//	/user/fines     -> user_fines
//	/user/settings  -> user_settings
func UserRouter(p *fail.RoutingParams) {
	switch p.Pop() {
	case "login":
		HandleUserLogin(p)
	case "register":
		HandleUserRegister(p)
	case "logout":
		HandleUserLogout(p)
	case "loans":
		HandleUserLoans(p)
	case "holds":
		HandleUserHolds(p)
	case "fines":
		HandleUserFines(p)
	case "dashboard":
		HandleUserDashboard(p)
	case "settings":
		HandleUserSettings(p)
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
		fail.Render(p, pages.LoginPage(p))
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

	errs := map[string]error{}
	formData := p.Form()
	email := strings.ToLower(strings.TrimSpace(formData["emailAddr"]))

	userObj := db.User{}
	if email == "" {
		errs["emailAddr"] = fmt.Errorf("Email cannot be blank")
	} else if err := db.Db().Where("email = ?", email).First(&userObj).Error; err != nil {
		// Same message as a wrong password — never reveal whether an email is registered.
		errs["secret"] = fmt.Errorf("Invalid email or password")
	} else if !userObj.TestSecret(formData["secret"]) {
		errs["secret"] = fmt.Errorf("Invalid email or password")
	} else if userObj.Status == db.UserStatusLocked || userObj.Status == db.UserStatusDeleted {
		errs["secret"] = fmt.Errorf("This account is unavailable. Please contact the library")
	}

	if len(errs) > 0 {
		(fail.HTMX{
			Retarget: "#formEntry",
			Reswap:   "outerHTML",
		}).Apply(p)
		fail.Render(p, user.FormTable(user.LoginFormEntries, formData, errs))
		return
	}

	// IssueJWT saves the user, which also persists any legacy-hash upgrade
	// performed by TestSecret above.
	jwt, err := userObj.IssueJWT()
	if err != nil {
		http.Error(p.W, "Failed to start session", http.StatusInternalServerError)
		return
	}
	setAuthCookie(p, jwt.Token)
	(fail.HTMX{
		Redirect: "/",
	}).Apply(p)
}

// setAuthCookie sets the session cookie with protective flags. Secure is not
// set because the app serves plain HTTP in local deployments; add it if TLS
// terminates in front of the server.
func setAuthCookie(p *fail.RoutingParams, token string) {
	http.SetCookie(p.W, &http.Cookie{
		Name:     "tok",
		Value:    token,
		Path:     "/",
		MaxAge:   int(db.JWT_LIFETIME),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func HandleUserRegister(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	switch p.Req.Method {
	case http.MethodGet:
		fail.Render(p, pages.RegisterPage(p))
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

	userObj := db.User{}
	errs := map[string]error{}
	formData := p.Form()

	if err := userObj.SetFirstName(formData["firstName"]); err != nil {
		errs["firstName"] = err
	}

	if err := userObj.SetLastName(formData["lastName"]); err != nil {
		errs["lastName"] = err
	}

	if err := userObj.SetEmail(formData["emailAddr"]); err != nil {
		errs["emailAddr"] = err
	}

	if err := db.TestSecretStrength(formData["secret"]); err != nil {
		errs["secret"] = err
	}

	if err := userObj.SetSecret(formData["secret"], formData["secret_again"]); err != nil {
		errs["secret_again"] = err
	}

	if len(errs) == 0 {
		userObj.Roles = db.UserRolePublic
		if err := db.Db().Create(&userObj).Error; err != nil {
			// Most likely the unique-email constraint racing a concurrent signup.
			errs["emailAddr"] = fmt.Errorf("Email already in use")
		}
	}

	if len(errs) > 0 {
		(fail.HTMX{
			Retarget: "#formEntry",
			Reswap:   "outerHTML",
		}).Apply(p)
		fail.Render(p, user.FormTable(user.RegisterFormEntries, formData, errs))
		return
	}

	jwt, err := userObj.IssueJWT()
	if err != nil {
		http.Error(p.W, "Failed to start session", http.StatusInternalServerError)
		return
	}
	setAuthCookie(p, jwt.Token)
	(fail.HTMX{
		Redirect: "/",
	}).Apply(p)
}

// Account (and subs) require a logged-in user.

// requirePatron gates patron-only handlers: guests are redirected to login
// (via HX-Redirect for HTMX requests), and the session's short user ID is
// decoded to the SqlUUID form required for user_id queries. Returns ok=false
// when a response has already been written.
func requirePatron(p *fail.RoutingParams) (db.SqlUUID, bool) {
	if p.User == nil {
		if p.Req.Header.Get("HX-Request") == "true" {
			p.W.Header().Set("HX-Redirect", "/user/login")
			p.W.WriteHeader(http.StatusOK)
		} else {
			http.Redirect(p.W, p.Req, "/user/login", http.StatusSeeOther)
		}
		return db.SqlUUID{}, false
	}
	uid, err := db.ParseShort(p.User.ID)
	if err != nil {
		http.Error(p.W, "Invalid session", http.StatusUnauthorized)
		return db.SqlUUID{}, false
	}
	return uid, true
}

// patronHoldViews loads a patron's open holds via raw SQL. Hold.BookWorkID
// stores raw Google Books string IDs, which corrupt SqlUUID row scans — never
// Preload("BookWork") or Find() holds directly (see CLAUDE.md). bookID
// optionally filters to one work; limit <= 0 means unlimited.
func patronHoldViews(uid db.SqlUUID, bookID string, limit int) []pages.PatronHoldView {
	q := `
		SELECT h.id as hold_id, h.book_work_id as book_work_raw_id, h.format,
		       h.requested_date, COALESCE(bw.title, '') as book_title
		FROM holds h
		LEFT JOIN book_works bw ON bw.id = h.book_work_id
		WHERE h.user_id = ? AND h.fulfilled_date = ? AND h.cancelled_date = ?
		  AND h.deleted_at IS NULL`
	args := []any{uid, db.NilTime, db.NilTime}
	if bookID != "" {
		q += " AND h.book_work_id = ?"
		args = append(args, bookID)
	}
	q += " ORDER BY h.requested_date ASC"
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}

	var holds []pages.PatronHoldView
	db.Db().Raw(q, args...).Scan(&holds)
	return holds
}

func HandleUserAccount(p *fail.RoutingParams) {
	uid, ok := requirePatron(p)
	if !ok {
		return
	}
	if fail.Done(p) {
		return
	}
	var data pages.AccountData

	// Active loan count
	db.Db().Model(&db.Loan{}).
		Where("user_id = ? AND date_returned = ?", uid, db.NilTime).
		Count(&data.LoanCount)

	// Recent loans (up to 5)
	db.Db().Preload("BookCopy.BookWork").
		Where("user_id = ? AND date_returned = ?", uid, db.NilTime).
		Order("date_checkout DESC").
		Limit(5).
		Find(&data.RecentLoans)

	// Active hold count
	db.Db().Model(&db.Hold{}).
		Where("user_id = ? AND fulfilled_date = ? AND cancelled_date = ?", uid, db.NilTime, db.NilTime).
		Count(&data.HoldCount)

	// Active holds preview (up to 3)
	data.ActiveHolds = patronHoldViews(uid, "", 3)

	// Outstanding fines (join through loans — Fine.UserID is unreliable)
	db.Db().Table("fines").
		Joins("JOIN loans ON loans.id = fines.loan_id").
		Where("loans.user_id = ? AND fines.amount_remaining > 0", uid).
		Select("COALESCE(SUM(fines.amount_remaining), 0)").
		Scan(&data.FineBalance)

	fail.Render(p, pages.AccountPage(p, data))
}

func HandleUserLogout(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}
	// Revoke server-side so the token is dead even if a copy was captured.
	if cookie, err := p.Req.Cookie("tok"); err == nil {
		_ = db.RevokeJWT(cookie.Value)
	}
	http.SetCookie(p.W, &http.Cookie{
		Name:     "tok",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(p.W, p.Req, "/", http.StatusSeeOther)
}

func HandleUserDashboard(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}
	http.Redirect(p.W, p.Req, "/user", http.StatusMovedPermanently)
}

func HandleUserLoans(p *fail.RoutingParams) {
	uid, ok := requirePatron(p)
	if !ok {
		return
	}
	if fail.Done(p) {
		return
	}

	var loans []db.Loan
	db.Db().Preload("BookCopy.BookWork").
		Where("user_id = ? AND date_returned = ?", uid, db.NilTime).
		Order("date_checkout DESC").
		Find(&loans)

	fail.Render(p, pages.LoansPage(p, loans))
}

func HandleUserHolds(p *fail.RoutingParams) {
	uid, ok := requirePatron(p)
	if !ok {
		return
	}
	if fail.Done(p) {
		return
	}

	// Handle POST (cancel action)
	if p.Req.Method == http.MethodPost {
		HandleCancelHold(p, uid)
		return
	}

	holds := patronHoldViews(uid, "", 0)

	// Determine if all holds are postponed (overdue items or at loan limit).
	isPostponed := false
	if fullUser, err := p.User.Fetch(); err == nil {
		if overdue, _ := fullUser.HasOverdueBooks(); overdue {
			isPostponed = true
		} else if loans, _ := fullUser.CheckedOut(); len(loans) >= db.LOAN_LIMIT {
			isPostponed = true
		}
	}

	fail.Render(p, pages.HoldsPage(p, holds, isPostponed))
}

func HandleCancelHold(p *fail.RoutingParams, userUUID db.SqlUUID) {
	if fail.Form(p) {
		return
	}
	formData := p.Form()
	holdID := formData["hold_id"]

	// Targeted update with ownership check in WHERE — RowsAffected == 0 means not found or not owned.
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

	// HTMX: return empty response — hx-swap="outerHTML" on the <tr> removes it
	p.W.WriteHeader(http.StatusOK)
}

func HandleUserFines(p *fail.RoutingParams) {
	uid, ok := requirePatron(p)
	if !ok {
		return
	}
	if fail.Done(p) {
		return
	}

	var fines []db.Fine
	db.Db().
		Joins("JOIN loans ON loans.id = fines.loan_id").
		Where("loans.user_id = ?", uid).
		Preload("Loan.BookCopy.BookWork").
		Order("fines.issue_date DESC").
		Find(&fines)

	var totalOutstanding float32
	for _, f := range fines {
		totalOutstanding += f.AmountRemaining
	}

	fail.Render(p, pages.FinesPage(p, fines, totalOutstanding))
}

func HandleUserSettings(p *fail.RoutingParams) {
	if p.User == nil {
		http.Redirect(p.W, p.Req, "/user/login", http.StatusSeeOther)
		return
	}

	switch p.Pop() {
	case "":
		// GET /user/settings
		u, err := p.User.Fetch()
		if err != nil {
			http.Error(p.W, "User not found", http.StatusInternalServerError)
			return
		}
		fail.Render(p, pages.UserSettingsPage(p, u))
	case "profile":
		HandleSettingsProfile(p)
	case "password":
		HandleSettingsPassword(p)
	default:
		http.Error(p.W, "Not Found", http.StatusNotFound)
	}
}

func HandleSettingsProfile(p *fail.RoutingParams) {
	if fail.Form(p) {
		return
	}
	formData := p.Form()

	u, err := p.User.Fetch()
	if err != nil {
		http.Error(p.W, "User not found", http.StatusInternalServerError)
		return
	}

	errs := map[string]error{}
	if err := u.SetFirstName(formData["firstName"]); err != nil {
		errs["firstName"] = err
	}
	if err := u.SetLastName(formData["lastName"]); err != nil {
		errs["lastName"] = err
	}
	if err := u.SetEmail(formData["emailAddr"]); err != nil {
		errs["emailAddr"] = err
	}

	if len(errs) > 0 {
		fail.Render(p, pages.SettingsProfileForm(u, errs, false))
		return
	}

	if err := db.Db().Save(&u).Error; err != nil {
		http.Error(p.W, "Failed to save profile", http.StatusInternalServerError)
		return
	}
	fail.Render(p, pages.SettingsProfileForm(u, nil, true))
}

func HandleSettingsPassword(p *fail.RoutingParams) {
	if fail.Form(p) {
		return
	}
	formData := p.Form()

	u, err := p.User.Fetch()
	if err != nil {
		http.Error(p.W, "User not found", http.StatusInternalServerError)
		return
	}

	errs := map[string]error{}
	if !u.TestSecret(formData["currentPassword"]) {
		errs["currentPassword"] = fmt.Errorf("Incorrect current password")
	}
	if err := db.TestSecretStrength(formData["newPassword"]); err != nil {
		errs["newPassword"] = err
	}
	if err := u.SetSecret(formData["newPassword"], formData["confirmPassword"]); err != nil {
		errs["confirmPassword"] = err
	}

	if len(errs) > 0 {
		fail.Render(p, pages.SettingsPasswordForm(errs, false))
		return
	}

	if err := db.Db().Save(&u).Error; err != nil {
		http.Error(p.W, "Failed to save password", http.StatusInternalServerError)
		return
	}
	fail.Render(p, pages.SettingsPasswordForm(nil, true))
}
