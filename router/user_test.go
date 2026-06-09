package router

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"voxelprismatic/library-management-senior-project/db"
	"voxelprismatic/library-management-senior-project/router/fail"

	"gotest.tools/v3/assert"
)

// makeTestParams creates a RoutingParams that has already consumed the given
// path segments (SubPtr == len(path)), simulating the state after routing.
func makeTestParams(t *testing.T, method, path string, segments []string) (*fail.RoutingParams, *httptest.ResponseRecorder) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	p := &fail.RoutingParams{
		W:        rec,
		Req:      req,
		SubPtr:   len(segments),
		FullPath: segments,
		User:     nil,
	}
	return p, rec
}

func TestHandleUserLogoutClearsCookieAndRedirects(t *testing.T) {
	p, rec := makeTestParams(t, http.MethodGet, "/user/logout", []string{"user", "logout"})

	HandleUserLogout(p)

	res := rec.Result()
	assert.Equal(t, res.StatusCode, http.StatusSeeOther)

	loc := res.Header.Get("Location")
	assert.Equal(t, loc, "/")

	// tok cookie must be cleared (Max-Age=0 in the Set-Cookie header)
	cookieHeader := res.Header.Get("Set-Cookie")
	assert.Assert(t, cookieHeader != "", "Set-Cookie header must be present")

	cookies := res.Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "tok" {
			found = true
			// Go's cookie parser maps Max-Age=0 in the Set-Cookie header to MaxAge=-1
		assert.Equal(t, c.MaxAge, -1, "tok cookie MaxAge must signal deletion (MaxAge=-1 after parsing)")
		}
	}
	assert.Assert(t, found, "tok cookie must appear in Set-Cookie response")
}

func TestHandleUserDashboardRedirectsToUser(t *testing.T) {
	p, rec := makeTestParams(t, http.MethodGet, "/user/dashboard", []string{"user", "dashboard"})

	HandleUserDashboard(p)

	res := rec.Result()
	assert.Equal(t, res.StatusCode, http.StatusMovedPermanently)

	loc := res.Header.Get("Location")
	assert.Equal(t, loc, "/user")
}

func TestCancelHoldOwnershipCheck(t *testing.T) {
	tx := db.TestDb()
	defer tx.Rollback()

	user1 := db.User{FirstName: "Alice", LastName: "Doe", Email: "alice@example.com", Status: db.UserStatusActive}
	tx.Save(&user1)
	user2 := db.User{FirstName: "Bob", LastName: "Doe", Email: "bob@example.com", Status: db.UserStatusActive}
	tx.Save(&user2)

	hold := db.Hold{UserID: user1.ID, RequestedDate: time.Now()}
	tx.Create(&hold)

	partial2 := user2.Partial()
	formVals := url.Values{"action": {"cancel"}, "hold_id": {hold.ID.String()}}
	p, rec := makeTestParamsWithUser(t, http.MethodPost, "/user/holds", []string{"user", "holds"}, &partial2, formVals)

	HandleCancelHold(p)

	res := rec.Result()
	assert.Equal(t, res.StatusCode, http.StatusForbidden)

	var updated db.Hold
	tx.Where("id = ?", hold.ID).First(&updated)
	assert.Assert(t, updated.CancelledDate.IsZero(), "CancelledDate must not be set after failed ownership check")
}

func TestCancelHoldSuccess(t *testing.T) {
	tx := db.TestDb()
	defer tx.Rollback()

	user := db.User{FirstName: "Carol", LastName: "Doe", Email: "carol@example.com", Status: db.UserStatusActive}
	tx.Save(&user)

	hold := db.Hold{UserID: user.ID, RequestedDate: time.Now()}
	tx.Create(&hold)

	partial := user.Partial()
	formVals := url.Values{"action": {"cancel"}, "hold_id": {hold.ID.String()}}
	p, rec := makeTestParamsWithUser(t, http.MethodPost, "/user/holds", []string{"user", "holds"}, &partial, formVals)

	HandleCancelHold(p)

	res := rec.Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	var updated db.Hold
	tx.Where("id = ?", hold.ID).First(&updated)
	assert.Assert(t, !updated.CancelledDate.IsZero(), "CancelledDate must be set after successful cancel")
}
