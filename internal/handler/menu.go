package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"goProject/internal/repository"
	"goProject/internal/service"
	"goProject/pkg/httpx"
)

// MenuService 声明 handler 需要的菜单能力，由 service.Menu 实现。
type MenuService interface {
	Tree(ctx context.Context) ([]*service.MenuNode, error)
	TreeForRole(ctx context.Context, roleCode string) ([]*service.MenuNode, error)
	Create(ctx context.Context, role string, in service.MenuInput) (int64, error)
	Update(ctx context.Context, role string, id int64, in service.MenuInput) error
	Delete(ctx context.Context, role string, id int64) error
}

type menuRequest struct {
	ParentID  int64  `json:"parent_id"`
	Title     string `json:"title"`
	Path      string `json:"path"`
	Icon      string `json:"icon"`
	Component string `json:"component"`
	Sort      int    `json:"sort"`
	Status    *int8  `json:"status"` // 指针用于区分“没传”和“传了 0（禁用）”
}

func (r menuRequest) toInput() service.MenuInput {
	status := int8(1)
	if r.Status != nil {
		status = *r.Status
	}
	return service.MenuInput{
		ParentID:  r.ParentID,
		Title:     r.Title,
		Path:      r.Path,
		Icon:      r.Icon,
		Component: r.Component,
		Sort:      r.Sort,
		Status:    status,
	}
}

// listMenus 处理 GET /api/menus，按当前登录用户的角色返回菜单树。
func (s *Server) listMenus(w http.ResponseWriter, r *http.Request) {
	if s.menu == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "菜单服务未配置")
		return
	}

	roleCode := ""
	if claims := claimsFrom(r.Context()); claims != nil {
		roleCode = claims.Role
	}

	tree, err := s.menu.TreeForRole(r.Context(), roleCode)
	if err != nil {
		s.writeMenuError(w, "list_menus", err)
		return
	}

	// 保证空数据也返回 []，前端不用判空
	if tree == nil {
		tree = []*service.MenuNode{}
	}
	httpx.OK(w, tree)
}

// createMenu 处理 POST /api/menus。
func (s *Server) createMenu(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	var req menuRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	id, err := s.menu.Create(r.Context(), claims.Role, req.toInput())
	if err != nil {
		s.writeMenuError(w, "create_menu", err)
		return
	}

	httpx.JSON(w, http.StatusCreated, httpx.Envelope{
		Code:    httpx.CodeSuccess,
		Message: "新增成功",
		Data:    map[string]int64{"id": id},
	})
}

// updateMenu 处理 PUT /api/menus/{id}。
func (s *Server) updateMenu(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req menuRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := s.menu.Update(r.Context(), claims.Role, id, req.toInput()); err != nil {
		s.writeMenuError(w, "update_menu", err)
		return
	}

	httpx.Success(w, "保存成功")
}

// deleteMenu 处理 DELETE /api/menus/{id}。
func (s *Server) deleteMenu(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "请先登录")
		return
	}

	id, ok := pathID(w, r)
	if !ok {
		return
	}

	if err := s.menu.Delete(r.Context(), claims.Role, id); err != nil {
		s.writeMenuError(w, "delete_menu", err)
		return
	}

	httpx.Success(w, "删除成功")
}

// pathID 解析路径参数 {id}。
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "菜单 ID 不合法")
		return 0, false
	}
	return id, true
}

// writeMenuError 把菜单业务错误映射成 HTTP 状态码。
func (s *Server) writeMenuError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrMenuInvalid), errors.Is(err, service.ErrMenuHasChildren):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrMenuNotFound), errors.Is(err, repository.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, service.ErrMenuNotFound.Error())
	default:
		slog.Error("menu handler failed", "action", action, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "服务器内部错误")
	}
}
