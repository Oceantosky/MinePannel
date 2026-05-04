<template>
  <div class="flex-1 flex flex-col overflow-hidden relative">
    <!-- Ambient Background Effects -->
    <div class="absolute top-0 right-0 -mr-48 -mt-48 w-[40rem] h-[40rem] bg-indigo-500/10 rounded-full blur-3xl pointer-events-none opacity-50"></div>
    <div class="absolute bottom-0 left-0 -ml-48 -mb-48 w-[40rem] h-[40rem] bg-teal-500/10 rounded-full blur-3xl pointer-events-none opacity-50"></div>

    <header class="h-24 border-b border-slate-800 bg-slate-900/60 backdrop-blur-2xl flex items-center justify-between px-10 z-10 sticky top-0 shadow-sm">
      <div>
        <h2 class="text-2xl font-black bg-gradient-to-r from-slate-100 to-slate-400 bg-clip-text text-transparent tracking-tight">实例控制台</h2>
        <p class="text-sm text-slate-500 mt-1 font-medium">全局资源概览与集群状态</p>
      </div>
      <div class="flex space-x-4">
        <button @click="fetchInstances" class="bg-slate-800 hover:bg-slate-700 text-slate-200 px-5 py-2.5 rounded-xl font-medium transition-all shadow-sm flex items-center border border-slate-700/50">
          <svg class="w-5 h-5 mr-2 text-slate-400" :class="{ 'animate-spin': loading }" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
          刷新状态
        </button>
        <button @click="showCreateModal = true" class="bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white px-6 py-2.5 rounded-xl font-semibold transition-all shadow-[0_0_20px_rgba(79,70,229,0.3)] flex items-center transform hover:-translate-y-0.5 border border-blue-400/20">
          <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path></svg>
          新建实例
        </button>
      </div>
    </header>

    <div class="flex-1 overflow-auto p-10 z-10 space-y-10">
      <section class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div class="bg-slate-800/40 backdrop-blur-md border border-slate-700/50 rounded-2xl p-6 relative overflow-hidden group">
          <p class="text-slate-400 text-sm font-semibold uppercase tracking-wider mb-2">活跃实例</p>
          <div class="flex items-baseline space-x-2"><h3 class="text-4xl font-black text-slate-100">{{ instances.filter(i => i.status !== '暂停').length }}</h3><span class="text-slate-500 font-medium">/ {{ instances.length }}</span></div>
        </div>
        <div class="bg-slate-800/40 backdrop-blur-md border border-slate-700/50 rounded-2xl p-6 relative overflow-hidden group"><p class="text-slate-400 text-sm font-semibold mb-2">总并发玩家</p><h3 class="text-4xl font-black text-slate-100">{{ stats.active_players }}</h3></div>
        <div class="bg-slate-800/40 backdrop-blur-md border border-slate-700/50 rounded-2xl p-6 relative overflow-hidden group"><p class="text-slate-400 text-sm font-semibold mb-2">系统负载</p><h3 class="text-4xl font-black text-slate-100">{{ stats.memory_usage_pct.toFixed(1) }}%</h3><p class="text-xs text-slate-500 mt-1">{{ stats.memory_usage_mb.toFixed(0) }} / {{ stats.memory_total_mb.toFixed(0) }} MB</p></div>
        <div class="bg-slate-800/40 backdrop-blur-md border border-slate-700/50 rounded-2xl p-6 relative overflow-hidden group"><p class="text-slate-400 text-sm font-semibold mb-2">出站流量</p><h3 class="text-4xl font-black text-slate-100">{{ stats.response_mb.toFixed(2) }}</h3><p class="text-xs text-slate-500 mt-1">累计 MB</p></div>
      </section>

      <section>
        <div class="flex items-center justify-between mb-6"><h3 class="text-xl font-bold text-slate-200">运行中实例</h3></div>
        <div v-if="loading" class="grid grid-cols-1 xl:grid-cols-2 2xl:grid-cols-3 gap-6 animate-pulse">
           <div v-for="i in 3" :key="i" class="bg-slate-800/40 h-64 rounded-2xl border border-slate-700/50"></div>
        </div>
        <div v-else class="grid grid-cols-1 xl:grid-cols-2 2xl:grid-cols-3 gap-6">
          <div v-for="instance in instances" :key="instance.id" class="bg-slate-800/60 backdrop-blur-xl border border-slate-700/50 rounded-2xl overflow-hidden hover:border-blue-500/50 hover:shadow-[0_0_30px_rgba(59,130,246,0.15)] transition-all duration-300 group flex flex-col relative" :class="{ 'grayscale-[0.5] opacity-80': instance.status === '暂停' }">
            <div class="absolute top-4 right-4 flex space-x-1 z-20 opacity-0 group-hover:opacity-100 transition-opacity">
              <button v-if="currentUser?.role === 'admin'" @click.stop="openPermModal(instance)" class="p-2 text-slate-500 hover:text-amber-400 hover:bg-amber-500/10 rounded-lg transition-all" title="管理权限">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"></path></svg>
              </button>
              <button @click="$emit('navigate', 'instance-settings', { instanceId: instance.id })" class="p-2 text-slate-500 hover:text-blue-400 hover:bg-blue-500/10 rounded-lg transition-all" title="实例设置"><svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path></svg></button>
            </div>
            <div class="p-6 border-b border-slate-700/50 flex justify-between items-start bg-gradient-to-b from-slate-800/80 to-transparent"><div class="flex items-center space-x-4"><div class="w-14 h-14 rounded-2xl bg-gradient-to-br from-blue-500 to-indigo-600 p-0.5 shadow-lg shadow-blue-500/30 group-hover:scale-105 transition-transform"><div class="w-full h-full bg-slate-900 rounded-2xl flex items-center justify-center"><span class="text-2xl">🌍</span></div></div><div><div class="flex items-center"><h3 class="text-lg font-bold text-slate-100 group-hover:text-blue-400 transition-colors">{{ instance.display_name }}</h3><span class="ml-3 relative flex h-3 w-3"><span v-if="instance.status === '正常'" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span><span class="relative inline-flex rounded-full h-3 w-3" :class="instance.status === '正常' ? 'bg-emerald-500' : 'bg-slate-500'"></span></span></div><p class="text-xs text-slate-400 mt-1 font-mono bg-slate-900/50 px-2 py-0.5 rounded inline-block uppercase tracking-tighter">HASH: {{ instance.id }}</p></div></div></div>
            <div class="p-6 flex-1 flex flex-col justify-center"><p class="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-1">当前业务状态</p><p class="text-2xl font-black tracking-tight transition-colors" :class="instance.status === '正常' ? 'text-emerald-400' : 'text-slate-400'">{{ instance.status }}</p><div class="grid grid-cols-1 gap-3 mt-6"><div class="bg-slate-900/60 rounded-xl p-3 border border-slate-700/50"><p class="text-[10px] text-slate-500 uppercase mb-1 font-semibold">分发链路</p><p class="text-xs text-slate-300 font-mono truncate">{{ instance.full_pack_url }}</p></div></div></div>
            <div class="p-4 bg-slate-900/40 border-t border-slate-700/50 flex gap-2"><button @click="$emit('navigate', 'console', { instanceId: instance.id })" class="flex-1 bg-slate-700/50 hover:bg-slate-600 text-slate-200 py-2 rounded-lg text-sm font-medium transition-colors flex items-center justify-center border border-slate-600/50"><svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path></svg>控制台</button><button @click="$emit('navigate', 'explorer', { instanceId: instance.id })" class="flex-1 bg-slate-700/50 hover:bg-slate-600 text-slate-200 py-2 rounded-lg text-sm font-medium transition-colors flex items-center justify-center border border-slate-600/50"><svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"></path></svg>文件</button><button @click="triggerPacketUp(instance.id)" class="flex-[1.5] bg-blue-500/10 hover:bg-blue-500/20 text-blue-400 hover:text-blue-300 py-2 rounded-lg text-sm font-bold transition-all flex items-center justify-center border border-blue-500/30" :disabled="instance.status === '暂停'"><svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path></svg>分发同步</button></div>
          </div>
          <div @click="showCreateModal = true" class="bg-slate-800/10 border-2 border-slate-700/50 border-dashed rounded-2xl flex flex-col items-center justify-center text-slate-500 hover:bg-slate-800/30 hover:text-slate-300 hover:border-slate-600 transition-all cursor-pointer group min-h-[300px]"><svg class="w-10 h-10 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path></svg><p class="font-bold">接入新服务器实例</p></div>
        </div>
      </section>
    </div>

    <!-- Create Instance Modal -->
    <div v-if="showCreateModal" class="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div class="absolute inset-0 bg-slate-900/80 backdrop-blur-md" @click="showCreateModal = false"></div>
      <div class="bg-slate-800 border border-slate-700 w-full max-w-md rounded-2xl shadow-2xl z-10 overflow-hidden transform transition-all">
        <div class="p-6 border-b border-slate-700 font-bold text-slate-100 text-center text-xl">新实例接入</div>
        <div class="p-8 space-y-4 text-center">
          <div class="w-16 h-16 bg-blue-500/10 rounded-full flex items-center justify-center mx-auto mb-2 text-blue-400"><svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path></svg></div>
          <div class="space-y-2"><label class="text-[10px] font-bold text-slate-500 uppercase tracking-widest">请指定实例显示名称</label><input v-model="newInstanceName" type="text" placeholder="例如：1.20 极限生存" class="w-full bg-slate-900 border border-slate-700 rounded-xl px-4 py-3.5 text-center text-slate-100 text-lg outline-none focus:border-blue-500 transition-colors shadow-inner"></div>
          <p class="text-[10px] text-slate-500 italic">唯一识别哈希 (Hash ID) 将由系统自动生成</p>
        </div>
        <div class="p-6 bg-slate-900/50 flex gap-3"><button @click="showCreateModal = false" class="flex-1 px-4 py-3 rounded-xl font-bold text-slate-400 hover:bg-slate-800 transition-colors">取消</button><button @click="createInstance" :disabled="creating" class="flex-[2] bg-blue-600 hover:bg-blue-500 text-white py-3 rounded-xl font-bold transition-all shadow-lg shadow-blue-600/20 disabled:opacity-50">{{ creating ? '分配空间中...' : '开始创建' }}</button></div>
      </div>
    </div>

    <!-- Permission Management Modal -->
    <div v-if="showPermModal" class="fixed inset-0 z-[100] flex items-center justify-center p-4">
      <div class="absolute inset-0 bg-slate-950/80 backdrop-blur-xl" @click="showPermModal = false"></div>
      <div class="bg-slate-900 border border-white/10 w-full max-w-lg rounded-3xl shadow-2xl z-10 overflow-hidden">
        <div class="p-6 border-b border-slate-700/50">
          <h3 class="text-lg font-black text-white">实例权限管理</h3>
          <p class="text-sm text-slate-400 mt-1">为 [{{ permInstanceName }}] 分配用户访问权限。管理员自动拥有所有实例的访问权限。</p>
        </div>
        <div class="p-6 max-h-80 overflow-y-auto space-y-2">
          <div v-if="permLoading" class="text-center text-slate-500 text-sm py-8">加载用户列表...</div>
          <div v-for="user in permUsers" :key="user.id" class="flex items-center justify-between px-3 py-2.5 rounded-xl hover:bg-slate-800/50 transition-colors">
            <div class="flex items-center space-x-3">
              <div class="w-8 h-8 rounded-full bg-slate-800 border border-slate-700 flex items-center justify-center text-xs font-bold"
                :class="user.role === 'admin' ? 'text-amber-400' : 'text-slate-400'">
                {{ user.username[0].toUpperCase() }}
              </div>
              <div>
                <p class="text-sm font-bold text-slate-200">{{ user.username }}</p>
                <p class="text-[10px] text-slate-500">{{ user.role === 'admin' ? '管理员 (自动全权限)' : '普通用户' }}</p>
              </div>
            </div>
            <button v-if="user.role !== 'admin'"
              @click="toggleUserAccess(user)"
              class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all"
              :class="user.has_access ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 hover:bg-emerald-500/20' : 'bg-slate-700/50 text-slate-500 border border-slate-700 hover:bg-slate-600'">
              {{ user.has_access ? '已授权' : '未授权' }}
            </button>
            <span v-else class="px-3 py-1.5 rounded-lg text-xs font-bold bg-amber-500/10 text-amber-400 border border-amber-500/20">全部实例</span>
          </div>
        </div>
        <div class="p-6 bg-slate-900/50 border-t border-slate-700/50 flex justify-end">
          <button @click="showPermModal = false" class="px-6 py-2.5 rounded-xl font-bold text-slate-400 hover:bg-slate-800 transition-colors border border-slate-800">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { apiGetInstances, apiCreateInstance, apiPacketUp, apiListUsers, apiUpdateUser, apiGetStats, getCurrentUser } from '../api';

