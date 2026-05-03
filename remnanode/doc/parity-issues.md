# Parity Issues: Go vs TypeScript

审计基准：Go 实现（`remnanode/`）对比 TypeScript/NestJS 实现（`src/`），参考 v2.7.0 同步后的状态（见 `plan.md`）。  
本文档只记录**真正的 bug 和行为差异**，不重复 plan.md 已记录的有意架构差异（supervisord vs os/exec、plugin 系统未实现等）。  
分级说明：P0 = 阻断主控通信，P1 = 安全/正确性缺陷，P2 = 行为差异（不阻断但应修），P3 = 已知/有意差异（仅记录）。

---

## P0 — 阻断主控通信的 wire-protocol / 语义 bug

### 1. Vision 路由前缀缺失 `/vision`

TS 契约 `libs/contract/api/controllers/vision.ts:1` 声明 `VISION_CONTROLLER = 'vision'`，controller 注册路径为 `/vision/block-ip`、`/vision/unblock-ip`（`src/modules/vision/vision.controller.ts:17`）。  
Go 把这两个路由注册在根路径：`remnanode/internal/server/routes.go:119-120`

```go
s.internalRouter.POST("/block-ip", v.BlockIP)
s.internalRouter.POST("/unblock-ip", v.UnblockIP)
```

主控发往 `http://127.0.0.1:61001/vision/block-ip` 的请求会收到 `404`。

**修复**：将 routes.go 中的路径改为 `/vision/block-ip` 和 `/vision/unblock-ip`。

---

### 2. Vision IP 阻断逻辑是未实现的 stub

`remnanode/internal/xray_client/router_ops.go:23-26`：

```go
_, err := c.router.AddRule(ctx, &routerService.AddRuleRequest{
    Config:       nil, // Will be set based on xray-core API
    ShouldAppend: true,
})
```

`Config: nil` 会导致 xray gRPC 直接报错，流量路由规则**永远不会被写入**。同样的 stub 在 `AddSrcIPRule`（`router_ops.go:67-70`）。  
TS 实现 `src/modules/vision/vision.service.ts:26-31` 正确调用 `addSrcIpRule({ruleTag, outbound:'BLOCK', ip})`。

**结论**：Go 的 block-ip 端点永远不会实际屏蔽任何 IP。

---

### 3. Vision 规则 tag MD5 输入格式不兼容

`remnanode/internal/xray_client/router_ops.go:57-59`：

```go
func generateRuleTag(ip string) string {
    hash := md5.Sum([]byte(ip))  // md5(raw IP bytes)
    return fmt.Sprintf("%x", hash)
}
```

TS `src/modules/vision/vision.service.ts:94`：

```ts
return objectHash(ip, { algorithm: 'md5', encoding: 'hex' })
// objectHash 先把 ip 序列化为 "string:<ip>" 再 md5
```

两种实现对同一 IP 生成不同的 hash，导致：
- 如果主控从 TS 节点切换到 Go 节点，之前 TS 写入的阻断规则，Go 的 `UnblockIP` 找不到（tag 不匹配）。
- 即使同一 Go 节点，因为 issue #2 规则本来就没写入，unblock 也无法清理。

---

### 4. `GetNodeHealthCheck` JSON 字段名不同

| 字段 | TS（合约） | Go |
|------|-----------|-----|
| node 是否存活 | `isAlive` | `isHealthy` |
| xray 状态 | `xrayInternalStatusCached` | `isXrayOnline` |

TS 契约 `libs/contract/commands/xray/get-node-health-check.command.ts:10-11`：

```ts
isAlive: z.boolean(),
xrayInternalStatusCached: z.boolean(),
```

Go `remnanode/internal/xray/models.go:36-40`：

```go
type GetNodeHealthCheckResponse struct {
    IsHealthy    bool   `json:"isHealthy"`
    IsXrayOnline bool   `json:"isXrayOnline"`
    ...
}
```

主控读取 `isAlive` 会得到 `undefined`，无法判断节点健康状态。

---

### 5. `GetUserOnlineStatus` 字段名错误且语义不同

