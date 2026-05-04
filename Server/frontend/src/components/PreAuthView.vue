<template>
  <div class="flex-1 flex flex-col overflow-hidden relative">
    <!-- Ambient Background -->
    <div class="absolute bottom-0 right-0 -mr-32 -mb-32 w-96 h-96 bg-cyan-500/10 rounded-full blur-3xl pointer-events-none"></div>

    <header class="h-24 border-b border-slate-800 bg-slate-900/60 backdrop-blur-2xl flex items-center px-10 z-10">
      <div>
        <h2 class="text-2xl font-black bg-gradient-to-r from-cyan-400 to-blue-500 bg-clip-text text-transparent tracking-tight">预授权密钥管理</h2>
        <p class="text-sm text-slate-500 mt-1 font-medium">创建预授权密钥，管理设备绑定与访问权限</p>
      </div>
    </header>

    <div class="flex-1 overflow-auto p-10 z-10">
      <div class="max-w-5xl space-y-8">

        <!-- Key Actions -->
        <div class="flex items-center justify-between">
          <p class="text-sm text-slate-500">密钥格式 <code class="text-cyan-400 bg-cyan-500/10 px-1.5 py-0.5 rounded text-xs font-mono">CLOUD-XXXX-XXXX-XXXX</code>，玩家输入后完成设备绑定即可访问服务器资源。</p>
          <button @click="showAddKeyModal = true" class="bg-cyan-500/10 hover:bg-cyan-500/20 text-cyan-400 px-4 py-2 rounded-xl text-sm font-bold transition-colors border border-cyan-500/20 flex items-center shrink-0 ml-4">
            <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path></svg>
            生成预授权密钥
          </button>
        </div>

        <!-- Key List -->
        <div class="bg-slate-800/30 border border-slate-700/50 rounded-2xl overflow-hidden">
          <div class="grid grid-cols-8 gap-4 px-5 py-3 bg-slate-900/40 border-b border-slate-700/50 text-[10px] font-bold text-slate-500 uppercase tracking-widest">
            <div class="col-span-2">密钥码 / 玩家名</div>
            <div>设备数</div>
            <div>实例</div>
            <div>状态</div>
            <div>创建时间</div>
            <div>操作</div>
          </div>
          <div v-if="loadingKeys" class="px-5 py-10 text-center text-slate-500 text-xs">加载中...</div>
          <div v-if="!loadingKeys && keys.length === 0" class="px-5 py-16 text-center">
            <svg class="w-10 h-10 text-slate-600 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"></path></svg>
            <p class="text-slate-500 text-sm">尚未创建任何预授权密钥</p>
            <p class="text-slate-600 text-xs mt-1">点击上方按钮生成第一个密钥</p>
          </div>
          <div v-for="key in keys" :key="key.id" class="grid grid-cols-8 gap-4 px-5 py-3 border-b border-slate-700/30 hover:bg-slate-700/20 transition-colors items-center text-sm">
            <div class="col-span-2">
              <p class="font-mono text-slate-200 text-xs">{{ key.key_code }}</p>
              <p class="text-xs text-slate-500">{{ key.player_name || '未命名' }}</p>
            </div>
            <div class="text-slate-400 font-mono text-xs">{{ key.devices_bound }}/{{ key.max_devices }}</div>
            <div class="text-slate-400 text-xs">{{ instanceSummary(key) }}</div>
            <div>
              <span class="px-2 py-0.5 text-[10px] font-bold rounded" :class="key.is_active ? 'bg-emerald-500/10 text-emerald-400' : 'bg-rose-500/10 text-rose-400'">{{ key.is_active ? 'Active' : 'Revoked' }}</span>
            </div>
            <div class="text-slate-500 text-xs">{{ new Date(key.created_at).toLocaleDateString() }}</div>
            <div class="flex space-x-1">
              <button v-if="key.devices_bound > 0" @click="viewBindings(key)" class="p-1.5 text-slate-500 hover:text-cyan-400 hover:bg-cyan-500/10 rounded-lg transition-colors" title="查看设备绑定">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
              </button>
              <button v-else class="p-1.5 text-slate-700 cursor-not-allowed rounded-lg" title="暂无设备绑定">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
              </button>
              <button @click="editKey(key)" class="p-1.5 text-slate-500 hover:text-blue-400 hover:bg-blue-500/10 rounded-lg transition-colors" title="编辑">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path></svg>
              </button>
              <button @click="deleteKey(key)" class="p-1.5 text-slate-500 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-colors" title="删除">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
              </button>
            </div>
          </div>
        </div>

        <!-- Add/Edit Key Modal -->
        <div v-if="showAddKeyModal || showEditKeyModal" class="fixed inset-0 z-[100] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-slate-950/80 backdrop-blur-xl" @click="closeKeyModal"></div>
          <div class="bg-slate-900 border border-white/10 w-full max-w-md rounded-3xl shadow-2xl z-10 overflow-hidden">
            <div class="p-6 space-y-4">
              <h3 class="text-lg font-black text-white">{{ showEditKeyModal ? '编辑预授权密钥' : '生成预授权密钥' }}</h3>
              <div class="space-y-3">
                <div>
                  <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">玩家名称 (备注)</label>
                  <input v-model="keyForm.player_name" type="text" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 outline-none focus:border-cyan-500/50 transition-colors mt-1">
                </div>
                <div>
                  <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">最大设备数</label>
                  <input v-model.number="keyForm.max_devices" type="number" min="1" max="10" class="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-3 py-2 text-slate-200 outline-none focus:border-cyan-500/50 transition-colors mt-1">
                </div>
                <div>
                  <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">可访问实例 <span class="text-slate-600 font-normal">(不选 = 全部实例)</span></label>
                  <div class="mt-1 bg-slate-800/50 border border-slate-700 rounded-xl p-2 max-h-40 overflow-y-auto space-y-1">
                    <label v-for="inst in instances" :key="inst.id" class="flex items-center gap-2 px-2 py-1 rounded-lg hover:bg-slate-700/30 cursor-pointer text-xs text-slate-300">
                      <input type="checkbox" :value="inst.id" v-model="keyForm.instance_ids" class="rounded border-slate-600 bg-slate-700 text-cyan-500 focus:ring-cyan-500/50">
                      <span class="font-mono text-slate-400">{{ inst.id }}</span>
                      <span v-if="inst.display_name" class="text-slate-500">— {{ inst.display_name }}</span>
                    </label>
                    <div v-if="instances.length === 0" class="text-xs text-slate-600 px-2 py-1">暂无可用实例</div>
                  </div>
                </div>
                <div v-if="showEditKeyModal">
                  <label class="flex items-center space-x-2 mt-2">
                    <input v-model="keyForm.is_active" type="checkbox" class="rounded">
                    <span class="text-xs font-bold text-slate-500 uppercase tracking-widest">密钥激活</span>
                  </label>
                </div>
              </div>
              <div v-if="keyFormError" class="text-sm text-rose-400 bg-rose-500/10 rounded-lg px-3 py-2">{{ keyFormError }}</div>
              <div class="flex gap-3 pt-2">
                <button @click="closeKeyModal" class="flex-1 px-4 py-2.5 rounded-xl font-bold text-slate-400 hover:bg-slate-800 transition-colors border border-slate-800">取消</button>
                <button @click="submitKeyForm" :disabled="submittingKey" class="flex-[2] bg-cyan-600 hover:bg-cyan-500 text-white py-2.5 rounded-xl font-bold transition-all disabled:opacity-50">{{ submittingKey ? '保存中...' : (showEditKeyModal ? '保存修改' : '生成密钥') }}</button>
              </div>
            </div>
          </div>
        </div>

        <!-- View Bindings Modal -->
        <div v-if="showBindingsModal" class="fixed inset-0 z-[100] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-slate-950/80 backdrop-blur-xl" @click="showBindingsModal = false"></div>
          <div class="bg-slate-900 border border-white/10 w-full max-w-2xl rounded-3xl shadow-2xl z-10 overflow-hidden">
            <div class="p-6 space-y-4">
              <div class="flex items-center justify-between">
                <h3 class="text-lg font-black text-white">设备绑定列表</h3>
                <button @click="showBindingsModal = false" class="p-1.5 text-slate-500 hover:text-slate-300 rounded-lg">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18 18 6M6 6l12 12"></path></svg>
                </button>
              </div>
              <p class="text-xs text-slate-500 font-mono">{{ viewingKey?.key_code }} — {{ viewingKey?.player_name }}</p>
              <div class="bg-slate-800/30 border border-slate-700/50 rounded-2xl overflow-hidden">
                <div class="grid grid-cols-4 gap-4 px-5 py-3 bg-slate-900/40 border-b border-slate-700/50 text-[10px] font-bold text-slate-500 uppercase tracking-widest">
                  <div>设备标签</div>
                  <div>首次绑定</div>
                  <div>最近活跃</div>
                  <div>操作</div>
                </div>
                <div v-if="loadingBindings" class="px-5 py-10 text-center text-slate-500 text-xs">加载中...</div>
                <div v-if="!loadingBindings && currentBindings.length === 0" class="px-5 py-10 text-center text-slate-500 text-xs">暂无绑定设备</div>
                <div v-for="b in currentBindings" :key="b.id" class="grid grid-cols-4 gap-4 px-5 py-3 border-b border-slate-700/30 hover:bg-slate-700/20 transition-colors items-center text-sm">
                  <div>
                    <p class="text-slate-200 text-xs">{{ b.device_label }}</p>
                    <p class="text-[10px] text-slate-500 font-mono truncate" :title="b.binding_token">{{ b.binding_token.slice(0, 12) }}...</p>
                  </div>
                  <div class="text-slate-500 text-xs">{{ new Date(b.first_bound_at).toLocaleDateString() }}</div>
                  <div class="text-slate-500 text-xs">{{ new Date(b.last_seen_at).toLocaleDateString() }}</div>
                  <div>
                    <button v-if="b.is_active" @click="revokeBinding(b)" class="px-2 py-1 text-[10px] font-bold text-rose-400 hover:bg-rose-500/10 rounded-lg transition-colors">解绑</button>
                    <span v-else class="text-[10px] font-bold text-slate-600">已解绑</span>
                  </div>
                </div>
              </div>
              <button @click="showBindingsModal = false" class="w-full px-4 py-2.5 rounded-xl font-bold text-slate-400 hover:bg-slate-800 transition-colors border border-slate-800 mt-2">关闭</button>
            </div>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { apiListPreAuthKeys, apiCreatePreAuthKey, apiUpdatePreAuthKey, apiDeletePreAuthKey, apiListKeyBindings, apiRevokeBinding, apiListInstances } from '../api';

