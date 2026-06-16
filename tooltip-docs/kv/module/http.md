# module=http

## 作用

- 我们后台经常需要发http请求到其他服务器，比如获取企业微信用户信息，调用集成第三方服务的接口
- 前台肯定是不能直接调用，需要后台调用，处理完成之后，前台再处理
- 其主要参考思路是ajax 配置化发送http请求，一般语言像java、python、甚至go对http配置发请求能力还是比较弱，一般封装一个工具类
- 注意module: http 表示是本服务是给其他服务器发请求，而配置http：true 表示改服务对外暴露http接口，允许外部调用，否则只能内部调用

## 常见用途

- 我们后台经常需要发http请求到其他服务器，比如获取企业微信用户信息，调用集成第三方服务的接口
- 前台肯定是不能直接调用，需要后台调用，处理完成之后，前台再处理

## 执行阶段（低代码视角）

- 模块执行阶段：在 `handler_params` 完成后执行，是服务主体能力；执行结束后进入 `result_handler`。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| module | string | 是 | http |
| http_json | json | 是 | http配置文件路径，整个文件对象模板渲染 |
| success | template |  | 判断结果是否成功，变量直接返回数据的json字段。true 表示成功 |
| http_json.url | string | 是 | 请求地址 |
| http_json.method | string |  | 请求方法，GET,POST |
| http_json.result_json | string |  | 结果转json对象 |
| http_json.header | json |  | 请求头 |
| http_json.data | json |  | 请求参数，支持模板转义 |
| http_json.basic_auth | json |  | basic账号密码认证 |
| http_json.basic_auth.username | string |  | 登陆账号 |
| http_json.basic_auth.password | string |  | 登陆密码 |

## 示例

```yml
- key: gettoken
  module: http
  http_json: gettoken.json
  success: "{{ if .access_token }}true{{ else }}false{{ end }}"
  result_handler:
    - key: result2params
      fields:
        - from: "[access_token]"
          to: "[access_token]"
    - key: param2result
      field: "[access_token]"
```

## 注意事项

- 后台发http请求
- 支持require另外一个公共文件
- 模板渲染生成http配置

## 元信息

- 来源：`服务文档 -> 模块处理 -> 9.http / http请求`
- 页面标题：`http(http请求)`
