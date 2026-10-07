# AICalendar

个人 AI 活动日历。按日期回顾与 AI 的交流、主题和进展，支持搜索、标签、手动修订与可重复运行的批量导入。

服务接收规范化 JSON；平台资料由外部 agent 或独立 `ai-calendar-import` skill 按需整理。平时使用 AI 不需要上报、hooks 或常驻采集程序。

- 月历、每日时间线、记录检索与导入历史。
- `source + external_id` 去重，批次幂等、乐观版本检查与整批事务。
- 用户标题、摘要、标签与隐藏状态独立保存，重导不会覆盖。
- 未知次数保留 null，时间区间可选，不把 AI 执行耗时当成个人专注时间。
- Go + SQLite；Vue 前端在本机构建后嵌入单个程序。

## 本地运行

需要 go.mod 指定的 Go 工具链和 Node/npm。

```sh
npm --prefix web ci
make build
./bin/aicalendar init --db var/dev.sqlite
./bin/aicalendar serve --db var/dev.sqlite --demo
```

打开 `http://127.0.0.1:18086/aicalendar/`。`--demo` 仅用于回环开发环境；正式服务需要配置 public origin 并通过已认证的反向代理访问。

导入 API 验证可用 `aicalendar token --out <private-file>` 生成私有 token 和 SHA-256 文件，再通过 `--token-hash-file <private-file>.sha256` 启用导入认证。原文 token 不打印到终端。

## 文档

- [架构与数据语义](docs/DESIGN.md)
- [API、Schema 与合成示例](docs/API.md)
- [运维与备份](docs/OPERATIONS.md)
- [首次安装、本机发布与回退](docs/DEPLOYMENT.md)
- [验证范围](docs/VERIFICATION.md)
- [手动导入 skill](https://github.com/Crashmere/agent-config/tree/main/skills/ai-calendar-import)

GitHub 保存源码和配置，推送不触发生产部署。真实数据、凭据和备份均不在仓库内。
