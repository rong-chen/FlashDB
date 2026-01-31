<script setup>
import { ref, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps({
  databases: Array,
  connectionId: String
})

const emit = defineEmits(['close'])

// 从父组件导入的 API
import { GetTables, GetColumns } from '../../wailsjs/go/services/ExplorerService'

const sourceDb = ref('')
const targetDb = ref('')
const loading = ref(false)
const result = ref(null)

// 重置结果当数据库选择改变
watch([sourceDb, targetDb], () => {
  result.value = null
})

async function compare() {
  if (!sourceDb.value || !targetDb.value) {
    ElMessage.warning('请选择两个数据库')
    return
  }
  if (sourceDb.value === targetDb.value) {
    ElMessage.warning('请选择不同的数据库')
    return
  }
  
  loading.value = true
  try {
    // 获取两个数据库的表列表
    const [sourceTables, targetTables] = await Promise.all([
      GetTables(props.connectionId, sourceDb.value),
      GetTables(props.connectionId, targetDb.value)
    ])
    
    if (!sourceTables.success || !targetTables.success) {
      ElMessage.error('获取表列表失败')
      return
    }
    
    const sourceSet = new Set(sourceTables.data.map(t => t.name))
    const targetSet = new Set(targetTables.data.map(t => t.name))
    
    const onlyInSource = [...sourceSet].filter(t => !targetSet.has(t))
    const onlyInTarget = [...targetSet].filter(t => !sourceSet.has(t))
    const inBoth = [...sourceSet].filter(t => targetSet.has(t))
    
    // 对比所有共同表的结构
    const tableDiffs = []
    for (const tableName of inBoth) {
      const [srcCols, tgtCols] = await Promise.all([
        GetColumns(props.connectionId, sourceDb.value, tableName),
        GetColumns(props.connectionId, targetDb.value, tableName)
      ])
      
      if (srcCols.success && tgtCols.success) {
        const srcColMap = new Map(srcCols.data.map(c => [c.name, c]))
        const tgtColMap = new Map(tgtCols.data.map(c => [c.name, c]))
        
        const added = []
        const removed = []
        const modified = []
        
        // 新增的列
        for (const [name, col] of tgtColMap) {
          if (!srcColMap.has(name)) {
            added.push({ name, type: col.type, nullable: col.nullable })
          }
        }
        
        // 删除的列
        for (const [name, col] of srcColMap) {
          if (!tgtColMap.has(name)) {
            removed.push({ name, type: col.type, nullable: col.nullable })
          }
        }
        
        // 修改的列
        for (const [name, srcCol] of srcColMap) {
          const tgtCol = tgtColMap.get(name)
          if (tgtCol) {
            const diffs = []
            if (srcCol.type !== tgtCol.type) {
              diffs.push({ field: '类型', from: srcCol.type, to: tgtCol.type })
            }
            if (srcCol.nullable !== tgtCol.nullable) {
              diffs.push({ field: '可空', from: srcCol.nullable, to: tgtCol.nullable })
            }
            if (srcCol.default !== tgtCol.default) {
              diffs.push({ field: '默认值', from: srcCol.default || 'NULL', to: tgtCol.default || 'NULL' })
            }
            if (srcCol.key !== tgtCol.key) {
              diffs.push({ field: '索引', from: srcCol.key || '无', to: tgtCol.key || '无' })
            }
            if (diffs.length > 0) {
              modified.push({ name, diffs })
            }
          }
        }
        
        if (added.length || removed.length || modified.length) {
          tableDiffs.push({ table: tableName, added, removed, modified })
        }
      }
    }
    
    result.value = {
      sourceDb: sourceDb.value,
      targetDb: targetDb.value,
      onlyInSource,
      onlyInTarget,
      tableDiffs,
      commonTables: inBoth.length,
      hasDiff: onlyInSource.length > 0 || onlyInTarget.length > 0 || tableDiffs.length > 0
    }
  } catch (e) {
    ElMessage.error('对比失败：' + e.message)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="schema-diff-page">
    <!-- Header -->
    <div class="page-header">
      <div class="header-left">
        <button class="back-btn" @click="emit('close')">← 返回</button>
        <h2>Schema Diff - 结构对比</h2>
      </div>
    </div>
    
    <!-- Selector Bar -->
    <div class="selector-bar">
      <div class="selector-group">
        <label>源数据库</label>
        <el-select v-model="sourceDb" placeholder="选择源数据库" style="width: 200px">
          <el-option v-for="db in databases" :key="db.name" :label="db.name" :value="db.name" />
        </el-select>
      </div>
      <div class="arrow">→</div>
      <div class="selector-group">
        <label>目标数据库</label>
        <el-select v-model="targetDb" placeholder="选择目标数据库" style="width: 200px">
          <el-option v-for="db in databases" :key="db.name" :label="db.name" :value="db.name" />
        </el-select>
      </div>
      <el-button type="primary" @click="compare" :loading="loading" size="large">
        开始对比
      </el-button>
    </div>
    
    <!-- Results -->
    <div class="diff-content" v-if="result">
      <!-- Summary -->
      <div class="summary-bar">
        <div class="summary-item">
          <span class="label">源库</span>
          <span class="value">{{ result.sourceDb }}</span>
        </div>
        <div class="summary-item">
          <span class="label">目标库</span>
          <span class="value">{{ result.targetDb }}</span>
        </div>
        <div class="summary-item">
          <span class="label">共同表</span>
          <span class="value">{{ result.commonTables }}</span>
        </div>
        <div class="summary-item removed" v-if="result.onlyInSource.length">
          <span class="label">仅源库</span>
          <span class="value">{{ result.onlyInSource.length }}</span>
        </div>
        <div class="summary-item added" v-if="result.onlyInTarget.length">
          <span class="label">仅目标库</span>
          <span class="value">{{ result.onlyInTarget.length }}</span>
        </div>
        <div class="summary-item modified" v-if="result.tableDiffs.length">
          <span class="label">有差异</span>
          <span class="value">{{ result.tableDiffs.length }}</span>
        </div>
      </div>
      
      <!-- No Diff -->
      <div v-if="!result.hasDiff" class="no-diff">
        <div class="icon">✓</div>
        <div class="text">两个数据库结构完全相同</div>
      </div>
      
      <!-- Diff Panels -->
      <div v-else class="diff-panels">
        <!-- Tables only in source -->
        <div v-if="result.onlyInSource.length" class="diff-panel">
          <div class="panel-header removed">
            <span class="icon">−</span>
            <span>仅在源库存在的表 ({{ result.onlyInSource.length }})</span>
          </div>
          <div class="panel-content">
            <div class="table-tags">
              <span v-for="t in result.onlyInSource" :key="t" class="table-tag removed">{{ t }}</span>
            </div>
          </div>
        </div>
        
        <!-- Tables only in target -->
        <div v-if="result.onlyInTarget.length" class="diff-panel">
          <div class="panel-header added">
            <span class="icon">+</span>
            <span>仅在目标库存在的表 ({{ result.onlyInTarget.length }})</span>
          </div>
          <div class="panel-content">
            <div class="table-tags">
              <span v-for="t in result.onlyInTarget" :key="t" class="table-tag added">{{ t }}</span>
            </div>
          </div>
        </div>
        
        <!-- Column differences -->
        <div v-if="result.tableDiffs.length" class="diff-panel column-diffs">
          <div class="panel-header modified">
            <span class="icon">~</span>
            <span>列结构差异 ({{ result.tableDiffs.length }} 个表)</span>
          </div>
          <div class="panel-content">
            <div v-for="diff in result.tableDiffs" :key="diff.table" class="table-diff-card">
              <div class="table-name">{{ diff.table }}</div>
              
              <!-- Added columns -->
              <div v-if="diff.added.length" class="column-section">
                <div class="section-label added">+ 新增列</div>
                <table class="column-table">
                  <tr v-for="col in diff.added" :key="col.name">
                    <td class="col-name">{{ col.name }}</td>
                    <td class="col-type">{{ col.type }}</td>
                    <td class="col-nullable">{{ col.nullable === 'YES' ? 'NULL' : 'NOT NULL' }}</td>
                  </tr>
                </table>
              </div>
              
              <!-- Removed columns -->
              <div v-if="diff.removed.length" class="column-section">
                <div class="section-label removed">− 删除列</div>
                <table class="column-table">
                  <tr v-for="col in diff.removed" :key="col.name">
                    <td class="col-name">{{ col.name }}</td>
                    <td class="col-type">{{ col.type }}</td>
                    <td class="col-nullable">{{ col.nullable === 'YES' ? 'NULL' : 'NOT NULL' }}</td>
                  </tr>
                </table>
              </div>
              
              <!-- Modified columns -->
              <div v-if="diff.modified.length" class="column-section">
                <div class="section-label modified">~ 修改列</div>
                <table class="column-table">
                  <tr v-for="col in diff.modified" :key="col.name">
                    <td class="col-name">{{ col.name }}</td>
                    <td class="col-changes">
                      <span v-for="d in col.diffs" :key="d.field" class="change-item">
                        {{ d.field }}: <span class="from">{{ d.from }}</span> → <span class="to">{{ d.to }}</span>
                      </span>
                    </td>
                  </tr>
                </table>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    
    <!-- Empty State -->
    <div v-else class="empty-state">
      <p>选择两个数据库进行结构对比</p>
    </div>
  </div>
</template>

<style scoped>
.schema-diff-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #0a0a0a;
  color: #fff;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(40, 40, 40, 0.8);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.back-btn {
  background: none;
  border: 1px solid rgba(100, 100, 100, 0.5);
  color: #888;
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}

.back-btn:hover {
  background: rgba(100, 100, 100, 0.2);
  color: #fff;
}

.page-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.selector-bar {
  display: flex;
  align-items: flex-end;
  gap: 20px;
  padding: 20px;
  background: rgba(20, 20, 20, 0.8);
  border-bottom: 1px solid rgba(40, 40, 40, 0.8);
}

.selector-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.selector-group label {
  font-size: 12px;
  color: #888;
  text-transform: uppercase;
}

.arrow {
  font-size: 24px;
  color: #555;
  margin-top: 18px;  /* 和选择框对齐 */
}

.diff-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.summary-bar {
  display: flex;
  gap: 20px;
  padding: 16px 20px;
  background: rgba(30, 30, 30, 0.6);
  border-radius: 8px;
  margin-bottom: 20px;
}

.summary-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.summary-item .label {
  font-size: 11px;
  color: #666;
  text-transform: uppercase;
}

.summary-item .value {
  font-size: 16px;
  font-weight: 600;
}

.summary-item.removed .value { color: #f44336; }
.summary-item.added .value { color: #4caf50; }
.summary-item.modified .value { color: #ff9800; }

.no-diff {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px;
  color: #4caf50;
}

.no-diff .icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.no-diff .text {
  font-size: 18px;
}

.diff-panels {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.diff-panel {
  background: rgba(25, 25, 25, 0.8);
  border-radius: 10px;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 20px;
  font-weight: 600;
  font-size: 14px;
}

.panel-header.removed { background: rgba(244, 67, 54, 0.15); color: #f44336; }
.panel-header.added { background: rgba(76, 175, 80, 0.15); color: #4caf50; }
.panel-header.modified { background: rgba(255, 152, 0, 0.15); color: #ff9800; }

.panel-header .icon {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  font-size: 16px;
  font-weight: bold;
}

.panel-header.removed .icon { background: rgba(244, 67, 54, 0.3); }
.panel-header.added .icon { background: rgba(76, 175, 80, 0.3); }
.panel-header.modified .icon { background: rgba(255, 152, 0, 0.3); }

.panel-content {
  padding: 16px 20px;
}

.table-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.table-tag {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 12px;
  font-family: monospace;
}

.table-tag.removed {
  background: rgba(244, 67, 54, 0.15);
  color: #f44336;
  border: 1px solid rgba(244, 67, 54, 0.3);
}

.table-tag.added {
  background: rgba(76, 175, 80, 0.15);
  color: #4caf50;
  border: 1px solid rgba(76, 175, 80, 0.3);
}

.table-diff-card {
  background: rgba(20, 20, 20, 0.6);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
}

.table-diff-card:last-child {
  margin-bottom: 0;
}

.table-name {
  font-weight: 600;
  font-size: 14px;
  color: #fff;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(60, 60, 60, 0.5);
}

.column-section {
  margin-bottom: 12px;
}

.column-section:last-child {
  margin-bottom: 0;
}

.section-label {
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 8px;
  padding: 4px 8px;
  border-radius: 4px;
  display: inline-block;
}

.section-label.added { background: rgba(76, 175, 80, 0.2); color: #4caf50; }
.section-label.removed { background: rgba(244, 67, 54, 0.2); color: #f44336; }
.section-label.modified { background: rgba(255, 152, 0, 0.2); color: #ff9800; }

.column-table {
  width: 100%;
  font-size: 12px;
}

.column-table tr {
  border-bottom: 1px solid rgba(40, 40, 40, 0.5);
}

.column-table td {
  padding: 8px 12px;
}

.col-name {
  font-weight: 600;
  color: #fff;
  width: 150px;
}

.col-type {
  color: #888;
  font-family: monospace;
  width: 150px;
}

.col-nullable {
  color: #67c23a;
  font-size: 11px;
}

.col-changes {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.change-item {
  color: #888;
}

.change-item .from {
  color: #f44336;
  text-decoration: line-through;
}

.change-item .to {
  color: #4caf50;
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #555;
  font-size: 16px;
}
</style>
