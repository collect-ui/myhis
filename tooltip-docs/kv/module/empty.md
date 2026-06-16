# module=empty

## 作用

- 空模块是个非常常用的模块，其主要目的是为了处理参数
- 空模块就是主体没有做任何事情，主要在handler_params 运行你服务
- 比如要运行数据保存的服务，先调主体空服务，空服务在转service2field 调用你的数据保存服务，本质没有任何区别
- 可能就一些传参区别

## 常见用途

- 空模块是个非常常用的模块，其主要目的是为了处理参数
- 空模块就是主体没有做任何事情，主要在handler_params 运行你服务

## 执行阶段（低代码视角）

- 模块执行阶段：在 `handler_params` 完成后执行，是服务主体能力；执行结束后进入 `result_handler`。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| module | string | 是 | empty |

## 示例

```yml
- key: project_router
  http: true
  handler_params:
    - key: file2datajson
      save_field: data
    - key: param2result
      field: data
  data_file: project_router.json
  module: empty
```

## 注意事项

- 主要为了处理参数，一般搭配handler_params
- 在handler_params可以通过enable模板动态判断是否需要运行某个服务

## 元信息

- 来源：`服务文档 -> 模块处理 -> 8.empty / 空模块`
- 页面标题：`empty(空模块)`
