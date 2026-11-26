<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getRoleMenus, listMenus, setRoleMenus } from '@/api'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  row: { type: Object, default: null }
})

const emit = defineEmits(['update:modelValue', 'success'])

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const loading = ref(false)
const saving = ref(false)
const all = ref(false)
const menuTree = ref([])
const treeRef = ref()

const treeProps = { label: 'title', children: 'children' }

/** 递归收集菜单树上所有节点 ID */
function collectMenuIds(nodes, out = []) {
  for (const node of nodes) {
    out.push(node.id)
    if (node.children?.length) collectMenuIds(node.children, out)
  }
  return out
}

watch(visible, async (open) => {
  if (!open || !props.row) return
  loading.value = true
  try {
    const [tree, perm] = await Promise.all([listMenus(), getRoleMenus(props.row.id)])
    menuTree.value = tree || []
    all.value = !!perm.all
    await nextTick()
    treeRef.value?.setCheckedKeys(
      all.value ? collectMenuIds(menuTree.value) : perm.menu_ids || []
    )
  } catch {
    visible.value = false
  } finally {
    loading.value = false
  }
})

/**
 * 勾选子菜单时自动补上父级，否则侧边栏里父目录会缺失，子页面也就进不去。
 * 树用 check-strictly 独立勾选，避免父级一勾就把所有子级连带勾上。
 */
function handleCheck(data, info) {
  const tree = treeRef.value
  if (!tree || !info.checkedKeys.includes(data.id)) return

  let parentId = data.parent_id
  let depth = 0
  while (parentId && depth < 32) {
    const parent = tree.getNode(parentId)
    if (!parent) break
    tree.setChecked(parentId, true, false)
    parentId = parent.data.parent_id
    depth += 1
  }
}

async function save() {
  if (!treeRef.value) return

  saving.value = true
  try {
    await setRoleMenus(props.row.id, treeRef.value.getCheckedKeys())
    ElMessage.success('权限已保存，相关用户重新登录后生效')
    visible.value = false
    emit('success')
  } catch {
    // 已由拦截器提示
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="`分配权限 - ${row?.name || ''}`"
    width="560px"
    :close-on-click-modal="false"
  >
    <el-alert
      v-if="all"
      type="warning"
      :closable="false"
      show-icon
      class="perm-tip"
      title="超级管理员默认拥有全部菜单权限，无需分配，也无法修改。"
    />
    <el-alert
      v-else
      type="info"
      :closable="false"
      show-icon
      class="perm-tip"
      title="勾选该角色可访问的菜单；勾选子菜单会自动补上父级目录。"
    />

    <div v-loading="loading" class="perm-body" :class="{ 'is-readonly': all }">
      <el-tree
        ref="treeRef"
        :data="menuTree"
        :props="treeProps"
        node-key="id"
        show-checkbox
        check-strictly
        default-expand-all
        @check="handleCheck"
      />
    </div>

    <template #footer>
      <el-button @click="visible = false">{{ all ? '关闭' : '取消' }}</el-button>
      <el-button v-if="!all" type="primary" :loading="saving" @click="save">
        {{ saving ? '保存中...' : '保存' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.perm-tip {
  margin-bottom: 12px;
}

.perm-body {
  max-height: 380px;
  overflow-y: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  padding: 8px;
  min-height: 120px;
}

.perm-body.is-readonly {
  pointer-events: none;
  opacity: 0.75;
  background: var(--el-fill-color-lighter);
}
</style>
