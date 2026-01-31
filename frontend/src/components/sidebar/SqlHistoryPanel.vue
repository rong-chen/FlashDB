<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

const emit = defineEmits(['copy-sql'])

// State
const sqlHistory = ref([])
const showHistory = ref(false)

// 从 localStorage 加载历史
function loadSqlHistory() {
  try {
    const saved = localStorage.getItem('flashdb_sql_history')
    if (saved) {
      const history = JSON.parse(saved)
      // 自动清理7天前的记录
      const sevenDaysAgo = Date.now() - 7 * 24 * 60 * 60 * 1000
      sqlHistory.value = history.filter(item => new Date(item.timestamp).getTime() > sevenDaysAgo)
      // 如果有清理，保存更新后的数据
      if (sqlHistory.value.length !== history.length) {
        localStorage.setItem('flashdb_sql_history', JSON.stringify(sqlHistory.value))
      }
    }
  } catch (e) {
    console.error('Failed to load SQL history:', e)
  }
}

// 添加到历史
function addToHistory(sql, success, executionTime) {
  const entry = {
    id: Date.now(),
    sql: sql.trim(),
    success,
    executionTime,
    timestamp: new Date().toISOString()
  }
  sqlHistory.value.unshift(entry)
  // 最多保留 100 条
  if (sqlHistory.value.length > 100) {
    sqlHistory.value = sqlHistory.value.slice(0, 100)
  }
  localStorage.setItem('flashdb_sql_history', JSON.stringify(sqlHistory.value))
}

// 清空历史
function clearHistory() {
  sqlHistory.value = []
  localStorage.removeItem('flashdb_sql_history')
  ElMessage.success('历史已清空')
}

// 复制历史 SQL
async function copyHistorySql(sql) {
  await navigator.clipboard.writeText(sql)
  emit('copy-sql', sql)
  ElMessage.success('已复制 SQL')
}

// Expose for parent
defineExpose({
  addToHistory,
  loadSqlHistory
})

onMounted(() => {
  loadSqlHistory()
})
</script>

<template>
  <div class="history-section">
    <div class="history-header" @click="showHistory = !showHistory">
      <span>SQL 历史</span>
      <span class="history-toggle">{{ showHistory ? '▼' : '▶' }}</span>
    </div>
    <div v-show="showHistory" class="history-content">
      <div v-if="sqlHistory.length === 0" class="empty-history">
        暂无执行记录
      </div>
      <div v-else>
        <div class="history-actions">
          <el-button size="small" type="danger" text @click="clearHistory">清空</el-button>
        </div>
        <div class="history-list">
          <div 
            v-for="item in sqlHistory.slice(0, 20)" 
            :key="item.id" 
            class="history-item"
            :class="{ success: item.success, failed: !item.success }"
            @click="copyHistorySql(item.sql)"
          >
            <div class="history-sql">{{ item.sql.substring(0, 100) }}{{ item.sql.length > 100 ? '...' : '' }}</div>
            <div class="history-meta">
              <span class="history-time">{{ new Date(item.timestamp).toLocaleTimeString() }}</span>
              <span v-if="item.executionTime" class="history-duration">{{ item.executionTime }}ms</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.history-section {
  border-top: 1px solid #222;
  margin-top: auto;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  cursor: pointer;
  font-size: 12px;
  color: #888;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.history-header:hover {
  background: #1a1a1a;
}

.history-toggle {
  font-size: 10px;
}

.history-content {
  max-height: 300px;
  overflow-y: auto;
}

.empty-history {
  padding: 20px;
  text-align: center;
  color: #666;
  font-size: 13px;
}

.history-actions {
  padding: 4px 12px;
  text-align: right;
}

.history-list {
  padding: 0 8px 8px;
}

.history-item {
  padding: 8px 10px;
  margin-bottom: 4px;
  background: #1a1a1a;
  border-radius: 6px;
  cursor: pointer;
  border-left: 3px solid #444;
  transition: all 0.15s;
}

.history-item:hover {
  background: #252525;
}

.history-item.success {
  border-left-color: #67c23a;
}

.history-item.failed {
  border-left-color: #f56c6c;
}

.history-sql {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: #ccc;
  word-break: break-all;
}

.history-meta {
  display: flex;
  gap: 12px;
  margin-top: 4px;
  font-size: 11px;
  color: #666;
}
</style>
