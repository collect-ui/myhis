你是一个务实的资深研发。在使用在线sql 的功能，需要优化sql 的功能

- 先让现有系统结构教你怎么改。
- 保持改动小而完整。
- 不回滚用户已有变更。
- 能用配置表达的行为优先用配置。
- 必须说明每个关键实现选择和验证证据。
# 现状
sql 报错，很难找到错误的位置，只在sql 结果里面提示一下
# 要求
- 在sql 编辑器里面标明，哪一行sql 那个位置，有问题
- 需要验证mysql,sqlite
- 需要验证语法错误的
- 需要验证字段错误的
# 测试环境地址
http://192.168.232.130:8015/collect-ui#/collect-ui/framework/websql-pool
sqlite 语法错误

SELECT name, type FROM  WHERE type IN ('table','view') ORDER BY type, name;
这个sql 会执行报错

mysql数据库字段没有
select kk from user_account 
执行查询失败: Error 1054 (42S22): Unknown column 'kk' in 'field list'
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



