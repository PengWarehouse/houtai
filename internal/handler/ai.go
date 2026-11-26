package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"goProject/internal/service"
	"goProject/pkg/httpx"
)

// AIService 声明 handler 需要的 AI 能力，由 service.AI 实现。
type AIService interface {
	Chat(ctx context.Context, messages []service.ChatMessage) (string, error)
	Extract(ctx context.Context, kind, question string) ([]service.ExtractedItem, error)
}

// aiChat 处理 POST /api/ai/chat，请求体 { messages: [{role, content}] }。
// 密钥只在服务端保存，前端不接触 DeepSeek。
func (s *Server) aiChat(w http.ResponseWriter, r *http.Request) {
	if s.ai == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "AI 服务不可用")
		return
	}

	var req struct {
		Messages []service.ChatMessage `json:"messages"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	// 基础校验：过滤空消息、限制条数和长度，避免被当成免费代理刷 token
	if len(req.Messages) == 0 || len(req.Messages) > 50 {
		httpx.Error(w, http.StatusBadRequest, "消息数量需在 1-50 条之间")
		return
	}
	clean := make([]service.ChatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role != "user" && m.Role != "assistant" && m.Role != "system" {
			httpx.Error(w, http.StatusBadRequest, "消息角色不合法")
			return
		}
		if len(m.Content) == 0 || len(m.Content) > 12000 {
			httpx.Error(w, http.StatusBadRequest, "单条消息长度需在 1-12000 字节之间")
			return
		}
		clean = append(clean, m)
	}

	answer, err := s.ai.Chat(r.Context(), clean)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAINotConfigured):
			httpx.Error(w, http.StatusServiceUnavailable, "AI 服务未配置密钥")
		case errors.Is(err, service.ErrAIUpstream):
			slog.Warn("ai upstream failed", "err", err)
			httpx.Error(w, http.StatusBadGateway, err.Error())
		default:
			slog.Warn("ai chat failed", "err", err)
			httpx.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	httpx.OK(w, map[string]string{"content": answer})
}

// aiExtract 处理 POST /api/ai/extract { kind, question }。
// 只返回候选条目，不写库；是否入库由用户勾选后调用 /api/info/items 决定。
func (s *Server) aiExtract(w http.ResponseWriter, r *http.Request) {
	if s.ai == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "AI 服务不可用")
		return
	}

	var req struct {
		Kind     string `json:"kind"`
		Question string `json:"question"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind != "news" && kind != "video" {
		httpx.Error(w, http.StatusBadRequest, "资讯类型只能是 news 或 video")
		return
	}
	question := strings.TrimSpace(req.Question)
	if question == "" || len(question) > 4000 {
		httpx.Error(w, http.StatusBadRequest, "问询内容长度需在 1-4000 字节之间")
		return
	}

	items, err := s.ai.Extract(r.Context(), kind, question)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAINotConfigured):
			httpx.Error(w, http.StatusServiceUnavailable, "AI 服务未配置密钥")
		case errors.Is(err, service.ErrAIUpstream):
			slog.Warn("ai extract failed", "err", err)
			httpx.Error(w, http.StatusBadGateway, err.Error())
		default:
			slog.Warn("ai extract failed", "err", err)
			httpx.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	if items == nil {
		items = []service.ExtractedItem{}
	}
	httpx.OK(w, map[string]any{"items": items})
}