const keys = ref<any[]>([]);
const loadingKeys = ref(false);
const showAddKeyModal = ref(false);
const showEditKeyModal = ref(false);
const editingKeyId = ref<number | null>(null);
const submittingKey = ref(false);
const keyFormError = ref('');
const instances = ref<any[]>([]);
const keyForm = ref({
  player_name: '',
  max_devices: 3,
  is_active: true,
  instance_ids: [] as string[],
});

const showBindingsModal = ref(false);
const viewingKey = ref<any>(null);
const currentBindings = ref<any[]>([]);
const loadingBindings = ref(false);

async function fetchKeys() {
  loadingKeys.value = true;
  try {
    const data = await apiListPreAuthKeys();
    const list = data.data ?? data;
    keys.value = Array.isArray(list) ? list : [];
  } catch (e) {
    console.error('Failed to fetch pre-auth keys:', e);
  } finally {
    loadingKeys.value = false;
  }
}

function closeKeyModal() {
  showAddKeyModal.value = false;
  showEditKeyModal.value = false;
  editingKeyId.value = null;
  keyFormError.value = '';
  keyForm.value = { player_name: '', max_devices: 3, is_active: true, instance_ids: [] };
}

function editKey(key: any) {
  editingKeyId.value = key.id;
  let ids: string[] = [];
  if (key.instance_ids) {
    try { ids = typeof key.instance_ids === 'string' ? JSON.parse(key.instance_ids) : key.instance_ids; } catch { ids = []; }
    if (!Array.isArray(ids)) ids = [];
  }
  keyForm.value = {
    player_name: key.player_name,
    max_devices: key.max_devices,
    is_active: key.is_active,
    instance_ids: ids,
  };
  showEditKeyModal.value = true;
  keyFormError.value = '';
}

