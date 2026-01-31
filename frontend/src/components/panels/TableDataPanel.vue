<script setup>
import { defineProps, defineEmits } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowDown } from '@element-plus/icons-vue'
import { SaveFile } from '../../wailsjs/go/services/FileService'
import VirtualDataGrid from './VirtualDataGrid.vue'

const props = defineProps({
  tableData: {
    type: Object,
    default: () => ({ columns: null, rows: [] })
  },
  tableName: String,
  loading: Boolean,
  editable: Boolean,
  primaryKey: String
})

const emit = defineEmits(['save'])

// ========== 数据导出功能 ==========
function handleExport(format) {
  if (!props.tableData.columns || !props.tableData.rows?.length) {
    ElMessage.warning('没有数据可导出')
    return
  }
  
  const columns = props.tableData.columns.map(c => c.name)
  const rows = props.tableData.rows
  const tableName = props.tableName || 'data'
  
  let content = ''
  let filename = ''
  let mimeType = ''
  
  switch (format) {
    case 'csv':
      content = exportToCSV(columns, rows)
      filename = `${tableName}.csv`
      mimeType = 'text/csv;charset=utf-8'
      break
    case 'json':
      content = exportToJSON(columns, rows)
      filename = `${tableName}.json`
      mimeType = 'application/json;charset=utf-8'
      break
    case 'markdown':
      content = exportToMarkdown(columns, rows)
      filename = `${tableName}.md`
      mimeType = 'text/markdown;charset=utf-8'
      break
  }
  
  downloadFile(content, filename, mimeType)
}

function exportToCSV(columns, rows) {
  const header = columns.join(',')
  const lines = rows.map(row => 
    row.map(cell => {
      if (cell === null) return ''
      const str = String(cell)
      if (str.includes(',') || str.includes('\n') || str.includes('"')) {
        return `"${str.replace(/"/g, '""')}"`
      }
      return str
    }).join(',')
  )
  return [header, ...lines].join('\n')
}

function exportToJSON(columns, rows) {
  const data = rows.map(row => {
    const obj = {}
    columns.forEach((col, idx) => {
      obj[col] = row[idx]
    })
    return obj
  })
  return JSON.stringify(data, null, 2)
}

function exportToMarkdown(columns, rows) {
  const header = '| ' + columns.join(' | ') + ' |'
  const separator = '| ' + columns.map(() => '---').join(' | ') + ' |'
  const lines = rows.map(row => 
    '| ' + row.map(cell => cell === null ? 'NULL' : String(cell)).join(' | ') + ' |'
  )
  return [header, separator, ...lines].join('\n')
}

async function downloadFile(content, filename, mimeType) {
  const ext = filename.split('.').pop()
  const filterPattern = `*.${ext}`
  const filterName = ext.toUpperCase() + ' 文件'
  
  const res = await SaveFile(filename, content, filterPattern, filterName)
  if (!res.success) {
    if (res.error !== '用户取消') {
      throw new Error(res.error)
    }
    return false
  }
  return true
}

function handleSave(changes) {
  emit('save', changes)
}
</script>

<template>
  <div class="table-data-panel">
    <div v-if="!tableData.columns && !tableData.rows?.length" class="empty-state">
      <p>选择表查看数据</p>
    </div>
    <template v-else>
      <div class="panel-header">
        <div class="header-left">
          <span class="panel-title">{{ tableName || 'SQL 查询结果' }}</span>
          <span class="data-count" v-if="tableData.rows">{{ tableData.rows.length }} 行</span>
        </div>
        <div class="header-actions" v-if="tableData.rows?.length">
          <el-dropdown trigger="click" @command="handleExport">
            <el-button size="small" type="default">
              导出 <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="csv">CSV</el-dropdown-item>
                <el-dropdown-item command="json">JSON</el-dropdown-item>
                <el-dropdown-item command="markdown">Markdown</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
      <VirtualDataGrid
        :columns="tableData.columns || []"
        :rows="tableData.rows || []"
        :loading="loading"
        :editable="editable"
        :primary-key="primaryKey"
        @save="handleSave"
      />
    </template>
  </div>
</template>

<style scoped>
.table-data-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.empty-state {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #666;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  background: #1a1a1a;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.panel-title {
  font-weight: 500;
  font-size: 14px;
  color: #fff;
}

.data-count {
  font-size: 12px;
  color: #888;
  padding: 2px 8px;
  background: #2a2a2a;
  border-radius: 4px;
}

.header-actions {
  display: flex;
  gap: 8px;
}
</style>
