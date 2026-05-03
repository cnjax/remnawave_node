# Parity Issues: Go vs TypeScript

审计基准：当前 Go 实现（`remnanode/`）对比当前 TypeScript/NestJS 实现（`src/`）。本文重点记录 panel 下发 start/restart 后 `rw-core` 不启动相关问题，以及本轮修复后的剩余差异。

---

## 本轮已修复

### 1. `rw-core` API 配置生成已对齐 TS

Go 旧实现会额外注入 `dokodemo-door` API inbound 和 routing rule，并把 `api.tag` 写成 `api`，同时缺少 `api.listen`。这与当前 TS 的 `generateApiConfig()` 不一致，可能导致 `rw-core` 启动后没有 gRPC API，或因 61000 端口冲突立刻退出。

已修复为 TS runtime 形状：
- `stats: {}`
- `api.services = ["HandlerService", "StatsService", "RoutingService"]`
- `api.listen = "127.0.0.1:61000"`
- `api.tag = "REMNAWAVE_API"`
- `policy.levels["0"]` 保留既有字段，并覆盖 stats flags
- 不再注入 API inbound/routing
- 不再原地修改 `req.XrayConfig`

相关文件：
- `remnanode/internal/xray/config_generator.go`
- `remnanode/internal/xray/config_generator_test.go`

### 2. `rw-core` 直接进程管理增加启动失败诊断

Go 仍不使用 supervisord。`rw-core` 继续由 Go 直接以子进程启动：

```text
/usr/local/bin/rw-core -config http://127.0.0.1:61001/internal/get-config -format json
```

已修复：
- `Start()` 后等待 500ms，检测进程是否立刻退出
- 缓存最近 stdout/stderr tail
- 记录 `lastPID`、`lastExitTime`、`lastExitError`
- start/restart 失败时把诊断信息拼进 `response.error`

相关文件：
- `remnanode/internal/process/manager.go`
- `remnanode/internal/xray/service.go`

### 3. 当前 TS stats 响应已对齐

当前 TS `GetSystemStatsResponseModel` 返回 xray sys stats 的扁平结构。Go 旧实现返回 `{xrayInfo, plugins, system}` 新结构，已改回当前 TS runtime 形状。

相关文件：
- `remnanode/internal/stats/models.go`
- `remnanode/internal/stats/service.go`

### 4. 其他修复

- `XRAY_CORE_VERSION` 已做 semver coerce，行为接近 TS `semver.coerce()`。
- `GetInboundUsers` response model 已改为 `username/level/protocol`，对齐当前 TS runtime。
- `AddUsers` 不再完全吞掉 gRPC 错误；全部 add 失败时返回 `success=false`，部分失败会打 warn 日志。

---

## 系统服务

Go 版本不使用 supervisord。本轮新增了系统服务模板，用于托管 `remnanode` 进程本身：

- systemd: `remnanode/contrib/systemd/remnanode.service`
- OpenRC: `remnanode/contrib/openrc/remnanode`

`rw-core` 仍由 `remnanode` 根据 panel 的 start/restart/stop 命令直接管理。这样可以保证每次 panel 下发新 config 后，Go 先更新 `/internal/get-config` 的内存配置，再启动 `rw-core` 拉取该配置。

---

## 剩余差异 / 风险

### 1. `/internal/get-config` 空配置仍返回 `{}` 和 200

TS 也是这个行为，但 Go 直接管理 `rw-core` 后，误启动时 `rw-core` 会拿到空配置并退出。

建议后续处理：
- 在 `processManager.Start()` 前断言 state 中已有非空 generated config。
- 或仅在 Go 实现中让空配置返回 503；这会偏离 TS，需要先确认 `rw-core` 拉配置时的重试行为。

### 2. `GetInboundUsers` 的底层 xray client 仍是 stub

Go response model 已对齐 TS 的 `username/level/protocol`，但 `xray_client.GetInboundUsers()` 当前仍返回空数组。这个问题不影响 `rw-core` 启动，但会影响 inbound 用户查询。

相关文件：
- `remnanode/internal/xray_client/handler_ops.go`

### 3. 非 VLESS 用户的 hash 字段命名仍复用 `vlessUuid`

Go 扩展了 `shadowsocks22`、`hysteria` 类型，但 DTO 仍使用 `hashData.vlessUuid` 维护内部 HashedSet。若 panel 对新增类型使用不同 hash 字段，后续 hashed-set restart 判断可能不同步。

建议后续处理：
- 内部字段改名为 `HashUUID`，JSON 兼容 `vlessUuid`。
- 为新增用户类型明确 hash 来源。
