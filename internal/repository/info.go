package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// InfoItem 对应 info_items 表的一行。
type InfoItem struct {
	ID          int64
	Kind        string
	Title       string
	URL         string
	Source      string
	PublishedAt sql.NullTime
	CreatedAt   time.Time
}

// InfoInput 是写入资讯条目的入参。
type InfoInput struct {
	Kind        string
	Title       string
	URL         string
	Source      string
	PublishedAt string // YYYY-MM-DD，允许为空
}

// URLPrefixLen 与 info_items 表的唯一索引前缀保持一致，用于去重判断。
const URLPrefixLen = 191

// InfoRepository 提供 info_items 表的数据访问。
type InfoRepository struct {
	store *Store
}

// NewInfoRepository 创建 InfoRepository。
func NewInfoRepository(store *Store) *InfoRepository {
	return &InfoRepository{store: store}
}

// List 分页查询资讯：kind 为空表示不限类型，keyword 只匹配标题。
// page 从 1 开始；size <= 0 时使用默认每页 10 条。返回当页数据与总数。
func (r *InfoRepository) List(ctx context.Context, kind, keyword string, page, size int) ([]InfoItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	where := " WHERE 1 = 1"
	args := make([]any, 0, 2)
	if kind != "" {
		where += ` AND kind = ?`
		args = append(args, kind)
	}
	if keyword != "" {
		where += ` AND title LIKE ?`
		args = append(args, "%"+keyword+"%")
	}

	var total int64
	if err := r.store.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM info_items`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计资讯数量失败: %w", err)
	}

	query := `SELECT id, kind, title, url, source, published_at, created_at
	            FROM info_items` + where + ` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
	dataArgs := append(append(make([]any, 0, len(args)+2), args...), size, (page-1)*size)

	rows, err := r.store.DB().QueryContext(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询资讯列表失败: %w", err)
	}
	defer rows.Close()

	list := make([]InfoItem, 0, size)
	for rows.Next() {
		var item InfoItem
		if err := rows.Scan(&item.ID, &item.Kind, &item.Title, &item.URL,
			&item.Source, &item.PublishedAt, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描资讯条目失败: %w", err)
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}

// ExistingPrefixes 返回该类型下已入库的 URL 前缀集合，用于批量去重。
func (r *InfoRepository) ExistingPrefixes(ctx context.Context, kind string, prefixes []string) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(prefixes) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(prefixes))
	args := make([]any, 0, len(prefixes)+1)
	args = append(args, kind)
	for i, p := range prefixes {
		placeholders[i] = "?"
		args = append(args, p)
	}

	query := `SELECT LEFT(url, ` + fmt.Sprintf("%d", URLPrefixLen) + `) FROM info_items
	          WHERE kind = ? AND LEFT(url, ` + fmt.Sprintf("%d", URLPrefixLen) + `) IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := r.store.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询已存在资讯失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var prefix string
		if err := rows.Scan(&prefix); err != nil {
			return nil, fmt.Errorf("扫描已存在资讯失败: %w", err)
		}
		result[prefix] = true
	}
	return result, rows.Err()
}

// Create 新增一条资讯，URL 前缀重复时返回 ErrDuplicate。
func (r *InfoRepository) Create(ctx context.Context, item *InfoInput) (int64, error) {
	var publishedAt any
	if item.PublishedAt != "" {
		publishedAt = item.PublishedAt
	}

	res, err := r.store.DB().ExecContext(ctx,
		`INSERT INTO info_items (kind, title, url, source, published_at) VALUES (?, ?, ?, ?, ?)`,
		item.Kind, item.Title, item.URL, item.Source, publishedAt)
	if isDuplicateKey(err) {
		return 0, ErrDuplicate
	}
	if err != nil {
		return 0, fmt.Errorf("新增资讯条目失败: %w", err)
	}
	return res.LastInsertId()
}
