<!-- 查询页：实时验证码面板，自动轮询显示最新验证码 -->
<template>
  <div class="query-page">
    <!-- 特定卡密查询模式 -->
    <div v-if="isSpecificCardQuery" class="specific-card-mode">
      <h2>验证码查询</h2>
      
      <div class="main-content">
        <!-- 左侧：验证码卡片 -->
        <div class="codes-section">
          <!-- 验证中 -->
          <div v-if="cardLoading" class="code-card single-card">
            <div class="row">
              <span class="label">手机号</span>
              <span class="value phone">{{ cardPhone }}</span>
            </div>
            <div class="status-row">
              <span class="status-text waiting">验证中...</span>
            </div>
          </div>
          
          <!-- 手机号不存在 -->
          <div v-else-if="cardError" class="code-card single-card error-card">
            <div class="row">
              <span class="label">手机号</span>
              <span class="value phone">{{ cardPhone }}</span>
            </div>
            <div class="status-row">
              <span class="status-text error">{{ cardError }}</span>
            </div>
          </div>
          
          <!-- 卡号存在，显示验证码 -->
          <div v-else-if="cardVerified" class="code-card single-card">
            <div class="row">
              <span class="label">手机号</span>
              <span class="value phone">{{ cardMatchedCode?.phone || cardPhone }}</span>
            </div>
            
            <div v-if="cardMatchedCode && cardRemainingTime > 0">
              <div class="row">
                <span class="label">验证码</span>
                <div class="code-wrapper">
                  <span class="value code">{{ cardMatchedCode.card_code || '暂无验证码' }}</span>
                  <button 
                    v-if="cardMatchedCode.card_code" 
                    @click="copyCode(cardMatchedCode.card_code)" 
                    class="btn-copy"
                    :class="{ 'copied': copiedCode === cardMatchedCode.card_code }"
                  >
                    {{ copiedCode === cardMatchedCode.card_code ? '已复制' : '复制' }}
                  </button>
                </div>
              </div>
              
              <div class="row">
                <span class="label">时间</span>
                <span class="value time">{{ formatTime(cardMatchedCode.created_at) }}</span>
              </div>
              
              <!-- 倒计时进度条 -->
              <div class="progress-bar">
                <div 
                  class="progress" 
                  :style="{ width: (cardRemainingTime / 56 * 100) + '%' }"
                  :class="{ 'warning': cardRemainingTime < 15 }"
                ></div>
              </div>
              <div class="countdown">{{ Math.ceil(cardRemainingTime) }}秒后消失</div>
            </div>
            
            <div v-else-if="cardMatchedCode && cardRemainingTime <= 0" class="status-row">
              <span class="status-text expired">验证码已过期，请重新获取</span>
            </div>
            
            <div v-else class="status-row">
              <span class="status-text waiting">暂无验证码，请稍候...</span>
            </div>
          </div>
        </div>
        
        <!-- 右侧：公告 -->
        <div class="notice-section">
          <div class="notice-card">
            <h3>📢 常见问题解决方法</h3>
            
            <div class="notice-item">
              <h4><strong>不来码验证码</strong></h4>
              <p>检查我提供的手机号是否输入正确，区号是否改为香港+852。上述没问题，稍后一分钟再试（可以切换网络尝试一下）。</p>
            </div>
            
            <div class="notice-item">
              <h4><strong>手机号不存在</strong></h4>
              <p>区号未改为香港+852。</p>
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
    
    <!-- 实时验证码面板模式 -->
    <template v-else>
      <h2>实时验证码面板</h2>
      <p class="subtitle">自动刷新，每条显示1分钟后自动消失</p>
      
      <!-- 密码验证 -->
      <div v-if="!isVerified" class="password-section">
        <div class="password-card">
          <h3>🔒 请输入访问密码</h3>
          <input
            v-model="inputPassword"
            type="password"
            placeholder="请输入密码"
            class="password-input"
            @keyup.enter="verifyPassword"
          />
          <button @click="verifyPassword" class="btn-verify">进入</button>
          <div v-if="passwordError" class="password-error">{{ passwordError }}</div>
        </div>
      </div>
      
      <!-- 验证码列表（验证通过后显示） -->
      <template v-else>
        <!-- 用户筛选 -->
        <div class="user-filter" v-if="userIDList.length > 0">
          <label>用户筛选：</label>
          <select 
            v-model="selectedUserID" 
            @change="onUserIDChange"
            @focus="isDropdownOpen = true"
            @blur="isDropdownOpen = false"
          >
            <option value="">全部用户</option>
            <option v-for="uid in userIDList" :key="uid" :value="uid">{{ uid || '未命名' }}</option>
          </select>
        </div>
        
        <div class="main-content center-content">
          <!-- 验证码列表居中 -->
          <div class="codes-section center-section">
          <div class="codes-list">
            <div 
              v-for="item in visibleCodes" 
              :key="item.id" 
              class="code-card"
              :class="{ 'expiring': item.remainingTime < 15 }"
            >
              <div class="row">
                <span class="label">手机号</span>
                <span class="value phone">{{ item.phone }}</span>
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
                  :style="{ width: (item.remainingTime / 60 * 100) + '%' }"
                  :class="{ 'warning': item.remainingTime < 15 }"
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
            <span v-if="codes.length > MAX_DISPLAY_COUNT" class="data-limit-hint">
              (显示最新 {{ MAX_DISPLAY_COUNT }}/{{ codes.length }} 条)
            </span>
          </div>
        </div>
      </div>
      
      <div class="footer">验证码查询系统 v2.0</div>
      </template>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

