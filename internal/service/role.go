package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"goProject/internal/repository"
)

// 角色相关业务错误，handler 据此映射状态码。
var (
	ErrRoleNotFound  = errors.New("角色不存在")
	ErrRoleInvalid   = errors.New("角色参数不合法")
	ErrRoleExists    = errors.New("角色标识已存在")
	ErrRoleProtected = errors.New("超级管理员为系统内置角色，不允许该操作")
	ErrRoleInUse     = errors.New("该角色下还有用户，请先调整用户角色")
)

// roleCodePattern 角色标识只允许小写字母开头 + 小写字母/数字/下划线。
var roleCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}$`)

// RoleDetail 是返回给前端的角色视图。
type RoleDetail struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Status      int8   `json:"status"`
	UserCount   int    `json:"user_count"`
	MenuCount   int    `json:"menu_count"`
	// Builtin 标记系统内置角色（超级管理员），前端据此禁用删除、停用等操作
	Builtin bool `json:"builtin"`
}

// RoleInput 是新增/编辑角色的入参。
type RoleInput struct {
	Code        string
	Name        string
	Description string
	Sort        int
	Status      int8
}

// RoleMenus 是角色权限查询结果。All=true 表示不受菜单限制（超级管理员）。
type RoleMenus struct {
	All     bool    `json:"all"`
	MenuIDs []int64 `json:"menu_ids"`
}

// roleStore 是 Role 需要的数据访问能力，由 repository.RoleRepository 实现。
type roleStore interface {
	List(ctx context.Context) ([]repository.Role, error)
	ByID(ctx context.Context, id int64) (*repository.Role, error)
	ByCode(ctx context.Context, code string) (*repository.Role, error)
	Create(ctx context.Context, role *repository.Role) (int64, error)
	Update(ctx context.Context, role *repository.Role) error
	Delete(ctx context.Context, id int64) error
	MenuIDs(ctx context.Context, roleID int64) ([]int64, error)
	ReplaceMenus(ctx context.Context, roleID int64, menuIDs []int64) error
}

// Role 处理角色的查询与维护。
type Role struct {
	roles roleStore
}

// NewRole 创建 Role。
func NewRole(roles roleStore) *Role {
	return &Role{roles: roles}
}

// List 返回全部角色，查询对所有登录用户开放（用户表单需要选角色）。
func (r *Role) List(ctx context.Context) ([]RoleDetail, error) {
	list, err := r.roles.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]RoleDetail, 0, len(list))
	for _, item := range list {
		out = append(out, toRoleDetail(item))
	}
	return out, nil
}

// Create 新增角色，仅超管可操作。
func (r *Role) Create(ctx context.Context, operatorRole string, in RoleInput) (int64, error) {
	if err := requireSuperAdmin(operatorRole); err != nil {
		return 0, err
	}

	role, err := normalizeRole(in, true)
	if err != nil {
		return 0, err
	}
	if role.Code == RoleSuperAdmin {
		return 0, fmt.Errorf("%w: 超级管理员角色已存在", ErrRoleExists)
	}

	id, err := r.roles.Create(ctx, role)
	if errors.Is(err, repository.ErrDuplicate) {
		return 0, ErrRoleExists
	}
	return id, err
}

// Update 修改角色，仅超管可操作；code 不可修改。
func (r *Role) Update(ctx context.Context, operatorRole string, id int64, in RoleInput) error {
	if err := requireSuperAdmin(operatorRole); err != nil {
		return err
	}

	current, err := r.byID(ctx, id)
	if err != nil {
		return err
	}

	role, err := normalizeRole(in, false)
	if err != nil {
		return err
	}
	role.ID = id
	role.Code = current.Code

	// 超管角色不允许停用，否则整套权限体系会失去唯一的全权限入口
	if current.Code == RoleSuperAdmin && role.Status == 0 {
		return fmt.Errorf("%w: 超级管理员不能被停用", ErrRoleProtected)
	}

	return r.roles.Update(ctx, role)
}

// Delete 删除角色，仅超管可操作；内置角色和仍有用户的角色拒绝删除。
func (r *Role) Delete(ctx context.Context, operatorRole string, id int64) error {
	if err := requireSuperAdmin(operatorRole); err != nil {
		return err
	}

	current, err := r.byID(ctx, id)
	if err != nil {
		return err
	}
	if current.Code == RoleSuperAdmin {
		return fmt.Errorf("%w: 超级管理员角色不能删除", ErrRoleProtected)
	}
	if current.UserCount > 0 {
		return fmt.Errorf("%w（当前 %d 个）", ErrRoleInUse, current.UserCount)
	}

	return r.roles.Delete(ctx, id)
}

// MenuIDs 查询角色已分配的菜单；超级管理员返回 all=true。
func (r *Role) MenuIDs(ctx context.Context, operatorRole string, id int64) (*RoleMenus, error) {
	if err := requireSuperAdmin(operatorRole); err != nil {
		return nil, err
	}

	current, err := r.byID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Code == RoleSuperAdmin {
		return &RoleMenus{All: true, MenuIDs: []int64{}}, nil
	}

	ids, err := r.roles.MenuIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []int64{}
	}
	return &RoleMenus{All: false, MenuIDs: ids}, nil
}

// SetMenus 覆盖角色的菜单权限，仅超管可操作。
func (r *Role) SetMenus(ctx context.Context, operatorRole string, id int64, menuIDs []int64) error {
	if err := requireSuperAdmin(operatorRole); err != nil {
		return err
	}

	current, err := r.byID(ctx, id)
	if err != nil {
		return err
	}
	if current.Code == RoleSuperAdmin {
		return fmt.Errorf("%w: 超级管理员默认拥有全部权限，无需分配", ErrRoleProtected)
	}

	// 去重，避免前端全选时把父级和子级重复传上来
	seen := make(map[int64]struct{}, len(menuIDs))
	clean := make([]int64, 0, len(menuIDs))
	for _, menuID := range menuIDs {
		if menuID <= 0 {
			continue
		}
		if _, ok := seen[menuID]; ok {
			continue
		}
		seen[menuID] = struct{}{}
		clean = append(clean, menuID)
	}

	return r.roles.ReplaceMenus(ctx, id, clean)
}

// byID 查询角色并把「不存在」转成业务错误。
func (r *Role) byID(ctx context.Context, id int64) (*repository.Role, error) {
	current, err := r.roles.ByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrRoleNotFound
	}
	return current, err
}

// normalizeRole 校验入参并补默认值。create 为 false 时不校验 code（不允许修改）。
func normalizeRole(in RoleInput, create bool) (*repository.Role, error) {
	name := strings.TrimSpace(in.Name)
	description := strings.TrimSpace(in.Description)
	code := strings.TrimSpace(in.Code)

	if name == "" {
		return nil, fmt.Errorf("%w: 角色名称不能为空", ErrRoleInvalid)
	}
	if len([]rune(name)) > 64 {
		return nil, fmt.Errorf("%w: 角色名称不能超过 64 个字符", ErrRoleInvalid)
	}
	if len([]rune(description)) > 255 {
		return nil, fmt.Errorf("%w: 角色说明不能超过 255 个字符", ErrRoleInvalid)
	}

	if create {
		if !roleCodePattern.MatchString(code) {
			return nil, fmt.Errorf("%w: 角色标识需以小写字母开头，仅含小写字母、数字、下划线，长度 2-32", ErrRoleInvalid)
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

	return &repository.Role{
		Code:        code,
		Name:        name,
		Description: description,
		Sort:        sort,
		Status:      status,
	}, nil
}

// toRoleDetail 转成对外视图，内置角色带上 builtin 标记。
func toRoleDetail(role repository.Role) RoleDetail {
	return RoleDetail{
		ID:          role.ID,
		Code:        role.Code,
		Name:        role.Name,
		Description: role.Description,
		Sort:        role.Sort,
		Status:      role.Status,
		UserCount:   role.UserCount,
		MenuCount:   role.MenuCount,
		Builtin:     role.Code == RoleSuperAdmin,
	}
}
