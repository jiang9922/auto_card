<!-- 查询页：实时验证码面板，自动轮询显示最新验证码 -->
<template>
  <div class="query-page">
    <h2>实时验证码面板</h2>
    <p class="subtitle">自动刷新，每条显示2分钟后自动消失</p>
    
    <div class="main-content">
      <!-- 左侧：验证码列表 -->
      <div class="codes-section">
        <div class="codes-list">
          <div 
            v-for="item in visibleCodes" 
            :key="item.id" 
            class="code-card"
            :class="{ 'expiring': item.remainingTime < 30 }"
          >
            <div class="row">
              <span class="label">手机号</span>
              <span class="value phone">
                <span class="masked">******</span><span class="visible">{{ getLast5Digits(item.phone) }}</span>
              </span>
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
      </div>
      
      <!-- 右侧：公告 -->
      <div class="notice-section">
        <div class="notice-card">
          <h3>📢 常见问题解决方法</h3>
          
          <div class="notice-item">
            <h4><strong>不来码验证码</strong></h4>
            <p>检查我提供的手机号是否输入正确，区号是否改为美国+1。上述没问题，稍后一分钟再试（可以切换网络尝试一下）。</p>
          </div>
          
          <div class="notice-item">
            <h4><strong>手机号不存在</strong></h4>
            <p>区号未改为美国+1。</p>
          </div>
          
          <div class="notice-item">
            <h4><strong>填入验证码提示错误</strong></h4>
            <p>验证码超时或者重复点了两次，重新获取即可。</p>
          </div>
          
          <div class="notice-item">
            <h4><strong>登陆出现绑定</strong></h4>
            <p>请返回取消，去应用商店更新一下腾讯视频版本即可直登。</p>
          </div>
          
          <div class="notice-item">
            <h4><strong>播放验证</strong></h4>
            <p>切换主身份登陆播放视频，点立即验证网址接码即可恢复。</p>
          </div>
          
          <div class="notice-item">
            <h4><strong>掉线可以重登</strong></h4>
            <p>本商品验证码链接一个月有效，可以重复登陆，掉线自行重登即可。</p>
            <p>非直充，我提供账号给你登陆，五端通用，任选一台登陆，切换设备退出上一台。</p>
            <p>电视只支持新版云视听极光，不支持NEW极光，不支持第三方定制的电视版本。</p>
          </div>
          
          <div class="notice-item">
            <h4><strong>如需登陆视频联系客服</strong></h4>
          </div>
          
          <div class="notice-footer">
            <p>非上述问题联系客服，异常可换号，不支持退款，谢谢。</p>
          </div>
        </div>
      </div>
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

// 手机号脱敏 - 显示后5位
function maskPhone(phone: string): string {
  if (!phone || phone.length < 11) return phone || '—'
  return '******' + phone.substring(phone.length - 5)
}

// 获取手机号后5位
function getLast5Digits(phone: string): string {
  if (!phone || phone.length < 5) return phone || ''
  return phone.substring(phone.length - 5)
}

// 开始轮询
function startPolling() {
  isPolling.value = true
  fetchLiveCodes() // 立即获取一次
  pollTimer = setInterval(fetchLiveCodes, 1000) // 每1秒刷新
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
  max-width: 1200px;
  margin: 20px 40px 20px auto;
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

/* 主内容区：左右两栏 */
.main-content {
  display: flex;
  gap: 24px;
  align-items: flex-start;
}

/* 左侧：验证码列表 */
.codes-section {
  flex: 1;
  max-width: 600px;
}

/* 右侧：公告 */
.notice-section {
  width: 350px;
  flex-shrink: 0;
}

.notice-card {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border-left: 4px solid #ff6b6b;
}

.notice-card h3 {
  color: #333;
  margin-bottom: 16px;
  font-size: 16px;
  border-bottom: 1px solid #eee;
  padding-bottom: 12px;
}

.notice-item {
  margin-bottom: 16px;
}

.notice-item h4 {
  color: #007bff;
  font-size: 14px;
  margin-bottom: 6px;
}

.notice-item p {
  color: #666;
  font-size: 13px;
  line-height: 1.6;
  margin: 0;
}

.notice-footer {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid #eee;
}

.notice-footer p {
  color: #dc3545;
  font-size: 13px;
  font-weight: 500;
  margin: 0;
}

/* 移动端适配 */
@media (max-width: 900px) {
  .query-page {
    margin: 10px auto;
    padding: 0 12px;
  }
  
  h2 {
    font-size: 20px;
    margin-bottom: 6px;
  }
  
  .subtitle {
    font-size: 12px;
    margin-bottom: 16px;
  }
  
  .main-content {
    flex-direction: column;
    gap: 16px;
  }
  
  .codes-section {
    max-width: 100%;
    width: 100%;
  }
  
  .notice-section {
    width: 100%;
  }
  
  .notice-card {
    padding: 16px;
    border-left: 3px solid #ff6b6b;
  }
  
  .notice-card h3 {
    font-size: 15px;
    margin-bottom: 12px;
    padding-bottom: 10px;
  }
  
  .notice-item {
    margin-bottom: 12px;
  }
  
  .notice-item h4 {
    font-size: 13px;
    margin-bottom: 4px;
  }
  
  .notice-item p {
    font-size: 12px;
    line-height: 1.5;
  }
  
  .notice-footer {
    margin-top: 12px;
    padding-top: 10px;
  }
  
  .code-card {
    width: 100%;
    box-sizing: border-box;
    padding: 14px;
    border-left-width: 3px;
  }
  
  .row {
    padding: 6px 0;
  }
  
  .label {
    font-size: 13px;
  }
  
  .value {
    font-size: 14px;
  }
  
  .code {
    font-size: 22px;
  }
  
  .btn-copy {
    padding: 5px 10px;
    font-size: 12px;
  }
  
  .time {
    font-size: 12px;
  }
  
  .countdown {
    font-size: 11px;
  }
  
  .status {
    font-size: 13px;
    margin-top: 16px;
  }
  
  .footer {
    font-size: 12px;
    margin-top: 20px;
  }
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

.phone .visible {
  font-weight: 700;
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
