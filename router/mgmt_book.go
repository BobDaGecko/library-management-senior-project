package router

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"voxelprismatic/library-management-senior-project/db"
	"voxelprismatic/library-management-senior-project/fetch"
	"voxelprismatic/library-management-senior-project/router/fail"
	"voxelprismatic/library-management-senior-project/web/pages"
)

// fetchAndSaveBook pulls a volume from Google Books and upserts it as a local
// BookWork. Shared by the add-search PUT, add-confirm, and book-detail PUT flows.
func fetchAndSaveBook(googleId string) (db.BookWork, error) {
	details, err := fetch.GBooksVolume(googleId)
	if err != nil {
		return db.BookWork{}, fmt.Errorf("failed to fetch book from Google Books: %w", err)
	}
	book := details.ToLocalStruct()
	if err := db.Db().Save(&book).Error; err != nil {
		return db.BookWork{}, err
	}
	return book, nil
}

func ManagementBooksRouter(p *fail.RoutingParams) {
	if fail.Auth(p, db.UserRoleLibrarian) {
		return
	}

	segment := p.Pop()
	switch segment {
	case "add":
		HandleManagementBooksAdd(p)
	case "":
		HandleManagementCatalog(p)
	default:
		// Treat as Google Books Volume ID for add/edit operations
		HandleManagementBook(p, segment)
	}
}

func HandleManagementCatalog(p *fail.RoutingParams) {
	if fail.Done(p) {
		return
	}
	q := p.Req.URL.Query().Get("q")
	page := parsePageParam(p)
	if p.Req.Header.Get("HX-Request") == "true" {
		fail.Render(p, pages.MgmtCatalogResults(q, page))
		return
	}
	fail.Render(p, pages.MgmtCatalog(p, q, page))
}

