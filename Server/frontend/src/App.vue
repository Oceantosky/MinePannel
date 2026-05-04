<template>
  <!-- Loading -->
  <div v-if="!authChecked" class="h-screen w-screen bg-[#0f172a] flex items-center justify-center">
    <div class="text-slate-500 animate-pulse font-bold text-sm">验证身份中...</div>
  </div>

  <!-- Login View -->
  <LoginView v-else-if="!isLoggedIn" @login-success="handleLoginSuccess" />

  <!-- Main Application -->
  <div v-else class="h-screen w-screen bg-[#0f172a] text-slate-200 flex font-sans overflow-hidden">
    
    <!-- L1 Sidebar (Global Panel Controls) - Fixed width -->
    <aside class="w-20 bg-slate-950 border-r border-slate-800/80 flex flex-col items-center py-6 shrink-0 z-30 shadow-2xl">
      <!-- Logo -->
      <div 
        @click="handleNavigate('portal')" 
        class="w-12 h-12 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-2xl flex items-center justify-center cursor-pointer mb-10 shadow-[0_0_20px_rgba(59,130,246,0.3)] transition-transform hover:scale-105"
        title="主页"
      >
        <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path></svg>
      </div>

      <!-- Global Navigation -->
      <nav class="flex flex-col space-y-6 flex-1 w-full items-center">
        <div class="relative group">
          <button 
            @click="handleNavigate('portal')" 
            class="w-12 h-12 rounded-2xl flex items-center justify-center transition-all"
            :class="['portal'].includes(currentView) && !currentInstanceId ? 'bg-blue-500/20 text-blue-400' : 'text-slate-500 hover:text-slate-300 hover:bg-slate-800'"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"></path></svg>
          </button>
          <!-- Tooltip -->
          <div class="absolute left-full top-1/2 -translate-y-1/2 ml-4 px-3 py-1.5 bg-slate-800 border border-slate-700 rounded shadow-xl opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none whitespace-nowrap z-50 text-xs font-bold text-slate-200">
            实例列表
          </div>
        </div>

        <div class="relative group">
          <button 
            @click="handleNavigate('nodes')" 
            class="w-12 h-12 rounded-2xl flex items-center justify-center transition-all"
            :class="currentView === 'nodes' ? 'bg-blue-500/20 text-blue-400' : 'text-slate-500 hover:text-slate-300 hover:bg-slate-800'"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path></svg>
          </button>
          <!-- Tooltip -->
          <div class="absolute left-full top-1/2 -translate-y-1/2 ml-4 px-3 py-1.5 bg-slate-800 border border-slate-700 rounded shadow-xl opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none whitespace-nowrap z-50 text-xs font-bold text-slate-200">
            节点监控
          </div>
        </div>

        <div class="relative group">
          <button
            @click="handleNavigate('preauth')"
            class="w-12 h-12 rounded-2xl flex items-center justify-center transition-all"
            :class="currentView === 'preauth' ? 'bg-cyan-500/20 text-cyan-400' : 'text-slate-500 hover:text-slate-300 hover:bg-slate-800'"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"></path></svg>
          </button>
          <!-- Tooltip -->
          <div class="absolute left-full top-1/2 -translate-y-1/2 ml-4 px-3 py-1.5 bg-slate-800 border border-slate-700 rounded shadow-xl opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none whitespace-nowrap z-50 text-xs font-bold text-slate-200">
            预授权管理
          </div>
        </div>

        <div class="relative group">
          <button
            @click="handleNavigate('settings')"
            class="w-12 h-12 rounded-2xl flex items-center justify-center transition-all"
            :class="currentView === 'settings' ? 'bg-blue-500/20 text-blue-400' : 'text-slate-500 hover:text-slate-300 hover:bg-slate-800'"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path></svg>
          </button>
          <!-- Tooltip -->
          <div class="absolute left-full top-1/2 -translate-y-1/2 ml-4 px-3 py-1.5 bg-slate-800 border border-slate-700 rounded shadow-xl opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none whitespace-nowrap z-50 text-xs font-bold text-slate-200">
            系统设置
          </div>
        </div>
      </nav>

      <!-- Bottom User Profile -->
      <div class="mt-auto relative group">
        <button class="w-10 h-10 rounded-full bg-slate-800 border-2 border-slate-700 flex items-center justify-center font-black text-xs transition-colors"
          :class="currentUser?.role === 'admin' ? 'text-amber-400 border-amber-500/30' : 'text-slate-400 hover:border-slate-500'"
          :title="currentUser?.username || 'User'">
          {{ (currentUser?.username || 'U')[0].toUpperCase() }}
        </button>
        <div class="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg shadow-xl opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none whitespace-nowrap z-50">
          <p class="text-xs font-bold text-slate-200">{{ currentUser?.username }}</p>
          <p class="text-[10px] text-slate-500">{{ currentUser?.role === 'admin' ? '管理员' : '用户' }}</p>
        </div>
      </div>
      <button @click="logout" class="w-8 h-8 flex items-center justify-center text-slate-600 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-all mt-1" title="退出登录">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"></path></svg>
      </button>
    </aside>

    <!-- L2 Sidebar (Instance specific features - "Cloud PCL") -->
    <!-- Fixed width, conditionally rendered based on currentInstanceId -->
    <aside 
      v-if="currentInstanceId"
      class="w-64 bg-slate-900 border-r border-slate-800/80 flex flex-col shrink-0 z-20 transition-all duration-300 transform origin-left shadow-2xl"
    >
      <div class="h-20 flex flex-col justify-center px-6 border-b border-slate-800/50 shrink-0">
        <h2 class="text-sm font-black text-slate-200 truncate" :title="instanceMeta?.display_name || currentInstanceId">
          {{ instanceMeta?.display_name || 'MinePannel 实例' }}
        </h2>
        <p class="text-[10px] font-mono text-slate-500 uppercase tracking-widest mt-1">{{ currentInstanceId }}</p>
      </div>

      <div class="flex-1 overflow-y-auto py-6 px-4 space-y-8 custom-scrollbar">
        
        <!-- PCL Navigation Group -->
        <div class="space-y-2">
          <h3 class="px-2 text-[10px] font-black text-slate-500 uppercase tracking-widest mb-3">实例控制台</h3>
          
          <button @click="handleNavigate('explorer', { instanceId: currentInstanceId })" class="w-full flex items-center px-3 py-2.5 rounded-xl text-sm font-bold transition-all" :class="currentView === 'explorer' || currentView === 'editor' ? 'text-blue-400 bg-blue-500/10 border border-blue-500/20' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800 border border-transparent'">
            <svg class="w-4 h-4 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path></svg>
            资源管理器
          </button>
          
          <button @click="handleNavigate('market', { instanceId: currentInstanceId })" class="w-full flex items-center px-3 py-2.5 rounded-xl text-sm font-bold transition-all group" :class="currentView === 'market' ? 'text-indigo-400 bg-indigo-500/10 border border-indigo-500/20' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800 border border-transparent'">
            <svg class="w-4 h-4 mr-3 group-hover:animate-bounce" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"></path></svg>
            云端装机市场 <span class="ml-auto bg-indigo-500/20 text-indigo-400 text-[8px] px-1.5 py-0.5 rounded uppercase">New</span>
          </button>
          
          <button @click="handleNavigate('console', { instanceId: currentInstanceId })" class="w-full flex items-center px-3 py-2.5 rounded-xl text-sm font-bold transition-all" :class="currentView === 'console' ? 'text-blue-400 bg-blue-500/10 border border-blue-500/20' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800 border border-transparent'">
            <svg class="w-4 h-4 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path></svg>
            运行日志
          </button>

          <button @click="handleNavigate('instance-settings', { instanceId: currentInstanceId })" class="w-full flex items-center px-3 py-2.5 rounded-xl text-sm font-bold transition-all" :class="currentView === 'instance-settings' ? 'text-blue-400 bg-blue-500/10 border border-blue-500/20' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800 border border-transparent'">
            <svg class="w-4 h-4 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4"></path></svg>
            实例配置
          </button>

          <button @click="handleNavigate('version', { instanceId: currentInstanceId })" class="w-full flex items-center px-3 py-2.5 rounded-xl text-sm font-bold transition-all" :class="currentView === 'version' ? 'text-purple-400 bg-purple-500/10 border border-purple-500/20' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800 border border-transparent'">
            <svg class="w-4 h-4 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7v8a2 2 0 002 2h6M8 7V5a2 2 0 012-2h4.586a1 1 0 01.707.293l4.414 4.414a1 1 0 01.293.707V15a2 2 0 01-2 2h-2M8 7H6a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2v-2"/>
            </svg>
            版本控制
          </button>
        </div>

        <!-- Git Version Tree -->
        <div class="space-y-2 pt-6 border-t border-slate-800/50">
          <h3 class="px-2 text-[10px] font-black text-slate-500 uppercase tracking-widest mb-4">时间轴 (快照树)</h3>
          
          <div class="space-y-1 relative pl-4">
            <div class="absolute left-[21px] top-2 bottom-2 w-[1px] bg-slate-700/50"></div>

            <div 
              v-for="rel in sortedReleases" 
              :key="rel.version_id"
              @click="switchVersion(rel.version_id)"
              class="flex items-center px-2 py-2 rounded-xl cursor-pointer transition-all group relative"
              :class="rel.version_id === instanceMeta?.active_version ? 'bg-slate-800/60' : 'hover:bg-slate-800/40'"
            >
              <div class="z-10 mr-3 shrink-0">
                <div class="w-2 h-2 rounded-full border border-slate-900 transition-all shadow-md" :class="rel.version_id === instanceMeta?.active_version ? 'bg-emerald-500 scale-150 shadow-[0_0_10px_#10b981]' : 'bg-slate-600 group-hover:bg-slate-400'"></div>
              </div>
              <div class="min-w-0">
                <p class="text-xs font-mono font-bold truncate transition-colors" :class="rel.version_id === instanceMeta?.active_version ? 'text-emerald-400' : 'text-slate-400 group-hover:text-slate-200'">
                  v{{ rel.version_id }}
                </p>
              </div>
            </div>
            
            <div v-if="!sortedReleases.length" class="px-2 py-6 text-left text-[10px] text-slate-600 italic">尚未生成任何快照</div>
          </div>
        </div>

      </div>
    </aside>

    <!-- Main Content Area -->
    <main class="flex-1 flex flex-col min-w-0 bg-[#0a0f1c] relative z-0">
      <!-- Dynamic View Component -->
      <component
        :is="currentViewComponent"
        @navigate="handleNavigate"
        @refresh-meta="fetchInstanceMeta"
        :instanceId="currentInstanceId"
        :editorData="editorData"
        :initialType="initialMarketType"
        class="flex-1 overflow-auto custom-scrollbar"
      />
    </main>
  </div><!-- end main app -->
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import PortalView from './components/PortalView.vue'
import ExplorerView from './components/ExplorerView.vue'
import EditorView from './components/EditorView.vue'
import ConsoleView from './components/ConsoleView.vue'
import NodesView from './components/NodesView.vue'
import SettingsView from './components/SettingsView.vue'
import InstanceSettingsView from './components/InstanceSettingsView.vue'
import MarketView from './components/MarketView.vue'
import PreAuthView from './components/PreAuthView.vue'
import VersionControlView from './components/VersionControlView.vue'
import { apiGetMetadata, apiSaveMetadata, apiAuthStatus, apiLogout, clearCurrentUser, getCurrentUser } from './api'
import LoginView from './components/LoginView.vue'