const emit = defineEmits(['navigate']);
const instances = ref<any[]>([]);
const loading = ref(true);
const showCreateModal = ref(false);
const creating = ref(false);
const newInstanceName = ref('');

const currentUser = computed(() => getCurrentUser());
const showPermModal = ref(false);
const permInstanceId = ref('');
const permInstanceName = ref('');
const permUsers = ref<any[]>([]);
const permLoading = ref(false);

async function fetchInstances() {
  loading.value = true;
  try {
    const data = await apiGetInstances();
    instances.value = data.data?.instance_list || data.instance_list || [];
  } catch (e) { console.error(e); } finally { loading.value = false; }
}

async function createInstance() {
  if (!newInstanceName.value.trim()) return;
  creating.value = true;
  try {
    await apiCreateInstance(newInstanceName.value);
    showCreateModal.value = false; newInstanceName.value = ''; fetchInstances();
  } catch (e) { console.error(e); } finally { creating.value = false; }
}

async function triggerPacketUp(id: string) {
  try {
    await apiPacketUp(id);
    emit('navigate', 'console', { instanceId: id });
  } catch (e) { console.error(e); }
}

async function openPermModal(instance: any) {
  permInstanceId.value = instance.id;
  permInstanceName.value = instance.display_name || instance.id;
  showPermModal.value = true;
  permLoading.value = true;
  try {
    const data = await apiListUsers();
    const list = data.data ?? data;
    const usersList = Array.isArray(list) ? list : [];
    permUsers.value = usersList.map((u: any) => ({
      ...u,
      has_access: u.role === 'admin' || (u.instances || []).includes(instance.id),
    }));
  } catch (e) { console.error(e); } finally { permLoading.value = false; }
}

