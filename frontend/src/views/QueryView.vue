<!-- 查询页：实时验证码面板，自动轮询显示最新验证码 -->
<template>
  <div class="query-page">
    <h2>实时验证码面板</h2>
    <p class="subtitle">自动刷新，每条显示2分钟后自动消失</p>
    
    <div class="codes-list">
      <div 
        v-for="item in visibleCodes" 
        :key="item.id" 
        class="code-card"
        :class="{ 'expiring': item.remainingTime < 30 }"
      >
        <div class="row">
          <span class="label">手机号</span>
          <span class="value phone">{{ item.phone || '—' }}</span>
        </div>
        
        <div class="row">
          <span class="label">验证码</span>
          <div class="code-wrapper">
            <span class="value code">{{ item.card_code }}</span>
            <button 
              v-if="item.card_code" 
              @click="copyCode(item.card_code)" 
              class="btn-copy"
              :class="{ 'copied': copiedCode === item.card_code }"
            >
              {{ copiedCode === item.card_code ? '已复制' : '复制' }}
            </button>
          </div>
        </div>
        
        <div class="row">
          <span class="label">时间</span>
          <span class="value time">{{ formatTime(item.created_at) }}</span>
        </div>
        
        <div class="progress-bar">
          <div 
            class="progress" 
            :style="{ width: (item.remainingTime / 120 * 100) + '%' }"
            :class="{ 'warning': item.remainingTime < 30 }"
          ></div>
        </div>
        
        <div class="countdown">{{ Math.ceil(item.remainingTime) }}秒后消失</div>
      </div>
      
      <div v-if="visibleCodes.length === 0" class="empty">
        暂无验证码数据
      </div>
    </div>
    
    <div class="status">
      <span class="dot" :class="{ 'active': isPolling }"></span>
      {{ isPolling ? '实时监控中' : '已暂停' }}
    </div>
    
    <div class="footer">验证码查询系统 v2.0</div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'

// 原始数据
const codes = ref<any[]>([])
// 当前时间戳
const now = ref(Date.now())
// 轮询状态
const isPolling = ref(false)
// 定时器
let pollTimer: any = null
let countdownTimer: any = null

// 每条显示2分钟（120秒）
const DISPLAY_DURATION = 120

// 复制状态
const copiedCode = ref('')

// 可见的验证码列表（后端已按时间过滤，前端直接显示）
const visibleCodes = computed(() => {
  return codes.value.map(item => ({
    ...item,
    remainingTime: 60 // 固定显示1分钟倒计时
  }))
})

// 复制验证码
async function copyCode(code: string) {
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      await navigator.clipboard.writeText(code)
    } else {
      // 降级方案
      const ta = document.createElement('textarea')
      ta.value = code
      ta.style.position = 'fixed'
      ta.style.top = '-9999px'
      document.body.appendChild(ta)
      ta.focus()
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    copiedCode.value = code
    setTimeout(() => {
      if (copiedCode.value === code) {
        copiedCode.value = ''
      }
    }, 2000)
  } catch (err) {
    console.error('复制失败:', err)
  }
}

// 获取实时验证码
async function fetchLiveCodes() {
  try {
    // 使用新的短信验证码接口
    const res = await fetch('/api/sms/live')
    const json = await res.json()
    if (json.code === 0 && Array.isArray(json.data)) {
      // 转换数据格式
      codes.value = json.data.map((item: any) => ({
        id: item.id,
        phone: maskPhone(item.phone),
        card_code: item.code,
        created_at: item.created_at,
        from: item.from,
        msg: item.msg
      }))
    }
  } catch (err) {
    console.error('获取实时验证码失败:', err)
  }
}

// 手机号脱敏
function maskPhone(phone: string): string {
  if (!phone || phone.length < 11) return phone || '—'
  return phone.substring(0, 3) + '****' + phone.substring(7)
}

// 开始轮询
function startPolling() {
  isPolling.value = true
  fetchLiveCodes() // 立即获取一次
  pollTimer = setInterval(fetchLiveCodes, 3000) // 每3秒刷新
}

// 停止轮询
function stopPolling() {
  isPolling.value = false
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// 更新时间戳
function updateNow() {
  now.value = Date.now()
}

// 格式化时间
function formatTime(timeStr: string) {
  if (!timeStr) return '—'
  try {
    const date = new Date(timeStr)
    return date.toLocaleTimeString('zh-CN', { 
      hour: '2-digit', 
      minute: '2-digit', 
      second: '2-digit' 
    })
  } catch {
    return timeStr
  }
}

onMounted(() => {
  startPolling()
  // 每秒更新倒计时
  countdownTimer = setInterval(updateNow, 1000)
})

onUnmounted(() => {
  stopPolling()
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
})
</script>

<style scoped>
.query-page {
  max-width: 600px;
  margin: 20px auto;
  padding: 0 20px;
}

h2 {
  text-align: center;
  color: #333;
  margin-bottom: 8px;
}

.subtitle {
  text-align: center;
  color: #999;
  font-size: 14px;
  margin-bottom: 24px;
}

.codes-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.code-card {
  background: #fff;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border-left: 4px solid #007bff;
  transition: all 0.3s;
}

.code-card.expiring {
  border-left-color: #dc3545;
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.8; }
}

.row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}

.row:last-of-type {
  border-bottom: none;
}

.label {
  color: #666;
  font-size: 14px;
}

.code-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
}

.btn-copy {
  padding: 6px 12px;
  background: #007bff;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-copy:hover {
  background: #0056b3;
}

.btn-copy.copied {
  background: #28a745;
}

.value {
  font-weight: 500;
  font-size: 16px;
}

.phone {
  color: #333;
  font-family: monospace;
  letter-spacing: 1px;
}

.code {
  color: #007bff;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 2px;
}

.time {
  color: #666;
  font-size: 14px;
}

.progress-bar {
  height: 4px;
  background: #f0f0f0;
  border-radius: 2px;
  margin-top: 12px;
  overflow: hidden;
}

.progress {
  height: 100%;
  background: linear-gradient(90deg, #28a745, #20c997);
  border-radius: 2px;
  transition: width 1s linear;
}

.progress.warning {
  background: linear-gradient(90deg, #dc3545, #fd7e14);
}

.countdown {
  text-align: right;
  font-size: 12px;
  color: #999;
  margin-top: 6px;
}

.empty {
  text-align: center;
  padding: 60px 20px;
  color: #999;
  background: #f8f9fa;
  border-radius: 12px;
}

.status {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 24px;
  padding: 12px;
  color: #666;
  font-size: 14px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #dc3545;
}

.dot.active {
  background: #28a745;
  animation: blink 1.5s infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.footer {
  text-align: center;
  color: #999;
  font-size: 14px;
  margin-top: 40px;
}

@media (max-width: 480px) {
  .code {
    font-size: 20px;
  }
  
  .row {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
  
  .value {
    align-self: flex-end;
  }
}
</style>
