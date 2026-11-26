# goProject

一个基于 Go 标准库 + MySQL 的 HTTP 服务骨架，采用 `cmd / internal / pkg` 分层目录，克隆后填好数据库密码即可运行。

## 技术栈

| 用途 | 选型 |
| --- | --- |
| 语言 | Go 1.27 |
| HTTP 服务 | `net/http`（使用 Go 1.22+ 的 `GET /path` 路由语法） |
| 日志 | `log/slog` 结构化日志 |
| 数据库 | MySQL 8.4 + `github.com/go-sql-driver/mysql` |
| 密码哈希 | `golang.org/x/crypto/bcrypt` |
| 认证 | JWT（HS256）+ `github.com/golang-jwt/jwt/v5` |
| 配置 | 环境变量 + 可选 `configs/.env` |
| 测试 | 标准库 `testing` + `net/http/httptest`（不依赖真实数据库） |

## 目录结构

```
goProject/
├── cmd/
│   ├── server/
│   │   └── main.go            # 程序入口：装配依赖、连库、启动 HTTP 服务、优雅退出
│   └── migrate/
│       └── main.go            # 执行 db/ 下的 SQL 脚本，建表与初始化数据
├── internal/                  # 私有代码，外部模块无法 import
│   ├── auth/
│   │   └── token.go           # JWT 签发与校验（HS256）
│   ├── config/
│   │   └── config.go          # 配置加载（env 文件 + 环境变量 + 默认值）
│   ├── handler/
│   │   ├── handler.go         # 路由注册、访问日志中间件、DB 健康检查
│   │   ├── auth.go            # 登录、用户信息、改密、鉴权中间件
│   │   ├── menu.go            # 菜单增删改查
│   │   ├── user.go            # 用户管理（列表/新增/编辑/删除/重置密码）
│   │   ├── role.go            # 角色管理与菜单权限分配
│   │   ├── cors.go            # CORS 中间件
│   │   └── handler_test.go    # httptest + fake 依赖的接口测试
│   ├── repository/
│   │   ├── store.go           # MySQL 连接池、DSN 组装、Ping / Version
│   │   ├── user.go            # users 表数据访问
│   │   ├── menu.go            # menus 表数据访问
│   │   └── role.go            # roles / role_menus 数据访问
│   └── service/
│       ├── service.go         # 示例业务逻辑，不感知 HTTP
│       ├── auth.go            # 登录/改密业务规则与错误定义
│       ├── menu.go            # 菜单树组装、按角色过滤、成环校验
│       ├── user.go            # 用户校验与自我保护规则
│       └── role.go            # 角色校验、内置角色保护、权限分配
├── pkg/                       # 可被其他项目复用的通用包
│   ├── httpx/
│   │   └── response.go        # 统一 JSON 响应封装
│   └── password/
│       └── password.go        # bcrypt 哈希与校验
├── db/
│   ├── schema.sql             # users 建表语句（幂等）
│   ├── seed.sql               # 初始化数据：超级管理员账号
│   ├── menu.sql               # menus 建表 + 初始菜单
│   └── role.sql               # roles / role_menus 建表 + 内置角色
├── configs/
│   ├── .env.example           # 配置样例，复制为 .env 生效
│   └── .env                   # 本地配置（含密码，已被 .gitignore 忽略）
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
└── README.md
```

各目录职责：

- **`cmd/`**：每个可执行程序一个子目录，只放 `main` 包，负责依赖装配。
- **`internal/`**：业务代码。`config` 管配置，`repository` 管数据访问，`service` 管业务规则，`handler` 管协议层。
- **`pkg/`**：与业务无关、可对外复用的工具包。
- **`configs/`**：配置文件，`configs/.env` 已被 `.gitignore` 忽略。

依赖方向为单向：`cmd → handler → service`，`cmd → repository`，两者都依赖 `config`；`service` 通过接口依赖 `repository` 的数据访问与 `auth.Manager` 的签发能力。`handler` 只声明自己需要的接口（`Database` / `AuthService` / `TokenParser`），不直接引用 `repository`，所以测试里塞假实现就能跑。

## 快速开始

### 1. 准备数据库

本机 MySQL 8.4（服务名 `MySQL84`，端口 3306）。库不存在时先建：

