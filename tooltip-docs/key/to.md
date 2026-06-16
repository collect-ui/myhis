# to

## 作用

- `to` 表示映射目标字段，用于把来源值写入目标字段。

## 常见用途

- `result2params`、`data_handler.fields`、字段重命名与结构转换。

## 执行阶段（低代码视角）

- 数据映射阶段：来源值解析后写入该目标字段。

## 怎么用

### 配置位置

- `*.fields[].to`

## 示例

```yml
    fields:
      - from: DataFile
        to: FileData
        name: 将data_file 转换成文件数据
      - from: CountFile
        to: CountFileData
```

来源文件：`/data/project/sport/collect/service_router.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`to`