// 判断是否是特定卡密查询模式（有card参数）
const cardParam = computed(() => route.query.card as string | undefined)
const isSpecificCardQuery = computed(() => !!cardParam.value)

// ===== 实时验证码面板模式 =====
// 原始数据
const codes = ref<any[]>([])
// 当前时间戳
const now = ref(Date.now())
// 轮询状态
const isPolling = ref(false)
// 定时器
let pollTimer: any = null
let countdownTimer: any = null
let cleanupTimer: any = null

// ===== 特定卡密查询模式 =====
const cardMatchedCode = computed(() => {
  // 从 query_token 提取纯卡号（去掉 _随机后缀）
  const token = cardParam.value || ''
  const pureCardNo = token.split('_')[0] || ''
  
  // 在实时验证码中查找匹配的（后5位匹配即可）
  const matched = codes.value.find(item => {
    const itemLast5 = item.phone.slice(-5)
    return itemLast5 === pureCardNo.slice(-5)
  })
  
  // 调试日志
  if (isSpecificCardQuery.value) {
    console.log('[cardMatchedCode] 卡号:', pureCardNo, 'codes数量:', codes.value.length, '匹配:', matched)
  }
  
  return matched
})

// 从查询参数提取手机号（用于始终显示）- 完整显示
const cardPhone = computed(() => {
  const token = cardParam.value || ''
  return token.split('_')[0] || ''
})

// 卡号验证状态
const cardLoading = ref(false)
const cardVerified = ref(false)
const cardError = ref('')

// 查询状态轮询（保留以备异步模式使用）
let queryStatusTimer: any = null
const queryStatus = ref<'pending' | 'querying' | 'completed' | 'failed'>('pending')
const queryError = ref('')
const currentCardData = ref<any>(null)

// 验证卡号并触发查询（按需查询模式 - 异步+轮询）
async function verifyCard(token: string) {
  cardLoading.value = true
  cardError.value = ''
  cardVerified.value = false
  queryStatus.value = 'pending'
  queryError.value = ''
  
  try {
    // 步骤1: 触发异步查询（不带sync=1）
    const res = await fetch(`/api/cards/query?card=${encodeURIComponent(token)}`)
    const json = await res.json()
    
    if (json.code !== 0) {
      // 卡号不存在或其他错误
      cardError.value = json.message || '手机号不存在'
      cardLoading.value = false
      return
    }
    
    // 卡号存在，开始轮询查询状态
    cardVerified.value = true
    
    // 如果已有数据，先显示
    if (json.data?.card) {
      currentCardData.value = json.data.card
      updateCardDisplay(json.data.card)
    }
    
    // 步骤2: 开始轮询查询状态（每1秒一次，最多60次 = 60秒）
    startQueryStatusPolling(token)
    
  } catch (err) {
    cardError.value = '验证失败'
    cardLoading.value = false
  }
}

