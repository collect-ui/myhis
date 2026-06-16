# item

## 作用

- `item` 定义循环时“当前项”的变量名，供模板和字段规则引用。

## 常见用途

- `update_array`、`filter_arr`、`check_array` 等循环处理器。

## 执行阶段（低代码视角）

- 循环执行阶段：每次迭代把当前元素绑定到该变量名。

## 怎么用

### 配置位置

- `*.item`（与 `foreach` 成对出现）

## 示例

```yml
      - key: update_array
        foreach: "[service_list]"
        item: item
        fields:
          - field: field
            template: "[item.service]"
```

来源文件：`/data/project/sport/collect/work_task/bulk/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`item`
