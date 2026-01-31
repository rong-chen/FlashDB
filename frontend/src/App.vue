<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  ListConnections, 
  CreateConnection, 
  Connect, 
  Disconnect, 
  TestConnection,
  DeleteConnection,
  UpdateConnection
} from '../wailsjs/go/services/ConnectionService'
import { 
  GetDatabases, 
  GetTables, 
  GetColumns, 
  GetIndexes,
  GetTableDDL,
  TruncateTable,
  DropTable,
  CloneDatabase,
  DropDatabase
} from '../wailsjs/go/services/ExplorerService'
import { Execute, ExecuteRaw, BatchUpdate } from '../wailsjs/go/services/QueryService'
import { SaveFile } from '../wailsjs/go/services/FileService'
import { format as formatSql } from 'sql-formatter'

// Components
import VirtualDataGrid from './components/VirtualDataGrid.vue'
import SqlEditor from './components/SqlEditor.vue'
import TabBar from './components/TabBar.vue'
import StructurePanel from './components/StructurePanel.vue'
import SchemaDiff from './components/SchemaDiff.vue'

// State
const connections = ref([])
const activeConnection = ref(null)
const databases = ref([])
const showConnectionForm = ref(false)
const isEditing = ref(false)
const loading = reactive({
  connections: false,
  databases: false,
  query: false
})

// Schema Diff state
const showSchemaDiff = ref(false)

// Connection form
const connectionForm = ref({
  name: '',
  type: 'mysql',
  host: 'localhost',
  port: 3306,
  username: 'root',
  password: '',
  database: ''
})

// Query state
const currentDatabase = ref('')

// Tree data
const treeData = ref([])
const expandedKeys = ref([])
const searchKeyword = ref('')

// Selected table structure
const selectedTable = ref(null)
const tableColumns = ref([])
const tableIndexes = ref([])
const loadingStructure = ref(false)
const tables = ref([])
const tableData = ref({ columns: null, rows: [] })
const loadingData = ref(false)

// SQL 执行历史
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
  ElMessage.success('已复制 SQL')
}

// Primary key for current table
const tablePrimaryKey = computed(() => {
  const pkCol = tableColumns.value.find(c => c.key === 'PRI')
  return pkCol?.name || ''
})

// Multi-tab state
let tabIdCounter = 1
function createTab() {
  return {
    id: String(tabIdCounter++),
    title: `查询 ${tabIdCounter - 1}`,
    sql: 'SELECT 1',
    result: null,
    error: ''
  }
}
const tabs = ref([createTab()])
const activeTabId = ref(tabs.value[0].id)

const activeTab = computed(() => tabs.value.find(t => t.id === activeTabId.value))

function addTab() {
  const newTab = createTab()
  tabs.value.push(newTab)
  activeTabId.value = newTab.id
}

function closeTab(id) {
  const idx = tabs.value.findIndex(t => t.id === id)
  if (idx > -1 && tabs.value.length > 1) {
    tabs.value.splice(idx, 1)
    if (activeTabId.value === id) {
      activeTabId.value = tabs.value[Math.max(0, idx - 1)].id
    }
  }
}

// SQL Editor ref for getting selected text
const sqlEditorRef = ref(null)

// Context menu (table)
const contextMenu = reactive({
  visible: false,
  x: 0,
  y: 0,
  node: null
})

// Context menu (database)
const dbContextMenu = reactive({
  visible: false,
  x: 0,
  y: 0,
  database: null
})

// Clone database dialog
const cloneDialog = reactive({
  visible: false,
  sourceDb: '',
  targetDb: '',
  includeData: false,
  loading: false
})

// Drop database dialog
const dropDbDialog = reactive({
  visible: false,
  database: '',
  confirmName: '',
  loading: false
})

// Initialize
onMounted(async () => {
  await loadConnections()
})

async function loadConnections() {
  loading.connections = true
  try {
    const res = await ListConnections()
    if (res.success) {
      connections.value = res.data || []
    }
  } catch (err) {
    console.error('Failed to load connections:', err)
  } finally {
    loading.connections = false
  }
}

function openNewConnectionForm() {
  isEditing.value = false
  connectionForm.value = {
    name: '',
    type: 'mysql',
    host: 'localhost',
    port: 3306,
    username: 'root',
    password: '',
    database: ''
  }
  showConnectionForm.value = true
}

function editConnection(conn) {
  isEditing.value = true
  connectionForm.value = {
    id: conn.id,
    name: conn.name,
    type: conn.type,
    host: conn.host,
    port: conn.port,
    username: conn.username,
    password: '',
    database: conn.database
  }
  showConnectionForm.value = true
}

async function testConnection() {
  try {
    const res = await TestConnection(connectionForm.value)
    if (res.success) {
      ElMessage.success('连接成功!')
    } else {
      ElMessage.error('连接失败: ' + res.error)
    }
  } catch (err) {
    ElMessage.error('连接失败: ' + err.message)
  }
}

async function saveConnection() {
  try {
    let res
    if (isEditing.value) {
      res = await UpdateConnection(connectionForm.value.id, connectionForm.value)
    } else {
      res = await CreateConnection(connectionForm.value)
    }
    if (res.success) {
      ElMessage.success(isEditing.value ? '更新成功!' : '创建成功!')
      showConnectionForm.value = false
      await loadConnections()
    } else {
      ElMessage.error('保存失败: ' + res.error)
    }
  } catch (err) {
    ElMessage.error('保存失败: ' + err.message)
  }
}

async function handleConnect(conn) {
  if (conn.connected) {
    const res = await Disconnect(conn.id)
    if (res.success) {
      conn.connected = false
      if (activeConnection.value?.id === conn.id) {
        activeConnection.value = null
        databases.value = []
        treeData.value = []
      }
      ElMessage.success('已断开连接')
    }
  } else {
    loading.databases = true
    try {
      const res = await Connect(conn.id)
      if (res.success) {
        conn.connected = true
        activeConnection.value = conn
        await loadDatabases(conn.id)
        ElMessage.success('连接成功!')
      } else {
        ElMessage.error('连接失败: ' + res.error)
      }
    } catch (err) {
      ElMessage.error('连接失败: ' + err.message)
    } finally {
      loading.databases = false
    }
  }
}

async function deleteConn(conn) {
  try {
    await ElMessageBox.confirm('确定删除此连接?', '确认删除', { type: 'warning' })
    const res = await DeleteConnection(conn.id)
    if (res.success) {
      ElMessage.success('已删除')
      await loadConnections()
      if (activeConnection.value?.id === conn.id) {
        activeConnection.value = null
        databases.value = []
        treeData.value = []
      }
    }
  } catch (err) {}
}

async function loadDatabases(connectionId) {
  const res = await GetDatabases(connectionId)
  if (res.success) {
    databases.value = res.data || []
    treeData.value = databases.value.map(db => ({
      id: db.name,
      label: db.name,
      type: 'database',
      children: [],
      isLeaf: false
    }))
  }
}

