package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"goProject/internal/service"
	"goProject/pkg/httpx"
)

// RoleService 声明 handler 需要的角色管理能力，由 service.Role 实现。
type RoleService interface {
	List(ctx context.Context) ([]service.RoleDetail, error)
	Create(ctx context.Context, operatorRole string, in service.RoleInput) (int64, error)
	Update(ctx context.Context, operatorRole string, id int64, in service.RoleInput) error
	Delete(ctx context.Context, operatorRole string, id int64) error
	MenuIDs(ctx context.Context, operatorRole string, id int64) (*service.RoleMenus, error)
	SetMenus(ctx context.Context, operatorRole string, id int64, menuIDs []int64) error
}

type roleRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Status      *int8  `json:"status"`
}

func (r roleRequest) toInput() service.RoleInput {
	status := int8(1)
	if r.Status != nil {
		status = *r.Status
	}
	return service.RoleInput{
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		Sort:        r.Sort,
		Status:      status,
	}
}

type roleMenusRequest struct {
	MenuIDs []int64 `json:"menu_ids"`
}

// listRoles 处理 GET /api/roles，所有登录用户可读（用户表单需要选角色）。
func (s *Server) listRoles(w http.ResponseWriter, r *http.Request) {
	if s.role == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "角色服务未配置")
		return
	}

	list, err := s.role.List(r.Context())
	if err != nil {
		s.writeRoleError(w, "list_roles", err)
		return
	}
	if list == nil {
		list = []service.RoleDetail{}
	}
	httpx.OK(w, list)
}

// createRole 处理 POST /api/roles。
func (s *Server) createRole(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	var req roleRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	id, err := s.role.Create(r.Context(), claims.Role, req.toInput())
	if err != nil {
		s.writeRoleError(w, "create_role", err)
		return
	}

	httpx.JSON(w, http.StatusCreated, httpx.Envelope{
		Code:    httpx.CodeSuccess,
		Message: "新增成功",
		Data:    map[string]int64{"id": id},
	})
}

// updateRole 处理 PUT /api/roles/{id}。
func (s *Server) updateRole(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req roleRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := s.role.Update(r.Context(), claims.Role, id, req.toInput()); err != nil {
		s.writeRoleError(w, "update_role", err)
		return
	}
	httpx.Success(w, "保存成功")
}

// deleteRole 处理 DELETE /api/roles/{id}。
func (s *Server) deleteRole(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	id, ok := pathID(w, r)
	if !ok {
		return
	}

	if err := s.role.Delete(r.Context(), claims.Role, id); err != nil {
		s.writeRoleError(w, "delete_role", err)
		return
	}
	httpx.Success(w, "删除成功")
}

// roleMenus 处理 GET /api/roles/{id}/menus，返回角色已分配的菜单 ID。
func (s *Server) roleMenus(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	id, ok := pathID(w, r)
	if !ok {
		return
	}

	result, err := s.role.MenuIDs(r.Context(), claims.Role, id)
	if err != nil {
		s.writeRoleError(w, "role_menus", err)
		return
	}
	httpx.OK(w, result)
}

// setRoleMenus 处理 PUT /api/roles/{id}/menus，覆盖角色的菜单权限。
func (s *Server) setRoleMenus(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req roleMenusRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := s.role.SetMenus(r.Context(), claims.Role, id, req.MenuIDs); err != nil {
		s.writeRoleError(w, "set_role_menus", err)
		return
	}
	httpx.Success(w, "权限已保存")
}

// writeRoleError 把角色业务错误映射成 HTTP 状态码。
func (s *Server) writeRoleError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrRoleNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrRoleExists), errors.Is(err, service.ErrRoleInUse):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrRoleInvalid), errors.Is(err, service.ErrRoleProtected):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	default:
		slog.Error("role handler failed", "action", action, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "服务器内部错误")
	}
}
