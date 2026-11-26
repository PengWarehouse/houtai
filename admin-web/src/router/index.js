import { createRouter, createWebHistory } from 'vue-router'
import Layout from '@/layout/index.vue'
import { useUserStore } from '@/stores/user'
import { useMenuStore } from '@/stores/menu'

/**
 * 所有页面组件，菜单表里存的是相对 views 的路径（如 system/menu/index）
 * 用 glob 懒加载，新增页面不用改这里
 */
const viewModules = import.meta.glob('../views/**/*.vue')

/** 把菜单里的组件路径解析成真正的组件，找不到返回 null */
export function resolveView(component) {
  if (!component) return null
  const key = `../views/${component.replace(/^\/+/, '')}.vue`
  return viewModules[key] || null
}

/**
 * 基础路由：登录页、布局容器、404
 * 业务页面全部由后端菜单动态注册到 name 为 Root 的布局路由下
 */
const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue'),
    meta: { title: '登录', public: true }
  },
  {
    path: '/',
    name: 'Root',
    component: Layout,
    redirect: '/dashboard',
    children: []
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/error/404.vue'),
    meta: { title: '页面不存在', public: true }
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior: () => ({ top: 0 })
})

/**
 * 按菜单树注册动态路由，支持无限层级
 * 目录（component 为空）只做分组，不注册路由
 * 返回解析失败的组件路径，方便排查菜单里填错的组件
 */
export function registerMenus(nodes = []) {
  const missing = []

  const walk = (list) => {
    list.forEach((node) => {
      if (node.children?.length) {
        walk(node.children)
      }
      if (node.status !== 1 || !node.component) return

      const view = resolveView(node.component)
      if (!view) {
        missing.push(node.component)
        return
      }

      // name 用 menu-{id}，重复注册会覆盖旧路由，不会出现重复命名报错
      router.addRoute('Root', {
        path: node.path,
        name: `menu-${node.id}`,
        component: view,
        meta: { title: node.title, icon: node.icon, menuId: node.id }
      })
    })
  }

  walk(nodes)
  return missing
}

/** 删除菜单后把对应路由摘掉，避免还能访问已删除页面 */
export function unregisterMenu(id) {
  const name = `menu-${id}`
  if (router.hasRoute(name)) {
    router.removeRoute(name)
  }
}

/**
 * 全局前置守卫：标题 + 恢复登录态 + 加载菜单 + 登录校验
 * VITE_REQUIRE_AUTH=false 时跳过登录校验（方便先看页面）
 */
router.beforeEach(async (to) => {
  const appTitle = import.meta.env.VITE_APP_TITLE || '后台管理系统'
  document.title = to.meta?.title ? `${to.meta.title} - ${appTitle}` : appTitle

  const userStore = useUserStore()
  const menuStore = useMenuStore()

  // 未登录时清掉菜单缓存，下次登录重新拉取
  if (!userStore.isLogin) {
    menuStore.reset()
  }

  // 有 token 但菜单没加载（登录后首次进入、或刷新页面）：先拉菜单再注册路由
  // 注意不能只判断 !to.meta.public：首次访问 / 会先 redirect 到尚未注册的 /dashboard，
  // 那时 to 匹配的是公开的 404 路由，若跳过就会渲染 404 页
  if (to.name !== 'Login' && userStore.isLogin && !menuStore.loaded) {
    await userStore.ensureProfile()
    await menuStore.ensure()
    registerMenus(menuStore.menus)
    // 路由刚加进来，用同一个地址重新匹配一次
    // 这里只带 path/query/hash，避免把 404 的 name、params 一起带过去
    return { path: to.path, query: to.query, hash: to.hash, replace: true }
  }

  const requireAuth = import.meta.env.VITE_REQUIRE_AUTH === 'true'
  if (to.meta?.public || !requireAuth) {
    return true
  }

  if (!userStore.isLogin) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  return true
})

export default router