async function handleNodeExpand(node) {
  if (node.type === 'database' && node.children.length === 0) {
    loading.databases = true
    try {
      const res = await GetTables(activeConnection.value.id, node.id)
      if (res.success) {
        node.children = (res.data || []).map(t => ({
          id: `${node.id}.${t.name}`,
          label: t.name,
          type: 'table',
          database: node.id,
          rows: t.rows,
          children: [],
          isLeaf: true
        }))
      }
    } finally {
      loading.databases = false
    }
  }
}

async function handleNodeClick(node) {
  if (node.type === 'database') {
    currentDatabase.value = node.id
    selectedTable.value = null
  } else if (node.type === 'table') {
    currentDatabase.value = node.database
    sqlContent.value = `SELECT * FROM \`${node.label}\` LIMIT 100`
    // Load table structure
    selectedTable.value = node
    loadingStructure.value = true
    try {
      const [colRes, idxRes] = await Promise.all([
        GetColumns(activeConnection.value.id, node.database, node.label),
        GetIndexes(activeConnection.value.id, node.database, node.label)
      ])
      tableColumns.value = colRes.success ? colRes.data || [] : []
      tableIndexes.value = idxRes.success ? idxRes.data || [] : []
    } finally {
      loadingStructure.value = false
    }
  }
}

// Custom list handlers
async function toggleDb(db) {
  if (!db.expanded) {
    // Load tables if not loaded
    if (!db.children || db.children.length === 0) {
      loading.databases = true
      try {
        const res = await GetTables(activeConnection.value.id, db.id)
        if (res.success) {
          db.children = (res.data || []).map(t => ({
            id: `${db.id}.${t.name}`,
            label: t.name,
            type: 'table',
            database: db.id,
            rows: t.rows
          }))
        }
      } finally {
        loading.databases = false
      }
    }
  }
  db.expanded = !db.expanded
}

async function handleDbClick(db) {
  currentDatabase.value = db.id
  selectedTable.value = null
  tableColumns.value = []
  tableIndexes.value = []
  
  // Load tables for this database
  loading.databases = true
  try {
    const res = await GetTables(activeConnection.value.id, db.id)
    if (res.success) {
      tables.value = res.data || []
    }
  } finally {
    loading.databases = false
  }
}

async function handleTableClick(table) {
  currentDatabase.value = table.database
  selectedTable.value = table
  
  // Load structure and data in parallel
  loadingStructure.value = true
  loadingData.value = true
  
  try {
    const [colRes, idxRes, dataRes] = await Promise.all([
      GetColumns(activeConnection.value.id, table.database, table.label),
      GetIndexes(activeConnection.value.id, table.database, table.label),
      ExecuteRaw(activeConnection.value.id, table.database, `SELECT * FROM \`${table.label}\` LIMIT 100`)
    ])
    
    console.log('Columns response:', colRes)
    console.log('Indexes response:', idxRes)
    console.log('Data response:', dataRes)
    
    tableColumns.value = colRes.success ? colRes.data || [] : []
    tableIndexes.value = idxRes.success ? idxRes.data || [] : []
    
    if (dataRes.success && dataRes.data) {
      tableData.value = {
        columns: dataRes.data.columns || [],
        rows: dataRes.data.rows || []
      }
    } else {
      tableData.value = { columns: null, rows: [] }
    }
  } finally {
    loadingStructure.value = false
    loadingData.value = false
  }
}

async function selectTable(table) {
  selectedTable.value = {
    id: `${currentDatabase.value}.${table.name}`,
    label: table.name,
    database: currentDatabase.value,
    rows: table.rows
  }
  
  // Load structure and data in parallel
  loadingStructure.value = true
  loadingData.value = true
  
  try {
    const [colRes, idxRes, dataRes] = await Promise.all([
      GetColumns(activeConnection.value.id, currentDatabase.value, table.name),
      GetIndexes(activeConnection.value.id, currentDatabase.value, table.name),
      ExecuteRaw(activeConnection.value.id, currentDatabase.value, `SELECT * FROM \`${table.name}\` LIMIT 100`)
    ])
    tableColumns.value = colRes.success ? colRes.data || [] : []
    tableIndexes.value = idxRes.success ? idxRes.data || [] : []
    
    if (dataRes.success && dataRes.data) {
      tableData.value = {
        columns: dataRes.data.columns || [],
        rows: dataRes.data.rows || []
      }
    } else {
      tableData.value = { columns: null, rows: [] }
    }
  } finally {
    loadingStructure.value = false
    loadingData.value = false
  }
}

// Context menu
function handleContextMenu(e, data, node) {
  e.preventDefault()
  if (data.type === 'table') {
    // Close database context menu first
    dbContextMenu.visible = false
    
    contextMenu.visible = true
    contextMenu.x = e.clientX
    contextMenu.y = e.clientY
    contextMenu.node = data
  }
}

function hideContextMenu() {
  contextMenu.visible = false
  dbContextMenu.visible = false
}

// ========== Database Context Menu ==========
function showDbContextMenu(e, db) {
  // Filter system databases
  const systemDBs = ['information_schema', 'mysql', 'performance_schema', 'sys']
  if (systemDBs.includes(db.name || db.label || db.id)) {
    return // Don't show menu for system databases
  }
  
  e.preventDefault()
  // Close table context menu first
  contextMenu.visible = false
  
  dbContextMenu.x = e.clientX
  dbContextMenu.y = e.clientY
  dbContextMenu.database = db.name || db.label || db.id
  dbContextMenu.visible = true
}

function openCloneDialog() {
  cloneDialog.sourceDb = dbContextMenu.database
  cloneDialog.targetDb = dbContextMenu.database + '_copy'
  cloneDialog.includeData = false
  cloneDialog.visible = true
  hideContextMenu()
}

async function cloneDatabase() {
  if (!cloneDialog.targetDb.trim()) {
    ElMessage.warning('请输入目标数据库名')
    return
  }
  
  cloneDialog.loading = true
  try {
    const res = await CloneDatabase(
      activeConnection.value.id, 
      cloneDialog.sourceDb, 
      cloneDialog.targetDb,
      cloneDialog.includeData
    )
    if (res.success) {
      const dataMsg = cloneDialog.includeData ? '（含数据）' : '（仅结构）'
      ElMessage.success(`成功克隆 ${res.data.tablesCopied}/${res.data.tablesTotal} 个表到 ${cloneDialog.targetDb} ${dataMsg}`)
      cloneDialog.visible = false
      // Refresh database list
      await loadDatabases(activeConnection.value.id)
    } else {
      ElMessage.error(res.error)
    }
  } catch (e) {
    ElMessage.error('克隆失败: ' + e.message)
  } finally {
    cloneDialog.loading = false
  }
}

function openDropDbDialog() {
  dropDbDialog.database = dbContextMenu.database
  dropDbDialog.confirmName = ''
  dropDbDialog.visible = true
  hideContextMenu()
}

