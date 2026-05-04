<template>
  <div class="flex-1 flex flex-col overflow-hidden relative">
    <!-- Ambient Background -->
    <div class="absolute top-0 left-0 -ml-32 -mt-32 w-96 h-96 bg-blue-500/10 rounded-full blur-3xl pointer-events-none"></div>

    <header class="h-24 border-b border-slate-800 bg-slate-900/60 backdrop-blur-2xl flex items-center justify-between px-10 z-10">
      <div>
        <h2 class="text-2xl font-black bg-gradient-to-r from-slate-100 to-slate-400 bg-clip-text text-transparent tracking-tight">边缘节点监控</h2>
        <p class="text-sm text-slate-500 mt-1 font-medium">分布式分发网络 (EDN) 边缘缓存节点状态</p>
      </div>
      <button @click="fetchConfig" class="bg-slate-800 hover:bg-slate-700 text-slate-200 px-5 py-2.5 rounded-xl font-medium transition-all shadow-sm flex items-center border border-slate-700/50">
        <svg class="w-5 h-5 mr-2 text-slate-400" :class="{ 'animate-spin': loading }" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
        刷新节点
      </button>
    </header>

    <div class="flex-1 overflow-auto p-10 z-10 space-y-8">
      <!-- Cluster Map/Status Overview -->
      <section class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div v-for="node in edgeNodes" :key="node.name" class="bg-slate-800/40 backdrop-blur-md border border-slate-700/50 rounded-2xl p-6 hover:border-blue-500/30 transition-all group relative">
          
          <!-- Delete Node Button (Float Top-Right) -->
          <button 
            @click="deleteNode(node.name)"
            class="absolute top-4 right-4 p-2 text-slate-600 hover:text-rose-500 hover:bg-rose-500/10 rounded-lg transition-all z-20 opacity-0 group-hover:opacity-100"
            title="移除节点"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
          </button>

          <div class="flex justify-between items-start mb-6">
            <div class="flex items-center space-x-4">
              <div class="w-12 h-12 rounded-xl bg-slate-900 flex items-center justify-center text-2xl shadow-inner border border-white/5">
                {{ node.flag }}
              </div>
              <div>
                <h3 class="font-bold text-slate-100">{{ node.display_name }}</h3>
                <p class="text-xs text-slate-500 font-mono">{{ node.ip }}</p>
              </div>
            </div>
            <span 
              v-if="nodeStatuses[node.name]?.online"
              class="px-2 py-1 rounded-md text-[10px] font-bold tracking-widest uppercase bg-emerald-500/10 text-emerald-400">
              ONLINE
            </span>
            <span 
              v-else
              class="px-2 py-1 rounded-md text-[10px] font-bold tracking-widest uppercase bg-rose-500/10 text-rose-400">
              OFFLINE
            </span>
          </div>

          <div class="space-y-4">
            <div>
              <div class="flex justify-between text-xs mb-1.5">
                <span class="text-slate-500">同步进度</span>
                <span class="text-slate-300 font-mono">{{ nodeStatuses[node.name]?.online ? '100%' : '0%' }}</span>
              </div>
              <div class="w-full bg-slate-900/50 rounded-full h-1.5 overflow-hidden">
                <div class="bg-blue-500 h-1.5 rounded-full w-full" :style="{ width: nodeStatuses[node.name]?.online ? '100%' : '0%' }"></div>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-4 pt-2">
              <div class="bg-black/20 rounded-lg p-3 border border-white/5">
                <p class="text-[10px] text-slate-500 uppercase font-bold mb-1">响应延迟</p>
                <p class="text-sm font-mono text-blue-400" :class="{'text-rose-400': !nodeStatuses[node.name]?.online}">
                  {{ nodeStatuses[node.name]?.online ? nodeStatuses[node.name]?.latency + 'ms' : '--' }}
                </p>
              </div>
              <div class="bg-black/20 rounded-lg p-3 border border-white/5">
                <p class="text-[10px] text-slate-500 uppercase font-bold mb-1">节点角色</p>
                <p class="text-sm font-mono text-emerald-400 capitalize">{{ node.name.includes('master') ? 'Master' : 'Slave' }}</p>
              </div>
            </div>
          </div>

          <div class="mt-6 pt-6 border-t border-slate-700/50 flex space-x-3">
            <button @click="showDiagnosticInfo" class="flex-1 bg-slate-700/50 hover:bg-slate-600 text-slate-200 py-2 rounded-lg text-xs font-bold transition-colors">
              连接诊断
            </button>
            <button @click="showForceSyncInfo" class="flex-1 bg-blue-500/10 hover:bg-blue-500/20 text-blue-400 py-2 rounded-lg text-xs font-bold transition-colors border border-blue-500/20">
              强制同步
            </button>
          </div>
        </div>

        <!-- Add Node Placeholder -->
        <div @click="showAddNodeModal = true" class="bg-slate-800/10 border-2 border-slate-700/50 border-dashed rounded-2xl flex flex-col items-center justify-center text-slate-500 hover:bg-slate-800/30 hover:text-slate-300 transition-all cursor-pointer group min-h-[280px]">
           <svg class="w-8 h-8 mb-3 opacity-50 group-hover:scale-110 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path></svg>
           <p class="font-bold text-sm">部署新边缘节点</p>
        </div>
      </section>
    </div>

    <!-- Add Node Modal -->
    <div v-if="showAddNodeModal" class="fixed inset-0 z-[100] flex items-center justify-center p-4">
      <div class="absolute inset-0 bg-slate-950/80 backdrop-blur-xl" @click="showAddNodeModal = false"></div>
      <div class="bg-slate-900 border border-white/10 w-full max-w-md rounded-3xl shadow-2xl z-10 overflow-hidden">
        <div class="p-6 space-y-4">
          <h3 class="text-lg font-black text-white">部署新边缘节点</h3>
          <div class="space-y-3">
            <div>
              <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">节点标识名</label>
              <input v-model="newNodeForm.name" type="text" placeholder="例如: us_west_node_1" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 outline-none focus:border-blue-500/50 transition-colors mt-1">
            </div>
            <div>
              <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">显示名称</label>
              <input v-model="newNodeForm.display_name" type="text" placeholder="例如: 硅谷缓存节点" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 outline-none focus:border-blue-500/50 transition-colors mt-1">
            </div>
            <div>
              <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">IP 地址或域名</label>
              <input v-model="newNodeForm.ip" type="text" placeholder="例如: 192.168.1.100" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 outline-none focus:border-blue-500/50 transition-colors mt-1">
            </div>
            <div>
              <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">图标</label>
              <input v-model="newNodeForm.flag" type="text" placeholder="例如: 🇺🇸" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 outline-none focus:border-blue-500/50 transition-colors mt-1">
            </div>
          </div>
          <div class="flex gap-3 pt-2">
            <button @click="showAddNodeModal = false" class="flex-1 px-4 py-2.5 rounded-xl font-bold text-slate-400 hover:bg-slate-800 transition-colors border border-slate-800">取消</button>
            <button @click="submitAddNode" :disabled="addingNode" class="flex-[2] bg-blue-600 hover:bg-blue-500 text-white py-2.5 rounded-xl font-bold transition-all disabled:opacity-50">{{ addingNode ? '保存中...' : '确认部署' }}</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { apiReadConfig, apiWriteConfig, apiGetNodesStatus } from '../api';

