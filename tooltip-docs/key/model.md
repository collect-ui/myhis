# model

## 作用

- `model` 指定 ORM 模型名（常见于 Python 低代码 `model_update/model_save` 语义）。

## 常见用途

- 按模型更新/查询时通过模型名定位实体定义。

## 执行阶段（低代码视角）

- 模块执行阶段：根据模型/表配置构造数据库操作。

## 怎么用

### 配置位置

- 模块配置节点（与 `module: model_*` 搭配）

## 示例

```yml
      system_prompt:
        default: ""
      model:
        default: "gpt-5-mini"
      input_text:
        check:
```

来源文件：`/data/project/sport/collect/agent/run/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`model`
