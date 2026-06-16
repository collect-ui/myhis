# 问题1 
 查看表下面的字段有问题
##复现步骤
- 选择sqlite 数据库
- 点开tables
- 点开 agent_message
- 然后就会发现所有表都没有了，只有agent_message 对立面表字段
# 问题2
右键查看ddl 有问题
## 复选步骤
- 选择sqlite 数据库
- 点开tables 
- 选择agent_message右键查看ddl 没有任何反应
## 猜测
store 取值不正确，
- 参考示例 http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool
- 请你看http 面板是怎么弹框的，弹出新增框和编辑
- 注意store 的取值控制变量形式
# 需求
- 表字段个图标，类似小熊一样
- 比如数字类型的用那种123 的图标，不要某INTEGER
- 比如字符串用字符串的图标，这样我一眼就就能看到
- 还有主键的用绿色小点
## 测试要求
- 用无头浏览器反复验证，点开修复上面的个bug
- 确保页面显示正常