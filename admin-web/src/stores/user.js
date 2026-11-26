import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { login as loginApi, getProfile } from '@/api'
import { getToken, setToken, removeToken } from '@/utils/auth'

/**
 * 用户状态：token + 用户信息
 * 使用组合式写法（setup store），和 Vue 3 的 ref/computed 一致
 *
 * 注意：token 存在 localStorage，刷新后还在；但 userInfo 只在内存里，刷新会丢。
 * 所以路由守卫首次进入时会调 ensureProfile() 重新拉一次，页面才不会显示「未登录」。
 */
export const useUserStore = defineStore('user', () => {
  const token = ref(getToken())
  const userInfo = ref({})
  // 本次会话是否已成功拉取过用户信息，避免每次路由跳转都请求
  const profileLoaded = ref(false)

  const isLogin = computed(() => Boolean(token.value))
  const role = computed(() => userInfo.value.role || '')
  const nickname = computed(() => userInfo.value.nickname || userInfo.value.username || '未登录')

  /** 登录：拿到 token 后持久化 */
  async function login(form) {
    const data = await loginApi(form)
    token.value = data.token
    setToken(data.token)
    if (data.user) {
      userInfo.value = data.user
      profileLoaded.value = true
    }
    return data
  }

  /** 拉取当前用户信息 */
  async function fetchProfile() {
    const data = await getProfile()
    userInfo.value = data || {}
    profileLoaded.value = true
    return userInfo.value
  }

  /**
   * 确保用户信息就绪：刷新页面后由路由守卫调用
   * 没 token 或已加载过直接返回；token 失效则清登录态，交给守卫跳登录页
   */
  async function ensureProfile() {
    if (!token.value || profileLoaded.value) {
      return userInfo.value
    }

    try {
      return await fetchProfile()
    } catch (error) {
      // 只有明确的 401 才清登录态；网络抖动或后端没启动不该把人踢下线
      if (error?.response?.status === 401) {
        logout()
      }
      return null
    }
  }

  /** 退出登录：清状态 + 清本地存储 */
  function logout() {
    token.value = ''
    userInfo.value = {}
    profileLoaded.value = false
    removeToken()
  }

  return {
    token,
    userInfo,
    profileLoaded,
    isLogin,
    role,
    nickname,
    login,
    fetchProfile,
    ensureProfile,
    logout
  }
})
