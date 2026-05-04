# CloudAbroad v6 开发报告 — 第四轮安全修复

> 日期：2026-05-02
> 基于：server_review_v3.md 中记录的 8 项未修复问题

---

## 本轮修复清单（5 项完成，2 项延后）

### ✅ #6 — TOCTOU 实例锁竞态【严重】

**文件**：`main.go`（3 处 handler：commitHandler、packetUpHandler、uploadPclHandler）

将非原子的 `Load` + `Store` 模式改为 `LoadOrStore` 原子操作：

```go
// 修复前（非原子）
if _, locked := instanceLocks.Load(id); locked { ... return }
instanceLocks.Store(id, "Committing")

// 修复后（原子）
if _, loaded := instanceLocks.LoadOrStore(id, "Committing"); loaded { ... return }
```

两个并发请求无法同时通过检查，根除了竞态窗口。

---

### ✅ #10 — SSE handler 裸类型断言【高危】

**文件**：`main.go`（sseHandler 中 2 处）

参照 `broadcastLog` 的修复方式，将裸断言改为安全的 `ok` 模式：

```go
// 修复前
logChannels.Store(id, append(actual.([]chan string), ch))
cs := a.([]chan string)

// 修复后
if chs, ok := actual.([]chan string); ok { ... } else { /* 降级处理 */ }
if chs, ok := a.([]chan string); ok { ... }
```

类型不匹配时不会 panic，有降级处理。

---

### ✅ #8 — 残留错误抑制【高危】（完成剩余部分）

**文件**：`main.go`（3 个函数）

v3 中 `loadConfig` 已修复。本轮完成剩余 3 处：

| 函数 | 修复内容 |
|------|---------|
| `loadInstanceMetadata` | 解析失败返回错误，不再静默返回空结构体 |
| `saveInstanceMetadata` | 序列化/写入失败输出 `fmt.Fprintf(os.Stderr, ...)` 日志 |
| `saveConfig` | 序列化/写入失败输出日志，不再静默丢失 |

---

### ✅ #7 — JWT localStorage → httpOnly Cookie【严重】

这是本轮修改量最大的项目，涉及前后端联动改造。

**后端改动**（`main.go` + `user_handlers.go`）：

1. 新增 4 个 JWT Cookie 辅助函数：
   - `extractJWT(r)` — 从 Cookie → Authorization Header → Query Param 三级降级提取
   - `setJWTCookie(w, token)` — 写入 httpOnly + SameSite=Strict Cookie（24h 有效期）
   - `clearJWTCookie(w)` — 清除 Cookie（MaxAge=-1）
   - `authorizeJWT(w, r)` — 验证 JWT 并设置 X-User-ID/Role/Username 请求头

2. 简化认证中间件：
   - `withJWT` / `withOptionalJWT` / `withWebAuth` / `withPreAuth` 均使用上述辅助函数
   - 移除重复的 token 提取逻辑

3. Login handler（`user_handlers.go`）：
   - 登录成功时调用 `setJWTCookie` 写入 httpOnly Cookie
   - 响应体仍保留 `token` 字段以兼容非浏览器客户端
   - 修复了 sed 全局替换造成的文件损坏（重复 setJWTCookie、authStatusHandler 中的无效调用）

4. 新增 logout handler + 路由 `POST /api/v1/auth/logout`
   - 调用 `clearJWTCookie` 清除 Cookie

5. CORS 中间件增加 `Access-Control-Allow-Credentials: true`

**前端改动**（`api.ts` + `LoginView.vue` + `App.vue`）：

1. `api.ts` — 完全移除 localStorage JWT 模式：
   - 删除 `getJWT()`、`setJWT()`、`clearJWT()`、`jwtHeaders()`
   - 新增 `setCurrentUser()`、`clearCurrentUser()`（仅缓存用户信息，不含 Token）
   - 新增 `apiLogout()` 调用后端 logout 端点
   - 所有 `fetch()` 调用添加 `credentials: 'include'`
   - 所有 `XMLHttpRequest` 上传改为 `xhr.withCredentials = true`

2. `LoginView.vue` — `setJWT(data.token)` → `setCurrentUser(data.user)`

3. `App.vue` — `clearJWT()` → `apiLogout()` + `clearCurrentUser()`

---

### ✅ #15 — 后台协程无优雅退出【中等】

**文件**：`main.go`（heartbeatLoop）

```go
// 修复前
func heartbeatLoop() {
    for range ticker.C { ... }
}

// 修复后
func heartbeatLoop(done <-chan struct{}) {
    for {
        select {
        case <-done:
            return
        case <-ticker.C:
            ...
        }
    }
}
```