```sql
CREATE DATABASE IF NOT EXISTS mydb
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_0900_ai_ci;
```

### 2. 填写连接信息

编辑 `configs/.env`，**把 `DB_PASSWORD` 换成你的 MySQL root 密码**：

```ini
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=你的密码
DB_NAME=mydb
```

### 3. 建表并初始化数据

```bash
go run ./cmd/migrate      # 等价于 make migrate
```

依次执行 `db/schema.sql`（建 `users` 表）、`db/seed.sql`（插入超级管理员）、`db/menu.sql`（建 `menus` 表并预置菜单）和 `db/role.sql`（建 `roles` / `role_menus` 并预置角色）。四个脚本都写成幂等，可以反复执行：

```
INFO database connected addr=127.0.0.1:3306 db=mydb
INFO applied file=db/schema.sql cost=2ms
INFO applied file=db/seed.sql cost=1ms
INFO applied file=db/menu.sql cost=4ms
INFO applied file=db/role.sql cost=42ms
INFO migrate done files=4
```

也可以只跑指定文件：`go run ./cmd/migrate -file db/schema.sql`（`-file` 可重复）。

### 4. 启动

Go 1.27 安装在本机 `C:\Program Files\Go\bin`，未加入 `PATH` 时用完整路径：

```powershell
& 'C:\Program Files\Go\bin\go.exe' run ./cmd/server
```

加入 `PATH` 后即可 `go run ./cmd/server`。启动日志：

```
level=INFO msg="database connected" addr=127.0.0.1:3306 db=mydb
level=INFO msg="server starting" app=goProject env=dev addr=0.0.0.0:1104
```

启动时会先 Ping 一次数据库，**连接失败直接退出并打印原因**（如 `Access denied` 就是密码不对）。按 `Ctrl+C` 触发优雅退出，先停 HTTP 服务再关闭连接池。

## 接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/healthz` | 存活探针，顺带 Ping 数据库；库不可用返回 **503** |
| GET | `/api/hello?name=Gopher` | 示例业务接口，`name` 可选，默认 `world` |
| GET | `/api/info` | 服务信息 + 数据库状态、MySQL 版本 |
| POST | `/api/login` | 登录，返回 JWT token |
| GET | `/api/profile` | 当前登录用户信息（需 token） |
| POST | `/api/password` | 修改密码（需 token） |
| GET | `/api/menus` | 菜单树（需 token），**按当前用户角色过滤**，超管返回全部 |
| POST | `/api/menus` | 新增菜单（需 token，仅 `super_admin`） |
| PUT | `/api/menus/{id}` | 修改菜单（需 token，仅 `super_admin`） |
| DELETE | `/api/menus/{id}` | 删除菜单（需 token，仅 `super_admin`；有子菜单时拒绝） |
| GET | `/api/users` | 用户分页列表，支持 `page` / `size` / `keyword` / `role` / `status` |
| POST | `/api/users` | 新增用户（仅 `super_admin`） |
| PUT | `/api/users/{id}` | 修改用户（仅 `super_admin`；不能改自己的角色或状态） |
| DELETE | `/api/users/{id}` | 删除用户（仅 `super_admin`；不能删自己和超管） |
| POST | `/api/users/{id}/password` | 重置指定用户密码（仅 `super_admin`） |
| GET | `/api/roles` | 角色列表，附用户数与已分配菜单数 |
| POST | `/api/roles` | 新增角色（仅 `super_admin`） |
| PUT | `/api/roles/{id}` | 修改角色（仅 `super_admin`；`code` 不可改） |
| DELETE | `/api/roles/{id}` | 删除角色（仅 `super_admin`；内置角色或在用角色会被拒绝） |
| GET | `/api/roles/{id}/menus` | 角色已分配菜单，超管返回 `{all:true}` |
| PUT | `/api/roles/{id}/menus` | 覆盖角色菜单权限（仅 `super_admin`；超管不可分配） |

请求示例（端口以 `APP_PORT` 为准，当前为 `1104`）：

```bash
curl "http://127.0.0.1:1104/healthz"
curl "http://127.0.0.1:1104/api/info"
```

`/healthz` 正常时：

```json
{"code": 200, "message": "healthy", "data": {"db": "up"}}
```

