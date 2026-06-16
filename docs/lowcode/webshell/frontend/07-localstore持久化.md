# LocalStore 持久化

## 建议持久化字段

- `workspaceDrawerMode`
- `workspaceDrawerWidth`
- `workspaceVisible`

## 读写时机

- 页面初始化：`localstore.read`
- 点击按钮/关闭 drawer：`localstore.write`

## 键名建议

- `workspace-drawer-mode`
- `workspace-drawer-width`
- `workspace-visible`
