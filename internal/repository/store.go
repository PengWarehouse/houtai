// Package repository 管理数据库连接，并提供数据访问入口。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"

	"goProject/internal/config"
)

// Store 封装连接池，是数据访问的唯一入口。
type Store struct {
	db *sql.DB
}

// Open 创建连接池并做一次 Ping 校验，校验失败返回错误。
func Open(ctx context.Context, cfg *config.Config) (*Store, error) {
	driverCfg := mysql.NewConfig()
	driverCfg.User = cfg.DB.User
	driverCfg.Passwd = cfg.DB.Password
	driverCfg.Net = "tcp"
	driverCfg.Addr = net.JoinHostPort(cfg.DB.Host, strconv.Itoa(cfg.DB.Port))
	driverCfg.DBName = cfg.DB.Name
	driverCfg.ParseTime = true
	driverCfg.Loc = time.Local
	driverCfg.Timeout = cfg.DB.ConnectTimeout
	driverCfg.Params = map[string]string{"charset": "utf8mb4"}

	db, err := sql.Open("mysql", driverCfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("初始化数据库驱动失败: %w", err)
	}

	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)
	db.SetConnMaxIdleTime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DB.ConnectTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("连接 MySQL 失败 (%s/%s): %w", cfg.DB.Address(), cfg.DB.Name, err)
	}

	return &Store{db: db}, nil
}

// DB 返回底层连接池，供具体的 Repository 实现使用。
func (s *Store) DB() *sql.DB {
	return s.db
}

// Ping 检查数据库连通性。
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Version 查询服务端版本，用于验证真实可执行 SQL。
func (s *Store) Version(ctx context.Context) (string, error) {
	var version string
	if err := s.db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		return "", err
	}
	return version, nil
}

// Close 关闭连接池。
func (s *Store) Close() error {
	return s.db.Close()
}
