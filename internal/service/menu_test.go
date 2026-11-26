package service

import (
	"context"
	"errors"
	"testing"

	"goProject/internal/repository"
)

// fakeMenuStore 用内存数据模拟 menus 表。
type fakeMenuStore struct {
	list        []repository.Menu
	byID        map[int64]repository.Menu
	hasChildren bool
	created     *repository.Menu
	updated     *repository.Menu
	deletedID   int64
	allowed     map[string][]int64 // 角色 -> 可访问菜单 ID，用于过滤测试
}

func (f *fakeMenuStore) List(context.Context) ([]repository.Menu, error) {
	return f.list, nil
}

func (f *fakeMenuStore) ByID(_ context.Context, id int64) (*repository.Menu, error) {
	m, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &m, nil
}

func (f *fakeMenuStore) Create(_ context.Context, m *repository.Menu) (int64, error) {
	f.created = m
	return 99, nil
}

func (f *fakeMenuStore) Update(_ context.Context, m *repository.Menu) error {
	f.updated = m
	return nil
}

func (f *fakeMenuStore) Delete(_ context.Context, id int64) error {
	f.deletedID = id
	return nil
}

func (f *fakeMenuStore) HasChildren(context.Context, int64) (bool, error) {
	return f.hasChildren, nil
}

func newFakeStore() *fakeMenuStore {
	list := []repository.Menu{
		{ID: 1, ParentID: 0, Title: "仪表盘", Path: "/dashboard", Component: "dashboard/index", Sort: 10, Status: 1},
		{ID: 2, ParentID: 0, Title: "系统管理", Path: "/system", Sort: 90, Status: 1},
		{ID: 3, ParentID: 2, Title: "菜单管理", Path: "/system/menu", Component: "system/menu/index", Sort: 10, Status: 1},
	}

	byID := make(map[int64]repository.Menu, len(list))
	for _, m := range list {
		byID[m.ID] = m
	}

	return &fakeMenuStore{list: list, byID: byID, allowed: map[string][]int64{}}
}

func TestMenuTreeNestsChildren(t *testing.T) {
	store := newFakeStore()
	tree, err := NewMenu(store, store).Tree(context.Background())
	if err != nil {
		t.Fatalf("Tree() error = %v", err)
	}

	if len(tree) != 2 {
		t.Fatalf("roots = %d, want 2", len(tree))
	}
	if tree[1].Title != "系统管理" {
		t.Errorf("second root = %q, want 系统管理", tree[1].Title)
	}
	if len(tree[1].Children) != 1 || tree[1].Children[0].Title != "菜单管理" {
		t.Errorf("children = %+v, want [菜单管理]", tree[1].Children)
	}
	if len(tree[0].Children) != 0 {
		t.Errorf("leaf should have no children, got %d", len(tree[0].Children))
	}
}

func TestMenuTreeOrphanBecomesRoot(t *testing.T) {
	store := newFakeStore()
	// 父级不存在时不能丢节点
	store.list = append(store.list, repository.Menu{ID: 4, ParentID: 99, Title: "孤儿", Path: "/orphan", Status: 1})

	tree, err := NewMenu(store, store).Tree(context.Background())
	if err != nil {
		t.Fatalf("Tree() error = %v", err)
	}
	if len(tree) != 3 {
		t.Fatalf("roots = %d, want 3", len(tree))
	}
}

func TestCreateMenuRequiresSuperAdmin(t *testing.T) {
	store := newFakeStore()
	_, err := NewMenu(store, store).Create(context.Background(), "user", MenuInput{Title: "x", Path: "/x"})

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
	if store.created != nil {
		t.Error("should not touch store when role is not super admin")
	}
}

func TestCreateMenuValidatesPath(t *testing.T) {
	store := newFakeStore()
	_, err := NewMenu(store, store).Create(context.Background(), RoleSuperAdmin, MenuInput{Title: "坏路径", Path: "system/x"})

	if !errors.Is(err, ErrMenuInvalid) {
		t.Fatalf("err = %v, want ErrMenuInvalid", err)
	}
}

func TestCreateMenuRejectsMissingParent(t *testing.T) {
	store := newFakeStore()
	_, err := NewMenu(store, store).Create(context.Background(), RoleSuperAdmin,
		MenuInput{ParentID: 88, Title: "子菜单", Path: "/system/child"})

	if !errors.Is(err, ErrMenuInvalid) {
		t.Fatalf("err = %v, want ErrMenuInvalid", err)
	}
}

