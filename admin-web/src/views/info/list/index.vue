<script setup>
import { onMounted, ref } from 'vue'
import ProTable from '@/components/ProTable.vue'
import { listInfoItems } from '@/api'

// 只列具体分类；不展示「全部」，默认空值即查询全部
const kindOptions = [
  { value: 'news', label: '新闻' },
  { value: 'video', label: '视频' }
]

const kind = ref('')
const keyword = ref('')
const list = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(50)
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    // 未选分类时不传 kind，后端返回所有类型，按 created_at 倒序
    const data = await listInfoItems({
      kind: kind.value || undefined,
      keyword: keyword.value.trim(),
      page: page.value,
      size: size.value
    })
    list.value = data?.items || []
    total.value = data?.total || 0
  } catch {
    // 失败原因由 axios 拦截器统一弹出
  } finally {
    loading.value = false
  }
}

// 条件变化回到第一页，避免停在没有数据的页码上
function reloadFromStart() {
  page.value = 1
  load()
}

function handleSizeChange(value) {
  size.value = value
  page.value = 1
  load()
}

function handleCurrentChange(value) {
  page.value = value
  load()
}

function handleReset() {
  kind.value = ''
  keyword.value = ''
  reloadFromStart()
}

onMounted(load)
</script>

<template>
  <div class="info-page page-fill">
    <ProTable
      :data="list"
      :loading="loading"
      :total="total"
      :current-page="page"
      :page-size="size"
      border
      @search="reloadFromStart"
      @reset="handleReset"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
    >
      <template #search>
        <el-form-item label="分类">
          <el-select
            v-model="kind"
            placeholder="全部分类"
            clearable
            style="width: 140px"
            @change="reloadFromStart"
          >
            <el-option
              v-for="item in kindOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="标题">
          <el-input
            v-model="keyword"
            placeholder="按标题搜索"
            clearable
            style="width: 240px"
            @keyup.enter="reloadFromStart"
          />
        </el-form-item>
      </template>

     
      <el-table-column label="分类" width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="row.kind === 'video' ? 'success' : 'primary'" size="small">
            {{ row.kind === 'video' ? '视频' : '新闻' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="title" label="标题" min-width="260" show-overflow-tooltip />
      <el-table-column prop="source" label="来源" width="130">
        <template #default="{ row }">{{ row.source || '-' }}</template>
      </el-table-column>
      <el-table-column prop="published_at" label="发布日期" width="120" align="center">
        <template #default="{ row }">{{ row.published_at || '-' }}</template>
      </el-table-column>
      <el-table-column label="链接地址" min-width="260" show-overflow-tooltip>
        <template #default="{ row }">
          <el-link type="primary" :href="row.url" target="_blank" :underline="false">
            {{ row.url }}
          </el-link>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="入库时间" width="165" align="center" />
    </ProTable>
  </div>
</template>
