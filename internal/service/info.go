package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"goProject/internal/repository"
)

// 资讯相关业务错误，handler 据此映射状态码。
var (
	ErrInfoInvalid = errors.New("资讯参数不合法")
	ErrInfoKind    = errors.New("资讯类型只能是 news 或 video")
)

// InfoDetail 是返回给前端的资讯条目。
type InfoDetail struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at"`
	CreatedAt   string `json:"created_at"`
}

// InfoListResult 是资讯列表响应，带分页总数。
type InfoListResult struct {
	Items []InfoDetail `json:"items"`
	Total int64        `json:"total"`
}

// InfoSaveInput 是批量入库的入参。
type InfoSaveInput struct {
	Kind  string         `json:"kind"`
	Items []InfoSaveItem `json:"items"`
}

// InfoSaveItem 是单条待入库资讯。
type InfoSaveItem struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at"`
}

// InfoSaveFailure 记录一条未入库的原因。
type InfoSaveFailure struct {
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

// InfoSaveResult 汇总入库结果：成功条数、跳过条数及每条被跳过的原因。
type InfoSaveResult struct {
	Inserted int               `json:"inserted"`
	Skipped  int               `json:"skipped"`
	Failures []InfoSaveFailure `json:"failures"`
}

// infoStore 是 Info 需要的数据访问能力，由 repository.InfoRepository 实现。
type infoStore interface {
	List(ctx context.Context, kind, keyword string, page, size int) ([]repository.InfoItem, int64, error)
	ExistingPrefixes(ctx context.Context, kind string, prefixes []string) (map[string]bool, error)
	Create(ctx context.Context, item *repository.InfoInput) (int64, error)
}

// InfoCatalog 处理资讯查询与勾选入库。
// 名称避开已有的 service.Info（服务运行信息）。
type InfoCatalog struct {
	items infoStore
}

// NewInfoCatalog 创建 InfoCatalog。
func NewInfoCatalog(items infoStore) *InfoCatalog {
	return &InfoCatalog{items: items}
}

// normalizeInfoKind 校验并归一化类型；allowAll=true 时空值表示「全部」。
func normalizeInfoKind(kind string, allowAll bool) (string, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" && allowAll {
		return "", nil
	}
	if kind != "news" && kind != "video" {
		return "", ErrInfoKind
	}
	return kind, nil
}

func validInfoKind(kind string) (string, error) {
	return normalizeInfoKind(kind, false)
}

// List 分页查询资讯，kind 为空表示全部类型，keyword 按标题模糊匹配。
func (s *InfoCatalog) List(ctx context.Context, kind, keyword string, page, size int) (*InfoListResult, error) {
	kind, err := normalizeInfoKind(kind, true)
	if err != nil {
		return nil, err
	}
	rows, total, err := s.items.List(ctx, kind, strings.TrimSpace(keyword), page, size)
	if err != nil {
		return nil, err
	}

	out := make([]InfoDetail, 0, len(rows))
	for _, row := range rows {
		published := ""
		if row.PublishedAt.Valid {
			published = row.PublishedAt.Time.Format("2006-01-02")
		}
		out = append(out, InfoDetail{
			ID:          row.ID,
			Kind:        row.Kind,
			Title:       row.Title,
			URL:         row.URL,
			Source:      row.Source,
			PublishedAt: published,
			CreatedAt:   row.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &InfoListResult{Items: out, Total: total}, nil
}

// normalizeInfoURL 去空白与末尾斜杠，供去重和展示统一使用。
func normalizeInfoURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

// validateInfoURL 只接受 http/https 且带主机名的完整链接。
func validateInfoURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("链接必须以 http:// 或 https:// 开头且包含域名")
	}
	if len(raw) > 1000 {
		return "", fmt.Errorf("链接过长（最多 1000 字符）")
	}
	return normalizeInfoURL(raw), nil
}

// urlPrefix 取用于去重判断的 URL 前缀，与数据库唯一索引保持一致。
func urlPrefix(raw string) string {
	if len(raw) > repository.URLPrefixLen {
		return raw[:repository.URLPrefixLen]
	}
	return raw
}

// Save 校验候选条目并写入数据库，返回成功与跳过的明细。
// 全部条目不合法或重复时 inserted=0，不会新增任何记录。
func (s *InfoCatalog) Save(ctx context.Context, input InfoSaveInput) (*InfoSaveResult, error) {
	kind, err := validInfoKind(input.Kind)
	if err != nil {
		return nil, err
	}
	if len(input.Items) == 0 || len(input.Items) > 50 {
		return nil, fmt.Errorf("%w: 条目数量需在 1-50 条之间", ErrInfoInvalid)
	}

	result := &InfoSaveResult{Failures: make([]InfoSaveFailure, 0)}
	type candidate struct {
		item   repository.InfoInput
		prefix string
	}
	candidates := make([]candidate, 0, len(input.Items))
	batchSeen := make(map[string]bool, len(input.Items))

	// 第一遍：本地校验 + 批内去重
	for _, raw := range input.Items {
		title := strings.TrimSpace(raw.Title)
		if title == "" {
			result.Failures = append(result.Failures, InfoSaveFailure{Title: title, Reason: "标题为空"})
			continue
		}
		if utf8.RuneCountInString(title) > 200 {
			result.Failures = append(result.Failures, InfoSaveFailure{Title: title, Reason: "标题过长（最多 200 字）"})
			continue
		}
		link, err := validateInfoURL(raw.URL)
		if err != nil {
			result.Failures = append(result.Failures, InfoSaveFailure{Title: title, Reason: err.Error()})
			continue
		}
		prefix := urlPrefix(link)
		if batchSeen[prefix] {
			result.Failures = append(result.Failures, InfoSaveFailure{Title: title, Reason: "本次请求中链接重复"})
			continue
		}
		batchSeen[prefix] = true

		published := strings.TrimSpace(raw.PublishedAt)
		if published != "" {
			if _, err := time.Parse("2006-01-02", published); err != nil {
				published = ""
			}
		}
		candidates = append(candidates, candidate{
			item: repository.InfoInput{
				Kind:        kind,
				Title:       title,
				URL:         link,
				Source:      strings.TrimSpace(raw.Source),
				PublishedAt: published,
			},
			prefix: prefix,
		})
	}

	// 第二遍：查库去重，已存在的不再插入
	if len(candidates) > 0 {
		prefixes := make([]string, 0, len(candidates))
		for _, c := range candidates {
			prefixes = append(prefixes, c.prefix)
		}
		existing, err := s.items.ExistingPrefixes(ctx, kind, prefixes)
		if err != nil {
			return nil, err
		}

		for _, c := range candidates {
			if existing[c.prefix] {
				result.Failures = append(result.Failures, InfoSaveFailure{Title: c.item.Title, Reason: "数据库中已存在相同链接"})
				continue
			}
			_, err := s.items.Create(ctx, &c.item)
			if errors.Is(err, repository.ErrDuplicate) {
				result.Failures = append(result.Failures, InfoSaveFailure{Title: c.item.Title, Reason: "数据库中已存在相同链接"})
				continue
			}
			if err != nil {
				return nil, err
			}
			result.Inserted++
		}
	}

	result.Skipped = len(result.Failures)
	return result, nil
}
