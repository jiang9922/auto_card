<template>
  <div class="live-codes-page">
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
          <span class="value code">{{ item.card_code }}</span>
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useToast } from '../composables/useToast'

const toast = useToast()

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

// 可见的验证码列表（带剩余时间计算）
const visibleCodes = computed(() => {
  const currentTime = now.value
  return codes.value
    .map(item => {
      const createdTime = new Date(item.created_at).getTime()
      const elapsed = (currentTime - createdTime) / 1000
      const remainingTime = Math.max(0, DISPLAY_DURATION - elapsed)
      return {
        ...item,
        remainingTime
      }
    })
    .filter(item => item.remainingTime > 0)
    .sort((a, b) => b.remainingTime - a.remainingTime)
})

// 获取实时验证码
async function fetchLiveCodes() {
  try {
    const res = await fetch('/api/cards/live?limit=50')
    const json = await res.json()
    if (json.code === 0 && Array.isArray(json.data)) {
      codes.value = json.data
    }
  } catch (err) {
    console.error('获取实时验证码失败:', err)
  }
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
.live-codes-page {
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
