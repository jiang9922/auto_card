<template>
  <div class="card">
    <h3>验证码查询</h3>
    <div class="field">
      <label>卡号</label>
      <input v-model="cardNo" @keyup.enter="query" placeholder="请输入卡号" />
    </div>
    <div class="field">
      <label>验证码</label>
      <div class="code-line">
        <input :value="code" readonly placeholder="点击查询获取" />
        <button v-show="code" @click="copyCode" class="btn-copy">
          {{ copyText }}
        </button>
      </div>
    </div>
    <button @click="query" :disabled="loading" class="btn-query">
      {{ loading ? '查询中...' : '立即查询' }}
    </button>

    <div v-if="result" class="result">
      <div class="item"><span>验证码</span><strong>{{ code }}</strong></div>
      <div class="item"><span>获取时间</span><span>{{ formatTime(codeTime) }}</span></div>
    </div>
    <div v-if="error" class="error">{{ error }}</div>
  </div>
</template>

<script setup lang="ts">
// 组件职责：
// - 输入卡号并请求后端 `/api/cards/query?card=...`
// - 展示验证码、获取时间与过期时间
// - 支持从 URL（`card_enc`/`card`/`card_no`）预填卡号，便于分享或从后台跳转
// - 提供复制验证码到剪贴板
import { ref, onMounted } from 'vue'
import { useToast } from '../composables/useToast'

const toast = useToast()
// 展示数据与状态
const cardNo = ref('')           // 显示的卡号（纯卡号）
const queryToken = ref('')       // 完整的 query_token（用于后端查询）
const code = ref('')
const codeTime = ref('')
const cardLink = ref('')
const result = ref(false)
const error = ref('')
const loading = ref(false)
const copyText = ref('复制')

onMounted(() => {
  // 从 URL 参数恢复卡号
  const params = new URLSearchParams(window.location.search)
  const enc = params.get('card_enc')
  if (enc) {
    try { 
      const decoded = atob(enc)
      // 保存完整 query_token 用于查询
      queryToken.value = decoded
      // 显示纯卡号（去掉后缀）
      cardNo.value = decoded.split('_')[0] || decoded
    } catch {}
  } else {
    const plain = params.get('card') || params.get('card_no')
    if (plain) {
      // 保存完整 query_token 用于查询
      queryToken.value = plain
      // 显示纯卡号（去掉后缀）
      cardNo.value = plain.split('_')[0] || plain
    }
  }
})

async function query() {
  // 基本校验与 UI 状态复位
  if (!cardNo.value.trim()) return toast('请输入卡号', 'error')
  loading.value = true
  error.value = ''
  result.value = false
  code.value = ''
  codeTime.value = ''

  // 如果用户手动输入了卡号，需要构建 query_token
  const tokenToQuery = queryToken.value || cardNo.value
  
  console.log('Query debug:', {
    cardNo: cardNo.value,
    queryToken: queryToken.value,
    tokenToQuery: tokenToQuery,
    url: `/api/cards/query?card=${encodeURIComponent(tokenToQuery)}`
  })

  try {
    // 查询页调用后端接口，使用完整的 query_token 查询
    const url = `/api/cards/query?card=${encodeURIComponent(tokenToQuery)}`
    const res = await fetch(url)
    const json = await res.json()
    if (json.code !== 0 || !json.data) {
      code.value = ''
      codeTime.value = ''
      result.value = false
      throw new Error(json.message || '未获取到验证码')
    }
    const d = json.data
    code.value = d.card_code || ''
    if (d.card_note) {
      const note = JSON.parse(d.card_note)
      codeTime.value = note.data?.code_time || ''
    }
    if (!code.value && d.card_note) {
      const m = JSON.parse(d.card_note).data?.code?.match(/\d+/)
      if (m) code.value = m[0]
    }
    if (!code.value) {
      code.value = ''
      result.value = false
      throw new Error('未提取到验证码')
    }
    result.value = true
  } catch (e: any) {
    code.value = ''
    codeTime.value = ''
    result.value = false
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function copyCode() {
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      await navigator.clipboard.writeText(code.value)
      copyText.value = '已复制！'
      toast('复制成功', 'success')
      setTimeout(() => (copyText.value = '复制'), 1500)
      return
    }
    const ta = document.createElement('textarea')
    ta.value = code.value
    ta.style.position = 'fixed'
    ta.style.top = '-9999px'
    document.body.appendChild(ta)
    ta.focus()
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    if (ok) {
      copyText.value = '已复制！'
      toast('复制成功', 'success')
      setTimeout(() => (copyText.value = '复制'), 1500)
    } else {
      throw new Error('复制失败')
    }
  } catch {
    toast('复制失败', 'error')
  }
}

function formatTime(t: string) {
  // 显示时间：若为 RFC3339 则转本地中文格式，否则原样显示
  if (!t) return '—'
  try { return t.includes('T') ? new Date(t).toLocaleString('zh-CN') : t }
  catch { return t }
}
</script>

<style scoped>
.card { background:#fff; padding:24px; border-radius:16px; box-shadow:0 4px 20px rgba(0,0,0,.08); }
h3 { margin-bottom:20px; color:#007bff; font-size:20px; text-align:center; }
.field { margin-bottom:18px; }
label { display:block; margin-bottom:8px; font-weight:500; color:#555; }
input { width:100%; padding:14px 16px; border:1px solid #ddd; border-radius:10px; font-size:16px; }
input:focus { outline:none; border-color:#007bff; }
.code-line { display:flex; gap:10px; align-items:center; }
input[readonly] { background:#f8f9fa; }
.btn-query, .btn-copy {
  padding:14px; border:none; border-radius:10px; font-size:16px; transition:all .2s;
}
.btn-query { background:#007bff; color:#fff; width:100%; font-weight:500; }
.btn-query:hover { background:#0056b3; }
.btn-query:disabled { background:#aaa; cursor:not-allowed; }
.btn-copy { background:#6c757d; color:#fff; padding:10px 16px; font-size:14px; }
.btn-copy:hover { background:#545b62; }
.result { margin-top:24px; padding:16px; background:#f8f9fa; border-radius:10px; }
.item { display:flex; justify-content:space-between; padding:8px 0; border-bottom:1px solid #eee; }
.item:last-child { border:none; }
strong { color:#007bff; font-weight:600; }
.error { color:#e54545; margin-top:12px; text-align:center; }
</style>
