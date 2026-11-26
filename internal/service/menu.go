package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"goProject/internal/repository"
)

// 角色常量，菜单的写操作只允许超管。
const RoleSuperAdmin = "super_admin"

// 菜单相关业务错误，handler 据此映射状态码。
var (
	ErrMenuNotFound    = errors.New("菜单不存在")
	ErrMenuHasChildren = errors.New("该菜单下还有子菜单，请先删除子菜单")
	ErrMenuInvalid     = errors.New("菜单参数不合法")
	ErrForbidden       = errors.New("没有操作权限")
)

// MenuNode 是返回给前端的菜单节点，children 为空时会被省略。
type MenuNode struct {
	ID        int64       `json:"id"`
	ParentID  int64       `json:"parent_id"`
	Title     string      `json:"title"`
	Path      string      `json:"path"`
	Icon      string      `json:"icon"`
	Component string      `json:"component"`
	Sort      int         `json:"sort"`
	Status    int8        `json:"status"`
	Children  []*MenuNode `json:"children,omitempty"`
}

// MenuInput 是新增/编辑菜单的入参。
type MenuInput struct {
	ParentID  int64
	Title     string
	Path      string
	Icon      string
	Component string
	Sort      int
	Status    int8
}

// menuStore 是 Menu 需要的数据访问能力，由 repository.MenuRepository 实现。
type menuStore interface {
	List(ctx context.Context) ([]repository.Menu, error)
	ByID(ctx context.Context, id int64) (*repository.Menu, error)
	Create(ctx context.Context, m *repository.Menu) (int64, error)
	Update(ctx context.Context, m *repository.Menu) error
	Delete(ctx context.Context, id int64) error
	HasChildren(ctx context.Context, id int64) (bool, error)
}

// menuPermissionStore 提供「某角色能访问哪些菜单」，由 repository.RoleRepository 实现。
type menuPermissionStore interface {
	MenuIDsByRoleCode(ctx context.Context, code string) ([]int64, error)
}

// Menu 处理菜单的查询与维护。
type Menu struct {
	menus       menuStore
	permissions menuPermissionStore
}

// NewMenu 创建 Menu。permissions 为空时不做角色过滤（全部放行）。
func NewMenu(menus menuStore, permissions menuPermissionStore) *Menu {
	return &Menu{menus: menus, permissions: permissions}
}

// Tree 返回完整菜单树（含禁用的菜单，前端按 status 自行过滤）。
func (m *Menu) Tree(ctx context.Context) ([]*MenuNode, error) {
	list, err := m.menus.List(ctx)
	if err != nil {
		return nil, err
	}
	return buildTree(list), nil
}

// TreeForRole 返回指定角色可见的菜单树。
// 超级管理员默认拥有全部权限，直接返回完整树，不查关联表；
// 其他角色按 role_menus 过滤，并自动补齐可见菜单的上级目录，避免子菜单孤悬。
func (m *Menu) TreeForRole(ctx context.Context, roleCode string) ([]*MenuNode, error) {
	list, err := m.menus.List(ctx)
	if err != nil {
		return nil, err
	}

	if roleCode == RoleSuperAdmin || m.permissions == nil {
		return buildTree(list), nil
	}

	allowed, err := m.permissions.MenuIDsByRoleCode(ctx, roleCode)
	if err != nil {
		return nil, err
	}

	visible := visibleMenuIDs(list, allowed)
	filtered := make([]repository.Menu, 0, len(visible))
	for _, item := range list {
		if _, ok := visible[item.ID]; ok {
			filtered = append(filtered, item)
		}
	}
	return buildTree(filtered), nil
}

// visibleMenuIDs 把授权菜单连同其所有上级目录一起标记为可见。
func visibleMenuIDs(list []repository.Menu, allowed []int64) map[int64]struct{} {
	parentOf := make(map[int64]int64, len(list))
	for _, item := range list {
		parentOf[item.ID] = item.ParentID
	}

	visible := make(map[int64]struct{}, len(allowed))
	for _, id := range allowed {
		current := id
		// depth 兜底：数据成环时不要死循环
		for depth := 0; current != 0 && depth < 32; depth++ {
			if _, ok := visible[current]; ok {
				break
			}
			visible[current] = struct{}{}

			parent, ok := parentOf[current]
			if !ok {
				break
			}
			current = parent
		}
	}
	return visible
}

// Create 新增菜单，仅超管可操作。
func (m *Menu) Create(ctx context.Context, role string, in MenuInput) (int64, error) {
	if err := requireSuperAdmin(role); err != nil {
		return 0, err
	}

	menu, err := normalize(in)
	if err != nil {
		return 0, err
	}

	if menu.ParentID != 0 {
		if _, err := m.menus.ByID(ctx, menu.ParentID); err != nil {
			return 0, wrapMenuLookup(err, "上级菜单不存在")
		}
	}

	return m.menus.Create(ctx, menu)
}

