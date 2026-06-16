# 06 Agent 测试回归页面

本页记录 `Agent测试回归` 低代码页面的落地位置，便于直接联调。

## 页面文件

- `collect/frontend/page_data/data/system/agent_regression.json`

页面能力：

- 创建/刷新 `agent session`
- 同步发送真实消息
- 刷新消息记录和运行记录
- 显示最近一轮助手回复

## 删除会话按钮写法

列表行内的删除确认，直接使用按钮自身的 `confirm` 配置，不要额外包一层 `tag: confirm` 再在 `onOk` 里发请求。

删除后的列表刷新也保持简单：

- `agent.session_query` 的 `adapt` 优先只写 `sessionList: "${data}"`。
- 删除后如果需要切换当前选中项、清空消息区、回填表单，使用后续独立的 `update-store`、`update-form` 处理。
- 禁止把 `sessionInfo`、`pageForm`、`activeSessionId` 这些状态在一个 `ajax.adapt` 里通过长函数一起重算。
- 禁止在 `ajax.adapt` 里复制其他对象并混入旧 store，这种写法可读性差，也容易把行对象、运行时对象带进 store。

推荐写法：

```json
{
  "tag": "button",
  "type": "text",
  "danger": true,
  "children": "删除",
  "confirm": {
    "title": "删除会话",
    "description": "${'确认删除会话【'+(row.title||row.session_key||'未命名对话')+'】吗？'}"
  },
  "action": [
    {
      "tag": "ajax",
      "api": "post:/template_data/data?service=agent.session_delete",
      "data": {
        "agent_session_id": "${row.agent_session_id||''}"
      }
    }
  ]
}
```

这里删除请求的参数直接取当前行对象 `row` 即可，不要先把 `row` 写进 store 再从 store 取值。

原因：

- 列表运行时的 `row` 往往不是纯 JSON 对象，里面可能带有 `_parentStore` 等运行时引用。
- 如果把整条 `row` 写入 store，框架在序列化 store 时可能触发循环引用报错。
- 当前场景只需要当前行字段时，直接在按钮 `action` 中读取 `row.xxx` 更简单，也更稳定。
- 同理，删除后的查询结果如果只是列表刷新，直接 `sessionList: "${data}"` 就够了，不要把状态切换逻辑硬塞进 `adapt`。

## frontend service

已在 `collect/frontend/page_data/index.yml` 注册：

- `frontend.agent_regression`

## 菜单 seed

可重复执行 SQL：

- `sql/release/1.0.0/agent_regression_menu.sql`

当前 seed 配置：

- `menu_name=Agent测试回归`
- `menu_code=agent_regression`
- `router_group=framework`
- `parent_id=7c8b9620-db64-4586-97f6-a715c6d477b7`
- `api=post:/template_data/data?service=frontend.agent_regression`
- `url=/framework/agent_regression`
- `is_common=1`
- `belong_project=base`

## 本地验证

本地库 `database/price.db` 已插入菜单记录。

可验证项：

1. `system.menu_query` 能查到 `agent_regression`
2. `frontend.agent_regression` 能返回页面 schema
3. 默认测试会话 `agent-regression-demo` 可跑通真实消息

## 访问方式

菜单路径：

- `框架 -> 系统设置 -> Agent测试回归`

直接路由：

- `/framework/agent_regression`
