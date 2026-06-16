# module

## 作用

- params->handler_params->module->result_handler
- 服务可以操作的部分目前是四个， 第一步参数定义 第二步参数处理 第三步模块执行 第四步结果处理 四步中途如果有一个错误则直接返回 请求拦截和定期任务没有定义生命周期
- 它属于更上一层的包装，而且基本不怎么操作
- 这些模块只有module 是必须的

## 常见用途

- params->handler_params->module->result_handler
- 服务可以操作的部分目前是四个， 第一步参数定义 第二步参数处理 第三步模块执行 第四步结果处理 四步中途如果有一个错误则直接返回 请求拦截和定期任务没有定义生命周期

## 执行阶段（低代码视角）

- 模块执行阶段：服务主体逻辑所在阶段。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| params | json |  | 参数定义，http请求参数+内部定义 |
| handler_params | array |  | 参数处理。处理参数、增加修改参数 |
| module | string | 是 | 运行的模块 |
| result_handler | array |  | 和参数处理是一样，所以result_handler的文档是参数处理是一致的 |

## 示例

```yml
service:
  - key: demo_query
    module: sql
    params:
      page:
        default: 1
    handler_params: []
    result_handler: []
```

## 注意事项

- 这些模块只有module 是必须的

## 元信息

- 来源：`服务文档 -> 如何编写服务 -> 4.生命周期 / 服务描述`
- 页面标题：`生命周期(服务描述)`
