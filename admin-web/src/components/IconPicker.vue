<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import * as ElementPlusIcons from '@element-plus/icons-vue'
import { customIcons } from '@/icons'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '点击选择图标' },
  disabled: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue'])

// 官方图标 + 项目自定义图标，同名时自定义优先
const allIcons = { ...ElementPlusIcons, ...customIcons }
const customNames = Object.keys(customIcons).sort()
const normalNames = Object.keys(ElementPlusIcons).sort()

const visible = ref(false)
const keyword = ref('')
const searchRef = ref()

/** 按关键字过滤，无关键字时保持「自定义在前、官方在后」 */
const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  const names = [...customNames, ...normalNames]
  if (!kw) return names
  return names.filter((name) => name.toLowerCase().includes(kw))
})

/** 分组渲染：搜索时不分段，避免出现只有标题的空组 */
const sections = computed(() => {
  if (keyword.value.trim()) {
    return [{ label: '', names: filtered.value }]
  }
  return [
    { label: '自定义', names: customNames },
    { label: 'Element Plus', names: normalNames }
  ].filter((section) => section.names.length)
})

const currentIcon = computed(() => (props.modelValue ? allIcons[props.modelValue] : null))

// 打开时清空关键字并聚焦搜索框，避免上次的筛选残留
watch(visible, async (open) => {
  if (!open) return
  keyword.value = ''
  await nextTick()
  searchRef.value?.focus()
})

function select(name) {
  emit('update:modelValue', name)
  visible.value = false
}

function clear() {
  emit('update:modelValue', '')
  visible.value = false
}
</script>

<template>
  <el-popover
    v-model:visible="visible"
    trigger="click"
    placement="bottom-start"
    :width="368"
    :disabled="disabled"
    popper-class="icon-picker-popover"
  >
    <template #reference>
      <div class="icon-trigger" :class="{ 'is-empty': !modelValue, 'is-disabled': disabled }">
        <el-icon v-if="currentIcon" class="current"><component :is="currentIcon" /></el-icon>
        <el-icon v-else class="current placeholder-icon"><Plus /></el-icon>
        <span class="text">{{ modelValue || placeholder }}</span>
        <el-icon class="arrow"><ArrowDown /></el-icon>
      </div>
    </template>

    <div class="icon-panel">
      <el-input
        ref="searchRef"
        v-model="keyword"
        placeholder="搜索图标名，如 Role / User / Setting"
        clearable
        size="default"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>

      <div v-if="filtered.length" class="icon-grid">
        <template v-for="section in sections" :key="section.label || 'all'">
          <div v-if="section.label" class="group-label">{{ section.label }}</div>
          <div
            v-for="name in section.names"
            :key="name"
            class="icon-cell"
            :class="{ active: name === modelValue }"
            :title="customNames.includes(name) ? `${name} · 自定义` : name"
            @click="select(name)"
          >
            <el-icon :size="18"><component :is="allIcons[name]" /></el-icon>
          </div>
        </template>
      </div>
      <el-empty v-else description="没有匹配的图标" :image-size="60" />

      <div class="icon-footer">
        <span class="count">
          共 {{ filtered.length }} 个<template v-if="customNames.length">（自定义 {{ customNames.length }}）</template>
        </span>
        <div class="footer-actions">
          <el-button link type="primary" :disabled="!modelValue" @click="clear">清除</el-button>
          <el-button link @click="visible = false">关闭</el-button>
        </div>
      </div>
    </div>
  </el-popover>
</template>

<style scoped lang="scss">
.icon-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 32px;
  padding: 0 11px;
  box-sizing: border-box;
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-blank);
  color: var(--el-text-color-regular);
  cursor: pointer;
  transition: border-color 0.2s;
}

.icon-trigger:hover {
  border-color: var(--el-border-color-hover);
}

.icon-trigger.is-empty .text {
  color: var(--el-text-color-placeholder);
}

.icon-trigger.is-disabled {
  background: var(--el-disabled-bg-color);
  border-color: var(--el-disabled-border-color);
  color: var(--el-disabled-text-color);
  cursor: not-allowed;
}

.icon-trigger .current {
  font-size: 16px;
  color: var(--el-color-primary);
  flex-shrink: 0;
}

.icon-trigger .placeholder-icon {
  color: var(--el-text-color-placeholder);
}

.icon-trigger .text {
  flex: 1;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 14px;
}

.icon-trigger .arrow {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}

.icon-panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.icon-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 4px;
  max-height: 260px;
  overflow-y: auto;
  padding: 2px;
}

.group-label {
  grid-column: 1 / -1;
  margin: 2px 0 -2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.icon-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 38px;
  border: 1px solid transparent;
  border-radius: var(--el-border-radius-base);
  color: var(--el-text-color-regular);
  cursor: pointer;
  transition: all 0.15s;
}

.icon-cell:hover {
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
}

.icon-cell.active {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.icon-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 6px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.icon-footer .count {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.footer-actions {
  display: flex;
  gap: 4px;
}
</style>
