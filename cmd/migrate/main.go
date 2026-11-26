// Command migrate 按顺序执行 SQL 文件，用于本地建表与初始化数据。
// 脚本需要写成幂等（IF NOT EXISTS / INSERT IGNORE），才能重复执行。
//
//	go run ./cmd/migrate                                  # 默认执行 db/schema.sql + db/seed.sql
//	go run ./cmd/migrate -file db/schema.sql              # 只执行指定文件（可重复 -file）
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"goProject/internal/config"
)

// stringSlice 支持重复指定 -file。
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }

func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	var files stringSlice
	flag.Var(&files, "file", "要执行的 SQL 文件，可重复指定")
	flag.Parse()

	cfg := config.Load()

	targets := []string(files)
	useDefaults := len(targets) == 0
	if useDefaults {
		targets = []string{"db/schema.sql", "db/seed.sql", "db/menu.sql", "db/role.sql"}
	}

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
	// 允许一次执行多条语句，SQL 文件才能整段提交给服务端
	driverCfg.MultiStatements = true

	db, err := sql.Open("mysql", driverCfg.FormatDSN())
	if err != nil {
		slog.Error("初始化数据库驱动失败", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pingCtx, pingCancel := context.WithTimeout(ctx, cfg.DB.ConnectTimeout)
	defer pingCancel()

	if err := db.PingContext(pingCtx); err != nil {
		slog.Error("连接数据库失败", "addr", cfg.DB.Address(), "db", cfg.DB.Name, "err", err)
		os.Exit(1)
	}
	slog.Info("database connected", "addr", cfg.DB.Address(), "db", cfg.DB.Name)

	applied := 0
	for _, path := range targets {
		content, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) && useDefaults {
			slog.Warn("跳过不存在的默认脚本", "file", path)
			continue
		}
		if err != nil {
			slog.Error("读取 SQL 文件失败", "file", path, "err", err)
			os.Exit(1)
		}

		start := time.Now()
		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			slog.Error("执行 SQL 失败", "file", path, "err", err)
			os.Exit(1)
		}
		applied++
		slog.Info("applied", "file", path, "cost", time.Since(start).Round(time.Millisecond).String())
	}

	slog.Info("migrate done", "files", applied)
}
