// 组合式函数：获取全局 `$toast` 方法
// 使用：在组件中 `const toast = useToast(); toast('消息', 'success')`
import { getCurrentInstance } from 'vue'

export function useToast() {
  // 从当前组件实例的 app 上下文读取全局属性
  const { appContext } = getCurrentInstance()!
  return appContext.app.config.globalProperties.$toast
}