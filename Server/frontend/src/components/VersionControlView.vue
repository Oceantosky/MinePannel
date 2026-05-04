<template>
  <div class="h-full flex flex-col bg-[#0a0f1c]">
    <!-- Header -->
    <div class="shrink-0 px-8 py-6 border-b border-slate-800/50">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-xl font-black text-slate-100 flex items-center gap-3">
            <svg class="w-6 h-6 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7v8a2 2 0 002 2h6M8 7V5a2 2 0 012-2h4.586a1 1 0 01.707.293l4.414 4.414a1 1 0 01.293.707V15a2 2 0 01-2 2h-2M8 7H6a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2v-2"/>
            </svg>
            {{ meta?.display_name || instanceId }} / version-history
          </h1>
          <p class="text-xs text-slate-500 mt-2 flex items-center gap-2">
            <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-[10px] font-bold">
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
              Active
            </span>
            <code class="font-mono text-slate-400">{{ meta?.active_version || '—' }}</code>
          </p>
        </div>
        <button
          @click="fetchReleases"
          class="px-4 py-2 rounded-xl bg-slate-800 border border-slate-700 text-slate-300 text-sm font-bold hover:bg-slate-700 hover:text-slate-100 transition-all flex items-center gap-2"
        >
          <svg class="w-4 h-4" :class="{ 'animate-spin': loading }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
          </svg>
          刷新
        </button>
      </div>
    </div>

    <!-- Filter Bar -->
    <div class="shrink-0 px-8 py-3 border-b border-slate-800/30 flex items-center gap-2">
      <button
        v-for="f in filters"
        :key="f.key"
        @click="activeFilter = f.key"
        class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all"
        :class="activeFilter === f.key
          ? 'bg-purple-500/10 text-purple-400 border border-purple-500/20'
          : 'text-slate-500 hover:text-slate-300 hover:bg-slate-800 border border-transparent'"
      >
        {{ f.label }}
        <span class="ml-1.5 text-[10px] opacity-60">{{ f.count }}</span>
      </button>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-auto custom-scrollbar">
      <!-- Loading -->
      <div v-if="loading" class="px-8 py-6 space-y-4">
        <div v-for="i in 5" :key="i" class="flex gap-4 animate-pulse">
          <div class="flex flex-col items-center">
            <div class="w-3 h-3 rounded-full bg-slate-700"></div>
            <div class="w-px h-full bg-slate-800 mt-1"></div>
          </div>
          <div class="flex-1 space-y-2 pb-6">
            <div class="h-3 bg-slate-800 rounded w-32"></div>
            <div class="h-3 bg-slate-800 rounded w-64"></div>
            <div class="h-3 bg-slate-800 rounded w-20"></div>
          </div>
        </div>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="px-8 py-16 text-center">
        <div class="text-rose-400 text-sm font-bold mb-2">加载失败</div>
        <p class="text-slate-500 text-xs mb-4">{{ error }}</p>
        <button @click="fetchReleases" class="px-4 py-2 rounded-xl bg-slate-800 border border-slate-700 text-slate-300 text-xs font-bold hover:bg-slate-700 transition-all">重试</button>
      </div>

      <!-- Empty -->
      <div v-else-if="!filteredReleases.length" class="px-8 py-16 text-center">
        <svg class="w-16 h-16 mx-auto mb-4 text-slate-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z"/>
        </svg>
        <p class="text-slate-500 text-sm font-bold mb-1">尚无版本历史</p>
        <p class="text-slate-600 text-xs">请先执行 Commit 或部署 PCL 包来创建版本记录</p>
      </div>

      <!-- Timeline -->
      <div v-else class="relative">
        <!-- Vertical line -->
        <div class="absolute left-[59px] top-6 bottom-6 w-px bg-slate-800/60"></div>

        <div
          v-for="release in filteredReleases"
          :key="release.version_id"
          class="relative px-8 py-3 transition-colors"
          :class="release.version_id === meta?.active_version ? 'bg-emerald-500/5' : 'hover:bg-slate-800/20'"
        >
          <!-- Active version left accent -->
          <div
            v-if="release.version_id === meta?.active_version"
            class="absolute left-0 top-0 bottom-0 w-1 bg-emerald-500/60"
          ></div>

          <div class="flex gap-4">
            <!-- Dot -->
            <div class="flex flex-col items-center shrink-0 z-10" style="width: 24px">
              <div
                class="rounded-full border-2 border-slate-900 transition-all shadow-md"
                :class="getRelType(release) === 'packet'
                  ? 'w-3 h-3 bg-indigo-500 shadow-[0_0_10px_rgba(99,102,241,0.5)]'
                  : 'w-2 h-2 bg-slate-600'"
              ></div>
            </div>

            <!-- Content -->
            <div class="flex-1 min-w-0 pb-2">
              <div class="flex items-center gap-2 flex-wrap mb-1">
                <!-- Type badge -->
                <span
                  class="text-[10px] font-bold px-2 py-0.5 rounded uppercase tracking-wider"
                  :class="getRelType(release) === 'packet'
                    ? 'bg-indigo-500/15 text-indigo-400 border border-indigo-500/20'
                    : 'bg-slate-800 text-slate-500 border border-slate-700'"
                >
                  {{ getRelType(release) }}
                </span>

                <!-- Version ID -->
                <code
                  class="font-mono text-sm font-bold"
                  :class="release.version_id === meta?.active_version ? 'text-emerald-400' : 'text-slate-200'"
                >
                  {{ release.version_id }}
                </code>

                <!-- Active label -->
                <span
                  v-if="release.version_id === meta?.active_version"
                  class="text-[10px] font-bold px-2 py-0.5 rounded bg-emerald-500/10 border border-emerald-500/20 text-emerald-400"
                >
                  Active
                </span>

                <!-- Spacer -->
                <span class="flex-1"></span>

                <!-- Actions -->
                <div class="flex items-center gap-1">
                  <!-- Download (packet only) -->
                  <button
                    v-if="getRelType(release) === 'packet'"
                    @click="handleDownload(release)"
                    class="px-2.5 py-1 rounded-lg text-[10px] font-bold transition-all flex items-center gap-1 bg-slate-800 border border-slate-700 text-slate-400 hover:text-indigo-400 hover:border-indigo-500/30 hover:bg-indigo-500/10"
                    title="下载 ZIP"
                  >
                    <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
                    下载
                  </button>

                  <!-- Rollback -->
                  <button
                    v-if="release.version_id !== meta?.active_version"
                    @click="handleRollback(release.version_id)"
                    class="px-2.5 py-1 rounded-lg text-[10px] font-bold transition-all flex items-center gap-1 bg-slate-800 border border-slate-700 text-slate-400 hover:text-amber-400 hover:border-amber-500/30 hover:bg-amber-500/10"
                    title="回滚到此版本"
                  >
                    <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"/></svg>
                    回滚
                  </button>
                </div>
              </div>

              <!-- Message -->
              <p
                class="text-sm"
                :class="getRelType(release) === 'packet' ? 'text-slate-300 font-semibold' : 'text-slate-400'"
              >
                {{ release.message || '—' }}
              </p>

              <!-- Meta line -->
              <p class="text-[11px] text-slate-600 mt-1">
                {{ formatRelativeTime(release.time) }}
                <template v-if="getRelType(release) === 'packet' && release.size">
                  · {{ formatSize(release.size) }}
                </template>
                <template v-if="release.file_name">
                  · <code class="text-[10px] text-slate-500">{{ release.file_name }}</code>
                </template>
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { apiGetMetadata, apiSaveMetadata, apiDownloadRelease, getReleaseType } from '../api'
import type { ReleaseEntry } from '../api'