func TestCreateMenuNormalizesDefaults(t *testing.T) {
	store := newFakeStore()
	if _, err := NewMenu(store, store).Create(context.Background(), RoleSuperAdmin,
		MenuInput{ParentID: 2, Title: "  用户管理  ", Path: "/system/user", Status: 1}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if store.created == nil {
		t.Fatal("menu not created")
	}
	if store.created.Title != "用户管理" {
		t.Errorf("title = %q, want trimmed", store.created.Title)
	}
	if store.created.Sort != 100 {
		t.Errorf("sort = %d, want default 100", store.created.Sort)
	}
}

func TestCreateMenuKeepsDisabledStatus(t *testing.T) {
	store := newFakeStore()
	// Status 的零值就是 0（禁用），service 不能擅自改成 1，
	// 默认值由 handler 根据字段是否出现来决定
	if _, err := NewMenu(store, store).Create(context.Background(), RoleSuperAdmin,
		MenuInput{Title: "隐藏页", Path: "/hidden", Status: 0}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if store.created.Status != 0 {
		t.Errorf("status = %d, want 0 (disabled)", store.created.Status)
	}
}

func TestDeleteMenuWithChildren(t *testing.T) {
	store := newFakeStore()
	store.byID = map[int64]repository.Menu{2: {ID: 2, Title: "系统管理"}}
	store.hasChildren = true

	err := NewMenu(store, store).Delete(context.Background(), RoleSuperAdmin, 2)
	if !errors.Is(err, ErrMenuHasChildren) {
		t.Fatalf("err = %v, want ErrMenuHasChildren", err)
	}
	if store.deletedID != 0 {
		t.Error("should not delete a menu that has children")
	}
}

func TestDeleteMenuNotFound(t *testing.T) {
	store := newFakeStore()
	store.byID = map[int64]repository.Menu{}

	err := NewMenu(store, store).Delete(context.Background(), RoleSuperAdmin, 404)
	if !errors.Is(err, ErrMenuNotFound) {
		t.Fatalf("err = %v, want ErrMenuNotFound", err)
	}
}

func TestUpdateMenuRejectsCycle(t *testing.T) {
	store := newFakeStore()
	// 2 是 3 的父级，把 2 挂到 3 下面就成环了
	store.byID = map[int64]repository.Menu{
		2: {ID: 2, ParentID: 0, Title: "系统管理", Path: "/system"},
		3: {ID: 3, ParentID: 2, Title: "菜单管理", Path: "/system/menu"},
	}

	err := NewMenu(store, store).Update(context.Background(), RoleSuperAdmin, 2,
		MenuInput{ParentID: 3, Title: "系统管理", Path: "/system"})

	if !errors.Is(err, ErrMenuInvalid) {
		t.Fatalf("err = %v, want ErrMenuInvalid", err)
	}
	if store.updated != nil {
		t.Error("should not update when cycle detected")
	}
}

// MenuIDsByRoleCode 让 fakeMenuStore 同时满足 menuPermissionStore。
func (f *fakeMenuStore) MenuIDsByRoleCode(ctx context.Context, code string) ([]int64, error) {
	return f.allowed[code], nil
}

// 非超管按 role_menus 过滤，且自动补齐上级目录
func TestTreeForRoleFiltersWithAncestors(t *testing.T) {
	store := newFakeStore()
	store.allowed["admin"] = []int64{3} // 只授权「菜单管理」

	tree, err := NewMenu(store, store).TreeForRole(context.Background(), "admin")
	if err != nil {
		t.Fatalf("TreeForRole() error = %v", err)
	}

	if len(tree) != 1 || tree[0].ID != 2 {
		t.Fatalf("roots = %d, want 只有上级目录 系统管理", len(tree))
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].ID != 3 {
		t.Errorf("children = %v, want 菜单管理", tree[0].Children)
	}
}

func TestTreeForRoleEmptyWhenNoPermission(t *testing.T) {
	store := newFakeStore()

	tree, err := NewMenu(store, store).TreeForRole(context.Background(), "guest")
	if err != nil {
		t.Fatalf("TreeForRole() error = %v", err)
	}
	if len(tree) != 0 {
		t.Errorf("roots = %d, want 0", len(tree))
	}
}

// 超级管理员默认拥有全部权限，不查关联表
func TestTreeForRoleSuperAdminGetsAll(t *testing.T) {
	store := newFakeStore()

	tree, err := NewMenu(store, store).TreeForRole(context.Background(), RoleSuperAdmin)
	if err != nil {
		t.Fatalf("TreeForRole() error = %v", err)
	}
	if len(tree) != 2 {
		t.Fatalf("roots = %d, want 2", len(tree))
	}
	if len(tree[1].Children) != 1 {
		t.Errorf("系统管理 children = %d, want 1", len(tree[1].Children))
	}
}
