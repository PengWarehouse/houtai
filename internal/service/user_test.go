package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"goProject/internal/repository"
)

// fakeUserAdminStore 用内存数据模拟 users 表。
type fakeUserAdminStore struct {
	users       map[int64]repository.User
	created     *repository.User
	updated     *repository.User
	deleted     int64
	lastFilter  repository.UserFilter
	lastPwdHash string
	lastPwdID   int64
	duplicate   bool
}

func newFakeUserStore() *fakeUserAdminStore {
	return &fakeUserAdminStore{
		users: map[int64]repository.User{
			1: {ID: 1, Username: "admin", Nickname: "超级管理员", Role: RoleSuperAdmin, Status: 1},
			2: {ID: 2, Username: "zhangsan", Nickname: "张三", Role: "user", Status: 1},
		},
	}
}

func (f *fakeUserAdminStore) ByID(_ context.Context, id int64) (*repository.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &u, nil
}

func (f *fakeUserAdminStore) List(_ context.Context, filter repository.UserFilter) ([]repository.UserDetail, int64, error) {
	f.lastFilter = filter
	list := make([]repository.UserDetail, 0, len(f.users))
	for _, u := range f.users {
		list = append(list, repository.UserDetail{
			ID: u.ID, Username: u.Username, Nickname: u.Nickname, Role: u.Role, Status: u.Status,
		})
	}
	return list, int64(len(list)), nil
}

func (f *fakeUserAdminStore) Create(_ context.Context, u *repository.User) (int64, error) {
	if f.duplicate {
		return 0, repository.ErrDuplicate
	}
	f.created = u
	return 100, nil
}

func (f *fakeUserAdminStore) UpdateProfile(_ context.Context, u *repository.User) error {
	f.updated = u
	return nil
}

func (f *fakeUserAdminStore) Delete(_ context.Context, id int64) error {
	f.deleted = id
	return nil
}

func (f *fakeUserAdminStore) UpdatePassword(_ context.Context, id int64, hash string) error {
	f.lastPwdID, f.lastPwdHash = id, hash
	return nil
}

// fakeRoleLookup 只关心角色是否存在。
type fakeRoleLookup struct{ codes map[string]bool }

func (f fakeRoleLookup) ByCode(_ context.Context, code string) (*repository.Role, error) {
	if f.codes[code] {
		return &repository.Role{Code: code}, nil
	}
	return nil, repository.ErrNotFound
}

func newFakeUserService(store *fakeUserAdminStore) *User {
	return NewUser(store, fakeRoleLookup{codes: map[string]bool{
		RoleSuperAdmin: true, "admin": true, "user": true,
	}})
}

func TestUserWriteRequiresSuperAdmin(t *testing.T) {
	svc := newFakeUserService(newFakeUserStore())

	if _, err := svc.Create(context.Background(), "admin", UserCreateInput{
		Username: "newbie", Password: "123456", Nickname: "新用户", Role: "user",
	}); !errors.Is(err, ErrForbidden) {
		t.Errorf("Create() err = %v, want ErrForbidden", err)
	}
	if err := svc.Delete(context.Background(), 2, "admin", 1); !errors.Is(err, ErrForbidden) {
		t.Errorf("Delete() err = %v, want ErrForbidden", err)
	}
}

func TestUserCreateValidation(t *testing.T) {
	svc := newFakeUserService(newFakeUserStore())
	ctx := context.Background()

	cases := []struct {
		name string
		in   UserCreateInput
	}{
		{"用户名非法", UserCreateInput{Username: "1abc", Password: "123456", Nickname: "x", Role: "user"}},
		{"密码太短", UserCreateInput{Username: "abc", Password: "12345", Nickname: "x", Role: "user"}},
		{"昵称为空", UserCreateInput{Username: "abc", Password: "123456", Nickname: "  ", Role: "user"}},
		{"角色不存在", UserCreateInput{Username: "abc", Password: "123456", Nickname: "x", Role: "ghost"}},
		{"邮箱格式", UserCreateInput{Username: "abc", Password: "123456", Nickname: "x", Role: "user", Email: "bad-email"}},
	}

	for _, c := range cases {
		if _, err := svc.Create(ctx, RoleSuperAdmin, c.in); !errors.Is(err, ErrUserInvalid) {
			t.Errorf("%s: err = %v, want ErrUserInvalid", c.name, err)
		}
	}
}

