package handler

import (
	"context"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"goProject/internal/auth"
	"goProject/internal/service"
	"goProject/pkg/httpx"
)

// Database 声明 handler 需要的数据库能力。
// 用接口而非具体类型，方便替换实现、也便于测试注入。
type Database interface {
	Ping(ctx context.Context) error
	Version(ctx context.Context) (string, error)
}

// AuthService 声明 handler 需要的认证能力，由 service.Auth 实现。
type AuthService interface {
	Login(ctx context.Context, username, plainPassword string) (*service.LoginResult, error)
	Profile(ctx context.Context, userID int64) (*service.Profile, error)
	ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error
}

// TokenParser 校验 token 并返回 claims，由 auth.Manager 实现。
type TokenParser interface {
	Parse(token string) (*auth.Claims, error)
}

// Deps 汇总 handler 的依赖，由 main 装配后传入。
type Deps struct {
	Service *service.Service
	DB      Database
	Auth    AuthService
	Tokens  TokenParser
	Menu    MenuService
	User    UserService
	Role    RoleService
	AI      AIService
	Info    InfoService
	// AllowedOrigins 是允许跨域访问的前端来源，为空表示不启用 CORS。
	AllowedOrigins []string
}

// Server 组合业务依赖，负责把 HTTP 请求映射到 Service。
type Server struct {
	svc    *service.Service
	db     Database
	auth   AuthService
	tokens TokenParser
	menu   MenuService
	user   UserService
	role    RoleService
	ai      AIService
	infoSvc InfoService
}

// dbStatus 是对外暴露的数据库状态。
type dbStatus struct {
	Driver  string `json:"driver"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Status  string `json:"status"`
}

// infoResponse 在服务信息中附带数据库状态。
type infoResponse struct {
	service.Info
	Database dbStatus `json:"database"`
}

// NewRouter 注册路由并返回 http.Handler。依赖允许为 nil（调试/测试时按需裁剪）。
func NewRouter(deps Deps) http.Handler {
	s := &Server{
		svc:    deps.Service,
		db:     deps.DB,
		auth:   deps.Auth,
		tokens: deps.Tokens,
		menu:   deps.Menu,
		user:   deps.User,
		role:    deps.Role,
		ai:      deps.AI,
		infoSvc: deps.Info,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/hello", s.hello)
	mux.HandleFunc("GET /api/info", s.info)

	// 认证相关
	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("GET /api/profile", s.requireAuth(s.profile))
	mux.HandleFunc("POST /api/password", s.requireAuth(s.changePassword))

	// 菜单：查询按当前用户角色过滤，增删改仅超管（在 service 层校验角色）
	mux.HandleFunc("GET /api/menus", s.requireAuth(s.listMenus))
	mux.HandleFunc("POST /api/menus", s.requireAuth(s.createMenu))
	mux.HandleFunc("PUT /api/menus/{id}", s.requireAuth(s.updateMenu))
	mux.HandleFunc("DELETE /api/menus/{id}", s.requireAuth(s.deleteMenu))

	// 用户管理：查询对所有登录用户开放，增删改与重置密码仅超管
	mux.HandleFunc("GET /api/users", s.requireAuth(s.listUsers))
	mux.HandleFunc("POST /api/users", s.requireAuth(s.createUser))
	mux.HandleFunc("PUT /api/users/{id}", s.requireAuth(s.updateUser))
	mux.HandleFunc("DELETE /api/users/{id}", s.requireAuth(s.deleteUser))
	mux.HandleFunc("POST /api/users/{id}/password", s.requireAuth(s.resetUserPassword))

	// 角色管理：查询对所有登录用户开放，增删改与权限分配仅超管
	mux.HandleFunc("GET /api/roles", s.requireAuth(s.listRoles))
	mux.HandleFunc("POST /api/roles", s.requireAuth(s.createRole))
	mux.HandleFunc("PUT /api/roles/{id}", s.requireAuth(s.updateRole))
	mux.HandleFunc("DELETE /api/roles/{id}", s.requireAuth(s.deleteRole))
	mux.HandleFunc("GET /api/roles/{id}/menus", s.requireAuth(s.roleMenus))
	mux.HandleFunc("PUT /api/roles/{id}/menus", s.requireAuth(s.setRoleMenus))

	// AI 问询：需登录，密钥只在服务端使用
	mux.HandleFunc("POST /api/ai/chat", s.requireAuth(s.aiChat))
	mux.HandleFunc("POST /api/ai/extract", s.requireAuth(s.aiExtract))

	// 资讯条目：查询与勾选入库均需登录
	mux.HandleFunc("GET /api/info/items", s.requireAuth(s.listInfoItems))
	mux.HandleFunc("POST /api/info/items", s.requireAuth(s.saveInfoItems))

	// 顺序：日志 → CORS → 路由，预检请求也能被记录
	return logRequests(cors(deps.AllowedOrigins)(mux))
}

// health 用于存活探针，数据库不可用时返回 503。
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	status := "up"
	if s.db == nil {
		status = "disabled"
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := s.db.Ping(ctx); err != nil {
			slog.Warn("database health check failed", "err", err)
			httpx.JSON(w, http.StatusServiceUnavailable, httpx.Envelope{
				Code:    httpx.CodeFailure,
				Message: "database unavailable",
				Data:    map[string]string{"db": "down", "error": err.Error()},
			})
			return
		}
	}

	httpx.JSON(w, http.StatusOK, httpx.Envelope{
		Code:    httpx.CodeSuccess,
		Message: "healthy",
		Data:    map[string]string{"db": status},
	})
}

// hello 返回示例业务数据，支持 ?name= 参数。
func (s *Server) hello(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, s.svc.Greeting(r.URL.Query().Get("name")))
}

// info 返回服务与数据库运行信息。
func (s *Server) info(w http.ResponseWriter, r *http.Request) {
	cfg := s.svc.Config()

	db := dbStatus{Driver: "mysql", Name: cfg.DB.Name, Status: "disabled"}
	if s.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		db.Status = "up"
		version, err := s.db.Version(ctx)
		if err != nil {
			db.Status = "down"
			slog.Warn("query database version failed", "err", err)
		} else {
			db.Version = version
		}
	}

	httpx.OK(w, infoResponse{
		Info: service.Info{
			App:       cfg.AppName,
			Env:       cfg.Env,
			GoVersion: runtime.Version(),
			Uptime:    s.svc.Uptime().Round(1e6).String(),
		},
		Database: db,
	})
}

// logRequests 是简单的访问日志中间件。
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
