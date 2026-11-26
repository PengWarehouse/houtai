package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"goProject/internal/auth"
	"goProject/internal/repository"
	"goProject/internal/service"
	"goProject/pkg/httpx"
)

type ctxKey string

const ctxKeyClaims ctxKey = "auth_claims"

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// login 处理 POST /api/login。
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "登录服务未配置")
		return
	}

	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	result, err := s.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		s.writeAuthError(w, "login", err)
		return
	}

	httpx.OK(w, result)
}

// profile 处理 GET /api/profile，需要登录。
func (s *Server) profile(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	profile, err := s.auth.Profile(r.Context(), claims.UserID)
	if err != nil {
		s.writeAuthError(w, "profile", err)
		return
	}

	httpx.OK(w, profile)
}

// changePassword 处理 POST /api/password，需要登录。
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	var req changePasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := s.auth.ChangePassword(r.Context(), claims.UserID, req.OldPassword, req.NewPassword); err != nil {
		s.writeAuthError(w, "change_password", err)
		return
	}

	httpx.Success(w, "密码修改成功")
}

// requireAuth 校验 Authorization: Bearer <token>，通过后把 claims 放进 context。
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.tokens == nil {
			httpx.Error(w, http.StatusInternalServerError, "鉴权未配置")
			return
		}

		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			httpx.Error(w, http.StatusUnauthorized, "请先登录")
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" {
			httpx.Error(w, http.StatusUnauthorized, "请先登录")
			return
		}

		claims, err := s.tokens.Parse(token)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "登录已过期，请重新登录")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyClaims, claims)
		next(w, r.WithContext(ctx))
	}
}

// claimsFrom 从 context 取出登录信息。
func claimsFrom(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(ctxKeyClaims).(*auth.Claims)
	return claims
}

// decodeJSON 解析请求体，限制 1MB 防止超大请求。
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(dst); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求参数格式错误")
		return false
	}
	return true
}

// writeAuthError 把业务错误映射成 HTTP 状态码。
func (s *Server) writeAuthError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, service.ErrUserDisabled):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrOldPasswordWrong),
		errors.Is(err, service.ErrPasswordUnchanged),
		errors.Is(err, service.ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "用户不存在")
	default:
		slog.Error("auth handler failed", "action", action, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "服务器内部错误")
	}
}
