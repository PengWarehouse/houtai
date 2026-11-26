<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'

const REMEMBER_KEY = 'admin_remember_username'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const formRef = ref(null)
const loading = ref(false)
const form = reactive({
  username: 'admin',
  password: '',
  remember: true
})

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 32, message: '用户名长度为 2-32 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' }
  ]
}

onMounted(() => {
  const saved = localStorage.getItem(REMEMBER_KEY)
  if (saved) {
    form.username = saved
  }
  // 进入页面直接聚焦密码框，省一次点击
  formRef.value?.$el?.querySelector('input[type="password"]')?.focus()
})

function syncRemember(username) {
  if (form.remember) {
    localStorage.setItem(REMEMBER_KEY, username)
  } else {
    localStorage.removeItem(REMEMBER_KEY)
  }
}

async function handleSubmit() {
  if (loading.value) return

  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    await userStore.login({ username: form.username, password: form.password })
    syncRemember(form.username)
    ElMessage.success('登录成功')
    router.replace(route.query.redirect || '/')
  } catch {
    // 错误提示由 axios 拦截器统一处理，这里只做表单复位
    form.password = ''
  } finally {
    loading.value = false
  }
}

function handleForgetPassword() {
  ElMessage.info('请联系系统管理员重置密码')
}

const isDev = import.meta.env.DEV
const requireAuth = import.meta.env.VITE_REQUIRE_AUTH === 'true'
// 模板表达式里不能用 import.meta，先在 script 里取好
const appTitle = import.meta.env.VITE_APP_TITLE || '后台管理系统'
const currentYear = new Date().getFullYear()
</script>

<template>
  <div class="login-page">
    <!-- 左侧品牌区：窄屏自动隐藏 -->
    <section class="login-brand">
      <div class="brand-inner">
        <div class="brand-logo">
          <el-icon :size="28"><Platform /></el-icon>
        </div>
        <h1 class="brand-title">{{ appTitle }}</h1>
        <p class="brand-desc">Go + Vue 3 全栈后台管理系统</p>

        <ul class="brand-points">
          <li><el-icon><CircleCheck /></el-icon> 用户与角色权限管理</li>
          <li><el-icon><CircleCheck /></el-icon> 服务状态实时监控</li>
          <li><el-icon><CircleCheck /></el-icon> 统一请求封装与错误提示</li>
        </ul>
      </div>

      <p class="brand-footer">© {{ currentYear }} {{ appTitle }}</p>
    </section>

    <!-- 右侧表单区 -->
    <section class="login-panel">
      <div class="login-form-wrap">
        <header class="form-header">
          <h2>欢迎回来</h2>
          <p>请使用管理员账号登录系统</p>
        </header>

        <el-form ref="formRef" :model="form" :rules="rules" size="large" @submit.prevent>
          <el-form-item prop="username">
            <el-input
              v-model.trim="form.username"
              placeholder="用户名"
              :prefix-icon="'User'"
              autocomplete="username"
              clearable
            />
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="密码"
              :prefix-icon="'Lock'"
              autocomplete="current-password"
              show-password
              @keyup.enter="handleSubmit"
            />
          </el-form-item>

          <div class="form-extra">
            <el-checkbox v-model="form.remember">记住用户名</el-checkbox>
            <el-link type="primary" :underline="false" @click="handleForgetPassword">
              忘记密码？
            </el-link>
          </div>

          <el-button
            type="primary"
            size="large"
            class="submit-btn"
            :loading="loading"
            @click="handleSubmit"
          >
            {{ loading ? '登录中...' : '登 录' }}
          </el-button>
        </el-form>

        <el-alert
          v-if="isDev && !requireAuth"
          type="info"
          :closable="false"
          class="dev-tip"
          title="开发模式：未开启登录校验"
          description="登录接口已就绪，把 .env.development 的 VITE_REQUIRE_AUTH 改成 true 即可启用强制登录。"
        />
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.login-page {
  display: flex;
  height: 100%;
  background-color: #fff;
}

/* ---------- 左侧品牌区 ---------- */
.login-brand {
  position: relative;
  flex: 1 1 54%;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 56px 8% 32px;
  color: #fff;
  background: linear-gradient(145deg, #0f1c2e 0%, #1b3b5f 48%, #2b5876 100%);
  overflow: hidden;

  &::before,
  &::after {
    content: '';
    position: absolute;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(64, 158, 255, 0.35), transparent 70%);
    pointer-events: none;
  }

  &::before {
    width: 560px;
    height: 560px;
    top: -200px;
    right: -180px;
  }

  &::after {
    width: 400px;
    height: 400px;
    bottom: -160px;
    left: -120px;
  }
}

.brand-inner {
  position: relative;
  z-index: 1;
  max-width: 420px;
}

.brand-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  margin-bottom: 24px;
  border-radius: 14px;
  background-color: rgba(255, 255, 255, 0.12);
  backdrop-filter: blur(4px);
}

.brand-title {
  margin: 0 0 10px;
  font-size: 28px;
  font-weight: 600;
  letter-spacing: 1px;
}

.brand-desc {
  margin: 0 0 36px;
  font-size: 14px;
  color: rgba(255, 255, 255, 0.72);
}

.brand-points {
  margin: 0;
  padding: 0;
  list-style: none;

  li {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 14px;
    font-size: 14px;
    color: rgba(255, 255, 255, 0.85);
  }

  .el-icon {
    color: #67c23a;
  }
}

.brand-footer {
  position: relative;
  z-index: 1;
  margin: 0;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.45);
}

/* ---------- 右侧表单区 ---------- */
.login-panel {
  flex: 1 1 46%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 24px;
}

.login-form-wrap {
  width: 100%;
  max-width: 360px;
  animation: fade-up 0.45s ease both;
}

.form-header {
  margin-bottom: 28px;

  h2 {
    margin: 0 0 8px;
    font-size: 24px;
    font-weight: 600;
    color: #303133;
  }

  p {
    margin: 0;
    font-size: 13px;
    color: #909399;
  }
}

.form-extra {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: -4px 0 22px;
}

.submit-btn {
  width: 100%;
  letter-spacing: 2px;
}

.dev-tip {
  margin-top: 24px;
}

@keyframes fade-up {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 窄屏隐藏品牌区，表单居中 */
@media (max-width: 900px) {
  .login-brand {
    display: none;
  }

  .login-panel {
    flex: 1 1 100%;
    background-color: #f5f7fa;
  }

  .login-form-wrap {
    max-width: 380px;
    padding: 36px 28px;
    border-radius: 12px;
    background-color: #fff;
    box-shadow: 0 8px 30px rgba(0, 21, 41, 0.08);
  }
}
</style>
