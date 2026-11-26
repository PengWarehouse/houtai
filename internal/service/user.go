package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"goProject/internal/repository"
	"goProject/pkg/password"
)

// 用户相关业务错误，handler 据此映射状态码。
var (
	ErrUserNotFound  = errors.New("用户不存在")
	ErrUserInvalid   = errors.New("用户参数不合法")
	ErrUserExists    = errors.New("用户名已存在")
	ErrUserProtected = errors.New("系统内置账号，不允许该操作")
)

var (
	usernamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{2,31}$`)
	emailPattern    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

// UserItem 是返回给前端的用户视图，绝不包含密码哈希。
type UserItem struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Nickname    string     `json:"nickname"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	RoleName    string     `json:"role_name"`
	Status      int8       `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// UserListResult 是分页结果。
type UserListResult struct {
	List  []UserItem `json:"list"`
	Total int64      `json:"total"`
	Page  int        `json:"page"`
	Size  int        `json:"size"`
}

// UserQuery 是用户列表查询条件。
type UserQuery struct {
	Keyword string
	Role    string
	Status  *int8
	Page    int
	Size    int
}

// UserCreateInput 是新增用户入参。
type UserCreateInput struct {
	Username string
	Password string
	Nickname string
	Email    string
	Role     string
	Status   int8
}

// UserUpdateInput 是编辑用户入参；用户名与密码不在这里修改。
type UserUpdateInput struct {
	Nickname string
	Email    string
	Role     string
	Status   int8
}

// userAdminStore 是 User（管理侧）需要的数据访问能力，由 repository.UserRepository 实现。
type userAdminStore interface {
	ByID(ctx context.Context, id int64) (*repository.User, error)
	List(ctx context.Context, f repository.UserFilter) ([]repository.UserDetail, int64, error)
	Create(ctx context.Context, u *repository.User) (int64, error)
	UpdateProfile(ctx context.Context, u *repository.User) error
	Delete(ctx context.Context, id int64) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
}

// roleLookup 用于校验用户所属角色是否存在。
type roleLookup interface {
	ByCode(ctx context.Context, code string) (*repository.Role, error)
}

// User 处理用户的查询与维护。
type User struct {
	users userAdminStore
	roles roleLookup
}

// NewUser 创建 User。
func NewUser(users userAdminStore, roles roleLookup) *User {
	return &User{users: users, roles: roles}
}

// List 分页查询用户。
func (u *User) List(ctx context.Context, q UserQuery) (*UserListResult, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	size := q.Size
	if size < 1 || size > 500 {
		size = 50
	}

	list, total, err := u.users.List(ctx, repository.UserFilter{
		Keyword: strings.TrimSpace(q.Keyword),
		Role:    strings.TrimSpace(q.Role),
		Status:  q.Status,
		Page:    page,
		Size:    size,
	})
	if err != nil {
		return nil, err
	}

	items := make([]UserItem, 0, len(list))
	for _, item := range list {
		items = append(items, toUserItem(item))
	}

	return &UserListResult{List: items, Total: total, Page: page, Size: size}, nil
}

// Create 新增用户，仅超管可操作。
func (u *User) Create(ctx context.Context, operatorRole string, in UserCreateInput) (int64, error) {
	if err := requireSuperAdmin(operatorRole); err != nil {
		return 0, err
	}

	username := strings.TrimSpace(in.Username)
	if !usernamePattern.MatchString(username) {
		return 0, fmt.Errorf("%w: 用户名需以字母开头，仅含字母、数字、下划线，长度 3-32", ErrUserInvalid)
	}

	if err := validatePassword(in.Password); err != nil {
		return 0, err
	}

	nickname, email, role, err := u.normalizeProfile(ctx, in.Nickname, in.Email, in.Role)
	if err != nil {
		return 0, err
	}

	hash, err := password.Hash(in.Password)
	if err != nil {
		return 0, err
	}

	id, err := u.users.Create(ctx, &repository.User{
		Username:     username,
		PasswordHash: hash,
		Nickname:     nickname,
		Email:        nullString(email),
		Role:         role,
		Status:       normalizeStatus(in.Status),
	})
	if errors.Is(err, repository.ErrDuplicate) {
		return 0, ErrUserExists
	}
	return id, err
}

// Update 修改用户资料，仅超管可操作；不允许改动自己的角色或状态，避免把自己锁在外面。
func (u *User) Update(ctx context.Context, operatorID int64, operatorRole string, id int64, in UserUpdateInput) error {
	if err := requireSuperAdmin(operatorRole); err != nil {
		return err
	}

	current, err := u.byID(ctx, id)
	if err != nil {
		return err
	}

	nickname, email, role, err := u.normalizeProfile(ctx, in.Nickname, in.Email, in.Role)
	if err != nil {
		return err
	}
	status := normalizeStatus(in.Status)

	if current.ID == operatorID {
		if role != current.Role {
			return fmt.Errorf("%w: 不能修改自己的角色", ErrUserInvalid)
		}
		if status == 0 {
			return fmt.Errorf("%w: 不能停用当前登录账号", ErrUserInvalid)
		}
	}

	// 内置超管账号保留角色与启用状态，防止权限体系被改坏
	if current.Role == RoleSuperAdmin && (role != RoleSuperAdmin || status == 0) {
		return fmt.Errorf("%w: 不能修改超级管理员的角色或状态", ErrUserProtected)
	}

	current.Nickname = nickname
	current.Email = nullString(email)
	current.Role = role
	current.Status = status

	return u.users.UpdateProfile(ctx, current)
}

// Delete 删除用户，仅超管可操作；不能删除自己，也不能删除内置超管账号。
func (u *User) Delete(ctx context.Context, operatorID int64, operatorRole string, id int64) error {
	if err := requireSuperAdmin(operatorRole); err != nil {
		return err
	}

	if id == operatorID {
		return fmt.Errorf("%w: 不能删除当前登录账号", ErrUserInvalid)
	}

	current, err := u.byID(ctx, id)
	if err != nil {
		return err
	}
	if current.Role == RoleSuperAdmin {
		return fmt.Errorf("%w: 超级管理员账号不能删除", ErrUserProtected)
	}

	return u.users.Delete(ctx, id)
}

// ResetPassword 由超管直接重置他人密码。
func (u *User) ResetPassword(ctx context.Context, operatorRole string, id int64, newPassword string) error {
	if err := requireSuperAdmin(operatorRole); err != nil {
		return err
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	if _, err := u.byID(ctx, id); err != nil {
		return err
	}

	hash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}
	return u.users.UpdatePassword(ctx, id, hash)
}

// byID 查询用户并把「不存在」转成业务错误。
func (u *User) byID(ctx context.Context, id int64) (*repository.User, error) {
	current, err := u.users.ByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	return current, err
}

// normalizeProfile 校验昵称、邮箱、角色三个共用字段。
func (u *User) normalizeProfile(ctx context.Context, nickname, email, role string) (string, string, string, error) {
	nickname = strings.TrimSpace(nickname)
	email = strings.TrimSpace(email)
	role = strings.TrimSpace(role)

	if nickname == "" {
		return "", "", "", fmt.Errorf("%w: 昵称不能为空", ErrUserInvalid)
	}
	if len([]rune(nickname)) > 64 {
		return "", "", "", fmt.Errorf("%w: 昵称不能超过 64 个字符", ErrUserInvalid)
	}
	if email != "" {
		if !emailPattern.MatchString(email) || len(email) > 128 {
			return "", "", "", fmt.Errorf("%w: 邮箱格式不正确", ErrUserInvalid)
		}
	}
	if role == "" {
		return "", "", "", fmt.Errorf("%w: 请选择角色", ErrUserInvalid)
	}
	if _, err := u.roles.ByCode(ctx, role); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", "", "", fmt.Errorf("%w: 角色不存在", ErrUserInvalid)
		}
		return "", "", "", err
	}

	return nickname, email, role, nil
}

// validatePassword 密码长度限制：下限防弱口令，上限是 bcrypt 只处理前 72 字节的硬限制。
func validatePassword(plain string) error {
	if len(plain) < 6 || len(plain) > 72 {
		return fmt.Errorf("%w: 密码长度需为 6-72 个字符", ErrUserInvalid)
	}
	return nil
}

// normalizeStatus 只接受 0/1，其余一律按启用处理。
func normalizeStatus(status int8) int8 {
	if status == 0 {
		return 0
	}
	return 1
}

func toUserItem(item repository.UserDetail) UserItem {
	out := UserItem{
		ID:        item.ID,
		Username:  item.Username,
		Nickname:  item.Nickname,
		Email:     item.Email,
		Role:      item.Role,
		RoleName:  item.RoleName,
		Status:    item.Status,
		CreatedAt: item.CreatedAt,
	}
	if item.LastLoginAt.Valid {
		t := item.LastLoginAt.Time
		out.LastLoginAt = &t
	}
	return out
}

func nullString(v string) sql.NullString {
	return sql.NullString{String: v, Valid: v != ""}
}
