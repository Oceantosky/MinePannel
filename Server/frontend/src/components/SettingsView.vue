<template>
  <div class="flex-1 flex flex-col overflow-hidden relative">
    <!-- Ambient Background -->
    <div class="absolute bottom-0 right-0 -mr-32 -mb-32 w-96 h-96 bg-purple-500/10 rounded-full blur-3xl pointer-events-none"></div>

    <header class="h-24 border-b border-slate-800 bg-slate-900/60 backdrop-blur-2xl flex items-center justify-between px-10 z-10">
      <div>
        <h2 class="text-2xl font-black bg-gradient-to-r from-slate-100 to-slate-400 bg-clip-text text-transparent tracking-tight">系统设置</h2>
        <p class="text-sm text-slate-500 mt-1 font-medium">配置集群核心参数与安全策略</p>
      </div>
      <button @click="saveSettings" :disabled="saving" class="bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white px-8 py-2.5 rounded-xl font-bold transition-all shadow-lg shadow-indigo-500/25 flex items-center disabled:opacity-50">
        <svg v-if="saving" class="w-5 h-5 mr-2 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
        <svg v-else class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"></path></svg>
        保存全局配置
      </button>
    </header>

    <div class="flex-1 overflow-auto p-10 z-10">
      <div class="max-w-4xl space-y-10">
        
        <!-- General Section -->
        <section class="space-y-6">
          <div class="flex items-center space-x-2 border-l-4 border-indigo-500 pl-4">
            <h3 class="text-lg font-bold text-slate-200">常规标识</h3>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div class="space-y-2">
              <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">节点显示名称</label>
              <input v-model="config.node_name" type="text" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-4 py-3 text-slate-200 outline-none focus:border-indigo-500/50 transition-colors">
            </div>
            <div class="space-y-2">
              <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">公网访问 IP / 域名</label>
              <input v-model="config.public_ip" type="text" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-4 py-3 text-slate-200 outline-none focus:border-indigo-500/50 transition-colors">
            </div>
          </div>
        </section>

        <!-- Security Section -->
        <section class="space-y-6">
          <div class="flex items-center space-x-2 border-l-4 border-rose-500 pl-4">
            <h3 class="text-lg font-bold text-slate-200">安全与鉴权</h3>
          </div>
          <div class="space-y-2">
            <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">集群预共享密钥 (PSK Token)</label>
            <div class="relative">
              <input :type="showKey ? 'text' : 'password'" v-model="config.secret_key" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-4 py-3 text-slate-200 outline-none focus:border-rose-500/50 transition-colors font-mono">
              <button @click="showKey = !showKey" class="absolute right-4 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300">
                <svg v-if="!showKey" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"></path></svg>
                <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.542-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18"></path></svg>
              </button>
            </div>
            <p class="text-[10px] text-slate-500 mt-2">注意：修改此密钥将导致所有正在运行的 Slave 节点和 PCL-CE 客户端失去连接，直到它们更新为新密钥。</p>
          </div>
        </section>

        <!-- Pre-Auth Section -->
        <section class="space-y-6">
          <div class="flex items-center space-x-2 border-l-4 border-cyan-500 pl-4">
            <h3 class="text-lg font-bold text-slate-200">预授权与设备绑定</h3>
          </div>
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">启用预授权模式</label>
                <p class="text-[10px] text-slate-500 mt-1">开启后，客户端必须持有有效的预授权密钥并完成设备绑定，才能访问 API 和下载文件。</p>
              </div>
              <button @click="config.pre_auth_enabled = !config.pre_auth_enabled" class="relative w-14 h-7 rounded-full transition-colors duration-200" :class="config.pre_auth_enabled ? 'bg-cyan-600' : 'bg-slate-700'">
                <div class="absolute top-0.5 w-6 h-6 rounded-full bg-white shadow transition-all duration-200" :class="config.pre_auth_enabled ? 'left-7' : 'left-0.5'"></div>
              </button>
            </div>

            <div v-if="config.pre_auth_enabled" class="space-y-2 p-4 bg-cyan-500/5 border border-cyan-500/20 rounded-2xl">
              <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">预授权 Pepper (PreAuth Secret)</label>
              <div class="relative">
                <input :type="showPreAuthKey ? 'text' : 'password'" v-model="config.pre_auth_secret" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-4 py-3 text-slate-200 outline-none focus:border-cyan-500/50 transition-colors font-mono text-sm">
                <button @click="showPreAuthKey = !showPreAuthKey" class="absolute right-4 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300">
                  <svg v-if="!showPreAuthKey" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"></path></svg>
                  <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.542-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18"></path></svg>
                </button>
              </div>
              <p class="text-[10px] text-slate-500">用于对设备指纹加盐哈希。修改此值将导致所有现有设备绑定失效。</p>
            </div>

            <p class="text-[10px] text-slate-500">
              <span class="text-cyan-400 font-bold">提示：</span>开启预授权后，通过左侧栏"预授权管理"创建密钥并分发给玩家。
            </p>
          </div>
        </section>

        <!-- Distribution Section -->
        <section class="space-y-6">
          <div class="flex items-center space-x-2 border-l-4 border-emerald-500 pl-4">
            <h3 class="text-lg font-bold text-slate-200">分发引擎逻辑</h3>
          </div>
          <div class="space-y-4">
            <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">主分发策略</label>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div @click="config.dist_policy = 'direct'" :class="config.dist_policy === 'direct' ? 'border-emerald-500/50 bg-emerald-500/5' : 'border-slate-700 bg-slate-800/30'" class="border-2 rounded-2xl p-5 cursor-pointer transition-all hover:bg-slate-800/50">
                <div class="flex items-center justify-between mb-2">
                  <span class="font-bold text-slate-200">DIRECT (直接分发)</span>
                  <div v-if="config.dist_policy === 'direct'" class="w-4 h-4 rounded-full bg-emerald-500"></div>
                </div>
                <p class="text-xs text-slate-500 leading-relaxed">客户端直接从 Master 或 Slave 节点的 55001 数据面端口拉取全量包与增量文件。适用于私有化部署或带宽充足的场景。</p>
              </div>
              <div class="border-2 border-slate-800 bg-slate-900/50 rounded-2xl p-5 cursor-not-allowed opacity-50 relative overflow-hidden">
                <div class="absolute -right-6 top-4 bg-rose-500 text-[10px] font-black px-8 py-1 rotate-45 text-white">WIP</div>
                <div class="flex items-center justify-between mb-2">
                  <span class="font-bold text-slate-500">CDN (策略重定向)</span>
                </div>
                <p class="text-xs text-slate-600 leading-relaxed">全量包下载请求将重定向至实例配置的第三方直链（如 123 盘、R2）。清单与增量哈希仍由自建节点分发。极大地降低源站带宽成本。即将开放。</p>
              </div>
            </div>
          </div>
        </section>

        <!-- User Management Section (Admin Only) -->
        <section v-if="currentUser?.role === 'admin'" class="space-y-6">
          <div class="flex items-center space-x-2 border-l-4 border-amber-500 pl-4">
            <h3 class="text-lg font-bold text-slate-200">用户管理</h3>
            <span class="text-[10px] text-slate-500 ml-2">Admin Only</span>
          </div>

          <div class="flex items-center justify-between">
            <p class="text-sm text-slate-500">管理系统用户账户与实例权限分配</p>
            <button @click="showAddUserModal = true; fetchAllInstances()" class="bg-amber-500/10 hover:bg-amber-500/20 text-amber-400 px-4 py-2 rounded-xl text-sm font-bold transition-colors border border-amber-500/20 flex items-center">
              <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path></svg>
              添加用户
            </button>
          </div>

          <!-- User List -->
          <div class="bg-slate-800/30 border border-slate-700/50 rounded-2xl overflow-hidden">
            <div class="grid grid-cols-7 gap-4 px-5 py-3 bg-slate-900/40 border-b border-slate-700/50 text-[10px] font-bold text-slate-500 uppercase tracking-widest">
              <div class="col-span-2">用户名</div>
              <div>角色</div>
              <div>实例数</div>
              <div>状态</div>
              <div>最后登录</div>
              <div>操作</div>
            </div>
            <div v-if="loadingUsers" class="px-5 py-10 text-center text-slate-500 text-xs">加载中...</div>
            <div v-for="u in users" :key="u.id" class="grid grid-cols-7 gap-4 px-5 py-3 border-b border-slate-700/30 hover:bg-slate-700/20 transition-colors items-center text-sm">
              <div class="col-span-2 font-medium text-slate-200">{{ u.username }}</div>
              <div>
                <span class="px-2 py-0.5 text-[10px] font-bold rounded uppercase" :class="u.role === 'admin' ? 'bg-amber-500/10 text-amber-400' : 'bg-slate-500/10 text-slate-400'">{{ u.role }}</span>
              </div>
              <div class="text-slate-400 font-mono">{{ u.instances?.length || 0 }}</div>
              <div>
                <span class="px-2 py-0.5 text-[10px] font-bold rounded" :class="u.is_active ? 'bg-emerald-500/10 text-emerald-400' : 'bg-rose-500/10 text-rose-400'">{{ u.is_active ? 'Active' : 'Disabled' }}</span>
              </div>
              <div class="text-slate-500 text-xs">{{ u.last_login_at ? new Date(u.last_login_at).toLocaleDateString() : '从未' }}</div>
              <div class="flex space-x-1">
                <button @click="editUser(u)" class="p-1.5 text-slate-500 hover:text-blue-400 hover:bg-blue-500/10 rounded-lg transition-colors" title="编辑">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path></svg>
                </button>
                <button @click="openResetPasswordModal(u)" class="p-1.5 text-slate-500 hover:text-purple-400 hover:bg-purple-500/10 rounded-lg transition-colors" title="重置密码">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"></path></svg>
                </button>
                <button v-if="u.id !== currentUser?.id" @click="deleteUser(u)" class="p-1.5 text-slate-500 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-colors" title="删除">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                </button>
              </div>
            </div>
          </div>

          <!-- Password Reset Modal -->
          <div v-if="showResetPasswordModal" class="fixed inset-0 z-[100] flex items-center justify-center p-4">
            <div class="absolute inset-0 bg-slate-950/80 backdrop-blur-xl" @click="closeResetPasswordModal"></div>
            <div class="bg-slate-900 border border-white/10 w-full max-w-sm rounded-3xl shadow-2xl z-10 overflow-hidden">
              <div class="p-6 space-y-4">
                <h3 class="text-lg font-black text-white">重置密码: {{ resetPasswordUser?.username }}</h3>
                <div class="space-y-3">
                  <div>
                    <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">新密码</label>
                    <input v-model="resetPasswordInput" type="password" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 outline-none focus:border-amber-500/50 transition-colors mt-1">
                  </div>
                </div>
                <div class="flex gap-3 pt-2">
                  <button @click="closeResetPasswordModal" class="flex-1 px-4 py-2.5 rounded-xl font-bold text-slate-400 hover:bg-slate-800 transition-colors border border-slate-800">取消</button>
                  <button @click="submitResetPassword" class="flex-1 bg-purple-600 hover:bg-purple-500 text-white py-2.5 rounded-xl font-bold transition-all">重置密码</button>
                </div>
              </div>
            </div>
          </div>

          <!-- Add/Edit User Modal -->
          <div v-if="showAddUserModal || showEditUserModal" class="fixed inset-0 z-[100] flex items-center justify-center p-4">
            <div class="absolute inset-0 bg-slate-950/80 backdrop-blur-xl" @click="closeUserModal"></div>
            <div class="bg-slate-900 border border-white/10 w-full max-w-lg rounded-3xl shadow-2xl z-10 overflow-hidden">
              <div class="p-6 space-y-4">
                <h3 class="text-lg font-black text-white">{{ showEditUserModal ? '编辑用户' : '添加用户' }}</h3>
                <div class="space-y-3">
                  <div>
                    <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">用户名</label>
                    <input v-model="userForm.username" type="text" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 outline-none focus:border-amber-500/50 transition-colors mt-1">
                  </div>
                  <div>
                    <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">{{ showEditUserModal ? '新密码 (留空不修改)' : '密码' }}</label>
                    <input v-model="userForm.password" type="password" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 outline-none focus:border-amber-500/50 transition-colors mt-1">
                  </div>
                  <div>
                    <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">角色</label>
                    <select v-model="userForm.role" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 outline-none focus:border-amber-500/50 transition-colors mt-1 appearance-none cursor-pointer" style="background-image: url('data:image/svg+xml,%3Csvg xmlns=%27http://www.w3.org/2000/svg%27 fill=%27none%27 stroke=%27%2394a3b8%27 stroke-width=%272%27 viewBox=%270 0 24 24%27%3E%3Cpath stroke-linecap=%27round%27 stroke-linejoin=%27round%27 d=%27m6 9 6 6 6-6%27/%3E%3C/svg%3E'); background-repeat: no-repeat; background-position: right 0.75rem center; background-size: 1.25rem; padding-right: 2.5rem;">
                      <option value="user" class="bg-slate-900 text-slate-200">User — 仅管理分配的实例</option>
                      <option value="admin" class="bg-slate-900 text-slate-200">Admin — 管理全部实例与用户</option>
                    </select>
                  </div>
                  <div v-if="showEditUserModal">
                    <label class="flex items-center space-x-2 mt-2">
                      <input v-model="userForm.is_active" type="checkbox" class="rounded">
                      <span class="text-xs font-bold text-slate-500 uppercase tracking-widest">账户激活</span>
                    </label>
                  </div>
                  <div>
                    <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">分配实例</label>
                    <!-- Multi-select instance dropdown -->
                    <div class="relative mt-1" ref="instanceDropdownRef">
                      <button @click="showInstanceDropdown = !showInstanceDropdown" type="button" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 outline-none focus:border-amber-500/50 transition-colors text-left flex items-center justify-between">
                        <span v-if="userForm.selectedInstances.length === 0" class="text-slate-500 text-sm">请选择实例...</span>
                        <span v-else class="text-sm text-slate-200">{{ userForm.selectedInstances.length }} 个实例已选中</span>
                        <svg class="w-4 h-4 text-slate-500 shrink-0 ml-2 transition-transform" :class="{ 'rotate-180': showInstanceDropdown }" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m6 9 6 6 6-6"></path></svg>
                      </button>
                      <div v-if="showInstanceDropdown" class="absolute z-50 top-full left-0 right-0 mt-1 bg-slate-800 border border-slate-700 rounded-xl shadow-2xl max-h-52 overflow-y-auto p-1">
                        <div v-if="allInstances.length === 0" class="px-3 py-4 text-center text-slate-500 text-xs">暂无可分配的实例</div>
                        <label v-for="inst in allInstances" :key="inst.id" @click.stop class="flex items-center px-3 py-2 rounded-lg hover:bg-slate-700/50 transition-colors cursor-pointer">
                          <input type="checkbox" :value="inst.id" v-model="userForm.selectedInstances" class="w-4 h-4 rounded border-slate-600 bg-slate-700 text-amber-500 focus:ring-amber-500/50 mr-3 shrink-0">
                          <div class="min-w-0">
                            <p class="text-sm text-slate-200 truncate">{{ inst.display_name || inst.id }}</p>
                            <p class="text-[10px] text-slate-500 font-mono truncate">{{ inst.id }}</p>
                          </div>
                        </label>
                      </div>
                    </div>
                    <!-- Selected instances tags -->
                    <div v-if="userForm.selectedInstances.length > 0" class="flex flex-wrap gap-1.5 mt-2">
                      <span v-for="id in userForm.selectedInstances" :key="id" class="inline-flex items-center px-2 py-1 rounded-lg bg-amber-500/10 border border-amber-500/20 text-[10px] text-amber-400 font-mono">
                        {{ id }}
                        <button @click="userForm.selectedInstances = userForm.selectedInstances.filter(i => i !== id)" class="ml-1.5 text-amber-500 hover:text-rose-400">
                          <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18 18 6M6 6l12 12"></path></svg>
                        </button>
                      </span>
                    </div>
                  </div>
                </div>
                <div v-if="userFormError" class="text-sm text-rose-400 bg-rose-500/10 rounded-lg px-3 py-2">{{ userFormError }}</div>
                <div class="flex gap-3 pt-2">
                  <button @click="closeUserModal" class="flex-1 px-4 py-2.5 rounded-xl font-bold text-slate-400 hover:bg-slate-800 transition-colors border border-slate-800">取消</button>
                  <button @click="submitUserForm" :disabled="submittingUser" class="flex-[2] bg-amber-600 hover:bg-amber-500 text-white py-2.5 rounded-xl font-bold transition-all disabled:opacity-50">{{ submittingUser ? '保存中...' : '保存' }}</button>
                </div>
              </div>
            </div>
          </div>
        </section>

      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, nextTick } from 'vue';
