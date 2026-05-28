/**
 * 环境类型声明文件 (env.d.ts)
 * 为 Vite 环境和 Vue 单文件组件提供 TypeScript 类型声明
 */

// 引入 Vite 客户端类型声明
// 提供 import.meta.env、import.meta.hot 等 Vite 特有 API 的类型支持
/// <reference types="vite/client" />

// 声明 .vue 文件的模块类型
// 使 TypeScript 能够正确识别和导入 .vue 单文件组件
declare module '*.vue' {
  // 导入 Vue 3 的组件类型定义
  import type { DefineComponent } from 'vue'
  // 将每个 .vue 文件声明为一个 Vue 组件
  // 泛型参数分别为 Props、Emit、Slot 的类型，这里使用宽松的 any
  const component: DefineComponent<{}, {}, any>
  // 默认导出该组件
  export default component
}
