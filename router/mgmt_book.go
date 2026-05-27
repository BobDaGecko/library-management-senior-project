package router

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"voxelprismatic/library-management-senior-project/db"
	"voxelprismatic/library-management-senior-project/fetch"
	"voxelprismatic/library-management-senior-project/router/fail"
	"voxelprismatic/library-management-senior-project/web/pages"
)

func ManagementBooksRouter(p *fail.RoutingParams) {
	if fail.Auth(p, db.UserRoleLibrarian) {
		return
	}

	segment := p.Pop()
	if segment == "" {
		fail.Redirect(p)
		return
	}
	switch segment {
	case "add":
		HandleManagementBooksAdd(p)
	default:
		// Treat as Google Books Volume ID for add/edit operations
		HandleManagementBook(p, segment)
	}
}

func HandleManagementBooksAdd(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}

	switch p.Req.Method {
	case http.MethodGet:
		query := p.Req.URL.Query().Get("q")
		fail.Render(p, pages.BookMgmtSearchFull(query, p))
	case http.MethodPost:
		if fail.Form(p) {
			return
		}
		query := p.Req.Form.Get("q")
		p.W.Header().Set(
			"hx-replace-url",
			fmt.Sprintf("%s?q=%s", p.Req.URL.String(), url.QueryEscape(query)),
		)
		fail.Render(p, pages.BookMgmtSearchGrid(query))
	default:
		http.Error(p.W, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleManagementBook handles GET (edit page placeholder), PUT (fetch from Google & overwrite), PATCH (form update with reflect)
// for /management/books/:id where :id is Google Books volume ID.
// GET now gracefully handles books not yet in the local library.
func HandleManagementBook(p *fail.RoutingParams, id string) {
	if fail.Done(p) {
		return
	}

	switch p.Req.Method {
	case http.MethodGet:
		var book db.BookWork
		err := db.Db().Where("id = ?", id).First(&book).Error

		if err != nil {
			// Book not in local library → fetch from Google and show warning banner + Add button
			details, gErr := fetch.GBooksVolume(id)
			if gErr != nil {
				http.Error(p.W, "Book not found in library and could not be fetched from Google Books", http.StatusNotFound)
				return
			}
			book = details.ToLocalStruct()

			p.W.Header().Set("Content-Type", "text/html; charset=utf-8")
			// Simple but functional warning + add UI (we can replace with a proper Templ component later)
			fmt.Fprintf(p.W, `
<div class="container">
	<div class="alert alert-warning mb-4">
		<strong>Warning:</strong> This book is not yet in your library.
	</div>

	<div class="card">
		<div class="card-body">
			<h2>%s</h2>
			<p class="text-secondary">by %s</p>
			%s
			<div class="mt-4">
				<button 
					hx-put="/management/books/%s" 
					hx-target="closest .card" 
					hx-swap="outerHTML"
					class="btn btn-primary">
					Add to Library
				</button>
			</div>
		</div>
	</div>
</div>
`, 
				book.Title, 
				strings.Join(book.Authors, ", "), 
				book.Description, 
				id)
			return
		}

		// Book exists locally → placeholder for full edit page
		http.Error(p.W, "Edit page for book not yet implemented (501)", http.StatusNotImplemented)

	case http.MethodPut:
		// Fetch all details from Google Books and overwrite in DB
		details, err := fetch.GBooksVolume(id)
		if err != nil {
			http.Error(p.W, "Failed to fetch book from Google Books: "+err.Error(), http.StatusBadGateway)
			return
		}
		book := details.ToLocalStruct()
		if err := db.Db().Save(&book).Error; err != nil {
			http.Error(p.W, err.Error(), http.StatusInternalServerError)
			return
		}

		// Response based on referrer (for HTMX flows from add page or edit page)
		referrer := p.Req.Referer()
		if strings.Contains(referrer, "/management/books/add") {
			// From add/search page: return a checkmark button (success indicator)
			p.W.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(p.W, `<button disabled class="btn btn-success" title="Added to library">✓</button>`)
			return
		} else if strings.Contains(referrer, "/management/books/"+id) {
			// Already on the detail/edit page: trigger HTMX page refresh
			p.W.Header().Set("HX-Refresh", "true")
			return
		} else {
			// Otherwise: HTMX redirect to the detail page
			p.W.Header().Set("HX-Redirect", fmt.Sprintf("/management/books/%s", id))
			return
		}

	case http.MethodPatch:
		// Update DB from form fields using reflection (flexible for new fields)
		if fail.Form(p) {
			return
		}
		form := p.Req.Form

		var book db.BookWork
		if err := db.Db().Where("id = ?", id).First(&book).Error; err != nil {
			http.Error(p.W, "Book not found", http.StatusNotFound)
			return
		}

		updateBookFromForm(&book, form)

		if err := db.Db().Save(&book).Error; err != nil {
			http.Error(p.W, err.Error(), http.StatusInternalServerError)
			return
		}

		// PATCH redirects to public book view
		p.W.Header().Set("HX-Redirect", fmt.Sprintf("/books/%s", id))

	default:
		http.Error(p.W, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// updateBookFromForm uses reflection to update struct fields from form values by matching key names.
// Supports string, int, bool, time.Time. Extend as needed for slices (e.g. SqlStringList).
func updateBookFromForm(target interface{}, form map[string][]string) {
	v := reflect.ValueOf(target).Elem()
	t := v.Type()
	for key, vals := range form {
		if len(vals) == 0 {
			continue
		}
		valStr := vals[0]
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			if strings.EqualFold(sf.Name, key) {
				fv := v.Field(i)
				if !fv.CanSet() {
					continue
				}
				switch fv.Kind() {
				case reflect.String:
					fv.SetString(valStr)
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if iVal, err := strconv.Atoi(valStr); err == nil {
						fv.SetInt(int64(iVal))
					}
				case reflect.Bool:
					fv.SetBool(valStr == "true" || valStr == "1" || valStr == "on" || valStr == "yes")
				case reflect.Struct:
					if fv.Type() == reflect.TypeOf(time.Time{}) {
						if tVal, err := time.Parse(time.RFC3339, valStr); err == nil {
							fv.Set(reflect.ValueOf(tVal))
						} else if tVal, err := time.Parse("2006-01-02", valStr); err == nil {
							fv.Set(reflect.ValueOf(tVal))
						}
					}
				// Note: For SqlStringList, you can extend this to split comma-separated values if needed.
				}
				break
			}
		}
	}
}
