# 现状
目前 http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell
ctrl+v 有效，但win+v 选择其他历史记录就粘贴不了
# 重现步骤

- 打开网址 http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell
- 找到服务器 192.168.232.130,可以在目录里面搜索，也可以在点虚拟机这个节点，双击 192.168.232.130 进行登录
- window 操作系统必须打开win+v 历史记录存储
- win+v  粘贴选择，选择完成粘贴没有效果
# 源码位置
/data/project/sport-ui/src/components/ssh.tsx
# 测试要求
- 无头浏览器验证
- 开启win+v 存储多个文本历史
- 验证win+v 能粘贴文本，选择文本
- webshell 能正确粘贴文本
