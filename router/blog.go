package router

import (
	"net/http"

	"voxelprismatic/library-management-senior-project/router/fail"
	"voxelprismatic/library-management-senior-project/web/pages"
)

// BlogRouter handles the /blog tree (public content):
//
//	/blog      -> blog (BlogPage)
//	/blog/:id  -> blog_details (BlogDetailsPage)
func BlogRouter(p *fail.RoutingParams) {
	if fail.Remainder(p) {
		id := p.Pop()
		HandleBlogDetails(p, id)
		return
	}
	if fail.Done(p) {
		return
	}
	fail.Render(p, pages.BlogPage(p))
}

func HandleBlogDetails(p *fail.RoutingParams, id string) {
	if fail.Done(p) {
		return
	}
	post, ok := pages.FindPost(id)
	if !ok {
		http.Error(p.W, "Post not found", http.StatusNotFound)
		return
	}
	fail.Render(p, pages.BlogDetailsPage(p, post))
}
