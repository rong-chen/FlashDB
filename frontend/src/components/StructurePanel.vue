<script setup>
import { ref, reactive, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import loader from '@monaco-editor/loader'

const props = defineProps({
  columns: { type: Array, default: () => [] },
  indexes: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  tableName: { type: String, default: '' },
  connectionId: { type: String, default: '' },
  database: { type: String, default: '' },
  panelHeight: { type: Number, default: 200 }
})

const emit = defineEmits(['execute-sql'])

const activeTab = ref('columns')
const editorContainer = ref(null)
const panelRef = ref(null)
const editorHeight = ref('200px')
const contentHeight = ref(200)  // 内容区域高度
let editor = null
let monaco = null
let resizeObserver = null

// SQL Tabs
const sqlTabs = ref([])
const activeSqlTab = ref(null)

// Load from localStorage
onMounted(async () => {
  loadSqlTabs()
  monaco = await loader.init()
  
  // Configure SQL autocomplete
  monaco.languages.registerCompletionItemProvider('sql', {
    provideCompletionItems: (model, position) => {
      const word = model.getWordUntilPosition(position)
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn
      }
      
      const keywords = [
        'SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'INSERT', 'INTO', 'VALUES',
        'UPDATE', 'SET', 'DELETE', 'CREATE', 'TABLE', 'DROP', 'ALTER',
        'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER', 'ON', 'GROUP', 'BY',
        'ORDER', 'ASC', 'DESC', 'LIMIT', 'OFFSET', 'HAVING', 'COUNT',
        'SUM', 'AVG', 'MAX', 'MIN', 'AS', 'DISTINCT', 'UNION', 'ALL',
        'LIKE', 'IN', 'NOT', 'NULL', 'IS', 'BETWEEN', 'EXISTS', 'CASE',
        'WHEN', 'THEN', 'ELSE', 'END', 'PRIMARY', 'KEY', 'FOREIGN',
        'REFERENCES', 'INDEX', 'UNIQUE', 'DEFAULT', 'AUTO_INCREMENT'
      ]
      
      const suggestions = keywords.map(k => ({
        label: k,
        kind: monaco.languages.CompletionItemKind.Keyword,
        insertText: k,
        range
      }))
      
      props.columns.forEach(col => {
        suggestions.push({
          label: col.name,
          kind: monaco.languages.CompletionItemKind.Field,
          insertText: col.name,
          detail: col.type,
          range
        })
      })
      
      if (props.tableName) {
        suggestions.push({
          label: props.tableName,
          kind: monaco.languages.CompletionItemKind.Class,
          insertText: props.tableName,
          detail: '当前表',
          range
        })
      }
      
      return { suggestions }
    }
  })
})

function loadSqlTabs() {
  try {
    const saved = localStorage.getItem('flashdb_sql_tabs')
    if (saved) {
      sqlTabs.value = JSON.parse(saved)
      if (sqlTabs.value.length > 0) {
        activeSqlTab.value = sqlTabs.value[0].id
      }
    }
  } catch (e) {}
  
  if (sqlTabs.value.length === 0) {
    addSqlTab()
  }
}

function saveSqlTabs() {
  try {
    localStorage.setItem('flashdb_sql_tabs', JSON.stringify(sqlTabs.value))
  } catch (e) {}
}

function addSqlTab() {
  const id = Date.now().toString()
  const num = sqlTabs.value.length + 1
  sqlTabs.value.push({
    id,
    name: `查询 ${num}`,
    content: ''
  })
  activeSqlTab.value = id
  saveSqlTabs()
  nextTick(() => {
    if (activeTab.value === 'sql') initEditor()
  })
}

function removeSqlTab(id, e) {
  e.stopPropagation()
  if (sqlTabs.value.length <= 1) return
  
  const idx = sqlTabs.value.findIndex(t => t.id === id)
  sqlTabs.value.splice(idx, 1)
  
  if (activeSqlTab.value === id) {
    activeSqlTab.value = sqlTabs.value[Math.min(idx, sqlTabs.value.length - 1)].id
  }
  saveSqlTabs()
  nextTick(() => {
    if (activeTab.value === 'sql') initEditor()
  })
}

