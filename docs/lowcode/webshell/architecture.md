# 架构与数据流

## 整体链路

1. 页面 JSON 定义组件树（tabs/tree/panel-group/editor）
2. 用户动作触发 action（update-store/arr-push/method/localstore/ajax）
3. store 更新驱动渲染
4. 后台接口返回经 adapt 映射到页面状态字段

## 关键边界

- tabs 外层负责项目切换与懒加载
- pane 内 `layout-fit` 负责项目内运行态（tree/fileMap/panelList）
- backend 提供 tree/fileMap 或文件内容 API
