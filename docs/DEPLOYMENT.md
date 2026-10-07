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

## 回退与配置

`make rollback COMMIT=<完整旧提交>` 复用同一发布保护。先确认旧程序与 schema 兼容。真实数据恢复独立确认，使用恢复到新目录的已验证快照。

unit、env、token hash、Nginx 与 root 发布脚本不随普通程序更新；需要时按共享授权表核对并安装。不要在仓库保存生产地址、token 或数据库。

发布验收：回环健康、网页深链接、未授权网页/API、无效 token、公开图标、授权导入只读查询、门户声明回执、backup.service Result。文档另用共享 sync-docs.sh 同步。
