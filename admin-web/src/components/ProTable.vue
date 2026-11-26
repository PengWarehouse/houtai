<script setup>
import { useSlots } from 'vue'

defineOptions({ name: 'ProTable', inheritAttrs: false })

const props = defineProps({
  // 表格数据
  data: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  // 分页相关
  total: { type: Number, default: 0 },
  currentPage: { type: Number, default: 1 },
  pageSize: { type: Number, default: 50 },
  pageSizes: { type: Array, default: () => [50, 100, 200, 500] },
  layout: { type: String, default: 'total, sizes, prev, pager, next' },
  // 是否显示分页
  pagination: { type: Boolean, default: true },
  // 是否显示内置的查询 / 重置按钮
  showSearchButtons: { type: Boolean, default: true }
})

const emit = defineEmits(['search', 'reset', 'size-change', 'current-change'])
const slots = useSlots()

const hasSearch = Boolean(slots.search || slots['search-actions'] || props.showSearchButtons)

function handleSizeChange(size) {
  emit('size-change', size)
}

function handleCurrentChange(page) {
  emit('current-change', page)
}
</script>

<template>
  <div class="pro-table">
    <!-- 头部：搜索 + 工具栏，统一白色面板 -->
    <div v-if="hasSearch || $slots.toolbar" class="pro-table__header">
      <!-- 搜索区：左侧条件（过多自动换行），右侧查询 / 重置 + 扩展插槽 -->
      <el-form v-if="hasSearch" class="pro-table__form" @submit.prevent>
        <div class="pro-table__conditions">
          <slot name="search" />
        </div>
        <div class="pro-table__actions">
          <slot name="search-actions" />
          <template v-if="showSearchButtons">
            <el-button type="primary" :icon="'Search'" @click="emit('search')">查询</el-button>
            <el-button :icon="'Refresh'" @click="emit('reset')">重置</el-button>
          </template>
        </div>
      </el-form>

      <!-- 工具栏：新增按钮等，位于搜索表单下方 -->
      <div v-if="$slots.toolbar" class="pro-table__toolbar">
        <slot name="toolbar" />
      </div>
    </div>

    <!-- 表格：占满搜索区和分页之外的剩余高度 -->
    <div class="pro-table__body">
      <el-table v-loading="loading" :data="data" height="100%" v-bind="$attrs">
        <slot />
      </el-table>
    </div>

    <!-- 分页：可配置关闭 -->
    <div v-if="pagination" class="pro-table__pager">
      <el-pagination
        background
        :layout="layout"
        :total="total"
        :current-page="currentPage"
        :page-size="pageSize"
        :page-sizes="pageSizes"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.pro-table {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

/* 头部面板：搜索 + 工具栏同属一块白色区域 */
.pro-table__header {
  flex-shrink: 0;
  padding: 16px 16px 4px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  background-color: #fff;
}

.pro-table__form {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px 12px;

  :deep(.el-form-item) {
    margin-bottom: 0;
  }
}

.pro-table__conditions {
  display: flex;
  flex: 1 1 320px;
  flex-wrap: wrap;
  align-items: center;
  min-width: 0;

  :deep(.el-form-item) {
    margin-right: 12px;
    margin-bottom: 12px;
  }
}

.pro-table__actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.pro-table__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

/* 表格面板：与头部同宽同边框 */
.pro-table__body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border-radius: 4px;
  background-color: #fff;

  /* 表头样式：深色文字 + 浅灰底 + 取消默认分隔线 */
  :deep(.el-table) {
    --el-table-header-bg-color: #f8f9fb;
    --el-table-header-text-color: #303133;
    --el-table-border-color: #ebeef5;
  }

  :deep(.el-table__header-wrapper th.el-table__cell) {
    height: 44px;
    font-size: 13px;
    font-weight: 600;
    letter-spacing: 0.3px;
    background-color: #f8f9fb;
  }

  :deep(.el-table__body td.el-table__cell) {
    padding: 9px 0;
  }
}

.pro-table__pager {
  display: flex;
  flex-shrink: 0;
  justify-content: flex-end;
}
</style>