async function toggleUserAccess(user: any) {
  const instances = [...(user.instances || [])];
  if (user.has_access) {
    const idx = instances.indexOf(permInstanceId.value);
    if (idx !== -1) instances.splice(idx, 1);
  } else {
    instances.push(permInstanceId.value);
  }
  try {
    await apiUpdateUser(user.id, user.username, user.role, user.is_active, instances);
    user.has_access = !user.has_access;
    user.instances = instances;
  } catch (e: any) {
    alert(e.message || '更新失败');
  }
}

interface SystemStats {
  active_players: number
  memory_usage_mb: number
  memory_total_mb: number
  memory_usage_pct: number
  goroutines: number
  uptime_seconds: number
  response_mb: number
}

const stats = ref<SystemStats>({
  active_players: 0,
  memory_usage_mb: 0,
  memory_total_mb: 0,
  memory_usage_pct: 0,
  goroutines: 0,
  uptime_seconds: 0,
  response_mb: 0,
})

let statsTimer: ReturnType<typeof setInterval> | null = null

async function fetchStats() {
  try {
    const data = await apiGetStats()
    if (data.success && data.data) {
      stats.value = { ...stats.value, ...data.data }
    }
  } catch (e) { /* stats fetch is non-critical */ }
}

onMounted(() => {
  fetchInstances()
  fetchStats()
  statsTimer = setInterval(fetchStats, 5000)
})

onUnmounted(() => {
  if (statsTimer) {
    clearInterval(statsTimer)
    statsTimer = null
  }
})
</script>
