<script setup>
import { ref, defineProps, defineEmits } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Connection, CircleCheck, MoreFilled, Plus 
} from '@element-plus/icons-vue'
import {
  ListConnections,
  CreateConnection,
  UpdateConnection,
  DeleteConnection,
  TestConnection,
  Connect,
  Disconnect
} from '../../wailsjs/go/services/ConnectionService'

const props = defineProps({
  activeConnection: Object,
  loadingConnections: Boolean
})

const emit = defineEmits([
  'update:activeConnection',
  'connected',
  'disconnected',
  'connections-loaded'
])

// State
const connections = ref([])
const showConnectionForm = ref(false)
const isEditing = ref(false)
const connectionForm = ref({
  name: '',
  type: 'mysql',
  host: 'localhost',
  port: 3306,
  username: 'root',
  password: '',
  database: ''
})

// Load connections on mount
async function loadConnections() {
  try {
    const res = await ListConnections()
    if (res.success) {
      connections.value = res.data || []
      emit('connections-loaded', connections.value)
    }
  } catch (err) {
    console.error('Failed to load connections:', err)
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
      if (props.activeConnection?.id === conn.id) {
        emit('update:activeConnection', null)
      }
      emit('disconnected', conn)
      ElMessage.success('已断开连接')
    }
  } else {
    try {
      const res = await Connect(conn.id)
      if (res.success) {
        conn.connected = true
        emit('update:activeConnection', conn)
        emit('connected', conn)
        ElMessage.success('连接成功!')
      } else {
        ElMessage.error('连接失败: ' + res.error)
      }
    } catch (err) {
      ElMessage.error('连接失败: ' + err.message)
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
      if (props.activeConnection?.id === conn.id) {
        emit('update:activeConnection', null)
        emit('disconnected', conn)
      }
    }
  } catch (err) {}
}

// Expose for parent
defineExpose({
  loadConnections
})

// Initial load
loadConnections()
</script>

<template>
  <div class="connection-panel">
    <!-- Connection Header -->
    <div class="connection-header">
      <span class="section-title">连接</span>
      <el-button type="primary" size="small" circle @click="openNewConnectionForm">
        <el-icon><Plus /></el-icon>
      </el-button>
    </div>
    
    <!-- Connection List -->
    <div class="connection-list" v-loading="loadingConnections">
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
.connection-panel {
  display: flex;
  flex-direction: column;
}

.connection-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid #222;
}

.section-title {
  font-size: 12px;
  font-weight: 500;
  color: #888;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.connection-list {
  max-height: 200px;
  overflow-y: auto;
  padding: 8px 12px;
}

.connection-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.connection-item:hover {
  background: #1a1a1a;
}

.connection-item.active {
  background: #252525;
}

.connection-item.connected .icon {
  color: #67c23a;
}

.connection-item .name {
  flex: 1;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #666;
  gap: 8px;
}

.empty-state.small {
  padding: 32px 20px;
}
</style>
