# if_template

## 作用

- `if_template` 是循环项条件模板，用于决定当前项是否参与处理。

## 常见用途

- 在 `filter_arr`、类似过滤处理器里按条件保留/剔除项。

## 执行阶段（低代码视角）

- 数组循环处理中：每项先算条件，再决定是否进入结果集。

## 怎么用

### 配置位置

- `handler_params[].if_template`
- `result_handler[].if_template`

## 示例

```yml
handler_params:
  - key: filter_arr
    foreach: "[mail_account_list]"
    item: item
    if_template: '{{eq (printf "%v" .item.exists) "1"}}'
    save_field: skip_list
```

来源文件：`/data/project/sport/collect/system/mail_account/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`if_template`
