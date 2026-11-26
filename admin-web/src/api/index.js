/**
 * 接口定义集中在这一个文件，组件里不要直接写 url
 * 基础路径见 VITE_API_BASE_URL（默认 /api）
 * 最终请求：/api/login、/api/profile ...
 */
import request from '@/utils/request'

/** 登录：POST /api/login { username, password } -> { token, expires_in, user } */
export function login(data) {
  return request.post('/login', data)
}

/** 当前登录用户信息：GET /api/profile（需 token） */
export function getProfile() {
  return request.get('/profile')
}

/** 修改密码：POST /api/password { old_password, new_password }（需 token） */
export function changePassword(data) {
  return request.post('/password', data)
}

/** 服务运行信息：GET /api/info */
export function getServerInfo() {
  return request.get('/info')
}

/** 菜单树：GET /api/menus（需 token） */
export function listMenus() {
  return request.get('/menus')
}

/** 新增菜单：POST /api/menus（仅超管） */
export function createMenu(data) {
  return request.post('/menus', data)
}

/** 修改菜单：PUT /api/menus/{id}（仅超管） */
export function updateMenu(id, data) {
  return request.put(`/menus/${id}`, data)
}

/** 删除菜单：DELETE /api/menus/{id}（仅超管，有子级会被拒绝） */
export function deleteMenu(id) {
  return request.delete(`/menus/${id}`)
}

/** 用户列表：GET /api/users?page=&size=&keyword=&role=&status= */
export function listUsers(params) {
  return request.get('/users', { params })
}

/** 新增用户：POST /api/users（仅超管） */
export function createUser(data) {
  return request.post('/users', data)
}

/** 修改用户：PUT /api/users/{id}（仅超管） */
export function updateUser(id, data) {
  return request.put(`/users/${id}`, data)
}

/** 删除用户：DELETE /api/users/{id}（仅超管） */
export function deleteUser(id) {
  return request.delete(`/users/${id}`)
}

/** 重置用户密码：POST /api/users/{id}/password（仅超管） */
export function resetUserPassword(id, password) {
  return request.post(`/users/${id}/password`, { password })
}

/** 角色列表：GET /api/roles（登录即可） */
export function listRoles() {
  return request.get('/roles')
}

/** 新增角色：POST /api/roles（仅超管） */
export function createRole(data) {
  return request.post('/roles', data)
}

/** 修改角色：PUT /api/roles/{id}（仅超管） */
export function updateRole(id, data) {
  return request.put(`/roles/${id}`, data)
}

/** 删除角色：DELETE /api/roles/{id}（仅超管） */
export function deleteRole(id) {
  return request.delete(`/roles/${id}`)
}

/** 角色已分配的菜单：GET /api/roles/{id}/menus -> { all, menu_ids } */
export function getRoleMenus(id) {
  return request.get(`/roles/${id}/menus`)
}

/** 保存角色菜单权限：PUT /api/roles/{id}/menus { menu_ids } */
export function setRoleMenus(id, menuIds) {
  return request.put(`/roles/${id}/menus`, { menu_ids: menuIds })
}

/** AI 问询：POST /api/ai/chat { messages: [{ role, content }] } -> { content } */
export function chatAI(messages) {
  return request.post('/ai/chat', { messages })
}

/** AI 资讯提取：POST /api/ai/extract { kind, question } -> { items: [{ title, url, source, published_at }] } */
export function extractAI(kind, question) {
  return request.post('/ai/extract', { kind, question })
}

/** 资讯列表：GET /api/info/items?kind=&keyword=（需 token） */
export function listInfoItems(params) {
  return request.get('/info/items', { params })
}

/** 勾选入库：POST /api/info/items { kind, items } -> { inserted, skipped, failures } */
export function saveInfoItems(data) {
  return request.post('/info/items', data)
}
