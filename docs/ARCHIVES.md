# 完整聊天存档

## 保存内容

每个来源会话保存不可变快照，包含统一后的逐条消息和原始来源文件。通过 `source + conversation_id` 与日历记录关联；日历按天分组，聊天快照保留跨日上下文，不按日截断。标题、摘要、次数仍是独立记录；隐藏日历记录不会删除原文。

可选 source_label 保存用户给出的来源标签，例如“ChatGPT · 个人账号”，去除首尾空白后最多 80 字，在日历及存档中展示和筛选。它是可修改的展示元数据，独立于稳定 source 和会话标识；未指定时显示 source。标签在服务清单中保存，不写入不可变 gzip 内容，下载后的当前标签可从清单接口取回。

统一消息字段为 id、role、at、content、part/parts 和 attributes。role 支持 user、assistant、tool、system、developer、unknown；时间未知为 null。消息正文原样保存，长消息按 UTF-8 边界分段，读取时可合并。工具参数、返回、图片结构等可作为结构化内容的文本表示；原始文件同时保留，避免转换遗漏平台特有字段。存档消息数包含 AI 和工具，不能当作用户使用次数。

来源已缺失的消息无法由存档补回。附件实体只有随导出提供并作为来源文件打包时才保存；仅有 URL、路径或临时 ID 时保留引用，并在完整性说明中标记局限。服务不抓取链接，也不读取日志提及的任意本地文件。

## 文件格式

单个快照为 gzip 压缩的 UTF-8 JSONL，每行以 LF 结束。维护契约：`api/archive.schema.json`。按顺序包含：

1. 一个 header：format=`aicalendar.conversation.v1`、来源、会话 ID、标题、原始来源清单。每份来源记录原名、类型、字节数、SHA-256 和分块数。
2. message 行：按来源顺序记录消息，连续 part 从 0 到 parts-1；正文每段最多 65536 UTF-8 字节，attributes 最多 16384 字节。空消息保留；不同修订使用独立稳定 ID。
3. source_chunk 行：来源 ID、顺序编号、原始字节的 base64；每块最多 384 KiB。可以保存导出 JSON、日志、图片或其他二进制附件。下载后可逐份还原并校验，base64 不代表丢弃原始二进制。

存档最多压缩后 128 MiB、展开后 512 MiB、200000 条消息、100 份原始文件；超限明确拒绝，不截断。超大来源可由整理端按连续分卷保存，每卷独立原始文件、准确范围说明和完整性状态。

## 上传协议

浏览器 `/api/v1/archives` 使用门户设备认证；外部 agent `/ingest/v1/archives` 使用现有导入 Bearer。Nginx 和普通 API 继续保持 1 MiB 请求限制，gzip 文件以固定 384 KiB 原始压缩字节分块上传。

| 请求 | 行为 |
| --- | --- |
| GET /archives/schema | 返回存档契约 |
| POST /archives/prepare | 提交清单，取得稳定存档 ID 与已上传分块；相同内容幂等 |
| POST /archives/{id}/parts | 上传 index、data(base64)、sha256；同编号同内容可重试，不允许替换 |
| POST /archives/{id}/commit | 校验分块、gzip、统一消息、原始文件顺序/大小/哈希，完整通过后原子提交 |
| GET /archives/{id} | 查询状态、清单及未完成上传的 received_parts |
| GET /archives | 仅列已完成快照；source、conversation_id、label（来源标签关键词）、q（标题或来源标签关键词）、offset、limit，最多 100 条 |
| GET /archives/{id}/messages | 读取原样消息分段，after 默认 -1，limit 默认 20/最多 200；单页累计约 2 MiB 后提前结束（至少 1 段），以 has_more 判断是否继续；返回 next_after、has_more |
| GET /archives/{id}/download | 下载完整 .jsonl.gz，支持 HEAD，需认证 |

清单包含 source、可选 source_label、conversation_id、title、captured_at、coverage、note、压缩及展开后的 bytes/SHA-256、parts、message_count、source_count。ID 由来源、会话和压缩内容哈希确定；captured_at 的重复打包时间差不新增同内容副本。对同一文件重新 prepare 可通过非空 source_label 更新标签，省略或空值保留已有标签，存档 ID 和数量不变。其他清单差异冲突；内容新增或修订保存新快照，旧快照不覆盖。

prepare/parts 不会出现在聊天列表，commit 事务失败不留下可见消息索引。网络中断不代表提交失败：服务可在接受 commit 后继续最多两分钟验证，重试前读取状态或重用同一固定计划。上传分块保留供恢复，不自动清理未完成资料。客户端提交后下载实际字节比对哈希，不能仅凭 HTTP 200 声称信息完整。

网页按快照分页阅读消息，文本始终转义，不执行聊天中的 HTML/脚本，不自动加载远程图片。整包下载保留完整正文和源文件。现有 `/api/v1/export` 仍导出日历摘要、注释和导入回执；完整存档通过上述列表与下载接口单独取回。

## 存储、备份与升级

所有存档都在现有 `data/aicalendar.sqlite` 中：conversation_archives 保存清单/状态，archive_parts 保存 gzip 文件分块，archive_messages 保存消息分段的压缩阅读索引。没有新增数据目录或备份类型。存档不进入每日备份和发布前备份：快照只包含日历记录、注释与导入审计，以及空的存档表结构，见 [运维](OPERATIONS.md#备份与恢复)。

这是核心 schema 1 上的附加 archive_schema version 1。新 init 一并建表；已有库用 `migrate-archives --db <existing>` 显式事务添加，不重写旧活动/注释，不更改核心 user_version。旧程序仍能打开、展示旧数据并完整备份数据库，但没有存档接口；回退不删除新增表和已存档内容。

升级前使用当前程序生成原生备份，再以 aicalendar 身份运行同提交候选的 migrate-archives，检查备份与扩展，随后按正常 make deploy 发布。未启用扩展时旧功能保持可用，存档接口返回 503。迁移可重复执行；不能通过创建空库修复失败。check 接受原核心库，存在扩展时也检查扩展版本、表和 SQLite 完整性/外键。

原文不自动过期或删除。快照会随内容变化增长，来源重复内容由 gzip 压缩但不同快照仍独立保存。需要真正删除或压缩历史存档时单独设计明确的数据维护操作。