`/api/info` 会真实执行 `SELECT VERSION()`，能返回版本号就说明整条链路（配置 → 连接池 → SQL）是通的：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "app": "goProject",
    "env": "dev",
    "go_version": "go1.27.4",
    "uptime": "3.21s",
    "database": {
      "driver": "mysql",
      "name": "mydb",
      "version": "8.4.9",
      "status": "up"
    }
  }
}
```

### 响应约定

| 场景 | HTTP 状态 | `code` | `message` |
| --- | --- | --- | --- |
| 成功 | 200（新增为 201） | `200` | `ok` 或业务提示，如「保存成功」 |
| 业务失败 | 400 / 401 / 403 / 404 | `0` | 失败原因，如「用户名或密码错误」 |
| 服务端异常 | 500 | `0` | 固定为「服务器内部错误」，不暴露内部细节 |

前端以 `code === 200` 判定成功、直接取 `data`；其余一律视为失败，把 `message` 弹给用户。HTTP 状态码保持语义正确，方便网关、代理和浏览器按标准处理。

## 认证（登录 / 修改密码）

登录成功后签发 **JWT（HS256）**，后续请求放在 `Authorization: Bearer <token>` 头里。token 是无状态的，服务重启后只要 `JWT_SECRET` 不变就仍然有效。

```bash
# 1. 登录
curl -X POST http://127.0.0.1:1104/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"Admin@123456"}'
```

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 43200,
    "user": {
      "id": 1,
      "username": "admin",
      "nickname": "超级管理员",
      "email": "admin@example.com",
      "role": "super_admin",
      "last_login_at": "2026-11-25T13:56:47+08:00"
    }
  }
}
```

```bash
# 2. 带 token 查当前用户
curl http://127.0.0.1:1104/api/profile -H "Authorization: Bearer <token>"

# 3. 修改密码（原密码必须正确，新密码 6-72 位）
curl -X POST http://127.0.0.1:1104/api/password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"old_password":"Admin@123456","new_password":"NewPass@1234"}'
```

token 的 payload 里带 `uid` / `username` / `role`，有效期由 `JWT_EXPIRE_HOURS` 控制。

**错误状态码**

| 场景 | HTTP | message |
| --- | --- | --- |
| 用户名或密码错误（含账号不存在） | 401 | 用户名或密码错误 |
| 账号被禁用（`status = 0`） | 403 | 账号已被禁用，请联系管理员 |
| 未带 token / token 无效过期 | 401 | 请先登录 / 登录已过期，请重新登录 |
| 原密码不正确 | 400 | 原密码不正确 |
| 新密码不合法（过短/超 72 字节） | 400 | 参数不合法: ... |
| 新旧密码相同 | 400 | 新密码不能与原密码相同 |

几个实现上的取舍：

- **账号不存在与密码错误返回同一句话**，避免被拿去枚举用户名。
- 登录成功会更新 `users.last_login_at`，更新失败只记 warn，不影响登录。
- 改密后**已签发的 token 在过期前仍然可用**（JWT 无状态）。要做到改密即踢下线，需要引入 token 黑名单或版本号（`users.token_version`）。
- bcrypt 只处理前 72 字节，所以新密码长度上限定在 72 字节。

## 数据表

### users（用户表）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 主键，自增 |
| `username` | `VARCHAR(64)` | 登录名，唯一索引 `uk_users_username` |
| `password_hash` | `VARCHAR(255)` | bcrypt 哈希（`$2a$10$...`），**不存明文** |
| `nickname` | `VARCHAR(64)` | 昵称 |
| `email` | `VARCHAR(128)` | 邮箱，可空 |
| `role` | `VARCHAR(32)` | 角色：`super_admin` / `admin` / `user`，默认 `user` |
| `status` | `TINYINT` | `1` 正常、`0` 禁用，默认 `1` |
| `last_login_at` | `DATETIME` | 最后登录时间，可空 |
| `created_at` / `updated_at` | `DATETIME` | 创建/更新时间，自动维护 |

预置的超级管理员（由 `db/seed.sql` 插入）：

| 用户名 | 初始密码 | 昵称 | 角色 |
| --- | --- | --- | --- |
| `admin` | `Admin@123456` | 超级管理员 | `super_admin` |