const props = defineProps<{ instanceId: string }>()
const emit = defineEmits<{ (e: 'refresh-meta'): void }>()

const meta = ref<any>(null)
const releases = ref<ReleaseEntry[]>([])
const loading = ref(false)
const error = ref('')
const activeFilter = ref<'all' | 'commit' | 'packet'>('all')

const filters = computed(() => {
  const commits = releases.value.filter(r => getReleaseType(r) === 'commit')
  const packets = releases.value.filter(r => getReleaseType(r) === 'packet')
  return [
    { key: 'all' as const, label: 'All', count: releases.value.length },
    { key: 'commit' as const, label: 'Commits', count: commits.length },
    { key: 'packet' as const, label: 'Packets', count: packets.length },
  ]
})

const filteredReleases = computed(() => {
  let list = [...releases.value]
  if (activeFilter.value === 'commit') {
    list = list.filter(r => getReleaseType(r) === 'commit')
  } else if (activeFilter.value === 'packet') {
    list = list.filter(r => getReleaseType(r) === 'packet')
  }
  return list.sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime())
})

function getRelType(r: ReleaseEntry) {
  return getReleaseType(r)
}

async function fetchReleases() {
  if (!props.instanceId) return
  loading.value = true
  error.value = ''
  try {
    const res = await apiGetMetadata(props.instanceId)
    const data = res.data || res
    meta.value = data
    releases.value = data.releases || []
  } catch (e: any) {
    error.value = e.message || 'Failed to load releases'
  } finally {
    loading.value = false
  }
}

async function handleRollback(versionId: string) {
  if (!meta.value) return
  const confirmed = confirm(`确定要将版本回滚至 ${versionId} 吗？\n\n此操作将更改客户端分发版本。`)
  if (!confirmed) return
  meta.value.active_version = versionId
  try {
    await apiSaveMetadata(props.instanceId, meta.value)
    emit('refresh-meta')
    alert(`版本已成功回滚至: ${versionId}`)
    await fetchReleases()
  } catch (e: any) {
    alert('回滚失败: ' + (e.message || 'Unknown error'))
  }
}

async function handleDownload(_release: ReleaseEntry) {
  const downloadable = releases.value.find(r => getReleaseType(r) === 'packet' && r.file_name)
  if (!downloadable) {
    alert('没有可下载的 Packet 包，请先执行 Packet Up 操作生成分发包。')
    return
  }
  try {
    await apiDownloadRelease(props.instanceId)
  } catch (e: any) {
    alert('下载失败: ' + (e.message || 'Unknown error'))
  }
}

function formatRelativeTime(t: string): string {
  const diff = Date.now() - new Date(t).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins} 分钟前`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} 天前`
  return new Date(t).toLocaleDateString('zh-CN')
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

onMounted(fetchReleases)
watch(() => props.instanceId, fetchReleases)
</script>