type ViewState = 'portal' | 'explorer' | 'editor' | 'console' | 'nodes' | 'settings' | 'instance-settings' | 'market' | 'preauth' | 'version'
const currentView = ref<ViewState>('portal')
const currentInstanceId = ref('')
const editorData = ref({ path: '', content: '' })
const instanceMeta = ref<any>(null)
const initialMarketType = ref('')
const isLoggedIn = ref(false)
const currentUser = ref<any>(null)
const authChecked = ref(false)

const currentViewComponent = computed(() => {
  switch (currentView.value) {
    case 'portal': return PortalView
    case 'explorer': return ExplorerView
    case 'editor': return EditorView
    case 'console': return ConsoleView
    case 'nodes': return NodesView
    case 'settings': return SettingsView
    case 'instance-settings': return InstanceSettingsView
    case 'market': return MarketView
    case 'preauth': return PreAuthView
    case 'version': return VersionControlView
    default: return PortalView
  }
})

const sortedReleases = computed(() => {
  if (!instanceMeta.value?.releases) return []
  return [...instanceMeta.value.releases]
    .sort((a: any, b: any) => new Date(b.time).getTime() - new Date(a.time).getTime())
    .slice(0, 10)
})

async function fetchInstanceMeta() {
  if (!currentInstanceId.value) { instanceMeta.value = null; return }
  try {
    const res = await apiGetMetadata(currentInstanceId.value)
    instanceMeta.value = res.data || res
  } catch (e) { console.error(e) }
}