func HandleManagementBooksAdd(p *fail.RoutingParams) {
	if fail.Remainder(p) {
		googleId := p.Pop()
		switch p.Pop() {
		case "preview":
			HandleManagementBookAddPreview(p, googleId)
		case "confirm":
			HandleManagementBookAddConfirm(p, googleId)
		default:
			fail.Done(p)
		}
		return
	}

	switch p.Req.Method {
	case http.MethodGet:
		query := p.Req.URL.Query().Get("q")
		fail.Render(p, pages.MgmtBookSearchFull(query, p))
	case http.MethodPost:
		if fail.Form(p) {
			return
		}
		query := p.Req.Form.Get("q")
		p.W.Header().Set(
			"hx-replace-url",
			fmt.Sprintf("%s?q=%s", p.Req.URL.String(), url.QueryEscape(query)),
		)
		fail.Render(p, pages.MgmtBookSearchGrid(query))
	case http.MethodPut:
		if fail.Form(p) {
			return
		}
		id := p.Req.Form.Get("id")
		if id == "" {
			http.Error(p.W, "missing id", http.StatusBadRequest)
			return
		}
		if _, err := fetchAndSaveBook(id); err != nil {
			http.Error(p.W, err.Error(), http.StatusBadGateway)
			return
		}
		p.W.Header().Set("Content-Type", "text/html; charset=utf-8")
		// id is form-supplied — escape it before reflecting into the attribute.
		fmt.Fprintf(p.W, `<a href="/management/books/%s" class="btn btn-sm btn-subtle">In Library →</a>`,
			html.EscapeString(url.PathEscape(id)))
		return
	default:
		http.Error(p.W, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleManagementBookAddPreview(p *fail.RoutingParams, googleId string) {
	if fail.Done(p) {
		return
	}
	if p.Req.Method != http.MethodGet {
		http.Error(p.W, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	details, err := fetch.GBooksVolume(googleId)
	if err != nil {
		http.Error(p.W, "Failed to fetch book from Google Books: "+err.Error(), http.StatusBadGateway)
		return
	}
	fail.Render(p, pages.MgmtBookPreview(details.ToLocalStruct(), googleId, p))
}

func HandleManagementBookAddConfirm(p *fail.RoutingParams, googleId string) {
	if fail.Done(p) {
		return
	}
	if p.Req.Method != http.MethodPost {
		http.Error(p.W, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, err := fetchAndSaveBook(googleId); err != nil {
		http.Error(p.W, err.Error(), http.StatusBadGateway)
		return
	}
	http.Redirect(p.W, p.Req, "/management/books/"+url.PathEscape(googleId), http.StatusSeeOther)
}

// HandleManagementBook handles GET (edit page), PUT (fetch from Google & overwrite), PATCH (form update with reflect)
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
		case "edit":
			HandleManagementBookEdit(p, bookId)
		case "delete":
			HandleManagementBookDelete(p, bookId)
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

			fail.Render(p, pages.MgmtBookNotInLibraryWarning(book, bookId, p))
			return
		}

		fail.Render(p, pages.MgmtBookDetail(p, book))

	case http.MethodPut:
		// Fetch all details from Google Books and overwrite in DB
		if _, err := fetchAndSaveBook(bookId); err != nil {
			http.Error(p.W, err.Error(), http.StatusBadGateway)
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

		p.W.Header().Set("HX-Redirect", fmt.Sprintf("/management/books/%s", bookId))

	default:
		http.Error(p.W, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleManagementBookEdit(p *fail.RoutingParams, bookId string) {
	if fail.Done(p) {
		return
	}
	if p.Req.Method != http.MethodGet {
		http.Error(p.W, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var book db.BookWork
	if err := db.Db().Where("id = ?", bookId).First(&book).Error; err != nil {
		http.Error(p.W, "Book not found", http.StatusNotFound)
		return
	}
	fail.Render(p, pages.MgmtBookEdit(bookId, book, p))
}

func HandleManagementBookDelete(p *fail.RoutingParams, bookId string) {
	if fail.Done(p) {
		return
	}
	if p.Req.Method != http.MethodPost {
		http.Error(p.W, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Block deletion if any copies are currently checked out
	var activeLoans int64
	db.Db().Table("loans").
		Joins("JOIN book_copies ON book_copies.id = loans.book_copy_id").
		Where("book_copies.book_work_id = ?", bookId).
		Where("loans.date_returned = ?", db.NilTime).
		Where("loans.deleted_at IS NULL").
		Count(&activeLoans)
	if activeLoans > 0 {
		http.Error(p.W, "Cannot delete: book has active loans", http.StatusConflict)
		return
	}

	// Cancel open holds, soft-delete copies, then the book work — atomically,
	// so a failure partway can't leave a half-deleted book behind.
	err := db.Db().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&db.Hold{}).
			Where("book_work_id = ?", bookId).
			Where("fulfilled_date = ?", db.NilTime).
			Where("cancelled_date = ?", db.NilTime).
			Update("cancelled_date", time.Now()).Error; err != nil {
			return err
		}
		if err := tx.Where("book_work_id = ?", bookId).Delete(&db.BookCopy{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", bookId).Delete(&db.BookWork{}).Error
	})
	if err != nil {
		http.Error(p.W, "failed to delete book: "+err.Error(), http.StatusInternalServerError)
		return
	}

	p.W.Header().Set("HX-Redirect", "/management/books")
	p.W.WriteHeader(http.StatusOK)
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

	book, copiesMap, err := GetBookAndCopies(bookID)
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
		_, freshMap, _ := GetBookAndCopies(bookID)
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

	if p.Req.Method == http.MethodPost {
		if fail.Form(p) {
			return
		}
		switch p.Req.FormValue("action") {
		case "delete":
			_, histTotal, _ := copy.History(1, 1)
			if histTotal > 0 {
				http.Error(p.W, "Cannot delete a copy that has checkout or repair history", http.StatusConflict)
				return
			}
			// Each chained GORM call returns its own *gorm.DB, so errors must
			// be checked per statement — the parent tx.Error never gets set.
			err := db.Db().Transaction(func(tx *gorm.DB) error {
				if err := tx.Where("book_copy_id = ?", copy.ID).Delete(&db.Loan{}).Error; err != nil {
					return err
				}
				if err := tx.Where("book_copy_id = ?", copy.ID).Delete(&db.RepairLog{}).Error; err != nil {
					return err
				}
				return tx.Unscoped().Delete(&copy).Error
			})
			if err != nil {
				http.Error(p.W, "Failed to delete copy: "+err.Error(), http.StatusInternalServerError)
				return
			}
			p.W.Header().Set("HX-Redirect", fmt.Sprintf("/management/books/%s/copies", bookID))
			return

		case "start_repair":
			if err := db.Db().Model(&db.BookCopy{}).Where("id = ?", copy.ID).Update("status", db.CopyStatusRepairing).Error; err != nil {
				http.Error(p.W, err.Error(), http.StatusInternalServerError)
				return
			}
			copy.Status = db.CopyStatusRepairing
			fail.Render(p, pages.CopyManagementSection(bookID, copy))
			return

		case "complete_repair":
			technician := p.Req.FormValue("technician")
			condInt, err := strconv.Atoi(p.Req.FormValue("condition"))
			if err != nil || condInt < int(db.ConditionMint) || condInt > int(db.ConditionLost) {
				// Never default silently — a parse failure would record the
				// repaired copy as factory-mint (ConditionFlag zero value).
				http.Error(p.W, "invalid condition", http.StatusBadRequest)
				return
			}
			repairLog := db.RepairLog{
				BookCopyID:     copy.ID,
				Date:           time.Now(),
				IncomingStatus: copy.Status,
				OutgoingStatus: db.CopyStatusPublic,
				TechnicianName: technician,
			}
			err = db.Db().Transaction(func(tx *gorm.DB) error {
				if err := tx.Create(&repairLog).Error; err != nil {
					return err
				}
				return tx.Model(&db.BookCopy{}).Where("id = ?", copy.ID).Updates(map[string]any{
					"status":    db.CopyStatusPublic,
					"condition": db.ConditionFlag(condInt),
				}).Error
			})
			if err != nil {
				http.Error(p.W, "failed to complete repair: "+err.Error(), http.StatusInternalServerError)
				return
			}
			copy.Status = db.CopyStatusPublic
			copy.Condition = db.ConditionFlag(condInt)

			const perPage = 25
			entries, histTotal, _ := copy.History(1, perPage)
			fail.Render(p, pages.CopyManagementWithHistoryOOB(bookID, copy, entries, 1, totalPages(histTotal, perPage), histTotal))
			return

		case "discard":
			db.Db().Model(&db.BookCopy{}).Where("id = ?", copy.ID).Update("status", db.CopyStatusDiscarded)
			copy.Status = db.CopyStatusDiscarded
			fail.Render(p, pages.CopyManagementSection(bookID, copy))
			return

		case "restore":
			db.Db().Model(&db.BookCopy{}).Where("id = ?", copy.ID).Update("status", db.CopyStatusPublic)
			copy.Status = db.CopyStatusPublic
			fail.Render(p, pages.CopyManagementSection(bookID, copy))
			return
		}
		// fallthrough: pagination POST from common.Pagination component
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
	pageCount := totalPages(total, perPage)

	// HTMX pagination requests (the common.Pagination component does hx-post) -> render only the history table fragment
	if p.Req.Header.Get("HX-Request") == "true" {
		fail.Render(p, pages.MgmtBookCopyHistoryTable(bookID, copy, entries, page, pageCount, total))
		return
	}

	fail.Render(p, pages.MgmtBookCopyDetail(bookID, copy, entries, page, pageCount, total, p))
}

// totalPages returns the 1-based page count for a paginated table.
func totalPages(total int64, perPage int) int {
	if total <= 0 {
		return 1
	}
	return int((total + int64(perPage) - 1) / int64(perPage))
}

// GetBookAndCopies loads the BookWork and its copies grouped by Format using the existing MapFormats helper.
// Exported for use by public book details router.
func GetBookAndCopies(bookID string) (db.BookWork, db.FormatsMap[db.CopyList], error) {
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

// bookEditableFields is the allow-list for the metadata PATCH form. Without
// it, any BookWork field — including the primary key — could be overwritten
// by an extra form key (mass assignment).
var bookEditableFields = map[string]bool{
	"title": true, "subtitle": true, "authors": true, "publisher": true,
	"publisheddate": true, "version": true, "isbn13": true, "isbn10": true,
	"description": true, "pagecount": true, "ismature": true,
	"categories": true, "coverthumb": true, "coverimage": true,
}

// updateBookFromForm copies allow-listed form values onto the BookWork via reflection.
func updateBookFromForm(target interface{}, form map[string][]string) {
	v := reflect.ValueOf(target).Elem()
	t := v.Type()
	for key, vals := range form {
		if len(vals) == 0 || !bookEditableFields[strings.ToLower(key)] {
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
				case reflect.Slice:
					if fv.Type().Elem().Kind() == reflect.String {
						parts := []string{}
						for _, line := range strings.Split(valStr, "\n") {
							for _, p := range strings.Split(line, ",") {
								if t := strings.TrimSpace(p); t != "" {
									parts = append(parts, t)
								}
							}
						}
						fv.Set(reflect.ValueOf(parts).Convert(fv.Type()))
					}
				}
				break
			}
		}
	}
}