// Update 修改菜单，仅超管可操作。
func (m *Menu) Update(ctx context.Context, role string, id int64, in MenuInput) error {
	if err := requireSuperAdmin(role); err != nil {
		return err
	}

	menu, err := normalize(in)
	if err != nil {
		return err
	}
	menu.ID = id

	if _, err := m.menus.ByID(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrMenuNotFound
		}
		return err
	}

	// 防止把菜单挂到自己或自己的子孙下面，形成环
	if err := m.checkParent(ctx, id, menu.ParentID, 0); err != nil {
		return err
	}

	return m.menus.Update(ctx, menu)
}

// Delete 删除菜单，有子菜单时拒绝，避免留下孤儿节点。
func (m *Menu) Delete(ctx context.Context, role string, id int64) error {
	if err := requireSuperAdmin(role); err != nil {
		return err
	}

	if _, err := m.menus.ByID(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrMenuNotFound
		}
		return err
	}

	hasChildren, err := m.menus.HasChildren(ctx, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return ErrMenuHasChildren
	}

	return m.menus.Delete(ctx, id)
}

// checkParent 沿 parent_id 向上走，确认没有绕回自己。
func (m *Menu) checkParent(ctx context.Context, id, parentID int64, depth int) error {
	if parentID == 0 {
		return nil
	}
	if depth > 32 { // 数据已经成环时兜底，避免死循环
		return fmt.Errorf("%w: 菜单层级异常", ErrMenuInvalid)
	}

	parent, err := m.menus.ByID(ctx, parentID)
	if err != nil {
		return wrapMenuLookup(err, "上级菜单不存在")
	}
	if parent.ID == id {
		return fmt.Errorf("%w: 上级菜单不能是自己或自己的子菜单", ErrMenuInvalid)
	}

	return m.checkParent(ctx, id, parent.ParentID, depth+1)
}

// buildTree 把按 sort 排好序的平铺列表组装成树。
// 父级不存在的节点会被当作顶级，避免数据异常导致整棵树丢失。
func buildTree(list []repository.Menu) []*MenuNode {
	nodes := make(map[int64]*MenuNode, len(list))
	for _, item := range list {
		nodes[item.ID] = &MenuNode{
			ID:        item.ID,
			ParentID:  item.ParentID,
			Title:     item.Title,
			Path:      item.Path,
			Icon:      item.Icon,
			Component: item.Component,
			Sort:      item.Sort,
			Status:    item.Status,
		}
	}

	roots := make([]*MenuNode, 0, len(list))
	for _, item := range list {
		node := nodes[item.ID]
		parent, ok := nodes[item.ParentID]
		if item.ParentID == 0 || !ok {
			roots = append(roots, node)
			continue
		}
		parent.Children = append(parent.Children, node)
	}
	return roots
}

// normalize 校验并补默认值，转成 repository 结构。
func normalize(in MenuInput) (*repository.Menu, error) {
	title := strings.TrimSpace(in.Title)
	path := strings.TrimSpace(in.Path)
	component := strings.TrimSpace(in.Component)
	icon := strings.TrimSpace(in.Icon)

	if title == "" {
		return nil, fmt.Errorf("%w: 菜单名称不能为空", ErrMenuInvalid)
	}
	if len([]rune(title)) > 64 {
		return nil, fmt.Errorf("%w: 菜单名称不能超过 64 个字符", ErrMenuInvalid)
	}

	if path == "" || !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("%w: 路由路径必须以 / 开头", ErrMenuInvalid)
	}
	if len(path) > 128 {
		return nil, fmt.Errorf("%w: 路由路径过长", ErrMenuInvalid)
	}

	if component != "" {
		// 只允许 views 下的相对路径，挡住 ../../ 之类跳出目录的写法
		if strings.HasPrefix(component, "/") || strings.Contains(component, "..") {
			return nil, fmt.Errorf("%w: 组件路径应形如 system/menu/index", ErrMenuInvalid)
		}
	}

	status := in.Status
	if status != 0 && status != 1 {
		status = 1
	}

	sort := in.Sort
	if sort == 0 {
		sort = 100
	}

	parentID := in.ParentID
	if parentID < 0 {
		parentID = 0
	}

	return &repository.Menu{
		ParentID:  parentID,
		Title:     title,
		Path:      path,
		Icon:      icon,
		Component: component,
		Sort:      sort,
		Status:    status,
	}, nil
}

// requireSuperAdmin 校验角色。
func requireSuperAdmin(role string) error {
	if role != RoleSuperAdmin {
		return ErrForbidden
	}
	return nil
}

// wrapMenuLookup 把 repository.ErrNotFound 转成业务错误。
func wrapMenuLookup(err error, message string) error {
	if errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("%w: %s", ErrMenuInvalid, message)
	}
	return err
}
