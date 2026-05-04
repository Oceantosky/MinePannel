<template>
  <div
    class="flex-1 flex flex-col overflow-hidden relative"
    @dragenter.prevent="handleDragEnter"
    @dragleave.prevent="handleDragLeave"
    @dragover.prevent
    @drop.prevent="handleDrop"
  >
    <!-- Drag & Drop Hover Overlay -->
    <div v-if="dragCounter > 0 && !isUploading && !showPclModal" class="absolute inset-0 z-50 bg-indigo-600/20 backdrop-blur-sm border-4 border-dashed border-indigo-500 flex flex-col items-center justify-center pointer-events-none">
      <div class="bg-slate-900/80 p-8 rounded-3xl shadow-2xl flex flex-col items-center border border-indigo-500/50">
        <svg class="w-16 h-16 text-indigo-400 animate-bounce mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"></path></svg>
        <p class="text-xl font-black text-white tracking-tight">释放文件以开始上传</p>
      </div>
    </div>

    <!-- PCL Upload Modal -->
    <div v-if="showPclModal" class="fixed inset-0 z-[100] flex items-center justify-center p-4">
      <div class="absolute inset-0 bg-slate-950/80 backdrop-blur-xl" @click="!isUploading && (showPclModal = false)"></div>
      <div class="bg-slate-900 border border-white/10 w-full max-w-2xl rounded-3xl shadow-2xl z-10 overflow-hidden transform transition-all animate-in zoom-in duration-300">
        <div class="p-8 text-center space-y-6">
          <div class="w-20 h-20 bg-blue-500/10 rounded-3xl flex items-center justify-center mx-auto text-blue-400">
            <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"></path></svg>
          </div>
          <div>
            <h3 class="text-2xl font-black text-white">部署 PCL 客户包</h3>
            <p class="text-slate-400 mt-2 text-sm font-medium">拖入 PCL 导出的 .zip 压缩包，云端将自动解压、补全依赖并部署环境</p>
          </div>
          <div @drop.prevent="handlePclDrop" @dragover.prevent class="border-2 border-dashed border-slate-700 rounded-2xl p-12 hover:border-blue-500/50 transition-colors cursor-pointer group bg-black/20">
            <div v-if="!isUploading" class="space-y-2"><p class="text-slate-500 group-hover:text-blue-400 transition-colors font-bold">将 ZIP 包拖到此处</p><p class="text-[10px] text-slate-600 uppercase tracking-widest">或点击下方按钮选择文件</p></div>
            <div v-else class="space-y-4">
              <div class="flex justify-between text-xs font-black text-blue-400 uppercase tracking-tighter"><span>正在同步资产...</span><span>{{ Math.round(uploadProgress) }}%</span></div>
              <div class="w-full bg-slate-800 rounded-full h-2.5 overflow-hidden"><div class="bg-blue-500 h-full rounded-full shadow-[0_0_15px_rgba(59,130,246,0.6)] transition-all duration-300" :style="{ width: uploadProgress + '%' }"></div></div>
            </div>
          </div>
          <div class="flex gap-4"><button @click="showPclModal = false" :disabled="isUploading" class="flex-1 px-6 py-3 rounded-xl font-bold text-slate-400 hover:bg-slate-800 transition-colors border border-slate-800">取消</button><button @click="pclInput?.click()" :disabled="isUploading" class="flex-[2] bg-blue-600 hover:bg-blue-500 text-white py-3 rounded-xl font-bold transition-all shadow-lg shadow-blue-600/20">选择 PCL 压缩包</button><input type="file" ref="pclInput" class="hidden" accept=".zip" @change="handlePclSelect"></div>
        </div>
      </div>
    </div>

    <!-- Common Upload Overlay -->
    <div v-if="isUploading && !showPclModal" class="absolute inset-0 z-[60] bg-slate-900/90 backdrop-blur-md flex flex-col items-center justify-center animate-in fade-in duration-300">
      <div class="w-full max-w-md p-10 bg-slate-800/50 rounded-3xl border border-white/10 shadow-2xl">
        <div class="flex items-center justify-between mb-6">
          <div class="flex-1 min-w-0"><h3 class="text-xl font-bold text-white">同步资产中...</h3><p class="text-xs text-slate-400 mt-1 truncate pr-4">{{ currentFileName }}</p></div>
          <span class="text-2xl font-black text-indigo-400 font-mono">{{ Math.round(uploadProgress) }}%</span>
        </div>
        <div class="w-full bg-slate-900 rounded-full h-3 mb-6 p-0.5 border border-white/5"><div class="bg-gradient-to-r from-indigo-500 via-purple-500 to-blue-500 h-full rounded-full transition-all duration-300 shadow-[0_0_15px_rgba(99,102,241,0.4)]" :style="{ width: uploadProgress + '%' }"></div></div>
      </div>
    </div>

    <!-- Header -->
    <header class="h-20 border-b border-slate-800 bg-slate-900/50 backdrop-blur-xl flex flex-col justify-center px-8 z-10 sticky top-0">
      <div class="flex items-center justify-between w-full space-x-4">
        <!-- Left Side: Breadcrumbs -->
        <div class="flex items-center space-x-2 text-slate-300 flex-1 min-w-0 max-w-[450px] whitespace-nowrap">
          <button @click="goToParentDir" class="hover:text-blue-400 transition-colors flex items-center text-slate-500 hover:bg-slate-800 p-1 rounded shrink-0"><svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path></svg></button>
          <span class="text-slate-600 shrink-0">/</span>
          <div class="flex items-center space-x-2 flex-1 min-w-0">
            <button @click="currentPathSegments = []" class="hover:text-blue-400 transition-colors font-medium text-sm px-1.5 py-0.5 rounded hover:bg-slate-800 shrink-0 truncate max-w-[100px]" :class="currentPathSegments.length === 0 ? 'text-slate-100 font-bold' : 'text-slate-400'">{{ instanceId }}</button>
            <template v-for="(segment, idx) in visibleBreadcrumbs" :key="'bc-'+idx">
              <span class="text-slate-600 shrink-0">/</span>
              <span v-if="segment.isEllipsis" class="text-slate-500 px-1 shrink-0">...</span>
              <button v-else @click="navigateToBreadcrumb(segment.index)" class="hover:text-blue-400 transition-colors font-medium text-sm px-1.5 py-0.5 rounded hover:bg-slate-800 truncate" style="max-width: 180px;" :class="segment.index === currentPathSegments.length - 1 ? 'text-slate-100 font-bold' : 'text-slate-400'" :title="segment.text">
                {{ segment.text }}
              </button>
            </template>
          </div>
        </div>

        <!-- Right Side: Controls -->
        <div class="flex items-center space-x-3 shrink-0">
          <div class="relative group w-48 xl:w-64">
            <div class="absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none">
              <svg class="w-4 h-4 text-slate-500 group-focus-within:text-slate-300 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
            </div>
            <input
              v-model="searchQuery"
              type="text"
              class="w-full bg-transparent border border-slate-700/80 hover:border-slate-600 text-slate-200 text-sm font-medium rounded-lg focus:ring-1 focus:ring-blue-500 focus:border-blue-500 block pl-9 p-1.5 transition-all placeholder-slate-500"
              placeholder="搜索当前目录..."
            />
            <div v-if="!searchQuery" class="absolute inset-y-0 right-0 flex items-center pr-2.5 pointer-events-none">
              <span class="border border-slate-700 rounded px-1.5 text-[10px] text-slate-500 font-mono font-bold bg-slate-800/50">/</span>
            </div>
            <button v-if="searchQuery" @click="searchQuery = ''" class="absolute inset-y-0 right-0 flex items-center pr-2.5 text-slate-500 hover:text-slate-300 transition-colors">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
            </button>
          </div>

          <!-- Commit / Packet Up / Push -->
          <div class="flex bg-slate-800/50 rounded-lg p-1 border border-slate-700/50 shadow-inner">
            <button @click="triggerCommit" class="px-3 py-1.5 text-[10px] font-black text-slate-400 hover:text-purple-400 transition-colors tracking-widest uppercase">Commit</button>
            <div class="w-px h-4 bg-slate-700 my-auto"></div>
            <button @click="triggerPacketUp" class="px-3 py-1.5 text-[10px] font-black text-slate-400 hover:text-blue-400 transition-colors tracking-widest uppercase">Packet Up</button>
            <div class="w-px h-4 bg-slate-700 my-auto"></div>
            <button @click="triggerPush" class="px-3 py-1.5 text-[10px] font-black text-slate-400 hover:text-emerald-400 transition-colors tracking-widest uppercase">Push</button>
          </div>

          <button @click="fetchFiles" class="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"><svg class="w-5 h-5" :class="{ 'animate-spin': loading }" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg></button>
          <button @click="showPclModal = true" class="bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white px-5 py-2 rounded-lg text-sm font-black transition-all shadow-lg shadow-blue-500/25 flex items-center transform hover:-translate-y-0.5 border border-blue-400/20"><svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path></svg>部署 PCL 客户包</button>
        </div>
      </div>
    </header>

    <!-- File List -->
    <div class="flex-1 overflow-auto px-8 pt-8 pb-4 z-10 flex flex-col">
      <div class="bg-slate-800/60 backdrop-blur-md border border-slate-700/50 rounded-t-2xl shadow-xl flex-1 flex flex-col overflow-hidden min-h-[400px]">
        <div class="grid grid-cols-12 gap-4 px-6 py-4 bg-slate-900/40 border-b border-slate-700/50 text-xs font-bold text-slate-500 uppercase tracking-widest shrink-0">
          <div class="col-span-6 flex items-center gap-3">
            <span>名称</span>
          </div>
          <div class="col-span-2 flex items-center justify-end">大小</div>
          <div class="col-span-3 flex items-center justify-end">修改时间</div>
          <div class="col-span-1 flex items-center justify-end"></div>
        </div>
        <div class="divide-y divide-slate-700/30 overflow-y-auto custom-scrollbar flex-1">
          <!-- New Folder Input Row -->
          <div v-if="showNewFolderInput" class="grid grid-cols-12 gap-4 px-6 py-3 bg-indigo-500/5 items-center animate-in slide-in-from-top-2 duration-200">
            <div class="col-span-6 flex items-center">
              <svg class="w-6 h-6 text-blue-400 mr-3" fill="currentColor" viewBox="0 0 20 20"><path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z"></path></svg>
              <input v-model="newFolderName" @keyup.enter="createFolder" @keyup.esc="showNewFolderInput = false" v-focus type="text" placeholder="文件夹名称..." class="bg-slate-900 border border-indigo-500/50 rounded px-2 py-1 text-sm text-slate-200 outline-none w-64">
            </div>
            <div class="col-span-6 flex justify-end space-x-4">
              <button @click="createFolder" class="text-xs font-black text-indigo-400 hover:text-indigo-300 uppercase tracking-widest">Confirm</button>
              <button @click="showNewFolderInput = false" class="text-xs font-bold text-slate-500 hover:text-slate-400 uppercase">Cancel</button>
            </div>
          </div>
          <!-- File Rows -->
          <div v-for="file in filteredFiles" :key="file.name"
            @click="toggleFileSelection(file.name)"
            @dblclick="handleFileOpen(file)"
            class="grid grid-cols-12 gap-4 px-6 py-3.5 hover:bg-slate-700/30 transition-all items-center group cursor-pointer border-l-2"
            :class="isSelected(file.name) ? 'bg-blue-500/10 border-blue-500' : 'border-transparent hover:border-blue-500/50'"
          >
            <div class="col-span-6 flex items-center">
              <input v-if="multiSelectMode" type="checkbox" :checked="isSelected(file.name)" @click.stop @change="toggleFileSelection(file.name)" class="mr-3 w-4 h-4 rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500/50 shrink-0" />
              <svg v-if="file.is_dir" class="w-6 h-6 text-blue-400 mr-3 shrink-0" fill="currentColor" viewBox="0 0 20 20"><path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z"></path></svg>
              <svg v-else-if="isEditable(file.name)" class="w-6 h-6 text-purple-400 mr-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
              <svg v-else class="w-6 h-6 text-slate-400 mr-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"></path></svg>
              <span class="font-medium transition-colors min-w-0 break-all select-none" :class="file.is_dir ? 'text-blue-100 group-hover:text-blue-300' : 'text-slate-200 group-hover:text-purple-300'">
                {{ file.name }}
              </span>
            </div>
            <div class="col-span-2 text-right text-slate-500 font-mono text-xs">{{ file.is_dir ? '-' : formatSize(file.size) }}</div>
            <div class="col-span-3 text-right text-slate-500 font-mono text-xs">{{ formatDate(file.time) }}</div>
            <div class="col-span-1 flex justify-end opacity-0 group-hover:opacity-100 transition-opacity">
              <button @click.stop="deleteSingleFile(file.name)" class="p-1.5 text-slate-500 hover:text-rose-500 hover:bg-rose-500/10 rounded-lg transition-all" title="删除">
                <svg class="w-4.5 h-4.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
              </button>
            </div>
          </div>
          <div v-if="!loading && filteredFiles.length === 0 && files.length > 0" class="px-6 py-16 text-center text-slate-500 italic">没有找到符合 "{{ searchQuery }}" 的文件</div>
          <div v-if="!loading && files.length === 0" class="px-6 py-16 text-center text-slate-600 italic">当前目录下空空如也</div>
          <div v-if="loading" class="px-6 py-16 text-center text-slate-500 animate-pulse font-bold tracking-widest uppercase text-xs">读取资产索引中...</div>
        </div>
      </div>
    </div>

    <!-- Bottom Action Bar -->
    <div class="h-14 border-t border-slate-800 bg-slate-900/80 backdrop-blur-xl flex items-center justify-between px-6 z-10 shrink-0">
      <div class="flex items-center space-x-1">
        <span v-if="selectedFiles.size > 0" class="text-xs font-bold text-slate-300 mr-3">已选 {{ selectedFiles.size }} 项</span>
        <span v-else class="text-xs text-slate-500 mr-3">单击选择，双击打开</span>
      </div>
      <div class="flex items-center space-x-2">
        <!-- Multi-Select Toggle -->
        <button @click="multiSelectMode = !multiSelectMode" class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5"
          :class="multiSelectMode ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' : 'bg-slate-800 text-slate-400 hover:text-slate-200 border border-slate-700 hover:bg-slate-700'">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"></path></svg>
          <span>多选</span>
        </button>
        <div class="w-px h-5 bg-slate-700 mx-1"></div>
        <!-- New Folder -->
        <button @click="showNewFolderInput = true" :disabled="selectedHasItems || multiSelectMode" class="px-3 py-1.5 rounded-lg text-xs font-bold text-slate-400 hover:text-slate-200 bg-slate-800 hover:bg-slate-700 border border-slate-700 transition-all flex items-center space-x-1.5 disabled:opacity-40 disabled:cursor-not-allowed">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
          <span>新建文件夹</span>
        </button>
        <!-- Download Release -->
        <button @click="downloadRelease" :disabled="multiSelectMode" class="px-3 py-1.5 rounded-lg text-xs font-bold text-slate-400 hover:text-sky-400 bg-slate-800 hover:bg-slate-700 border border-slate-700 transition-all flex items-center space-x-1.5 disabled:opacity-40 disabled:cursor-not-allowed">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"></path></svg>
          <span>下载发行包</span>
        </button>
        <!-- Rename -->
        <button @click="renameSelected" :disabled="!selectedIsSingleFile" class="px-3 py-1.5 rounded-lg text-xs font-bold text-slate-400 hover:text-amber-400 bg-slate-800 hover:bg-slate-700 border border-slate-700 transition-all flex items-center space-x-1.5 disabled:opacity-40 disabled:cursor-not-allowed">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path></svg>
          <span>重命名</span>
        </button>
        <!-- Download -->
        <button @click="downloadSelected" :disabled="!selectedIsSingleFile" class="px-3 py-1.5 rounded-lg text-xs font-bold text-slate-400 hover:text-emerald-400 bg-slate-800 hover:bg-slate-700 border border-slate-700 transition-all flex items-center space-x-1.5 disabled:opacity-40 disabled:cursor-not-allowed">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"></path></svg>
          <span>下载</span>
        </button>
        <!-- Upload -->
        <button @click="uploadInput?.click()" class="px-3 py-1.5 rounded-lg text-xs font-bold text-slate-400 hover:text-indigo-400 bg-slate-800 hover:bg-slate-700 border border-slate-700 transition-all flex items-center space-x-1.5">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path></svg>
          <span>上传</span>
        </button>
        <input type="file" ref="uploadInput" class="hidden" multiple @change="handleUploadSelect" />
        <div class="w-px h-5 bg-slate-700 mx-1"></div>
        <!-- Delete Selected -->
        <button @click="deleteSelected" :disabled="!selectedHasItems" class="px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center space-x-1.5"
          :class="selectedHasItems ? 'text-rose-400 hover:text-rose-300 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/30' : 'text-slate-500 bg-slate-800 border border-slate-700 disabled:cursor-not-allowed'">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
          <span>删除 {{ selectedFiles.size > 0 ? '(' + selectedFiles.size + ')' : '' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { apiListFiles, apiReadFile, apiMkdir, apiDeleteFile, apiRenameFile, apiCommit, apiPush, apiPacketUp, apiUploadFile, apiUploadPclPack, apiDownloadRelease } from '../api';

const props = defineProps<{ instanceId: string }>();
const emit = defineEmits(['navigate']);

const files = ref<any[]>([]);
const loading = ref(false);
const showNewFolderInput = ref(false);
const showPclModal = ref(false);
const newFolderName = ref('');
const pclInput = ref<HTMLInputElement | null>(null);
const uploadInput = ref<HTMLInputElement | null>(null);
const dragCounter = ref(0);
const isUploading = ref(false);
const uploadProgress = ref(0);
const currentFileName = ref('');
const searchQuery = ref('');
const selectedFiles = ref<Set<string>>(new Set());
const multiSelectMode = ref(false);

const selectedIsSingleFile = computed(() => {
  if (multiSelectMode.value) return false;
  if (selectedFiles.value.size !== 1) return false;
  const name = Array.from(selectedFiles.value)[0];
  const f = files.value.find(x => x.name === name);
  return f && !f.is_dir;
});
const selectedHasItems = computed(() => selectedFiles.value.size > 0);

// PCL-CE fuzzy search algorithm
function searchSimilarity(source: string, query: string): number {
  let qp = 0, lenSum = 0;
  source = source.toLowerCase().replace(/\s/g, '');
  query = query.toLowerCase().replace(/\s/g, '');
  const sourceLength = source.length, queryLength = query.length;
  if (queryLength === 0) return 1;

  while (qp < queryLength) {
    let sp = 0, lenMax = 0, spMax = 0;
    while (sp < source.length) {
      let len = 0;
      while ((qp + len) < queryLength && (sp + len) < source.length && source[sp + len] === query[qp + len]) {
        len++;
      }
      if (len > lenMax) {
        lenMax = len;
        spMax = sp;
      }
      sp += Math.max(1, len);
    }
    if (lenMax > 0) {
      source = source.substring(0, spMax) + source.substring(spMax + lenMax);
      let incWeight = (Math.pow(1.4, 3 + lenMax) - 3.6);
      incWeight *= 1 + 0.3 * Math.max(0, 3 - Math.abs(qp - spMax));
      lenSum += incWeight;
    }
    qp += Math.max(1, lenMax);
  }
  return (lenSum / queryLength) * (3 / Math.pow(sourceLength + 15, 0.5)) * (queryLength <= 2 ? 3 - queryLength : 1);
}

const filteredFiles = computed(() => {
  if (!searchQuery.value.trim()) {
    return [...files.value].sort((a, b) => {
      if (a.is_dir && !b.is_dir) return -1;
      if (!a.is_dir && b.is_dir) return 1;
      return a.name.localeCompare(b.name);
    });
  }

  const query = searchQuery.value.trim().toLowerCase();
  const queryParts = query.split(/\s+/);

  const results = files.value.map(file => {
    const similarity = searchSimilarity(file.name, query);
    const sourceLower = file.name.toLowerCase().replace(/\s/g, '');
    let absoluteRight = true;
    for (const part of queryParts) {
      if (!sourceLower.includes(part)) {
        absoluteRight = false;
        break;
      }
    }
    return { ...file, _similarity: similarity, _absoluteRight: absoluteRight };
  });

  return results
    .filter(f => f._absoluteRight || f._similarity > 0.1)
    .sort((a, b) => {
      if (a._absoluteRight && !b._absoluteRight) return -1;
      if (!a._absoluteRight && b._absoluteRight) return 1;
      if (a._similarity !== b._similarity) return b._similarity - a._similarity;
      if (a.is_dir && !b.is_dir) return -1;
      if (!a.is_dir && b.is_dir) return 1;
      return a.name.localeCompare(b.name);
    });
});

const vFocus = { mounted: (el: HTMLElement) => el.focus() };
const currentPath = ref('');
const currentPathSegments = ref<string[]>([]);
const visibleBreadcrumbs = computed(() => {
  const segments = currentPathSegments.value;
  if (segments.length <= 2) {
    return segments.map((text, index) => ({ text, index, isEllipsis: false }));
  }
  return [
    { text: '...', index: -1, isEllipsis: true },
    { text: segments[segments.length - 2], index: segments.length - 2, isEllipsis: false },
    { text: segments[segments.length - 1], index: segments.length - 1, isEllipsis: false }
  ];
});
watch(currentPathSegments, (s) => { currentPath.value = s.join('/'); selectedFiles.value.clear(); fetchFiles(); }, { deep: true });


async function fetchFiles() {
  if (!props.instanceId || isUploading.value) return;
  loading.value = true;
  try {
    const res = await apiListFiles(props.instanceId, currentPath.value);
    files.value = Array.isArray(res.data) ? res.data : (Array.isArray(res) ? res : []);
  } catch (e) { console.error(e); } finally { loading.value = false; }
}

// --- Selection ---
function isSelected(name: string): boolean {
  return selectedFiles.value.has(name);
}

function toggleFileSelection(name: string) {
  if (multiSelectMode.value) {
    const s = new Set(selectedFiles.value);
    if (s.has(name)) {
      s.delete(name);
    } else {
      s.add(name);
    }
    selectedFiles.value = s;
  } else {
    const s = new Set<string>();
    if (!selectedFiles.value.has(name)) {
      s.add(name);
    }
    selectedFiles.value = s;
  }
}

// --- Double-click to open ---
function handleFileOpen(file: any) {
  if (file.is_dir) {
    currentPathSegments.value = [...currentPathSegments.value, file.name];
  } else {
    const fullPath = (currentPath.value ? currentPath.value + '/' : '') + file.name;
    apiReadFile(props.instanceId, fullPath).then(content => emit('navigate', 'editor', { instanceId: props.instanceId, path: fullPath, content }));
  }
}

// --- Bottom Bar Actions ---
function renameSelected() {
  const names = Array.from(selectedFiles.value);
  if (names.length !== 1) return;
  const oldName = names[0];
  const newName = prompt('重命名为：', oldName);
  if (!newName || newName === oldName) return;
  if (newName.includes('/') || newName.includes('\\') || newName.includes(':') || newName.includes('..')) {
    alert('文件名无效，不能包含目录分隔符或冒号');
    return;
  }
  const src = (currentPath.value ? currentPath.value + '/' : '') + oldName;
  const dst = (currentPath.value ? currentPath.value + '/' : '') + newName;
  apiRenameFile(props.instanceId, src, dst).then(() => {
    selectedFiles.value = new Set([newName]);
    fetchFiles();
  }).catch(e => alert('重命名失败: ' + (e.message || e)));
}

function downloadSelected() {
  const names = Array.from(selectedFiles.value);
  if (names.length !== 1) return;
  const name = names[0];
  const fullPath = (currentPath.value ? currentPath.value + '/' : '') + name;
  apiReadFile(props.instanceId, fullPath).then(content => {
    const blob = new Blob([content], { type: 'application/octet-stream' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = name;
    a.click();
    URL.revokeObjectURL(url);
  }).catch(e => alert('下载失败: ' + (e.message || e)));
}

async function downloadRelease() {
  try {
    await apiDownloadRelease(props.instanceId)
  } catch (e: any) {
    alert('下载发行包失败: ' + (e.message || e))
  }
}

function deleteSingleFile(name: string) {
  if (!confirm(`确定要物理删除 [${name}] 吗？`)) return;
  const path = (currentPath.value ? currentPath.value + '/' : '') + name;
  apiDeleteFile(props.instanceId, path).then(() => fetchFiles());
}

function deleteSelected() {
  const names = Array.from(selectedFiles.value);
  if (names.length === 0) return;
  if (!confirm(`确定要物理删除 ${names.length} 个文件/文件夹吗？\n\n${names.join('\n')}`)) return;
  Promise.allSettled(names.map(name => {
    const path = (currentPath.value ? currentPath.value + '/' : '') + name;
    return apiDeleteFile(props.instanceId, path);
  })).then(results => {
    const failed = results.filter(r => r.status === 'rejected').length;
    if (failed > 0) alert(`删除完成：${names.length - failed} 成功，${failed} 失败`);
    selectedFiles.value.clear();
    fetchFiles();
  });
}

function handleUploadSelect(e: any) {
  if (e.target.files) uploadFiles(e.target.files);
  e.target.value = '';
}

// --- Existing functions ---
async function uploadFiles(fileList: FileList | File[], isPcl = false) {
  if (isUploading.value) return;
  isUploading.value = true;
  const filesArray = Array.from(fileList);
  for (let i = 0; i < filesArray.length; i++) {
    const file = filesArray[i];
    currentFileName.value = file.name;
    try {
      const onProgress = (pct: number) => {
        uploadProgress.value = (i / filesArray.length * 100) + (pct / filesArray.length);
      };
      if (isPcl) {
        await apiUploadPclPack(props.instanceId, file, onProgress);
      } else {
        await apiUploadFile(props.instanceId, currentPath.value, file, onProgress);
      }
    } catch (e) { console.error('Upload failed:', e); }
  }
  isUploading.value = false; showPclModal.value = false; fetchFiles();
  if (isPcl && filesArray.length > 0) {
    emit('navigate', 'console', { instanceId: props.instanceId });
  }
}

function handlePclDrop(e: DragEvent) { if (e.dataTransfer?.files) uploadFiles(e.dataTransfer.files, true); }
function handlePclSelect(e: any) { if (e.target.files) uploadFiles(e.target.files, true); }
function handleDrop(e: DragEvent) { dragCounter.value = 0; if (e.dataTransfer?.files) uploadFiles(e.dataTransfer.files); }
function handleDragEnter() { dragCounter.value++; }
function handleDragLeave() { dragCounter.value--; }

async function createFolder() {
  const path = (currentPath.value ? currentPath.value + '/' : '') + newFolderName.value;
  await apiMkdir(props.instanceId, path);
  showNewFolderInput.value = false; newFolderName.value = ''; fetchFiles();
}

async function triggerCommit() {
  await apiCommit(props.instanceId);
  emit('navigate', 'console', { instanceId: props.instanceId });
}

async function triggerPush() {
  await apiPush(props.instanceId);
  emit('navigate', 'console', { instanceId: props.instanceId });
}

async function triggerPacketUp() {
  await apiPacketUp(props.instanceId);
  emit('navigate', 'console', { instanceId: props.instanceId });
}

function goToParentDir() {
  if (currentPathSegments.value.length > 0) {
    currentPathSegments.value = currentPathSegments.value.slice(0, -1);
  }
}
function navigateToBreadcrumb(i: number) { currentPathSegments.value = currentPathSegments.value.slice(0, i + 1); }
function isEditable(n: string) { return ['yml', 'yaml', 'properties', 'json', 'toml', 'txt', 'log'].includes(n.split('.').pop()?.toLowerCase() || ''); }
function formatSize(b: number) { if (b === 0) return '0 B'; const k = 1024; const i = Math.floor(Math.log(b) / Math.log(k)); return parseFloat((b / Math.pow(k, i)).toFixed(2)) + ' ' + ['B', 'KB', 'MB', 'GB'][i]; }
function formatDate(s: string) { return new Date(s).toLocaleString(); }
onMounted(fetchFiles);
</script>