function switchSqlTab(id) {
  // Save current content
  const current = sqlTabs.value.find(t => t.id === activeSqlTab.value)
  if (current && editor) {
    current.content = editor.getValue()
    saveSqlTabs()
  }
  
  activeSqlTab.value = id
  nextTick(() => {
    if (editor) {
      const tab = sqlTabs.value.find(t => t.id === id)
      editor.setValue(tab?.content || '')
    }
  })
}

function getCurrentTabContent() {
  if (editor) return editor.getValue()
  const tab = sqlTabs.value.find(t => t.id === activeSqlTab.value)
  return tab?.content || ''
}

watch(activeTab, (newTab) => {
  if (newTab === 'sql') {
    setTimeout(() => {
      if (editorContainer.value && monaco) {
        if (!editor) {
          initEditor()
        }
        updateEditorHeight()
      }
    }, 100)
  }
}, { immediate: true })

// 监听面板高度变化
watch(() => props.panelHeight, () => {
  if (activeTab.value === 'sql') {
    updateEditorHeight()
  }
})

function initEditor() {
  if (!editorContainer.value) return
  
  if (editor) {
    editor.dispose()
    editor = null
  }
  
  const tab = sqlTabs.value.find(t => t.id === activeSqlTab.value)
  
  editor = monaco.editor.create(editorContainer.value, {
    value: tab?.content || '',
    language: 'sql',
    theme: 'vs-dark',
    minimap: { enabled: false },
    fontSize: 13,
    lineNumbers: 'on',
    scrollBeyondLastLine: false,
    automaticLayout: true,
    wordWrap: 'on',
    padding: { top: 12, bottom: 12 },
    suggestOnTriggerCharacters: true,
    quickSuggestions: true,
    tabSize: 2,
    scrollbar: {
      verticalScrollbarSize: 8,
      horizontalScrollbarSize: 8,
      useShadows: false
    }
  })
  
  editor.onDidChangeModelContent(() => {
    const tab = sqlTabs.value.find(t => t.id === activeSqlTab.value)
    if (tab) {
      tab.content = editor.getValue()
      saveSqlTabs()
    }
  })
  
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
    executeSQL()
  })
}

onMounted(() => {
  // Watch for panel resize
  if (panelRef.value) {
    resizeObserver = new ResizeObserver(() => {
      updateEditorHeight()
    })
    resizeObserver.observe(panelRef.value)
  }
})

onBeforeUnmount(() => {
  if (resizeObserver) {
    resizeObserver.disconnect()
  }
  if (editor) {
    editor.dispose()
    editor = null
  }
})

function updateEditorHeight() {
  // 获取面板的实际渲染高度
  const actualPanelHeight = panelRef.value?.offsetHeight || props.panelHeight
  
  // Tab 按钮区域高度约 50px（包含 margin）
  const tabButtonsHeight = 50
  // SQL Tab Bar 高度约 36px
  const sqlTabBarHeight = 36
  // SQL Actions 区域高度约 44px
  const sqlActionsHeight = 44
  // Padding 24px (12px top + 12px bottom)
  const padding = 24
  
  // 计算内容区域高度
  const availableHeight = actualPanelHeight - tabButtonsHeight - padding
  contentHeight.value = Math.max(availableHeight, 100)
  
  // 计算编辑器高度 = 内容区域高度 - SQL Tab Bar - SQL Actions - gaps
  const editorHeightValue = contentHeight.value - sqlTabBarHeight - sqlActionsHeight - 16
  editorHeight.value = Math.max(editorHeightValue, 60) + 'px'
  
  if (editor) {
    editor.layout()
  }
}

function executeSQL() {
  const sql = getCurrentTabContent()
  if (!sql.trim()) return
  emit('execute-sql', sql)
}

