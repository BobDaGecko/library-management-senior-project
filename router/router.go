package router

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"voxelprismatic/library-management-senior-project/router/fail"
	"voxelprismatic/library-management-senior-project/web/pages"
)

func Router(w http.ResponseWriter, r *http.Request) {
	p := fail.MakeParams(w, r)
	if p == nil {
		panic("unreachable")
	}
	fmt.Printf("\n\n\n\n%v\n\n\n\n", p)

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
		fail.Render(p, pages.PublicHomePage())
	}
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
	fail.Render(p, pages.SearchPage())
}

func HandleHelp(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}
	fail.Render(p, pages.HelpPage())
}
