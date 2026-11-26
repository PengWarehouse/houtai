<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { deleteUser, listRoles, listUsers } from '@/api'
import ProTable from '@/components/ProTable.vue'
import UserFormDialog from './components/UserFormDialog.vue'
import ResetPasswordDialog from './components/ResetPasswordDialog.vue'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const roles = ref([])

const query = reactive({
  keyword: '',
  role: '',
  status: '',
  page: 1,
  size: 50
})

const formVisible = ref(false)
const formRow = ref(null)
const resetVisible = ref(false)
const resetRow = ref(null)

async function load() {
  loading.value = true
  try {
    const data = await listUsers({
      page: query.page,
      size: query.size,
      keyword: query.keyword || undefined,
      role: query.role || undefined,
      status: query.status === '' ? undefined : query.status
    })
    list.value = data.list || []
    total.value = data.total || 0
  } finally {
    loading.value = false
  }
}

async function loadRoles() {
  roles.value = await listRoles()
}

onMounted(async () => {
  await loadRoles()
  await load()
})

function handleSearch() {
  query.page = 1
  load()
}

function handleReset() {
  query.keyword = ''
  query.role = ''
  query.status = ''
  query.page = 1
  load()
}

function openCreate() {
  formRow.value = null
  formVisible.value = true
}

function openEdit(row) {
  formRow.value = row
  formVisible.value = true
}

function openReset(row) {
  resetRow.value = row
  resetVisible.value = true
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`确定删除用户「${row.username}」吗？该操作不可恢复。`, '提示', {
      type: 'warning'
    })
  } catch {
    return
  }

  try {
    await deleteUser(row.id)
    ElMessage.success('删除成功')
    if (list.value.length === 1 && query.page > 1) query.page -= 1
    await load()
  } catch {
    // 内置账号或自己的账号会被后端拒绝，拦截器已提示
  }
}

function handleSizeChange(size) {
  query.size = size
  query.page = 1
  load()
}

function handlePageChange(page) {
  query.page = page
  load()
}

function formatTime(value) {
  if (!value) return '-'
  return String(value).replace('T', ' ').slice(0, 19)
}
</script>

<template>
  <div class="user-page page-fill">
    <ProTable
      :data="list"
      :loading="loading"
      :total="total"
      :current-page="query.page"
      :page-size="query.size"
      border
      @search="handleSearch"
      @reset="handleReset"
      @size-change="handleSizeChange"
      @current-change="handlePageChange"
    >
      <template #search>
        <el-form-item label="关键字">
          <el-input
            v-model="query.keyword"
            placeholder="用户名 / 昵称 / 邮箱"
            clearable
            style="width: 200px"
            @keyup.enter="handleSearch"
            @clear="handleSearch"
          />
        </el-form-item>

        <el-form-item label="角色">
          <el-select v-model="query.role" clearable placeholder="全部角色" style="width: 150px">
            <el-option v-for="r in roles" :key="r.code" :label="r.name" :value="r.code" />
          </el-select>
        </el-form-item>

        <el-form-item label="状态">
          <el-select v-model="query.status" clearable placeholder="全部" style="width: 110px">
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
        </el-form-item>
      </template>

      <template #toolbar>
        <el-button type="primary" :icon="'Plus'" @click="openCreate">新增用户</el-button>
      </template>

      <el-table-column prop="id" label="ID" width="70" align="center" />
      <el-table-column prop="username" label="用户名" min-width="110" />
      <el-table-column prop="nickname" label="昵称" min-width="110" />

      <el-table-column prop="email" label="邮箱" min-width="150" show-overflow-tooltip>
        <template #default="{ row }">{{ row.email || '-' }}</template>
      </el-table-column>

      <el-table-column label="角色" min-width="110">
        <template #default="{ row }">
          <el-tag :type="row.role === 'super_admin' ? 'danger' : 'primary'" size="small">
            {{ row.role_name || row.role }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="状态" width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
            {{ row.status === 1 ? '启用' : '禁用' }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="最后登录" width="165">
        <template #default="{ row }">{{ formatTime(row.last_login_at) }}</template>
      </el-table-column>

      <el-table-column label="创建时间" width="165">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>

      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="primary" @click="openReset(row)">重置密码</el-button>
          <el-button
            link
            type="danger"
            :disabled="row.role === 'super_admin'"
            @click="handleDelete(row)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </ProTable>

    <UserFormDialog v-model="formVisible" :roles="roles" :row="formRow" @success="load" />
    <ResetPasswordDialog v-model="resetVisible" :row="resetRow" />
  </div>
</template>
