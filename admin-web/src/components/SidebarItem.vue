<script setup>
/**
 * 侧边栏菜单项，递归渲染任意层级
 * 有 children 时渲染 el-sub-menu，否则渲染可点击的 el-menu-item
 */
defineProps({
  item: {
    type: Object,
    required: true
  }
})
</script>

<template>
  <el-sub-menu v-if="item.children && item.children.length" :index="item.path || `menu-${item.id}`">
    <template #title>
      <el-icon v-if="item.icon"><component :is="item.icon" /></el-icon>
      <span>{{ item.title }}</span>
    </template>

    <SidebarItem v-for="child in item.children" :key="child.id" :item="child" />
  </el-sub-menu>

  <el-menu-item v-else :index="item.path">
    <el-icon v-if="item.icon"><component :is="item.icon" /></el-icon>
    <template #title>{{ item.title }}</template>
  </el-menu-item>
</template>
