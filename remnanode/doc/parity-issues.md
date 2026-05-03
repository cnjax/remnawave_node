# Parity Issues: Go vs TypeScript v2.7

审计基准：当前 Go 实现（`remnanode/`）对比 TypeScript/NestJS v2.7 实现（`src/`）。本文以 TS 为准，列出所有 Go 侧的偏差。  
分级：P0 = 阻断或严重不兼容，P1 = 安全/正确性问题，P2 = 行为差异，P3 = 已知/有意差异（仅记录）。

---

## P0 — 阻断或严重不兼容

### ✅ 1. `generateApiConfig` — dokodemo-door API inbound + 路由规则 + mTLS

**已修复**（`remnanode/internal/xray/config_generator.go`）：
- 移除了硬编码 `api.listen`；
- 在 `inbounds` 数组**最前**注入 `REMNAWAVE_API_INBOUND`（dokodemo-door）inbound；
- TLS 物料（caCertPem/nodeCertPem/nodeKeyPem）按行拆分后写入 `streamSettings.tlsSettings`；
- `XRAY_ROUTING_RULES_MODEL` 作为 `routing.rules[0]` 注入；
- `policy.levels['0'].statsUserOnline` 由 `hasCapNetAdmin` 参数控制；
- 函数签名更新为 `GenerateAPIConfig(config, xtlsPort, caCertPem, nodeCertPem, nodeKeyPem, hasCapNetAdmin)`；
- 测试文件（`config_generator_test.go`）同步更新。

### ✅ 2. xray gRPC client 升级为 mTLS（使用 ephemeral 证书）

**已修复**（`remnanode/internal/xray_client/client.go`、新建 `remnanode/pkg/mtls/gen.go`）：
- TS 实现在启动时通过 `generateMTLSCertificates()` 生成独立的 ephemeral mTLS 证书集（CA + server + client），**不使用** SECRET_KEY 中的 `nodeCertPem`；
- Go 实现同步：新建 `pkg/mtls/gen.go`，启动时生成 ephemeral 证书集，server 证书含 `DNS: internal.remnawave.local` SAN（Go 1.15+ 要求 SAN，不接受仅 CN 的证书）；
- `NewClient(ip, port string)` 通过 `mtls.Get()` 获取 client cert + CA，构造 mTLS；
- `config_generator.go` 也改用 ephemeral server cert，dokodemo-door TLS 配置添加 `{usage: "verify", certificate: caCertPEM}` 用于验证 client cert；
- `ServerName: "internal.remnawave.local"`；
- `Reconnect()` 同步使用 mTLS；
- `MaxMessageSize = 100MB` 保持不变。

### ✅ 3. 内部端点 token 鉴权 + `/internal/webhook`

**已修复**：
- 新增 `INTERNAL_REST_TOKEN` 必填环境变量（`config/env.go`）；
- 新增 `middleware.TokenAuth(token)` 中间件，token 不匹配时 hijack+close（`middleware/port_guard.go`）；
- 内部路由器统一应用 `TokenAuth` 中间件（`server/server.go`）；
- 新增 `POST /internal/webhook` 端点（接收 body、记录日志、丢弃）（`internal_api/handler.go`、`routes.go`）。

### ✅ 4. `StartXrayResponse` 字段 `systemInformation` → `system`，补全 stats/interface

**已修复**（`remnanode/internal/xray/models.go`、`xray/service.go`）：
- JSON 字段 `systemInformation` 改为 `system`；
- `NodeSystem` 结构体包含嵌套的 `info`（NodeSystemInfo）、`stats`（SystemStats）、`interface`（NetworkInterfaceStats）；
- `buildNodeSystem()` 在每次 StartXray 响应时动态采集 `sysinfo.GetSystemStats()`。

### ✅ 5. `addUsers` 中 hysteria / shadowsocks22 / shadowsocks 实现错误

**已修复**（`remnanode/internal/handler/service.go`）：
- `hysteria` → `user.UserData.VlessUUID`（不再用 `TrojanPassword`）；
- `shadowsocks22` → `base64.StdEncoding.EncodeToString([]byte(user.UserData.SSPassword))`；
- `shadowsocks` → `CipherTypeUnknown(0)`（不再用 CHACHA20POLY1305）；
- 同步修复了单用户 `AddUser` 路径中的 shadowsocks22 base64 编码。

### ✅ 6. `statsUserOnline` 基于 CAP_NET_ADMIN 探测

**已修复**：
- 新建 `pkg/capabilities/capabilities.go`，通过读取 `/proc/self/status` 的 `CapEff` 字段检测 `CAP_NET_ADMIN`；
- `xray/service.go` 在 `NewService` 时调用 `capabilities.HasCapNetAdmin()` 并缓存结果；
- `GenerateAPIConfig` 接受 `hasCapNetAdmin bool` 参数，用于设置 `policy.levels['0'].statsUserOnline`。

