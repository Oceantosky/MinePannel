<template>
  <div class="h-screen w-screen bg-[#0f172a] flex items-center justify-center relative overflow-hidden">
    <!-- Ambient Background -->
    <div class="absolute top-0 left-0 -ml-32 -mt-32 w-96 h-96 bg-blue-500/10 rounded-full blur-3xl pointer-events-none"></div>
    <div class="absolute bottom-0 right-0 -mr-32 -mb-32 w-96 h-96 bg-indigo-500/10 rounded-full blur-3xl pointer-events-none"></div>

    <div class="relative z-10 w-full max-w-md px-6">
      <!-- Logo -->
      <div class="text-center mb-10">
        <div class="w-16 h-16 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-2xl flex items-center justify-center mx-auto mb-4 shadow-[0_0_20px_rgba(59,130,246,0.3)]">
          <svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path></svg>
        </div>
        <h1 class="text-2xl font-black text-white tracking-tight">CloudAbroad</h1>
        <p class="text-sm text-slate-500 mt-1">PCL-CE 云端管理平台</p>
      </div>

      <!-- Login Card -->
      <div class="bg-slate-800/60 backdrop-blur-md border border-slate-700/50 rounded-2xl p-8 shadow-2xl">
        <!-- First-time password setup -->
        <div v-if="needsPasswordSetup" class="space-y-5">
          <div class="text-center">
            <h2 class="text-lg font-bold text-white">设置管理员密码</h2>
            <p class="text-sm text-slate-400 mt-1">首次登录，请为 admin 账户设置安全密码</p>
          </div>
          <div class="space-y-2">
            <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">新密码</label>
            <input v-model="newPassword" type="password" @keyup.enter="setupPassword" class="w-full bg-slate-900/60 border border-slate-700 rounded-xl px-4 py-3 text-slate-200 outline-none focus:border-blue-500/50 transition-colors" placeholder="输入密码...">
          </div>
          <div class="space-y-2">
            <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">确认密码</label>
            <input v-model="confirmPassword" type="password" @keyup.enter="setupPassword" class="w-full bg-slate-900/60 border border-slate-700 rounded-xl px-4 py-3 text-slate-200 outline-none focus:border-blue-500/50 transition-colors" placeholder="再次输入密码...">
          </div>
          <div v-if="errorMsg" class="text-sm text-rose-400 bg-rose-500/10 rounded-lg px-3 py-2 border border-rose-500/20">{{ errorMsg }}</div>
          <button @click="setupPassword" :disabled="settingUp" class="w-full bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white py-3 rounded-xl font-bold transition-all shadow-lg shadow-blue-500/25 disabled:opacity-50">
            {{ settingUp ? '设置中...' : '设置密码并登录' }}
          </button>
        </div>

        <!-- Normal login form -->
        <div v-else class="space-y-5">
          <div class="space-y-2">
            <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">用户名</label>
            <input v-model="username" type="text" @keyup.enter="login" class="w-full bg-slate-900/60 border border-slate-700 rounded-xl px-4 py-3 text-slate-200 outline-none focus:border-blue-500/50 transition-colors" placeholder="输入用户名...">
          </div>
          <div class="space-y-2">
            <label class="text-xs font-bold text-slate-500 uppercase tracking-widest">密码</label>
            <input v-model="password" type="password" @keyup.enter="login" class="w-full bg-slate-900/60 border border-slate-700 rounded-xl px-4 py-3 text-slate-200 outline-none focus:border-blue-500/50 transition-colors" placeholder="输入密码...">
          </div>
          <div v-if="errorMsg" class="text-sm text-rose-400 bg-rose-500/10 rounded-lg px-3 py-2 border border-rose-500/20">{{ errorMsg }}</div>
          <button @click="login" :disabled="loggingIn" class="w-full bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white py-3 rounded-xl font-bold transition-all shadow-lg shadow-blue-500/25 disabled:opacity-50">
            {{ loggingIn ? '登录中...' : '登录' }}
          </button>
        </div>
      </div>

      <p class="text-center text-xs text-slate-600 mt-6">MinePannel v6 — Secure Multi-User Edition</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { apiLogin, setCurrentUser } from '../api';

const emit = defineEmits(['login-success']);

const username = ref('admin');
const password = ref('');
const newPassword = ref('');
const confirmPassword = ref('');
const errorMsg = ref('');
const loggingIn = ref(false);
const settingUp = ref(false);
const needsPasswordSetup = ref(false);

async function login() {
  errorMsg.value = '';
  if (!username.value) { errorMsg.value = '请输入用户名'; return; }
  loggingIn.value = true;
  try {
    const data = await apiLogin(username.value, password.value);
    if (data.needs_password_setup) {
      needsPasswordSetup.value = true;
      return;
    }
    setCurrentUser(data.user);
    emit('login-success', data.user);
  } catch (e: any) {
    errorMsg.value = e.message || '登录失败';
  } finally {
    loggingIn.value = false;
  }
}

async function setupPassword() {
  errorMsg.value = '';
  if (!newPassword.value) { errorMsg.value = '请输入密码'; return; }
  if (newPassword.value !== confirmPassword.value) { errorMsg.value = '两次输入的密码不一致'; return; }
  settingUp.value = true;
  try {
    // Login with the new password to set it
    const data = await apiLogin(username.value, newPassword.value);
    setCurrentUser(data.user);
    emit('login-success', data.user);
  } catch (e: any) {
    errorMsg.value = e.message || '设置失败';
  } finally {
    settingUp.value = false;
  }
}
</script>
