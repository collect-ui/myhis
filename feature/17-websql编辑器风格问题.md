# 现状
目前websql 是黑色、深色的风格
http://192.168.232.130:8015/collect-ui#/collect-ui/framework/websql-pool
实际最开始我是要求浅色风格的，但是
底层都monaco ,似乎底层用同一套实例风格一样，我先打开websql 开始是白色的
然后在打开editor 工作台
http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool
点开文件 会变成黑色
然后我再点开 websql 又会变成白色，并且它会把之前的文件editor 改成白色的
# 要求
- websql editor  和编辑器的editor 2个不影响
- websql editor 是亮色、白色系
- 文本编辑器editor 是黑色系
- 2个分别新增文本，不相互影响
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