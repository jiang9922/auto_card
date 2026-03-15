<template>
  <div class="admin">
    <div class="toolbar">
      <button @click="confirmDelete" :disabled="selected.length === 0" class="btn-del">
        删除选中项
      </button>
      <button @click="exportSelected" :disabled="selected.length === 0" class="btn-export">
        导出选中项
      </button>
      <!-- 分页大小设置 -->
      <div class="page-size-control">
        <label>每页显示：</label>
        <input 
          type="number" 
          v-model.number="pageSize" 
          @change="changePageSize" 
          min="1" 
          max="1000"
          class="page-size-input"
        />
        <span>条</span>
      </div>
    </div>
    
    <!-- 筛选区域 -->
    <div class="filter-section">
      <div class="filter-row">
        <div class="filter-item">
          <label>卡号搜索：</label>
          <input type="text" v-model="filters.cardNo" @input="onCardNoSearch" placeholder="输入卡号搜索" />
        </div>
        <div class="filter-item">
          <label>日期筛选：</label>
          <input type="date" v-model="filters.date" @change="load(1)" />
        </div>
        <div class="filter-item">
          <label>状态筛选：</label>
          <select v-model="filters.status" @change="load(1)">
            <option value="">全部</option>
            <option value="checked">已获取</option>
            <option value="unchecked">未获取</option>
          </select>
        </div>
        <div class="filter-item">
          <button @click="clearFilters" class="btn-clear">清除筛选</button>
        </div>
      </div>
    </div>
    
    <table>
      <thead>
        <tr>
          <th>
            <label class="select-all">
              <input type="checkbox" :checked="isAllSelected" @change="toggleSelectAll" />
              全选
            </label>
          </th>
          <th>序号</th>
          <th>备注</th>
          <th>卡号</th>
          <th>随机码</th>
          <th>链接</th>
          <th>状态</th>
          <th>添加时间</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(c, index) in displayedCards" :key="c.id">
          <td><input type="checkbox" :value="c.id" v-model="selected" /></td>
          <td>{{ (pagination.page - 1) * pagination.page_size + index + 1 }}</td>
          <td><input type="text" v-model="c.remark" @blur="saveRemark(c)" placeholder="添加备注" class="remark-input" /></td>
          <td>
            {{ c.card_no }}
            <a :href="c.query_url || `/query?card=${c.card_no}`" class="query-link">查询</a>
          </td>
          <td>
            <span class="random-code">{{ getRandomSuffix((c as any).query_token || '') }}</span>
          </td>
          <td>
            <a :href="c.card_link" target="_blank" class="link" :title="c.card_link">
              {{ truncate(c.card_link, 50) }}
            </a>
          </td>
          <td>{{ c.card_check ? '已获取' : '未获取' }}</td>
          <td>
            <div class="time-cell">
              <div class="date">{{ formatDate(c.created_at) }}</div>
              <div class="time">{{ formatTime(c.created_at) }}</div>
            </div>
          </td>
        </tr>
      </tbody>
    </table>

    <!-- 分页控件 -->
    <div class="pagination" v-if="pagination.total_pages > 1">
      <button @click="goToPage(1)" :disabled="pagination.page === 1">首页</button>
      <button @click="goToPage(pagination.page - 1)" :disabled="pagination.page === 1">上一页</button>
      <span class="page-info">
        第 {{ pagination.page }} 页，共 {{ pagination.total_pages }} 页
      </span>
      <button @click="goToPage(pagination.page + 1)" :disabled="pagination.page === pagination.total_pages">下一页</button>
      <button @click="goToPage(pagination.total_pages)" :disabled="pagination.page === pagination.total_pages">末页</button>
    </div>

    <div class="add-section">
      <h3>批量添加卡密</h3>
      <p>格式：每行一个 卡号----链接</p>
      <textarea v-model="input" placeholder="888888----http://localhost:8081/api12828798dss