function exportSQL() {
  const tab = sqlTabs.value.find(t => t.id === activeSqlTab.value)
  if (!tab || !tab.content.trim()) return
  
  const blob = new Blob([tab.content], { type: 'text/sql' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${tab.name}.sql`
  a.click()
  URL.revokeObjectURL(url)
}

function exportAllSQL() {
  const content = sqlTabs.value.map(t => `-- ${t.name}\n${t.content}`).join('\n\n')
  const blob = new Blob([content], { type: 'text/sql' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'all_queries.sql'
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <div class="structure-panel" ref="panelRef">
    <!-- Tab Buttons - 固定高度 -->
    <div class="tab-buttons">
      <button :class="{ active: activeTab === 'columns' }" @click="activeTab = 'columns'">
        列 <span class="badge" v-if="columns.length">{{ columns.length }}</span>
      </button>
      <button :class="{ active: activeTab === 'indexes' }" @click="activeTab = 'indexes'">
        索引 <span class="badge" v-if="indexes.length">{{ indexes.length }}</span>
      </button>
      <button :class="{ active: activeTab === 'sql' }" @click="activeTab = 'sql'">
        语句 <span class="badge" v-if="sqlTabs.length">{{ sqlTabs.length }}</span>
      </button>
    </div>
    
    <!-- Content Area - 填满剩余空间 -->
    <div class="content-area">
      <!-- Columns Tab -->
      <div v-if="activeTab === 'columns'" class="structure-list">
        <div v-if="loading" class="panel-loading"><p>加载中...</p></div>
        <div v-else-if="!tableName" class="panel-empty"><p>选择表查看结构</p></div>
        <template v-else>
          <div v-for="col in columns" :key="col.name" class="structure-row">
            <span class="col-name">{{ col.name }}</span>
            <span class="col-type">{{ col.type }}</span>
            <span v-if="col.key === 'PRI'" class="col-badge pk">PK</span>
            <span v-if="col.nullable === 'YES'" class="col-badge null">NULL</span>
          </div>
          <div v-if="!columns.length" class="panel-empty"><p>无列信息</p></div>
        </template>
      </div>
      
      <!-- Indexes Tab -->
      <div v-if="activeTab === 'indexes'" class="structure-list">
        <div v-if="loading" class="panel-loading"><p>加载中...</p></div>
        <div v-else-if="!tableName" class="panel-empty"><p>选择表查看结构</p></div>
        <template v-else>
          <div v-for="idx in indexes" :key="idx.name" class="structure-row">
            <span class="col-name">{{ idx.name }}</span>
            <span class="col-type">{{ idx.columns?.join(', ') }}</span>
            <span v-if="idx.primary" class="col-badge pk">主键</span>
            <span v-else-if="idx.unique" class="col-badge unique">唯一</span>
          </div>
          <div v-if="!indexes.length" class="panel-empty"><p>无索引</p></div>
        </template>
      </div>
      
      <!-- SQL Tab - 独立布局 -->
      <div v-show="activeTab === 'sql'" class="sql-tab-wrapper">
        <!-- SQL Tab Bar -->
        <div class="sql-tab-bar">
          <div class="sql-tabs">
            <div 
              v-for="tab in sqlTabs" 
              :key="tab.id"
              class="sql-tab-item"
              :class="{ active: tab.id === activeSqlTab }"
              @click="switchSqlTab(tab.id)"
            >
              <span class="tab-name">{{ tab.name }}</span>
              <span 
                v-if="sqlTabs.length > 1" 
                class="tab-close" 
                @click="removeSqlTab(tab.id, $event)"
              >×</span>
            </div>
            <button class="add-tab-btn" @click="addSqlTab">+</button>
          </div>
          <div class="sql-tab-actions">
            <button class="action-btn" @click="exportSQL" title="导出当前">↓</button>
            <button class="action-btn" @click="exportAllSQL" title="导出全部">⇊</button>
          </div>
        </div>
        
        <!-- Editor Container - 使用计算的高度 -->
        <div class="editor-container" ref="editorContainer" :style="{ height: editorHeight }"></div>
        
        <!-- SQL Actions -->
        <div class="sql-actions">
          <span class="hint">Ctrl/Cmd + Enter 执行 | 自动保存</span>
          <button class="run-btn" @click="executeSQL">▶ 执行</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.structure-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: rgba(10, 10, 10, 0.8);
  padding: 12px 16px;
  box-sizing: border-box;
  overflow: hidden;
}

.tab-buttons {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.tab-buttons button {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 16px;
  background: transparent;
  border: none;
  color: #888;
  cursor: pointer;
  font-size: 13px;
  border-radius: 6px;
  transition: all 0.15s;
}

.tab-buttons button:hover {
  background: rgba(109, 213, 237, 0.08);
  color: #fff;
}

.tab-buttons button.active {
  background: linear-gradient(135deg, rgba(109, 213, 237, 0.2), rgba(33, 147, 176, 0.2));
  color: #fff;
}

.badge {
  font-size: 11px;
  padding: 1px 6px;
  background: rgba(109, 213, 237, 0.2);
  border-radius: 10px;
  color: #6dd5ed;
}

/* Content Area - 填满剩余空间 */
.content-area {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.structure-list {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow-y: auto;
  min-height: 0;
}

.structure-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: rgba(20, 20, 20, 0.6);
  border-radius: 6px;
  flex-shrink: 0;
}

.col-name {
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  min-width: 120px;
}

.col-type {
  color: #888;
  font-size: 12px;
  font-family: 'SF Mono', Monaco, Consolas, monospace;
  flex: 1;
}

.col-badge {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 600;
}

.col-badge.pk { background: rgba(255, 193, 7, 0.2); color: #ffc107; }
.col-badge.null { background: rgba(108, 117, 125, 0.2); color: #6c757d; }
.col-badge.unique { background: rgba(109, 213, 237, 0.2); color: #6dd5ed; }

.panel-loading,
.panel-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #666;
}

/* SQL Tab Wrapper - 使用 flex 布局 */
.sql-tab-wrapper {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 8px;
  padding-right: 12px;
}

.sql-tab-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.sql-tabs {
  display: flex;
  gap: 4px;
  flex: 1;
  overflow-x: auto;
}

.sql-tab-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: rgba(40, 40, 40, 0.6);
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  color: #888;
  white-space: nowrap;
  transition: all 0.15s;
}

.sql-tab-item:hover {
  background: rgba(60, 60, 60, 0.8);
  color: #ccc;
}

.sql-tab-item.active {
  background: rgba(109, 213, 237, 0.15);
  color: #6dd5ed;
}

.tab-name {
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tab-close {
  font-size: 14px;
  opacity: 0.5;
  cursor: pointer;
}

.tab-close:hover {
  opacity: 1;
  color: #f56c6c;
}

.add-tab-btn {
  padding: 4px 10px;
  background: rgba(109, 213, 237, 0.1);
  border: none;
  border-radius: 4px;
  color: #6dd5ed;
  font-size: 14px;
  cursor: pointer;
}

.add-tab-btn:hover {
  background: rgba(109, 213, 237, 0.2);
}

.sql-tab-actions {
  display: flex;
  gap: 4px;
}

.action-btn {
  padding: 4px 8px;
  background: rgba(40, 40, 40, 0.6);
  border: none;
  border-radius: 4px;
  color: #888;
  font-size: 12px;
  cursor: pointer;
}

.action-btn:hover {
  background: rgba(60, 60, 60, 0.8);
  color: #fff;
}

.editor-container {
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid rgba(60, 60, 60, 0.5);
}

.editor-container :deep(.monaco-editor .view-lines) {
  padding-right: 12px !important;
}

.sql-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.sql-actions .hint {
  font-size: 11px;
  color: #555;
}

.run-btn {
  padding: 6px 16px;
  background: linear-gradient(135deg, #6dd5ed, #2193b0);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.run-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(109, 213, 237, 0.3);
}
</style>
