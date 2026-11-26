<script setup>
/**
 * 修改密码弹窗
 * 用法：<PasswordDialog v-model="visible" />
 * 成功后清 token 并跳登录页（旧 token 虽仍有效，但强迫重新登录更符合习惯）
 */
import { nextTick, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { changePassword } from '@/api'
import { useUserStore } from '@/stores/user'

const visible = defineModel({ type: Boolean, default: false })

const router = useRouter()
const userStore = useUserStore()

const formRef = ref()
const submitting = ref(false)
const form = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const rules = {
  oldPassword: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, max: 72, message: '新密码长度需在 6-72 位之间', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value !== form.newPassword) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

// 每次打开都清空，避免残留上一次输入
watch(visible, async (open) => {
  if (!open) return
  Object.assign(form, { oldPassword: '', newPassword: '', confirmPassword: '' })
  await nextTick()
  formRef.value?.clearValidate()
})

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true

  try {
    await changePassword({
      old_password: form.oldPassword,
      new_password: form.newPassword
    })

    visible.value = false
    ElMessage.success('密码修改成功，请重新登录')
    userStore.logout()
    router.replace('/login')
  } catch {
    // 具体错误已由 axios 拦截器统一提示，这里只负责恢复按钮状态
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    title="修改密码"
    width="420px"
    :close-on-click-modal="false"
    append-to-body
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="86px"
      @submit.prevent="handleSubmit"
    >
      <el-form-item label="原密码" prop="oldPassword">
        <el-input
          v-model="form.oldPassword"
          type="password"
          placeholder="请输入当前密码"
          show-password
          autocomplete="current-password"
        />
      </el-form-item>

      <el-form-item label="新密码" prop="newPassword">
        <el-input
          v-model="form.newPassword"
          type="password"
          placeholder="6-72 位，建议含字母与数字"
          show-password
          autocomplete="new-password"
        />
      </el-form-item>

      <el-form-item label="确认密码" prop="confirmPassword">
        <el-input
          v-model="form.confirmPassword"
          type="password"
          placeholder="请再次输入新密码"
          show-password
          autocomplete="new-password"
          @keyup.enter="handleSubmit"
        />
      </el-form-item>
    </el-form>

    <p class="tip">修改成功后需要使用新密码重新登录。</p>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        {{ submitting ? '提交中...' : '确定修改' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.tip {
  margin: 0 0 4px;
  padding-left: 86px;
  font-size: 12px;
  line-height: 1.6;
  color: #909399;
}
</style>
