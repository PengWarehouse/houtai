package config

import (
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 保存服务运行所需的全部配置。
// 默认值即可直接启动，通过环境变量或 configs/.env 覆盖。
type Config struct {
	AppName          string        // 应用名，用于日志标识
	Env              string        // 运行环境：dev / test / prod
	Host             string        // 监听地址
	Port             int           // 监听端口
	LogLevel         slog.Level    // 日志级别：debug / info / warn / error
	ShutdownTimeout  time.Duration // 优雅退出等待时长
	DB               DBConfig      // 数据库配置
	Auth             AuthConfig    // 认证配置
	AI               AIConfig      // AI（DeepSeek）配置
	CORSAllowOrigins []string      // 允许跨域的前端来源，空表示不启用
}

// AIConfig 是 DeepSeek 兼容接口的配置。
type AIConfig struct {
	APIKey  string        // DeepSeek API Key，为空则 AI 接口返回未配置
	BaseURL string        // OpenAI 兼容根地址，默认 https://api.deepseek.com
	Model   string        // 模型名，默认 deepseek-chat
	Timeout time.Duration // 单次请求超时
}

// AuthConfig 是 JWT 相关配置。
type AuthConfig struct {
	JWTSecret string        // 签名密钥，留空则启动时生成临时密钥
	JWTExpire time.Duration // token 有效期
}

// DBConfig 是 MySQL 连接与连接池配置。
type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnectTimeout  time.Duration
}

// Address 返回 host:port，用于日志与错误信息。
func (c DBConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// Load 读取配置：先加载 env 文件，再读取进程环境变量。
// ENV_FILE 可指定配置文件路径，默认为 configs/.env（不存在则跳过）。
func Load() *Config {
	loadEnvFile(os.Getenv("ENV_FILE"))

	return &Config{
		AppName:         getEnv("APP_NAME", "goProject"),
		Env:             getEnv("APP_ENV", "dev"),
		Host:            getEnv("APP_HOST", "0.0.0.0"),
		Port:            getEnvInt("APP_PORT", 8080),
		LogLevel:        getEnvLevel("LOG_LEVEL", slog.LevelInfo),
		ShutdownTimeout: time.Duration(getEnvInt("SHUTDOWN_TIMEOUT_SEC", 10)) * time.Second,
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "127.0.0.1"),
			Port:            getEnvInt("DB_PORT", 3306),
			User:            getEnv("DB_USER", "root"),
			Password:        getEnv("DB_PASSWORD", ""),
			Name:            getEnv("DB_NAME", "mydb"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_SEC", 1800)) * time.Second,
			ConnectTimeout:  time.Duration(getEnvInt("DB_CONNECT_TIMEOUT_SEC", 5)) * time.Second,
		},
		Auth: AuthConfig{
			JWTSecret: getEnv("JWT_SECRET", ""),
			JWTExpire: time.Duration(getEnvInt("JWT_EXPIRE_HOURS", 12)) * time.Hour,
		},
		AI: AIConfig{
			APIKey:  getEnv("DEEPSEEK_API_KEY", ""),
			BaseURL: getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
			Model:   getEnv("DEEPSEEK_MODEL", "deepseek-chat"),
			Timeout: time.Duration(getEnvInt("DEEPSEEK_TIMEOUT_SEC", 60)) * time.Second,
		},
		// 默认放行本机 Vite 开发服务器；生产环境改成真实前端域名，或用 * 放开
		CORSAllowOrigins: getEnvList("CORS_ALLOW_ORIGINS", []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}),
	}
}

// Addr 返回 host:port 形式的监听地址。
func (c *Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// loadEnvFile 解析简单的 KEY=VALUE 文件，不覆盖已存在的环境变量。
// 支持空行、# 注释、行尾空白与成对引号。
func loadEnvFile(path string) {
	if path == "" {
		path = "configs/.env"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return // 文件不存在属于正常情况，使用默认值
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return n
}

// getEnvList 解析逗号分隔的列表，如 "http://a.com,http://b.com"。
func getEnvList(key string, fallback []string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}

	parts := strings.Split(v, ",")
	list := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			list = append(list, p)
		}
	}
	if len(list) == 0 {
		return fallback
	}
	return list
}

func getEnvLevel(key string, fallback slog.Level) slog.Level {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}

	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return fallback
	}
}
