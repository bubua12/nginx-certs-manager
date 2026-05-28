/**
 * 应用入口文件 (main.ts)
 * 负责初始化 Vue 3 应用，注册全局插件和组件
 */

// 从 Vue 核心库导入创建应用实例的方法
import { createApp } from 'vue'
// 导入 Pinia 状态管理库（用于全局状态管理，如用户认证信息）
import { createPinia } from 'pinia'
// 导入 Element Plus 组件库（提供 UI 组件）
import ElementPlus from 'element-plus'
// 导入 Element Plus 的全局样式
import 'element-plus/dist/index.css'
// 导入 Element Plus 中文语言包，使组件显示中文文本
import zhCn from 'element-plus/es/locale/lang/zh-cn'
// 导入 Element Plus 所有图标组件，用于全局注册
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

// 导入根组件
import App from './App.vue'
// 导入路由配置
import router from './router'

// 创建 Vue 应用实例，挂载根组件
const app = createApp(App)

// 遍历所有 Element Plus 图标组件，逐个注册为全局组件
// 这样在任意组件中都可以直接使用 <el-icon><Lock /></el-icon> 等图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

// 注册 Pinia 状态管理插件（全局可用的 store）
app.use(createPinia())
// 注册路由插件（页面导航和路由守卫）
app.use(router)
// 注册 Element Plus 插件，并设置语言为中文
app.use(ElementPlus, { locale: zhCn })

// 将应用挂载到 index.html 中 id 为 "app" 的 DOM 元素上
app.mount('#app')
