package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func newTestAuth(t *testing.T) (*Auth, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	auth, err := NewAuth(db, nil)
	if err != nil {
		t.Fatal(err)
	}
	return auth, db
}

func TestCreateUsersTable(t *testing.T) {
	_, db := newTestAuth(t)
	defer db.Close()

	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&name)
	if err != nil || name != "users" {
		t.Fatal("users table not created")
	}
}

func TestDefaultUserCreated(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	if !auth.HasUsers() {
		t.Fatal("expected HasUsers() = true after NewAuth (default admin)")
	}

	user, err := auth.Authenticate("admin", "admin123")
	if err != nil {
		t.Fatal("default admin should authenticate:", err)
	}
	if user.Username != "admin" || user.Role != "admin" {
		t.Fatal("unexpected default admin data")
	}
}

func TestCreateUserAndHasUsers(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	user, err := auth.CreateUser("joao", "senha123", "user")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "joao" || user.Role != "user" {
		t.Fatal("unexpected user data")
	}
	if !auth.HasUsers() {
		t.Fatal("expected HasUsers() = true")
	}
}

func TestCreateUserDuplicate(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	_, err := auth.CreateUser("admin", "outra456", "user")
	if err == nil {
		t.Fatal("expected error for duplicate username")
	}
	if !strings.Contains(err.Error(), "ja existe") {
		t.Fatal("expected 'ja existe' error, got:", err)
	}
}

func TestCreateUserValidation(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	_, err := auth.CreateUser("ab", "senha123", "admin")
	if err == nil {
		t.Fatal("expected error for short username")
	}

	_, err = auth.CreateUser("teste", "12345", "admin")
	if err == nil {
		t.Fatal("expected error for short password")
	}

	_, err = auth.CreateUser("teste", "senha123", "superuser")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestAuthenticate(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	user, err := auth.Authenticate("admin", "admin123")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "admin" || user.Role != "admin" {
		t.Fatal("unexpected user data after auth")
	}
}

func TestAuthenticateBadPassword(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	_, err := auth.Authenticate("admin", "wrongpass")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestAuthenticateUnknownUser(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	_, err := auth.Authenticate("ghost", "senha123")
	if err == nil {
		t.Fatal("expected error for unknown user")
	}
}

func TestListUsers(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	auth.CreateUser("joao", "senha456", "user")

	users, err := auth.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatal("expected 2 users, got:", len(users))
	}
	for _, u := range users {
		if u.PasswordHash != "" {
			t.Fatal("password hash should not be returned")
		}
	}
}

func TestUpdatePassword(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	user, _ := auth.Authenticate("admin", "admin123")

	err := auth.UpdatePassword(user.ID, "newpass123")
	if err != nil {
		t.Fatal(err)
	}

	_, err = auth.Authenticate("admin", "admin123")
	if err == nil {
		t.Fatal("old password should not work")
	}

	_, err = auth.Authenticate("admin", "newpass123")
	if err != nil {
		t.Fatal("new password should work:", err)
	}
}

func TestUpdateRole(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	admin, _ := auth.Authenticate("admin", "admin123")
	user, _ := auth.CreateUser("joao", "senha456", "user")

	err := auth.UpdateRole(user.ID, "admin")
	if err != nil {
		t.Fatal(err)
	}

	err = auth.UpdateRole(admin.ID, "user")
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateRoleLastAdmin(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	admin, _ := auth.Authenticate("admin", "admin123")
	err := auth.UpdateRole(admin.ID, "user")
	if err == nil {
		t.Fatal("expected error when removing last admin")
	}
	if !strings.Contains(err.Error(), "ultimo admin") {
		t.Fatal("expected 'ultimo admin' error, got:", err)
	}
}

func TestDeleteUser(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	user, _ := auth.CreateUser("joao", "senha456", "user")

	err := auth.DeleteUser(user.ID)
	if err != nil {
		t.Fatal(err)
	}

	users, _ := auth.ListUsers()
	if len(users) != 1 {
		t.Fatal("expected 1 user after delete, got:", len(users))
	}
}

func TestDeleteUserLastAdmin(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	admin, _ := auth.Authenticate("admin", "admin123")
	err := auth.DeleteUser(admin.ID)
	if err == nil {
		t.Fatal("expected error when deleting last admin")
	}
	if !strings.Contains(err.Error(), "ultimo admin") {
		t.Fatal("expected 'ultimo admin' error, got:", err)
	}
}

func TestCreateAndGetSession(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	user, _ := auth.Authenticate("admin", "admin123")
	token := auth.CreateSession(user)

	if len(token) != 64 {
		t.Fatal("expected 64-char hex token, got length:", len(token))
	}

	session := auth.GetSession(token)
	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if session.Username != "admin" || session.Role != "admin" {
		t.Fatal("unexpected session data")
	}
}

func TestExpiredSessionNil(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	user, _ := auth.Authenticate("admin", "admin123")
	token := auth.CreateSession(user)

	auth.mu.Lock()
	auth.sessions[token].Expiry = time.Now().Add(-1 * time.Hour).Unix()
	auth.mu.Unlock()

	session := auth.GetSession(token)
	if session != nil {
		t.Fatal("expected nil for expired session")
	}
}

func TestDestroySession(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	user, _ := auth.Authenticate("admin", "admin123")
	token := auth.CreateSession(user)

	auth.DestroySession(token)
	if auth.GetSession(token) != nil {
		t.Fatal("session should be destroyed")
	}
}

func TestDestroyUserSessions(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	user, _ := auth.Authenticate("admin", "admin123")
	t1 := auth.CreateSession(user)
	t2 := auth.CreateSession(user)

	auth.DestroyUserSessions(user.ID)
	if auth.GetSession(t1) != nil || auth.GetSession(t2) != nil {
		t.Fatal("all user sessions should be destroyed")
	}
}

func TestMiddlewareBlocksNoCookie(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	handler := auth.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/config", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatal("expected 401 for API without cookie, got:", rr.Code)
	}

	req = httptest.NewRequest("GET", "/", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 302 {
		t.Fatal("expected 302 redirect for page without cookie, got:", rr.Code)
	}
}

func TestMiddlewareAllowsValidCookie(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	user, _ := auth.Authenticate("admin", "admin123")
	token := auth.CreateSession(user)

	var gotSession *AuthSession
	handler := auth.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		gotSession = GetSessionFromCtx(r)
		rw.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/config", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatal("expected 200 with valid cookie, got:", rr.Code)
	}
	if gotSession == nil || gotSession.Username != "admin" {
		t.Fatal("expected session in context")
	}
}

func TestMiddlewareExemptRoutes(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	handler := auth.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(200)
	}))

	exemptPaths := []string{"/login", "/api/login", "/api/auth/status", "/static/style.css", "/static/foo.js"}
	for _, path := range exemptPaths {
		req := httptest.NewRequest("GET", path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Fatalf("expected 200 for exempt path %s, got: %d", path, rr.Code)
		}
	}
}