func TestUserCreateOkAndDuplicate(t *testing.T) {
	store := newFakeUserStore()
	svc := newFakeUserService(store)

	id, err := svc.Create(context.Background(), RoleSuperAdmin, UserCreateInput{
		Username: " newbie ", Password: "123456", Nickname: "新用户", Role: "user", Status: 1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if id != 100 || store.created == nil {
		t.Fatalf("id = %d, created = %v", id, store.created)
	}
	if store.created.Username != "newbie" {
		t.Errorf("username = %q, want trimmed", store.created.Username)
	}
	if store.created.PasswordHash == "" || store.created.PasswordHash == "123456" {
		t.Errorf("password 未哈希: %q", store.created.PasswordHash)
	}
	if store.created.Status != 1 {
		t.Errorf("status = %d, want 1", store.created.Status)
	}

	store.duplicate = true
	if _, err := svc.Create(context.Background(), RoleSuperAdmin, UserCreateInput{
		Username: "newbie", Password: "123456", Nickname: "重复", Role: "user", Status: 1,
	}); !errors.Is(err, ErrUserExists) {
		t.Errorf("err = %v, want ErrUserExists", err)
	}
}

func TestUserUpdateGuards(t *testing.T) {
	store := newFakeUserStore()
	svc := newFakeUserService(store)
	ctx := context.Background()

	// 不能改自己的角色
	if err := svc.Update(ctx, 1, RoleSuperAdmin, 1, UserUpdateInput{
		Nickname: "超管", Role: "admin", Status: 1,
	}); !errors.Is(err, ErrUserInvalid) {
		t.Errorf("改自己角色 err = %v, want ErrUserInvalid", err)
	}

	// 不能停用自己
	if err := svc.Update(ctx, 1, RoleSuperAdmin, 1, UserUpdateInput{
		Nickname: "超管", Role: RoleSuperAdmin, Status: 0,
	}); !errors.Is(err, ErrUserInvalid) {
		t.Errorf("停用自己 err = %v, want ErrUserInvalid", err)
	}

	// 不能改内置超管账号的角色
	if err := svc.Update(ctx, 2, RoleSuperAdmin, 1, UserUpdateInput{
		Nickname: "超管", Role: "admin", Status: 1,
	}); !errors.Is(err, ErrUserProtected) {
		t.Errorf("改超管角色 err = %v, want ErrUserProtected", err)
	}

	// 普通用户可以被正常修改
	if err := svc.Update(ctx, 1, RoleSuperAdmin, 2, UserUpdateInput{
		Nickname: "张三改", Email: "zs@test.com", Role: "admin", Status: 0,
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if store.updated == nil || store.updated.Nickname != "张三改" || store.updated.Role != "admin" {
		t.Errorf("updated = %+v", store.updated)
	}
	if !store.updated.Email.Valid || store.updated.Email.String != "zs@test.com" {
		t.Errorf("email = %+v", store.updated.Email)
	}
	if err := svc.Update(ctx, 1, RoleSuperAdmin, 999, UserUpdateInput{
		Nickname: "x", Role: "user", Status: 1,
	}); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("不存在的用户 err = %v, want ErrUserNotFound", err)
	}
}

func TestUserDeleteGuards(t *testing.T) {
	store := newFakeUserStore()
	svc := newFakeUserService(store)
	ctx := context.Background()

	if err := svc.Delete(ctx, 1, RoleSuperAdmin, 1); !errors.Is(err, ErrUserInvalid) {
		t.Errorf("删自己 err = %v, want ErrUserInvalid", err)
	}
	if err := svc.Delete(ctx, 2, RoleSuperAdmin, 1); !errors.Is(err, ErrUserProtected) {
		t.Errorf("删超管 err = %v, want ErrUserProtected", err)
	}
	if err := svc.Delete(ctx, 1, RoleSuperAdmin, 2); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if store.deleted != 2 {
		t.Errorf("deleted = %d, want 2", store.deleted)
	}
}

func TestUserResetPassword(t *testing.T) {
	store := newFakeUserStore()
	svc := newFakeUserService(store)
	ctx := context.Background()

	if err := svc.ResetPassword(ctx, RoleSuperAdmin, 2, "123"); !errors.Is(err, ErrUserInvalid) {
		t.Errorf("短密码 err = %v, want ErrUserInvalid", err)
	}
	if err := svc.ResetPassword(ctx, RoleSuperAdmin, 999, "123456"); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("不存在 err = %v, want ErrUserNotFound", err)
	}
	if err := svc.ResetPassword(ctx, RoleSuperAdmin, 2, "newpass123"); err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if store.lastPwdID != 2 || store.lastPwdHash == "" || store.lastPwdHash == "newpass123" {
		t.Errorf("密码未正确哈希: id=%d hash=%q", store.lastPwdID, store.lastPwdHash)
	}
}

func TestUserListClampsPaging(t *testing.T) {
	store := newFakeUserStore()
	svc := newFakeUserService(store)

	result, err := svc.List(context.Background(), UserQuery{Page: 0, Size: 999})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Page != 1 {
		t.Errorf("page = %d, want 1", result.Page)
	}
	if result.Size != 50 {
		t.Errorf("size = %d, want 50", result.Size)
	}
	if store.lastFilter.Page != 1 || store.lastFilter.Size != 50 {
		t.Errorf("filter = %+v", store.lastFilter)
	}

	// 上限 500 应被保留（前端可选 50/100/200/500）
	store2 := newFakeUserStore()
	svc2 := newFakeUserService(store2)
	result2, err := svc2.List(context.Background(), UserQuery{Page: 1, Size: 500})
	if err != nil {
		t.Fatalf("List(size=500) error = %v", err)
	}
	if result2.Size != 500 || store2.lastFilter.Size != 500 {
		t.Errorf("size = %d, filter size = %d, want 500", result2.Size, store2.lastFilter.Size)
	}
}

// 保证 UserDetail 到 UserItem 的时间转换正确（last_login_at 可空）。
func TestToUserItemLastLogin(t *testing.T) {
	now := time.Now()

	item := toUserItem(repository.UserDetail{ID: 1, LastLoginAt: sql.NullTime{Time: now, Valid: true}})
	if item.LastLoginAt == nil || !item.LastLoginAt.Equal(now) {
		t.Errorf("LastLoginAt = %v", item.LastLoginAt)
	}

	item = toUserItem(repository.UserDetail{ID: 2})
	if item.LastLoginAt != nil {
		t.Errorf("LastLoginAt = %v, want nil", item.LastLoginAt)
	}
}
