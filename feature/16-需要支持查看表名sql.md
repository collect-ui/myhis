# 现状
目前支持执行sql ，但是有时候我查看sql 需要直接能看看表结构
# 示例sql
SELECT 
    a.project_code, 
    a.project_name
FROM sys_projects a 
WHERE IFNULL(a.flag_del, '0') = '0'
# 地址
http://192.168.232.130:8015/collect-ui#/collect-ui/framework/websql-pool

# 要求
- 当鼠标移动sys_projects 右键查看表字段、ddl
测试要求：
- 目前每次点一个mysql 出现一个，show tables ,这个不要,建立一个空查询即可


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