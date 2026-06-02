package router

import (
	"net/http"

	"voxelprismatic/library-management-senior-project/router/fail"
	"voxelprismatic/library-management-senior-project/web/pages"
)

// BookRouter handles the public /book tree (distinct from /management/books):
//
//	/book/:id  -> book_details (BookDetailsPage)
func BookRouter(p *fail.RoutingParams) {
	id := ""
	if fail.Remainder(p) {
		id = p.Pop()
	}
	if id == "" {
		fail.Redirect(p)
		return
	}
	if fail.Remainder(p) {
		fail.Done(p)
		return
	}

	book, copiesMap, err := GetBookAndCopies(id)
	if err != nil {
		http.Error(p.W, "Book not found", http.StatusNotFound)
		return
	}

	fail.Render(p, pages.BookDetailsPage(id, book, copiesMap))
}
