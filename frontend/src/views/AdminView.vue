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
            <button @click="createBackup" :disabled="loading" class="btn-primary">
              {{ loading ? '创建中...' : '📥 创建新备份' }}
            </button>
            <button @click="loadBackups" :disabled="loading" class="btn-secondary">
              🔄 刷新列表
            </button>
          </div>
          
          <!-- 备份列表 -->
          <div class="backup-list">
            <div v-if="backups.length === 0" class="empty">
              {{ loading ? '加载中...' : '暂无备份文件' }}
            </div>
            <div v-for="backup in backups" :key="backup.name" class="backup-item">
              <div class="backup-info">
                <span class="backup-name">{{ backup.name }}</span>
                <span class="backup-meta">
                  {{ formatSize(backup.size) }} · {{ backup.createdAt }}
                </span>
              </div>
              <div class="backup-actions">
                <button @click="downloadBackup(backup.name)" class="btn-icon" title="下载">⬇️</button>
                <button @click="restoreBackup(backup.name)" class="btn-icon" title="恢复">↩️</button>
                <button @click="deleteBackup(backup.name)" class="btn-icon btn-danger" title="删除">🗑️</button>
              </div>
            </div>
          </div>
          
          <!-- 提示信息 -->
          <div class="backup-tips">
            <p>💡 提示：</p>
            <ul>
              <li>备份文件包含所有卡密数据，请妥善保管</li>
              <li>恢复备份会覆盖当前数据，操作前会自动备份当前状态</li>
              <li>建议定期创建备份，防止数据丢失</li>
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
import { onMounted, ref } from 'vue'
import { useToast } from '../composables/useToast'

const router = useRouter()
const toast = useToast()

// 备份管理
const showBackupModal = ref(false)
const loading = ref(false)
const backups = ref<any[]>([])

interface Backup {
  name: string
  size: number
  createdAt: string
}

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

// 获取 API 基础地址
function getBaseURL() {
  return import.meta.env.VITE_API_BASE_URL || ''
}

// 加载备份列表
async function loadBackups() {
  loading.value = true
  try {
    const res = await fetch(`${getBaseURL()}/api/admin/backups`)
    const json = await res.json()
    if (json.code === 0) {
      backups.value = json.data || []
    } else {
      toast(json.message || '加载失败', 'error')
    }
  } catch (err) {
    toast('加载备份列表失败', 'error')
  } finally {
    loading.value = false
  }
}

// 创建备份
async function createBackup() {
  loading.value = true
  try {
    const res = await fetch(`${getBaseURL()}/api/admin/backup`, { method: 'POST' })
    const json = await res.json()
    if (json.code === 0) {
      toast('备份创建成功', 'success')
      loadBackups()
    } else {
      toast(json.message || '创建失败', 'error')
    }
  } catch (err) {
    toast('创建备份失败', 'error')
  } finally {
    loading.value = false
  }
}

// 下载备份
async function downloadBackup(name: string) {
  try {
    const res = await fetch(`${getBaseURL()}/api/admin/backup/download?name=${encodeURIComponent(name)}`)
    if (!res.ok) {
      // 如果下载接口不存在，使用备用方案：直接通过后端文件路径下载
      const link = document.createElement('a')
      link.href = `${getBaseURL()}/backups/${name}`
      link.download = name
      link.click()
      return
    }
    
    const blob = await res.blob()
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = name
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
    toast('下载已开始', 'success')
  } catch (err) {
    toast('下载失败', 'error')
  }
}

// 恢复备份
function restoreBackup(name: string) {
  confirmModal.value = {
    show: true,
    title: '确认恢复备份',
    message: `确定要恢复备份 "${name}" 吗？\n当前数据会被自动备份，可后续手动恢复。`,
    onConfirm: async () => {
      confirmModal.value.show = false
      loading.value = true
      try {
        const res = await fetch(`${getBaseURL()}/api/admin/restore`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name })
        })
        const json = await res.json()
        if (json.code === 0) {
          toast('恢复成功，页面即将刷新', 'success')
          setTimeout(() => window.location.reload(), 1500)
        } else {
          toast(json.message || '恢复失败', 'error')
        }
      } catch (err) {
        toast('恢复备份失败', 'error')
      } finally {
        loading.value = false
      }
    }
  }
}

// 删除备份
function deleteBackup(name: string) {
  confirmModal.value = {
    show: true,
    title: '确认删除备份',
    message: `确定要删除备份 "${name}" 吗？\n此操作不可恢复。`,
    onConfirm: async () => {
      confirmModal.value.show = false
      loading.value = true
      try {
        const res = await fetch(`${getBaseURL()}/api/admin/backup/${encodeURIComponent(name)}`, {
          method: 'DELETE'
        })
        const json = await res.json()
        if (json.code === 0) {
          toast('删除成功', 'success')
          loadBackups()
        } else {
          toast(json.message || '删除失败', 'error')
        }
      } catch (err) {
        toast('删除备份失败', 'error')
      } finally {
        loading.value = false
      }
    }
  }
}

// 格式化文件大小
function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

onMounted(() => {
  // 进入页面时若无 token 则重定向到登录
  const token = localStorage.getItem('admin_token')
  if (!token) {
    router.replace('/login')
  }
})

// 打开备份弹窗时加载列表
import { watch } from 'vue'
watch(showBackupModal, (val) => {
  if (val) loadBackups()
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