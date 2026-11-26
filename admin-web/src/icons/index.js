import NewsPaperIcon from './NewsPaper.vue'
import RoleIcon from './RoleIcon.vue'

/**
 * 项目自定义图标
 * Element Plus 图标包（@element-plus/icons-vue）里没有的，统一放这里，
 * 在 main.js 里和官方图标一起全局注册，之后模板中可直接使用组件名：
 *   <el-icon><Role /></el-icon>
 * 数据库里菜单的 icon 字段保存的就是这里的键名（如 "Role"）。
 */
export const customIcons = {
  Role: RoleIcon,
  NewsPaper: NewsPaperIcon
}