### ✅ 7. `GetUserOnlineStatus` 改用 `GetStats` online 计数器

**已修复**（`remnanode/internal/xray_client/stats_ops.go`）：
- 改为调用 `c.stats.GetStats(..., Name: "user>>>username>>>online", Reset_: false)`；
- 读取 `resp.Stat.Value > 0` 判断在线状态，与 TS xtls-sdk `getUserOnlineStatus` 一致。

### ✅ 8. 缺失路由

**已修复**（`remnanode/internal/server/routes.go`、`plugin/handler.go`）：
- `POST /node/handler/remove-users` — 已注册（service 函数已存在）；
- `POST /node/plugin/torrent-blocker/collect` — 返回 `{accepted: false}` 占位；
- `POST /node/plugin/nftables/block-ips` — 返回 `{accepted: false}` 占位；
- `POST /node/plugin/nftables/unblock-ips` — 返回 `{accepted: false}` 占位；
- `POST /node/plugin/nftables/recreate-tables` — 返回 `{accepted: false}` 占位；
- `POST /internal/webhook` — 见 §3。

---

## P1 — 安全/正确性问题

### ✅ 9. 错误响应缺少 Zod-style 校验错误透传

**已修复**（所有 handler 文件）：
- 新增 `ErrValidation`（A020, HTTP 400）；
- 所有 `ShouldBindJSON` 失败从 `ErrInternalServer(500)` 改为 `ErrValidation(400)`，并传递 validator 错误详情。

### ✅ 10. `Recovery` 中间件用了非标准错误形状

**已修复**（`middleware/recovery.go`）：
- panic 响应改为 `errors.SendError(c, errors.ErrInternalServer)`，形状统一为 `{timestamp, path, message, errorCode}`。

### ✅ 11. JWT 失败行为：Go 返回 401 JSON，TS 直接断连

**已修复**（`middleware/jwt.go`、新建 `middleware/hijack.go`）：
- JWT 校验失败（含 Header 缺失、Bearer 格式错误、token 无效）时改为 `hijackAndClose`，不再返回 JSON。

### ✅ 12. `NotFoundException` 行为：Go 返回 404，TS 断连

**已修复**（`server/server.go`）：
- 注册 `NoRoute` handler，调用 `middleware.CloseUnknownRoute` → `hijackAndClose`。

### ✅ 13. 内部端口缺少 token 鉴权

见 §3（已修复）。

### ✅ 14. `xrayClient.GetInboundUsers` 是空 stub

**已修复**（`xray_client/handler_ops.go`）：
- 调用 `HandlerService.GetInboundUsers`（`GetInboundUserRequest{Tag: tag}`），返回真实 `username/email/level`。

### ✅ 15. `RemoveUser` (xray gRPC) 吞错

**已修复**（`xray_client/handler_ops.go`）：
- "not found" / `code=NotFound` 错误忽略；其他错误向上传递。

---

## P2 — 行为差异（不阻断但应修）

### ✅ 16. `InboundUser` 响应字段：Go 多了 `protocol`，少了 `email`

**已修复**（`handler/models.go`、`xray_client/handler_ops.go`、`handler/service.go`）：
- 删除 `protocol` 字段，补充 `email` 可选字段，对齐 contract `{username, email?, level?}`。

### ✅ 17. `Hysteria` 单用户分支 DTO 校验错位

**已修复**（`handler/service.go`）：
- `validateInboundUser` 中 hysteria 分支改为 require `UUID`（不再 require `password`）。

### ✅ 21. `InternalOnly` 中间件 errorCode 复用了 `A004 Forbidden role`

**已修复**（`errors/codes.go`、`middleware/port_guard.go`）：
- 新增独立错误码 `A019` (`ErrInternalOnlyAccess`)，日志可区分。

### ✅ 19. `compression()` 出向 gzip 压缩

**已修复**（`middleware/gzip.go`、`server/server.go`）：
- 新增 `GzipCompress()` 中间件，使用 `klauspost/compress/gzip`（已有依赖）；
- 当客户端 `Accept-Encoding: gzip` 时包装 ResponseWriter 进行压缩，并设置 `Content-Encoding: gzip`；
- 通过 `sync.Pool` 复用 gzip.Writer，避免频繁分配；
- 已应用到主路由器（matching TS `app.use(compression())`）。

### ✅ 18. `Stop()` 加入 `WithPluginCleanup` / `WithOnlineCheck` 参数

**已修复**（`xray/service.go`）：
- 新增 `StopXrayOptions{WithOnlineCheck, WithPluginCleanup}` 结构体；
- `StopXray(opts ...StopXrayOptions)` 使用可变参数，现有调用方无需修改；
- `WithOnlineCheck=true` 且具备 `CAP_NET_ADMIN` 时，停止前调用 `dropAllOnlineConnections()`，
  遍历所有在线用户获取其 IP 列表，再通过 `connkill.DropByIPs` RST 所有 TCP 连接；
