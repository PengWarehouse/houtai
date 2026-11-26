# admin-web

Vue 3 + Vite 的后台管理前端脚手架。**纯 JavaScript，不使用 TypeScript**。

- UI：Element Plus（全量引入 + 图标全局注册）
- 状态：Pinia（组合式 store）
- 路由：Vue Router 4，带登录守卫与菜单配置
- 请求：axios 二次封装，统一解包 `{ code, message, data }`
- 环境：`.env` / `.env.development` / `.env.production` 三份配置

## 目录结构

```
admin-web/
├── .env                    # 公共变量
├── .env.development        # 开发环境（npm run dev）
├── .env.production         # 生产环境（npm run build）
├── index.html
├── vite.config.js          # 别名 @、dev 代理、构建配置
├── jsconfig.json           # 编辑器识别 @ 别名
└── src/
    ├── main.js             # 挂载 Vue、Element Plus、Pinia、Router
    ├── App.vue             # 只有 <router-view />
    ├── api/
    │   └── index.js        # 所有接口定义，组件不直接写 url
    ├── utils/
    │   ├── request.js      # axios 实例 + 拦截器
    │   └── auth.js         # token 的 localStorage 存取
    ├── stores/
    │   ├── index.js        # pinia 实例
    │   └── user.js         # 用户状态：token / userInfo / 登录登出
    ├── router/
    │   └── index.js        # 路由表 + 菜单配置 + 全局守卫
    ├── layout/
    │   └── index.vue       # 后台布局：侧边栏 + 顶栏 + 内容区
    ├── components/
    │   ├── PasswordDialog.vue  # 修改密码弹窗
    │   ├── IconPicker.vue      # 图标选择器
    │   └── SidebarItem.vue     # 递归渲染菜单项
    ├── icons/
    │   ├── index.js        # 自定义图标登记
    │   └── RoleIcon.vue    # 角色图标（盾牌+人物）
    ├── views/
    │   ├── login/index.vue
    │   ├── dashboard/index.vue
    │   ├── system/
    │   │   ├── menu/index.vue  # 菜单管理
    │   │   ├── user/index.vue  # 用户管理
    │   │   └── role/index.vue  # 角色管理
    │   └── error/404.vue
    └── styles/
        └── index.scss      # 全局样式与重置
```

## 快速开始

```bash
cd admin-web
npm install
npm run dev        # http://localhost:5173
```

开发服务器已配好代理：页面上请求 `/api/**` 会被转发到 `VITE_PROXY_TARGET`（默认 `http://127.0.0.1:1104`，即本仓库的 Go 后端），所以浏览器里没有跨域问题。**要先启动 Go 后端**，仪表盘才能取到数据：

```bash
cd ..                       # 回到 goProject
go run ./cmd/server
```

构建与本地预览：

```bash
npm run build      # 产物在 dist/
npm run preview    # 本地起静态服务预览 dist
```

## 环境变量

Vite 只注入 `VITE_` 前缀的变量，通过 `import.meta.env.XXX` 读取。

| 变量 | 值 | 说明 |
| --- | --- | --- |
| `VITE_APP_TITLE` | `后台管理系统` | 站点标题，`index.html` 里用 `%VITE_APP_TITLE%` 注入 |
| `VITE_API_BASE_URL` | `/api` | axios 的 baseURL；**用相对路径走代理，改成绝对地址会触发跨域** |
| `VITE_API_TIMEOUT` | `15000` | 请求超时（毫秒） |
| `VITE_PORT` | `5173` | dev server 端口 |
| `VITE_PROXY_TARGET` | `http://127.0.0.1:1104` | dev 代理目标 |
| `VITE_REQUIRE_AUTH` | `false` / `true` | 是否强制登录，路由守卫读取 |

优先级：`.env.<mode>` 覆盖 `.env`。`npm run dev` 加载 `.env.development`，`npm run build` 加载 `.env.production`。