async function submitKeyForm() {
  keyFormError.value = '';
  submittingKey.value = true;
  try {
    if (showEditKeyModal.value && editingKeyId.value) {
      await apiUpdatePreAuthKey(editingKeyId.value, keyForm.value.player_name, keyForm.value.max_devices, keyForm.value.is_active, keyForm.value.instance_ids);
    } else {
      const data = await apiCreatePreAuthKey(keyForm.value.player_name, keyForm.value.max_devices, keyForm.value.instance_ids);
      const newKey = data.data || data;
      if (newKey.key_code) {
        try { 
          await navigator.clipboard.writeText(newKey.key_code); 
          alert(`密钥已生成并复制到剪贴板:\n${newKey.key_code}`);
        } catch {
          prompt('密钥已生成，剪贴板不可用，请手动复制保存：', newKey.key_code);
        }
      }
    }
    closeKeyModal();
    fetchKeys();
  } catch (e: any) {
    keyFormError.value = e.message || '操作失败';
  } finally {
    submittingKey.value = false;
  }
}

async function deleteKey(key: any) {
  if (!confirm(`确定要删除密钥 [${key.key_code}] 吗？其下所有设备绑定将失效。`)) return;
  try {
    await apiDeletePreAuthKey(key.id);
    fetchKeys();
  } catch (e: any) {
    alert(e.message || '删除失败');
  }
}

