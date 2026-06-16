# 接口对接与 Mock 切换

## 1. 推荐演进路径

1. 先用 pane 局部 `workspaceDataMap` 跑通交互
2. 再把“加载 tree/fileMap”替换成 `ajax`
3. 保留 mock 作为失败兜底（可选）

## 2. 参数规范

- 入参统一：`project_id`
- 前端来源：`row.path`（或回退 `row.value`）

## 3. 返回结构规范

```json
{
  "tree": [],
  "fileMap": {
    "/xx/main.go": {
      "language": "go",
      "content": "..."
    }
  }
}
```

## 4. adapt 示例

- `workspaceTree: ${data.tree || []}`
- `workspaceFileMap: ${data.fileMap || {}}`

## 5. 防空策略

- 后端无数据时返回空数组/空对象，不返回 null
- 前端写入前做数组/对象兜底
