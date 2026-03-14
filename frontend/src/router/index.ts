// src/router/index.ts
// 路由配置与守卫说明
// 1. 定义应用的页面路由：登录、查询、管理后台
// 2. 受保护页面（requiresAuth）进入前需后端校验管理员 token
// 3. 跳转时设置文档标题，未登录则重定向到登录页
import { createRouter, createWebHistory } from 'vue-router'
import type { RouteLocationNormalized } from 'vue-router'
import LoginView from '../views/LoginView.vue'
import QueryView from '../views/QueryView.vue'
import AdminView from '../views/AdminView.vue'

// 路由表：
// - '/' 重定向到 '/query'
// - '/login' 登录页面
// - '/query' 验证码查询页
// - '/admin' 管理后台（需要登录，路由守卫会校验）
const routes = [
  { path: '/', redirect: '/query' },
  { path: '/login', component: LoginView, meta: { title: '登录' } },
  { path: '/admin', redirect: '/login' },
  { path: '/query', component: QueryView, meta: { title: '实时验证码' } },
  {
    path: '/admin/manage',
    component: AdminView,
    meta: { title: '卡密管理', requiresAuth: true }
  },
   // 通配符路由：除了上面明确指定的管理路径外，所有其他路径都重定向到查询页面
  { path: '/:pathMatch(.*)*', redirect: '/query' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 全局路由守卫说明：
// - 设置页面标题
// - 若目标路由 `requiresAuth` 为真：
//   1) 从 `localStorage.admin_token` 读取 token
//   2) 请求后端 `GET /api/admin/verify` 校验
//   3) 校验失败则清除本地 token 并跳转登录
router.beforeEach(async (to: RouteLocationNormalized, from: RouteLocationNormalized, next) => {
  // 设置页面标题
  const title = to.meta.title as string | undefined
  document.title = title || 'TX系统'

  // 仅对需要登录的路由进行校验
  if (!to.meta.requiresAuth) {
    next()
    return
  }

  // 读取本地 token；没有则跳转登录
  const token = localStorage.getItem('admin_token')
  if (!token) {
    next('/login')
    return
  }

  // 调用后端校验接口；失败则清除本地 token 并跳转登录
  try {
    const res = await fetch('/api/admin/verify', {
      method: 'GET',
      headers: { Authorization: `Bearer ${token}` }
    })
    if (res.status !== 200) {
      localStorage.removeItem('admin_token')
      next('/login')
      return
    }
    const json = await res.json()
    if (json.code !== 0) {
      localStorage.removeItem('admin_token')
      next('/login')
      return
    }
    next()
  } catch (e) {
    localStorage.removeItem('admin_token')
    next('/login')
  }
})

export default router