> **改了 env 必须重启 dev server**：Vite 在启动时一次性读取环境变量，热更新不会重新加载 `.env` 文件。

### 关于跨域

`VITE_API_BASE_URL` 保持 `/api`，请求经 vite proxy 转发到 Go 后端，浏览器视作同源，不会产生 CORS 预检。

若改成 `http://127.0.0.1:1104/api` 这种绝对地址，请求会绕过代理直连后端，属于跨域，必须同时满足：

1. 后端 `configs/.env` 的 `CORS_ALLOW_ORIGINS` 包含前端来源（默认已含 `http://localhost:5173` 与 `http://127.0.0.1:5173`）；
2. 用的是 `localhost:5173` 而不是 `127.0.0.1:5173` 时，两者在浏览器里是**不同来源**，需分别列出。

生产环境建议同样用相对路径 + nginx 反代，从根上避免跨域。

## axios 封装

`src/utils/request.js` 做了三件事：

1. **请求拦截**：自动把 `localStorage` 里的 token 塞进 `Authorization: Bearer xxx`。
2. **响应解包**：后端约定 `{ code, message, data }`，`code === 200` 时直接返回 `data`，所以业务代码拿到的就是数据本身：

   ```js
   // src/api/index.js
   export function getServerInfo() {
     return request.get('/info')   // 实际请求 GET /api/info
   }

   // 组件里
   const info = await getServerInfo()  // 已经是 data，不用再 .data.data
   ```

3. **错误统一提示**：`code !== 0` 弹 `message`；HTTP 401 清 token 并跳登录，403/404/5xx/超时/断网都翻译成中文，用 `ElMessage.error` 提示，同时 `reject` 让调用方还能 `catch`。

不符合 `{ code, message, data }` 约定的响应（比如纯文本或第三方接口），会原样返回，不会被吞掉。

新增接口只改 `src/api/index.js`，组件里不要写 url：

```js
export function createUser(data) {
  return request.post('/users', data)
}
export function deleteUser(id) {
  return request.delete(`/users/${id}`)
}
```

## 路由与菜单

路由和侧边栏都由**后端菜单表**驱动，`src/router/index.js` 里只保留登录页、布局容器和 404 三条基础路由。

流程：

1. 登录后（或刷新页面）守卫检测到「有 token 但菜单未加载」，先 `GET /api/menus` 拿菜单树；
2. `registerMenus()` 递归遍历菜单，用 `import.meta.glob('../views/**/*.vue')` 把 `component` 字段解析成组件，`router.addRoute('Root', ...)` 挂到布局下；
3. 注册完用同一地址重新匹配一次（`return { path, query, hash, replace: true }`），否则首次访问 `/` 会先落到 404；
4. 侧边栏 `SidebarItem.vue` 递归渲染，有 `children` 的用 `el-sub-menu`，叶子用 `el-menu-item`。

新增页面的步骤（不用改前端代码）：

1. 在 `src/views/` 下建文件，例如 `src/views/system/user/index.vue`；
2. 进「系统管理 → 菜单管理」新增一条菜单：路由路径 `/system/user`、组件路径 `system/user/index`；
3. 保存后侧边栏立即出现，路由也已注册，直接点就能进。

约定：

- `component` 是**相对 `views` 的路径且不带 `.vue`**，留空表示目录（只分组、不产生页面）；
- `status = 0` 的菜单不显示、也不注册路由；
- 删除菜单时会 `router.removeRoute('menu-{id}')`，避免删掉的页面还能访问；
- 基础路由里的 `meta.public = true` 表示免登录（`/login`、404）。

### 图标选择

菜单表单的「图标」用 `src/components/IconPicker.vue`：点击输入框弹出浮窗，面板按「自定义 / Element Plus」分组展示，支持按名称模糊搜索，点一下即选中并回填图标名（如 `Setting`），随表单一起提交给后端。

组件是标准 `v-model`，任何地方都能复用：

```vue
<IconPicker v-model="form.icon" />
```

