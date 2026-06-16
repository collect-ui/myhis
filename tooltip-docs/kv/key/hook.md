# key=hook

## 作用

- 把当前结果异步传给另一个服务执行（线程触发），当前主链路不等待完成。
- 适合做旁路通知、缓存刷新、审计写入等“非阻塞后处理”。

## 常见用途

- 把当前结果异步传给另一个服务执行（线程触发），当前主链路不等待完成。
- 适合做旁路通知、缓存刷新、审计写入等“非阻塞后处理”。

## 执行阶段（低代码视角）

- 结果处理阶段：在模块执行后运行，用于重组输出或联动后续动作。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| params.service | object | 是 | 要异步触发的服务配置 |
| params.result_name | string | 否 | 注入结果字段名，默认 `_` |

## 示例

```yml
result_handler:
  - key: hook
    params:
      result_name: _
      service:
        service: message.reinitalertcache
```

## 注意事项

- 主流程立即返回原结果，不等待 hook 服务结束。
- hook 失败会写错误日志，但不回滚主流程。

### 源码定位
- Python类路径：`collect.service_imp.result_handlers.handlers.hook.Hook`
- 本次核对源码：`/tmp/collect-wheel-0.0.86/collect/service_imp/result_handlers/handlers/hook.py`

## 元信息

- 来源：`result_handler -> hook / 异步触发服务`
- 页面标题：`hook(异步触发服务)`