- `WithPluginCleanup=true` 当前为 stub（log + no-op），等 plugin 模块到位后接入。

### ✅ 20. `RemoveUser` 单用户触发连接断开（DropConnectionsEvent）

**已修复**（`handler/service.go`）：
- `handler.Service` 新增 `hasCapNetAdmin bool`（`NewService` 调用 `capabilities.HasCapNetAdmin()` 初始化）；
- `RemoveUser` 成功后若 `hasCapNetAdmin=true`，调用 `dropUserConnections(username)`：
  读取 `user>>>username>>>online` 的 IP 列表，再通过 `connkill.DropByIPs` RST，与 TS `DropConnectionsEvent` 行为对齐。

### ✅ 23. `Drop*` 端点加 `hasCapNetAdmin` 守卫

**已修复**（`handler/service.go`）：
- `DropUsersConnections` 和 `DropIps` 在 `hasCapNetAdmin=false` 时直接返回 `{success: true}` 并记录 warn 日志，
  避免在无权限容器中因 `SOCK_DESTROY` 失败而误报错误。

### 22. `/health` 与 `/node/xray/status` 是 Go-only

TS 没有这两个端点。保留为 Go 扩展，见 P3。

---

## P3 — 已知/有意差异（仅记录）

| 项目 | 说明 |
|---|---|
| 进程管理 | TS 通过 supervisord 管理 xray，Go 直接 `os/exec`。有意架构差异。 |
| 监听传输 | TS 内部走 Unix socket，Go 走 `127.0.0.1:61001` TCP。已加 token 鉴权（见 §3）。 |
| 进程崩溃自愈 | supervisord `autorestart=false`，TS 也不自愈；Go 行为相同。 |
| `/health` / `/node/xray/status` | Go-only 扩展端点。 |
| `XTLS_API_PORT` 别名 | Go 对 `XTLS_PORT` 兼容。 |
| `XRAY_CORE_VERSION` 解析 | Go `coerceSemver` 模仿 TS `semver.coerce()`，行为已对齐。 |
| `connkill`（Linux TCP RST）| Go-only，未来与 plugin 整合。 |
| 新增 user types `shadowsocks22`/`hysteria` | 实现已按 §5/§17 修复。 |
| `SUPERVISORD_*` 环境变量 | Go 不需要。 |
| `XRAY_BINARY_PATH` 默认 | TS 默认 `/usr/local/bin/xray`，Go 默认 `/usr/local/bin/rw-core`（有意改名）。 |

---

## 修复优先级汇总

| 优先级 | 编号 | 简述 | 状态 |
|---|---|---|---|
| P0 | §1 | API config 缺 dokodemo-door + mTLS inbound + routing rule | ✅ 已修复 |
| P0 | §2 | xray gRPC client 必须升级为 mTLS | ✅ 已修复 |
| P0 | §3 | 内部端口加 `?token=` 鉴权 + 新增 `/internal/webhook` | ✅ 已修复 |
| P0 | §4 | `StartXrayResponse` 字段 `systemInformation` → `system`，补全 stats/interface | ✅ 已修复 |
| P0 | §5 | `addUsers` 中 hysteria/ss22/ss 字段错位 | ✅ 已修复 |
| P0 | §6 | `statsUserOnline` 基于 CAP_NET_ADMIN 探测 | ✅ 已修复 |
| P0 | §7 | `GetUserOnlineStatus` 改用 `GetStats` online 计数器 | ✅ 已修复 |
| P0 | §8 | 缺失路由：`remove-users`、plugin/torrent-blocker、plugin/nftables/* | ✅ 已修复 |
| P1 | §9 | 校验错误返回 400 + 结构化 | ✅ 已修复 |
| P1 | §10 | Recovery 响应使用统一 errorPayload | ✅ 已修复 |
| P1 | §11/§12 | JWT/404 失败时 hijack+close | ✅ 已修复 |
| P1 | §14 | 实现真实 `GetInboundUsers` | ✅ 已修复 |
| P1 | §15 | `RemoveUser` gRPC 错误别吞 | ✅ 已修复 |
| P2 | §16 | `InboundUser` 响应字段对齐 contract | ✅ 已修复 |
| P2 | §17 | hysteria 单用户 DTO 校验用 UUID | ✅ 已修复 |
| P2 | §21 | InternalOnly 独立 errorCode A019 | ✅ 已修复 |
| P2 | §19 | 出向 gzip 响应压缩 | ✅ 已修复 |
| P2 | §18 | StopXray WithOnlineCheck/WithPluginCleanup 选项 | ✅ 已修复 |
| P2 | §20 | RemoveUser 后触发连接断开 | ✅ 已修复 |
| P2 | §23 | Drop* 端点加 CAP_NET_ADMIN 守卫 | ✅ 已修复 |
| P2 | §22 | /health, /node/xray/status Go-only 扩展端点 | P3（保留） |