async function dropDatabaseConfirm() {
  if (dropDbDialog.confirmName !== dropDbDialog.database) {
    ElMessage.warning('请输入完整的数据库名称以确认删除')
    return
  }
  
  dropDbDialog.loading = true
  try {
    const res = await DropDatabase(
      activeConnection.value.id,
      dropDbDialog.database,
      dropDbDialog.confirmName
    )
    if (res.success) {
      ElMessage.success(res.data)
      dropDbDialog.visible = false
      // Refresh database list
      await loadDatabases(activeConnection.value.id)
    } else {
      ElMessage.error(res.error)
    }
  } catch (e) {
    ElMessage.error('删除失败: ' + e.message)
  } finally {
    dropDbDialog.loading = false
  }
}

async function copyDbName() {
  await navigator.clipboard.writeText(dropDbDialog.database)
  ElMessage.success('已复制数据库名称，可直接粘贴')
}

async function copyTableName() {
  if (contextMenu.node) {
    await navigator.clipboard.writeText(contextMenu.node.label)
    ElMessage.success('已复制表名')
  }
  hideContextMenu()
}

async function viewTableDDL() {
  if (contextMenu.node && activeConnection.value) {
    const res = await GetTableDDL(activeConnection.value.id, contextMenu.node.database, contextMenu.node.label)
    if (res.success) {
      // 格式化 DDL
      try {
        sqlContent.value = formatSql(res.data, { language: 'mysql' })
      } catch {
        sqlContent.value = res.data
      }
      ElMessage.success('已加载 DDL')
    } else {
      ElMessage.error(res.error)
    }
  }
  hideContextMenu()
}

// 格式化当前 SQL 内容
function formatCurrentSql() {
  if (!sqlContent.value) {
    ElMessage.warning('没有 SQL 内容')
    return
  }
  try {
    sqlContent.value = formatSql(sqlContent.value, { language: 'mysql' })
    ElMessage.success('已格式化')
  } catch (e) {
    ElMessage.error('格式化失败：' + e.message)
  }
}

async function truncateTable() {
  if (!contextMenu.node || !activeConnection.value) return
  try {
    await ElMessageBox.confirm(`确定清空表 ${contextMenu.node.label}?`, '危险操作', { type: 'error' })
    const res = await TruncateTable(activeConnection.value.id, contextMenu.node.database, contextMenu.node.label)
    if (res.success) {
      ElMessage.success('表已清空')
    } else {
      ElMessage.error(res.error)
    }
  } catch (err) {}
  hideContextMenu()
}

async function dropTable() {
  if (!contextMenu.node || !activeConnection.value) return
  try {
    await ElMessageBox.confirm(`确定删除表 ${contextMenu.node.label}? 此操作不可恢复!`, '危险操作', { type: 'error' })
    const res = await DropTable(activeConnection.value.id, contextMenu.node.database, contextMenu.node.label)
    if (res.success) {
      ElMessage.success('表已删除')
      // Only refresh tables for current database, keep expanded state
      const dbName = contextMenu.node.database
      const dbNode = treeData.value.find(d => d.id === dbName)
      if (dbNode) {
        const tablesRes = await GetTables(activeConnection.value.id, dbName)
        if (tablesRes.success) {
          dbNode.children = (tablesRes.data || []).map(t => ({
            id: `${dbName}.${t.name}`,
            label: t.name,
            type: 'table',
            database: dbName
          }))
        }
      }
      // Clear selection if deleted table was selected
      if (selectedTable.value?.label === contextMenu.node.label) {
        selectedTable.value = null
        tableData.rows = []
        tableData.columns = []
      }
    } else {
      ElMessage.error(res.error)
    }
  } catch (err) {}
  hideContextMenu()
}

// ========== Schema Diff ==========
function openSchemaDiff() {
  if (!activeConnection.value) {
    ElMessage.warning('请先连接数据库')
    return
  }
  showSchemaDiff.value = true
}

