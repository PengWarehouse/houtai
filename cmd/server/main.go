package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goProject/internal/auth"
	"goProject/internal/config"
	"goProject/internal/handler"
	"goProject/internal/repository"
	"goProject/internal/service"
)

func main() {
	cfg := config.Load()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := repository.Open(ctx, cfg)
	if err != nil {
		slog.Error("database init failed", "err", err)
		os.Exit(1)
	}
	slog.Info("database connected", "addr", cfg.DB.Address(), "db", cfg.DB.Name)

	// JWT 密钥：未配置时生成临时密钥，重启后旧 token 失效
	secret := cfg.Auth.JWTSecret
	if secret == "" {
		generated, err := auth.RandomSecret()
		if err != nil {
			slog.Error("generate jwt secret failed", "err", err)
			os.Exit(1)
		}
		secret = generated
		slog.Warn("JWT_SECRET 未配置，已生成临时密钥，重启后所有 token 失效")
	}

	tokens, err := auth.NewManager(secret, cfg.Auth.JWTExpire)
	if err != nil {
		slog.Error("init jwt manager failed", "err", err)
		os.Exit(1)
	}

	users := repository.NewUserRepository(store)
	authSvc := service.NewAuth(users, tokens)

	roles := repository.NewRoleRepository(store)
	roleSvc := service.NewRole(roles)
	userSvc := service.NewUser(users, roles)

	menus := repository.NewMenuRepository(store)
	menuSvc := service.NewMenu(menus, roles)

	aiSvc := service.NewAI(cfg.AI)
	if cfg.AI.APIKey == "" {
		slog.Warn("DEEPSEEK_API_KEY 未配置，AI 问询接口将返回未配置提示")
	}

	infoRepo := repository.NewInfoRepository(store)
	infoSvc := service.NewInfoCatalog(infoRepo)

	svc := service.New(cfg)
	srv := &http.Server{
		Addr: cfg.Addr(),
		Handler: handler.NewRouter(handler.Deps{
			Service:        svc,
			DB:             store,
			Auth:           authSvc,
			Tokens:         tokens,
			Menu:           menuSvc,
			User:           userSvc,
			Role:           roleSvc,
			AI:             aiSvc,
			Info:           infoSvc,
			AllowedOrigins: cfg.CORSAllowOrigins,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("server starting", "app", cfg.AppName, "env", cfg.Env, "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server exited unexpectedly", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}

	if err := store.Close(); err != nil {
		slog.Error("close database failed", "err", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
