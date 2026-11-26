package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"goProject/internal/service"
	"goProject/pkg/httpx"
)

// UserService 声明 handler 需要的用户管理能力，由 service.User 实现。
type UserService interface {
	List(ctx context.Context, q service.UserQuery) (*service.UserListResult, error)
	Create(ctx context.Context, operatorRole string, in service.UserCreateInput) (int64, error)
	Update(ctx context.Context, operatorID int64, operatorRole string, id int64, in service.UserUpdateInput) error
	Delete(ctx context.Context, operatorID int64, operatorRole string, id int64) error
	ResetPassword(ctx context.Context, operatorRole string, id int64, newPassword string) error
}

type userCreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   *int8  `json:"status"`
}

type userUpdateRequest struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   *int8  `json:"status"`
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

// listUsers 处理 GET /api/users，支持关键字、角色、状态筛选与分页。
func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	if s.user == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "用户服务未配置")
		return
	}

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	size, _ := strconv.Atoi(query.Get("size"))

	var status *int8
	if raw := query.Get("status"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && (v == 0 || v == 1) {
			value := int8(v)
			status = &value
		}
	}

	result, err := s.user.List(r.Context(), service.UserQuery{
		Keyword: query.Get("keyword"),
		Role:    query.Get("role"),
		Status:  status,
		Page:    page,
		Size:    size,
	})
	if err != nil {
		s.writeUserError(w, "list_users", err)
		return
	}
	httpx.OK(w, result)
}

// createUser 处理 POST /api/users。
func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	var req userCreateRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	status := int8(1)
	if req.Status != nil {
		status = *req.Status
	}

	id, err := s.user.Create(r.Context(), claims.Role, service.UserCreateInput{
		Username: req.Username,
		Password: req.Password,
		Nickname: req.Nickname,
		Email:    req.Email,
		Role:     req.Role,
		Status:   status,
	})
	if err != nil {
		s.writeUserError(w, "create_user", err)
		return
	}

	httpx.JSON(w, http.StatusCreated, httpx.Envelope{
		Code:    httpx.CodeSuccess,
		Message: "新增成功",
		Data:    map[string]int64{"id": id},
	})
}

// updateUser 处理 PUT /api/users/{id}。
func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req userUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	status := int8(1)
	if req.Status != nil {
		status = *req.Status
	}

	if err := s.user.Update(r.Context(), claims.UserID, claims.Role, id, service.UserUpdateInput{
		Nickname: req.Nickname,
		Email:    req.Email,
		Role:     req.Role,
		Status:   status,
	}); err != nil {
		s.writeUserError(w, "update_user", err)
		return
	}

	httpx.Success(w, "保存成功")
}

// deleteUser 处理 DELETE /api/users/{id}。
func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	id, ok := pathID(w, r)
	if !ok {
		return
	}

	if err := s.user.Delete(r.Context(), claims.UserID, claims.Role, id); err != nil {
		s.writeUserError(w, "delete_user", err)
		return
	}
	httpx.Success(w, "删除成功")
}

// resetUserPassword 处理 POST /api/users/{id}/password，由超管直接重置密码。
func (s *Server) resetUserPassword(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req resetPasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := s.user.ResetPassword(r.Context(), claims.Role, id, req.Password); err != nil {
		s.writeUserError(w, "reset_password", err)
		return
	}
	httpx.Success(w, "密码已重置")
}

// writeUserError 把用户业务错误映射成 HTTP 状态码。
func (s *Server) writeUserError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrUserNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrUserExists):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrUserInvalid), errors.Is(err, service.ErrUserProtected):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	default:
		slog.Error("user handler failed", "action", action, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "服务器内部错误")
	}
}
