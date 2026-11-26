<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { deleteMenu } from '@/api'
import { useMenuStore } from '@/stores/menu'
import { registerMenus, unregisterMenu } from '@/router'
import ProTable from '@/components/ProTable.vue'
import MenuFormDialog from './components/MenuFormDialog.vue'

const menuStore = useMenuStore()

const loading = ref(false)
const tableData = computed(() => menuStore.menus)

const formVisible = ref(false)
const formRow = ref(null)
const formParentId = ref(0)

/** 重新拉菜单并重新注册路由，侧边栏和可访问页面立即生效 */
async function load(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    await menuStore.fetchMenus()
    registerMenus(menuStore.menus)
  } finally {
    loading.value = false
  }
}

onMounted(() => load())

function openCreate(parentId = 0) {
  formRow.value = null
  formParentId.value = parentId
  formVisible.value = true
}

function openEdit(row) {
  formRow.value = row
  formParentId.value = row.parent_id
  formVisible.value = true
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`确定删除菜单「${row.title}」吗？`, '提示', { type: 'warning' })
  } catch {
    return
  }

  try {
    await deleteMenu(row.id)
    unregisterMenu(row.id)
    ElMessage.success('删除成功')
    await load(false)
  } catch {
    // 有子菜单时后端会返回「请先删除子菜单」，拦截器已提示
  }
}
</script>

<template>
  <div class="menu-page page-fill">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      class="tip"
      title="菜单改动会即时同步到左侧边栏与路由。组件路径填 views 下的相对路径，例如 system/menu/index；留空表示目录，只做分组不产生页面。"
    />

    <ProTable
      :data="tableData"
      :loading="loading"
      :pagination="false"
      :show-search-buttons="false"
      row-key="id"
      :tree-props="{ children: 'children' }"
      default-expand-all
      border
    >
      <template #search-actions>
        <el-button :icon="'Refresh'" @click="load()">刷新</el-button>
        <el-button type="primary" :icon="'Plus'" @click="openCreate(0)">新增菜单</el-button>
      </template>

      <el-table-column prop="title" label="菜单名称" min-width="180" />

      <el-table-column prop="path" label="路由路径" min-width="150" />

      <el-table-column prop="component" label="组件" min-width="180">
        <template #default="{ row }">
          <span v-if="row.component">{{ row.component }}</span>
          <el-tag v-else size="small" type="info">目录</el-tag>
        </template>
      </el-table-column>

      <el-table-column label="图标" width="90" align="center">
        <template #default="{ row }">
          <el-tooltip v-if="row.icon" :content="row.icon" placement="top">
            <el-icon :size="16"><component :is="row.icon" /></el-icon>
          </el-tooltip>
          <span v-else>-</span>
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
          <el-button link type="primary" @click="openCreate(row.id)">新增子级</el-button>
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </ProTable>

    <MenuFormDialog
      v-model="formVisible"
      :row="formRow"
      :parent-id="formParentId"
      :menus="menuStore.menus"
      @success="load(false)"
    />
  </div>
</template>

<style scoped lang="scss">
.tip {
  margin-bottom: 12px;
}
</style>
