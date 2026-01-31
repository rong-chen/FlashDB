<script setup>
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import loader from '@monaco-editor/loader'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:modelValue', 'execute'])

const editorContainer = ref(null)
let monacoEditor = null
let monaco = null

onMounted(async () => {
  loader.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs' } })
  monaco = await loader.init()
  
  await nextTick()
  if (editorContainer.value) {
    monacoEditor = monaco.editor.create(editorContainer.value, {
      value: props.modelValue,
      language: 'sql',
      theme: 'vs-dark',
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: 13,
      fontFamily: "'SF Mono', Monaco, Consolas, monospace",
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      wordWrap: 'on',
      tabSize: 2,
      padding: { top: 12 }
    })
    
    // Sync content changes
    monacoEditor.onDidChangeModelContent(() => {
      emit('update:modelValue', monacoEditor.getValue())
    })
    
    // Cmd/Ctrl + Enter to execute
    monacoEditor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
      emit('execute')
    })
  }
})

onUnmounted(() => {
  if (monacoEditor) {
    monacoEditor.dispose()
    monacoEditor = null
  }
})

// Watch for external value changes
watch(() => props.modelValue, (newVal) => {
  if (monacoEditor && monacoEditor.getValue() !== newVal) {
    monacoEditor.setValue(newVal)
  }
})

// Get selected text or full content
function getSelectedText() {
  if (!monacoEditor) return props.modelValue
  const selection = monacoEditor.getSelection()
  const model = monacoEditor.getModel()
  if (selection && !selection.isEmpty()) {
    return model.getValueInRange(selection)
  }
  return monacoEditor.getValue()
}

// Expose methods
defineExpose({
  getSelectedText
})
</script>

<template>
  <div class="sql-editor-container">
    <div ref="editorContainer" class="editor-content"></div>
  </div>
</template>

<style scoped>
.sql-editor-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
}

.editor-content {
  flex: 1;
  min-height: 100px;
}
</style>
