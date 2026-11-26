package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrNotFound 表示记录不存在，上层据此区分“没查到”和“查询出错”。
var ErrNotFound = errors.New("记录不存在")

// User 对应 users 表的一行。
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Nickname     string
	Email        sql.NullString
	Role         string
	Status       int8
	LastLoginAt  sql.NullTime
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserRepository 提供 users 表的数据访问。
type UserRepository struct {
	store *Store
}

// NewUserRepository 创建 UserRepository。
func NewUserRepository(store *Store) *UserRepository {
	return &UserRepository{store: store}
}

const userColumns = `id, username, password_hash, nickname, email, role, status, last_login_at, created_at, updated_at`

// ByUsername 按用户名查询，不存在返回 ErrNotFound。
func (r *UserRepository) ByUsername(ctx context.Context, username string) (*User, error) {
	row := r.store.DB().QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE username = ? LIMIT 1`, username)

	return scanUser(row)
}

// ByID 按主键查询，不存在返回 ErrNotFound。
func (r *UserRepository) ByID(ctx context.Context, id int64) (*User, error) {
	row := r.store.DB().QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = ? LIMIT 1`, id)

	return scanUser(row)
}

// UpdateLastLogin 更新最后登录时间。
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id int64, at time.Time) error {
	_, err := r.store.DB().ExecContext(ctx,
		`UPDATE users SET last_login_at = ? WHERE id = ?`, at, id)
	if err != nil {
		return fmt.Errorf("更新最后登录时间失败: %w", err)
	}
	return nil
}

// UpdatePassword 更新密码哈希。
func (r *UserRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	res, err := r.store.DB().ExecContext(ctx,
		`UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, id)
	if err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取更新结果失败: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// scanUser 把一行结果扫描成 User。
func scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Nickname,
		&u.Email, &u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &u, nil
}

// UserDetail 是用户列表使用的视图，额外带上角色名称。
type UserDetail struct {
	ID          int64
	Username    string
	Nickname    string
	Email       string
	Role        string
	RoleName    string
	Status      int8
	LastLoginAt sql.NullTime
	CreatedAt   time.Time
}

// UserFilter 是用户列表的查询条件。
type UserFilter struct {
	Keyword string // 用户名 / 昵称 / 邮箱模糊匹配
	Role    string // 角色标识精确匹配
	Status  *int8  // 状态精确匹配，nil 表示不限
	Page    int
	Size    int
}

// List 分页查询用户，返回当页数据与总数。
func (r *UserRepository) List(ctx context.Context, f UserFilter) ([]UserDetail, int64, error) {
	where := " WHERE 1 = 1"
	args := make([]any, 0, 4)

	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		where += " AND (u.username LIKE ? OR u.nickname LIKE ? OR u.email LIKE ?)"
		args = append(args, like, like, like)
	}
	if f.Role != "" {
		where += " AND u.role = ?"
		args = append(args, f.Role)
	}
	if f.Status != nil {
		where += " AND u.status = ?"
		args = append(args, *f.Status)
	}

	var total int64
	if err := r.store.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users u`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计用户数量失败: %w", err)
	}

	query := `SELECT u.id, u.username, u.nickname, COALESCE(u.email, ''), u.role,
	                 COALESCE(r.name, u.role), u.status, u.last_login_at, u.created_at
	            FROM users u
	            LEFT JOIN roles r ON r.code = u.role` + where +
		` ORDER BY u.id LIMIT ? OFFSET ?`

	offset := (f.Page - 1) * f.Size
	if offset < 0 {
		offset = 0
	}
	rows, err := r.store.DB().QueryContext(ctx, query, append(args, f.Size, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %w", err)
	}
	defer rows.Close()

	list := make([]UserDetail, 0, f.Size)
	for rows.Next() {
		var item UserDetail
		if err := rows.Scan(&item.ID, &item.Username, &item.Nickname, &item.Email, &item.Role,
			&item.RoleName, &item.Status, &item.LastLoginAt, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描用户失败: %w", err)
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}

// Create 新增用户，用户名重复返回 ErrDuplicate。
func (r *UserRepository) Create(ctx context.Context, u *User) (int64, error) {
	res, err := r.store.DB().ExecContext(ctx,
		`INSERT INTO users (username, password_hash, nickname, email, role, status)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		u.Username, u.PasswordHash, u.Nickname, u.Email, u.Role, u.Status)
	if isDuplicateKey(err) {
		return 0, ErrDuplicate
	}
	if err != nil {
		return 0, fmt.Errorf("新增用户失败: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("读取新增用户 ID 失败: %w", err)
	}
	return id, nil
}

// UpdateProfile 修改昵称、邮箱、角色与状态；用户名不可修改。
func (r *UserRepository) UpdateProfile(ctx context.Context, u *User) error {
	_, err := r.store.DB().ExecContext(ctx,
		`UPDATE users SET nickname = ?, email = ?, role = ?, status = ? WHERE id = ?`,
		u.Nickname, u.Email, u.Role, u.Status, u.ID)
	if err != nil {
		return fmt.Errorf("修改用户失败: %w", err)
	}
	return nil
}

// Delete 删除用户。
func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	if _, err := r.store.DB().ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}
	return nil
}

// CountByRole 统计某角色下的用户数，删除角色前用来兜底。
func (r *UserRepository) CountByRole(ctx context.Context, role string) (int, error) {
	var count int
	if err := r.store.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE role = ?`, role).Scan(&count); err != nil {
		return 0, fmt.Errorf("统计角色用户数失败: %w", err)
	}
	return count, nil
}