关闭序列中先 `close(heartbeatDone)` 通知心跳协程退出，再关闭 HTTP 服务器。

---

### ✅ #16 — HTTP Client 未复用【轻微】

**文件**：`main.go`

新增包级共享客户端，替换全部 7 处一次性创建：

```go
var (
    httpClient      = &http.Client{Timeout: 30 * time.Second}
    probeHTTPClient = &http.Client{Timeout: 5 * time.Second}
)
```

| 原位置 | 原超时 | 替换为 |
|--------|--------|--------|
| `downloadFromMaster` | 30s | `httpClient` |
| `nodesStatusHandler` | 3s | `probeHTTPClient` |
| `marketDownloadHandler` | 30s | `httpClient` |
| `curseProxyHandler` | 30s | `httpClient` |
| `modrinthProxyHandler` | 30s | `httpClient` |
| `probeNode` | 5s | `probeHTTPClient` |
| `sendConfigToSlave` | 10s | `httpClient` |

---

### ✅ #17 — config.example.json 与代码脱节【轻微】

**文件**：`backend/config.example.json`

更新为与实际 `Config` 结构体一致的格式：
- 新增：`node_name`、`role`、`public_ip`、`dist_policy`、`pre_auth_secret`、`pre_auth_enabled`、`nodes[]`
- 移除：`port`、`data_dir`、`login_protection`（已硬编码/废弃）

---

## 延后项目

### ⏸️ #11 巨石架构 & #12 全局状态【中等】

这两项是架构级重构（拆分 package、依赖注入），不影响安全性与功能正确性。当前修复已使代码安全基线达标，包拆分作为后续优化单独进行更为稳妥，避免引入回归。

---

## 编译验证

| 检查项 | 结果 |
|--------|------|
| `go build ./...` | ✅ 通过 |
| `go vet ./...` | ✅ 通过 |
| `npx tsc --noEmit`（前端） | ✅ 通过 |

---

## 修复追踪总表（17 项问题最终状态）

| # | 问题 | 等级 | 状态 |
|---|------|------|------|
| 1 | HMAC 主密钥泄露 | 严重 | ✅ v2 修复 |
| 2 | 路径穿越 URL 编码绕过 | 严重 | ✅ v2 修复 |
| 3 | 重放攻击窗口 300s | 高危 | ✅ v2 修复 |
| 4 | broadcastLog 类型断言 panic | 高危 | ✅ v2 修复 |
| 5 | generateRandomID 随机性 | 中等 | ✅ v2 修复 |
| 6 | TOCTOU 实例锁竞态 | 严重 | ✅ **本轮修复** |
| 7 | JWT localStorage | 严重 | ✅ **本轮修复** |
| 8 | 普遍错误忽略 | 高危 | ✅ **本轮修复** |
| 9 | SSRF 域名白名单 | 高危 | ✅ v3 修复 |
| 10 | SSE handler 裸断言 | 高危 | ✅ **本轮修复** |
| 11 | 巨石架构 | 中等 | ⏸️ 延后 |
| 12 | 全局状态 | 中等 | ⏸️ 延后 |
| 13 | 系统指标仅 Linux | 中等 | ✅ v3 修复 |
| 14 | SSE 资源泄漏 | 中等 | ✅ v3 修复 |
| 15 | 后台协程退出 | 中等 | ✅ **本轮修复** |
| 16 | HTTP Client 复用 | 轻微 | ✅ **本轮修复** |
| 17 | config.example 脱节 | 轻微 | ✅ **本轮修复** |

**总计**：17 项中发现并修复 15 项，延后 2 项（中等优先级架构改进）。

- 严重问题：4/4 已修复（100%）
- 高危问题：4/4 已修复（100%）
- 中等问题：5/7 已修复（71%）
- 轻微问题：3/3 已修复（100%）

---

## 变更文件清单

| 文件 | 变更类型 |
|------|---------|
| `Server/backend/main.go` | TOCTOU 锁、SSE 断言、错误抑制、JWT Cookie 中间件、心跳优雅退出、HTTP Client 复用 |
| `Server/backend/user_handlers.go` | Login/logout handler Cookie 化、修复 sed 损坏 |
| `Server/backend/config.example.json` | 更新为实际 Config 结构体 |
| `Server/frontend/src/api.ts` | 移除 localStorage JWT，改用 Cookie + credentials |
| `Server/frontend/src/components/LoginView.vue` | setJWT → setCurrentUser |
| `Server/frontend/src/App.vue` | clearJWT → apiLogout + clearCurrentUser |
