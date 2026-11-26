<script setup>
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { resetUserPassword } from '@/api'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  row: { type: Object, default: null }
})

const emit = defineEmits(['update:modelValue'])

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const submitting = ref(false)
const formRef = ref()
const form = reactive({ password: '' })

const rules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, max: 72, message: '密码长度 6-72 位', trigger: 'blur' }
  ]
}

watch(visible, (open) => {
  if (!open) return
  form.password = ''
  nextTick(() => formRef.value?.clearValidate())
})

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    await resetUserPassword(props.row.id, form.password)
    ElMessage.success('密码已重置，请通知用户使用新密码登录')
    visible.value = false
  } catch {
    // 已由拦截器提示
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    title="重置密码"
    width="440px"
    :close-on-click-modal="false"
  >
    <el-alert
      type="warning"
      :closable="false"
      show-icon
      class="reset-tip"
      :title="`即将重置「${row?.username || ''}」的登录密码，重置后旧密码立即失效。`"
    />

    <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
      <el-form-item label="新密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          show-password
          autocomplete="new-password"
          placeholder="6-72 位"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="submit">
        {{ submitting ? '提交中...' : '确定重置' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.reset-tip {
  margin-bottom: 16px;
}
</style>