图标名最终存进 `menus.icon`，渲染时靠 `main.js` 里的全局注册解析成组件，所以**自定义图标需要先全局注册**，否则只会渲染成空白。

#### 自定义图标

官方图标包没有的图标放 `src/icons/`，在 `src/icons/index.js` 里登记：

```js
import RoleIcon from './RoleIcon.vue'

export const customIcons = {
  Role: RoleIcon // 键名就是存进数据库的图标名
}
```

`main.js` 会把 `{ ...ElementPlusIconsVue, ...customIcons }` 一起全局注册，图标选择器也读同一份表，所以新增后**不用改选择器**，分组里会直接出现（同名以自定义为准）。

现有自定义图标：

| 名称 | 文件 | 样式 |
| --- | --- | --- |
| `Role` | `src/icons/RoleIcon.vue` | 盾牌 + 人物镂空，代表角色 / 权限 |

绘制约定：SFC 里直接写 `<svg viewBox="0 0 1024 1024">`，路径 `fill="currentColor"` 跟随文字颜色；`el-icon` 会给内部 `svg` 设 `width/height: 1em`，**不要写死宽高**，否则侧边栏和表格里尺寸会不一致。

## 登录说明

登录页调的接口契约是：

```
POST /api/login
{ "username": "admin", "password": "Admin@123456" }

-> { "code": 200, "message": "ok", "data": { "token": "...", "user": { "id": 1, "username": "admin", "role": "super_admin" } } }
```

**Go 后端已实现该接口**，用 `users` 表里的超管账号即可登录：`admin` / `Admin@123456`。

其他已封装的接口（`src/api/index.js`）：

| 接口 | 说明 |
| --- | --- |
| `GET /api/profile` | 当前用户信息，请求头需带 `Authorization: Bearer <token>` |
| `POST /api/password` | 修改密码，body `{ old_password, new_password }`，需 token |
| `GET /api/menus` | 菜单树，登录后守卫自动调用；**按当前角色过滤**，超管返回全部 |
| `POST /api/menus` | 新增菜单，body `{ parent_id, title, path, icon, component, sort, status }`，仅超管 |
| `PUT /api/menus/{id}` | 修改菜单，仅超管 |
| `DELETE /api/menus/{id}` | 删除菜单，仅超管；有子菜单时返回 400 |
| `GET /api/users` | 用户分页列表，参数 `page` / `size` / `keyword` / `role` / `status` |
| `POST /api/users` | 新增用户 `{ username, password, nickname, email, role, status }`，仅超管 |
| `PUT /api/users/{id}` | 修改用户 `{ nickname, email, role, status }`，仅超管 |
| `DELETE /api/users/{id}` | 删除用户，仅超管；不能删自己和内置超管 |
| `POST /api/users/{id}/password` | 重置密码 `{ password }`，仅超管 |
| `GET /api/roles` | 角色列表，含 `user_count` / `menu_count` / `builtin` |
| `POST /api/roles` | 新增角色 `{ code, name, description, sort, status }`，仅超管 |
| `PUT /api/roles/{id}` | 修改角色，仅超管；`code` 不可改 |
| `DELETE /api/roles/{id}` | 删除角色，仅超管；内置角色或在用角色返回 400/409 |
| `GET /api/roles/{id}/menus` | 角色菜单权限 `{ all, menu_ids }` |
| `PUT /api/roles/{id}/menus` | 保存权限 `{ menu_ids }`，仅超管 |

关于登录拦截：

- `.env.development` 目前是 `VITE_REQUIRE_AUTH=false`，**不登录也能浏览后台**，方便开发时反复调页面；
- 把它改成 `true`（或直接改 `.env.production`），未登录访问任意后台路由都会被重定向到 `/login?redirect=原路径`，登录成功后自动跳回；
- token 存在 `localStorage.admin_token`，`request.js` 会自动加到请求头，401 时清 token 并跳回登录页。

### 刷新页面后登录态怎么恢复

