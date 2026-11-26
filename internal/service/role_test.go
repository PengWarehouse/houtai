package service

import (
	"context"
	"errors"
	"testing"

	"goProject/internal/repository"
)

// fakeRoleStore 用内存数据模拟 roles 表。
type fakeRoleStore struct {
	roles   []repository.Role
	created *repository.Role
	updated *repository.Role
	deleted int64
	menus   map[int64][]int64
	saved   map[int64][]int64
	dupCode bool
}

func (f *fakeRoleStore) List(context.Context) ([]repository.Role, error) {
	return f.roles, nil
}

func (f *fakeRoleStore) ByID(_ context.Context, id int64) (*repository.Role, error) {
	for _, r := range f.roles {
		if r.ID == id {
			item := r
			return &item, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeRoleStore) ByCode(_ context.Context, code string) (*repository.Role, error) {
	for _, r := range f.roles {
		if r.Code == code {
			item := r
			return &item, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeRoleStore) Create(_ context.Context, role *repository.Role) (int64, error) {
	if f.dupCode {
		return 0, repository.ErrDuplicate
	}
	f.created = role
	return 99, nil
}

func (f *fakeRoleStore) Update(_ context.Context, role *repository.Role) error {
	f.updated = role
	return nil
}

func (f *fakeRoleStore) Delete(_ context.Context, id int64) error {
	f.deleted = id
	return nil
}

func (f *fakeRoleStore) MenuIDs(_ context.Context, roleID int64) ([]int64, error) {
	return f.menus[roleID], nil
}

func (f *fakeRoleStore) ReplaceMenus(_ context.Context, roleID int64, ids []int64) error {
	if f.saved == nil {
		f.saved = map[int64][]int64{}
	}
	f.saved[roleID] = ids
	return nil
}

func newFakeRoleStore() *fakeRoleStore {
	return &fakeRoleStore{
		roles: []repository.Role{
			{ID: 1, Code: RoleSuperAdmin, Name: "超级管理员", Sort: 10, Status: 1, UserCount: 1},
			{ID: 2, Code: "admin", Name: "管理员", Sort: 20, Status: 1, UserCount: 2},
			{ID: 3, Code: "guest", Name: "访客", Sort: 30, Status: 1},
		},
		menus: map[int64][]int64{2: {1, 2, 4}},
	}
}

func TestRoleWriteRequiresSuperAdmin(t *testing.T) {
	store := newFakeRoleStore()
	svc := NewRole(store)

	if _, err := svc.Create(context.Background(), "admin", RoleInput{Code: "editor", Name: "编辑"}); !errors.Is(err, ErrForbidden) {
		t.Errorf("Create() err = %v, want ErrForbidden", err)
	}
	if err := svc.Delete(context.Background(), "admin", 3); !errors.Is(err, ErrForbidden) {
		t.Errorf("Delete() err = %v, want ErrForbidden", err)
	}
	if err := svc.SetMenus(context.Background(), "admin", 3, []int64{1}); !errors.Is(err, ErrForbidden) {
		t.Errorf("SetMenus() err = %v, want ErrForbidden", err)
	}
}

func TestRoleCreateValidatesCode(t *testing.T) {
	svc := NewRole(newFakeRoleStore())

	if _, err := svc.Create(context.Background(), RoleSuperAdmin, RoleInput{Code: "Bad Code", Name: "x"}); !errors.Is(err, ErrRoleInvalid) {
		t.Errorf("err = %v, want ErrRoleInvalid", err)
	}
	if _, err := svc.Create(context.Background(), RoleSuperAdmin, RoleInput{Code: "ok_code", Name: "  "}); !errors.Is(err, ErrRoleInvalid) {
		t.Errorf("err = %v, want ErrRoleInvalid for empty name", err)
	}
	if _, err := svc.Create(context.Background(), RoleSuperAdmin, RoleInput{Code: RoleSuperAdmin, Name: "重复"}); !errors.Is(err, ErrRoleExists) {
		t.Errorf("err = %v, want ErrRoleExists", err)
	}

	store := newFakeRoleStore()
	store.dupCode = true
	if _, err := NewRole(store).Create(context.Background(), RoleSuperAdmin, RoleInput{Code: "dup", Name: "重复"}); !errors.Is(err, ErrRoleExists) {
		t.Errorf("err = %v, want ErrRoleExists on duplicate", err)
	}
}

func TestRoleCreateTrimsAndDefaults(t *testing.T) {
	store := newFakeRoleStore()

	id, err := NewRole(store).Create(context.Background(), RoleSuperAdmin,
		RoleInput{Code: " editor ", Name: "  编辑  ", Description: "  负责内容  ", Status: 1})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if id != 99 || store.created == nil {
		t.Fatalf("created = %v, id = %d", store.created, id)
	}
	if store.created.Code != "editor" || store.created.Name != "编辑" || store.created.Description != "负责内容" {
		t.Errorf("not trimmed: %+v", store.created)
	}
	if store.created.Sort != 100 {
		t.Errorf("sort = %d, want default 100", store.created.Sort)
	}
	if store.created.Status != 1 {
		t.Errorf("status = %d, want 1", store.created.Status)
	}

	// status 为 0 是「显式禁用」，不能被默认值覆盖
	store2 := newFakeRoleStore()
	if _, err := NewRole(store2).Create(context.Background(), RoleSuperAdmin,
		RoleInput{Code: "disabled_role", Name: "停用角色", Status: 0}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if store2.created.Status != 0 {
		t.Errorf("status = %d, want 0（保留禁用）", store2.created.Status)
	}
}

func TestRoleUpdateCannotDisableSuperAdmin(t *testing.T) {
	store := newFakeRoleStore()
	err := NewRole(store).Update(context.Background(), RoleSuperAdmin, 1,
		RoleInput{Name: "超级管理员", Status: 0})

	if !errors.Is(err, ErrRoleProtected) {
		t.Fatalf("err = %v, want ErrRoleProtected", err)
	}
	if store.updated != nil {
		t.Error("不该写入数据库")
	}
}

func TestRoleUpdateKeepsCode(t *testing.T) {
	store := newFakeRoleStore()
	err := NewRole(store).Update(context.Background(), RoleSuperAdmin, 2,
		RoleInput{Code: "hacked", Name: "管理员改", Status: 1})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if store.updated.Code != "admin" {
		t.Errorf("code = %q, want 原值 admin（不允许修改）", store.updated.Code)
	}
}

func TestRoleDeleteGuards(t *testing.T) {
	store := newFakeRoleStore()
	svc := NewRole(store)

	if err := svc.Delete(context.Background(), RoleSuperAdmin, 1); !errors.Is(err, ErrRoleProtected) {
		t.Errorf("删除内置角色 err = %v, want ErrRoleProtected", err)
	}
	if err := svc.Delete(context.Background(), RoleSuperAdmin, 2); !errors.Is(err, ErrRoleInUse) {
		t.Errorf("删除有用户的角色 err = %v, want ErrRoleInUse", err)
	}
	if err := svc.Delete(context.Background(), RoleSuperAdmin, 3); err != nil {
		t.Fatalf("删除空角色 error = %v", err)
	}
	if store.deleted != 3 {
		t.Errorf("deleted = %d, want 3", store.deleted)
	}
	if err := svc.Delete(context.Background(), RoleSuperAdmin, 999); !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("删除不存在 err = %v, want ErrRoleNotFound", err)
	}
}

func TestRoleMenusForSuperAdminIsAll(t *testing.T) {
	svc := NewRole(newFakeRoleStore())

	perm, err := svc.MenuIDs(context.Background(), RoleSuperAdmin, 1)
	if err != nil {
		t.Fatalf("MenuIDs() error = %v", err)
	}
	if !perm.All || len(perm.MenuIDs) != 0 {
		t.Errorf("perm = %+v, want all=true", perm)
	}

	perm, err = svc.MenuIDs(context.Background(), RoleSuperAdmin, 2)
	if err != nil {
		t.Fatalf("MenuIDs() error = %v", err)
	}
	if perm.All || len(perm.MenuIDs) != 3 {
		t.Errorf("perm = %+v, want all=false with 3 ids", perm)
	}
}

func TestRoleSetMenusDedupesAndProtectsSuperAdmin(t *testing.T) {
	store := newFakeRoleStore()
	svc := NewRole(store)

	if err := svc.SetMenus(context.Background(), RoleSuperAdmin, 1, []int64{1}); !errors.Is(err, ErrRoleProtected) {
		t.Errorf("给超管分配权限 err = %v, want ErrRoleProtected", err)
	}

	if err := svc.SetMenus(context.Background(), RoleSuperAdmin, 3, []int64{2, 1, 2, 0, -1}); err != nil {
		t.Fatalf("SetMenus() error = %v", err)
	}
	if got := store.saved[3]; len(got) != 2 || got[0] != 2 || got[1] != 1 {
		t.Errorf("saved = %v, want [2 1]", got)
	}
}
