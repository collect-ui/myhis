# 现状
目前 http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell
左右分屏的时候和上下分屏的时候，回会出现路径计算不正确的情况
# 重现步骤
- 打开网址 http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell
- 找到服务器 192.168.232.130,可以在目录里面搜索，也可以在点虚拟机这个节点，双击 192.168.232.130 进行登录
- cd 然后在终端cd /data 目录，验证左右分屏，和上下分屏，是不是都是进入data 目录
- 然后每一个tab 标签页上右键，点击文件管理，看看是否进入 /data ，有时候计算错误就进入就进入
/home/zz
# 源码位置
/data/project/sport-ui/src/components/ssh.tsx
注意我计算位置是截取websocket 的返回来的
# 测试要求
- 用无头浏览器测试，登录192.168.232.130
- 进入/data 目录
- 试试左右分屏和上下分屏，确保分屏进入目录都是data
- 然后每一个tab 都分别进入不同目录，比如/data,/home ,/data/project ，然后都在tab上进入文件管理
需要分别能进入不同的目录
- 反复验证、反复测试、多轮压力测试和调整

