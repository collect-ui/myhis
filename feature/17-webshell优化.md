测试要求：

- 使用无头浏览器打开目标页面，按用户真实路径完成操作。
- 记录 console error、pageerror、requestfailed。
- 保存 JSON 报告和关键截图。
- 失败时先根据截图和 DOM 证据修复，再重复验证直到通过。
# 环境
为了方便我看状态运行停止状态
tab 上加个loading
地址：http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell
# 需求
- tabs 增加一个 loading 增加一个状态 绿色loading
- 如果正在运行有输出，那么是绿色转一转的那个
- 如果停止了，就是灰色
# 要求
- 用低代码
# 重现步骤
- 打开网址 http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell
- 找到服务器 192.168.232.130,可以在目录里面搜索，也可以在点虚拟机这个节点，双击 192.168.232.130 进行登录
测试要求：

- 使用无头浏览器打开目标页面，按用户真实路径完成操作。
- 记录 console error、pageerror、requestfailed。
- 保存 JSON 报告和关键截图。
- 失败时先根据截图和 DOM 证据修复，再重复验证直到通过。