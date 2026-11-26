<script setup>
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { createRole, updateRole } from '@/api'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
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
  code: '',
  name: '',
  description: '',
  sort: 100,
  status: 1
})

const isEdit = computed(() => editingId.value > 0)

const rules = {
  code: [
    { required: true, message: '请输入角色标识', trigger: 'blur' },
    {
      pattern: /^[a-z][a-z0-9_]{1,31}$/,
      message: '小写字母开头，仅含小写字母、数字、下划线，长度 2-32',
      trigger: 'blur'
    }
  ],
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }]
}

watch(visible, (open) => {
  if (!open) return
  const row = props.row
  editingId.value = row?.id || 0
  Object.assign(form, {
    code: row?.code || '',
    name: row?.name || '',
    description: row?.description || '',
    sort: row?.sort ?? 100,
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
      await updateRole(editingId.value, {
        name: form.name,
        description: form.description,
        sort: form.sort,
        status: form.status
      })
      ElMessage.success('保存成功')
    } else {
      await createRole({ ...form })
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
    :title="isEdit ? '编辑角色' : '新增角色'"
    width="520px"
    :close-on-click-modal="false"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
      <el-form-item label="角色标识" prop="code">
        <el-input
          v-model.trim="form.code"
          :disabled="isEdit"
          placeholder="如：editor，创建后不可修改"
        />
      </el-form-item>

      <el-form-item label="角色名称" prop="name">
        <el-input v-model.trim="form.name" placeholder="如：内容编辑" maxlength="64" />
      </el-form-item>

      <el-form-item label="角色说明">
        <el-input
          v-model.trim="form.description"
          type="textarea"
          :rows="2"
          maxlength="255"
          show-word-limit
          placeholder="这个角色能做什么"
        />
      </el-form-item>

      <el-form-item label="排序">
        <el-input-number v-model="form.sort" :min="0" :max="9999" controls-position="right" />
        <span class="form-tip">越小越靠前</span>
      </el-form-item>

      <el-form-item label="状态">
        <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
        <span class="form-tip">超级管理员不允许停用</span>
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