import { apiReadConfig, apiWriteConfig, apiListUsers, apiCreateUser, apiUpdateUser, apiDeleteUser, apiResetUserPassword, apiGetInstances, getCurrentUser } from '../api';

const saving = ref(false);
const showKey = ref(false);
const showPreAuthKey = ref(false);
const currentUser = ref<any>(null);

const config = ref({
  node_name: '',
  secret_key: '',
  public_ip: '',
  dist_policy: 'direct',
  pre_auth_enabled: false,
  pre_auth_secret: '',
});

// --- User Management ---
const users = ref<any[]>([]);
const loadingUsers = ref(false);
const showAddUserModal = ref(false);
const showEditUserModal = ref(false);
const editingUserId = ref<number | null>(null);
const submittingUser = ref(false);
const userFormError = ref('');

const userForm = ref({
  username: '',
  password: '',
  role: 'user',
  is_active: true,
  selectedInstances: [] as string[],
});

const allInstances = ref<any[]>([]);
const showInstanceDropdown = ref(false);
const instanceDropdownRef = ref<HTMLElement | null>(null);

// Close dropdown on outside click
watch(showInstanceDropdown, async (open) => {
  if (!open) return;
  await nextTick();
  const handler = (e: MouseEvent) => {
    if (!instanceDropdownRef.value?.contains(e.target as Node)) {
      showInstanceDropdown.value = false;
    }
  };
  document.addEventListener('mousedown', handler, { once: true });
});

