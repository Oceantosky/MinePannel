<template>
  <div class="flex-1 flex flex-col overflow-hidden relative bg-[#0a0f1c]">

    <!-- Ambient Background -->
    <div class="absolute top-0 right-0 -mr-48 -mt-48 w-[40rem] h-[40rem] bg-indigo-500/10 rounded-full blur-3xl pointer-events-none opacity-50"></div>

    <!-- Header -->
    <header class="h-20 border-b border-slate-800/80 bg-slate-900/40 backdrop-blur-xl flex items-center justify-between px-10 z-10 shrink-0">
      <div class="flex items-center">
        <button @click="$emit('navigate', 'explorer', { instanceId })" class="text-slate-400 hover:text-white transition-colors mr-4 p-1.5 rounded hover:bg-slate-800">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path></svg>
        </button>
        <div class="w-10 h-10 bg-indigo-500/20 rounded-xl flex items-center justify-center mr-4 border border-indigo-500/30">
          <svg class="w-5 h-5 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"></path></svg>
        </div>
        <div>
          <h2 class="text-xl font-black text-slate-100 tracking-tight">{{ contentTypeLabel }}市场</h2>
          <p class="text-[11px] text-slate-500 mt-1 font-medium tracking-wide">
            {{ folderHint ? `当前位于 ${folderHint} 文件夹 · ` : '' }}云端引擎直连海外骨干网光速拉取
          </p>
        </div>
      </div>
      <!-- Source Selector -->
      <div class="flex items-center gap-1 bg-slate-800/50 rounded-xl p-1 border border-slate-700/50 shadow-inner">
        <button @click="switchSource('modrinth')"
          class="px-3 py-1.5 text-[10px] font-black uppercase tracking-widest rounded-lg transition-all"
          :class="searchSource === 'modrinth' ? 'bg-indigo-500/20 text-indigo-400 shadow-sm' : 'text-slate-500 hover:text-slate-300'">Modrinth</button>
        <button @click="switchSource('curseforge')"
          class="px-3 py-1.5 text-[10px] font-black uppercase tracking-widest rounded-lg transition-all"
          :class="searchSource === 'curseforge' ? 'bg-orange-500/20 text-orange-400 shadow-sm' : 'text-slate-500 hover:text-slate-300'">CurseForge</button>
        <button @click="switchSource('all')"
          class="px-3 py-1.5 text-[10px] font-black uppercase tracking-widest rounded-lg transition-all"
          :class="searchSource === 'all' ? 'bg-emerald-500/20 text-emerald-400 shadow-sm' : 'text-slate-500 hover:text-slate-300'">全部</button>
      </div>
    </header>

    <div class="flex-1 overflow-y-auto custom-scrollbar p-10 z-10 relative">
      <!-- Search Bar -->
      <div class="max-w-4xl mx-auto mb-10">
        <div class="relative group">
          <div class="absolute inset-y-0 left-0 pl-5 flex items-center pointer-events-none">
            <svg class="w-5 h-5" :class="isSearching ? 'text-indigo-400 animate-pulse' : 'text-slate-500 group-focus-within:text-indigo-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
          </div>
          <input
            v-model="searchQuery"
            @input="onSearchInput"
            @keyup.enter="searchMods"
            type="text"
            class="block w-full pl-14 pr-36 py-5 bg-slate-900 border border-slate-700/50 rounded-2xl leading-5 text-slate-200 placeholder-slate-500 focus:outline-none focus:bg-slate-800 focus:border-indigo-500/50 focus:ring-1 focus:ring-indigo-500/50 transition-all shadow-inner text-lg font-medium"
            :placeholder="searchPlaceholder"
          >
          <div class="absolute inset-y-0 right-3 flex items-center space-x-2">
            <button v-if="searchQuery" @click="clearSearch" class="p-2 text-slate-500 hover:text-slate-300 transition-colors" title="清除">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
            </button>
            <button @click="searchMods" :disabled="isSearching" class="bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white px-6 py-2.5 rounded-xl font-bold text-sm shadow-lg shadow-indigo-600/20 transition-all flex items-center">
              <svg v-if="isSearching" class="w-5 h-5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
              <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
            </button>
          </div>
        </div>

        <!-- Type filter chips -->
        <div class="flex items-center space-x-4 mt-6 ml-2">
          <button @click="switchType('mod')" :class="projectType === 'mod' ? 'bg-slate-800 text-white border-slate-600 shadow-sm' : 'text-slate-400 hover:text-slate-300 border-transparent hover:bg-slate-800/50'" class="px-5 py-2 rounded-xl text-xs font-bold transition-all border">模组</button>
          <button @click="switchType('resourcepack')" :class="projectType === 'resourcepack' ? 'bg-slate-800 text-white border-slate-600 shadow-sm' : 'text-slate-400 hover:text-slate-300 border-transparent hover:bg-slate-800/50'" class="px-5 py-2 rounded-xl text-xs font-bold transition-all border">资源包</button>
          <button @click="switchType('shader')" :class="projectType === 'shader' ? 'bg-slate-800 text-white border-slate-600 shadow-sm' : 'text-slate-400 hover:text-slate-300 border-transparent hover:bg-slate-800/50'" class="px-5 py-2 rounded-xl text-xs font-bold transition-all border">光影</button>
          <span class="text-slate-600 text-xs ml-4">按组件类型筛选</span>
        </div>
      </div>

      <!-- Recommendations / Results -->
      <div class="max-w-4xl mx-auto">

        <!-- Loading skeleton -->
        <div v-if="isSearching && searchResults.length === 0" class="space-y-4">
          <div v-for="i in 5" :key="i" class="bg-slate-800/30 border border-slate-700/30 rounded-2xl p-6 flex gap-6 animate-pulse">
            <div class="w-20 h-20 rounded-2xl bg-slate-700/30 shrink-0"></div>
            <div class="flex-1 space-y-3">
              <div class="h-5 bg-slate-700/30 rounded w-1/3"></div>
              <div class="h-3 bg-slate-700/20 rounded w-2/3"></div>
              <div class="h-3 bg-slate-700/20 rounded w-1/2"></div>
            </div>
          </div>
        </div>

        <!-- CurseForge Error Banner -->
        <div v-if="curseforgeError" class="max-w-4xl mx-auto mb-4 p-4 bg-orange-500/10 border border-orange-500/30 rounded-2xl flex items-start gap-3">
          <svg class="w-5 h-5 text-orange-400 shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4.5c-.77-.833-2.694-.833-3.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z"></path></svg>
          <div>
            <p class="text-sm font-bold text-orange-300">CurseForge 搜索异常</p>
            <p class="text-xs text-orange-400/80 mt-1">{{ curseforgeError }}</p>
          </div>
          <button @click="curseforgeError = ''" class="ml-auto p-1 text-orange-500 hover:text-orange-300 shrink-0">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
          </button>
        </div>

        <!-- Results list -->
        <div v-if="searchResults.length > 0" class="space-y-3">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-sm font-bold text-slate-400">
              {{ hasSearched ? `"${lastSearchQuery}" 的搜索结果` : `🔥 热门推荐 · ${contentTypeLabel}` }}
              <span class="text-slate-600 font-normal ml-2">({{ searchResults.length }} 项)</span>
            </h3>
          </div>

          <div v-for="mod in searchResults" :key="mod.project_id" @click="selectMod(mod)" class="bg-slate-800/40 border border-slate-700/50 rounded-2xl p-5 flex gap-5 hover:bg-slate-800 transition-all hover:border-indigo-500/40 group cursor-pointer relative overflow-hidden">
            <!-- Icon -->
            <div class="w-16 h-16 shrink-0 bg-slate-900 rounded-2xl border border-slate-700/50 overflow-hidden shadow-inner flex items-center justify-center relative z-10">
              <img v-if="mod.icon_url" :src="mod.icon_url" :alt="mod.title" class="w-full h-full object-cover" loading="lazy">
              <svg v-else class="w-8 h-8 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"></path></svg>
            </div>

            <!-- Content -->
            <div class="flex-1 min-w-0 relative z-10">
              <div class="flex items-start justify-between gap-4">
                <div class="min-w-0">
                  <h3 class="text-base font-black text-slate-100 truncate group-hover:text-indigo-400 transition-colors">{{ mod.title }}</h3>
                  <p class="text-xs text-slate-500 font-medium mt-0.5">{{ mod.author }}</p>
                </div>
                <button
                  @click.stop="selectMod(mod)"
                  class="shrink-0 px-5 py-2 bg-indigo-600/90 hover:bg-indigo-500 text-white rounded-xl text-xs font-bold transition-all shadow-lg shadow-indigo-600/20 flex items-center"
                >
                  <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"></path></svg>
                  部署
                </button>
              </div>
              <p class="text-sm text-slate-400 mt-2 line-clamp-2 leading-relaxed">{{ mod.description }}</p>
              <div class="mt-3 flex gap-3 text-[10px] font-black uppercase tracking-widest text-slate-500">
                <span class="flex items-center"><svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path></svg> {{ formatNumber(mod.downloads) }}</span>
                <span class="flex items-center"><svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-12 0 9 9 0 0112 0z"></path></svg> {{ formatDate(mod.date_modified) }}</span>
                <span v-if="mod.categories?.length" class="flex items-center gap-1">
                  <span v-for="cat in mod.categories.slice(0, 2)" :key="cat" class="px-1.5 py-0.5 rounded bg-slate-700/50 text-slate-400 normal-case">{{ cat }}</span>
                </span>
                <span v-if="mod._source" class="ml-auto px-1.5 py-0.5 rounded text-[9px] font-black uppercase tracking-wider" :class="mod._source === 'curseforge' ? 'bg-orange-500/20 text-orange-400' : 'bg-indigo-500/20 text-indigo-400'">{{ mod._source === 'curseforge' ? 'CurseForge' : 'Modrinth' }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Empty: no results -->
        <div v-else-if="!isSearching && hasSearched" class="text-center py-20">
          <svg class="w-16 h-16 mx-auto text-slate-700 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
          <p class="text-xl font-bold text-slate-500">没有找到匹配的结果</p>
          <p class="text-sm text-slate-600 mt-2">尝试更换关键词或组件类型</p>
        </div>

        <!-- Initial state (loading recommendations) -->
        <div v-if="!hasSearched && searchResults.length === 0 && !isSearching && !initialLoaded" class="text-center py-32 flex flex-col items-center">
          <div class="w-24 h-24 mb-6 rounded-full bg-slate-800/50 border-2 border-slate-700/50 flex items-center justify-center">
            <svg class="w-12 h-12 text-slate-600 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
          </div>
          <p class="text-2xl font-black text-slate-400 mb-2">正在加载热门{{ contentTypeLabel }}...</p>
          <p class="text-sm font-bold text-slate-600 uppercase tracking-widest">{{ loadingSourceText }}</p>
        </div>
      </div>
    </div>

    <!-- ====== Version Selection Modal ====== -->
    <div v-if="selectedMod" class="fixed inset-0 z-[100] flex items-center justify-center p-4">
      <div class="absolute inset-0 bg-slate-950/80 backdrop-blur-xl" @click="!deployingVersion && (selectedMod = null)"></div>
      <div class="bg-slate-900 border border-indigo-500/20 w-full max-w-2xl rounded-3xl shadow-2xl z-10 overflow-hidden transform transition-all max-h-[75vh] h-[75vh] flex flex-col">

        <!-- Modal Header -->
        <div class="p-6 border-b border-slate-800 flex items-center gap-4 shrink-0">
          <img v-if="selectedMod.icon_url && getSafeIconUrl(selectedMod.icon_url)" :src="getSafeIconUrl(selectedMod.icon_url)" class="w-12 h-12 rounded-xl border border-slate-700" loading="lazy">
          <div class="min-w-0 flex-1">
            <h3 class="text-lg font-black text-white truncate">{{ selectedMod.title }}</h3>
            <p class="text-xs text-slate-400">by {{ selectedMod.author }} · 选择版本并点击「部署」</p>
          </div>
          <button @click="selectedMod = null" class="p-2 text-slate-500 hover:text-white hover:bg-slate-800 rounded-lg transition-colors">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
          </button>
        </div>

        <!-- Filter Bar (outside scrollable area → dropdowns never clipped) -->
        <div v-if="modVersions.length > 0" class="flex items-center gap-4 px-6 py-4 border-b border-slate-800/50 shrink-0">
          <!-- Loader Dropdown -->
          <div class="relative" ref="loaderDropdownRef">
            <button @click="showLoaderDropdown = !showLoaderDropdown"
              class="flex items-center gap-2 px-4 py-2.5 bg-slate-800 border border-slate-700/50 rounded-xl text-sm font-bold text-slate-200 hover:border-slate-600 transition-all min-w-[130px]">
              <span>{{ selectedLoaderFilter ? loaderDisplay(selectedLoaderFilter) : '全部加载器' }}</span>
              <svg class="w-4 h-4 ml-auto text-slate-500 transition-transform" :class="showLoaderDropdown ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
            </button>
            <div v-if="showLoaderDropdown" class="absolute top-full left-0 mt-1.5 w-full min-w-[140px] bg-slate-800 border border-slate-700/50 rounded-xl overflow-hidden shadow-2xl z-20 py-1">
              <button v-for="opt in availableLoaderOptions" :key="opt.value" @click="selectLoader(opt.value)" class="w-full text-left px-4 py-2.5 text-sm font-bold transition-colors" :class="selectedLoaderFilter === opt.value ? 'text-indigo-400 bg-indigo-500/10' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-700'">{{ opt.label }}</button>
            </div>
          </div>
          <!-- MC Version Dropdown -->
          <div class="relative flex-1 max-w-xs" ref="versionDropdownRef">
            <button @click="showVersionDropdown = !showVersionDropdown"
              class="flex items-center gap-2 px-4 py-2.5 bg-slate-800 border border-slate-700/50 rounded-xl text-sm font-bold text-slate-200 hover:border-slate-600 transition-all min-w-[180px] w-full">
              <span>{{ selectedVersionFilter || '全部版本' }}</span>
              <svg class="w-4 h-4 ml-auto text-slate-500 transition-transform" :class="showVersionDropdown ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
            </button>
            <div v-if="showVersionDropdown" class="absolute top-full left-0 mt-1.5 w-full min-w-[200px] bg-slate-800 border border-slate-700/50 rounded-xl overflow-hidden shadow-2xl z-20 py-1 max-h-60 overflow-y-auto custom-scrollbar">
              <button @click="selectVersion('')" class="w-full text-left px-4 py-2.5 text-sm font-bold transition-colors" :class="!selectedVersionFilter ? 'text-indigo-400 bg-indigo-500/10' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-700'">全部</button>
              <button v-for="ver in versionOptions" :key="ver" @click="selectVersion(ver)" class="w-full text-left px-4 py-2.5 text-sm font-bold transition-colors" :class="selectedVersionFilter === ver ? 'text-indigo-400 bg-indigo-500/10' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-700'">{{ ver }}</button>
            </div>
          </div>
          <span class="text-[11px] text-slate-600 font-mono ml-auto">{{ versionGroups.length }} 组</span>
        </div>

        <!-- Scrollable Version Content -->
        <div class="flex-1 overflow-y-auto custom-scrollbar version-browser-scroll">
          <div v-if="isLoadingVersions" class="py-16 text-center">
            <svg class="w-8 h-8 mx-auto animate-spin text-indigo-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
            <p class="text-slate-500 mt-4 font-bold text-sm">正在拉取版本列表...</p>
          </div>

          <template v-else-if="modVersions.length > 0">
            <div class="p-6 space-y-3">
              <div v-if="versionGroups.length === 0" class="py-12 text-center text-slate-500 font-bold text-sm">
                没有匹配的版本组合
              </div>

              <div v-for="(group, gi) in versionGroups" :key="gi" class="bg-slate-800/20 border border-slate-700/40 rounded-2xl overflow-hidden transition-all">
                <div @click="toggleGroup(gi)" class="flex items-center gap-3 px-5 py-3.5 cursor-pointer hover:bg-slate-700/20 transition-colors select-none">
                  <span class="px-2.5 py-1 rounded-lg text-[11px] font-black uppercase tracking-wider shrink-0"
                    :class="group.loader === 'fabric' ? 'bg-orange-500/20 text-orange-400' : group.loader === 'forge' ? 'bg-blue-500/20 text-blue-400' : group.loader === 'neoforge' ? 'bg-green-500/20 text-green-400' : group.loader === 'quilt' ? 'bg-purple-500/20 text-purple-400' : 'bg-slate-600/30 text-slate-400'"
                  >{{ group.loader === 'fabric' ? 'Fabric' : group.loader === 'forge' ? 'Forge' : group.loader === 'neoforge' ? 'NeoForge' : group.loader === 'quilt' ? 'Quilt' : group.loader }}</span>
                  <span class="text-sm font-bold text-slate-200">{{ group.mcVer === 'SNAPSHOT' ? '快照版' : group.mcVer }}</span>
                  <span class="text-[10px] font-mono text-slate-600">{{ group.sortedVersions.length }} 个版本</span>
                  <span class="ml-auto text-slate-600 transition-transform duration-200" :class="expandedGroups.has(gi) ? 'rotate-180' : ''">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
                  </span>
                </div>

                <div v-if="expandedGroups.has(gi)" class="px-5 pb-4 space-y-2 animate-in slide-in-from-top-2 duration-200">
                  <div v-for="ver in group.sortedVersions" :key="ver.id" class="flex items-center gap-3 px-4 py-3 bg-slate-800/40 border border-slate-700/30 rounded-xl hover:bg-slate-700/30 transition-all group/ver">
                    <div class="flex-1 min-w-0">
                      <div class="flex items-center gap-2 flex-wrap">
                        <span class="text-sm font-bold text-slate-200 truncate">{{ ver.version_number || ver.name }}</span>
                        <span class="px-1.5 py-0.5 rounded text-[8px] font-black uppercase tracking-wider"
                          :class="ver.version_type === 'release' ? 'bg-emerald-500/20 text-emerald-400' : ver.version_type === 'beta' ? 'bg-amber-500/20 text-amber-400' : ver.version_type === 'alpha' ? 'bg-rose-500/20 text-rose-400' : 'bg-slate-600/30 text-slate-400'"
                        >{{ ver.version_type === 'release' ? '正式版' : ver.version_type === 'beta' ? '测试版' : ver.version_type === 'alpha' ? '预览版' : ver.version_type }}</span>
                        <span class="text-[10px] text-slate-500 font-mono">{{ formatDateOnly(ver.date_published) }}</span>
                      </div>
                    </div>
                    <div class="flex items-center gap-3 shrink-0">
                      <div class="text-right hidden lg:block">
                        <p class="text-[10px] font-mono text-slate-500">{{ formatFileSize(ver.files?.[0]?.size) }}</p>
                      </div>
                      <button @click.stop="deployVersion(ver)" :disabled="!!deployingVersion"
                        class="px-4 py-2 rounded-xl text-[11px] font-black transition-all shadow-lg whitespace-nowrap"
                        :class="deployingVersion === ver.id ? 'bg-indigo-500/30 text-indigo-300 animate-pulse' : deploySuccessful && deployingVersion === ver.id ? 'bg-emerald-500/20 text-emerald-400' : 'bg-indigo-600/80 hover:bg-indigo-500 text-white shadow-indigo-600/20'">
                        <span v-if="deployingVersion === ver.id && !deploySuccessful && !deployError">部署中</span>
                        <span v-else-if="deployingVersion === ver.id && deploySuccessful">已部署 ✓</span>
                        <span v-else>部署</span>
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </template>

          <div v-else class="py-16 text-center text-slate-500 font-bold text-sm">
            没有找到可用的版本文件。
          </div>
        </div>

        <!-- Modal Footer -->
        <div class="p-4 border-t border-slate-800 flex items-center justify-between text-xs text-slate-500 px-6 shrink-0">
          <span>目标路径: <span class="font-mono text-slate-400">{{ targetFolder }}</span></span>
          <div class="flex items-center gap-2">
            <span class="w-2 h-2 rounded-full" :class="deploySuccessful ? 'bg-emerald-500' : 'bg-slate-600'"></span>
            <span>{{ deploySuccessful ? '部署完成 ✓' : '选择版本后点击右侧「部署」' }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ====== Global Deploy Progress Toast ====== -->
    <div v-if="showDeployToast" class="fixed bottom-8 right-8 z-[200] animate-in slide-in-from-bottom-4 duration-300">
      <div class="bg-slate-900 border border-slate-700/50 rounded-2xl shadow-2xl p-5 max-w-sm backdrop-blur-xl">
        <div class="flex items-center gap-4">
          <div v-if="deployError" class="w-10 h-10 rounded-xl bg-rose-500/20 flex items-center justify-center shrink-0">
            <svg class="w-5 h-5 text-rose-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
          </div>
          <div v-else-if="deploySuccessful" class="w-10 h-10 rounded-xl bg-emerald-500/20 flex items-center justify-center shrink-0">
            <svg class="w-5 h-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>
          </div>
          <div v-else class="w-10 h-10 rounded-xl bg-indigo-500/20 flex items-center justify-center shrink-0">
            <svg class="w-5 h-5 text-indigo-400 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
          </div>
          <div class="min-w-0">
            <p class="text-sm font-bold text-slate-200">{{ deployStatusMessage }}</p>
            <p v-if="deployError" class="text-xs text-rose-400 mt-0.5 truncate">{{ deployError }}</p>
          </div>
          <button @click="showDeployToast = false" class="p-1 text-slate-600 hover:text-slate-400 shrink-0">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { apiCurseProxy, apiModrinthProxy, apiMarketDownload, apiCommit } from '../api';

function getSafeIconUrl(url: string): string {
  if (!url) return '';
  try {
    const parsed = new URL(url);
    const host = parsed.hostname;
    const allowed = ['cdn.modrinth.com', 'media.forgecdn.net', 'githubusercontent.com', 'avatars.githubusercontent.com'];
    if (allowed.some(d => host === d || host.endsWith('.' + d))) {
      return url;
    }
  } catch (e) {}
  return '';
}

const props = defineProps<{
  instanceId: string;
  initialType?: string;
}>();

const emit = defineEmits(['navigate']);

// --- Type context ---
const projectType = ref('mod');
const initialLoaded = ref(false);

const contentTypeLabel = computed(() => {
  switch (projectType.value) {
    case 'resourcepack': return '资源包';
    case 'shader': return '光影';
    default: return '模组';
  }
});

const folderHint = computed(() => {
  if (props.initialType) {
    switch (props.initialType) {
      case 'mod': return 'mods';
      case 'resourcepack': return 'resourcepacks';
      case 'shader': return 'shaderpacks';
    }
  }
  return '';
});

const targetFolder = computed(() => {
  switch (projectType.value) {
    case 'resourcepack': return `.minecraft/resourcepacks/`;
    case 'shader': return `.minecraft/shaderpacks/`;
    default: return `.minecraft/mods/`;
  }
});

const searchPlaceholder = computed(() => {
  return `搜索${contentTypeLabel.value}，例如 ${exampleSearch}`;
});

const exampleSearch = computed(() => {
  switch (projectType.value) {
    case 'resourcepack': return 'Faithful, Bare Bones...';
    case 'shader': return 'Complementary, BSL...';
    default: return 'JEI, Create, Sodium...';
  }
});

// --- Search ---
const searchQuery = ref('');
const lastSearchQuery = ref('');
const isSearching = ref(false);
const hasSearched = ref(false);
const searchResults = ref<any[]>([]);
const searchSource = ref<'modrinth' | 'curseforge' | 'all'>('all');

const loadingSourceText = computed(() => {
  switch (searchSource.value) {
    case 'curseforge': return '检索 CurseForge 全球库中';
    case 'all': return '检索多平台数据源中';
    default: return '检索 Modrinth 全球库中';
  }
});

let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;

function onSearchInput() {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
  if (searchQuery.value.trim().length >= 2) {
    searchDebounceTimer = setTimeout(() => {
      searchMods();
    }, 400);
  }
}

async function searchMods() {
  if (!searchQuery.value.trim() && !hasSearched.value) {
    await fetchRecommendations();
    return;
  }
  if (!searchQuery.value.trim()) return;

  isSearching.value = true;
  hasSearched.value = true;
  lastSearchQuery.value = searchQuery.value;
  searchResults.value = [];

  const promises: Promise<void>[] = [];
  if (searchSource.value === 'modrinth' || searchSource.value === 'all') {
    promises.push(searchModrinth());
  }
  if (searchSource.value === 'curseforge' || searchSource.value === 'all') {
    promises.push(searchCurseforge());
  }
  await Promise.all(promises);
  // Dedup prioritizing Modrinth (matching PCL-CE keeping Modrinth if both exist)
  searchResults.value.sort((a: any, b: any) => a._source === 'modrinth' ? -1 : b._source === 'modrinth' ? 1 : 0);
  searchResults.value = dedupResults(searchResults.value);
  // Apply blended score sort (mimicking PCL)
  searchResults.value.sort((a: any, b: any) => calculateScore(b, true) - calculateScore(a, true));
  isSearching.value = false;
}

function calculateScore(mod: any, isSearch: boolean): number {
  let multiplier = 1;
  const isModrinth = mod._source === 'modrinth';
  if (projectType.value === 'mod') {
    multiplier = isModrinth ? 7 : 1;
  } else if (projectType.value === 'resourcepack' || projectType.value === 'shader') {
    multiplier = isModrinth ? 5 : 1;
  }
  let downloadsScore = Math.log10(Math.max(mod.downloads || 1, 1) * multiplier) / 9;
  let relevanceScore = 0;
  if (isSearch) {
    relevanceScore = Math.max(0, 1 - ((mod._original_index || 0) * 0.05));
  }
  return relevanceScore + downloadsScore;
}

function getProcessedQuery(raw: string): string {
  if (!raw) return '';
  const lower = raw.toLowerCase();
  const rawNoSpace = lower.replace(/\s+/g, '');
  if (rawNoSpace.includes('optiforge')) return 'optiforge';
  if (rawNoSpace.includes('optifabric')) return 'optifabric';

  const ignoreWords = ['forge', 'fabric', 'for', 'mod', 'quilt'];
  const words = lower.split(/\s+/).map(w => w.replace(/^[\[\]]+|[\[\]]+$/g, '')).filter(w => w && !ignoreWords.includes(w));
  return Array.from(new Set(words)).join(' ') || raw;
}

async function searchModrinth() {
  try {
    const processedQuery = getProcessedQuery(searchQuery.value);
    const facets = `[["project_type:${projectType.value}"]]`;
    const path = `/v2/search?query=${encodeURIComponent(processedQuery)}&facets=${encodeURIComponent(facets)}&limit=30`;
    const data = await apiModrinthProxy(path);
    const hits = ((data.data || data).hits || []).map((h: any, i: number) => ({ ...h, _source: 'modrinth', _original_index: i }));
    searchResults.value = [...searchResults.value, ...hits];
  } catch (error) {
    console.error('Modrinth search failed:', error);
  }
}

const CURSEFORGE_CLASS: Record<string, number> = { mod: 6, resourcepack: 12, shader: 6552 };

const curseforgeError = ref('');

async function searchCurseforge() {
  curseforgeError.value = '';
  try {
    const processedQuery = getProcessedQuery(searchQuery.value);
    const classId = CURSEFORGE_CLASS[projectType.value] || 6;
    // sortField=2 (Popularity) aligns with PCL's default behavior, ensuring relevance via fuzzy matching
    const cfPath = `/v1/mods/search?gameId=432&classId=${classId}&searchFilter=${encodeURIComponent(processedQuery)}&pageSize=30&sortField=2&sortOrder=desc`;
    const data = await apiCurseProxy(cfPath);
    const items = (data.data?.data || data.data || []).map((cfMod: any, i: number) => {
      const mod = normalizeCurseforgeMod(cfMod);
      mod._original_index = i;
      return mod;
    });
    searchResults.value = [...searchResults.value, ...items];
  } catch (error: any) {
    curseforgeError.value = error.message || 'CurseForge 搜索失败';
    console.error('CurseForge search failed:', error);
  }
}

function normalizeCurseforgeMod(cfMod: any): any {
  return {
    _source: 'curseforge',
    project_id: String(cfMod.id),
    title: cfMod.name,
    author: cfMod.authors?.[0]?.name || 'unknown',
    description: cfMod.summary || '',
    icon_url: cfMod.logo?.url || null,
    downloads: cfMod.downloadCount || 0,
    date_modified: cfMod.dateModified || '',
    categories: (cfMod.categories || []).map((c: any) => c.name),
  };
}

function clearSearch() {
  searchQuery.value = '';
  hasSearched.value = false;
  searchResults.value = [];
  fetchRecommendations();
}

function switchType(type: string) {
  if (projectType.value === type) return;
  projectType.value = type;
  searchQuery.value = '';
  hasSearched.value = false;
  searchResults.value = [];
  fetchRecommendations();
}

function switchSource(source: 'modrinth' | 'curseforge' | 'all') {
  if (searchSource.value === source) return;
  searchSource.value = source;
  searchQuery.value = '';
  hasSearched.value = false;
  searchResults.value = [];
  fetchRecommendations();
}

// --- Recommendations ---
async function fetchRecommendations() {
  isSearching.value = true;
  hasSearched.value = false;
  searchResults.value = [];
  initialLoaded.value = false;

  try {
    const promises: Promise<void>[] = [];
    if (searchSource.value === 'modrinth' || searchSource.value === 'all') {
      promises.push(fetchModrinthRecommendations());
    }
    if (searchSource.value === 'curseforge' || searchSource.value === 'all') {
      promises.push(fetchCurseforgeRecommendations());
    }
    await Promise.all(promises);
    // Dedup prioritizing Modrinth (matching PCL-CE)
    searchResults.value.sort((a: any, b: any) => a._source === 'modrinth' ? -1 : b._source === 'modrinth' ? 1 : 0);
    searchResults.value = dedupResults(searchResults.value);
    // Apply blended score sort
    searchResults.value.sort((a: any, b: any) => calculateScore(b, false) - calculateScore(a, false));
  } catch (error) {
    console.error('Failed to fetch recommendations:', error);
  } finally {
    isSearching.value = false;
    initialLoaded.value = true;
  }
}

async function fetchModrinthRecommendations() {
  try {
    const facets = `[["project_type:${projectType.value}"]]`;
    const path = `/v2/search?facets=${encodeURIComponent(facets)}&limit=10&index=downloads`;
    const data = await apiModrinthProxy(path);
    const hits = ((data.data || data).hits || []).map((h: any) => ({ ...h, _source: 'modrinth' }));
    searchResults.value = [...searchResults.value, ...hits];
  } catch (error) {
    console.error('Modrinth recommendations failed:', error);
  }
}

async function fetchCurseforgeRecommendations() {
  try {
    const classId = CURSEFORGE_CLASS[projectType.value] || 6;
    const cfPath = `/v1/mods/search?gameId=432&classId=${classId}&pageSize=10&sortField=6&sortOrder=desc`;
    const data = await apiCurseProxy(cfPath);
    const items = (data.data?.data || data.data || []).map(normalizeCurseforgeMod);
    searchResults.value = [...searchResults.value, ...items];
  } catch (error) {
    console.error('CurseForge recommendations failed:', error);
  }
}

// --- Version selection & deploy ---
const selectedMod = ref<any>(null);
const modVersions = ref<any[]>([]);
const isLoadingVersions = ref(false);
const deployingVersion = ref('');
const deploySuccessful = ref(false);
const showDeployToast = ref(false);
const deployStatusMessage = ref('');
const deployError = ref('');

// --- PCL-style dropdown filters & version grouping ---
const selectedLoaderFilter = ref('');
const selectedVersionFilter = ref('');
const showLoaderDropdown = ref(false);
const showVersionDropdown = ref(false);
const expandedGroups = ref(new Set<number>());
const loaderDropdownRef = ref<HTMLElement | null>(null);
const versionDropdownRef = ref<HTMLElement | null>(null);

const loaderOptions = [
  { value: '', label: '全部加载器' },
  { value: 'fabric', label: 'Fabric' },
  { value: 'forge', label: 'Forge' },
  { value: 'neoforge', label: 'NeoForge' },
  { value: 'quilt', label: 'Quilt' },
];

function loaderDisplay(loader: string) {
  const found = loaderOptions.find(o => o.value === loader);
  return found ? found.label : loader;
}

function isSnapshot(ver: string): boolean {
  return /^\d+w/i.test(ver) || ver.includes('-snapshot') || ver.includes('-pre') || ver.includes('-rc');
}

const availableLoaderOptions = computed(() => {
  let pool = modVersions.value;
  if (selectedVersionFilter.value) {
    pool = pool.filter((v: any) =>
      selectedVersionFilter.value === '快照版'
        ? v.game_versions?.some((gv: string) => isSnapshot(gv))
        : v.game_versions?.includes(selectedVersionFilter.value)
    );
  }
  const present = new Set<string>();
  pool.forEach((v: any) => v.loaders?.forEach((l: string) => present.add(l)));
  return [
    { value: '', label: '全部加载器' },
    ...loaderOptions.filter(o => o.value && present.has(o.value)),
  ];
});

const versionOptions = computed(() => {
  let pool = modVersions.value;
  if (selectedLoaderFilter.value) {
    pool = pool.filter((v: any) => v.loaders?.includes(selectedLoaderFilter.value));
  }
  const numbered = new Set<string>();
  let hasSnapshot = false;
  pool.forEach((v: any) => {
    v.game_versions?.forEach((gv: string) => {
      if (isSnapshot(gv)) {
        hasSnapshot = true;
      } else {
        numbered.add(gv);
      }
    });
  });
  const result = Array.from(numbered).sort((a, b) => b.localeCompare(a, undefined, { numeric: true }));
  if (hasSnapshot) result.push('快照版');
  return result;
});

const versionGroups = computed(() => {
  const map = new Map<string, { loader: string; mcVer: string; sortedVersions: any[] }>();

  let filtered = modVersions.value;
  // Filter by loader
  if (selectedLoaderFilter.value) {
    filtered = filtered.filter((v: any) => v.loaders?.includes(selectedLoaderFilter.value));
  }
  // Filter by MC version group
  if (selectedVersionFilter.value) {
    if (selectedVersionFilter.value === '快照版') {
      filtered = filtered.filter((v: any) => v.game_versions?.some((gv: string) => isSnapshot(gv)));
    } else {
      filtered = filtered.filter((v: any) => v.game_versions?.includes(selectedVersionFilter.value));
    }
  }

  filtered.forEach((v: any) => {
    const loaders = v.loaders || ['unknown'];
    let mcVers = v.game_versions || ['unknown'];

    // When filtering by snapshot, use 'SNAPSHOT' as the group key
    if (selectedVersionFilter.value === '快照版') {
      mcVers = ['SNAPSHOT'];
    } else if (selectedVersionFilter.value) {
      mcVers = [selectedVersionFilter.value];
    }

    loaders.forEach((loader: string) => {
      mcVers.forEach((mcVer: string) => {
        const key = `${loader}|${mcVer}`;
        if (!map.has(key)) map.set(key, { loader, mcVer, sortedVersions: [] });
        if (!map.get(key)!.sortedVersions.some((ev: any) => ev.id === v.id)) {
          map.get(key)!.sortedVersions.push(v);
        }
      });
    });
  });

  return Array.from(map.values())
    .filter(g => g.sortedVersions.length > 0)
    .sort((a: any, b: any) => {
      const aSort = a.mcVer === 'SNAPSHOT' ? '￿' : a.mcVer;
      const bSort = b.mcVer === 'SNAPSHOT' ? '￿' : b.mcVer;
      const mcCmp = bSort.localeCompare(aSort, undefined, { numeric: true });
      if (mcCmp !== 0) return mcCmp;
      return a.loader.localeCompare(b.loader);
    });
});

function toggleGroup(index: number) {
  if (expandedGroups.value.has(index)) {
    expandedGroups.value.delete(index);
    expandedGroups.value = new Set(expandedGroups.value);
  } else {
    expandedGroups.value.add(index);
    expandedGroups.value = new Set(expandedGroups.value);
  }
}

function selectLoader(val: string) {
  selectedLoaderFilter.value = val;
  showLoaderDropdown.value = false;
  expandedGroups.value = new Set();
  scrollToTop();
}

function selectVersion(val: string) {
  selectedVersionFilter.value = val;
  showVersionDropdown.value = false;
  // Always expand ALL groups when a filter is selected
  expandedGroups.value = new Set(Array.from({ length: versionGroups.value.length }, (_, i) => i));
  scrollToTop();
}

function scrollToTop() {
  const el = document.querySelector('.version-browser-scroll');
  if (el) el.scrollTop = 0;
}

// --- CurseForge data normalization ---
const CURSEFORGE_RELEASE_MAP: Record<number, string> = { 1: 'release', 2: 'beta', 3: 'alpha' };
const CURSEFORGE_LOADER_MAP: Record<number, string> = { 1: 'forge', 4: 'fabric', 5: 'quilt', 6: 'neoforge' };

function normalizeCurseforgeFile(file: any, modId: number): any {
  return {
    _source: 'curseforge',
    id: String(file.id),
    name: file.fileName,
    version_number: file.fileName.replace(/\.jar$/i, ''),
    version_type: CURSEFORGE_RELEASE_MAP[file.releaseType] || 'release',
    game_versions: file.gameVersions || [],
    loaders: (file.modLoaders || []).map((l: number) => CURSEFORGE_LOADER_MAP[l] || String(l)),
    files: [{
      url: null,  // Resolved at deploy time
      filename: file.fileName,
      size: file.fileSize || 0,
      primary: true,
    }],
    date_published: file.fileDate || '',
    _curseforgeModId: modId,
    _curseforgeFileId: file.id,
  };
}

async function selectMod(mod: any) {
  selectedMod.value = mod;
  modVersions.value = [];
  isLoadingVersions.value = true;
  deployingVersion.value = '';
  deploySuccessful.value = false;
  selectedLoaderFilter.value = '';
  selectedVersionFilter.value = '';
  expandedGroups.value = new Set();
  deployError.value = '';
  showDeployToast.value = false;

  try {
    if (mod._source === 'curseforge') {
      await fetchCurseforgeVersions(mod);
    } else {
      await fetchModrinthVersions(mod);
    }
  } catch (error) {
    console.error('Failed to fetch versions', error);
  } finally {
    isLoadingVersions.value = false;
  }
}

async function fetchModrinthVersions(mod: any) {
  try {
    const data = await apiModrinthProxy(`/v2/project/${mod.project_id}/version`);
    const versions = data.data || data;
    modVersions.value = (Array.isArray(versions) ? versions : []).sort((a: any, b: any) => {
      const order: Record<string, number> = { release: 0, beta: 1, alpha: 2 };
      return (order[a.version_type] ?? 3) - (order[b.version_type] ?? 3) ||
             new Date(b.date_published).getTime() - new Date(a.date_published).getTime();
    });
  } catch (e) { console.error('Fetch Modrinth versions failed:', e); }
}

async function fetchCurseforgeVersions(mod: any) {
  const modId = parseInt(mod.project_id);
  try {
    const data = await apiCurseProxy(`/v1/mods/${modId}/files?pageSize=50`);
    const files = data.data?.data || data.data || [];
    modVersions.value = files
      .map((f: any) => normalizeCurseforgeFile(f, modId))
      .sort((a: any, b: any) => {
        const order: Record<string, number> = { release: 0, beta: 1, alpha: 2 };
        return (order[a.version_type] ?? 3) - (order[b.version_type] ?? 3);
      });
  } catch (e) { console.error('Fetch CurseForge versions failed:', e); }
}

async function deployVersion(ver: any) {
  if (deployingVersion.value) return;

  deployingVersion.value = ver.id;
  deployError.value = '';
  showDeployToast.value = true;
  deployStatusMessage.value = `正在部署 ${ver.name}...`;

  try {
    let downloadUrl: string;
    let fileName: string;

      if (ver._source === 'curseforge') {
      const idStr = String(ver._curseforgeFileId);
      const first4 = idStr.substring(0, 4);
      const rest = idStr.substring(4);
      fileName = ver.files[0]?.filename || `file_${ver._curseforgeFileId}.jar`;
      downloadUrl = `https://edge.forgecdn.net/files/${first4}/${rest}/${fileName}`;
      deployStatusMessage.value = `正在从 CurseForge 拉取 ${fileName}...`;
    } else {
      // Modrinth: find the primary file
      const targetFile = ver.files.find((f: any) => f.primary) || ver.files[0];
      if (!targetFile) throw new Error('未找到可下载的文件');
      downloadUrl = targetFile.url;
      fileName = targetFile.filename;
    }

    // 1. Download to server mods/resourcepacks/shaderpacks folder
    const type = projectType.value === 'resourcepack' ? 'resourcepacks' : projectType.value === 'shader' ? 'shaderpacks' : 'mods';

    deployStatusMessage.value = `正在从 ${ver._source === 'curseforge' ? 'CurseForge' : 'Modrinth'} 拉取 ${fileName}...`;

    await apiMarketDownload(props.instanceId, downloadUrl, fileName, type);

    // 2. Auto commit to create snapshot
    deployStatusMessage.value = '正在提交快照...';
    try {
      await apiCommit(props.instanceId);
      deployStatusMessage.value = `✓ ${fileName} 部署成功`;
    } catch (e) {
      deployStatusMessage.value = `${fileName} 已部署，但快照提交失败`;
    }

    deploySuccessful.value = true;

    setTimeout(() => {
      if (!deployError.value) {
        showDeployToast.value = false;
        selectedMod.value = null;
      }
    }, 4000);

  } catch (error: any) {
    console.error('Deploy failed:', error);
    deployError.value = error.message || '部署失败，请检查控制台日志';
    deployStatusMessage.value = '部署失败';
    deployingVersion.value = '';
  }
}

// --- Utilities ---
function formatNumber(num: number) {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
  return num.toString();
}

function formatDate(dateStr: string) {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - d.getTime();
  if (diff < 86400000) return '今天';
  if (diff < 172800000) return '昨天';
  return d.toLocaleDateString();
}

function formatDateOnly(dateStr: string) {
  if (!dateStr) return '';
  return new Date(dateStr).toLocaleDateString();
}

// PCL-CE-style cross-source dedup
// Checks: different source → slug/title/description alphanumeric match
function isLike(a: any, b: any): boolean {
  if (a._source === b._source) return false;
  const getRaw = (s: string) => (s || '').replace(/[^a-zA-Z0-9]/g, '').toLowerCase();
  // Compare slug (most reliable), then title, then description
  const slugA = getRaw(a.slug);
  const slugB = getRaw(b.slug);
  if (slugA && slugB && slugA === slugB) return true;
  const titleA = getRaw(a.title);
  const titleB = getRaw(b.title);
  if (titleA && titleB && titleA === titleB) return true;
  const descA = getRaw(a.description);
  const descB = getRaw(b.description);
  if (descA && descB && descA === descB) return true;
  return false;
}

function dedupResults(results: any[]): any[] {
  const unique: any[] = [];
  for (const item of results) {
    if (!unique.some(existing => isLike(existing, item))) {
      unique.push(item);
    }
  }
  return unique;
}

function formatFileSize(bytes: number) {
  if (!bytes) return '';
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + ['B', 'KB', 'MB', 'GB'][i];
}

// --- Init ---
function handleDocClick(e: MouseEvent) {
  if (loaderDropdownRef.value && !loaderDropdownRef.value.contains(e.target as Node)) showLoaderDropdown.value = false;
  if (versionDropdownRef.value && !versionDropdownRef.value.contains(e.target as Node)) showVersionDropdown.value = false;
}

onMounted(() => {
  document.addEventListener('click', handleDocClick);
  // Apply initialType from ExplorerView context
  if (props.initialType && ['mod', 'resourcepack', 'shader'].includes(props.initialType)) {
    projectType.value = props.initialType;
  }
  // Load recommendations on mount
  fetchRecommendations();
});

onUnmounted(() => {
  document.removeEventListener('click', handleDocClick);
});
</script>
