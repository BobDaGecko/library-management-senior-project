package router

import (
	"fmt"
	"net/http"
	"voxelprismatic/library-management-senior-project/db"
	"voxelprismatic/library-management-senior-project/router/fail"
	"voxelprismatic/library-management-senior-project/web/user"
)

func UserRouter(p *fail.RoutingParams) {
	switch p.Pop() {
	case "register":
		HandleUserRegister(p)
	case "login":
		HandleUserLogin(p)
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

	errs := map[string]error{}
	formData := p.Form()
	userObj := db.User{Email: formData["emailAddr"]}
	if userObj.Email == "" {
		errs["emailAddr"] = fmt.Errorf("email cannot be blank")
	} else if err := db.Db().Where(&userObj).First(&userObj).Error; err != nil {
		errs["emailAddr"] = fmt.Errorf("email not found")
	}

	if len(errs) == 0 && !userObj.TestSecret(formData["secret"]) {
		errs["secret"] = fmt.Errorf("incorrect password")
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
