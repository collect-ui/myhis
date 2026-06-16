# 地址
http://192.168.232.130:8015/collect-ui#/collect-ui/framework/agent_regression
# 需求
我希望agent 分屏模式和webshell 分屏模式是一致
http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell
# 要求
- 顶部还行需要框架的，目前没有框架
- 类型webshell 支持tab 标签页，左右分屏，右键点击分屏，而不是固定2个分屏，原型是示例一下
- 上下分屏也是一样的
- 左侧历史目录，请仔细参考原型，按照原型1：1还原
- 原型在这里/data/project/sport/feature/原型设计/agent/code.html

## 测试：无头浏览器

验证要求：

1. 打开真实 URL。
2. 等待页面资源加载完成。
3. 按用户路径点击、输入、保存。
4. 监听 console error、pageerror、requestfailed。
5. 保存 JSON 报告和关键截图。

断言内容：

- 页面可打开。
- 目标控件可见。
- 操作结果正确。
- 数据保存后可回读。
- 无前端错误和失败请求。