async function switchVersion(vid: string) {
  if (!instanceMeta.value) return
  instanceMeta.value.active_version = vid
  try {
    await apiSaveMetadata(currentInstanceId.value, instanceMeta.value)
    alert(`客户端分发版本已安全回滚至: ${vid}`)
    fetchInstanceMeta()
  } catch (e) { console.error(e) }
}

function handleNavigate(view: ViewState, payload?: any) {
  currentView.value = view
  if (['explorer', 'console', 'instance-settings', 'market', 'version'].includes(view) && payload?.instanceId) {
    currentInstanceId.value = payload.instanceId
  } else if (view === 'portal' || view === 'nodes' || view === 'settings' || view === 'preauth') {
    currentInstanceId.value = ''
  }

  // 从 ExplorerView 传递文件夹上下文到 MarketView
  if (view === 'market') {
    initialMarketType.value = payload?.initialType || ''
  } else {
    // 离开市场视图时清除
    initialMarketType.value = ''
  }

  if (view === 'editor' && payload) editorData.value = payload

  let hash = `#/${view}`
  if (currentInstanceId.value) hash += `/${currentInstanceId.value}`
  window.location.hash = hash
}

function handleLoginSuccess(user: any) {
  currentUser.value = user
  isLoggedIn.value = true
  currentView.value = 'portal'
}

