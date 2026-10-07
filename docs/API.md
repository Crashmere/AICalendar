# API v1

契约源：[JSON Schema](../api/import.schema.json)、[OpenAPI](../api/openapi.json)、[合成导入请求](../api/import-example.json)。路径均相对于 `/aicalendar/`。

## 身份

网页 API 为 `api/v1/`，由 ServerPortal 设备 Cookie 保护；写请求还需本站 Origin。程序调用使用 `ingest/v1/` 和 `Authorization: Bearer <token>`。不要从浏览器提取全站 Cookie 给 agent 使用。

| 方法 | 网页路径 | 导入凭据路径 | 作用 |
| --- | --- | --- | --- |
| GET | api/v1/info | — | 名称、默认时区、今日日期、演示状态 |
| GET | api/v1/schema | ingest/v1/schema | 当前运行版本的 JSON Schema |
| POST | api/v1/imports/preview | ingest/v1/imports/preview | 校验并返回逐条 insert/update/unchanged/conflict |
| POST | api/v1/imports | ingest/v1/imports | 整批原子提交 |
| GET | api/v1/imports | ingest/v1/imports | 导入列表与覆盖声明，不含逐条结果 |
| GET | api/v1/imports/{id} | ingest/v1/imports/{id} | 持久化批次回执 |
| GET | api/v1/activities | ingest/v1/activities | 查询规范化记录与注释 |
| PATCH | api/v1/activities/{id}/annotation | — | 乐观版本保护的用户注释替换 |
| GET | api/v1/export | — | 导出记录、注释和批次概况 |

健康检查为 `GET /healthz`，由回环发布检查使用，公网仍需要门户认证。

## 查询

activities 参数：`from`、`to`（YYYY-MM-DD，包含边界）、`source`、`q`、`tag`、`include_hidden=true`、`offset`、`limit`（1–5000，默认 500）。返回 `{items,total,offset,limit}`。每项含 `id`、来源 `version`、`annotation_version`、`record`、`annotation`、`updated_at`。

q 搜索最终展示的标题、摘要、主题标签和来源标签；用户注释优先于来源值。tag 精确匹配主题标签或 source_label。返回的 record 保持原始规范化来源，方便重新导入核对；隐藏项默认不返回。

## 导入

请求需 `schema_version:1`、`idempotency_key`、`mode` 和 `records`。可以附带 `coverage`。每批最多 500 条记录、50 个覆盖声明和 1 MiB；允许 records 为空但 coverage 非空。

默认 insert_only；相同唯一键与内容为 unchanged，不同内容为 conflict。upsert 更新已有记录必须提供 expected_version。次数是完整快照，不能增量相加。服务拒绝减少/清空已知次数、observed 次数降级、final 降为 partial、删除已知时间/时段、更换会话/日期/时区。

记录必须提供有意义的质量说明。time 支持 observed_interval、observed_timestamps、estimated_interaction_span、date_only、unknown；无区间依据时 spans 为空。日期与时区是记录身份的一部分。来源 source 支持字母数字及 `_.-`，external_id 由整理端稳定生成。

`record_state` 是单条记录的整理进度：待整理用 partial，所选资料已整理完成用 final。它独立于来源范围 `coverage[].status` 和聊天存档 `manifest.coverage`；来源覆盖 partial 不要求记录也为 partial。final 记录仍可按版本补充新资料，详见 [记录整理状态](DESIGN.md#记录整理状态)。

可选 source_label 是用户指定的展示来源，例如“ChatGPT · 个人账号”或“Trae · 工作电脑”，去除首尾空白后最多 80 字。日历显示该标签并允许筛选；省略时显示 source。source 继续作为稳定的来源/账户命名空间，不能随标签改名。更新标签使用普通 upsert 和 expected_version，不更换 external_id、不增加次数；手动注释中的主题标签与来源标签独立。

预览返回 200，冲突数量在 conflicts；写入遇到冲突返回 409、committed=false，整批无变更。成功回执包含 id、时间、inserted/updated/unchanged 和逐项 ID/版本。相同幂等键、相同请求重试返回 replay=true；同键不同请求返回 409。

解析失败、未知字段或校验失败返回 400。网络失败可以重试原请求；不要生成新标识绕过重试。所有 5xx 信息对外简化，详细原因记入 journal。

## 注释

PATCH 请求为 `{expected_version: <annotation_version>, annotation: {...}}`，完整替换注释对象。支持 title、summary、tags、hidden。省略或 null 的文本/标签回退到来源值；空标签数组表示用户清空标签。设置 `{hidden:false}` 可恢复全部来源展示。版本不匹配返回 409。

## 导出与恢复

导出结构为 `{schema_version,exported_at,activities,imports}`，其中 activities 包含 record 与 annotation。导出包不是直接导入请求；重导需提取 record，再显式规划注释恢复。完整恢复使用 SQLite 一致性备份；JSON 导出不能代替运行配置与导入审计的完整备份。

## 完整聊天

日历导入 v1 保持兼容。完整聊天使用同一认证命名空间中的 `/archives` 系列接口，以不可变会话快照保存统一消息和原始导出；分块格式、校验、分页阅读与下载见 [ARCHIVES.md](ARCHIVES.md)，Schema 为 `api/archive.schema.json`。
