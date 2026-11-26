<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { useMenuStore } from '@/stores/menu'
import PasswordDialog from '@/components/PasswordDialog.vue'
import SidebarItem from '@/components/SidebarItem.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const menuStore = useMenuStore()

const collapsed = ref(false)
const passwordVisible = ref(false)
const appTitle = import.meta.env.VITE_APP_TITLE || '后台管理系统'

const activeMenu = computed(() => route.path)
// 侧边栏由后端菜单驱动，只取启用的项
const menus = computed(() => menuStore.sidebarMenus)

async function handleUserCommand(command) {
  if (command === 'password') {
    passwordVisible.value = true
    return
  }

  if (command !== 'logout') return

  try {
    await ElMessageBox.confirm('确定要退出登录吗？', '提示', { type: 'warning' })
  } catch {
    return // 取消
  }

  userStore.logout()
  ElMessage.success('已退出登录')
  router.push('/login')
}
</script>

<template>
  <el-container class="app-wrapper">
    <el-aside :width="collapsed ? '64px' : '210px'" class="app-aside">
      <div class="logo">
        <el-icon :size="22"><Platform /></el-icon>
        <span v-show="!collapsed" class="logo-text">{{ appTitle }}</span>
      </div>

      <el-menu
        :default-active="activeMenu"
        :collapse="collapsed"
        :collapse-transition="false"
        router
        unique-opened
        background-color="#001529"
        text-color="#bfcbd9"
        active-text-color="#409eff"
        class="app-menu"
      >
        <SidebarItem v-for="item in menus" :key="item.id" :item="item" />
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="app-header">
        <div class="header-left">
          <el-icon class="collapse-btn" :size="20" @click="collapsed = !collapsed">
            <component :is="collapsed ? 'Expand' : 'Fold'" />
          </el-icon>
          <span class="page-title">{{ route.meta.title }}</span>
        </div>

        <div class="header-right">
          <el-dropdown trigger="click" @command="handleUserCommand">
            <span class="user-info">
              <el-avatar :size="28" class="avatar">
                {{ userStore.nickname.slice(0, 1) }}
              </el-avatar>
              <span class="nickname">{{ userStore.nickname }}</span>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item disabled>角色：{{ userStore.role || '未登录' }}</el-dropdown-item>
                <el-dropdown-item divided command="password">修改密码</el-dropdown-item>
                <el-dropdown-item command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="app-main">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>

    <PasswordDialog v-model="passwordVisible" />
  </el-container>
</template>

<style scoped lang="scss">
.app-wrapper {
  height: 100%;
}

.app-aside {
  background-color: #001529;
  transition: width 0.2s;
  overflow-x: hidden;
}

.logo {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 56px;
  padding: 0 18px;
  color: #fff;
  font-size: 16px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.app-menu {
  border-right: none;
}

.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 16px;
  background-color: #fff;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.collapse-btn {
  cursor: pointer;
  color: #606266;
}

.page-title {
  font-size: 15px;
  font-weight: 500;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  outline: none;
}

.avatar {
  background-color: #409eff;
}

.nickname {
  font-size: 14px;
  color: #303133;
}

.app-main {
  display: flex;
  flex-direction: column;
  padding: 16px;
  background-color: #f5f7fa;
  overflow-y: auto;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
