// src/main.ts
// 应用入口与全局逻辑说明
// 1. 创建并挂载 Vue 应用，注册路由
// 2. 注入全局 Toast 方法到 `app.config.globalProperties.$toast`
//    - 实际实现由组件 Toast.vue 暴露的 `toast` 方法提供
//    - 挂载后通过 DOM 查询拿到组件实例的 exposed 方法并赋值
// 3. 页面样式入口在 `./index.css`
import { createApp } from 'vue'
// @ts-ignore - 忽略 App.vue 的类型检查
import App from './App.vue'
import router from './router'
import './index.css'

const app = createApp(App)
app.use(router)

// 全局 Toast
// 使用方式：在任意组件中通过 `useToast()` 获取并调用
// 参数：`msg: string, type = 'info' | 'success' | 'error', duration(ms)`
let toastFn: Function | null = null
app.config.globalProperties.$toast = (msg: string, type = 'info', duration = 2000) => {
  toastFn?.(msg, type, duration)
}

app.mount('#app')
// 挂载后获取 Toast 实例
// 注意：此处通过 DOM 获取 Toast 组件实例的 exposed
// 使得全局 `$toast` 能实际调用组件内的显示逻辑
setTimeout(() => {
  const toastEl = document.querySelector('div.toast')
  // 修复 TypeScript 错误，添加类型检查
  if (toastEl && '__vueParentComponent' in toastEl) {
    const vueComponent = toastEl as any
    if (vueComponent.__vueParentComponent?.exposed) {
      toastFn = vueComponent.__vueParentComponent.exposed.toast as Function
    }
  }
}, 100)