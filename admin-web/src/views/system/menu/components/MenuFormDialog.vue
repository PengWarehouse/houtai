<script setup>
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { createMenu, updateMenu } from '@/api'
import IconPicker from '@/components/IconPicker.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  row: { type: Object, default: null },
  parentId: { type: Number, default: 0 },
  menus: { type: Array, default: () => [] }
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
  parent_id: 0,
  title: '',
  path: '',
  icon: '',
  component: '',
  sort: 100,
  status: 1
})

const isEdit = computed(() => editingId.value > 0)

const rules = {
  title: [{ required: true, message: '请输入菜单名称', trigger: 'blur' }],
  path: [
    { required: true, message: '请输入路由路径', trigger: 'blur' },
    { pattern: /^\//, message: '路由路径需以 / 开头', trigger: 'blur' }
  ]
}

// 上级菜单：加一个「顶级菜单」根节点，方便把菜单提升为一级
const parentOptions = computed(() => [{ id: 0, title: '顶级菜单', children: props.menus }])
const treeProps = { label: 'title', children: 'children' }

watch(visible, (open) => {
  if (!open) return
  const row = props.row
  editingId.value = row?.id || 0
  Object.assign(form, {
    parent_id: row ? row.parent_id : props.parentId,
    title: row?.title || '',
    path: row?.path || '',
    icon: row?.icon || '',
    component: row?.component || '',
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
    if (editingId.value) {
      await updateMenu(editingId.value, { ...form })
      ElMessage.success('保存成功')
    } else {
      await createMenu({ ...form })
      ElMessage.success('新增成功')
    }
    visible.value = false
    emit('success')
  } catch {
    // 错误已由 axios 拦截器统一提示
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="editingId ? '编辑菜单' : '新增菜单'"
    width="520px"
    :close-on-click-modal="false"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="92px">
      <el-form-item label="上级菜单">
        <el-tree-select
          v-model="form.parent_id"
          :data="parentOptions"
          :props="treeProps"
          node-key="id"
          check-strictly
          :render-after-expand="false"
          default-expand-all
          placeholder="不选则为顶级菜单"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="菜单名称" prop="title">
        <el-input v-model="form.title" placeholder="如：菜单管理" maxlength="64" />
      </el-form-item>

      <el-form-item label="路由路径" prop="path">
        <el-input v-model="form.path" placeholder="如：/system/menu" />
      </el-form-item>

      <el-form-item label="组件路径">
        <el-input v-model="form.component" placeholder="如：system/menu/index，目录留空" />
      </el-form-item>

      <el-form-item label="图标">
        <IconPicker v-model="form.icon" />
      </el-form-item>

      <el-form-item label="排序">
        <el-input-number v-model="form.sort" :min="0" :max="9999" controls-position="right" />
        <span class="form-tip">越小越靠前</span>
      </el-form-item>

      <el-form-item label="状态">
        <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
        <span class="form-tip">禁用后不出现在侧边栏，也无法访问</span>
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
