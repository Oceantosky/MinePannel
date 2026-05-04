// CloudSync API Client — JWT stored in httpOnly cookie, sent automatically

// --- User Cache (non-sensitive — used for UI state only) ---

export function setCurrentUser(user: any) {
  localStorage.setItem('current_user', JSON.stringify(user))
}

export function getCurrentUser(): any {
  const raw = localStorage.getItem('current_user')
  return raw ? JSON.parse(raw) : null
}

export function clearCurrentUser() {
  localStorage.removeItem('current_user')
}

// --- HMAC Auth (for PCL-CE client operations) ---
// The secret key should be configured per-deployment, not hardcoded.
// For the Web Panel, JWT is sent via httpOnly cookie automatically.
// For PCL-CE clients, the secret is distributed via the install package's PCL.ini.

async function computeHMAC(secret: string, data: string): Promise<string> {
  const encoder = new TextEncoder()
  const key = await crypto.subtle.importKey(
    'raw',
    encoder.encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  )
  const signature = await crypto.subtle.sign('HMAC', key, encoder.encode(data))
  return Array.from(new Uint8Array(signature))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('')
}

export async function authHeaders(secret: string, method: string, path: string, bodyText: string = ''): Promise<Record<string, string>> {
  const timestamp = Math.floor(Date.now() / 1000)
  const bodyHashBuffer = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(bodyText))
  const bodyHash = Array.from(new Uint8Array(bodyHashBuffer)).map(b => b.toString(16).padStart(2, '0')).join('')
  const data = `${timestamp}:${method}:${path}:${bodyHash}`
  const hmac = await computeHMAC(secret, data)
  return {
    'Authorization': `Bearer ${hmac}`,
    'X-Timestamp': timestamp.toString(),
  }
}

// --- Low-level fetch wrappers (JWT sent via httpOnly cookie) ---

async function authedGet(path: string): Promise<any> {
  const res = await fetch(path, { credentials: 'include' })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }))
    throw new Error(err.message || `HTTP ${res.status}`)
  }
  return res.json()
}

async function authedPost(path: string, body?: any): Promise<any> {
  const res = await fetch(path, {
    method: 'POST',
    credentials: 'include',
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }))
    throw new Error(err.message || `HTTP ${res.status}`)
  }
  return res.json()
}

async function authedPut(path: string, body?: any): Promise<any> {
  const res = await fetch(path, {
    method: 'PUT',
    credentials: 'include',
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }))
    throw new Error(err.message || `HTTP ${res.status}`)
  }
  return res.json()
}

async function authedDelete(path: string): Promise<any> {
  const res = await fetch(path, { method: 'DELETE', credentials: 'include' })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }))
    throw new Error(err.message || `HTTP ${res.status}`)
  }
  return res.json()
}

// --- Auth Endpoints ---

export async function apiLogin(username: string, password: string) {
  const res = await fetch('/api/v1/auth/login', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  })
  const data = await res.json()
  if (!res.ok || !data.success) throw new Error(data.message || 'Login failed')
  return data.data // { token, user, needs_password_setup? }
}

export async function apiLogout() {
  const res = await fetch('/api/v1/auth/logout', {
    method: 'POST',
    credentials: 'include',
  })
  const data = await res.json()
  if (!res.ok || !data.success) throw new Error(data.message || 'Logout failed')
  clearCurrentUser()
  return data
}

export function apiAuthStatus() {
  return authedGet('/api/v1/auth/status')
}

export function apiChangePassword(currentPassword: string, newPassword: string) {
  return authedPost('/api/v1/auth/change-password', {
    current_password: currentPassword,
    new_password: newPassword,
  })
}

// --- Admin: User Management ---

export function apiListUsers() {
  return authedGet('/api/v1/admin/users')
}

export function apiCreateUser(username: string, password: string, role: string, instances: string[]) {
  return authedPost('/api/v1/admin/users', { username, password, role, instances })
}

