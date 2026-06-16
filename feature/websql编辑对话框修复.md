# 现状 
路径http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell 
websql 有个问题，就新增连接和编辑连接时候，对话框浮动在drawer 后面去了，根本点不到
# 复现步骤
- 点开工作空间
- 展示sql
- 编辑websql 连接，或者点开新增 对话框在后面
- 能否换成dropdown 的形式，直接在下方浮动出来，不要弹框了，弹框的形式我不太喜欢
- 用低代码解决
- websql 模块，确定是底代码解决的，我怎么感觉对话框都是自己写的，按到道理都能用低代码解决
# 测试要求
- 用无头浏览器回归一下原始页面 http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool 能否点弹框
- 在验证在webshell 里面是否正常http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell 
- 反复验证，调整、测试、验证调整测试、直到全部通过

