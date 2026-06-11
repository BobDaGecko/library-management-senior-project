package router

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"voxelprismatic/library-management-senior-project/db"
	"voxelprismatic/library-management-senior-project/router/fail"
	"voxelprismatic/library-management-senior-project/web/pages"
)

func Router(w http.ResponseWriter, r *http.Request) {
	if !sameOriginOK(r) {
		http.Error(w, "cross-origin request rejected", http.StatusForbidden)
		return
	}

	p := fail.MakeParams(w, r)

	switch p.Pop() {
	case "assets":
		HandleAsset(p)

	case "search":
		HandleSearch(p)
	case "help":
		HandleHelp(p)
	case "blog":
		BlogRouter(p)
	case "book":
		BookRouter(p)

	case "management":
		ManagementRouter(p)

	case "user":
		UserRouter(p)

	case "":
		HandlePublicHome(p)

	default:
		http.NotFound(w, r)
	}
}

// sameOriginOK rejects state-changing requests that provably come from another
// origin (CSRF defense in addition to the SameSite cookie). Requests without
// an Origin header (same-origin navigations, curl, tests) are allowed.
func sameOriginOK(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}

	origin := r.Header.Get("Origin")
	if origin == "" || origin == "null" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return u.Host == r.Host
}

func HandleAsset(p *fail.RoutingParams) {
	// Prevents escaping the root directory, eg "./assets/../../.." won't go beneath "./assets"
	root, err := os.OpenRoot("./assets")
	if err != nil {
		http.Error(p.W, err.Error(), http.StatusInternalServerError)
		return
	}

	path := "." + p.SubPath()
	_, err = root.Stat(path)
	if err != nil {
		code := http.StatusInternalServerError
		switch {
		case os.IsNotExist(err):
			code = http.StatusNotFound
		case os.IsPermission(err):
			code = http.StatusForbidden
		case strings.Contains(err.Error(), "escapes"):
			code = http.StatusUnauthorized
		}
		http.Error(p.W, err.Error(), code)
		return
	}

	http.ServeFile(p.W, p.Req, "./assets"+p.SubPath())
}

// Flat public pages (not under a sub-tree router)

func HandleSearch(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	f := pages.SearchFilter{
		Q:     p.Param("q"),
		Genre: p.Param("genre"),
		Sort:  p.Param("sort"),
	}

	// Parse format
	if fmtStr := p.Param("format"); fmtStr != "" {
		var fmtInt int
		fmt.Sscanf(fmtStr, "%d", &fmtInt)
		f.Format = db.BookFmtFlag(fmtInt)
	}

	// Parse availability
	f.Available = p.Param("available") == "1"

	if p.Req.Header.Get("HX-Request") == "true" {
		fail.Render(p, pages.SearchResults(f))
		return
	}
	fail.Render(p, pages.SearchPage(p, f))
}

func HandleHelp(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}
	fail.Render(p, pages.HelpPage(p))
}

func HandlePublicHome(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	// Staff picks: 3 featured books (shown with covers, linking to the book
	// detail page). The 3 newest titles serve as the picks.
	var staffPickBooks []db.BookWork
	db.Db().Where("id != ''").Order("created_at DESC").Limit(3).Find(&staffPickBooks)

	// Recently Added: the next 6 titles, kept distinct from the staff picks above.
	var recentBooks []db.BookWork
	db.Db().Where("id != ''").Order("created_at DESC").Offset(3).Limit(6).Find(&recentBooks)

	fail.Render(p, pages.PublicHomePage(p, nil, staffPickBooks, recentBooks))
}
