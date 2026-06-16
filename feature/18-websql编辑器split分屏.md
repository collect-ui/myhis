# 
http://192.168.232.130:8015/collect-ui#/collect-ui/framework/websql-pool

websql 功能请参考webshell 能左右分屏 和上下分屏
# 参考webshell
http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell
shell 登录只有，在tab 标签页上右键操作，支持左右分屏和上下分屏
能有2个sql 编辑器一起工作
或者参考websql http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool
# 要求
- split 左右和上下，2个 一样sql面板
- 里面的内容是一样，能执行sql 查询结果
测试要求：

- 使用无头浏览器打开目标页面，按用户真实路径完成操作。
- 记录 console error、pageerror、requestfailed。
- 保存 JSON 报告和关键截图。
- 失败时先根据截图和 DOM 证据修复，再重复验证直到通过。
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