async function handleStructurePanelSQL(sql) {
  if (!activeConnection.value) {
    ElMessage.warning('请先连接数据库')
    return
  }
  if (!sql.trim()) return
  
  try {
    const res = await Execute({
      connectionId: activeConnection.value.id,
      database: currentDatabase.value,
      sql: sql,
      limit: 10000,
      offset: 0
    })
    
    if (res.success) {
      addToHistory(sql, true, res.data?.executionTime)
      if (res.data?.rows !== undefined) {
        // 将查询结果显示在上方表格区域
        tableData.value = {
          columns: res.data.columns || [],
          rows: res.data.rows || []
        }
        
        // 从 SQL 中解析表名并选中对应的表
        const tableMatch = sql.match(/(?:FROM|INTO|UPDATE|TABLE)\s+[`"]?(\w+)[`"]?/i)
        if (tableMatch && tableMatch[1]) {
          const tableName = tableMatch[1]
          // 查找左侧菜单中的对应表
          const foundTable = treeData.value
            .flatMap(db => db.children || [])
            .find(t => t.label === tableName)
          if (foundTable) {
            selectedTable.value = foundTable
          }
        }
        
        ElMessage.success(`执行成功，返回 ${res.data.rows.length} 行`)
      } else {
        ElMessage.success(`执行成功，影响 ${res.data?.affected || 0} 行`)
        // 如果是修改操作，刷新当前表数据
        if (selectedTable.value && /^(insert|update|delete|alter|drop|create)/i.test(sql.trim())) {
          await loadTableData(selectedTable.value, 0)
          await loadDatabases(activeConnection.value.id)
        }
      }
    } else {
      addToHistory(sql, false)
      ElMessage.error(res.error)
    }
  } catch (err) {
    addToHistory(sql, false)
    ElMessage.error('执行失败: ' + err.message)
  }
}

async function executeQuery() {
  if (!activeConnection.value) {
    ElMessage.warning('请先连接数据库')
    return
  }
  
  const tab = activeTab.value
  if (!tab) return
  
  loading.query = true
  tab.error = ''
  tab.result = null
  
  try {
    // Get selected text or full SQL from editor
    const sql = sqlEditorRef.value?.getSelectedText() || tab.sql
    
    const res = await Execute({
      connectionId: activeConnection.value.id,
      database: currentDatabase.value,
      sql: sql,
      limit: 10000,
      offset: 0
    })
    
    if (res.success) {
      tab.result = res.data
    } else {
      tab.error = res.error
    }
  } catch (err) {
    tab.error = err.message
  } finally {
    loading.query = false
  }
}

// Save table data changes
async function handleSaveTableData(changes) {
  if (!activeConnection.value || !selectedTable.value) {
    ElMessage.warning('请先选择表')
    return
  }
  
  loadingData.value = true
  
  try {
    const res = await BatchUpdate({
      connectionId: activeConnection.value.id,
      database: currentDatabase.value,
      table: selectedTable.value.label,
      updates: changes.updates || [],
      inserts: changes.inserts || [],
      deletes: changes.deletes || []
    })
    
    if (res.success) {
      ElMessage.success(`保存成功，影响 ${res.data?.affected || 0} 行`)
      // 记录到历史
      const summary = `[${selectedTable.value.label}] ${changes.updates?.length || 0} 更新, ${changes.inserts?.length || 0} 插入, ${changes.deletes?.length || 0} 删除`
      addToHistory(summary, true, res.data?.executionTime)
      // Refresh data
      if (res.data?.columns && res.data?.rows) {
        tableData.value = res.data
      }
    } else {
      ElMessage.error(res.error || '保存失败')
      const summary = `[${selectedTable.value.label}] 保存失败`
      addToHistory(summary, false, 0)
    }
  } catch (err) {
    ElMessage.error(err.message || '保存失败')
  } finally {
    loadingData.value = false
  }
}

// Filtered tree data
const filteredTreeData = computed(() => {
  if (!searchKeyword.value) return treeData.value
  return filterTree(treeData.value, searchKeyword.value.toLowerCase())
})

function filterTree(nodes, keyword) {
  const result = []
  for (const node of nodes) {
    if (node.label.toLowerCase().includes(keyword)) {
      result.push(node)
    } else if (node.children?.length) {
      const filteredChildren = filterTree(node.children, keyword)
      if (filteredChildren.length) {
        result.push({ ...node, children: filteredChildren })
      }
    }
  }
  return result
}

// ========== 数据导出功能 ==========
function handleExport(format) {
  if (!tableData.value.columns || !tableData.value.rows) {
    ElMessage.warning('没有数据可导出')
    return
  }
  
  const columns = tableData.value.columns.map(c => c.name)
  const rows = tableData.value.rows
  const tableName = selectedTable.value?.label || 'data'
  
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
      // 如果包含逗号、换行或引号，需要用引号包裹
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
  // Use Wails SaveFile dialog
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

async function exportTableData(format) {
  if (!contextMenu.node || !activeConnection.value) {
    ElMessage.error('请先选择要导出的表')
    return
  }
  
  const tableName = contextMenu.node.label
  const database = contextMenu.node.database
  
  if (!database || !tableName) {
    ElMessage.error('表信息不完整')
    return
  }
  
  hideContextMenu()
  
  try {
    // Fetch all data from table
    console.log('Exporting', tableName, 'from', database)
    const res = await Execute({
      connectionId: activeConnection.value.id,
      database: database,
      sql: `SELECT * FROM \`${tableName}\``,
      limit: 100000,
      offset: 0
    })
    
    console.log('Execute result:', res)
    
    if (!res.success) {
      ElMessage.error('获取数据失败: ' + (res.error || '未知错误'))
      return
    }
    
    // Check if columns is an array of column objects
    let columns = []
    if (Array.isArray(res.data?.columns)) {
      columns = res.data.columns.map(c => typeof c === 'string' ? c : c.name)
    }
    const rows = res.data?.rows || []
    
    let content = ''
    let ext = ''
    let mimeType = ''
    
    if (format === 'csv') {
      content = exportToCSV(columns, rows)
      ext = 'csv'
      mimeType = 'text/csv'
    } else if (format === 'json') {
      content = exportToJSON(columns, rows)
      ext = 'json'
      mimeType = 'application/json'
    } else if (format === 'sql') {
      // Generate INSERT statements
      const inserts = rows.map(row => {
        const values = row.map(v => {
          if (v === null) return 'NULL'
          if (typeof v === 'number') return v
          return `'${String(v).replace(/'/g, "''")}'`
        }).join(', ')
        return `INSERT INTO \`${tableName}\` (\`${columns.join('`, `')}\`) VALUES (${values});`
      })
      content = inserts.join('\n')
      ext = 'sql'
      mimeType = 'text/sql'
    }
    
    const saved = await downloadFile(content, `${tableName}.${ext}`, mimeType)
    if (saved) {
      ElMessage.success(`已导出 ${rows.length} 行数据`)
    }
    
  } catch (e) {
    console.error('Export error:', e)
    ElMessage.error('导出失败: ' + (e?.message || String(e) || '未知错误'))
  }
}

async function exportDatabaseSQL() {
  if (!dbContextMenu.database || !activeConnection.value) return
  
  const database = dbContextMenu.database
  hideContextMenu()
  
  ElMessage.info('正在导出数据库，请稍候...')
  
  try {
    // Get all tables
    const tablesRes = await GetTables(activeConnection.value.id, database)
    if (!tablesRes.success) {
      ElMessage.error('获取表列表失败')
      return
    }
    
    const tableInfos = tablesRes.data || []
    let sqlContent = `-- Database: ${database}\n-- Exported at: ${new Date().toISOString()}\n\n`
    
    for (const tableInfo of tableInfos) {
      const tableName = tableInfo.name || tableInfo
      // Get DDL
      const ddlRes = await GetTableDDL(activeConnection.value.id, database, tableName)
      if (ddlRes.success && ddlRes.data) {
        sqlContent += `-- Table: ${tableName}\n`
        sqlContent += ddlRes.data + ';\n\n'
      }
      
      // Get data
      const dataRes = await Execute({
        connectionId: activeConnection.value.id,
        database: database,
        sql: `SELECT * FROM \`${tableName}\``,
        limit: 100000,
        offset: 0
      })
      
      if (dataRes.success && dataRes.data?.rows?.length > 0) {
        // columns is array of ColumnMeta objects
        const columnNames = (dataRes.data.columns || []).map(c => typeof c === 'string' ? c : c.name)
        const rows = dataRes.data.rows
        
        sqlContent += `-- Data for ${tableName}\n`
        rows.forEach(row => {
          const values = row.map(v => {
            if (v === null) return 'NULL'
            if (typeof v === 'number') return v
            return `'${String(v).replace(/'/g, "''")}'`
          }).join(', ')
          sqlContent += `INSERT INTO \`${tableName}\` (\`${columnNames.join('`, `')}\`) VALUES (${values});\n`
        })
        sqlContent += '\n'
      }
    }
    
    const saved = await downloadFile(sqlContent, `${database}.sql`, 'text/sql')
    if (saved) {
      ElMessage.success(`已导出 ${tableInfos.length} 个表`)
    }
    
  } catch (e) {
    ElMessage.error('导出失败: ' + (e?.message || String(e) || '未知错误'))
  }
}

// Close context menu on click outside
onMounted(() => {
  document.addEventListener('click', hideContextMenu)
  // 页面刷新/关闭时断开连接
  window.addEventListener('beforeunload', handleBeforeUnload)
  // 加载 SQL 历史
  loadSqlHistory()
})

onUnmounted(() => {
  document.removeEventListener('click', hideContextMenu)
  window.removeEventListener('beforeunload', handleBeforeUnload)
  // 组件卸载时断开连接（用于 HMR）
  if (activeConnection.value) {
    Disconnect(activeConnection.value.id)
  }
})

// Vite HMR 热更新时断开连接
if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    if (activeConnection.value) {
      Disconnect(activeConnection.value.id)
    }
  })
}

