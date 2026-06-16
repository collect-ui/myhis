# from

## 作用

- `from` 表示映射来源字段，在字段拷贝/转换规则中使用。

## 常见用途

- `result2params`、`data_handler.fields`、字段搬运与重命名。

## 执行阶段（低代码视角）

- 数据映射阶段：读取来源字段后写入目标字段。

## 怎么用

### 配置位置

- `*.fields[].from`

## 示例

```yml
data_handler:
  - key: file_data_map
    fields:
      - from: ExcelConfig
        to: ExcelConfigContent
```

来源文件：`/data/project/sport/collect/service_router.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`from`