**字段名**：TS 契约 `libs/contract/commands/stats/get-user-online-status.command.ts:14` 期望 `isOnline`；  
Go `remnanode/internal/stats/models.go:45` 返回 `json:"online"`。主控看到 `isOnline` 始终为 `undefined`。

**语义**：TS 调用 xray 专用的 `GetStatsOnline` API（online 计数器）；  
Go `remnanode/internal/xray_client/stats_ops.go:79-82` 用 `QueryStats(pattern="user>>>name>>>traffic>>>")` 查流量统计，只要用户有过任何上下行流量就判定为在线 → **误报**（用户离线后仍被判为在线，永不归零，除非 reset）。

---

### 6. 错误响应 JSON 形状完全不同

TS `src/common/exception/http-exception.filter.ts:39-44`：

```json
{ "timestamp": "...", "path": "/node/...", "message": "...", "errorCode": "A009" }
```

Go `remnanode/internal/errors/response.go:24-30`：

```json
{ "isOk": false, "code": "A009", "message": "..." }
```

主控解析 `errorCode` / `timestamp` / `path` 时会失败（字段不存在）。  
Go 也缺少 TS 的 Zod 校验错误透传（`http-exception.filter.ts:29-31`），入参校验失败时返回 500 而非携带字段级错误的 400。

---

## P1 — 安全/正确性问题

### 7. 内部端口中间件可被 `X-Forwarded-For` 欺骗

`remnanode/internal/server/middleware/port_guard.go:24`：

```go
remoteAddr := c.ClientIP()  // Gin 默认信任 X-Forwarded-For
```

`c.ClientIP()` 在未调用 `SetTrustedProxies(nil)` 的 gin 实例上会读取 `X-Forwarded-For` 头，攻击者若能访问 61001 端口，添加该头即可绕过 localhost 检查。  
TS 使用 `req.socket.server.address().port`（服务端信息，不可伪造）并在验证失败时 `socket.destroy()`。

另注：`PortGuard()` 函数（`port_guard.go:11-18`）是**空函数**，注释说"for documentation purposes"，没有任何实际防护。

**修复**：改用 `c.Request.RemoteAddr`，并在 gin Engine 上调用 `SetTrustedProxies(nil)` 或 `[]string{}`。

---

### 8. JWT 中间件接受任何 RSA 算法，不限于 RS256

`remnanode/internal/server/middleware/jwt.go:43`：

```go
if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
    return nil, jwt.ErrSignatureInvalid
}
```

`*jwt.SigningMethodRSA` 匹配 RS256 / RS384 / RS512 / PS256（RSASSA-PSS）。  
TS `src/common/guards/jwt-guards/strategies/validate-token.ts:14` 明确 `algorithms: ['RS256']`。  
接受更弱的算法可能被降级攻击利用。

**修复**：补充 `token.Method.Alg() != "RS256"` 检查。

---

### 9. `POST /node/xray/start` 把完整 xray 配置明文打到日志

`remnanode/internal/xray/handler.go:50-55`：

```go
bodyStr := string(bodyBytes)
if len(bodyStr) > 500 {
    log.Info().Str("body_preview", bodyStr[:500]).Msg("Request body preview")
} else {
    log.Info().Str("body", bodyStr).Msg("Request body")
}
```

xray 配置中包含所有用户的 UUID/密码，这些内容会以 Info 级别写入日志。这是调试残留代码，**生产环境必须移除**。

---

### 10. 缺少请求体大小限制

TS `src/main.ts:62-64`：`bodyParser.json({ limit: '1000mb' })`  
Go gin 默认无限制，超大请求体会消耗全部内存，存在 DoS 隐患。

**修复**：添加 `http.MaxBytesReader` 或在 gin 中设置 `MaxMultipartMemory`，或用 `LimitedReader`。

---

### 11. 缺少 HTTP 安全响应头（`helmet` 等价物）

TS `src/main.ts:77` 使用 `helmet()`，设置 `X-Frame-Options`、`X-Content-Type-Options`、HSTS 等。  
Go 没有等价中间件。

---

### 12. `add-user` / `add-users` 缺少按类型的字段校验

