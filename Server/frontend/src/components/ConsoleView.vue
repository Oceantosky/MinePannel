<template>
  <div class="flex-1 flex flex-col overflow-hidden relative bg-[#0c0c0c]">
    <!-- Console Header -->
    <header class="h-16 border-b border-white/5 bg-black/40 backdrop-blur-md flex items-center justify-between px-8 z-10 sticky top-0">
      <div class="flex items-center">
        <button @click="$emit('navigate', 'portal')" class="text-slate-400 hover:text-white transition-colors mr-4 p-1.5 rounded hover:bg-white/5">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path></svg>
        </button>
        <div class="flex items-center space-x-3">
          <span class="w-2.5 h-2.5 rounded-full" :class="connected ? 'bg-emerald-500 animate-pulse' : 'bg-rose-500'"></span>
          <h2 class="text-slate-200 font-bold tracking-tight">控制台: {{ instanceId }}</h2>
        </div>
      </div>
      
      <div class="flex items-center space-x-4">
        <div class="text-[10px] font-mono text-slate-500 flex flex-col items-end">
          <span>STATUS: {{ connected ? 'CONNECTED' : 'DISCONNECTED' }}</span>
        </div>
      </div>
    </header>

    <!-- Terminal Output -->
    <div ref="terminalRef" class="flex-1 overflow-auto p-6 font-mono text-sm space-y-1.5 scrollbar-thin scrollbar-thumb-white/10">
      <div v-for="(line, index) in logs" :key="index" class="flex">
        <span class="text-slate-600 mr-4 select-none">[{{ line.time }}]</span>
        <span :class="getLogClass(line.level)">
          <span class="font-bold mr-2">[{{ line.level }}]</span>
          {{ line.content }}
        </span>
      </div>
      <div v-if="logs.length === 0" class="text-slate-600 italic">正在等待日志输出...</div>
    </div>

    <!-- Command Input -->
    <div class="p-4 bg-black/60 border-t border-white/5">
      <div class="relative flex items-center group">
        <span class="absolute left-4 text-emerald-500 font-bold select-none group-focus-within:scale-110 transition-transform">></span>
        <input 
          v-model="commandInput"
          @keyup.enter="sendCommand"
          type="text" 
          placeholder="输入指令并回车..."
          class="w-full bg-white/5 border border-white/10 rounded-xl py-3.5 pl-10 pr-4 text-slate-200 font-mono text-sm outline-none focus:border-emerald-500/50 focus:ring-1 focus:ring-emerald-500/30 transition-all"
        >
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue';

const props = defineProps<{
  instanceId: string
}>();

const emit = defineEmits(['navigate', 'refresh-meta']);

const terminalRef = ref<HTMLElement | null>(null);
const commandInput = ref('');
const logs = ref<any[]>([]);
const connected = ref(false);

let eventSource: EventSource | null = null;

function addLog(rawMessage: string) {
  let level = 'INFO';
  let content = rawMessage;

  if (rawMessage.startsWith('ERROR:')) {
    level = 'ERROR';
    content = rawMessage.replace('ERROR:', '').trim();
  } else if (rawMessage.startsWith('WARN:')) {
    level = 'WARN';
    content = rawMessage.replace('WARN:', '').trim();
  } else if (rawMessage.startsWith('SUCCESS:')) {
    level = 'SUCCESS';
    content = rawMessage.replace('SUCCESS:', '').trim();
    emit('refresh-meta');
  } else if (rawMessage.startsWith('DEBUG:')) {
    level = 'DEBUG';
    content = rawMessage.replace('DEBUG:', '').trim();
  }

  logs.value.push({
    time: new Date().toLocaleTimeString(),
    level,
    content
  });
  
  if (logs.value.length > 1000) logs.value.shift();
  
  nextTick(() => {
    if (terminalRef.value) {
      terminalRef.value.scrollTop = terminalRef.value.scrollHeight;
    }
  });
}

function getLogClass(level: string) {
  switch (level) {
    case 'ERROR': return 'text-rose-400';
    case 'WARN': return 'text-amber-300';
    case 'SUCCESS': return 'text-emerald-400';
    case 'DEBUG': return 'text-slate-500';
    default: return 'text-slate-300';
  }
}

function sendCommand() {
  if (!commandInput.value.trim()) return;
  // 这里暂时本地回显，未来可对接 /api/v1/master/console/command
  addLog(`USER: ${commandInput.value}`);
  commandInput.value = '';
}

onMounted(() => {
  const url = `/api/v1/master/console/stream?id=${props.instanceId}`;
  eventSource = new EventSource(url);
  
  eventSource.onopen = () => {
    connected.value = true;
    addLog('SUCCESS: 已建立实时日志连接');
  };

  eventSource.onmessage = (event) => {
    addLog(event.data);
  };

  eventSource.onerror = () => {
    connected.value = false;
    addLog('ERROR: 日志连接已断开，正在尝试重连...');
  };
});

onUnmounted(() => {
  if (eventSource) {
    eventSource.close();
  }
});
</script>

<style scoped>
.scrollbar-thin::-webkit-scrollbar {
  width: 6px;
}
.scrollbar-thin::-webkit-scrollbar-track {
  background: transparent;
}
.scrollbar-thin::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 10px;
}
.scrollbar-thin::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.2);
}
</style>