888888----http://localhost:8081/api871282"></textarea>
      <div class="add-options">
        <div class="option-row">
          <label class="checkbox-label">
            <input type="checkbox" v-model="allowDuplicates" />
            <span>允许重复添加（同一卡号可生成多个查询链接）</span>
          </label>
        </div>
        <div class="option-row">
          <label>添加倍数：</label>
          <input type="number" v-model.number="addMultiplier" min="1" max="10" class="multiplier-input" />
          <span class="hint">每条卡密复制{{ addMultiplier }}份</span>
        </div>
        <div class="option-row">
          <label>排序方式：</label>
          <select v-model="sortMode" class="sort-select">
            <option value="sequential">顺序排列 (1234512345)</option>
            <option value="paired">成对排列 (1122334455)</option>
            <option value="random">随机乱序</option>
          </select>
        </div>
        <div class="option-row">
          <input type="text" v-model="batchRemark" placeholder="批量备注（可选）" class="inline-remark-input" />
        </div>
      </div>
      <div class="validate">
        <span class="ok">有效 {{ validCount }} 条</span>
        <span class="bad" v-if="errors.length">无效 {{ errors.length }} 条</span>
        <ul v-if="errors.length">
          <li v-for="e in errors.slice(0,5)" :key="e">{{ e }}</li>
        </ul>
      </div>
      <button @click="add" :disabled="adding" class="btn-add">
        {{ adding ? '提交中...' : '提交添加' }}
      </button>
      <div v-if="msg" :class="msgType">{{ msg }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 管理表格逻辑说明：
// - 列表加载：`GET /api/cards`，支持日期与状态筛选，分页展示
// - 批量添加：`POST /api/cards`，按行解析"卡号----链接"
// - 批量删除：`DELETE /api/admin/batch-delete` 按 id 删除
// - 批量导出（前端生成）：导出为"卡号----查询页链接(card_enc)"文本
// - 选中项支持全选，分页切换时清空已选，以避免误删
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useToast } from '../composables/useToast'

const toast = useToast()
// 列表与分页数据
const cards = ref<any[]>([])
const displayedCards = ref<any[]>([])
const pagination = ref({
  page: 1,
  page_size: 10,
  total: 0,
  total_pages: 0
})

// 分页大小
const pageSize = ref(10)

// 批量添加输入与校验
const input = ref('')
const adding = ref(false)
const msg = ref('')
const msgType = ref('')
const allowDuplicates = ref(true) // 默认允许重复添加
const addMultiplier = ref(1) // 添加倍数，默认1倍
const sortMode = ref('sequential') // 排序方式：sequential/paired/random
const batchRemark = ref('') // 批量备注
// 选中项与校验错误统计
const selected = ref<number[]>([])
const errors = ref<string[]>([])
const validCount = ref(0)

// 筛选条件
const filters = ref({
  cardNo: '',
  date: '',
  status: ''
})

// 页面可见性变化处理：切回时刷新数据
function handleVisibilityChange() {
  if (document.visibilityState === 'visible') {
    // 页面重新可见时刷新当前页数据
    load(pagination.value.page)
  }
}

onMounted(() => {
  load(1)
  // 监听页面可见性变化
  document.addEventListener('visibilitychange', handleVisibilityChange)
})

// 清理定时器和事件监听
onUnmounted(() => {
  cleanupTimeout()
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})

async function load(page: number = 1) {
  // 构建列表查询 URL，并应用日期/状态筛选
  try {
    // 构建查询参数
    let url = `/api/cards?page=${page}&page_size=${pageSize.value}`
    
    // 添加筛选参数
    if (filters.value.cardNo) {
      url += `&card_no=${encodeURIComponent(filters.value.cardNo)}`
    }
    if (filters.value.date) {
      url += `&date=${filters.value.date}`
    }
    if (filters.value.status) {
      url += `&status=${filters.value.status}`
    }
    
    const res = await fetch(url)
    const json = await res.json()
    
    if (json.code === 0 && json.data) {
      cards.value = Array.isArray(json.data.cards) ? json.data.cards : []
      displayedCards.value = cards.value
      // 后端已返回筛选后的数据，不再进行前端筛选
      pagination.value = json.data.pagination || {
        page: 1,
        page_size: pageSize.value,
        total: 0,
        total_pages: 0
      }
    } else {
      cards.value = []
      displayedCards.value = []
      pagination.value = {
        page: 1,
        page_size: pageSize.value,
        total: 0,
        total_pages: 0
      }
      toast('加载失败', 'error')
    }
  } catch (err) {
    console.error('加载失败:', err)
    toast('加载失败', 'error')
  }
}

function changePageSize() {
  // 改变分页大小后回到第一页并重新拉取数据
  load(1)
}

// 卡号搜索：触发后端搜索
let searchTimeout: any = null

// 清理定时器
function cleanupTimeout() {
  if (searchTimeout) {
    clearTimeout(searchTimeout)
    searchTimeout = null
  }
}

function onCardNoSearch() {
  // 防抖处理，避免频繁请求
  cleanupTimeout()
  searchTimeout = setTimeout(() => {
    load(1)
  }, 300)
}

