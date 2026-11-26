<script setup>
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { createUser, updateUser } from '@/api'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  roles: { type: Array, default: () => [] },
  row: { type: Object, default: null }
})

const emit = defineEmits(['update:modelValue', 'success'])

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const submitting = ref(false)
const editingId = ref(0)
const formRef = ref()

const form = reactive({
  username: '',
  password: '',
  nickname: '',
  email: '',
  role: '',
  status: 1
})

const isEdit = computed(() => editingId.value > 0)

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    {
      pattern: /^[a-zA-Z][a-zA-Z0-9_]{2,31}$/,
      message: '字母开头，仅含字母、数字、下划线，长度 3-32',
      trigger: 'blur'
    }
  ],
  password: [
    { required: true, message: '请输入初始密码', trigger: 'blur' },
    { min: 6, max: 72, message: '密码长度 6-72 位', trigger: 'blur' }
  ],
  nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }],
  email: [{ type: 'email', message: '邮箱格式不正确', trigger: 'blur' }]
}

watch(visible, (open) => {
  if (!open) return
  const row = props.row
  editingId.value = row?.id || 0
  Object.assign(form, {
    username: row?.username || '',
    password: '',
    nickname: row?.nickname || '',
    email: row?.email || '',
    role: row?.role || props.roles[0]?.code || '',
    status: row?.status ?? 1
  })
  nextTick(() => formRef.value?.clearValidate())
})

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEdit.value) {
      await updateUser(editingId.value, {
        nickname: form.nickname,
        email: form.email,
        role: form.role,
        status: form.status
      })
      ElMessage.success('保存成功')
    } else {
      await createUser({ ...form })
      ElMessage.success('新增成功')
    }
    visible.value = false
    emit('success')
  } catch {
    // 失败原因由 axios 拦截器统一弹出
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑用户' : '新增用户'"
    width="520px"
    :close-on-click-modal="false"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
      <el-form-item label="用户名" prop="username">
        <el-input
          v-model.trim="form.username"
          :disabled="isEdit"
          placeholder="字母开头，3-32 位"
        />
      </el-form-item>

      <el-form-item v-if="!isEdit" label="初始密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          show-password
          autocomplete="new-password"
          placeholder="6-72 位，至少 6 位"
        />
      </el-form-item>

      <el-form-item label="昵称" prop="nickname">
        <el-input v-model.trim="form.nickname" placeholder="如：张三" maxlength="64" />
      </el-form-item>

      <el-form-item label="邮箱" prop="email">
        <el-input v-model.trim="form.email" placeholder="选填" maxlength="128" />
      </el-form-item>

      <el-form-item label="角色" prop="role">
        <el-select v-model="form.role" placeholder="请选择角色" style="width: 100%">
          <el-option v-for="r in roles" :key="r.code" :label="r.name" :value="r.code" />
        </el-select>
      </el-form-item>

      <el-form-item label="状态">
        <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
        <span class="form-tip">禁用后该账号无法登录</span>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="submit">
        {{ submitting ? '提交中...' : '确定' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.form-tip {
  margin-left: 8px;
  font-size: 12px;
  color: #909399;
}
</style>