export function apiUpdateUser(id: number, username: string, role: string, isActive: boolean, instances: string[]) {
  return authedPut(`/api/v1/admin/users/${id}`, { username, role, is_active: isActive, instances })
}

export function apiDeleteUser(id: number) {
  return authedDelete(`/api/v1/admin/users/${id}`)
}

export function apiResetUserPassword(id: number, password: string) {
  return authedPut(`/api/v1/admin/users/${id}/password`, { password })
}

// --- Public (no auth) endpoints ---

export function apiGetInstances() {
  return fetch('/api/v1/sync/instances', { credentials: 'include' }).then(r => r.json())
}

export function apiGetMetadata(instanceId: string) {
  return fetch(`/api/v1/master/instances/metadata?id=${instanceId}`, { credentials: 'include' }).then(r => r.json())
}

export function apiListFiles(instanceId: string, path: string) {
  return fetch(`/api/v1/master/files/list?id=${instanceId}&path=${encodeURIComponent(path)}`, { credentials: 'include' }).then(r => r.json())
}

export function apiReadFile(instanceId: string, path: string) {
  return fetch(`/api/v1/master/files/read?id=${instanceId}&path=${encodeURIComponent(path)}`, { credentials: 'include' }).then(r => r.text())
}

export function apiHealth() {
  return fetch('/health').then(r => r.json())
}

// --- Protected (JWT auth) endpoints ---

export async function apiCreateInstance(name: string): Promise<string> {
  const data = await authedPost(`/api/v1/master/instances/create?name=${encodeURIComponent(name)}`)
  return data.data?.id || ''
}

export async function apiDeleteInstance(id: string) {
  return authedPost(`/api/v1/master/instances/delete?id=${id}`)
}

export async function apiSaveMetadata(id: string, meta: any) {
  return authedPost(`/api/v1/master/instances/metadata?id=${id}`, meta)
}

export async function apiCommit(id: string) {
  return authedPost(`/api/v1/master/instances/commit?id=${id}`)
}

export async function apiPacketUp(id: string) {
  return authedPost(`/api/v1/master/packet_up?id=${id}`)
}

export async function apiPush(id: string) {
  return authedPost(`/api/v1/master/push?id=${id}`)
}

export async function apiWriteFile(id: string, path: string, content: string) {
  await fetch(`/api/v1/master/files/write?id=${id}&path=${encodeURIComponent(path)}`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'text/plain' },
    body: content,
  })
}

export async function apiMkdir(id: string, path: string) {
  return authedPost(`/api/v1/master/files/mkdir?id=${id}&path=${encodeURIComponent(path)}`)
}

export async function apiDeleteFile(id: string, path: string) {
  return authedPost(`/api/v1/master/files/delete?id=${id}&path=${encodeURIComponent(path)}`)
}

export async function apiRenameFile(id: string, path: string, to: string) {
  return authedPost(`/api/v1/master/files/rename?id=${id}&path=${encodeURIComponent(path)}&to=${encodeURIComponent(to)}`)
}

export async function apiUploadFile(id: string, path: string, file: File, onProgress?: (pct: number) => void): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    const formData = new FormData()
    formData.append('file', file)
    xhr.upload.addEventListener('progress', (e) => {
      if (e.lengthComputable && onProgress) onProgress((e.loaded / e.total) * 100)
    })
    xhr.onreadystatechange = () => {
      if (xhr.readyState === 4) {
        if (xhr.status === 200) resolve()
        else reject(new Error(`Upload failed: ${xhr.status}`))
      }
    }
    xhr.open('POST', `/api/v1/master/files/upload?id=${id}&path=${encodeURIComponent(path)}`, true)
    xhr.withCredentials = true
    xhr.send(formData)
  })
}