function applyFilters() {
  // 前端筛选：根据卡号搜索、日期、状态筛选
  let result = cards.value
  
  // 卡号搜索
  if (filters.value.cardNo.trim()) {
    const keyword = filters.value.cardNo.toLowerCase()
    result = result.filter(c => c.card_no.toLowerCase().includes(keyword))
  }
  
  // 状态筛选
  if (filters.value.status === 'checked') {
    result = result.filter(c => c.card_check)
  } else if (filters.value.status === 'unchecked') {
    result = result.filter(c => !c.card_check)
  }
  
  // 日期筛选
  if (filters.value.date) {
    result = result.filter(c => {
      const cardDate = c.created_at.split('T')[0]
      return cardDate === filters.value.date
    })
  }
  
  displayedCards.value = result
}

function clearFilters() {
  // 清除筛选条件
  filters.value.cardNo = ''
  filters.value.date = ''
  filters.value.status = ''
  // 重新加载数据（无筛选条件）
  load(1)
}

function goToPage(page: number) {
  // 分页跳转并清空选中项，避免跨页误操作
  if (page >= 1 && page <= pagination.value.total_pages && page !== pagination.value.page) {
    load(page)
    // 清空选中项
    selected.value = []
  }
}

async function del(ids: number[]) {
  // 后端批量删除接口：传递 id 数组
  try {
    await fetch('/api/admin/batch-delete', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ids })
    })
    toast('删除成功', 'success')
    selected.value = []
    // 重新加载当前页
    load(pagination.value.page)
  } catch { toast('删除失败', 'error') }
}

function confirmDelete() {
  // 弹窗确认后执行删除
  if (selected.value.length === 0) return
  if (confirm(`确定要删除选中的 ${selected.value.length} 项吗？`)) {
    del(selected.value)
  }
}

function exportSelected() {
  // 前端导出：生成 "卡号 空格 查询页链接" 文本并下载
  if (selected.value.length === 0) return
  
  // 根据选中 id 从 cards 查找对应数据
  const selectedCards = cards.value.filter((c: any) => selected.value.includes(c.id))
  
  // 生成导出文本：卡号 空格 查询链接（纯链接格式）
  const lines = selectedCards.map((c: any) => {
    // 优先使用后端返回的 query_url，其次用 query_token 构建
    const queryUrl = c.query_url || `${window.location.origin}/query?card=${encodeURIComponent(c.query_token || c.card_no)}`
    return `${c.card_no} ${queryUrl}`
  })
  
  const blob = new Blob([lines.join('\n')], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `cards_export_${new Date().toISOString().slice(0,10)}.txt`
  a.click()
  URL.revokeObjectURL(url)
}

// 输入校验：实时解析输入文本，统计有效/无效行
watch(input, () => {
  const lines = input.value.split('\n')
  errors.value = []
  let count = 0
  lines.forEach((line, idx) => {
    line = line.trim()
    if (!line) return
    const parts = line.split('----')
    // 格式校验：必须恰好两部分，卡号非空，链接符合 http 开头
    if (parts.length !== 2 || !parts[0]?.trim() || !parts[1]?.trim().startsWith('http')) {
      errors.value.push(`第${idx + 1}行格式错误`)
    } else {
      count++
    }
  })
  validCount.value = count
})

async function add() {
  // 批量添加：前端校验后 POST 到后端
  if (validCount.value === 0) return toast('请输入有效卡密', 'error')
  adding.value = true
  msg.value = ''
  try {
    // 处理输入：按倍数复制并排序
    const processedText = processInputWithOptions(input.value)
    
    const res = await fetch('/api/cards', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ 
        text: processedText,
        allow_duplicates: allowDuplicates.value,
        remark: batchRemark.value
      })
    })
    const json = await res.json()
    if (json.code !== 0) throw new Error(json.message)
    msg.value = json.message || `成功添加 ${json.data?.length || 0} 条`
    msgType.value = 'success'
    input.value = ''
    batchRemark.value = ''
    validCount.value = 0
    errors.value = []
    // 清除筛选条件，刷新列表显示新添加的卡密
    filters.value.cardNo = ''
    filters.value.date = ''
    filters.value.status = ''
    load(1)
  } catch (e: any) {
    msg.value = e.message || '添加失败'
    msgType.value = 'error'
  } finally {
    adding.value = false
  }
}

function truncate(s: string, len: number) {
  // 截断长字符串并加省略号
  return s.length > len ? s.slice(0, len) + '...' : s
}