async function viewBindings(key: any) {
  viewingKey.value = key;
  showBindingsModal.value = true;
  loadingBindings.value = true;
  try {
    const data = await apiListKeyBindings(key.id);
    const list = data.data ?? data;
    currentBindings.value = Array.isArray(list) ? list : [];
  } catch (e) {
    console.error('Failed to fetch bindings:', e);
  } finally {
    loadingBindings.value = false;
  }
}

async function revokeBinding(binding: any) {
  if (!confirm(`确定要解绑设备 [${binding.device_label}] 吗？`)) return;
  try {
    await apiRevokeBinding(binding.id);
    if (viewingKey.value) viewBindings(viewingKey.value);
    fetchKeys();
  } catch (e: any) {
    alert(e.message || '解绑失败');
  }
}

function instanceSummary(key: any): string {
  if (!key.instance_ids) return '全部'
  let ids: string[] = []
  try { ids = typeof key.instance_ids === 'string' ? JSON.parse(key.instance_ids) : key.instance_ids; } catch { return '全部' }
  if (!Array.isArray(ids) || ids.length === 0) return '全部'
  return ids.length + ' 个'
}

async function fetchInstances() {
  try {
    const data = await apiListInstances();
    const list = data?.data?.instance_list ?? data?.instance_list;
    instances.value = Array.isArray(list) ? list : [];
  } catch (e) {
    console.error('Failed to fetch instances:', e);
  }
}

onMounted(() => {
  fetchKeys();
  fetchInstances();
});
</script>
