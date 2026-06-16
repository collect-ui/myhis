# 00 最小接入模版 demo

新增一个最小可访问页面，至少需要这 4 个落点：

1. `collect/frontend/page_data/data/<group>/<page>.json`
2. `collect/frontend/page_data/index.yml`
3. `sys_menu` 里的菜单记录
4. 目标页面依赖的后端 service

最小页面接入方式：

- 页面 JSON 负责渲染和动作编排。
- `frontend.<page>` 在 `page_data/index.yml` 里映射到 JSON 文件。
- 菜单 `api` 指向 `post:/template_data/data?service=frontend.<page>`。
- 路由 `url` 一般挂到 `/framework/<page>`。

页面配置约束：

- `ajax` 优先只做取数和最小字段映射，例如 `dataList: "${data}"`、`count: "${count}"`。
- 不要在 `ajax.adapt` 里同时处理列表筛选、选中态切换、表单回填、删除后重选这类多段状态逻辑。
- 不要在 `ajax.adapt` 里复制其他 store 对象，例如把 `sessionInfo`、`pageForm`、`selection` 一起重算后回写。
- 这类状态变更应拆到单独的 `update-store` 或 `update-form`，让“取数”和“状态变更”分开。
- 能直接取 `row.xxx`、`data.xxx` 的场景，不要额外套自执行函数 `(()=>{})()`。
- 禁止为了省步骤，把长函数表达式堆进一个字段里；配置可读性优先，保持短、平、直。

最小验收：

- 页面路由可打开。
- 页面 title 或关键文本可见。
- 依赖接口能成功返回。
