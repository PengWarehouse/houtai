package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"goProject/internal/repository"
	"goProject/pkg/password"
)

// 认证相关的业务错误，handler 据此映射 HTTP 状态码。
var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserDisabled       = errors.New("账号已被禁用，请联系管理员")
	ErrOldPasswordWrong   = errors.New("原密码不正确")
	ErrPasswordUnchanged  = errors.New("新密码不能与原密码相同")
	ErrInvalidInput       = errors.New("参数不合法")
)

// userStore 是 Auth 需要的数据访问能力，由 repository.UserRepository 实现。
type userStore interface {
	ByUsername(ctx context.Context, username string) (*repository.User, error)
	ByID(ctx context.Context, id int64) (*repository.User, error)
	UpdateLastLogin(ctx context.Context, id int64, at time.Time) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
}

// tokenIssuer 负责签发 token，由 auth.Manager 实现。
type tokenIssuer interface {
	Issue(userID int64, username, role string) (string, error)
	TTL() time.Duration
}

// Auth 处理登录、用户信息、修改密码。
type Auth struct {
	users  userStore
	tokens tokenIssuer
}

// NewAuth 创建 Auth。
func NewAuth(users userStore, tokens tokenIssuer) *Auth {
	return &Auth{users: users, tokens: tokens}
}

// Profile 是对外暴露的用户信息，不含密码哈希。
type Profile struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Nickname    string     `json:"nickname"`
	Email       string     `json:"email,omitempty"`
	Role        string     `json:"role"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// LoginResult 是登录成功后的返回体。
type LoginResult struct {
	Token     string  `json:"token"`
	ExpiresIn int     `json:"expires_in"`
	User      Profile `json:"user"`
}

// Login 校验账号密码并签发 token。
func (a *Auth) Login(ctx context.Context, username, plainPassword string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || plainPassword == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := a.users.ByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// 与密码错误返回同一个错误，避免泄露账号是否存在
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if user.Status != 1 {
		return nil, ErrUserDisabled
	}

	if !password.Verify(user.PasswordHash, plainPassword) {
		return nil, ErrInvalidCredentials
	}

	token, err := a.tokens.Issue(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, err
	}

	// 登录时间更新失败不影响登录本身
	if err := a.users.UpdateLastLogin(ctx, user.ID, time.Now()); err != nil {
		slog.Warn("更新最后登录时间失败", "user_id", user.ID, "err", err)
	}

	return &LoginResult{
		Token:     token,
		ExpiresIn: int(a.tokens.TTL().Seconds()),
		User:      toProfile(user),
	}, nil
}

// Profile 查询指定用户的信息。
func (a *Auth) Profile(ctx context.Context, userID int64) (*Profile, error) {
	user, err := a.users.ByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	profile := toProfile(user)
	return &profile, nil
}

// ChangePassword 校验原密码后更新为新密码。
// 注意：token 是无状态的，改密后已签发的 token 在过期前仍然有效。
func (a *Auth) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	if oldPassword == "" || newPassword == "" {
		return fmt.Errorf("%w: 原密码与新密码都不能为空", ErrInvalidInput)
	}
	if len(newPassword) < 6 {
		return fmt.Errorf("%w: 新密码至少 6 位", ErrInvalidInput)
	}
	if len(newPassword) > 72 {
		// bcrypt 只处理前 72 字节，超长部分会被忽略
		return fmt.Errorf("%w: 新密码不能超过 72 字节", ErrInvalidInput)
	}
	if oldPassword == newPassword {
		return ErrPasswordUnchanged
	}

	user, err := a.users.ByID(ctx, userID)
	if err != nil {
		return err
	}

	if !password.Verify(user.PasswordHash, oldPassword) {
		return ErrOldPasswordWrong
	}

	hash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}

	return a.users.UpdatePassword(ctx, userID, hash)
}

// toProfile 去掉敏感字段后转成对外结构。
func toProfile(user *repository.User) Profile {
	profile := Profile{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
		Role:     user.Role,
	}
	if user.Email.Valid {
		profile.Email = user.Email.String
	}
	if user.LastLoginAt.Valid {
		last := user.LastLoginAt.Time
		profile.LastLoginAt = &last
	}
	return profile
}
