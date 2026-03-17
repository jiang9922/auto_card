<template>
    <div class="login-card">
      <h2>后台登录</h2>
      <div class="field">
        <input v-model="password" type="password" placeholder="请输入密码" @keyup.enter="login" />
      </div>
      <button @click="login" :disabled="loading" class="btn-login">
        {{ loading ? '登录中...' : '登录' }}
      </button>
      <div v-if="error" class="error">{{ error }}</div>
    </div>
  </template>
  
  <script setup lang="ts">
  // 登录页逻辑说明：
  // - 输入密码，点击或回车触发 `login()`
  // - 调用后端 `POST /api/admin/login`，成功写入 `localStorage.admin_token`
  // - 成功后跳转到 `/admin/manage` 管理页；失败显示错误或 Toast 提示
  import { ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { useToast } from '../composables/useToast'
  
  const toast = useToast()
  const router = useRouter()
  const password = ref('')
  const loading = ref(false)
  const error = ref('')
  
  async function login() {
    // 基本校验：为空时提示
    if (!password.value) return toast('请输入密码', 'error')
    loading.value = true
    error.value = ''
  
    try {
      // 后端登录请求
      const res = await fetch('/api/admin/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password: password.value })
      })
      const json = await res.json()
      if (json.code === 0) {
        // 保存 token 和用户信息
        localStorage.setItem('admin_token', json.data.token)
        localStorage.setItem('user_id', json.data.user_id)
        localStorage.setItem('is_admin', json.data.is_admin)
        toast('登录成功', 'success')
        router.push('/admin/manage')
      } else {
        error.value = json.message
      }
    } catch {
      error.value = '网络错误'
    } finally {
      loading.value = false
    }
  }
  </script>
  
  <style scoped>
  .login-card {
    max-width: 360px; margin: 100px auto; background: #fff;
    padding: 40px; border-radius: 16px; box-shadow: 0 8px 30px rgba(0,0,0,.12);
    text-align: center;
  }
  h2 { margin-bottom: 30px; color: #333; }
  .field { margin-bottom: 20px; }
  input {
    width: 100%; padding: 14px 16px; border: 1px solid #ddd;
    border-radius: 10px; font-size: 16px;
  }
  input:focus { outline: none; border-color: #007bff; }
  .btn-login {
    width: 100%; padding: 14px; background: #28a745; color: #fff;
    border: none; border-radius: 10px; font-size: 16px; font-weight: 500;
  }
  .btn-login:hover { background: #218838; }
  .btn-login:disabled { background: #aaa; }
  .error { color: #e74c3c; margin-top: 12px; }
  </style>
