# tooltip-docs 使用说明

这个目录用于维护低代码 YAML 的悬停说明文档。

## 目录规范

- `key/<key>.md`：某个配置键的说明（例如 `key/module.md`）。
- `kv/<key>/<value>.md`：某个键在特定值下的说明（例如 `kv/module/mysql.md`）。
- `value/<value>.md`：通用值说明（可选）。

编辑器悬停时会按下面优先级查找（同一个项目内）：

1. `<projectType>/section/<section>/kv/<key>/<value>.md`
2. `<projectType>/kv/<key>/<value>.md`
3. `<projectType>/key/<key>.md`
4. `<projectType>/value/<value>.md`
5. `section/<section>/kv/<key>/<value>.md`
6. `kv/<key>/<value>.md`
7. `key/<key>.md`
8. `value/<value>.md`

其中 `projectType` 一般为 `python` 或 `go`，`section` 可能为 `handler_params` / `result_handler` 等。

## 推荐写法

每个文档建议保持以下结构，方便悬停摘要：

```md
# 标题

一句话说明这个关键字（或 key+value）的作用。

## 配置位置
- 常见在哪个区块出现

## 示例
```yml
...示例...
```

## 注意事项
- 常见坑
```
