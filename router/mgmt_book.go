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
	switch segment {
	case "add":
		HandleManagementBooksAdd(p)
	case "":
		fail.Redirect(p)
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
// It also supports nested routes:
//
//	/management/books/:id/copies
//	/management/books/:id/copies/:copy-id
func HandleManagementBook(p *fail.RoutingParams, bookId string) {
	// Ensure context map exists and store the book ID
	if p.Context == nil {
		p.Context = make(map[string]string)
	}
	p.Context["book-id"] = bookId

	if fail.Remainder(p) {
		switch p.Pop() {
		case "copies":
			HandleManagementBookCopies(p, bookId)
		default:
			fail.Done(p)
		}
		return
	}

	switch p.Req.Method {
	case http.MethodGet:
		var book db.BookWork
		err := db.Db().Where("id = ?", bookId).First(&book).Error

		if err != nil {
			// Book not in local library → fetch from Google and show warning banner + Add button
			details, gErr := fetch.GBooksVolume(bookId)
			if gErr != nil {
				http.Error(p.W, "Book not found in library and could not be fetched from Google Books", http.StatusNotFound)
				return
			}
			book = details.ToLocalStruct()

			fail.Render(p, pages.BookMgmtNotInLibraryWarning(book, bookId))
			return
		}

		// Book exists locally → placeholder for full edit page
		http.Error(p.W, "Edit page for book not yet implemented (501)", http.StatusNotImplemented)

	case http.MethodPut:
		// Fetch all details from Google Books and overwrite in DB
		details, err := fetch.GBooksVolume(bookId)
		if err != nil {
			http.Error(p.W, "Failed to fetch book from Google Books: "+err.Error(), http.StatusBadGateway)
			return
		}
		book := details.ToLocalStruct()
		if err := db.Db().Save(&book).Error; err != nil {
			http.Error(p.W, err.Error(), http.StatusInternalServerError)
			return
		}

		// Response based on referrer
		referrer := p.Req.Referer()
		if strings.Contains(referrer, "/management/books/add") {
			p.W.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(p.W, `<button disabled class="btn btn-success" title="Added to library">Added to library</button>`)
			return
		} else if strings.Contains(referrer, "/management/books/"+bookId) {
			p.W.Header().Set("HX-Refresh", "true")
			return
		} else {
			p.W.Header().Set("HX-Redirect", fmt.Sprintf("/management/books/%s", bookId))
			return
		}

	case http.MethodPatch:
		if fail.Form(p) {
			return
		}
		form := p.Req.Form

		var book db.BookWork
		if err := db.Db().Where("id = ?", bookId).First(&book).Error; err != nil {
			http.Error(p.W, "Book not found", http.StatusNotFound)
			return
		}

		updateBookFromForm(&book, form)

		if err := db.Db().Save(&book).Error; err != nil {
			http.Error(p.W, err.Error(), http.StatusInternalServerError)
			return
		}

		p.W.Header().Set("HX-Redirect", fmt.Sprintf("/books/%s", bookId))

	default:
		http.Error(p.W, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleManagementBookCopies(p *fail.RoutingParams, bookID string) {
	if fail.Remainder(p) {
		segement := p.Pop()
		switch segement {
		case "":
			fail.Redirect(p)
		default:
			HandleManagementBookCopyDetail(p, bookID, segement)
		}
		return
	}

	book, copiesMap, err := getBookAndCopies(bookID)
	if err != nil {
		http.Error(p.W, "Book not found or error loading copies: "+err.Error(), http.StatusNotFound)
		return
	}

	switch p.Req.Method {
	case http.MethodGet:
		if p.Req.Header.Get("HX-Request") == "true" {
			fail.Render(p, pages.MgmtBookCopiesList(bookID, copiesMap))
			return
		}
		fail.Render(p, pages.MgmtBookCopies(bookID, book, copiesMap, p))
	case http.MethodPost:
		if fail.Form(p) {
			return
		}
		formatStr := p.Req.Form.Get("format")
		fInt, err := strconv.Atoi(formatStr)
		if err != nil {
			http.Error(p.W, "invalid format", http.StatusBadRequest)
			return
		}
		f := db.BookFmtFlag(fInt)

		newC := db.BookCopy{
			BookWorkID: bookID,
			Format:     f,
			Condition:  db.ConditionMint,
			Status:     db.CopyStatusPublic,
		}
		if err := db.Db().Create(&newC).Error; err != nil {
			http.Error(p.W, "failed to create copy: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Re-fetch for fresh grouped data
		_, freshMap, _ := getBookAndCopies(bookID)
		fail.Render(p, pages.MgmtBookCopiesList(bookID, freshMap))
	default:
		http.Error(p.W, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleManagementBookCopyDetail(p *fail.RoutingParams, bookID, copyID string) {
	if fail.Done(p) {
		return
	}

	// copyID in URL may be the 22-char Short() base64 or a full hex UUID
	copyUUID, err := db.FromString(copyID)
	if err != nil || copyUUID.IsEmpty() {
		http.Error(p.W, "Invalid copy identifier", http.StatusBadRequest)
		return
	}

	var copy db.BookCopy
	if err := db.Db().Preload("BookWork").Where("id = ?", copyUUID).First(&copy).Error; err != nil {
		http.Error(p.W, "Copy not found", http.StatusNotFound)
		return
	}
	if copy.BookWorkID != bookID {
		http.Error(p.W, "Copy does not belong to the specified book", http.StatusBadRequest)
		return
	}

	// Support delete (only offered in UI when history total == 0)
	if p.Req.Method == http.MethodPost {
		if fail.Form(p) {
			return
		}
		if p.Req.FormValue("action") == "delete" {
			// Re-check safety: must still have zero history
			_, histTotal, _ := copy.History(1, 1)
			if histTotal > 0 {
				http.Error(p.W, "Cannot delete a copy that has checkout or repair history", http.StatusConflict)
				return
			}

			// Clean related records then hard-delete the copy itself
			tx := db.Db().Begin()
			tx.Where("book_copy_id = ?", copy.ID).Delete(&db.Loan{})
			tx.Where("book_copy_id = ?", copy.ID).Delete(&db.RepairLog{})
			tx.Unscoped().Delete(&copy)
			if tx.Error != nil {
				tx.Rollback()
				http.Error(p.W, "Failed to delete copy: "+tx.Error.Error(), http.StatusInternalServerError)
				return
			}
			tx.Commit()

			p.W.Header().Set("HX-Redirect", fmt.Sprintf("/management/books/%s/copies", bookID))
			return
		}
		// fallthrough: POST without delete action = pagination request for history table
	}

	// Pagination (supports both ?page= on GET and form page= from the common Pagination component POST)
	page := 1
	pageStr := p.Param("page")
	if pageStr == "" && p.Req.Method == http.MethodPost {
		pageStr = p.Req.FormValue("page")
	}
	if pNum, err := strconv.Atoi(pageStr); err == nil && pNum > 0 {
		page = pNum
	}

	const perPage = 25
	entries, total, err := copy.History(page, perPage)
	if err != nil {
		http.Error(p.W, "Failed to load copy history: "+err.Error(), http.StatusInternalServerError)
		return
	}

	totalPages := 1
	if total > 0 {
		totalPages = int((total + int64(perPage) - 1) / int64(perPage))
	}

	// HTMX pagination requests (the common.Pagination component does hx-post) -> render only the history table fragment
	if p.Req.Header.Get("HX-Request") == "true" {
		fail.Render(p, pages.MgmtBookCopyHistoryTable(bookID, copy, entries, page, totalPages, total))
		return
	}

	fail.Render(p, pages.MgmtBookCopyDetail(bookID, copy, entries, page, totalPages, total, p))
}

// getBookAndCopies loads the BookWork and its copies grouped by Format using the existing MapFormats helper.
func getBookAndCopies(bookID string) (db.BookWork, db.FormatsMap[db.CopyList], error) {
	var book db.BookWork
	if err := db.Db().Where("id = ?", bookID).First(&book).Error; err != nil {
		return book, nil, err
	}

	var copies []db.BookCopy
	if err := db.Db().Where("book_work_id = ?", bookID).Order("created_at DESC").Find(&copies).Error; err != nil {
		return book, nil, err
	}

	return book, db.CopyList(copies).MapFormats(), nil
}

// updateBookFromForm ...
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
				}
				break
			}
		}
	}
}
