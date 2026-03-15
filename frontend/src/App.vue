<!-- 顶层应用壳：导航与页面容器、全局 Toast -->
<template>
  <div class="app">
<nav class="nav">
  <router-link to="/query" class="nav-link">实时验证码</router-link>
  <!-- 后台登录入口已屏蔽 -->
  <router-link v-if="authed && !isQueryPage" to="/admin/manage" class="nav-link">管理后台</router-link>
</nav>
    <router-view class="view" />
    <Toast />
  </div>
</template>

<script setup>
// 逻辑说明：
// - authed 从 localStorage 判断是否登录，用于控制“管理后台”导航显示
// - 通过 route 判断当前是否在查询页，避免重复显示导航项
// - 监听 storage 事件以响应其他标签页的登录状态变化
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import Toast from './components/Toast.vue'
const authed = ref(!!localStorage.getItem('admin_token'))
const route = useRoute()
const isQueryPage = computed(() => route.path === '/query')
onMounted(() => {
  window.addEventListener('storage', () => {
    authed.value = !!localStorage.getItem('admin_token')
  })
})
</script>

<style scoped>
.app { max-width: 1000px; margin: 0 auto; padding: 20px; }
.nav {
  display: flex; gap: 20px; margin-bottom: 30px; padding: 16px 0;
  border-bottom: 1px solid #eee; font-weight: 500;
}
.nav-link {
  color: #555; padding: 8px 0; position: relative;
}
.nav-link::after {
  content: ''; position: absolute; bottom: -9px; left: 0; width: 0;
  height: 3px; background: #007bff; transition: width .2s;
}
.nav-link.router-link-active::after { width: 100%; }
.view { animation: fade .3s; }
@keyframes fade { from { opacity: 0; } to { opacity: 1; } }
</style>
