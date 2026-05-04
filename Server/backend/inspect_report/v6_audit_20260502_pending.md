# CloudAbroad v6 未修复问题清单

**来源**: v6_audit_20260502.md
**提取日期**: 2026-05-02
**状态**: 36 项未修复（1 严重 + 8 高优 + 12 中等 + 15 低优）

---

## 🔴 严重问题 (1项)

### C1. 预编译二进制 + 密钥文件混入仓库

**文件**: 
- `Server/backend/pcl-sync-master` (16MB ELF)
- `Server/backend/pcl-sync-master.exe` (17MB PE)
- `Server/backend/server.exe` (17MB PE)  
- `Server/backend/cloudabroad-v6.exe` (17MB PE, **新增**)
- `Server/deploy.tar.gz`, `Server/deploy_v6.zip`, `Server/deploy.tar.gz`, `Server/web.tar.gz`
- `Server/backend/config.json` — 明文密钥：`"secret_key": "CloudAbroad_Secret_2026"`, `"jwt_secret": "CloudAbroad_JWT_e6a099c3"`
- `Server/deploy/config.json` — 同样含明文密钥

**影响**: 无法审计二进制安全性；构建产物无版本追踪；密钥可被所有代码访问者读取。

**修复**: 添加 `.gitignore` 忽略 `*.exe`、构建产出、`config.json`；提交 `config.example.json` 代替。

---

## 🟠 高优问题 (8项)

### H2. JWT Token 通过 URL 查询参数泄露

**文件**: `frontend/src/api.ts:273-283`
JWT 24h 有效期，出现在浏览器历史、服务器日志、Referer 头中。

### H3. IP 伪造 — `getRealIP()` 无条件信任代理头

**文件**: `backend/user_handlers.go:446-464`
CF-Connecting-IP、X-Real-IP、X-Forwarded-For 全部无条件信任，无 `trusted_proxy` 配置。

### H4. 管理员唯一性 TOCTOU 竞态

**文件**: `backend/store.go:221-228` (CreateUser), `backend/store.go:299-313` (UpdateUser)
SELECT COUNT 与 INSERT/UPDATE 间无事务保护。

### H5. 设备绑定与计数器非原子操作

**文件**: `backend/preauth_handlers.go:89-96`
`CreateDeviceBinding` 与 `IncrementPreAuthKeyDevicesBound` 分步执行，无事务包装。

### H6. 文件重命名缺前端路径穿越校验

**文件**: `frontend/src/components/ExplorerView.vue:376`
`prompt()` 输入直接拼入路径参数。

### H7. 密码通过 `prompt()` 明文输入

**文件**: `frontend/src/components/SettingsView.vue:370`
无掩码，有肩窥风险。

### H9. `config.json` / `deploy/config.json` 含明文密钥

同一密钥多份拷贝在仓库中。若仓库公开或被非授权访问，预共享密钥即泄露。

### H10. 全服务纯 HTTP 无 TLS

两端口（:55000, :55001）均为 `http.ListenAndServe`。JWT、admin 密码、PCL 包明文传输。

> **注**: 协议自适应设计已加入 implementation_plan.md §6.4，服务端 URL 已去除 `http://` 硬编码。Nginx TLS 终止需在生产部署时配置。

---

## 🟡 中等问题 (12项)

| 编号 | 问题 | 文件 |
|------|------|------|
| M1 | deleteInstance 不清理用户 instances 关联 | `backend/main.go:798-806` |
| M2 | getMasterIP() 多 Slave 时逻辑错误 | `backend/slave.go:222-229` |
| M3 | mergeDir 中 os.Rename 错误被忽略，跨设备时丢文件 | `backend/modpack.go:199-209` |
| M4 | Modpack 下载全部镜像失败时静默跳过 | `backend/modpack.go:94-110` |
| M5 | 密码仅检查非空，允许单字符 | `backend/user_handlers.go:161,361` |
| M6 | CORS 对非白名单 Origin 返回服务器地址而非请求 Origin | `backend/main.go:463-479` |
| M7 | PCL 多文件上传第一个完成即跳转，后续中断 | `frontend/src/components/ExplorerView.vue:448-450` |
| M8 | 侧边栏时间轴仅显示 packet，commit 不可见 | `frontend/src/App.vue:237` |
| M9 | ExplorerView 底栏下载发行包无前置检查 | `frontend/src/components/ExplorerView.vue:402-408` |
| M10 | 剪贴板写入失败空 catch → 密钥可能永久丢失 | `frontend/src/components/PreAuthView.vue:202` |
| M11 | 无请求体大小限制（可 OOM） | `backend/main.go:1799-1815` |
| M12 | CountLoginFailures SQL 字符串拼接 | `backend/store.go:379-389` |

---

## 🟢 低优问题 (15项)

| 编号 | 问题 | 文件 |
|------|------|------|
| L1 | `withOptionalJWT` 死代码 | `backend/main.go:248-268` |
| L2 | `loadConfig` 忽略 JSON Unmarshal 错误 | `backend/main.go:541-542` |
| L3 | `hmacGet`/`probeNode` 忽略 `http.NewRequest` 错误 | `backend/main.go:157,1605` |
| L4 | `loadInstanceMetadata`/`saveInstanceMetadata` 忽略错误 | `backend/main.go:532,538` |
| L5 | SSE 连接无心跳保活 | `backend/main.go:1418-1463` |
| L6 | `readTotalSystemMemoryMB` 仅 Linux | `backend/stats.go:37-54` |
| L7 | `/health` 无鉴权暴露 node name | `backend/main.go:594-600` |
| L8 | `hashchange` 监听器未在 `onUnmounted` 移除 | `frontend/src/App.vue:328` |
| L9 | EditorView 直接修改 props | `frontend/src/components/EditorView.vue:198` |
| L10 | HelloWorld.vue + 脚手架资产残留 | `frontend/src/components/HelloWorld.vue` |
| L11 | XHR 事件监听器未清理 | `frontend/src/api.ts:232-271` |
| L12 | `getCurrentUser()` 返回 `any` | `frontend/src/api.ts:18-21` |
| L13 | `index.html` 元数据: `lang="en"`, title="web" | `frontend/index.html:2,7` |
| L14 | 开发脚本混入源码目录 | `backend/fix_main.py`, `backend/gen_main.py` |
| L15 | `deploy/`、`deploy_pkg/` 目录冗余 | `Server/deploy/`, `Server/deploy_pkg/` |

---

## 优先修复建议 (Top 5)

| 优先级 | 问题 | 理由 |
|--------|------|------|
| **P0** | H9: 明文密钥 + C1: 二进制混入 | 仓库安全基线；解决后即可安全共享代码 |
| **P1** | H4: Admin TOCTOU | 可用并发请求创建多个管理员 |
| **P1** | H5: 设备绑定非原子 | 计数漂移影响预授权配额准确性 |
| **P1** | H2+H10: JWT泄露 + 无TLS | 生产部署前的硬安全需求 |
| **P2** | M3+M7: 丢文件 + 上传中断 | 用户直接可感知的数据丢失 |

---

*此清单从 v6_audit_20260502.md 提取，仅包含未修复项。3 项已修复问题（C1 SSE 竞态、H1 Slave 同步取错 Release、H8 外部图片 URL 校验）不在此列。*
