package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"goProject/internal/auth"
	"goProject/internal/config"
	"goProject/internal/service"
	"goProject/pkg/httpx"
)

// fakeDB 用于在不依赖真实数据库的情况下测试 handler。
type fakeDB struct {
	down bool
}

func (f fakeDB) Ping(ctx context.Context) error {
	if f.down {
		return errors.New("connection refused")
	}
	return nil
}

func (f fakeDB) Version(ctx context.Context) (string, error) {
	if f.down {
		return "", errors.New("connection refused")
	}
	return "8.4.9", nil
}

// fakeAuth 模拟 service.Auth。
type fakeAuth struct {
	loginErr error
	chgErr   error
}

func (f fakeAuth) Login(ctx context.Context, username, plainPassword string) (*service.LoginResult, error) {
	if f.loginErr != nil {
		return nil, f.loginErr
	}
	return &service.LoginResult{
		Token:     "issued-token",
		ExpiresIn: 3600,
		User:      service.Profile{ID: 1, Username: username, Nickname: "超级管理员", Role: "super_admin"},
	}, nil
}

func (f fakeAuth) Profile(ctx context.Context, userID int64) (*service.Profile, error) {
	return &service.Profile{ID: userID, Username: "admin", Role: "super_admin"}, nil
}

func (f fakeAuth) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	return f.chgErr
}

// fakeTokens 只认 valid-token，其余一律视为无效。
type fakeTokens struct{}

func (fakeTokens) Parse(token string) (*auth.Claims, error) {
	if token == "valid-token" {
		return &auth.Claims{UserID: 1, Username: "admin", Role: "super_admin"}, nil
	}
	return nil, errors.New("token invalid")
}

func newTestRouter(db Database, authSvc AuthService) http.Handler {
	svc := service.New(&config.Config{AppName: "goProject-test", Env: "test"})
	return NewRouter(Deps{Service: svc, DB: db, Auth: authSvc, Tokens: fakeTokens{}})
}

func TestHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(fakeDB{}, fakeAuth{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var env httpx.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Message != "healthy" {
		t.Errorf("message = %q, want %q", env.Message, "healthy")
	}
	if env.Code != httpx.CodeSuccess {
		t.Errorf("code = %d, want %d", env.Code, httpx.CodeSuccess)
	}
}

func TestHealthWhenDatabaseDown(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(fakeDB{down: true}, fakeAuth{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHello(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(fakeDB{}, fakeAuth{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/hello?name=Gopher", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var env struct {
		Data service.Message `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Data.Text != "Hello, Gopher! This is goProject-test." {
		t.Errorf("text = %q", env.Data.Text)
	}
}

func TestInfoIncludesDatabaseVersion(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(fakeDB{}, fakeAuth{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/info", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var env struct {
		Data infoResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Data.Database.Status != "up" || env.Data.Database.Version != "8.4.9" {
		t.Errorf("database = %+v", env.Data.Database)
	}
}

func TestInfoMethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(fakeDB{}, fakeAuth{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/info", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestLoginSuccess(t *testing.T) {
	body := `{"username":"admin","password":"Admin@123456"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(body))
	newTestRouter(fakeDB{}, fakeAuth{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var env struct {
		Data service.LoginResult `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Data.Token == "" {
		t.Error("token 为空")
	}
	if env.Data.User.Role != "super_admin" {
		t.Errorf("role = %q", env.Data.User.Role)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	body := `{"username":"admin","password":"wrong"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(body))
	newTestRouter(fakeDB{}, fakeAuth{loginErr: service.ErrInvalidCredentials}).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLoginBadJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader("{not json"))
	newTestRouter(fakeDB{}, fakeAuth{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProfileWithoutToken(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(fakeDB{}, fakeAuth{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/profile", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestProfileWithValidToken(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	newTestRouter(fakeDB{}, fakeAuth{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestChangePasswordSuccess(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/password",
		strings.NewReader(`{"old_password":"Admin@123456","new_password":"NewPass@123"}`))
	req.Header.Set("Authorization", "Bearer valid-token")
	newTestRouter(fakeDB{}, fakeAuth{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestChangePasswordOldWrong(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/password",
		strings.NewReader(`{"old_password":"bad","new_password":"NewPass@123"}`))
	req.Header.Set("Authorization", "Bearer valid-token")
	newTestRouter(fakeDB{}, fakeAuth{chgErr: service.ErrOldPasswordWrong}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// newCORSRouter 构造一个指定跨域白名单的路由。
func newCORSRouter(origins []string) http.Handler {
	svc := service.New(&config.Config{AppName: "goProject-test", Env: "test"})
	return NewRouter(Deps{Service: svc, AllowedOrigins: origins})
}

func TestCORSPreflightAllowedOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/login", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	newCORSRouter([]string{"http://localhost:5173"}).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("allow-origin = %q, want %q", got, "http://localhost:5173")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") {
		t.Errorf("allow-headers = %q, should contain Authorization", got)
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/login", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	newCORSRouter([]string{"http://localhost:5173"}).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("allow-origin = %q, want empty", got)
	}
}

func TestCORSDisabledWithoutOrigin(t *testing.T) {
	rec := httptest.NewRecorder()
	newCORSRouter([]string{"http://localhost:5173"}).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	// 同源请求不该带任何 CORS 头
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("allow-origin = %q, want empty for same-origin request", got)
	}
}

// 响应约定：成功 code=200，失败 code=0 且 message 带失败原因。
func TestEnvelopeCodeConvention(t *testing.T) {
	t.Run("成功返回 200", func(t *testing.T) {
		rec := httptest.NewRecorder()
		newTestRouter(fakeDB{}, fakeAuth{}).ServeHTTP(
			rec, httptest.NewRequest(http.MethodGet, "/api/info", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var env httpx.Envelope
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if env.Code != httpx.CodeSuccess {
			t.Errorf("code = %d, want %d", env.Code, httpx.CodeSuccess)
		}
	})

	t.Run("失败返回 0 并带原因", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/login",
			strings.NewReader(`{"username":"admin","password":"wrong"}`))
		rec := httptest.NewRecorder()
		newTestRouter(fakeDB{}, fakeAuth{loginErr: service.ErrInvalidCredentials}).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
		var env httpx.Envelope
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if env.Code != httpx.CodeFailure {
			t.Errorf("code = %d, want %d", env.Code, httpx.CodeFailure)
		}
		if env.Message != service.ErrInvalidCredentials.Error() {
			t.Errorf("message = %q, want %q", env.Message, service.ErrInvalidCredentials.Error())
		}
	})

	t.Run("无 token 访问受保护接口返回 0", func(t *testing.T) {
		rec := httptest.NewRecorder()
		newTestRouter(fakeDB{}, fakeAuth{}).ServeHTTP(
			rec, httptest.NewRequest(http.MethodGet, "/api/menus", nil))

		var env httpx.Envelope
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if env.Code != httpx.CodeFailure {
			t.Errorf("code = %d, want %d", env.Code, httpx.CodeFailure)
		}
		if env.Message == "" {
			t.Error("message 为空，前端无法展示失败原因")
		}
	})
}