async function fetchAllInstances() {
  try {
    const data = await apiGetInstances();
    const list = data.data?.instance_list ?? data.instance_list;
    allInstances.value = Array.isArray(list) ? list : [];
  } catch (e) {
    console.error('Failed to fetch instances:', e);
  }
}

async function fetchUsers() {
  loadingUsers.value = true;
  try {
    const data = await apiListUsers();
    const list = data.data ?? data;
    users.value = Array.isArray(list) ? list : [];
  } catch (e) {
    console.error('Failed to fetch users:', e);
  } finally {
    loadingUsers.value = false;
  }
}

function closeUserModal() {
  showAddUserModal.value = false;
  showEditUserModal.value = false;
  editingUserId.value = null;
  userFormError.value = '';
  showInstanceDropdown.value = false;
  userForm.value = { username: '', password: '', role: 'user', is_active: true, selectedInstances: [] };
}

function editUser(u: any) {
  editingUserId.value = u.id;
  userForm.value = {
    username: u.username,
    password: '',
    role: u.role,
    is_active: u.is_active,
    selectedInstances: [...(u.instances || [])],
  };
  showEditUserModal.value = true;
  userFormError.value = '';
  fetchAllInstances();
}

async function submitUserForm() {
  userFormError.value = '';
  if (!userForm.value.username) { userFormError.value = '请输入用户名'; return; }
  const instances = userForm.value.selectedInstances;

  submittingUser.value = true;
  try {
    if (showEditUserModal.value && editingUserId.value) {
      await apiUpdateUser(editingUserId.value, userForm.value.username, userForm.value.role, userForm.value.is_active, instances);
      if (userForm.value.password) {
        await apiResetUserPassword(editingUserId.value, userForm.value.password);
      }
    } else {
      await apiCreateUser(userForm.value.username, userForm.value.password, userForm.value.role, instances);
    }
    closeUserModal();
    fetchUsers();
  } catch (e: any) {
    userFormError.value = e.message || '操作失败';
  } finally {
    submittingUser.value = false;
  }
}

