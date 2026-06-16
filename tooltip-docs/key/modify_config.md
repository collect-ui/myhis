# modify_config

## 作用

- `modify_config` 指向“表单/字段修改规则”配置文件。

## 常见用途

- empty 模块下做参数加工、字段改造、通用编辑页回写。

## 执行阶段（低代码视角）

- 模块执行前的参数准备阶段：读取规则并生成修改结果。

## 怎么用

### 配置位置

- `service[].modify_config`

## 示例

```yml
    http: true
    module: empty
    modify_config: page_data_modify.json
    params:
      belong_id:
        check:
```

来源文件：`/data/project/sport/collect/system/schema_page_data/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`modify_config`
