package service

import (
	"fmt"
	"time"

	"goProject/internal/config"
)

// Service 承载业务逻辑，不感知 HTTP 细节，便于单独测试与复用。
type Service struct {
	cfg       *config.Config
	startedAt time.Time
}

// Message 是示例业务对象。
type Message struct {
	Text      string    `json:"text"`
	Env       string    `json:"env"`
	CreatedAt time.Time `json:"created_at"`
}

// Info 描述服务运行状态。
type Info struct {
	App       string `json:"app"`
	Env       string `json:"env"`
	GoVersion string `json:"go_version"`
	Uptime    string `json:"uptime"`
}

// New 创建 Service。
func New(cfg *config.Config) *Service {
	return &Service{
		cfg:       cfg,
		startedAt: time.Now(),
	}
}

// Greeting 返回一条问候消息，name 为空时使用默认称呼。
func (s *Service) Greeting(name string) Message {
	if name == "" {
		name = "world"
	}

	return Message{
		Text:      fmt.Sprintf("Hello, %s! This is %s.", name, s.cfg.AppName),
		Env:       s.cfg.Env,
		CreatedAt: time.Now(),
	}
}

// Uptime 返回服务已运行时长。
func (s *Service) Uptime() time.Duration {
	return time.Since(s.startedAt)
}

// Config 暴露只读配置，供上层拼装响应。
func (s *Service) Config() *config.Config {
	return s.cfg
}
