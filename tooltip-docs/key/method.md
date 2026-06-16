# method

## 作用

- `method` 指向实现类的方法名（如 `handler`、`filter`），用于反射调用。

## 常见用途

- 在 `service_router.yml` 注册模块/处理器/模板函数时声明入口方法。

## 执行阶段（低代码视角）

- 引擎加载后，在执行分发时按该方法调用具体实现。

## 怎么用

### 配置位置

- 注册表节点（`module_handler/request_handler/result_handler/filter_handler`）

## 示例

```yml
  - key: load_data_file
    name: 将文件路径转换成文件内容
    method: LoadDataFile
    fields:
      - from: DataFile
        to: FileData
```

来源文件：`/data/project/sport/collect/service_router.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`method`
