package router

import (
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
	fail.Render(p, pages.BlogPage())
}

func HandleBlogDetails(p *fail.RoutingParams, id string) {
	if fail.Done(p) {
		return
	}
	_ = id // TODO: when enhancing blog_details.templ, pass id/data here
	fail.Render(p, pages.BlogDetailsPage())
}
