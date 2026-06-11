package fail

import (
	"net/http"

	"voxelprismatic/library-management-senior-project/db"

	"github.com/a-h/templ"
)

// Render renders a templ.Component and sets Content-Type to text/html.
// Use this for both full pages and partial HTMX fragments.
func Render(p *RoutingParams, elem templ.Component) {
	p.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := elem.Render(p.Req.Context(), p.W)
	if err != nil {
		http.Error(p.W, err.Error(), http.StatusInternalServerError)
	}
}

// Returns 'true' if the form failed to parse
// Usage: `if fail.Form(p) { return }`
func Form(p *RoutingParams) bool {
	err := p.Req.ParseForm()
	if err != nil {
		http.Error(p.W, err.Error(), http.StatusBadRequest)
		return true
	}
	return false
}

// Returns 'true' if the user does NOT meet the minimum role requirements
// Usage: `if fail.Auth(p, UserRoleLibrarian) { return }`
func Auth(p *RoutingParams, minLevel db.UserRoleFlag) bool {
	if p.User != nil && minLevel <= p.User.Roles {
		return false
	}

	http.Error(p.W, "Forbidden", http.StatusForbidden)
	return true
}

func Redirect(p *RoutingParams) {
	http.Redirect(p.W, p.Req, p.SubPathTree(false), http.StatusPermanentRedirect)
}

// Fails if the path is not fully consumed
func Done(p *RoutingParams) bool {
	if p.SubPtr >= len(p.FullPath) {
		return false
	}

	// Ending /, but nothing nefarious
	if p.SubPtr == len(p.FullPath)-1 && p.FullPath[p.SubPtr] == "" {
		Redirect(p)
		return true
	}

	_, _ = p.W.Write([]byte("404: too far"))
	return true
}

// Inverse of Done(), but it does NOT issue http warnings
func Remainder(p *RoutingParams) bool {
	if p.SubPtr >= len(p.FullPath) {
		return false
	}
	return true
}
