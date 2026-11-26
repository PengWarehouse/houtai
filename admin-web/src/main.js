import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import 'element-plus/dist/index.css'

import App from './App.vue'
import router from './router'
import pinia from './stores'
import { customIcons } from './icons'
import '@/styles/index.scss'

const app = createApp(App)

// 全局注册图标：Element Plus 官方图标 + 项目自定义图标（同名以自定义为准）
for (const [name, component] of Object.entries({ ...ElementPlusIconsVue, ...customIcons })) {
  app.component(name, component)
}

app.use(pinia)
app.use(router)
app.use(ElementPlus, { locale: zhCn, size: 'default' })

app.mount('#app')
