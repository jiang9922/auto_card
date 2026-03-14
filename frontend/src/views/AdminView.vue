<template>
  <div class="admin-page">
    <div class="header">
      <h2>卡密管理后台</h2>
      <button @click="logout" class="btn-logout">退出登录</button>
    </div>
    <AdminTable />
  </div>
</template>

<script setup lang="ts">
// 管理页：承载 AdminTable，并提供登录校验与退出登录
import AdminTable from '../components/AdminTable.vue'
import { useRouter } from 'vue-router'
import { onMounted } from 'vue'
import { useToast } from '../composables/useToast'

const router = useRouter()
const toast = useToast()

onMounted(() => {
  // 进入页面时若无 token 则重定向到登录
  const token = localStorage.getItem('admin_token')
  if (!token) {
    router.replace('/login')
  }
})

function logout() {
  // 清除 token，提示后跳转到登录页
  localStorage.removeItem('admin_token')
  toast('已退出', 'info')
  router.push('/login')
}
</script>

<style scoped>
.admin-page { max-width: 1000px; margin: 20px auto; padding: 0 20px; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
h2 { color: #333; }
.btn-logout {
  background: #dc3545; color: #fff; border: none; padding: 8px 16px;
  border-radius: 8px; font-size: 14px;
}
.btn-logout:hover { background: #c82333; }
</style>