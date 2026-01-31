<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { useVirtualizer } from '@tanstack/vue-virtual'

const props = defineProps({
  columns: {
    type: Array,
    default: () => []
  },
  rows: {
    type: Array,
    default: () => []
  },
  loading: {
    type: Boolean,
    default: false
  },
  editable: {
    type: Boolean,
    default: false
  },
  primaryKey: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['cell-click', 'save', 'row-change'])

// ========== Edit State ==========
const editingCell = ref(null)       // { rowIdx, colIdx, x, y, width, height }
const editingValue = ref('')
const popupRef = ref(null)

// Change buffer: tracks all modifications
const changeBuffer = ref({
  modified: new Map(),  // rowIdx -> { colIdx -> { original, current } }
  inserted: [],         // Array of new row data
  deleted: new Set()    // Set of rowIdx to delete
})

// ========== Virtualizer ==========
const parentRef = ref(null)
const sortColumn = ref(null)
const sortDirection = ref('asc')

// Local copy of rows for editing
const localRows = ref([])
watch(() => props.rows, (newRows) => {
  localRows.value = newRows.map(row => [...row])
  // Clear change buffer when data reloads
  changeBuffer.value = { modified: new Map(), inserted: [], deleted: new Set() }
}, { immediate: true, deep: true })

// Sorted rows
const sortedRows = computed(() => {
  const rows = [...localRows.value, ...changeBuffer.value.inserted]
  if (!sortColumn.value) return rows
  
  const colIdx = props.columns.findIndex(c => c.name === sortColumn.value)
  if (colIdx < 0) return rows
  
  return [...rows].sort((a, b) => {
    const valA = a[colIdx]
    const valB = b[colIdx]
    
    if (valA === null) return sortDirection.value === 'asc' ? -1 : 1
    if (valB === null) return sortDirection.value === 'asc' ? 1 : -1
    
    if (typeof valA === 'number' && typeof valB === 'number') {
      return sortDirection.value === 'asc' ? valA - valB : valB - valA
    }
    
    const strA = String(valA).toLowerCase()
    const strB = String(valB).toLowerCase()
    return sortDirection.value === 'asc' 
      ? strA.localeCompare(strB) 
      : strB.localeCompare(strA)
  })
})

// Row virtualizer
const rowVirtualizer = useVirtualizer(
  computed(() => ({
    count: sortedRows.value.length,
    getScrollElement: () => parentRef.value,
    estimateSize: () => 36,
    overscan: 5
  }))
)

const virtualRows = computed(() => rowVirtualizer.value.getVirtualItems())
const totalSize = computed(() => rowVirtualizer.value.getTotalSize())

// Has changes?
const hasChanges = computed(() => {
  return changeBuffer.value.modified.size > 0 || 
         changeBuffer.value.inserted.length > 0 || 
         changeBuffer.value.deleted.size > 0
})

// ========== Column Width ==========
const columnWidths = ref({})

// 计算所有列的总宽度
const totalColumnsWidth = computed(() => {
  return props.columns.reduce((sum, col) => sum + getColumnWidth(col.name), 0) + 28 // +28 for row actions
})

function getColumnWidth(colName) {
  return columnWidths.value[colName] || 150
}

let resizing = null

function startResize(e, colName) {
  e.preventDefault()
  resizing = { colName, startX: e.clientX, startWidth: getColumnWidth(colName) }
  document.addEventListener('mousemove', handleResize)
  document.addEventListener('mouseup', stopResize)
}

function handleResize(e) {
  if (!resizing) return
  const diff = e.clientX - resizing.startX
  const newWidth = Math.max(60, resizing.startWidth + diff)
  columnWidths.value = { ...columnWidths.value, [resizing.colName]: newWidth }
}

function stopResize() {
  resizing = null
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
}

// ========== Sort ==========
function handleSort(colName) {
  if (sortColumn.value === colName) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortColumn.value = colName
    sortDirection.value = 'asc'
  }
}

// ========== Cell Editing ==========
function handleCellClick(e, rowIdx, colIdx, value) {
  if (changeBuffer.value.deleted.has(rowIdx)) return
  
  const canEdit = props.editable
  
  editingCell.value = { 
    rowIdx, 
    colIdx,
    isEditing: canEdit,
    originalValue: value  // 保存原始值
  }
  editingValue.value = value === null ? '' : String(value)
  
  nextTick(() => {
    const inputs = document.querySelectorAll('.cell-input-inline')
    if (inputs.length > 0) {
      inputs[0].focus()
      inputs[0].select()
    }
  })
  
  emit('cell-click', { rowIdx, colIdx, value })
}

