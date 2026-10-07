# 运维

## 服务形态

项目 AICalendar，回环 `127.0.0.1:18086`，URL `/aicalendar/`，根目录 `/opt/aicalendar`。独立运行用户 aicalendar、受限发布用户 aicalendar-deploy。当前交付状态见 [验证记录](VERIFICATION.md)。

- `bin/aicalendar`：Go 单文件服务，嵌入前端和 API Schema。
- `config/aicalendar.env`：数据库、监听、前缀、正式 Origin、token 哈希文件位置。
- `config/import-token.sha256`：导入 token 的 SHA-256，root:aicalendar 0640；原文只保留在维护端私有目录。
- `data/aicalendar.sqlite`：记录、注释、导入审计及完整聊天存档/上传分块/阅读索引；data/backups 0700。
- `backups/`：原生一致性 SQLite 快照。
- `releases/`：发布材料，遵守共享 3/5 保留策略。

服务进程限制 MemoryMax=192M、GOMEMLIMIT=144MiB，应用仅写自己的 data。前端在维护电脑构建，服务器不装 Node/Go。

## 启动与诊断

serve 不自动建库，不自动迁移。完整聊天扩展通过 migrate-archives 显式添加，升级与旧程序兼容性见 [ARCHIVES.md](ARCHIVES.md)。只有 init 可显式创建新数据库，目标必须不存在。诊断先查 `systemctl status aicalendar`、`journalctl -u aicalendar`、回环 `/healthz` 与配置路径；不能通过创建空库“修复”数据丢失。

页面/API 公网未登录应由门户返回登录跳转或 401。`/ingest/v1/activities` 无 token 必须 401；有 token 才能取得本应用数据。网页写请求 Origin 必须精确匹配正式地址。

token 轮换：在私有位置用 `aicalendar token --out <new-private-token>` 生成新 token/hash，对照授权后只替换服务器 hash 文件并重启本服务，同时更新维护端 token_file。旧 token 失效；不更改门户设备凭据。

## 备份与恢复

`aicalendar-backup.timer` 每天北京时间 04:15（加 0–5 分钟随机延迟）运行；daily 保留 14 份，新快照成功并检查后才轮换。backup.service 使用共享保留锁。门户 registry 自动发现原生 sqlite 备份契约。

```sh
aicalendar backup --db /path/aicalendar.sqlite --out /new/path/backup.sqlite
aicalendar check --db /path/backup.sqlite
aicalendar restore --from /path/backup.sqlite --db /new/path/restored.sqlite
```

backup 使用 VACUUM INTO 一致性快照，包含已提交 WAL 数据，目标必须不存在；check 使用只读连接检查 schema、完整性、外键与必要表；restore 只生成独立新副本。程序回退不恢复数据，真实恢复需确认时点并按停写替换流程执行。

每日快照与生产同盘；异机保障沿用服务器现有方案，不能把同盘快照当作异机备份。已上传的原始导出和完整聊天包含在数据库备份中；尚未上传的本地资料不在服务备份范围。

## 门户、图标与文档

资源声明维护源为 deploy/portal.json，正常发布会预检并同步到 `/opt/aicalendar/config/portal.json`，通过 registry.d 注册、核对加载回执。API 和资源改变时同步声明与 docs。

网站 favicon.svg 与 180×180 不透明 apple-touch-icon.png 位于 web/public；后者由 `python3 scripts/generate-icon.py` 可重建。Nginx 仅为这两个文件放行匿名 GET/HEAD。发布后用共享 check-site-icons.py 验证。

共享规则、发布、认证与保留策略从 [server-operations](https://github.com/Crashmere/agent-config/tree/main/skills/server-operations) 读取。提交推送后运行其 `scripts/sync-docs.sh AICalendar`；普通二进制发布不同步 docs。
