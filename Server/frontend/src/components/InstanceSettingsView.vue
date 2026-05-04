<template>
  <div class="flex-1 flex flex-col overflow-hidden relative">
    <div class="absolute top-0 right-0 -mr-32 -mt-32 w-96 h-96 bg-indigo-500/10 rounded-full blur-3xl pointer-events-none"></div>

    <header class="h-24 border-b border-slate-800 bg-slate-900/60 backdrop-blur-2xl flex items-center justify-between px-10 z-10 sticky top-0">
      <div class="flex items-center">
        <button @click="$emit('navigate', 'portal')" class="text-slate-400 hover:text-white transition-colors mr-6 p-2 rounded-xl hover:bg-slate-800 border border-transparent hover:border-slate-700">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path></svg>
        </button>
        <div>
          <h2 class="text-2xl font-black bg-gradient-to-r from-slate-100 to-slate-400 bg-clip-text text-transparent tracking-tight">实例管理</h2>
          <p class="text-sm text-slate-500 mt-1 font-mono uppercase tracking-tighter">ID: {{ instanceId }}</p>
        </div>
      </div>
      <button @click="saveMetadata" :disabled="saving" class="bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white px-8 py-2.5 rounded-xl font-bold transition-all shadow-lg shadow-blue-500/25 flex items-center disabled:opacity-50">
        <svg v-if="saving" class="w-5 h-5 mr-2 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
        <span v-else>保存更改</span>
      </button>
    </header>

    <div class="flex-1 overflow-auto p-10 z-10">
      <div class="max-w-4xl grid grid-cols-1 lg:grid-cols-3 gap-10">
        
        <!-- Left Column: Config -->
        <div class="lg:col-span-2 space-y-12">
          <section class="space-y-6">
            <div class="flex items-center space-x-2 border-l-4 border-blue-500 pl-4"><h3 class="text-lg font-bold text-slate-200">基本信息</h3></div>
            <div class="space-y-4">
              <div class="space-y-2">
                <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">显示名称</label>
                <input v-model="metadata.display_name" type="text" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-4 py-3 text-slate-200 outline-none focus:border-blue-500/50 transition-colors shadow-inner">
              </div>
            </div>
          </section>

          <section class="space-y-6">
            <div class="flex items-center space-x-2 border-l-4 border-amber-500 pl-4"><h3 class="text-lg font-bold text-slate-200">私有云数据面控制</h3></div>
            <div class="bg-slate-800/30 border border-slate-700/50 rounded-2xl p-6">
              <div class="flex items-center justify-between">
                <div><p class="font-bold text-slate-200 text-sm">私有云 55001 端口直连状态</p><p class="text-xs text-slate-500 mt-1">拦截或开启针对此实例的文件直接拉取</p></div>
                <button @click="metadata.is_paused = !metadata.is_paused" :class="metadata.is_paused ? 'bg-slate-700 text-slate-400' : 'bg-emerald-600/20 text-emerald-400 border-emerald-500/30'" class="px-6 py-2 rounded-xl text-xs font-black transition-all border border-transparent">{{ metadata.is_paused ? '已暂停分发' : '服务正常' }}</button>
              </div>
            </div>
          </section>

          <section class="space-y-6">
            <div class="flex items-center space-x-2 border-l-4 border-purple-500 pl-4"><h3 class="text-lg font-bold text-slate-200">外部加速策略</h3></div>
            <div class="bg-slate-800/30 border border-slate-700/50 rounded-2xl p-6">
              <div class="space-y-2 relative" :class="{ 'opacity-60 grayscale-[50%]': globalConfig?.dist_policy !== 'cdn' }">
                <div v-if="globalConfig?.dist_policy !== 'cdn'" class="absolute inset-0 z-10 cursor-not-allowed" title="全局系统设置中的 CDN 策略未开启，此功能已锁定"></div>
                <div class="flex justify-between items-center">
                  <label class="text-[10px] font-bold text-slate-500 uppercase tracking-widest">客户端完整压缩包 CDN 直链</label>
                  <span v-if="globalConfig?.dist_policy !== 'cdn'" class="text-[10px] bg-rose-500/10 text-rose-400 font-black px-2 py-0.5 rounded uppercase flex items-center">
                    <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path></svg>
                    全局已锁定
                  </span>
                </div>
                <input v-model="metadata.cdn_link" :disabled="globalConfig?.dist_policy !== 'cdn'" type="text" placeholder="https://..." class="w-full bg-slate-900 border border-slate-700 rounded-xl px-4 py-2 text-slate-200 outline-none focus:border-purple-500/50 transition-colors font-mono text-xs disabled:opacity-50 disabled:bg-slate-800">
                <p class="text-[10px] text-slate-500 mt-1 leading-relaxed">
                  此链接仅用于接管<strong>初次安装的完整客户端包 (Full Pack)</strong> 的分发，以极大地削减源站带宽成本。为了保证数据一致性，后续的增量同步哈希对象及清单文件仍将强制由云节点 55000/55001 端口直接下发。
                </p>
              </div>
            </div>
          </section>

          <section class="pt-10 border-t border-slate-800/50 space-y-6">
            <div class="flex items-center space-x-2 border-l-4 border-rose-500 pl-4"><h3 class="text-lg font-bold text-rose-500">危险区域</h3></div>
            <div class="bg-rose-500/5 border border-rose-500/20 rounded-2xl p-6 flex items-center justify-between">
              <div><p class="font-bold text-slate-200 text-sm">销毁此实例</p><p class="text-xs text-slate-500 mt-1">此操作将永久抹除所有文件且无法找回</p></div>
              <button @click="deleteInstance" class="bg-rose-600 hover:bg-rose-500 text-white px-6 py-2 rounded-xl text-xs font-black shadow-lg shadow-rose-600/20 transition-all">销毁实例</button>
            </div>
          </section>
        </div>

        <!-- Right Column: Version History (Git Style) -->
        <div class="space-y-6">
          <div class="flex items-center justify-between border-b border-slate-800 pb-4">
            <h3 class="font-black text-slate-400 uppercase tracking-widest text-xs flex items-center">
              <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-12 0 9 9 0 0112 0z"></path></svg>
              发布历史记录
            </h3>
            <span class="bg-blue-500/20 text-blue-400 text-[10px] font-black px-2 py-0.5 rounded-md">{{ metadata.releases?.length || 0 }}</span>
          </div>

          <div class="space-y-4 relative">
            <!-- Vertical Timeline Line -->
            <div class="absolute left-4 top-2 bottom-0 w-0.5 bg-slate-800"></div>

            <div v-for="rel in sortedReleases" :key="rel.version_id" class="relative pl-10 group">
              <!-- Timeline Dot -->
              <div class="absolute left-[13px] top-1.5 w-2.5 h-2.5 rounded-full border-2 border-slate-900 transition-all" :class="rel.version_id === metadata.active_version ? 'bg-emerald-500 shadow-[0_0_8px_#10b981]' : 'bg-slate-700 group-hover:bg-slate-500'"></div>
              
              <div class="bg-slate-800/40 border border-slate-700/50 rounded-xl p-4 hover:border-slate-600 transition-all">
                <div class="flex justify-between items-start mb-2">
                  <span class="text-xs font-mono font-bold" :class="rel.version_id === metadata.active_version ? 'text-emerald-400' : 'text-slate-300'">#{{ rel.version_id }}</span>
                  <span v-if="rel.version_id === metadata.active_version" class="text-[8px] font-black bg-emerald-500/10 text-emerald-400 px-1.5 py-0.5 rounded uppercase">Active</span>
                </div>
                <div class="flex items-center text-[10px] text-slate-500 space-x-3 font-medium">
                   <span>{{ formatSize(rel.size) }}</span>
                   <span>•</span>
                   <span>{{ new Date(rel.time).toLocaleString() }}</span>
                </div>
                <div class="mt-4 flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button @click="setActiveVersion(rel.version_id)" class="flex-1 bg-blue-600/20 hover:bg-blue-600 text-blue-400 hover:text-white py-1 rounded text-[10px] font-black uppercase transition-all">部署此版本</button>
                  <button class="px-2 bg-slate-700 hover:bg-rose-600 rounded text-slate-400 hover:text-white transition-colors"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg></button>
                </div>
              </div>
            </div>
            
            <div v-if="!metadata.releases?.length" class="text-center py-10 text-slate-600 italic text-xs">暂无任何发布包记录</div>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { apiGetMetadata, apiSaveMetadata, apiDeleteInstance, apiReadConfig } from '../api';