function closeCellEdit() {
  editingCell.value = null
  editingValue.value = ''
}

function saveEditedValue() {
  if (!editingCell.value?.isEditing) {
    closeCellEdit()
    return
  }
  
  const { rowIdx, colIdx, originalValue } = editingCell.value
  const inputValue = editingValue.value
  // 只有原来是 null 且输入为空时才保持 null，否则保持输入值
  const newValue = (originalValue === null && inputValue === '') ? null : inputValue
  const localRowCount = localRows.value.length
  
  // 比较原始值和新值是否相同
  const isSame = (originalValue === null && newValue === null) ||
                 (originalValue === newValue) ||
                 (String(originalValue ?? '') === String(newValue ?? ''))
  
  if (isSame) {
    // 值没变，直接关闭
    closeCellEdit()
    return
  }
  
  // 判断是否为新增行
  if (rowIdx >= localRowCount) {
    // 新增行：更新 changeBuffer.inserted
    const insertIdx = rowIdx - localRowCount
    const currentRow = changeBuffer.value.inserted[insertIdx]
    if (currentRow) {
      currentRow[colIdx] = newValue
    }
  } else {
    // 现有行：更新 localRows 并跟踪修改
    if (localRows.value[rowIdx]) {
      localRows.value[rowIdx][colIdx] = newValue
    }
    
    // Track modification
    if (!changeBuffer.value.modified.has(rowIdx)) {
      changeBuffer.value.modified.set(rowIdx, new Map())
    }
    changeBuffer.value.modified.get(rowIdx).set(colIdx, {
      original: originalValue,
      current: newValue
    })
    
    emit('row-change', { rowIdx, colIdx, original: originalValue, current: newValue })
  }
  
  closeCellEdit()
}

function handleInputKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    saveEditedValue()
  } else if (e.key === 'Escape') {
    closeCellEdit()
  }
}

// ========== Row Operations ==========
function addNewRow() {
  const newRow = props.columns.map(() => null)
  changeBuffer.value.inserted.push(newRow)
}

function markRowDeleted(rowIdx) {
  if (rowIdx >= localRows.value.length) {
    // It's an inserted row, just remove it
    const insertIdx = rowIdx - localRows.value.length
    changeBuffer.value.inserted.splice(insertIdx, 1)
  } else {
    changeBuffer.value.deleted.add(rowIdx)
  }
}

function unmarkRowDeleted(rowIdx) {
  changeBuffer.value.deleted.delete(rowIdx)
}

// ========== Save & Rollback ==========
function getChanges() {
  const pkIdx = props.columns.findIndex(c => c.name === props.primaryKey)
  
  const updates = []
  changeBuffer.value.modified.forEach((colChanges, rowIdx) => {
    if (changeBuffer.value.deleted.has(rowIdx)) return
    
    const changes = {}
    const primaryKey = {}
    
    if (pkIdx >= 0 && props.rows[rowIdx]) {
      primaryKey[props.primaryKey] = props.rows[rowIdx][pkIdx]
    }
    
    colChanges.forEach((change, colIdx) => {
      const colName = props.columns[colIdx]?.name
      if (colName) {
        changes[colName] = change.current
      }
    })
    
    if (Object.keys(changes).length > 0 && Object.keys(primaryKey).length > 0) {
      updates.push({ primaryKey, changes })
    }
  })
  
  const inserts = changeBuffer.value.inserted.map(row => {
    const values = {}
    props.columns.forEach((col, idx) => {
      if (row[idx] !== null) {
        values[col.name] = row[idx]
      }
    })
    return { values }
  })
  
  const deletes = []
  changeBuffer.value.deleted.forEach(rowIdx => {
    const primaryKey = {}
    if (pkIdx >= 0 && props.rows[rowIdx]) {
      primaryKey[props.primaryKey] = props.rows[rowIdx][pkIdx]
    }
    if (Object.keys(primaryKey).length > 0) {
      deletes.push({ primaryKey })
    }
  })
  
  return { updates, inserts, deletes }
}

function handleSave() {
  emit('save', getChanges())
}

function handleRollback() {
  localRows.value = props.rows.map(row => [...row])
  changeBuffer.value = { modified: new Map(), inserted: [], deleted: new Set() }
}

// ========== Row Styling ==========
function getRowClass(rowIdx) {
  if (changeBuffer.value.deleted.has(rowIdx)) return 'row-deleted'
  if (rowIdx >= localRows.value.length) return 'row-inserted'
  if (changeBuffer.value.modified.has(rowIdx)) return 'row-modified'
  return ''
}

