/**
 * axios 封装
 *
 * 响应约定 { code, message, data }：
 * - code === 200 视为成功，拦截器直接把 data 返回给调用方
 * - 其余 code（失败统一为 0）弹出后端返回的 message 并 reject
 * - HTTP 层错误（401/403/5xx/超时/断网）也优先展示后端 message，
 *   后端没给原因时才用本地翻译的兜底文案
 *
 * 用法：
 *   import request from '@/utils/request'
 *   const data = await request.get('/info')
 *   const res = await request.post('/login', { username, password })
 */
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { getToken, removeToken } from './auth'

/** 业务成功码 */
const CODE_SUCCESS = 200
/** 登录接口路径，401 时不按「登录过期」处理 */
const LOGIN_PATH = '/login'

const service = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ,
  timeout: Number(import.meta.env.VITE_API_TIMEOUT) || 15000,
  headers: { 'Content-Type': 'application/json;charset=utf-8' }
})

// 请求拦截：自动带上 token
service.interceptors.request.use(
  (config) => {
    const token = getToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

/** 取出后端返回的失败原因 */
function pickBackendMessage(data) {
  if (data && typeof data === 'object' && typeof data.message === 'string') {
    return data.message.trim()
  }
  return ''
}

// 响应拦截：解包 + 统一错误提示
service.interceptors.response.use(
  (response) => {
    const res = response.data

    // 非 JSON 或不符合 { code, message, data } 约定时，原样返回
    if (res === null || typeof res !== 'object' || !('code' in res)) {
      return res
    }

    if (res.code === CODE_SUCCESS) {
      return res.data
    }

    // HTTP 200 但业务失败：把后端给的原因弹出来
    const message = pickBackendMessage(res) || '请求失败'
    ElMessage.error(message)
    return Promise.reject(new Error(message))
  },
  (error) => {
    const status = error.response?.status
    const backendMessage = pickBackendMessage(error.response?.data)
    const isLoginApi = String(error.config?.url || '').includes(LOGIN_PATH)

    let message = backendMessage

    // 401 且不是登录接口本身 → 登录态失效，清 token 回登录页
    if (status === 401 && !isLoginApi) {
      removeToken()
      if (!window.location.pathname.startsWith(LOGIN_PATH)) {
        // 这里不 import router，避免循环依赖
        window.location.href = LOGIN_PATH
      }
    }

    if (!message) {
      if (status === 401) {
        message = isLoginApi ? '用户名或密码错误' : '登录已过期，请重新登录'
      } else if (status === 403) {
        message = '没有访问权限'
      } else if (status === 404) {
        message = '接口不存在（404）'
      } else if (status >= 500) {
        message = '服务器开小差了（500）'
      } else if (error.code === 'ECONNABORTED') {
        message = '请求超时，请稍后重试'
      } else if (!error.response) {
        message = '网络异常，请检查后端是否已启动'
      } else {
        message = error.message || '请求失败'
      }
    }

    ElMessage.error(message)
    return Promise.reject(error)
  }
)

export default service