async function logout() {
  try { await apiLogout() } catch { /* cookie cleared regardless */ }
	clearCurrentUser()
  isLoggedIn.value = false
  currentUser.value = null
  currentInstanceId.value = ''
  instanceMeta.value = null
}

async function checkAuth() {
  const savedUser = getCurrentUser()
  if (!savedUser) {
    authChecked.value = true
    return
  }
  try {
    const res = await apiAuthStatus()
    currentUser.value = res.data?.user || savedUser
    isLoggedIn.value = true
  } catch {
    clearCurrentUser()
  }
  authChecked.value = true
}

function syncStateFromHash() {
  const hash = window.location.hash.replace('#/', '')
  if (!hash) return
  const [view, id] = hash.split('/')
  if (view) {
    currentView.value = view as ViewState
    if (id) currentInstanceId.value = id
  }
}

watch(currentInstanceId, fetchInstanceMeta)

onMounted(async () => {
  await checkAuth()
  if (isLoggedIn.value) syncStateFromHash()
  window.addEventListener('hashchange', syncStateFromHash)
})
</script>

<style>
/* Global custom scrollbar for designated containers */
.custom-scrollbar::-webkit-scrollbar { width: 6px; height: 6px; }
.custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
.custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.08); border-radius: 10px; }
.custom-scrollbar::-webkit-scrollbar-thumb:hover { background: rgba(255, 255, 255, 0.15); }
</style>