function getCellClass(rowIdx, colIdx) {
  if (changeBuffer.value.modified.has(rowIdx)) {
    const colChanges = changeBuffer.value.modified.get(rowIdx)
    if (colChanges.has(colIdx)) return 'cell-modified'
  }
  return ''
}

// Close popup on outside click
function handleContainerClick(e) {
  if (editingCell.value && !e.target.closest('.cell-input')) {
    if (editingCell.value.isEditing) {
      saveEditedValue()
    } else {
      closeCellEdit()
    }
  }
}

// Expose for parent
defineExpose({ hasChanges, getChanges, handleSave, handleRollback, addNewRow })
</script>

<template>
  <div class="virtual-grid-container" @click="handleContainerClick">
    <!-- Toolbar -->
    <div v-if="editable" class="grid-toolbar">
      <button class="toolbar-btn" @click.stop="addNewRow" title="新增行">
        <span>+ 新增</span>
      </button>
      <div class="toolbar-spacer"></div>
      <template v-if="hasChanges">
        <button class="toolbar-btn btn-save" @click.stop="handleSave" title="保存">
          保存
        </button>
        <button class="toolbar-btn btn-rollback" @click.stop="handleRollback" title="回滚">
          回滚
        </button>
      </template>
    </div>
    
    <!-- Loading -->
    <div v-if="loading" class="grid-empty">
      <p>加载中...</p>
    </div>
    
    <!-- Empty state -->
    <div v-else-if="!rows.length && !changeBuffer.inserted.length" class="grid-empty">
      <p>暂无数据</p>
    </div>
    
    <!-- Virtual scrolling container -->
    <div v-else ref="parentRef" class="grid-scroll-container">
      <!-- Sticky Header -->
      <div class="grid-header" :style="{ minWidth: totalColumnsWidth + 'px' }">
        <div 
          v-for="col in columns" 
          :key="col.name"
          class="header-cell"
          :style="{ width: getColumnWidth(col.name) + 'px' }"
          @click.stop="handleSort(col.name)"
        >
          <span class="header-text">{{ col.name }}</span>
          <span v-if="sortColumn === col.name" class="sort-icon">
            {{ sortDirection === 'asc' ? '↑' : '↓' }}
          </span>
          <div class="resize-handle" @mousedown="startResize($event, col.name)" @click.stop></div>
        </div>
      </div>
      
      <!-- Virtual rows -->
      <div class="grid-content" :style="{ height: totalSize + 'px' }">
        <div
          v-for="virtualRow in virtualRows"
          :key="virtualRow.key"
          class="grid-row"
          :class="getRowClass(virtualRow.index)"
          :style="{
            position: 'absolute',
            top: 0,
            left: 0,
            minWidth: totalColumnsWidth + 'px',
            height: virtualRow.size + 'px',
            transform: `translateY(${virtualRow.start}px)`
          }"
        >
          <!-- Delete/Restore button -->
          <div v-if="editable" class="row-actions">
            <button 
              v-if="changeBuffer.deleted.has(virtualRow.index)"
              class="row-action-btn restore"
              @click.stop="unmarkRowDeleted(virtualRow.index)"
              title="恢复"
            >↩</button>
            <button 
              v-else
              class="row-action-btn delete"
              @click.stop="markRowDeleted(virtualRow.index)"
              title="删除"
            >×</button>
          </div>
          
          <div
            v-for="(col, colIdx) in columns"
            :key="col.name"
            class="grid-cell"
            :class="[
              getCellClass(virtualRow.index, colIdx),
              { active: editingCell?.rowIdx === virtualRow.index && editingCell?.colIdx === colIdx }
            ]"
            :style="{ width: getColumnWidth(col.name) + 'px' }"
            @click.stop="handleCellClick($event, virtualRow.index, colIdx, sortedRows[virtualRow.index][colIdx])"
          >
            <!-- 编辑模式：显示 input -->
            <input
              v-if="editingCell?.rowIdx === virtualRow.index && editingCell?.colIdx === colIdx && editingCell?.isEditing"
              ref="popupRef"
              type="text"
              v-model="editingValue"
              class="cell-input-inline"
              @blur="saveEditedValue"
              @keydown="handleInputKeydown"
              @click.stop
            />
            <!-- 普通模式：显示文本 -->
            <span 
              v-else
              class="cell-text"
              :class="{ 'null-value': sortedRows[virtualRow.index][colIdx] === null }"
            >
              {{ sortedRows[virtualRow.index][colIdx] === null ? 'NULL' : sortedRows[virtualRow.index][colIdx] }}
            </span>
          </div>
        </div>
      </div>
    </div>
    
    <!-- Debug info -->
    <div class="debug-info" v-if="rows.length || changeBuffer.inserted.length">
      Rows: {{ rows.length + changeBuffer.inserted.length }} | 
      Modified: {{ changeBuffer.modified.size }} |
      Inserted: {{ changeBuffer.inserted.length }} |
      Deleted: {{ changeBuffer.deleted.size }}
    </div>
  </div>
