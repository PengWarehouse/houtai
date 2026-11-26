package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

// ErrDuplicate 表示唯一键冲突（如角色标识重复）。
var ErrDuplicate = errors.New("记录已存在")

// isDuplicateKey 判断是否为 MySQL 唯一键冲突（错误码 1062）。
func isDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// Role 对应 roles 表的一行。
type Role struct {
	ID          int64
	Code        string
	Name        string
	Description string
	Sort        int
	Status      int8
	UserCount   int
	MenuCount   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RoleRepository 提供 roles / role_menus 的数据访问。
type RoleRepository struct {
	store *Store
}

// NewRoleRepository 创建 RoleRepository。
func NewRoleRepository(store *Store) *RoleRepository {
	return &RoleRepository{store: store}
}

const roleColumns = `r.id, r.code, r.name, r.description, r.sort, r.status, r.created_at, r.updated_at`

// 用户数、菜单数在列表里一起带出来，前端不用为每行再发请求
const roleCountColumns = `
	(SELECT COUNT(*) FROM users u WHERE u.role = r.code) AS user_count,
	(SELECT COUNT(*) FROM role_menus rm WHERE rm.role_id = r.id) AS menu_count`

// List 按 sort 升序返回全部角色，附带用户数与已分配菜单数。
func (r *RoleRepository) List(ctx context.Context) ([]Role, error) {
	rows, err := r.store.DB().QueryContext(ctx,
		`SELECT `+roleColumns+`,`+roleCountColumns+` FROM roles r ORDER BY r.sort, r.id`)
	if err != nil {
		return nil, fmt.Errorf("查询角色列表失败: %w", err)
	}
	defer rows.Close()

	list := make([]Role, 0, 8)
	for rows.Next() {
		var item Role
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description,
			&item.Sort, &item.Status, &item.CreatedAt, &item.UpdatedAt,
			&item.UserCount, &item.MenuCount); err != nil {
			return nil, fmt.Errorf("扫描角色失败: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ByID 按主键查询，不存在返回 ErrNotFound。
func (r *RoleRepository) ByID(ctx context.Context, id int64) (*Role, error) {
	row := r.store.DB().QueryRowContext(ctx,
		`SELECT `+roleColumns+`,`+roleCountColumns+` FROM roles r WHERE r.id = ? LIMIT 1`, id)

	var item Role
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Description,
		&item.Sort, &item.Status, &item.CreatedAt, &item.UpdatedAt,
		&item.UserCount, &item.MenuCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}
	return &item, nil
}

// ByCode 按角色标识查询，不存在返回 ErrNotFound。
func (r *RoleRepository) ByCode(ctx context.Context, code string) (*Role, error) {
	row := r.store.DB().QueryRowContext(ctx,
		`SELECT `+roleColumns+`,`+roleCountColumns+` FROM roles r WHERE r.code = ? LIMIT 1`, code)

	var item Role
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Description,
		&item.Sort, &item.Status, &item.CreatedAt, &item.UpdatedAt,
		&item.UserCount, &item.MenuCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}
	return &item, nil
}

// Create 新增角色，code 重复返回 ErrDuplicate。
func (r *RoleRepository) Create(ctx context.Context, role *Role) (int64, error) {
	res, err := r.store.DB().ExecContext(ctx,
		`INSERT INTO roles (code, name, description, sort, status) VALUES (?, ?, ?, ?, ?)`,
		role.Code, role.Name, role.Description, role.Sort, role.Status)
	if isDuplicateKey(err) {
		return 0, ErrDuplicate
	}
	if err != nil {
		return 0, fmt.Errorf("新增角色失败: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("读取新增角色 ID 失败: %w", err)
	}
	return id, nil
}

// Update 修改角色名称、说明、排序与状态；code 不允许修改。
func (r *RoleRepository) Update(ctx context.Context, role *Role) error {
	_, err := r.store.DB().ExecContext(ctx,
		`UPDATE roles SET name = ?, description = ?, sort = ?, status = ? WHERE id = ?`,
		role.Name, role.Description, role.Sort, role.Status, role.ID)
	if err != nil {
		return fmt.Errorf("修改角色失败: %w", err)
	}
	return nil
}

// Delete 删除角色。
func (r *RoleRepository) Delete(ctx context.Context, id int64) error {
	if _, err := r.store.DB().ExecContext(ctx, `DELETE FROM roles WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除角色失败: %w", err)
	}
	return nil
}

// MenuIDs 返回角色已分配的菜单 ID。
func (r *RoleRepository) MenuIDs(ctx context.Context, roleID int64) ([]int64, error) {
	rows, err := r.store.DB().QueryContext(ctx,
		`SELECT menu_id FROM role_menus WHERE role_id = ? ORDER BY menu_id`, roleID)
	if err != nil {
		return nil, fmt.Errorf("查询角色菜单失败: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0, 16)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("扫描角色菜单失败: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// MenuIDsByRoleCode 按角色标识查菜单 ID，用于登录后过滤侧边栏。
func (r *RoleRepository) MenuIDsByRoleCode(ctx context.Context, code string) ([]int64, error) {
	rows, err := r.store.DB().QueryContext(ctx,
		`SELECT rm.menu_id FROM role_menus rm
		   JOIN roles r ON r.id = rm.role_id
		  WHERE r.code = ? ORDER BY rm.menu_id`, code)
	if err != nil {
		return nil, fmt.Errorf("查询角色菜单失败: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0, 16)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("扫描角色菜单失败: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ReplaceMenus 用给定菜单 ID 覆盖角色的权限，事务内先删后插。
func (r *RoleRepository) ReplaceMenus(ctx context.Context, roleID int64, menuIDs []int64) error {
	tx, err := r.store.DB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM role_menus WHERE role_id = ?`, roleID); err != nil {
		return fmt.Errorf("清理角色菜单失败: %w", err)
	}

	for _, menuID := range menuIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO role_menus (role_id, menu_id) VALUES (?, ?)`, roleID, menuID); err != nil {
			if isDuplicateKey(err) {
				continue // 前端可能传了重复 ID，忽略即可
			}
			return fmt.Errorf("保存角色菜单失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}
