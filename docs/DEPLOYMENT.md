# 本机部署

遵循 [server-operations release](https://github.com/Crashmere/agent-config/blob/main/skills/server-operations/references/release.md)。本仓库 public，GitHub 只保存源码和配置，不配置 Actions 发布或生产 Secrets。

## 首次安装

先完成本地验证、提交推送和 `make release`。按共享规则确认新增运行/发布身份、导入认证与 Nginx 路由的具体安装；先读取远端 `/opt/AGENTS.md`，核对 18086、应用根目录和用户名都未占用。

维护端私有生成导入 token；服务器只安装 SHA-256 文件。传输已校验的 Linux amd64 程序、deploy 材料、hash 与既有部署公钥（不传管理员私钥），核对程序 SHA-256。

```sh
sudo bash deploy/install.sh <linux-program> <https-origin> <token-hash-file> <deploy-public-key>
```

脚本拒绝现有目录/账户/端口；创建空库、运行 unit、04:15 备份 timer 和受限发布身份，验证回环健康与首份备份，再添加本应用的 Nginx location 链接并验证 reload。共享 Nginx server 本身不修改，其他应用不重启。若 Nginx 语法检查失败，移除本次链接，不强行 reload。

首次完成后用正常 `make deploy` 登记已提交程序版本与门户声明，更新共享清单，推送同步项目与共享文档。初始化材料和本地凭据在 Git 外保留。

## 更新

```sh
npm --prefix web ci
make build
# 进行本次功能验证后，提交并 push origin main
make release
make deploy
```

release.sh 复用 server-operations/scripts/release.py，配置在 release.json。只构建已提交文件；产物在 `.local/releases/<commit>/`，包含程序、声明、哈希和构建清单。发布要求干净受跟踪文件与远端备份一致。

受限 SSH 用户仅能执行 deploy、portal-check、portal 三类命令，固定 root 脚本不允许上传者改写。发布先验证大小/哈希、停止本应用、旧程序备份、以运行用户校验候选，再原子替换启动；失败恢复旧程序，不回滚数据库。日志记录 recovery 与 result。

### 首次启用完整聊天存档

此版本追加 archive_schema 1，核心 user_version 仍为 1。先在隔离库验证迁移和旧程序兼容，再按共享授权执行一次显式扩展；普通 make deploy 不自动迁移数据库。

1. 完成本次验证、提交推送和 make release。通过管理员 SSH 上传同提交 program 到独立暂存目录，核对 manifest 中的 SHA-256，让 aicalendar 运行用户可执行候选。
2. 用当前生产程序的 backup 命令创建 `backups/before-archive-extension-<timestamp>.sqlite` 一致性快照；用旧程序 check 校验，记录旧活动与注释的数量/内容摘要。
3. 以 aicalendar 身份执行候选的 `migrate-archives --db /opt/aicalendar/data/aicalendar.sqlite`，再用候选和旧程序分别 check。核对核心 user_version 不变，旧活动/注释未改；新表不存在历史记录。迁移只增表，允许重复执行。
4. 正常 make deploy；核对存档 Schema、授权列表、未授权拒绝和门户声明。清理本次暂存程序，保留升级前快照。旧程序可回退且会保留存档表和字节，但不提供存档接口。

本扩展复用原数据库和所有备份，不新增存储根目录，不更改 Nginx 请求限制、运行身份或 root 发布脚本。格式与完整性约束见 [ARCHIVES.md](ARCHIVES.md)。

## 回退与配置

`make rollback COMMIT=<完整旧提交>` 复用同一发布保护。先确认旧程序与 schema 兼容。真实数据恢复独立确认，使用恢复到新目录的已验证快照。

unit、env、token hash、Nginx 与 root 发布脚本不随普通程序更新；需要时按共享授权表核对并安装。不要在仓库保存生产地址、token 或数据库。

发布验收：回环健康、网页深链接、未授权网页/API、无效 token、公开图标、授权导入只读查询、门户声明回执、backup.service Result。文档另用共享 sync-docs.sh 同步。