> 初始密码仅供本地开发，登录功能做完后请立即修改。
> 哈希由 `pkg/password.Hash("Admin@123456")` 生成，校验用 `pkg/password.Verify(hash, plain)`，
> 两者都封装了 bcrypt，业务代码里不要再自己调 bcrypt。

需要手工查库时（不把密码写进命令行历史）：

```bash
set -a; . <(tr -d '\r' < configs/.env); set +a
MYSQL_PWD="$DB_PASSWORD" mysql -u "$DB_USER" mydb -e "SELECT id, username, role, status FROM users;"
```

### menus（菜单表）

前端侧边栏和路由都由这张表驱动，`db/menu.sql` 建表并预置初始菜单。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 主键，自增 |
| `parent_id` | `BIGINT UNSIGNED` | 父级菜单 ID，`0` 表示顶级 |
| `title` | `VARCHAR(64)` | 菜单名称，显示在侧边栏与页面标题 |
| `path` | `VARCHAR(128)` | 路由路径，必须以 `/` 开头 |
| `icon` | `VARCHAR(64)` | Element Plus 图标组件名，如 `Menu` |
| `component` | `VARCHAR(128)` | `views` 下的相对路径，如 `system/menu/index`；**留空表示目录**，只分组不产生页面 |
| `sort` | `INT` | 排序值，越小越靠前，默认 `100` |
| `status` | `TINYINT` | `1` 启用、`0` 禁用；禁用后不出现在侧边栏，也不会注册路由 |
| `created_at` / `updated_at` | `DATETIME` | 自动维护 |

预置菜单：`仪表盘`（顶级）、`系统管理`（顶级目录）→ `菜单管理`、`用户管理`、`角色管理`（子级）。

写入规则（`internal/service/menu.go`）：

- 只有 `super_admin` 能增删改，普通角色返回 403；
- `path` 必须以 `/` 开头，`component` 不允许以 `/` 开头或包含 `..`；
- 上级菜单不存在、或把菜单挂到自己/自己的子孙下（成环）都会被拒绝；
- 有子菜单的菜单不能直接删除，避免留下孤儿节点。

### roles（角色表）

`db/role.sql` 建表并预置三个内置角色，`users.role` 存的就是这里的 `code`。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 主键 |
| `code` | `VARCHAR(32)` | 角色标识，唯一索引 `uk_roles_code`，**创建后不可修改** |
| `name` | `VARCHAR(64)` | 角色名称 |
| `description` | `VARCHAR(255)` | 角色说明 |
| `sort` / `status` | `INT` / `TINYINT` | 排序、启用状态 |
| `created_at` / `updated_at` | `DATETIME` | 自动维护 |

### role_menus（角色菜单关联）

一行表示某角色可以访问某个菜单，主键 `(role_id, menu_id)`。

| 角色 | 权限来源 |
| --- | --- |
| `super_admin` | **不写关联记录**，服务层直接放行全部菜单（默认拥有所有权限） |
| 其他角色 | 按 `role_menus` 过滤，`GET /api/menus` 只返回已授权菜单；父目录会自动补齐 |

权限修改后，对应用户**重新登录**（或刷新页面重新拉菜单）才会生效。

## 配置

配置优先级：**已存在的环境变量 > `configs/.env` > 代码内置默认值**。

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `APP_NAME` | `goProject` | 应用名，用于日志与响应 |
| `APP_ENV` | `dev` | 运行环境：`dev` / `test` / `prod` |
| `APP_HOST` | `0.0.0.0` | 监听地址 |
| `APP_PORT` | `8080` | 监听端口（`configs/.env` 中设为 `1104`） |
| `LOG_LEVEL` | `info` | 日志级别：`debug` / `info` / `warn` / `error` |
| `SHUTDOWN_TIMEOUT_SEC` | `10` | 优雅退出最长等待秒数 |
| `ENV_FILE` | `configs/.env` | 配置文件路径，可指向其他位置 |
| `DB_HOST` | `127.0.0.1` | MySQL 主机 |
| `DB_PORT` | `3306` | MySQL 端口 |
| `DB_USER` | `root` | 数据库账号 |
| `DB_PASSWORD` | 空 | 数据库密码，**必填** |
| `DB_NAME` | `mydb` | 数据库名 |
| `DB_MAX_OPEN_CONNS` | `25` | 最大打开连接数 |
| `DB_MAX_IDLE_CONNS` | `5` | 最大空闲连接数 |
| `DB_CONN_MAX_LIFETIME_SEC` | `1800` | 连接最长存活秒数 |
| `DB_CONNECT_TIMEOUT_SEC` | `5` | 建立连接/启动 Ping 的超时秒数 |
| `JWT_SECRET` | 空 | JWT 签名密钥，至少 16 位；留空则启动时生成临时密钥（重启后 token 失效） |
| `JWT_EXPIRE_HOURS` | `12` | token 有效期（小时） |
| `CORS_ALLOW_ORIGINS` | `http://localhost:5173,http://127.0.0.1:5173` | 允许跨域的前端来源，逗号分隔；`*` 表示全部放开（仅限本地调试） |

