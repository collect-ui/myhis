# err_msg

## 作用

- `err_msg` 定义校验失败时返回给调用方的错误信息，支持模板渲染。

## 常见用途

- 参数必填校验失败提示、数组逐项校验定位、业务规则失败说明。

## 执行阶段（低代码视角）

- 参数校验与处理器校验阶段：当对应 `template/check` 返回 false 时返回。

## 怎么用

### 配置位置

- `params.*.check.err_msg`
- `handler_params[].err_msg`
- `result_handler[].err_msg`

## 示例

```yml
params:
  work_task_version_id:
    check:
      template: "{{must .work_task_version_id}}"
      err_msg: 记录不能为空
```

来源文件：`/data/project/sport/collect/work_task/version/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`err_msg`
