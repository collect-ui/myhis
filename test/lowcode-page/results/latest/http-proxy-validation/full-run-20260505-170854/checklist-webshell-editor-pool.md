# Webshell Editor Pool 全流程测试清单

- 测试页面: http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool
- 执行时间: 2026-05-05
- 结果目录: /data/project/sport/test/lowcode-page/results/latest/http-proxy-validation/full-run-20260505-170854

## 清单与结果

| 序号 | 测试项 | 结果 | 证据 |
|---|---|---|---|
| 1 | 页面可达、HTTP树可见、可选中HTTP节点 | 通过 | custom-console-flow-check-v3.json, custom-console-flow-check-v3.png |
| 2 | HTTP控制台可打开、请求表单控件可用（模式/方法/URL/发送） | 通过 | custom-console-flow-check-v3.json |
| 3 | 控制台前台直发请求（/template_data/data） | 通过 | custom-console-flow-check-v3.json |
| 4 | 控制台后端代发请求（postman-echo） | 通过 | custom-console-flow-check-v3.json |
| 5 | 控制台保存为接口并落库（Header/Body/模式/方法/URL） | 通过 | webshell-editor-pool-console-save-to-doc-check.json |
| 6 | HTTP新增接口弹窗UI（布局对齐/请求头块） | 通过 | webshell-editor-pool-http-doc-dialog-ui-check.json |
| 7 | HTTP目录与接口全流程CRUD（分组+文档+发送+清理） | 通过 | webshell-editor-pool-http-full-flow-check.json |
| 8 | 请求模式全链路（frontend/backend 展示、发送、DB字段） | 通过 | webshell-editor-pool-http-mode-flow-check.json |
| 9 | 项目隔离（backend 与 collect-ui 相互隔离） | 通过 | webshell-editor-pool-http-project-isolation-check.json |
| 10 | test2 登录链路（登录 -> Cookie -> 获取用户） | 通过 | webshell-editor-pool-http-test2-login-chain-check.json |
| 11 | 内容搜索（弹窗、结果、点击命中打开文件） | 通过 | custom-content-search-check-v2.json, custom-content-search-check-v2.png |
| 12 | 控制台“测试数据存档”控件可见（查询/新增/更新/删除） | 失败 | custom-console-store-presence-check.json, custom-console-store-presence-check.png |

## 说明

- 首轮脚本中有 6 项失败是旧选择器导致（依赖 `HTTP目录` 文本或旧 class），已通过补测脚本覆盖关键链路。
- 当前页面已无 `.workspace-http-console-store-select` 与 `.workspace-http-console-store-btn`，仅显示“常用HTTP请求/占无数据”，因此“测试数据存档CRUD”项判定失败。