func TestRequireAdminFunc(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	adminUser, _ := auth.Authenticate("admin", "admin123")
	normalUser, _ := auth.CreateUser("joao", "senha456", "user")
	adminToken := auth.CreateSession(adminUser)
	userToken := auth.CreateSession(normalUser)

	handler := auth.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !RequireAdmin(rw, r) {
			return
		}
		rw.WriteHeader(200)
	}))

	req := httptest.NewRequest("POST", "/api/config", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: adminToken})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatal("expected 200 for admin, got:", rr.Code)
	}

	req = httptest.NewRequest("POST", "/api/config", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: userToken})
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 403 {
		t.Fatal("expected 403 for user, got:", rr.Code)
	}

	req = httptest.NewRequest("POST", "/api/config", nil)
	rr = httptest.NewRecorder()
	http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !RequireAdmin(rw, r) {
			return
		}
		rw.WriteHeader(200)
	}).ServeHTTP(rr, req)
	if rr.Code != 403 {
		t.Fatal("expected 403 for no session, got:", rr.Code)
	}
}

func TestLoginHandler(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	body := `{"username":"admin","password":"admin123"}`
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	auth.HandleLogin(rr, req)

	if rr.Code != 200 {
		t.Fatal("expected 200 for correct login, got:", rr.Code)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Fatal("expected ok=true")
	}
	if resp["role"] != "admin" {
		t.Fatal("expected role=admin")
	}
	cookies := rr.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == authCookieName {
			found = true
		}
	}
	if !found {
		t.Fatal("expected session cookie to be set")
	}

	body = `{"username":"admin","password":"wrong"}`
	req = httptest.NewRequest("POST", "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	auth.HandleLogin(rr, req)

	if rr.Code != 401 {
		t.Fatal("expected 401 for incorrect login, got:", rr.Code)
	}
}

func TestLogoutHandler(t *testing.T) {
	auth, db := newTestAuth(t)
	defer db.Close()

	user, _ := auth.Authenticate("admin", "admin123")
	token := auth.CreateSession(user)

	req := httptest.NewRequest("POST", "/api/logout", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	rr := httptest.NewRecorder()
	auth.HandleLogout(rr, req)

	if rr.Code != 200 {
		t.Fatal("expected 200, got:", rr.Code)
	}

	if auth.GetSession(token) != nil {
		t.Fatal("session should be destroyed after logout")
	}
}
