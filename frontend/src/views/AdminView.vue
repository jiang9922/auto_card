<template>
  <div class="admin-page">
    <div class="header">
      <h2>卡密管理后台</h2>
      <div class="header-actions">
        <button @click="showBackupModal = true" class="btn-backup">📦 备份管理</button>
        <button @click="logout" class="btn-logout">退出登录</button>
      </div>
    </div>
    <AdminTable />
    
    <!-- 备份管理弹窗 -->
    <div v-if="showBackupModal" class="modal-overlay" @click="showBackupModal = false">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>📦 数据库备份管理</h3>
          <button @click="showBackupModal = false" class="btn-close">×</button>
        </div>
        
        <div class="modal-body">
          <!-- 操作按钮 -->
          <div class="backup-actions">
            <button @click="exportFullCSV" :disabled="loading" class="btn-success">
              {{ loading ? '导出中...' : '📄 导出完整数据(CSV)' }}
            </button>
          </div>
          
          <!-- CSV 导入区域 -->
          <div class="import-section">
            <h4>📤 从 CSV 导入数据</h4>
            <div class="import-form">
              <input 
                type="file" 
                ref="csvFileInput" 
                accept=".csv" 
                style="display: none"
                @change="handleCSVUpload"
              />
              <button @click="triggerCSVUpload" :disabled="loading" class="btn-import">
                选择 CSV 文件
              </button>
              <span v-if="selectedFileName" class="file-name">{{ selectedFileName }}</span>
              <button 
                v-if="selectedFileName" 
                @click="confirmImport" 
                :disabled="loading" 
                class="btn-primary btn-import-confirm"
              >
                {{ loading ? '导入中...' : '开始导入' }}
              </button>
            </div>
            <div class="import-tips">
              <p>CSV 格式要求：</p>
              <ul>
                <li>第一行为表头：序号、卡号、原链接、查询链接、查询Token、状态、备注、创建时间</li>
                <li>数据从第二行开始</li>
                <li>已存在的查询链接会自动跳过</li>
              </ul>
            </div>
          </div>
          
          <!-- 提示信息 -->
          <div class="backup-tips">
            <p>💡 提示：</p>
            <ul>
              <li>导出 CSV 可保存到本地电脑，用 Excel 查看和编辑</li>
              <li>导入 CSV 可批量恢复数据，已存在的查询链接会自动跳过</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 确认对话框 -->
    <div v-if="confirmModal.show" class="modal-overlay">
      <div class="modal-content confirm-modal">
        <h4>{{ confirmModal.title }}</h4>
        <p>{{ confirmModal.message }}</p>
        <div class="confirm-actions">
          <button @click="confirmModal.show = false" class="btn-secondary">取消</button>
          <button @click="confirmModal.onConfirm" class="btn-primary">确认</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 管理页：承载 AdminTable，并提供登录校验与退出登录
import AdminTable from '../components/AdminTable.vue'
import { useRouter } from 'vue-router'
import { onMounted, ref, watch, type Ref } from 'vue'
import { useToast } from '../composables/useToast'

const router = useRouter()
const toast = useToast()

// 备份管理
const showBackupModal = ref(false)
const loading = ref(false)

interface ConfirmModal {
  show: boolean
  title: string
  message: string
  onConfirm: () => void
}

const confirmModal = ref<ConfirmModal>({
  show: false,
  title: '',
  message: '',
  onConfirm: () => {}
})

// CSV 导入相关
const csvFileInput = ref<HTMLInputElement | null>(null)
const selectedFileName = ref('')
const selectedFile: Ref<File | null> = ref(null)

// 获取 API 基础地址
function getBaseURL() {
  return import.meta.env.VITE_API_BASE_URL || ''
}

// 导出完整 CSV
async function exportFullCSV() {
  loading.value = true
  try {
    const res = await fetch(`${getBaseURL()}/api/admin/export/full`)
    if (!res.ok) {
      toast('导出失败', 'error')
      return
    }
    
    const blob = await res.blob()
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `cards_export_${new Date().toISOString().slice(0, 10).replace(/-/g, '')}.csv`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
    toast('CSV 导出成功', 'success')
  } catch (err) {
    toast('导出 CSV 失败', 'error')
  } finally {
    loading.value = false
  }
}

// 处理 CSV 文件选择
function handleCSVUpload(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (file) {
    selectedFile.value = file
    selectedFileName.value = file.name
  }
}

// 触发 CSV 文件选择
function triggerCSVUpload() {
  csvFileInput.value?.click()
}

// 确认导入
function confirmImport() {
  if (!selectedFile.value) {
    toast('请先选择 CSV 文件', 'error')
    return
  }
  
  confirmModal.value = {
    show: true,
    title: '确认导入数据',
    message: `确定要导入文件 "${selectedFileName.value}" 吗？\n已存在的查询链接会被自动跳过。`,
    onConfirm: async () => {
      confirmModal.value.show = false
      await importCSV()
    }
  }
}

