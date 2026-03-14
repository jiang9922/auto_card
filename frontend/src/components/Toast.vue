<!-- 全局轻提示组件：暴露 `toast(msg, type, duration)` 供外界调用 -->
<template>
  <Transition name="toast">
    <div v-if="visible" class="toast" :class="type">
      {{ message }}
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const visible = ref(false)
const message = ref('')
const type = ref('info')
let timer: any

// 核心方法：显示并在指定时长后自动隐藏
const toast = (msg: string, t: string = 'info', duration = 2000) => {
  message.value = msg
  type.value = t
  visible.value = true
  clearTimeout(timer)
  timer = setTimeout(() => { visible.value = false }, duration)
}

// 暴露给父组件（App）以注入全局 `$toast`
defineExpose({ toast })
watch(visible, (v) => { if (!v) clearTimeout(timer) })
</script>

<style scoped>
.toast {
  position: fixed; bottom: 30px; left: 50%; transform: translateX(-50%);
  padding: 12px 24px; border-radius: 8px; color: #fff; font-size: 15px;
  box-shadow: 0 4px 12px rgba(0,0,0,.15); z-index: 9999; min-width: 200px; text-align: center;
}
.toast.success { background: #27ae60; }
.toast.error { background: #e74c3c; }
.toast.info { background: #007bff; }
.toast-enter-active, .toast-leave-active { transition: all .3s; }
.toast-enter-from, .toast-leave-to { opacity: 0; transform: translateX(-50%) translateY(20px); }
</style>