// 页面关闭/刷新时断开数据库连接
function handleBeforeUnload() {
  if (activeConnection.value) {
    // 同步调用断开连接（beforeunload 中异步可能无法完成）
    Disconnect(activeConnection.value.id)
  }
}

// ========== 结构面板高度调整 ==========
const structurePanelHeight = ref(200)
let panelResizing = null

function startPanelResize(e) {
  e.preventDefault()
  panelResizing = { startY: e.clientY, startHeight: structurePanelHeight.value }
  document.addEventListener('mousemove', handlePanelResize)
  document.addEventListener('mouseup', stopPanelResize)
  document.body.style.cursor = 'ns-resize'
  document.body.style.userSelect = 'none'
}

function handlePanelResize(e) {
  if (!panelResizing) return
  // 向上拖动增加高度
  const diff = panelResizing.startY - e.clientY
  const maxHeight = window.innerHeight * 0.95  // 最多占用 95% 窗口高度
  const newHeight = Math.max(60, Math.min(maxHeight, panelResizing.startHeight + diff))
  structurePanelHeight.value = newHeight
}

function stopPanelResize() {
  panelResizing = null
  document.removeEventListener('mousemove', handlePanelResize)
  document.removeEventListener('mouseup', stopPanelResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}
</script>

<template>
  <div class="app-container dark">
    <!-- Title bar for macOS drag -->
    <div class="title-bar-drag"></div>
    
    <!-- Sidebar -->
    <div class="sidebar">
      <!-- Connection List -->
      <div class="connection-header">
        <span class="section-title">连接</span>
        <el-button type="primary" size="small" circle @click="openNewConnectionForm">
          <el-icon><Plus /></el-icon>
        </el-button>
      </div>
      
      <div class="connection-list" v-loading="loading.connections">
        <div v-if="connections.length === 0" class="empty-state small">
          <el-icon size="28" color="#666"><Connection /></el-icon>
          <p>暂无连接</p>
          <el-button type="primary" size="small" @click="openNewConnectionForm">创建连接</el-button>
        </div>
        
        <div 
          v-for="conn in connections" 
          :key="conn.id"
          class="connection-item"
          :class="{ active: activeConnection?.id === conn.id, connected: conn.connected }"
          @dblclick="handleConnect(conn)"
        >
          <el-icon class="icon" :size="16">
            <component :is="conn.connected ? 'CircleCheck' : 'Connection'" />
          </el-icon>
          <span class="name">{{ conn.name }}</span>
          <el-dropdown @command="cmd => cmd === 'edit' ? editConnection(conn) : cmd === 'delete' ? deleteConn(conn) : null">
            <el-button text size="small" @click.stop>
              <el-icon><MoreFilled /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="edit">编辑</el-dropdown-item>
                <el-dropdown-item v-if="conn.connected" @click="handleConnect(conn)">断开</el-dropdown-item>
                <el-dropdown-item v-else @click="handleConnect(conn)">连接</el-dropdown-item>
                <el-dropdown-item command="delete" divided style="color: #f56c6c;">删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
      
      <!-- Database Tree -->
      <div v-if="activeConnection" class="database-tree" v-loading="loading.databases">
        <div class="tree-header">
          <span class="section-title">数据库</span>
          <el-input
            v-model="searchKeyword"
            size="small"
            placeholder="搜索..."
            clearable
            style="width: 100%; margin-top: 8px;"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </div>
        <div class="db-list">
          <div 
            v-for="db in filteredTreeData" 
            :key="db.id" 
            class="db-group"
          >
            <div 
              class="db-item"
              :class="{ active: currentDatabase === db.id }"
              @click="toggleDb(db); handleDbClick(db)"
              @contextmenu="showDbContextMenu($event, db)"
            >
              <el-icon class="expand-icon" :class="{ expanded: db.expanded }"><ArrowRight /></el-icon>
              <span>{{ db.label }}</span>
            </div>
            <div v-if="db.expanded && db.children" class="table-list">
              <div 
                v-for="table in db.children" 
                :key="table.id"
                class="table-item"
                :class="{ active: selectedTable?.id === table.id }"
                @click="handleTableClick(table)"
                @contextmenu.prevent="handleContextMenu($event, table)"
              >
                <span>{{ table.label }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
      
      <!-- SQL History Panel -->
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
      
      <!-- Schema Diff Button -->
      <div class="sidebar-actions">
        <el-button size="small" @click="openSchemaDiff" :disabled="!activeConnection">
          Schema Diff
        </el-button>
      </div>
    </div>
    
    <!-- Main Content -->
    <div class="main-content">
      <!-- Schema Diff Page View -->
      <SchemaDiff
        v-if="showSchemaDiff"
        :databases="databases"
        :connection-id="activeConnection?.id"
        @close="showSchemaDiff = false"
      />
      
      <!-- Normal View -->
      <template v-else>
      <!-- SQL Editor (temporarily hidden)
      <TabBar 
        :tabs="tabs" 
        v-model:activeId="activeTabId"
        @add="addTab"
        @close="closeTab"
      />
      <div class="editor-section">
        <SqlEditor 
          v-if="activeTab"
          ref="sqlEditorRef"
          v-model="activeTab.sql"
          @execute="executeQuery"
        />
        <div class="execute-bar">
          <el-button type="primary" size="small" @click="executeQuery" :loading="loading.query">
            执行 (⌘↵)
          </el-button>
          <span v-if="activeTab?.result?.executionTime" class="exec-time">
            {{ activeTab.result.executionTime }}ms
          </span>
        </div>
      </div>
      <div class="result-section">
        <div v-if="activeTab?.error" class="error-panel">
          <p>{{ activeTab.error }}</p>
        </div>
        <VirtualDataGrid
          v-else-if="activeTab?.result?.columns"
          :columns="activeTab.result.columns"
          :rows="activeTab.result.rows"
          :loading="loading.query"
        />
        <div v-else class="empty-result">
          <p>执行查询查看结果</p>
        </div>
      </div>
      -->
      
      <!-- Table Data Panel -->
      <div 
        class="table-data-section" 
        :class="{ 'full-height': !selectedTable && !tableData.rows?.length }"
        :style="{ height: `calc(100% - ${structurePanelHeight + 6}px)` }"
      >
        <div v-if="!selectedTable && !tableData.rows?.length" class="empty-state">
          <p>选择表查看数据</p>
        </div>
        <template v-else>
          <div class="panel-header">
            <div class="header-left">
              <span class="panel-title">{{ selectedTable?.label || 'SQL 查询结果' }}</span>
              <span class="data-count" v-if="tableData.rows">{{ tableData.rows.length }} 行</span>
            </div>
            <div class="header-actions">
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
            :loading="loadingData"
            :editable="!!selectedTable"
            :primary-key="tablePrimaryKey"
            @save="handleSaveTableData"
          />
        </template>
      </div>
      
      <!-- Structure Panel Resize Handle -->
      <div 
        class="panel-resize-handle" 
        @mousedown="startPanelResize"
      ></div>
      
      <!-- Structure Panel -->
      <StructurePanel
        :columns="tableColumns"
        :indexes="tableIndexes"
        :loading="loadingStructure"
        :table-name="selectedTable?.label"
        :connection-id="activeConnection?.id"
        :database="currentDatabase"
        :panel-height="structurePanelHeight"
        class="structure-section"
        :style="{ height: structurePanelHeight + 'px' }"
        @execute-sql="handleStructurePanelSQL"
      />
      </template>
    </div>
    
    <div 
      v-show="contextMenu.visible" 
      class="context-menu"
      :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
    >
      <div class="menu-item" @click="copyTableName">
        <el-icon><CopyDocument /></el-icon>
        <span>复制表名</span>
      </div>
      <div class="menu-item" @click="viewTableDDL">
        <el-icon><Document /></el-icon>
        <span>查看 DDL</span>
      </div>
      <div class="menu-divider"></div>
      <div class="menu-item" @click="exportTableData('csv')">
        <el-icon><Download /></el-icon>
        <span>导出 CSV</span>
      </div>
      <div class="menu-item" @click="exportTableData('json')">
        <el-icon><Download /></el-icon>
        <span>导出 JSON</span>
      </div>
      <div class="menu-item" @click="exportTableData('sql')">
        <el-icon><Download /></el-icon>
        <span>导出 SQL</span>
      </div>
      <div class="menu-divider"></div>
      <div class="menu-item danger" @click="truncateTable">
        <el-icon><Delete /></el-icon>
        <span>清空表</span>
      </div>
      <div class="menu-item danger" @click="dropTable">
        <el-icon><DeleteFilled /></el-icon>
        <span>删除表</span>
      </div>
    </div>
    
    <!-- Database Context Menu -->
    <div 
      v-show="dbContextMenu.visible" 
      class="context-menu"
      :style="{ left: dbContextMenu.x + 'px', top: dbContextMenu.y + 'px' }"
    >
      <div class="menu-item" @click="openCloneDialog">
        <el-icon><CopyDocument /></el-icon>
        <span>克隆数据库</span>
      </div>
      <div class="menu-divider"></div>
      <div class="menu-item" @click="exportDatabaseSQL">
        <el-icon><Download /></el-icon>
        <span>导出 SQL</span>
      </div>
      <div class="menu-divider"></div>
      <div class="menu-item danger" @click="openDropDbDialog">
        <el-icon><DeleteFilled /></el-icon>
        <span>删除数据库</span>
      </div>
    </div>
    
    <!-- Clone Database Dialog -->
    <el-dialog
      v-model="cloneDialog.visible"
      title="克隆数据库"
      width="400px"
    >
      <el-form label-width="80px">
        <el-form-item label="源数据库">
          <el-input :value="cloneDialog.sourceDb" disabled />
        </el-form-item>
        <el-form-item label="目标名称">
          <el-input v-model="cloneDialog.targetDb" placeholder="新数据库名称" />
        </el-form-item>
        <el-form-item label="包含数据">
          <el-checkbox v-model="cloneDialog.includeData">复制表数据（可能较慢）</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cloneDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="cloneDatabase" :loading="cloneDialog.loading">
          克隆
        </el-button>
      </template>
    </el-dialog>
    
    <!-- Drop Database Dialog -->
    <el-dialog
      v-model="dropDbDialog.visible"
      title="删除数据库"
      width="450px"
    >
      <div class="drop-db-warning">
        <el-icon class="warning-icon"><WarningFilled /></el-icon>
        <p>此操作将<strong>永久删除</strong>数据库 <code class="copyable" @click="copyDbName" title="点击复制">{{ dropDbDialog.database }}</code> 及其所有数据！</p>
      </div>
      <div class="confirm-hint">请输入 <code class="copyable" @click="copyDbName">{{ dropDbDialog.database }}</code> 以确认删除：</div>
      <el-input 
        v-model="dropDbDialog.confirmName" 
        placeholder="输入数据库名称"
        style="width: 100%;"
      />
      <template #footer>
        <el-button @click="dropDbDialog.visible = false">取消</el-button>
        <el-button 
          type="danger" 
          @click="dropDatabaseConfirm" 
          :loading="dropDbDialog.loading"
          :disabled="dropDbDialog.confirmName !== dropDbDialog.database"
        >
          确认删除
        </el-button>
      </template>
    </el-dialog>
    
    <!-- Connection Form Dialog -->
    <el-dialog 
      v-model="showConnectionForm" 
      :title="isEditing ? '编辑连接' : '新建连接'"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form :model="connectionForm" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="connectionForm.name" placeholder="My Database" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="connectionForm.type" style="width: 100%;">
            <el-option label="MySQL" value="mysql" />
            <el-option label="PostgreSQL" value="postgres" disabled />
          </el-select>
        </el-form-item>
        <el-form-item label="主机">
          <el-input v-model="connectionForm.host" placeholder="localhost" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="connectionForm.port" :min="1" :max="65535" style="width: 100%;" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="connectionForm.username" placeholder="root" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="connectionForm.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-form-item label="数据库">
          <el-input v-model="connectionForm.database" placeholder="可选" />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="testConnection">测试连接</el-button>
        <el-button @click="showConnectionForm = false">取消</el-button>
        <el-button type="primary" @click="saveConnection">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.app-container {
  display: flex;
  height: 100vh;
  background: #000;
  user-select: none;
  -webkit-user-select: none;
}

.title-bar-drag {
  -webkit-app-region: drag;
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 28px;
  z-index: 100;
}

.sidebar {
  width: 200px;
  min-width: 200px;
  background: rgba(10, 10, 10, 0.85);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-right: none;
  display: flex;
  flex-direction: column;
  padding-top: 28px;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding-top: 28px;
  overflow: hidden;
  box-sizing: border-box;
}

/* Editor Section */
.editor-section {
  display: flex;
  flex-direction: column;
  height: 200px;
  min-height: 0;
  border-bottom: 1px solid rgba(40, 40, 40, 0.8);
}

.execute-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: rgba(15, 15, 15, 0.9);
}

.exec-time {
  font-size: 12px;
  color: #888;
}

/* Result Section */
.result-section {
  flex: 1;
  min-height: 150px;
  overflow: hidden;
}

.error-panel {
  padding: 16px;
  background: rgba(255, 100, 100, 0.1);
  color: #ff6b6b;
  font-size: 13px;
}

.empty-result {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #666;
}

/* Table Data Section */
.table-data-section {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.table-data-section.full-height {
  justify-content: center;
  align-items: center;
}

.table-data-section .panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: rgba(15, 15, 15, 0.9);
  flex-shrink: 0;
}

/* Panel Resize Handle */
.panel-resize-handle {
  height: 6px;
  background: rgba(40, 40, 40, 0.8);
  cursor: ns-resize;
  flex-shrink: 0;
  position: relative;
}

.panel-resize-handle:hover {
  background: rgba(109, 213, 237, 0.3);
}

.panel-resize-handle::after {
  content: '';
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 40px;
  height: 3px;
  background: rgba(100, 100, 100, 0.8);
  border-radius: 2px;
}

/* Structure Section */
.structure-section {
  flex: 0 0 auto;  /* 不伸缩，使用内联样式的高度 */
  min-height: 60px;
  overflow: hidden;
}

.section-title {
  font-size: 14px;
  color: #fff;
  font-weight: 700;
  letter-spacing: 0.5px;
}

/* SQL History Panel */
.history-section {
  border-top: 1px solid rgba(40, 40, 40, 0.8);
  flex-shrink: 0;
}

.history-header {
  padding: 10px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  font-size: 12px;
  color: #888;
  text-transform: uppercase;
  font-weight: 600;
}

.history-header:hover {
  background: rgba(109, 213, 237, 0.08);
  color: #fff;
}

.history-toggle {
  font-size: 10px;
}

.history-content {
  max-height: 250px;
  overflow-y: auto;
  padding: 0 12px 12px;
}

.empty-history {
  color: #666;
  font-size: 12px;
  text-align: center;
  padding: 20px;
}

.history-actions {
  text-align: right;
  margin-bottom: 8px;
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.history-item {
  padding: 8px 10px;
  border-radius: 6px;
  background: rgba(30, 30, 30, 0.8);
  cursor: pointer;
  border-left: 3px solid #4caf50;
}

.history-item.failed {
  border-left-color: #f44336;
}

.history-item:hover {
  background: rgba(50, 50, 50, 0.9);
}

.history-sql {
  font-size: 11px;
  color: #aaa;
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.history-meta {
  display: flex;
  gap: 10px;
  margin-top: 4px;
  font-size: 10px;
  color: #666;
}

.connection-header {
  padding: 12px 16px;
  border-bottom: none;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.connection-list {
  max-height: 200px;
  overflow-y: auto;
  padding: 8px 12px;
}

.connection-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  margin: 0 4px 4px 4px;
}

.connection-item:hover { background: rgba(109, 213, 237, 0.08); }
.connection-item.active { 
  background: linear-gradient(135deg, rgba(109, 213, 237, 0.15), rgba(33, 147, 176, 0.15));
}
.connection-item .icon { margin-right: 8px; color: var(--text-muted, #6e7681); }
.connection-item.connected .icon { color: #3fb950; }
.connection-item .name { flex: 1; font-size: 13px; }

.database-tree {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.tree-header {
  padding: 12px 16px 8px;
}

.tree-header :deep(.el-input__wrapper) {
  background: rgba(40, 40, 40, 1) !important;
  box-shadow: none !important;
  border: 1px solid rgba(80, 80, 80, 0.8) !important;
}

.tree-header :deep(.el-input__wrapper:hover) {
  border-color: rgba(109, 213, 237, 0.5) !important;
}

.tree-header :deep(.el-input__wrapper.is-focus) {
  border-color: rgba(109, 213, 237, 0.7) !important;
}

/* Custom Database List */
.db-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.db-group {
  margin-bottom: 4px;
}

.db-item, .table-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  margin: 2px 16px;
  border-radius: 8px;
  cursor: pointer;
  color: #fff;
  font-size: 13px;
  transition: background 0.15s;
}

.db-item:hover, .table-item:hover {
  background: rgba(109, 213, 237, 0.08);
}

.db-item.active, .table-item.active {
  background: linear-gradient(135deg, rgba(109, 213, 237, 0.2), rgba(33, 147, 176, 0.2));
}

.db-item .expand-icon {
  margin-right: 6px;
  transition: transform 0.2s;
  color: #888;
  font-size: 12px;
}

.db-item .expand-icon.expanded {
  transform: rotate(90deg);
}

.table-list {
  padding-left: 16px;
}

.table-item {
  font-size: 12px;
  padding: 6px 12px;
  color: #7dd3fc;  /* 浅蓝色区分表名 */
}

.row-count {
  color: #666;
  margin-left: auto;
  font-size: 11px;
}

/* Data Panel - Top (Table Data) */
.data-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 16px;
  overflow: hidden;
}

.data-count {
  color: #888;
  font-size: 12px;
}

.data-grid {
  flex: 1;
  overflow: auto;
  background: rgba(10, 10, 10, 0.6);
  border-radius: 8px;
  user-select: none;
}

.data-grid table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.data-grid th,
.data-grid td {
  padding: 8px 12px;
  text-align: left;
  white-space: nowrap;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.data-grid th {
  background: rgba(20, 20, 20, 0.9);
  font-weight: 600;
  position: sticky;
  top: 0;
  color: #888;
}

.data-grid td {
  color: #fff;
}

.data-grid tr:hover {
  background: rgba(109, 213, 237, 0.05);
}

.null-value {
  color: #666;
  font-style: italic;
}

/* Cell text ellipsis */
.cell-text {
  display: block;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Fixed row height */
.data-grid tr {
  height: 36px;
}

.data-grid td {
  height: 36px;
  max-height: 36px;
  cursor: pointer;
  position: relative;
  overflow: visible;
}

.data-grid td.editing {
  padding: 0;
  background: rgba(109, 213, 237, 0.1);
}

.cell-input {
  position: absolute;
  top: 0;
  left: 0;
  z-index: 100;
  min-width: 200px;
  max-width: 400px;
  min-height: 80px;
  max-height: 150px;
  padding: 8px;
  border: 1px solid rgba(109, 213, 237, 0.4);
  border-radius: 4px;
  background: rgba(20, 20, 20, 0.98);
  color: #fff;
  font-size: 13px;
  font-family: 'SF Mono', Monaco, Consolas, monospace;
  resize: both;
  outline: none;
  box-shadow: 0 4px 16px rgba(0,0,0,0.5);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.table-count {
  color: #666;
  font-size: 12px;
}

.table-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
  overflow-y: auto;
  flex: 1;
}

.table-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: rgba(20, 20, 20, 0.8);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
}

.table-card:hover {
  background: rgba(109, 213, 237, 0.1);
}

.table-card.active {
  background: linear-gradient(135deg, rgba(109, 213, 237, 0.2), rgba(33, 147, 176, 0.2));
}

.table-name {
  color: #fff;
  font-size: 13px;
}

.table-rows {
  color: #666;
  font-size: 11px;
}

/* Structure Panel - Bottom */
.structure-panel-bottom {
  height: 300px;
  min-height: 200px;
  display: flex;
  flex-direction: column;
  background: rgba(10, 10, 10, 0.8);
  padding: 16px;
}

.structure-panel-bottom.empty {
  justify-content: center;
  align-items: center;
}

.structure-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.tab-buttons {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.tab-buttons button {
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

.structure-content {
  flex: 1;
  overflow-y: auto;
}

.structure-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.structure-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  border-radius: 6px;
  background: rgba(20, 20, 20, 0.6);
}

.col-name {
  color: #fff;
  font-size: 13px;
  min-width: 150px;
}

.col-type {
  color: #888;
  font-size: 12px;
  flex: 1;
}

.col-badge {
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
}

.col-badge.pk {
  background: rgba(210, 153, 34, 0.2);
  color: #d29922;
}

.col-badge.null {
  background: rgba(136, 136, 136, 0.2);
  color: #888;
}

.col-badge.unique {
  background: rgba(63, 185, 80, 0.2);
  color: #3fb950;
}

.result-header {
  padding: 8px 16px;
  background: rgba(10, 10, 10, 0.9);
  border-bottom: none;
  font-size: 12px;
  color: #888;
}

.result-grid {
  flex: 1;
  overflow: auto;
}

.result-grid table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.result-grid th, .result-grid td {
  padding: 6px 12px;
  text-align: left;
  border-bottom: 1px solid #2a2a2a;
  white-space: nowrap;
}

.result-grid th {
  background: #1f1f1f;
  font-weight: 600;
  position: sticky;
  top: 0;
  color: #888;
}

.result-grid tr:hover { background: #252525; }
.null-value { color: #666; font-style: italic; }

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #666;
  gap: 8px;
}

.empty-state.small { padding: 32px 20px; }
.empty-state.error { color: #f56c6c; }
.empty-state.success { color: #67c23a; }

/* Context Menu */
.context-menu {
  position: fixed;
  background: #2a2a2a;
  border: 1px solid #444;
  border-radius: 6px;
  padding: 4px 0;
  min-width: 150px;
  z-index: 1000;
  box-shadow: 0 4px 12px rgba(0,0,0,0.4);
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  cursor: pointer;
  font-size: 13px;
}

.menu-item:hover { background: #3a3a3a; }
.menu-item.danger { color: #f56c6c; }
.menu-divider { height: 1px; background: #444; margin: 4px 0; }

/* Drop database warning */
.drop-db-warning {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  background: rgba(245, 108, 108, 0.1);
  border: 1px solid rgba(245, 108, 108, 0.3);
  border-radius: 8px;
  margin-bottom: 20px;
}
.drop-db-warning .warning-icon {
  font-size: 24px;
  color: #f56c6c;
  flex-shrink: 0;
}
.drop-db-warning p {
  margin: 0;
  line-height: 1.6;
  color: #f0f0f0;
}
.drop-db-warning code {
  background: rgba(0, 0, 0, 0.3);
  padding: 2px 6px;
  border-radius: 4px;
  color: #f56c6c;
  font-weight: bold;
}
.drop-db-warning code.copyable {
  cursor: pointer;
  transition: all 0.2s;
}
.drop-db-warning code.copyable:hover {
  background: rgba(0, 0, 0, 0.5);
  text-decoration: underline;
}
.confirm-hint {
  margin-bottom: 10px;
  color: #ccc;
  font-size: 14px;
}
.confirm-hint code {
  background: rgba(100, 180, 255, 0.15);
  padding: 2px 6px;
  border-radius: 4px;
  color: #66b1ff;
  font-weight: bold;
  cursor: pointer;
}
.confirm-hint code:hover {
  text-decoration: underline;
}

/* Drop database dialog input always show background */
:deep(.el-dialog) .el-input__wrapper {
  background-color: rgba(30, 30, 30, 0.8) !important;
  box-shadow: 0 0 0 1px rgba(100, 100, 100, 0.4) inset !important;
}

/* Element Plus overrides */
.el-tree { 
  background: transparent !important; 
  padding: 0 !important;
  color: #fff !important;
}

.el-tree .el-tree-node {
  padding: 0;
}

.el-tree .el-tree-node__content {
  border-radius: 8px !important;
  margin: 2px 16px !important;
  padding: 6px 12px !important;
}

.el-tree .el-tree-node__content:hover {
  background: rgba(109, 213, 237, 0.08) !important;
}

.el-tree .el-tree-node.is-current > .el-tree-node__content {
  background: linear-gradient(135deg, rgba(109, 213, 237, 0.15), rgba(33, 147, 176, 0.15)) !important;
}

/* Table Structure Panel */
.structure-panel {
  border-top: 1px solid #333;
  display: flex;
  flex-direction: column;
}

.structure-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #262626;
  font-size: 12px;
  font-weight: 600;
}

.structure-tabs {
  flex: 1;
  overflow: hidden;
  background: transparent !important;
  border: none !important;
}

.structure-tabs :deep(.el-tabs__header) {
  background: #1f1f1f;
  margin: 0;
}

.structure-tabs :deep(.el-tabs__content) {
  padding: 0;
  height: calc(100% - 40px);
  overflow-y: auto;
}

.structure-list {
  padding: 4px 0;
}

.structure-item {
  padding: 6px 12px;
  border-bottom: 1px solid #2a2a2a;
}

.structure-item:hover {
  background: #2a2a2a;
}

.col-main {
  display: flex;
  align-items: center;
  gap: 6px;
}

.col-name {
  font-size: 12px;
  font-weight: 500;
}

.col-info {
  display: flex;
  gap: 8px;
  font-size: 11px;
  color: #666;
  margin-top: 2px;
}

.col-type {
  color: #888;
}

.col-nullable {
  color: #67c23a;
}

/* Sidebar Actions */
.sidebar-actions {
  padding: 12px;
  border-top: 1px solid rgba(40, 40, 40, 0.8);
}

/* Schema Diff */
.diff-selects {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}

.diff-arrow {
  color: #888;
  font-size: 18px;
}

.diff-result {
  max-height: 400px;
  overflow-y: auto;
}

.diff-section {
  margin-bottom: 20px;
}

.diff-section h4 {
  margin-bottom: 8px;
  color: #fff;
  font-size: 14px;
}

.diff-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.diff-item {
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  font-family: monospace;
}

.diff-list.added .diff-item {
  background: rgba(76, 175, 80, 0.2);
  color: #4caf50;
}

.diff-list.removed .diff-item {
  background: rgba(244, 67, 54, 0.2);
  color: #f44336;
}

.column-diff {
  background: rgba(30, 30, 30, 0.8);
  padding: 10px;
  border-radius: 6px;
  margin-bottom: 8px;
}

.column-diff strong {
  color: #fff;
}

.diff-cols {
  font-size: 12px;
  font-family: monospace;
  margin-top: 6px;
}

.diff-cols.added {
  color: #4caf50;
}

.diff-cols.removed {
  color: #f44336;
}

.diff-same {
  text-align: center;
  color: #4caf50;
  font-size: 16px;
  padding: 40px;
}

/* Header Actions */
.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-actions {
  display: flex;
  gap: 8px;
}
</style>