// 执行 CSV 导入
async function importCSV() {
  if (!selectedFile.value) return
  
  loading.value = true
  try {
    const formData = new FormData()
    formData.append('file', selectedFile.value)
    
    const res = await fetch(`${getBaseURL()}/api/admin/import`, {
      method: 'POST',
      body: formData
    })
    const json = await res.json()
    if (json.code === 0) {
      toast(json.message, 'success')
      // 清空选择
      selectedFile.value = null
      selectedFileName.value = ''
      if (csvFileInput.value) {
        csvFileInput.value.value = ''
      }
    } else {
      toast(json.message || '导入失败', 'error')
    }
  } catch (err) {
    toast('导入 CSV 失败', 'error')
  } finally {
    loading.value = false
  }
}

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

.header-actions { display: flex; gap: 12px; align-items: center; }

.btn-backup {
  background: #28a745;
  color: #fff;
  border: none;
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
}
.btn-backup:hover { background: #218838; }

.btn-logout {
  background: #dc3545; color: #fff; border: none; padding: 8px 16px;
  border-radius: 8px; font-size: 14px; cursor: pointer;
}
.btn-logout:hover { background: #c82333; }

/* 弹窗样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background: #fff;
  border-radius: 12px;
  width: 90%;
  max-width: 600px;
  max-height: 80vh;
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #eee;
}

.modal-header h3 { margin: 0; color: #333; }

.btn-close {
  background: none;
  border: none;
  font-size: 24px;
  color: #999;
  cursor: pointer;
}
.btn-close:hover { color: #333; }

.modal-body {
  padding: 20px;
  overflow-y: auto;
  max-height: calc(80vh - 70px);
}

/* 备份操作按钮 */
.backup-actions {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.btn-primary {
  background: #007bff;
  color: #fff;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
}
.btn-primary:hover:not(:disabled) { background: #0056b3; }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-secondary {
  background: #6c757d;
  color: #fff;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
}
.btn-secondary:hover:not(:disabled) { background: #545b62; }
.btn-secondary:disabled { opacity: 0.6; cursor: not-allowed; }

/* 备份列表 */
.backup-list {
  border: 1px solid #eee;
  border-radius: 8px;
  max-height: 300px;
  overflow-y: auto;
}

.empty {
  padding: 40px;
  text-align: center;
  color: #999;
}

.backup-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #eee;
}
.backup-item:last-child { border-bottom: none; }
.backup-item:hover { background: #f8f9fa; }

.backup-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.backup-name {
  font-weight: 500;
  color: #333;
  font-family: monospace;
  font-size: 13px;
}

.backup-meta {
  font-size: 12px;
  color: #999;
}

.backup-item .backup-actions {
  display: flex;
  gap: 8px;
  margin: 0;
}

.btn-icon {
  background: #f8f9fa;
  border: 1px solid #ddd;
  padding: 6px 10px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}
.btn-icon:hover { background: #e9ecef; }

.btn-danger:hover { background: #dc3545; color: #fff; border-color: #dc3545; }

/* CSV 导入区域 */
.import-section {
  margin: 20px 0;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e9ecef;
}

.import-section h4 {
  margin: 0 0 12px 0;
  color: #333;
  font-size: 14px;
}

.import-form {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.btn-success {
  background: #28a745;
  color: #fff;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
}
.btn-success:hover:not(:disabled) { background: #218838; }
.btn-success:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-import {
  background: #17a2b8;
  color: #fff;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.2s;
}
.btn-import:hover:not(:disabled) { background: #138496; }
.btn-import:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-import-confirm {
  padding: 8px 16px;
  font-size: 13px;
}

.file-name {
  font-size: 13px;
  color: #666;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.import-tips {
  margin-top: 12px;
  padding: 10px 12px;
  background: #fff;
  border-radius: 6px;
  border-left: 3px solid #ffc107;
}

.import-tips p {
  margin: 0 0 6px 0;
  font-size: 12px;
  font-weight: 500;
  color: #333;
}

.import-tips ul {
  margin: 0;
  padding-left: 16px;
  font-size: 11px;
  color: #666;
}

.import-tips li {
  margin-bottom: 2px;
}

/* 提示信息 */
.backup-tips {
  margin-top: 20px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
  border-left: 4px solid #17a2b8;
}

.backup-tips p {
  margin: 0 0 8px 0;
  font-weight: 500;
  color: #333;
}

.backup-tips ul {
  margin: 0;
  padding-left: 20px;
  color: #666;
  font-size: 13px;
}

.backup-tips li { margin-bottom: 4px; }

/* 确认对话框 */
.confirm-modal {
  padding: 24px;
  text-align: center;
}

.confirm-modal h4 {
  margin: 0 0 12px 0;
  color: #333;
}

.confirm-modal p {
  color: #666;
  margin-bottom: 20px;
  white-space: pre-line;
}

.confirm-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}
</style>