export async function apiUploadPclPack(id: string, file: File, onProgress?: (pct: number) => void): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    const formData = new FormData()
    formData.append('file', file)
    xhr.upload.addEventListener('progress', (e) => {
      if (e.lengthComputable && onProgress) onProgress((e.loaded / e.total) * 100)
    })
    xhr.onreadystatechange = () => {
      if (xhr.readyState === 4) {
        if (xhr.status === 200) resolve()
        else reject(new Error(`PCL upload failed: ${xhr.status}`))
      }
    }
    xhr.open('POST', `/api/v1/master/files/upload_pcl_pack?id=${id}`, true)
    xhr.withCredentials = true
    xhr.send(formData)
  })
}

export async function apiDownloadRelease(id: string) {
  const ticketRes = await fetch(`/api/v1/master/releases/download-ticket?id=${encodeURIComponent(id)}`, {
    credentials: 'include'
  })
  if (!ticketRes.ok) throw new Error('无法获取下载凭证')
  const ticketData = await ticketRes.json()
  if (!ticketData.success) throw new Error(ticketData.message || '获取下载凭证失败')

  const downloadUrl = `/api/v1/master/releases/download-by-ticket?ticket=${ticketData.data.ticket}`
  const a = document.createElement('a')
  a.href = downloadUrl
  a.download = ''
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

export async function apiMarketDownload(id: string, url: string, fileName: string, type: string) {
  return authedPost(`/api/v1/master/market/download?id=${id}`, {
    url,
    file_name: fileName,
    type,
  })
}

export async function apiCurseProxy(path: string) {
  return authedGet(`/api/v1/master/market/curse-proxy?path=${encodeURIComponent(path)}`)
}

export async function apiModrinthProxy(path: string) {
  return authedGet(`/api/v1/master/market/modrinth-proxy?path=${encodeURIComponent(path)}`)
}

export async function apiReadConfig() {
  return authedGet('/api/v1/master/config/read')
}

export async function apiWriteConfig(config: any) {
  return authedPost('/api/v1/master/config/write', config)
}

export async function apiGetNodesStatus() {
  return authedGet('/api/v1/master/nodes/status')
}

export async function apiGetStats() {
  return authedGet('/api/v1/master/stats')
}

// --- ReleaseEntry types ---

export interface ReleaseEntry {
  version_id: string
  type: 'commit' | 'packet' | ''
  message: string
  file_name: string
  size: number
  time: string
}

export function getReleaseType(entry: ReleaseEntry): 'commit' | 'packet' | 'unknown' {
  if (entry.type === 'commit' || entry.type === 'packet') return entry.type
  if (entry.file_name && !entry.type) return 'packet'
  return 'unknown'
}

// --- Admin: Instance List ---

export async function apiListInstances() {
  return authedGet('/api/v1/sync/instances')
}

// --- Admin: Pre-Auth Key Management ---

export async function apiListPreAuthKeys() {
  return authedGet('/api/v1/admin/preauth/keys')
}

export async function apiCreatePreAuthKey(playerName: string, maxDevices: number, instanceIDs: string[] = []) {
  return authedPost('/api/v1/admin/preauth/keys', { player_name: playerName, max_devices: maxDevices, instance_ids: instanceIDs })
}

export async function apiUpdatePreAuthKey(id: number, playerName: string, maxDevices: number, isActive: boolean, instanceIDs: string[] = []) {
  return authedPut(`/api/v1/admin/preauth/keys/${id}`, { player_name: playerName, max_devices: maxDevices, is_active: isActive, instance_ids: instanceIDs })
}

export async function apiDeletePreAuthKey(id: number) {
  return authedDelete(`/api/v1/admin/preauth/keys/${id}`)
}

export async function apiListKeyBindings(keyId: number) {
  return authedGet(`/api/v1/admin/preauth/keys/${keyId}/bindings`)
}

export async function apiRevokeBinding(bindingId: number) {
  return authedDelete(`/api/v1/admin/preauth/bindings/${bindingId}`)
}
