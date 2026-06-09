package router

import (
	"fmt"
	"net/http"
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
	userObj := db.User{Email: formData["emailAddr"]}
	if userObj.Email == "" {
		errs["emailAddr"] = fmt.Errorf("Email cannot be blank")
	} else if err := db.Db().Where(&userObj).First(&userObj).Error; err != nil {
		errs["emailAddr"] = fmt.Errorf("Email not found")
	}

	if len(errs) == 0 && !userObj.TestSecret(formData["secret"]) {
		errs["secret"] = fmt.Errorf("Incorrect password")
	}

	if len(errs) > 0 {
		(fail.HTMX{
			Retarget: "#formEntry",
			Reswap:   "outerHTML",
		}).Apply(p)
		fail.Render(p, user.FormTable(user.LoginFormEntries, formData, errs))
		return
	}

	jwt := userObj.IssueJWT()
	p.W.Header().Set("Set-Cookie", fmt.Sprintf("tok=%s; path=/", jwt.Token))
	(fail.HTMX{
		Redirect: "/",
	}).Apply(p)
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

	if len(errs) > 0 {
		(fail.HTMX{
			Retarget: "#formEntry",
			Reswap:   "outerHTML",
		}).Apply(p)
		fail.Render(p, user.FormTable(user.RegisterFormEntries, formData, errs))
		return
	}

	jwt := userObj.IssueJWT()
	p.W.Header().Set("Set-Cookie", fmt.Sprintf("tok=%s; path=/", jwt.Token))
	(fail.HTMX{
		Redirect: "/",
	}).Apply(p)
}

// Account (and subs) require a logged-in user.

func HandleUserAccount(p *fail.RoutingParams) {
	if p.User == nil {
		http.Redirect(p.W, p.Req, "/user/login", http.StatusSeeOther)
		return
	}
	if fail.Done(p) {
		return
	}

	uid, err := db.ParseShort(p.User.ID)
	if err != nil {
		http.Error(p.W, "Invalid session", http.StatusUnauthorized)
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

	// Active holds preview (up to 3) — raw SQL to avoid SqlUUID scan on Google Books IDs.
	db.Db().Raw(`
		SELECT h.id as hold_id, h.book_work_id as book_work_raw_id, h.format,
		       h.requested_date, COALESCE(bw.title, '') as book_title
		FROM holds h
		LEFT JOIN book_works bw ON bw.id = h.book_work_id
		WHERE h.user_id = ? AND h.fulfilled_date = ? AND h.cancelled_date = ?
		  AND h.deleted_at IS NULL
		ORDER BY h.requested_date ASC LIMIT 3
	`, uid, db.NilTime, db.NilTime).Scan(&data.ActiveHolds)

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
	p.W.Header().Set("Set-Cookie", "tok=; Max-Age=0; path=/")
	http.Redirect(p.W, p.Req, "/", http.StatusSeeOther)
}

func HandleUserDashboard(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}
	http.Redirect(p.W, p.Req, "/user", http.StatusMovedPermanently)
}

func HandleUserLoans(p *fail.RoutingParams) {
	if p.User == nil {
		http.Redirect(p.W, p.Req, "/user/login", http.StatusSeeOther)
		return
	}
	if fail.Done(p) {
		return
	}

	uid, err := db.ParseShort(p.User.ID)
	if err != nil {
		http.Error(p.W, "Invalid session", http.StatusUnauthorized)
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
	if p.User == nil {
		http.Redirect(p.W, p.Req, "/user/login", http.StatusSeeOther)
		return
	}

	// Handle POST (cancel action)
	if p.Req.Method == http.MethodPost {
		HandleCancelHold(p)
		return
	}

	if fail.Done(p) {
		return
	}

	uid, err := db.ParseShort(p.User.ID)
	if err != nil {
		http.Error(p.W, "Invalid session", http.StatusUnauthorized)
		return
	}

	var holds []pages.PatronHoldView
	db.Db().Raw(`
		SELECT h.id as hold_id, h.book_work_id as book_work_raw_id, h.format,
		       h.requested_date, COALESCE(bw.title, '') as book_title
		FROM holds h
		LEFT JOIN book_works bw ON bw.id = h.book_work_id
		WHERE h.user_id = ? AND h.fulfilled_date = ? AND h.cancelled_date = ?
		  AND h.deleted_at IS NULL
		ORDER BY h.requested_date ASC
	`, uid, db.NilTime, db.NilTime).Scan(&holds)

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

func HandleCancelHold(p *fail.RoutingParams) {
	if fail.Form(p) {
		return
	}
	formData := p.Form()
	holdID := formData["hold_id"]

	userUUID, err := db.ParseShort(p.User.ID)
	if err != nil {
		http.Error(p.W, "Invalid session", http.StatusUnauthorized)
		return
	}

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
	if p.User == nil {
		http.Redirect(p.W, p.Req, "/user/login", http.StatusSeeOther)
		return
	}
	if fail.Done(p) {
		return
	}

	uid, err := db.ParseShort(p.User.ID)
	if err != nil {
		http.Error(p.W, "Invalid session", http.StatusUnauthorized)
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

	db.MustSave(&u)
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

	db.MustSave(&u)
	fail.Render(p, pages.SettingsPasswordForm(nil, true))
}