// 轮询查询状态
let pollCount = 0
const MAX_POLL_COUNT = 60

function startQueryStatusPolling(token: string) {
  pollCount = 0
  cardLoading.value = false // 立即结束加载状态，让用户看到界面
  
  // 清除之前的轮询
  if (queryStatusTimer) {
    clearInterval(queryStatusTimer)
  }
  
  queryStatusTimer = setInterval(async () => {
    pollCount++
    
    // 超过最大轮询次数，停止轮询
    if (pollCount > MAX_POLL_COUNT) {
      clearInterval(queryStatusTimer)
      queryStatusTimer = null
      queryStatus.value = 'completed'
      return
    }
    
    try {
      const res = await fetch(`/api/cards/status?card=${encodeURIComponent(token)}`)
      const json = await res.json()
      
      if (json.code === 0 && json.data) {
        queryStatus.value = json.data.task_status
        queryError.value = json.data.task_error || ''
        currentCardData.value = json.data.card
        
        // 更新显示
        if (currentCardData.value) {
          updateCardDisplay(currentCardData.value)
        }
        
        // 查询完成，停止轮询
        if (queryStatus.value === 'completed' || queryStatus.value === 'failed') {
          clearInterval(queryStatusTimer)
          queryStatusTimer = null
        }
      }
    } catch (err) {
      console.error('轮询查询状态失败:', err)
    }
  }, 1000) // 每1秒轮询一次
}

// 更新卡密显示
function updateCardDisplay(cardData: any) {
  // 更新 cardMatchedCode 用于显示
  if (cardData.card_code) {
    cardMatchedCode.value = {
      id: cardData.id,
      phone: cardData.phone || cardPhone.value,
      card_code: cardData.card_code,
      created_at: cardData.created_at
    }
    // 重置倒计时
    cardFetchedAt.value = Date.now()
  }
}

const cardFetchedAt = ref<number>(0)
const cardRemainingTime = ref(56)
let cardCountdownTimer: any = null

// 显示56秒
const CARD_DISPLAY_DURATION = 56

// 复制状态
const copiedCode = ref('')

// 用户筛选
const userIDList = ref<string[]>([])
const selectedUserID = ref('')
const isDropdownOpen = ref(false) // 下拉框是否打开

// 当用户选择变化时
function onUserIDChange() {
  isDropdownOpen.value = false
  // 强制刷新，不走缓存比较
  fetchLiveCodes(true)
}

// ===== 密码验证 =====
const isVerified = ref(false)
const inputPassword = ref('')
const passwordError = ref('')
const CORRECT_PASSWORD = 'jc123'

function verifyPassword() {
  if (inputPassword.value === CORRECT_PASSWORD) {
    isVerified.value = true
    passwordError.value = ''
  } else {
    passwordError.value = '密码错误'
    isVerified.value = false
  }
}

// 最大显示条数限制，防止大量数据导致页面卡顿
const MAX_DISPLAY_COUNT = 50

// 可见的验证码列表（根据创建时间计算剩余时间）
const visibleCodes = computed(() => {
  // 使用 now.value 确保每秒重新计算
  const currentTime = now.value || Date.now()
  // 只处理最新的 N 条数据，防止大量数据导致卡顿
  const recentCodes = codes.value.slice(0, MAX_DISPLAY_COUNT)
  return recentCodes.map(item => {
    const createdTime = new Date(item.created_at).getTime()
    const elapsed = Math.floor((currentTime - createdTime) / 1000)
    const remaining = Math.max(0, 60 - elapsed) // 60秒（1分钟）倒计时
    return {
      ...item,
      remainingTime: remaining
    }
  })
})