// 从 query_token 提取随机后缀（6位随机字母）
function getRandomSuffix(queryToken: any): string {
  if (!queryToken) return '-'
  const parts = String(queryToken).split('_')
  return parts.length > 1 && parts[1] ? parts[1] : '-'
}

// 处理输入：按倍数复制并排序
function processInputWithOptions(text: string): string {
  const lines = text.split('\n').filter(line => line.trim())
  let processed: string[] = []
  
  // 按倍数复制
  for (let i = 0; i < addMultiplier.value; i++) {
    processed.push(...lines)
  }
  
  // 排序
  if (sortMode.value === 'sequential') {
    // 顺序排列：1234512345
    processed = processed.sort((a, b) => {
      const cardA = a.split('----')[0] || ''
      const cardB = b.split('----')[0] || ''
      return cardA.localeCompare(cardB)
    })
  } else if (sortMode.value === 'paired') {
    // 成对排列：1122334455
    processed = sortPaired(processed)
  } else if (sortMode.value === 'random') {
    // 随机乱序
    processed = shuffleArray(processed)
  }
  
  return processed.join('\n')
}

// 成对排序：将相同的卡号放在一起
function sortPaired(lines: string[]): string[] {
  const groups: Record<string, string[]> = {}
  
  lines.forEach(line => {
    const cardNo = line.split('----')[0] || ''
    if (!groups[cardNo]) groups[cardNo] = []
    groups[cardNo].push(line)
  })
  
  const result: string[] = []
  Object.keys(groups).sort().forEach(cardNo => {
    const group = groups[cardNo]
    if (group) result.push(...group)
  })
  
  return result
}

// 随机乱序
function shuffleArray(array: string[]): string[] {
  const result: string[] = [...array]
  for (let i = result.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    // @ts-ignore
    const temp = result[i]
    // @ts-ignore
    result[i] = result[j]
    // @ts-ignore
    result[j] = temp
  }
  return result
}

function formatDate(s: string) {
  // 显示日期（YYYY-MM-DD），异常时返回原字符串
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`
}

function formatTime(s: string) {
  // 显示时间（HH:MM），异常时返回原字符串
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return `${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}

// 全选/取消全选 - 始终只操作当前一页（displayedCards）
const isAllSelected = computed(() => {
  if (displayedCards.value.length === 0) return false
  // 检查当前页每一项是否都被选中
  return displayedCards.value.every((c: any) => selected.value.includes(c.id))
})

function toggleSelectAll() {
  if (isAllSelected.value) {
    // 取消全选：清空所有选中（跨页）
    selected.value = []
  } else {
    // 全选：只勾选当前一页显示的卡密
    const currentIds = displayedCards.value.map((c: any) => c.id)
    selected.value = [...new Set([...selected.value, ...currentIds])]
  }
}

function encUrl(u: string) {
  try { return btoa(u) } catch { return '' }
}

// 保存备注
async function saveRemark(card: any) {
  try {
    const res = await fetch('/api/cards/' + card.id + '/remark', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ remark: card.remark })
    })
    const json = await res.json()
    if (json.code !== 0) {
      toast('保存备注失败', 'error')
    }
  } catch {
    toast('保存备注失败', 'error')
  }
}
</script>

