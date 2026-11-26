package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"goProject/internal/config"
)

// AI 相关的哨兵错误，handler 据此映射 HTTP 状态码。
var (
	ErrAINotConfigured = errors.New("AI 服务未配置密钥")
	ErrAIUpstream      = errors.New("AI 上游请求失败")
)

// ChatMessage 是 OpenAI 兼容格式的单条消息。
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AI 封装对 DeepSeek（OpenAI 兼容）接口的调用。
type AI struct {
	apiKey  string
	baseURL string
	model   string
	timeout time.Duration
	client  *http.Client
}

func NewAI(cfg config.AIConfig) *AI {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &AI{
		apiKey:  cfg.APIKey,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		model:   cfg.Model,
		timeout: timeout,
		client:  &http.Client{Timeout: timeout},
	}
}

// chatRequest / chatResponse 只覆盖非流式对话需要的字段。
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Chat 发送多轮消息，返回助手的回复文本。
func (a *AI) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	if a == nil || a.apiKey == "" {
		return "", ErrAINotConfigured
	}
	if len(messages) == 0 {
		return "", fmt.Errorf("消息不能为空")
	}

	payload, err := json.Marshal(chatRequest{
		Model:    a.model,
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrAIUpstream, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("%w: 读取响应失败: %v", ErrAIUpstream, err)
	}

	var result chatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("%w: 解析响应失败 (HTTP %d)", ErrAIUpstream, resp.StatusCode)
	}
	if result.Error != nil && result.Error.Message != "" {
		return "", fmt.Errorf("%w: %s", ErrAIUpstream, result.Error.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%w: HTTP %d", ErrAIUpstream, resp.StatusCode)
	}
	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("%w: 返回内容为空", ErrAIUpstream)
	}

	return result.Choices[0].Message.Content, nil
}

// ExtractedItem 是 AI 从问询中抽取出的候选资讯条目，尚未入库。
type ExtractedItem struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Source      string `json:"source,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
}

const extractSystemPrompt = `你是资讯整理助手。用户会给出资讯类型（news=新闻，video=视频）和整理要求。
请从要求中抽取候选条目，只输出 JSON，不输出任何解释文字、不要包裹代码块。
输出格式：JSON 数组，每个元素包含 title（标题，必填）、url（必须以 http:// 或 https:// 开头的完整链接）、source（来源，可为空）、published_at（YYYY-MM-DD，不确定则为空字符串）。
最多 10 条。优先使用用户明确给出的标题和链接；用户未给出具体链接时，可基于公开常识整理候选条目，URL 必须是完整且看起来可信的链接。结果只是候选，最终是否入库由用户勾选决定。`

// Extract 按类型从用户问询中抽取结构化资讯条目。
// 抽取结果只是候选数据，是否入库由用户勾选后走 /api/info/items 决定。
func (a *AI) Extract(ctx context.Context, kind, question string) ([]ExtractedItem, error) {
	if a == nil || a.apiKey == "" {
		return nil, ErrAINotConfigured
	}
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, fmt.Errorf("问询内容不能为空")
	}

	label := "新闻"
	if kind == "video" {
		label = "视频"
	}
	content, err := a.Chat(ctx, []ChatMessage{
		{Role: "system", Content: extractSystemPrompt},
		{Role: "user", Content: "类型：" + label + "\n要求：" + question},
	})
	if err != nil {
		return nil, err
	}
	return parseExtractedItems(content)
}

// parseExtractedItems 兼容纯数组、```json 代码块以及 { "items": [...] } 三种返回。
func parseExtractedItems(content string) ([]ExtractedItem, error) {
	s := strings.TrimSpace(content)
	if strings.HasPrefix(s, "```") {
		if nl := strings.IndexByte(s, '\n'); nl >= 0 {
			s = strings.TrimSpace(s[nl+1:])
		}
		s = strings.TrimSpace(strings.TrimSuffix(s, "```"))
	}

	var items []ExtractedItem
	if err := json.Unmarshal([]byte(s), &items); err == nil {
		return items, nil
	}

	var wrapped struct {
		Items []ExtractedItem `json:"items"`
	}
	if err := json.Unmarshal([]byte(s), &wrapped); err == nil {
		return wrapped.Items, nil
	}

	// 兜底：模型在 JSON 前后带了说明文字，截取最外层数组
	if start := strings.IndexByte(s, '['); start >= 0 {
		if end := strings.LastIndexByte(s, ']'); end > start {
			if err := json.Unmarshal([]byte(s[start:end+1]), &items); err == nil {
				return items, nil
			}
		}
	}
	return nil, fmt.Errorf("%w: 提取结果不是合法的 JSON 数组", ErrAIUpstream)
}