// 自动清理过期验证码 - 防止内存泄漏
function cleanupExpiredCodes() {
  const currentTime = Date.now()
  const beforeCount = codes.value.length
  
  // 只保留最新的 100 条数据（后端可能返回更多，前端限制内存占用）
  // 同时清理已过期超过 1 分钟的数据
  codes.value = codes.value
    .filter(item => {
      const createdTime = new Date(item.created_at).getTime()
      const elapsed = Math.floor((currentTime - createdTime) / 1000)
      return elapsed < 60 // 60秒内保留，超过则删除
    })
    .slice(0, 100) // 最多保留 100 条，防止内存无限增长
  
  const afterCount = codes.value.length
  if (beforeCount !== afterCount) {
    console.log(`清理过期验证码: ${beforeCount - afterCount} 条已删除，剩余 ${afterCount} 条`)
  }
}

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

// 获取实时验证码（面板模式）
async function fetchLiveCodes(force = false) {
  try {
    // 如果下拉框打开且不是强制刷新，跳过本次更新
    if (isDropdownOpen.value && !force) {
      return
    }
    
    // 构建URL，如果有选择用户则添加过滤参数
    let url = '/api/sms/live'
    if (selectedUserID.value) {
      url += `?user_id=${encodeURIComponent(selectedUserID.value)}`
    }
    
    const res = await fetch(url)
    const json = await res.json()
    if (json.code === 0 && json.data) {
      // 新的数据结构: { codes: [], user_ids: [] }
      const data = json.data.codes || json.data
      const userIDs = json.data.user_ids || []
      
      // 转换数据格式
      const newCodes = data.map((item: any) => ({
        id: item.id,
        phone: item.phone,
        card_code: item.code,
        created_at: item.created_at,
        from: item.from,
        msg: item.msg,
        user_id: item.user_id
      }))
      
      // 调试日志：检查卡密查询模式下的匹配情况
      if (isSpecificCardQuery.value && cardParam.value) {
        const token = cardParam.value
        const pureCardNo = token.split('_')[0] || ''
        const matched = newCodes.find((item: any) => {
          const itemLast5 = item.phone.slice(-5)
          return itemLast5 === pureCardNo.slice(-5)
        })
        console.log('[fetchLiveCodes] 卡号:', pureCardNo, '匹配结果:', matched, '总条数:', newCodes.length)
      }
      
      codes.value = newCodes
      
      // 只在 user_id 列表变化时更新
      const currentUserIDs = JSON.stringify(userIDList.value.sort())
      const newUserIDsStr = JSON.stringify(userIDs.sort())
      if (currentUserIDs !== newUserIDsStr) {
        userIDList.value = userIDs
      }
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

// 更新特定卡密倒计时（基于短信创建时间）
function updateCardCountdown() {
  if (cardMatchedCode.value && cardMatchedCode.value.created_at) {
    const createdTime = new Date(cardMatchedCode.value.created_at).getTime()
    const elapsed = Math.floor((Date.now() - createdTime) / 1000)
    cardRemainingTime.value = Math.max(0, CARD_DISPLAY_DURATION - elapsed)
  }
}

// 开始轮询（面板模式）
function startPolling() {
  isPolling.value = true
  fetchLiveCodes() // 立即获取一次（非强制）
  pollTimer = setInterval(() => fetchLiveCodes(), 1000) // 每1秒刷新（非强制）
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

// 处理页面可见性变化 - 修复切换标签页卡住问题
function handleVisibilityChange() {
  if (document.hidden) {
    // 页面切换到后台，暂停轮询（浏览器会自动冻结定时器）
    console.log('页面切换到后台，暂停轮询')
  } else {
    // 页面切回前台，立即刷新数据并重启轮询
    console.log('页面切回前台，恢复轮询')
    fetchLiveCodes(true) // 强制刷新
  }
}

onMounted(() => {
  // 所有模式都启动实时验证码轮询
  startPolling()
  // 每秒更新倒计时
  countdownTimer = setInterval(updateNow, 1000)
  
  // 启动自动清理过期验证码 - 每10秒清理一次，防止内存泄漏
  cleanupTimer = setInterval(cleanupExpiredCodes, 10000)
  
  // 监听页面可见性变化 - 修复切换标签页卡住
  document.addEventListener('visibilitychange', handleVisibilityChange)
  
  // 特定卡密查询模式额外启动倒计时和验证卡号
  const cardQuery = route.query.card as string | undefined
  if (cardQuery) {
    cardCountdownTimer = setInterval(updateCardCountdown, 1000)
    // 验证卡号是否存在
    verifyCard(cardQuery)
  }
})

onUnmounted(() => {
  stopPolling()
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
  if (cardCountdownTimer) {
    clearInterval(cardCountdownTimer)
    cardCountdownTimer = null
  }
  if (cleanupTimer) {
    clearInterval(cleanupTimer)
    cleanupTimer = null
  }
  if (queryStatusTimer) {
    clearInterval(queryStatusTimer)
    queryStatusTimer = null
  }
  // 移除页面可见性监听
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>

<style scoped>
.query-page {
  max-width: 1200px;
  margin: 20px 40px 20px auto;
  padding: 0 20px;
}

/* 特定卡密查询模式 */
.specific-card-mode {
  max-width: 1200px;
  margin: 20px 40px 20px auto;
  padding: 0 20px;
}

.specific-card-mode h2 {
  text-align: center;
  margin-bottom: 24px;
}

.specific-card-mode .main-content {
  display: flex;
  gap: 24px;
  align-items: flex-start;
}

.specific-card-mode .codes-section {
  flex: 1;
  max-width: 600px;
}

.specific-card-mode .notice-section {
  width: 350px;
  flex-shrink: 0;
}

/* 移动端适配 */
@media (max-width: 900px) {
  .specific-card-mode {
    margin: 10px auto;
    padding: 0 12px;
  }
  
  .specific-card-mode h2 {
    font-size: 20px;
    margin-bottom: 16px;
  }
  
  .specific-card-mode .main-content {
    flex-direction: column;
    gap: 16px;
  }
  
  .specific-card-mode .codes-section {
    max-width: 100%;
    width: 100%;
  }
  
  .specific-card-mode .notice-section {
    width: 100%;
  }
}

/* 密码验证 */
.password-section {
  max-width: 400px;
  margin: 40px auto;
}

.password-card {
  background: #fff;
  border-radius: 12px;
  padding: 30px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  text-align: center;
}

.password-card h3 {
  color: #333;
  margin-bottom: 20px;
  font-size: 18px;
}

.password-input {
  width: 100%;
  padding: 12px 16px;
  font-size: 16px;
  border: 2px solid #eee;
  border-radius: 8px;
  margin-bottom: 16px;
  box-sizing: border-box;
  transition: border-color 0.2s;
}

.password-input:focus {
  outline: none;
  border-color: #007bff;
}

.btn-verify {
  width: 100%;
  padding: 12px 24px;
  background: #007bff;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-verify:hover {
  background: #0056b3;
}

.password-error {
  color: #dc3545;
  margin-top: 12px;
  font-size: 14px;
}

.loading {
  text-align: center;
  padding: 40px;
  color: #666;
}

.error-message {
  text-align: center;
  padding: 40px;
  color: #dc3545;
  background: #f8f9fa;
  border-radius: 12px;
}

.single-card {
  max-width: 500px;
  margin: 0 auto;
}

.expired-info {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #eee;
  color: #999;
  font-size: 13px;
  text-align: center;
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

/* 用户筛选 */
.user-filter {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-bottom: 20px;
  padding: 12px 20px;
  background: #f8f9fa;
  border-radius: 8px;
  max-width: 500px;
  margin-left: auto;
  margin-right: auto;
}

.user-filter label {
  font-size: 14px;
  color: #666;
  font-weight: 500;
}

.user-filter select {
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 14px;
  background: white;
  min-width: 150px;
  cursor: pointer;
}

.user-filter select:focus {
  outline: none;
  border-color: #4a90d9;
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

/* 居中布局 */
.center-content {
  justify-content: center;
}

.center-section {
  max-width: 600px;
  margin: 0 auto;
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
    display: flex;
    flex-direction: column;
    align-items: center;
  }
  
  .codes-list {
    width: 100%;
    max-width: 500px;
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
    max-width: 500px;
    margin: 0 auto;
    box-sizing: border-box;
    padding: 16px;
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

.empty.expired {
  color: #dc3545;
  background: #fff5f5;
  border: 1px solid #ffcdd2;
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

.data-limit-hint {
  color: #999;
  font-size: 12px;
  margin-left: 8px;
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
