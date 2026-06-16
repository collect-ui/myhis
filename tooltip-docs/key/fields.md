# fields

## 作用

- `fields` 是字段规则数组，用于批量描述“哪些字段按什么规则处理”。

## 常见用途

- `update_array` 批量改字段、`check_field` 批量校验、映射/回填规则配置。

## 执行阶段（低代码视角）

- 参数处理或结果处理阶段：处理器逐条遍历 `fields` 规则执行。

## 怎么用

### 配置位置

- 处理器节点下（如 `update_array.fields`、`check_field.fields`、`result2params.fields`）

## 示例

```yml
    name: 将文件路径转换成文件内容
    method: LoadDataFile
    fields:
      - from: DataFile
        to: FileData
        name: 将data_file 转换成文件数据
```

来源文件：`/data/project/sport/collect/service_router.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`fields`
