<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { deleteRole, listRoles } from '@/api'
import ProTable from '@/components/ProTable.vue'
import RoleFormDialog from './components/RoleFormDialog.vue'
import PermissionDialog from './components/PermissionDialog.vue'

const loading = ref(false)
const roles = ref([])
const keyword = ref('')

const filteredRoles = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return roles.value
  return roles.value.filter(
    (r) => r.name.toLowerCase().includes(kw) || r.code.toLowerCase().includes(kw)
  )
})

const formVisible = ref(false)
const formRow = ref(null)
const permVisible = ref(false)
const permRow = ref(null)

async function load() {
  loading.value = true
  try {
    roles.value = await listRoles()
  } finally {
    loading.value = false
  }
}

onMounted(load)

function openCreate() {
  formRow.value = null
  formVisible.value = true
}

function openEdit(row) {
  formRow.value = row
  formVisible.value = true
}

function openPerm(row) {
  permRow.value = row
  permVisible.value = true
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(
      `确定删除角色「${row.name}」吗？删除后该角色的用户需要重新分配角色。`,
      '提示',
      { type: 'warning' }
    )
  } catch {
    return
  }

  try {
    await deleteRole(row.id)
    ElMessage.success('删除成功')
    await load()
  } catch {
    // 内置角色、或角色下仍有用户时后端会拒绝，拦截器已提示
  }
}
</script>

<template>
  <div class="role-page page-fill">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      class="tip"
      title="超级管理员为系统内置角色，默认拥有全部菜单权限，无需也不允许分配；其他角色的可见菜单由「分配权限」决定，用户重新登录后生效。"
    />

    <ProTable
      :data="filteredRoles"
      :loading="loading"
      :pagination="false"
      :show-search-buttons="false"
      border
    >
      <template #search>
        <el-form-item label="关键字">
          <el-input
            v-model="keyword"
            placeholder="搜索角色名称 / 标识"
            clearable
            style="width: 200px"
          />
        </el-form-item>
      </template>

      <template #search-actions>
        <el-button type="primary" :icon="'Plus'" @click="openCreate">新增角色</el-button>
      </template>

      <el-table-column prop="id" label="ID" width="70" align="center" />

      <el-table-column label="角色名称" min-width="140">
        <template #default="{ row }">
          <span>{{ row.name }}</span>
          <el-tag v-if="row.builtin" type="danger" size="small" class="builtin-tag">内置</el-tag>
        </template>
      </el-table-column>

      <el-table-column prop="code" label="角色标识" min-width="130" />

      <el-table-column prop="description" label="说明" min-width="200">
        <template #default="{ row }">{{ row.description || '-' }}</template>
      </el-table-column>

      <el-table-column label="用户数" width="90" align="center">
        <template #default="{ row }">{{ row.user_count }}</template>
      </el-table-column>

      <el-table-column label="菜单权限" width="110" align="center">
        <template #default="{ row }">
          <el-tag v-if="row.builtin" type="warning" size="small">全部</el-tag>
          <span v-else>{{ row.menu_count }} 个</span>
        </template>
      </el-table-column>

      <el-table-column prop="sort" label="排序" width="80" align="center" />

      <el-table-column label="状态" width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
            {{ row.status === 1 ? '启用' : '禁用' }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="primary" :disabled="row.builtin" @click="openPerm(row)">
            分配权限
          </el-button>
          <el-button link type="danger" :disabled="row.builtin" @click="handleDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </ProTable>

    <RoleFormDialog v-model="formVisible" :row="formRow" @success="load" />
    <PermissionDialog v-model="permVisible" :row="permRow" />
  </div>
</template>

<style scoped lang="scss">
.tip {
  margin-bottom: 12px;
}

.builtin-tag {
  margin-left: 6px;
}
</style>
