<template>
  <div class="flex-1 flex flex-col overflow-hidden relative bg-[#1e1e1e]">
    
    <!-- Editor Header -->
    <header class="h-14 border-b border-slate-800 bg-slate-900 flex items-center justify-between px-6 z-10 shrink-0">
      <div class="flex items-center">
        <button @click="$emit('navigate', 'explorer')" class="text-slate-400 hover:text-white transition-colors mr-4 p-1 rounded hover:bg-slate-800">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path></svg>
        </button>
        <svg class="w-5 h-5 text-purple-400 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
        <span class="font-mono text-sm text-slate-200">
          {{ editorData?.path || 'Untitled' }} 
          <span v-if="isDiffMode" class="ml-2 text-xs text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded border border-amber-500/20">Audit Mode (差异审查)</span>
          <span v-else-if="isModified" class="text-slate-400 ml-1">*</span>
        </span>
      </div>
      
      <div class="flex items-center space-x-3">
        <span v-if="saved" class="text-xs text-emerald-400 flex items-center bg-emerald-500/10 px-2 py-1 rounded-md transition-opacity duration-500">
          <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>
          已保存
        </span>
        
        <template v-if="!isDiffMode">
          <button 
            @click="requestSave" 
            :disabled="!isModified"
            class="bg-blue-600 hover:bg-blue-500 disabled:opacity-50 disabled:hover:bg-blue-600 text-white px-4 py-1.5 rounded-md text-sm font-medium transition-colors shadow shadow-blue-500/20 flex items-center"
          >
            <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"></path></svg>
            审查并保存 (Ctrl+S)
          </button>
        </template>
        <template v-else>
          <button 
            @click="isDiffMode = false" 
            class="bg-slate-700 hover:bg-slate-600 text-white px-4 py-1.5 rounded-md text-sm font-medium transition-colors"
          >
            取消
          </button>
          <button 
            @click="confirmSave" 
            :disabled="saving"
            class="bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white px-4 py-1.5 rounded-md text-sm font-medium transition-colors shadow shadow-emerald-500/20 flex items-center"
          >
            <svg v-if="saving" class="w-4 h-4 mr-1.5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
            <svg v-else class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>
            {{ saving ? '正在提交...' : '确认修改' }}
          </button>
        </template>
      </div>
    </header>

    <!-- Editor Area -->
    <div class="flex-1 relative z-0" @keydown.ctrl.s.prevent="requestSave">
      <template v-if="isDiffMode">
        <vue-monaco-diff-editor
          theme="vs-dark"
          :original="props.editorData?.content || ''"
          :modified="localContent"
          :language="editorLanguage"
          :options="diffOptions"
          @mount="handleDiffMount"
          class="h-full w-full"
        />
      </template>
      <template v-else>
        <vue-monaco-editor
          v-model:value="localContent"
          theme="vs-dark"
          :language="editorLanguage"
          :options="editorOptions"
          @mount="handleEditorMount"
          class="h-full w-full"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, shallowRef } from 'vue';
import { apiWriteFile } from '../api';
import { VueMonacoEditor, VueMonacoDiffEditor } from '@guolao/vue-monaco-editor';

const props = defineProps<{
  editorData?: { instanceId: string, path: string, content: string }
}>();

const emit = defineEmits(['navigate']);

const localContent = ref(props.editorData?.content || '');
const saving = ref(false);
const saved = ref(false);
const isDiffMode = ref(false);

const editorRef = shallowRef();
const diffEditorRef = shallowRef();

const isModified = computed(() => {
  return localContent.value !== (props.editorData?.content || '');
});

// Infer language from file extension
const editorLanguage = computed(() => {
  const path = props.editorData?.path || '';
  const ext = path.split('.').pop()?.toLowerCase();
  switch (ext) {
    case 'json': return 'json';
    case 'js':
    case 'mjs':
    case 'cjs': return 'javascript';
    case 'ts': return 'typescript';
    case 'html':
    case 'htm': return 'html';
    case 'css': return 'css';
    case 'yml':
    case 'yaml': return 'yaml';
    case 'xml': return 'xml';
    case 'md':
    case 'markdown': return 'markdown';
    case 'java': return 'java';
    case 'toml': return 'ini'; // Monaco uses ini for basic TOML-like files if no extension installed
    case 'properties': return 'ini';
    case 'sh':
    case 'bash': return 'shell';
    case 'bat':
    case 'cmd': return 'bat';
    case 'ini':
    case 'cfg': return 'ini';
    default: return 'plaintext';
  }
});

const editorOptions = {
  automaticLayout: true,
  formatOnType: true,
  formatOnPaste: true,
  minimap: { enabled: false },
  wordWrap: 'on',
  fontSize: 14,
  fontFamily: "'JetBrains Mono', 'Fira Code', Consolas, monospace",
  scrollBeyondLastLine: false,
  padding: { top: 16 }
};

const diffOptions = {
  automaticLayout: true,
  minimap: { enabled: false },
  renderSideBySide: true,
  readOnly: true, // Diff view is for audit only
  fontSize: 14,
  fontFamily: "'JetBrains Mono', 'Fira Code', Consolas, monospace",
  scrollBeyondLastLine: false,
  padding: { top: 16 }
};

const handleEditorMount = (editor: any) => {
  editorRef.value = editor;
};

const handleDiffMount = (diffEditor: any) => {
  diffEditorRef.value = diffEditor;
};

// Reset state when file changes
watch(() => props.editorData?.path, () => {
  localContent.value = props.editorData?.content || '';
  saved.value = false;
  isDiffMode.value = false;
});

// Update local content if original content is fetched again but path is the same
watch(() => props.editorData?.content, (newContent) => {
  if (!isModified.value) {
    localContent.value = newContent || '';
  }
});

function requestSave() {
  if (!isModified.value) return;
  // Enter diff mode to audit changes
  isDiffMode.value = true;
}

async function confirmSave() {
  if (!props.editorData || saving.value) return;

  saving.value = true;
  saved.value = false;

  try {
    await apiWriteFile(props.editorData.instanceId, props.editorData.path, localContent.value);
    
    // Mutate props if possible or emit an event to update parent state,
    // For now we assume parent fetches again or we just update our reference point
    if (props.editorData) {
      props.editorData.content = localContent.value;
    }
    
    saved.value = true;
    isDiffMode.value = false; // Exit diff mode on success
    
    setTimeout(() => { saved.value = false; }, 3000);
  } catch (error: any) {
    console.error('Save failed:', error);
    alert('保存失败：' + (error.message || '网络错误'));
  } finally {
    saving.value = false;
  }
}
</script>