</template>

<style scoped>
.virtual-grid-container {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  background: rgba(10, 10, 10, 0.6);
  border-radius: 8px;
  overflow: hidden;
  user-select: none;
  position: relative;
}

/* Toolbar */
.grid-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: rgba(20, 20, 20, 0.9);
  border-bottom: 1px solid rgba(60, 60, 60, 0.5);
  flex-shrink: 0;
}

.toolbar-btn {
  padding: 4px 12px;
  border: 1px solid rgba(100, 100, 100, 0.5);
  border-radius: 4px;
  background: rgba(40, 40, 40, 0.8);
  color: #ccc;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.toolbar-btn:hover {
  background: rgba(60, 60, 60, 0.8);
  color: #fff;
}

.btn-save {
  border-color: #4caf50;
  color: #4caf50;
}

.btn-save:hover {
  background: rgba(76, 175, 80, 0.2);
}

.btn-rollback {
  border-color: #ff9800;
  color: #ff9800;
}

.btn-rollback:hover {
  background: rgba(255, 152, 0, 0.2);
}

.toolbar-spacer {
  flex: 1;
}

/* Scroll Container */
.grid-scroll-container {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.grid-header {
  display: flex;
  position: sticky;
  top: 0;
  z-index: 10;
  background: rgba(20, 20, 20, 0.98);
  border-bottom: 1px solid rgba(60, 60, 60, 0.5);
  padding-left: 28px; /* Space for row actions */
}

.header-cell {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  font-size: 12px;
  font-weight: 600;
  color: #888;
  text-transform: uppercase;
  cursor: pointer;
  position: relative;
  flex-shrink: 0;
}

.header-cell:hover {
  background: rgba(109, 213, 237, 0.1);
  color: #fff;
}

.header-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.sort-icon {
  margin-left: 4px;
  color: #6dd5ed;
}

.resize-handle {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 6px;
  cursor: col-resize;
  background: transparent;
}

.resize-handle:hover {
  background: rgba(109, 213, 237, 0.3);
}

.grid-content {
  position: relative;
  width: 100%;
}

/* Row */
.grid-row {
  display: flex;
  border-bottom: 1px solid rgba(40, 40, 40, 0.8);
  padding-left: 28px; /* Space for row actions */
}

.grid-row:hover {
  background: rgba(109, 213, 237, 0.05);
}

.row-modified {
  background: rgba(255, 193, 7, 0.08) !important;
}

.row-inserted {
  background: rgba(76, 175, 80, 0.12) !important;
}

.row-deleted {
  background: rgba(244, 67, 54, 0.15) !important;
  opacity: 0.6;
  text-decoration: line-through;
}

/* Row Actions */
.row-actions {
  position: absolute;
  left: 4px;
  top: 50%;
  transform: translateY(-50%);
  width: 20px;
  display: flex;
  justify-content: center;
}

.row-action-btn {
  width: 18px;
  height: 18px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: #888;
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.row-action-btn.delete:hover {
  background: rgba(244, 67, 54, 0.3);
  color: #f44336;
}

.row-action-btn.restore:hover {
  background: rgba(76, 175, 80, 0.3);
  color: #4caf50;
}

/* Cell */
.grid-cell {
  display: flex;
  align-items: center;
  padding: 0 12px;
  height: 36px;
  font-size: 13px;
  color: #fff;
  cursor: pointer;
  flex-shrink: 0;
}

.grid-cell:hover {
  background: rgba(109, 213, 237, 0.08);
}

.grid-cell.active {
  background: rgba(109, 213, 237, 0.15);
  padding: 0; /* 编辑时去掉 padding */
}

.cell-modified {
  background: rgba(255, 193, 7, 0.15) !important;
}

.cell-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.null-value {
  color: #666;
  font-style: italic;
}

/* Inline Cell Input */
.cell-input-inline {
  width: 100%;
  height: 100%;
  padding: 0 12px;
  margin: 0;
  border: none;
  border-radius: 0;
  background: #1a2a1a;
  color: #fff;
  font-size: 13px;
  font-family: inherit;
  outline: 2px solid #4caf50;
  outline-offset: -2px;
  box-sizing: border-box;
}

.grid-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #666;
}

.debug-info {
  padding: 4px 12px;
  font-size: 11px;
  color: #666;
  background: rgba(0, 0, 0, 0.3);
  flex-shrink: 0;
}
</style>
