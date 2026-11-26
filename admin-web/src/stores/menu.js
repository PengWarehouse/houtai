import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { listMenus } from '@/api'

/**
 * 菜单状态
 * 后端返回完整菜单树（含禁用项），侧边栏只渲染启用的部分
 */
export const useMenuStore = defineStore('menu', () => {
  const menus = ref([])
  // 是否已经尝试加载过（成功或失败都算），避免守卫死循环
  const loaded = ref(false)

  /** 只保留启用状态，并递归过滤子级 */
  function filterEnabled(list) {
    return (list || [])
      .filter((item) => item.status === 1)
      .map((item) => ({
        ...item,
        children: item.children?.length ? filterEnabled(item.children) : undefined
      }))
  }

  const sidebarMenus = computed(() => filterEnabled(menus.value))

  /** 拉取菜单树 */
  async function fetchMenus() {
    const data = await listMenus()
    menus.value = data || []
    loaded.value = true
    return menus.value
  }

  /** 确保只加载一次；失败也不阻塞页面，仅侧边栏为空 */
  async function ensure() {
    if (loaded.value) return menus.value
    try {
      return await fetchMenus()
    } catch {
      return menus.value
    } finally {
      loaded.value = true
    }
  }

  /** 退出登录时清空 */
  function reset() {
    menus.value = []
    loaded.value = false
  }

  return { menus, loaded, sidebarMenus, fetchMenus, ensure, reset }
})