async function deleteUser(u: any) {
  if (!confirm(`确定要删除用户 [${u.username}] 吗？此操作不可撤销。`)) return;
  try {
    await apiDeleteUser(u.id);
    fetchUsers();
  } catch (e: any) {
    alert(e.message || '删除失败');
  }
}

const showResetPasswordModal = ref(false);
const resetPasswordUser = ref<any>(null);
const resetPasswordInput = ref('');

function openResetPasswordModal(u: any) {
  resetPasswordUser.value = u;
  resetPasswordInput.value = '';
  showResetPasswordModal.value = true;
}

function closeResetPasswordModal() {
  showResetPasswordModal.value = false;
  resetPasswordUser.value = null;
  resetPasswordInput.value = '';
}

async function submitResetPassword() {
  if (!resetPasswordInput.value) return;
  try {
    await apiResetUserPassword(resetPasswordUser.value.id, resetPasswordInput.value);
    alert('密码已更新');
    closeResetPasswordModal();
  } catch (e: any) {
    alert(e.message || '重置失败');
  }
}

async function fetchConfig() {
  try {
    const data = await apiReadConfig();
    const cfg = data.data || data;
    config.value = { ...cfg };
  } catch (e) {
    console.error(e);
  }
}

async function saveSettings() {
  saving.value = true;
  try {
    await apiWriteConfig(config.value);
    alert('系统配置已持久化到服务器。');
  } catch (e) {
    console.error(e);
    alert('保存失败，网络错误。');
  } finally {
    saving.value = false;
  }
}


onMounted(() => {
  currentUser.value = getCurrentUser();
  fetchConfig();
  if (currentUser.value?.role === 'admin') {
    fetchUsers();
  }
});
</script>