TS 契约 `libs/contract/commands/handler/add-user.command.ts:18-50` 使用 Zod `discriminatedUnion('type', …)`：
- `vless` 必须有 `uuid` + `flow ∈ {'xtls-rprx-vision', ''}`
- `trojan` 必须有 `password`
- `shadowsocks` 必须有 `password`, `cipherType`, `ivCheck`

Go `remnanode/internal/handler/dto.go:17-26` 是一个扁平结构，所有字段都 `omitempty`，类型相关的必填字段无法被 gin validator 强制。  
一个没有 `uuid` 的 `vless` 用户或没有 `password` 的 `trojan` 用户都能通过校验，直到 xray gRPC 调用失败才暴露问题。

---

### 13. xray 进程 `Stop` 发完 `SIGTERM` 不等待进程实际退出

`remnanode/internal/process/manager.go:149-156`：

```go
if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
    m.cmd.Process.Kill()
}
m.isRunning = false  // 立即标记为已停止，但进程可能还在运行
```

`isRunning = false` 立即被设置，但 `cmd.Wait()` 在另一个 goroutine（`manager.go:112`）中运行。`Stop()` 返回后，进程可能仍在退出中，若立即调用 `Start()` 会出现两个 xray 进程争用 gRPC 端口。  
TS 通过 supervisord 的 `stopProcess(wait=true)` 确保进程真正退出后才返回。

---

### 14. 缺少崩溃自动重启（supervisord `autorestart` 等价物）

`remnanode/internal/process/manager.go:111-122`：进程退出监控 goroutine 只记录日志，不重新拉起 xray。  
TS supervisord 配置了 `autorestart=true`，xray 崩溃后会自动恢复。Go 实现崩溃后节点进入无 xray 的降级状态，需要主控重新 POST `/node/xray/start`。

---

### 15. 关闭顺序倒置：先杀 xray 再关 HTTP

`remnanode/cmd/remnanode/main.go:118-121`：

```go
processManager.Cleanup()   // 1. 先杀 xray（断掉 gRPC 连接）
srv.Shutdown(ctx)          // 2. 再关闭 HTTP（正在处理的请求此时找不到 xray）
```

正在处理中的 `/node/handler/*` 请求在 xray 已关但 HTTP 还未关闭的窗口内会收到 gRPC 错误，返回 500。  
TS 顺序：先关 HTTP server（等待在途请求完成），再通过 NestJS lifecycle hook 关闭 supervisord/gRPC client。

**修复**：`srv.Shutdown()` 先于 `processManager.Cleanup()`。

---

## P2 — 行为差异（不阻断但应修）

### 16. 成功响应多包一层 `isOk: true`

Go `errors.SendSuccess()` (`remnanode/internal/errors/response.go:16-21`) 返回：

```json
{ "isOk": true, "response": { ... } }
```

TS controller 直接返回 `{ "response": { ... } }`（无 `isOk` 字段）。  
Zod `z.object()` 默认 strip 未知字段，宽松解析不会报错，但与契约不符；若主控使用 `.strict()` 会失败。

---

### 17. `error` 字段成功时返回 `""` 而非 `null`

`remnanode/internal/handler/models.go:5,12` 无 `omitempty`，成功时 JSON 为 `"error": ""`。  
TS 契约使用 `z.string().nullable()`，成功时期望 `null`。虽然 Zod 接受空字符串，但行为不一致。

---

### 18. `affectedInboundTags` 空数组可能被 gin 拒绝

`remnanode/internal/handler/dto.go:70`：`binding:"required"`  
go-playground/validator v10 对 slice 的 `required` 约束：空 `[]` 在部分版本中视为 zero-value 而被拒绝。  
TS 允许 `affectedInboundTags: []`（空数组代表"不涉及任何 inbound"）。

**建议**：改为 `binding:"omitempty"` 或明确 `min=0`，保留 `required` 的语义仅检查 key 存在。

---

### 19. `GetUserIpList` / `GetStatsOnlineIpList` 每次调用都强制 reset

`remnanode/internal/xray_client/stats_ops.go:290`：`Reset_: true` 硬编码。  
调用者无法在不清零的情况下读取数据；若主控轮询频率高于 xray 更新频率，数据会丢失。

---