token 在 `localStorage`，用户信息（昵称、角色）只在 Pinia 内存里，所以刷新后 store 会重建：

1. `token` 从 `localStorage` 读回来 → `isLogin` 仍为 `true`；
2. `userInfo` 是空对象 → 顶栏会显示「未登录」；
3. 路由守卫检测到「有 token 但 `profileLoaded === false`」，先 `await` 一次 `GET /api/profile`，成功后再放行渲染，所以用户信息回来之前页面不会闪。

`ensureProfile()` 只在 401 时清登录态（token 真过期或被踢），网络异常仅提示，不会把人踢下线。`profileLoaded` 保证一次会话只拉一次。

## 修改密码

入口在顶栏右上角的用户下拉菜单里（「修改密码」在「退出登录」上方），组件为 `src/components/PasswordDialog.vue`，任意页面可直接复用：

```vue
<PasswordDialog v-model="visible" />
```

行为：

- 三个字段：原密码 / 新密码 / 确认密码，均使用 `show-password`；
- 校验：新密码 6-72 位（bcrypt 只处理前 72 字节），确认密码需与新密码一致；
- 提交成功后弹提示、清 token、跳 `/login`，**强制用新密码重新登录**；
- 每次打开弹窗都会清空上次输入，`close-on-click-modal` 关闭，避免误点丢失输入；
- 接口错误由 axios 拦截器统一提示（原密码错误会显示后端返回的「原密码不正确」）。


登录页的几个细节：

| 行为 | 说明 |
| --- | --- |
| 记住用户名 | 勾选后写入 `localStorage.admin_remember_username`，下次自动填充；密码从不保存 |
| 回车提交 | 在密码框按 Enter 触发登录 |
| 自动聚焦 | 进入页面光标直接落在密码框 |
| 校验 | 用户名 2-32 位、密码至少 6 位，失焦即校验 |
| 登录失败 | 清空密码框，错误提示由 axios 拦截器统一弹出 |
| 忘记密码 | 弹提示引导联系管理员（未接后端） |

## 用户与角色管理

两个页面都在 `src/views/system/` 下，菜单由后端驱动，不用改路由文件：

| 页面 | 文件 | 能力 |
| --- | --- | --- |
| 用户管理 | `views/system/user/index.vue` | 关键字/角色/状态筛选、分页、新增、编辑、重置密码、删除 |
| 角色管理 | `views/system/role/index.vue` | 角色增删改、按名称/标识搜索、分配菜单权限 |

### 权限模型

- `super_admin` 默认拥有全部菜单权限，由后端放行（不写 `role_menus`），页面上显示「全部」且**禁用分配权限、删除**按钮；
- 其他角色的可见菜单由 `role_menus` 决定，进「分配权限」勾选即可；
- 权限树用 `check-strictly` 独立勾选，勾子菜单时自动补父级（否则侧边栏会缺目录，页面也进不去）；
- 保存后对应用户**重新登录或刷新页面**才生效，因为菜单只在守卫里拉一次。

### 自我保护

后端对以下操作会直接拒绝，前端对应的按钮也会置灰或提示：

- 删除自己、修改自己的角色或状态；
- 删除内置超管账号、把超管角色停用；
- 删除仍被用户引用的角色（提示先调整用户角色）。

## 部署

`npm run build` 产出 `dist/`，是纯静态文件。注意 Vue Router 用的是 history 模式，nginx 需要回退到 `index.html`：

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # 把 /api 转发给 Go 后端
    location /api/ {
        proxy_pass http://127.0.0.1:1104;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 约定

- 不用 TypeScript，但保留 `jsconfig.json` 让编辑器识别 `@/` 别名。
- 组件一律 `<script setup>` 写法。
- 样式：全局在 `src/styles/index.scss`，组件内用 `<style scoped lang="scss">`。
- 状态只在 Pinia 里放跨组件共享的部分，页面内状态用 `ref` / `reactive`。
