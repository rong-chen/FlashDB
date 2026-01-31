<script setup>
import { computed } from 'vue'

const props = defineProps({
  tabs: {
    type: Array,
    default: () => []
  },
  activeId: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:activeId', 'add', 'close'])

function selectTab(id) {
  emit('update:activeId', id)
}

function addTab() {
  emit('add')
}

function closeTab(id, e) {
  e.stopPropagation()
  emit('close', id)
}
</script>

<template>
  <div class="tab-bar">
    <div class="tabs-container">
      <div
        v-for="tab in tabs"
        :key="tab.id"
        class="tab-item"
        :class="{ active: activeId === tab.id }"
        @click="selectTab(tab.id)"
      >
        <span class="tab-title">{{ tab.title }}</span>
        <button 
          class="close-btn" 
          @click="closeTab(tab.id, $event)"
          v-if="tabs.length > 1"
        >
          ×
        </button>
      </div>
    </div>
    <button class="add-btn" @click="addTab">
      <span>+</span>
    </button>
  </div>
</template>

<style scoped>
.tab-bar {
  display: flex;
  align-items: center;
  background: rgba(15, 15, 15, 0.9);
  padding: 0 8px;
  height: 36px;
  gap: 4px;
}

.tabs-container {
  display: flex;
  flex: 1;
  overflow-x: auto;
  gap: 2px;
}

.tabs-container::-webkit-scrollbar {
  display: none;
}

.tab-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: rgba(30, 30, 30, 0.8);
  border-radius: 6px 6px 0 0;
  cursor: pointer;
  font-size: 12px;
  color: #888;
  white-space: nowrap;
  transition: all 0.15s;
}

.tab-item:hover {
  background: rgba(50, 50, 50, 0.8);
  color: #aaa;
}

.tab-item.active {
  background: #1e1e1e;
  color: #fff;
}

.tab-title {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: none;
  background: transparent;
  color: #666;
  font-size: 14px;
  cursor: pointer;
  border-radius: 3px;
}

.close-btn:hover {
  background: rgba(255, 100, 100, 0.3);
  color: #ff6b6b;
}

.add-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: #666;
  font-size: 18px;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.15s;
}

.add-btn:hover {
  background: rgba(109, 213, 237, 0.2);
  color: #6dd5ed;
}
</style>