const loading = ref(false);
const nodes = ref<any[]>([]);
const nodeStatuses = ref<Record<string, any>>({});
let pollInterval: ReturnType<typeof setInterval> | null = null;

const showAddNodeModal = ref(false);
const addingNode = ref(false);
const newNodeForm = ref({ name: '', display_name: '', ip: '', flag: '🌐' });

// 过滤掉本地的主节点，仅在页面上展示需要被监控的远程边缘节点
const edgeNodes = computed(() => {
  return nodes.value.filter(n => !n.display_name?.toLowerCase().includes('master') && n.ip !== '127.0.0.1');
});

async function fetchConfig() {
  loading.value = true;
  try {
    const data = await apiReadConfig();
    const cfg = data.data || data;
    nodes.value = cfg.nodes || [];
    await fetchStatuses();
  } catch (error) {
    console.error('Failed to fetch node config:', error);
  } finally {
    loading.value = false;
  }
}

async function fetchStatuses() {
  try {
    const res = await apiGetNodesStatus();
    const statusMap: Record<string, any> = {};
    for (const s of (res.data || [])) {
      statusMap[s.name] = s;
    }
    nodeStatuses.value = statusMap;
  } catch(e) {
    console.error('Failed to fetch node statuses:', e);
  }
}

async function submitAddNode() {
  if (!newNodeForm.value.name || !newNodeForm.value.ip) {
    alert("节点标识名和 IP 不能为空");
    return;
  }
  addingNode.value = true;
  try {
    const data = await apiReadConfig();
    const config = data.data || data;
    config.nodes = config.nodes || [];
    config.nodes.push({ ...newNodeForm.value });
    await apiWriteConfig(config);
    showAddNodeModal.value = false;
    newNodeForm.value = { name: '', display_name: '', ip: '', flag: '🌐' };
    await fetchConfig();
  } catch (error) {
    console.error('Add node failed:', error);
    alert("添加节点失败");
  } finally {
    addingNode.value = false;
  }
}

async function deleteNode(name: string) {
  if (!confirm(`确定要从集群中移除节点 [${name}] 吗？`)) return;

  try {
    const data = await apiReadConfig();
    const config = data.data || data;
    config.nodes = config.nodes.filter((n: any) => n.name !== name);
    await apiWriteConfig(config);
    alert('节点已移除');
    fetchConfig();
  } catch (error) {
    console.error('Delete node failed:', error);
  }
}

function showDiagnosticInfo() {
  alert("【连接诊断】功能开发中\n\n未来此功能将深入探测边缘节点的健康状况，包括：\n- 55000 控制端口和 55001 数据端口的双向连通性测试\n- 响应延迟波动（Jitter）及丢包率分析\n- TLS 证书有效性及到期时间监控\n- 带宽负载情况");
}

function showForceSyncInfo() {
  alert("【强制同步】功能开发中\n\n未来此功能将允许在忽略主动版本号对比的情况下，强行下发覆盖指令到边缘节点。这通常用于节点文件意外损坏或缓存未命中的紧急恢复。");
}

onMounted(() => {
  fetchConfig();
  // 开启每 5 秒一次的轮询
  pollInterval = setInterval(() => {
    fetchStatuses();
  }, 5000);
});

onUnmounted(() => {
  // 当用户离开这个组件（监控页面）时，清除轮询定时器
  if (pollInterval) {
    clearInterval(pollInterval);
    pollInterval = null;
  }
});
</script>