<style scoped>
.admin { background:#fff; padding:20px; border-radius:12px; box-shadow:0 2px 12px rgba(0,0,0,0.08); }

/* 工具栏 */
.toolbar { display:flex; gap:10px; margin-bottom:20px; flex-wrap:wrap; align-items:center; }
.btn-del { background:#dc3545; color:#fff; border:none; padding:8px 16px; border-radius:8px; font-size:14px; }
.btn-del:hover:not(:disabled) { background:#c82333; }
.btn-del:disabled { background:#aaa; cursor:not-allowed; }

.btn-export { background:#2980b9; color:#fff; border:none; padding:8px 16px; border-radius:8px; font-size:14px; }
.btn-export:hover:not(:disabled) { background:#2980b9; }

/* 分页大小控制样式 */
.page-size-control {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.page-size-control label {
  font-weight: 500;
  color: #333;
}

.page-size-control input, .page-size-control select {
  padding: 6px 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
}

.page-size-control input {
  width: 60px;
  text-align: center;
}

/* 筛选样式 */
.filter-section {
  margin-bottom: 20px;
  padding: 15px;
  background: #f8f9fa;
  border-radius: 8px;
}

.filter-row {
  display: flex;
  gap: 15px;
  flex-wrap: wrap;
  align-items: center;
}

.filter-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-item label {
  font-weight: 500;
  color: #555;
  font-size: 14px;
}

.filter-item input,
.filter-item select {
  padding: 6px 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
}

.btn-clear {
  background: #6c757d;
  color: #fff;
  border: none;
  padding: 6px 12px;
  border-radius: 4px;
  font-size: 14px;
}

.btn-clear:hover {
  background: #5a6268;
}

/* 表格 */
table { width:100%; border-collapse:collapse; margin-bottom:20px; font-size:14px; }
th, td { padding:12px; border-bottom:1px solid #eee; text-align:left; vertical-align:middle; }
th { background:#f8f9fa; font-weight:600; color:#555; }
tr:hover { background:#f8f9fa; }

.select-all { display:flex; align-items:center; gap:6px; cursor:pointer; }
.query-link { margin-left:8px; color:#007bff; font-size:12px; text-decoration:none; }
.query-link:hover { text-decoration:underline; }
.link { color:#007bff; text-decoration:none; }
.link:hover { text-decoration:underline; }

/* 随机码样式 */
.random-code {
  font-family: monospace;
  font-weight: 600;
  color: #007bff;
  background: #f0f7ff;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 13px;
  letter-spacing: 1px;
}

.time-cell { display:flex; flex-direction:column; gap:2px; font-size:13px; }
.time-cell .date { color:#333; }
.time-cell .time { color:#999; font-size:12px; }

/* 备注输入框 */
.remark-input {
  width: 120px;
  padding: 4px 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 13px;
  background: transparent;
}
.remark-input:focus {
  outline: none;
  border-color: #007bff;
  background: #fff;
}

/* 分页 */
.pagination { display:flex; justify-content:center; align-items:center; gap:10px; margin-top:20px; }
.pagination button {
  padding:8px 14px; border:1px solid #ddd; background:#fff; border-radius:6px; font-size:14px;
}
.pagination button:hover:not(:disabled) { background:#f0f0f0; }
.pagination button:disabled { opacity:0.5; cursor:not-allowed; }
.page-info { font-size:14px; color:#666; white-space:nowrap; }

/* 添加区域 */
.add-section { margin-top:30px; padding:20px; background:#f8f9fa; border-radius:12px; }
.add-section h3 { margin-bottom:12px; color:#333; font-size:18px; }
.add-section p { margin-bottom:10px; color:#666; font-size:14px; }
.add-section textarea {
  width:100%; height:120px; padding:12px; border:1px solid #ddd; border-radius:8px; font-size:14px;
  font-family:monospace; resize:vertical;
}

.add-options {
  margin: 12px 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.option-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.option-row label {
  font-weight: 500;
  color: #555;
  font-size: 14px;
  min-width: 80px;
}

.multiplier-input {
  padding: 6px 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
  width: 60px;
  text-align: center;
}

.sort-select {
  padding: 6px 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
  min-width: 200px;
}

.hint {
  color: #888;
  font-size: 13px;
}

.inline-remark-input {
  padding: 6px 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 13px;
  width: 180px;
}

.inline-remark-input:focus {
  outline: none;
  border-color: #007bff;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 14px;
  color: #555;
}

.checkbox-label input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
}

.validate { margin:12px 0; font-size:14px; }
.validate .ok { color:#28a745; font-weight:500; }
.validate .bad { color:#dc3545; margin-left:12px; }
.validate ul { margin:8px 0 0 20px; color:#dc3545; font-size:13px; }

.btn-add {
  background:#28a745; color:#fff; border:none; padding:10px 20px; border-radius:8px; font-size:15px;
}
.btn-add:hover:not(:disabled) { background:#218838; }
.btn-add:disabled { background:#aaa; cursor:not-allowed; }

.msg-success { color:#28a745; margin-top:10px; }
.msg-error { color:#dc3545; margin-top:10px; }

.page-info {
  font-size: 14px;
  color: #666;
  white-space: nowrap;
}

@media (max-width: 600px) {
  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }
  
  .page-size-control {
    margin-left: 0;
    justify-content: center;
  }
  
  .filter-row {
    flex-direction: column;
    align-items: stretch;
  }
  
  .filter-item {
    justify-content: space-between;
  }
  
  .pagination {
    gap: 5px;
  }
  
  .pagination button {
    padding: 6px 10px;
    font-size: 13px;
  }
  
  .page-info {
    font-size: 13px;
  }
}
</style>