package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Menu 对应 menus 表的一行。
type Menu struct {
	ID        int64
	ParentID  int64
	Title     string
	Path      string
	Icon      string
	Component string
	Sort      int
	Status    int8
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MenuRepository 提供 menus 表的数据访问。
type MenuRepository struct {
	store *Store
}

// NewMenuRepository 创建 MenuRepository。
func NewMenuRepository(store *Store) *MenuRepository {
	return &MenuRepository{store: store}
}

const menuColumns = `id, parent_id, title, path, icon, component, sort, status, created_at, updated_at`

// List 返回全部菜单，按 sort、id 升序（构树时直接复用这个顺序）。
func (r *MenuRepository) List(ctx context.Context) ([]Menu, error) {
	rows, err := r.store.DB().QueryContext(ctx,
		`SELECT `+menuColumns+` FROM menus ORDER BY sort, id`)
	if err != nil {
		return nil, fmt.Errorf("查询菜单失败: %w", err)
	}
	defer rows.Close()

	var list []Menu
	for rows.Next() {
		var m Menu
		if err := rows.Scan(
			&m.ID, &m.ParentID, &m.Title, &m.Path, &m.Icon,
			&m.Component, &m.Sort, &m.Status, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描菜单失败: %w", err)
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ByID 按主键查询，不存在返回 ErrNotFound。
func (r *MenuRepository) ByID(ctx context.Context, id int64) (*Menu, error) {
	var m Menu
	err := r.store.DB().QueryRowContext(ctx,
		`SELECT `+menuColumns+` FROM menus WHERE id = ? LIMIT 1`, id).
		Scan(&m.ID, &m.ParentID, &m.Title, &m.Path, &m.Icon,
			&m.Component, &m.Sort, &m.Status, &m.CreatedAt, &m.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询菜单失败: %w", err)
	}
	return &m, nil
}

// Create 插入菜单并返回自增 ID。
func (r *MenuRepository) Create(ctx context.Context, m *Menu) (int64, error) {
	res, err := r.store.DB().ExecContext(ctx,
		`INSERT INTO menus (parent_id, title, path, icon, component, sort, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.ParentID, m.Title, m.Path, m.Icon, m.Component, m.Sort, m.Status)
	if err != nil {
		return 0, fmt.Errorf("新增菜单失败: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("读取新增菜单 ID 失败: %w", err)
	}
	return id, nil
}

// Update 按主键更新菜单，记录不存在返回 ErrNotFound。
func (r *MenuRepository) Update(ctx context.Context, m *Menu) error {
	res, err := r.store.DB().ExecContext(ctx,
		`UPDATE menus SET parent_id = ?, title = ?, path = ?, icon = ?,
		        component = ?, sort = ?, status = ?
		 WHERE id = ?`,
		m.ParentID, m.Title, m.Path, m.Icon, m.Component, m.Sort, m.Status, m.ID)
	if err != nil {
		return fmt.Errorf("更新菜单失败: %w", err)
	}
	return affectedOrNotFound(res, "更新菜单")
}

// Delete 按主键删除菜单，记录不存在返回 ErrNotFound。
func (r *MenuRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.store.DB().ExecContext(ctx, `DELETE FROM menus WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除菜单失败: %w", err)
	}
	return affectedOrNotFound(res, "删除菜单")
}

// HasChildren 判断是否存在子菜单，删除前用来阻止留下孤儿节点。
func (r *MenuRepository) HasChildren(ctx context.Context, id int64) (bool, error) {
	var exists int
	err := r.store.DB().QueryRowContext(ctx,
		`SELECT 1 FROM menus WHERE parent_id = ? LIMIT 1`, id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("查询子菜单失败: %w", err)
	}
	return true, nil
}

// affectedOrNotFound 把“影响行数为 0”统一翻译成 ErrNotFound。
func affectedOrNotFound(res sql.Result, action string) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s结果读取失败: %w", action, err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