> `configs/.env` 使用**相对于当前工作目录**的路径，请在项目根目录启动程序。
> 该文件中的值不会覆盖 shell 里已导出的同名环境变量，便于容器或 CI 覆盖。
> DSN 由 `mysql.Config` 生成，密码含 `@` `:` `/` 等特殊字符也无需手动转义。

## 跨域（CORS）

后端内置 CORS 中间件（`internal/handler/cors.go`），行为如下：

| 场景 | 结果 |
| --- | --- |
| 请求不带 `Origin`（同源、curl、服务端调用） | 原样放行，不写任何 CORS 头 |
| `Origin` 命中 `CORS_ALLOW_ORIGINS` | 回显该 Origin，并返回 `Allow-Methods` / `Allow-Headers` / `Max-Age=600` |
| 预检请求（`OPTIONS` + `Access-Control-Request-Method`）命中 | 返回 `204`，不进业务路由 |
| `Origin` 不在白名单，且是预检 | 返回 `403`，日志打 `cors origin rejected` |
| `Origin` 不在白名单，且是普通请求 | 交给业务处理，但不写 CORS 头（浏览器自行拦截） |

**开发环境不需要 CORS**：前端 `.env.development` 里 `VITE_API_BASE_URL=/api` 是相对路径，
请求发往 Vite dev server 的 `/api`，再由 `vite.config.js` 的 proxy 转发到后端，
浏览器视角是同源。只有把 baseURL 改成 `http://127.0.0.1:1104/api` 这种绝对地址时，
才需要后端 CORS 白名单里有对应来源。

**生产环境**推荐同样用相对路径 + nginx 反代（同源，无跨域）；前后端分域部署时：

```ini
CORS_ALLOW_ORIGINS=https://admin.example.com
```

注意中间件回显的是具体 Origin 而非 `*`，所以将来要带 cookie 也能用；当前用 Bearer token，
不依赖 `Access-Control-Allow-Credentials`。

## 常用命令

| 命令 | 说明 |
| --- | --- |
| `make run` | 启动服务 |
| `make build` | 编译到 `bin/goProject` |
| `make test` | 运行全部测试（不需要数据库） |
| `make vet` | 静态检查 |
| `make fmt` | 格式化代码 |
| `make migrate` | 建表 + 初始化超级管理员 |
| `make clean` | 清理编译产物 |

未安装 `make` 时，等价的原生命令：

```bash
go run ./cmd/server           # 启动
go build -o bin/app ./cmd/server
go test ./...
go vet ./...
gofmt -w .
```

## 扩展建议

- **加表与查询**：在 `internal/repository/` 下按实体建文件（参考 `user.go`），SQL 全部写在这一层，`service` 只依赖接口方法，不碰 `*sql.DB`。
- **新增表**：把 DDL 追加到 `db/schema.sql`（保持 `IF NOT EXISTS` 幂等），或按版本拆成 `db/00x_xxx.sql` 用 `-file` 指定；迁移变复杂后再引入 golang-migrate。
- **角色鉴权**：token 的 payload 里已经带 `role`，在 `requireAuth` 之内再套一层 `requireRole("super_admin")` 就能限制接口。
- **强制下线**：JWT 无状态，改密后旧 token 仍在有效期内可用；需要立即失效就给 `users` 加 `token_version`，签发时写入、校验时比对。
- **加中间件**：在 `handler.NewRouter` 外层继续包裹 `func(http.Handler) http.Handler`。
- **换框架**：`handler` 层是唯一与 `net/http` 耦合的位置，换成 Gin/Echo 不影响 `service` 与 `repository`。