### 20. `DISABLE_HASHED_SET_CHECK` 在 handler 模块无效

环境变量被读取并传给 `xray/service.go:49`（控制 `IsNeedRestartCore` 逻辑），但 handler 的 `AddUser`/`RemoveUser` 始终无条件操作 HashedSet，与该 flag 的语义不符。

---

### 21. `WriteTimeout: 30s` 可能截断长时 stats 响应

`remnanode/internal/server/server.go:80-81`：`WriteTimeout: 30 * time.Second`  
`GetCombinedStats` 或 `GetAllInboundsStats` 在 xray 负载高时可能超过 30s，导致客户端收到截断响应。  
TS 无 WriteTimeout 限制。

---

### 22. 版本号硬编码为 `"2.5.0"` 而非 `"2.7.0"`

`remnanode/cmd/remnanode/main.go:27`：`nodeVersion = "2.5.0"`  
尽管 plan.md 记录已同步到 v2.7.0，二进制报告的版本仍是旧版本。主控若依赖版本判断兼容性，可能行为异常。

---

### 23. `GetInboundUsers` 三方字段不一致

| 来源 | 字段 |
|------|------|
| TS 契约 `libs/contract/commands/handler/get-inbound-users.command.ts` | `username`, `email?`, `level?` |
| TS runtime mapper `src/modules/handler/` | `username`, `level`, `protocol` |
| Go `remnanode/internal/handler/models.go:16-20` | `username`, `email?`, `level?` |

Go 与 TS 契约一致，但与 TS runtime 不一致（TS runtime 多出 `protocol`，缺 `email`）。需要先确认上游意图再对齐。

---

### 24. 请求日志在生产环境始终开启

Go 所有请求都会打日志（`internal/server/middleware/logging.go`）。  
TS `src/main.ts:79-81` 仅在 `NODE_ENV=development` 时启用 morgan 请求日志。  
高频 stats 轮询会产生大量日志噪声。

---

## P3 — 已知 / 有意差异（仅记录）

| 项目 | 说明 |
|------|------|
| `GetSystemStats` 响应形状 | v2.7.0 重构为 `{xrayInfo, plugins, system.stats}` 是有意设计；但本仓库 `libs/contract/commands/stats/get-system-stats.command.ts` 仍是旧扁平格式，需要同步上游契约更新 |
| `XTLS_IP` 写死 127.0.0.1 | v2.7.0 故意移除 env var，与上游保持一致 |
| `plugins.torrentBlocker.reportsCount` 恒 0 | plugin 系统未实现，plan.md 已记录 |
| 新增端点（Go-only） | `drop-users-connections`, `drop-ips`, `get-user-ip-list`, `get-users-ip-list`；上游契约尚未补充这些路由 |
| 新增用户类型 | `shadowsocks22`, `hysteria`；上游 TS 契约尚未包含 |
| `connkill`（Linux TCP RST） | Go-only 能力，TS 无此功能；通过 `/drop-ips` 和 `/drop-users-connections` 暴露 |
| supervisord vs `os/exec` | 刻意的架构差异 |
| `XTLS_API_PORT` env var | Go 新增别名，向后兼容 `XTLS_PORT`；TS 只有 `XTLS_PORT` |

---

## 修复优先级汇总

| 优先级 | 问题编号 | 核心影响 |
|--------|----------|---------|
| P0 | #1 | Vision 路由 404 |
| P0 | #2 | IP 阻断无效（stub） |
| P0 | #3 | 跨实现规则 tag 不兼容 |
| P0 | #4 | healthcheck 字段映射失败 |
| P0 | #5 | 在线状态字段名错 + 语义错 |
| P0 | #6 | 错误 JSON 字段全部不同 |
| P1 | #7 | 内部端口可伪造访问 |
| P1 | #8 | JWT 算法未严格限制 RS256 |
| P1 | #9 | 生产日志泄露 UUID（立即删除调试代码） |
| P1 | #13 | Stop 不等进程退出，重启时端口冲突 |
| P1 | #15 | 关闭顺序导致 500 |
| P2 | #22 | 版本号与实际不符 |
| P2 | #17/#18 | error 字段 null vs ""，空数组校验 |
