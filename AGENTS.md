# AICalendar 维护入口

本项目是个人 AI 活动日历，负责活动记录、完整聊天和原始来源存档的存储、展示、检索和幂等导入。平台导出和资料整理属于外部 agent / `ai-calendar-import` skill，不在服务端建立平台连接器、hooks 或自动采集。

- 先读 `docs/README.md`；服务器工作先用 server-operations，并显式读远端 `/opt/AGENTS.md`。
- 代码和配置保存在本 public 仓库；真实记录、原始聊天、凭据、生产地址与备份不入 Git。
- 导入次数是快照值；`source + external_id` 唯一。版本检查、幂等、整批事务、未知值语义、手动注释与来源数据分离必须保持。
- 导入令牌只授权 `/ingest/v1/` 的活动/存档导入、下载和核验。浏览器页面和 `/api/v1/` 继承门户设备认证；修改边界按 server-operations 授权表执行。
- 数据库仅 `init` 显式新建；serve、check、backup 不创建空库掩盖丢失。SQLite backup 使用一致性快照；恢复只写不存在的新路径。
- 本地验证本次功能后构建发布，不设全量回归门槛，不默认保留永久测试。数据导入覆盖、备份恢复使用隔离副本；生产验收只读。
- 网页默认拦截 Tab / Shift+Tab 切换焦点，取消 focus 装饰；保留鼠标、文字输入、选择、修饰键和输入法行为。检查桌面与 375×667 窄屏。
- API、数据目录、端口、unit、备份或路径变化，同步更新 deploy/portal.json、项目 docs 和共享服务器清单。GitHub 只备份源码，本机发布和文档同步分开执行。
- 维护 API 契约时更新 `api/import.schema.json`、`api/archive.schema.json`、`api/openapi.json` 与 docs/API.md，并核对个人 skill 客户端。两边不要维护重复 Schema。