const props = defineProps<{ instanceId: string }>();
const emit = defineEmits(['navigate']);

const metadata = ref<any>({ display_name: '', cdn_link: '', is_paused: false, releases: [], active_version: '' });
const globalConfig = ref<any>(null);
const saving = ref(false);

const sortedReleases = computed(() => {
  if (!metadata.value.releases) return [];
  return [...metadata.value.releases].sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime());
});

async function fetchMetadata() {
  try {
    const res = await apiGetMetadata(props.instanceId);
    metadata.value = res.data || res;
    const cfgRes = await apiReadConfig();
    globalConfig.value = cfgRes.data || cfgRes;
  } catch (e) { console.error(e); }
}

async function saveMetadata() {
  saving.value = true;
  try {
    await apiSaveMetadata(props.instanceId, metadata.value);
    alert('实例配置已更新'); emit('navigate', 'portal');
  } catch (e) { console.error(e); }
  saving.value = false;
}

async function setActiveVersion(vid: string) {
  metadata.value.active_version = vid;
  await saveMetadata();
}

async function deleteInstance() {
  if (!confirm(`高度警告：您正在销毁实例 [${props.instanceId}]。所有的游戏文件、资产索引都将被物理抹除！确认继续吗？`)) return;
  try {
    await apiDeleteInstance(props.instanceId);
    alert('实例已注销'); emit('navigate', 'portal');
  } catch (e) { console.error(e); }
}

function formatSize(b: number) { if (b === 0) return '0 B'; const k = 1024; const i = Math.floor(Math.log(b) / Math.log(k)); return parseFloat((b / Math.pow(k, i)).toFixed(2)) + ' ' + ['B', 'KB', 'MB', 'GB'][i]; }

onMounted(fetchMetadata);
</script>
