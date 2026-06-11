package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"voxelprismatic/library-management-senior-project/db"
	"voxelprismatic/library-management-senior-project/router/fail"

	"gotest.tools/v3/assert"
)

// makeTestParamsWithUser creates a RoutingParams with an authenticated user.
func makeTestParamsWithUser(t *testing.T, method, path string, segments []string, user *db.UserPartial, body url.Values) (*fail.RoutingParams, *httptest.ResponseRecorder) {
	t.Helper()
	rec := httptest.NewRecorder()

	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(body.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	p := &fail.RoutingParams{
		W:        rec,
		Req:      req,
		SubPtr:   len(segments),
		FullPath: segments,
		User:     user,
	}
	return p, rec
}

// TestHandleBookHold_Unauthenticated checks that an unauthenticated POST sets HX-Redirect to /user/login.
func TestHandleBookHold_Unauthenticated(t *testing.T) {
	tx := db.TestDb()
	defer tx.Rollback()

	p, rec := makeTestParamsWithUser(t, http.MethodPost, "/book/test-book-id/hold",
		[]string{"book", "test-book-id", "hold"}, nil, nil)
	// The hold form always posts via HTMX, which is what triggers the
	// HX-Redirect (vs. plain 303) guest handling.
	p.Req.Header.Set("HX-Request", "true")

	HandleBookHold(p, "test-book-id")

	res := rec.Result()
	assert.Equal(t, res.Header.Get("HX-Redirect"), "/user/login")
	assert.Equal(t, res.StatusCode, http.StatusOK)
}

// TestHandleBookHold_CreatesHold checks that an authenticated user without an existing hold gets one created.
func TestHandleBookHold_CreatesHold(t *testing.T) {
	tx := db.TestDb()
	defer tx.Rollback()

	user := db.User{
		FirstName: "Test",
		LastName:  "Patron",
		Email:     "patron@test.com",
		Roles:     db.UserRolePublic,
		Status:    db.UserStatusActive,
	}
	tx.Save(&user)

	book := db.BookWork{
		ID:    "test-book-abc",
		Title: "Test Book",
	}
	tx.Save(&book)

	userPartial := user.Partial()

	formVals := url.Values{"format": {fmt.Sprintf("%d", int(db.BookFmtPaperback))}}
	p, rec := makeTestParamsWithUser(t, http.MethodPost, "/book/test-book-abc/hold",
		[]string{"book", "test-book-abc", "hold"}, &userPartial, formVals)

	HandleBookHold(p, "test-book-abc")

	res := rec.Result()
	// Should succeed — no error redirect
	assert.Assert(t, res.StatusCode != http.StatusInternalServerError, "expected non-500 response")

	// Verify a hold was created in the DB
	var count int64
	tx.Model(&db.Hold{}).
		Where("user_id = ? AND fulfilled_date = ? AND cancelled_date = ?",
			user.ID, db.NilTime, db.NilTime).
		Count(&count)
	assert.Equal(t, count, int64(1), "expected exactly one open hold to be created")
}

// TestHandleBookHold_NoDuplicate checks that submitting a hold twice does not create a duplicate.
func TestHandleBookHold_NoDuplicate(t *testing.T) {
	tx := db.TestDb()
	defer tx.Rollback()

	user := db.User{
		FirstName: "Test",
		LastName:  "Patron",
		Email:     "patron2@test.com",
		Roles:     db.UserRolePublic,
		Status:    db.UserStatusActive,
	}
	tx.Save(&user)

	book := db.BookWork{
		ID:    "test-book-dup",
		Title: "Duplicate Hold Book",
	}
	tx.Save(&book)

	// Pre-create an open hold for this user+book
	hold := db.Hold{
		UserID:        user.ID,
		RequestedDate: time.Now(),
	}
	tx.Create(&hold)
	tx.Model(&db.Hold{}).Where("id = ?", hold.ID).Update("book_work_id", "test-book-dup")

	userPartial := user.Partial()
	formVals := url.Values{"format": {fmt.Sprintf("%d", int(db.BookFmtPaperback))}}
	p, rec := makeTestParamsWithUser(t, http.MethodPost, "/book/test-book-dup/hold",
		[]string{"book", "test-book-dup", "hold"}, &userPartial, formVals)

	HandleBookHold(p, "test-book-dup")

	res := rec.Result()
	assert.Assert(t, res.StatusCode != http.StatusInternalServerError)

	// Body should contain "already" text
	body := rec.Body.String()
	assert.Assert(t, strings.Contains(body, "already"), "expected 'already' in response body, got: "+body)

	// Still only one hold in DB
	var count int64
	tx.Model(&db.Hold{}).
		Where("user_id = ? AND fulfilled_date = ? AND cancelled_date = ?",
			user.ID, db.NilTime, db.NilTime).
		Count(&count)
	assert.Equal(t, count, int64(1), "expected exactly one hold (no duplicate created)")
}
