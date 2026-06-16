# 开发配置踩坑记录

本文档记录在本项目开发过程中遇到的非显而易见的配置陷阱和解决方案，供后续开发参考。

---

## 1. 前端构建体系：三项目依赖链

本项目的前端不是直接从 `collect-ui` 构建的，而是经过一条依赖链：

```
collect-ui (组件库，symlink)
    ↓ npm 依赖 (symlink)
sport-ui (应用壳，注册自定义组件/动作)
    ↓ vite build
his/frontend/collect-ui/ (最终部署的静态资源)
```

**关键结论**：
- 修改 `collect-ui/src/` 后，需要在 `sport-ui/` 目录执行 `NODE_OPTIONS="--max-old-space-size=8192" npx vite build` 重新构建
- 构建产物在 `sport-ui/build/`，需手动复制到 `his/frontend/collect-ui/`
- **绝不能**用 `collect-ui` 的 `demo:build`（`npm run demo:build`）替代——那是组件文档站点，不是应用

**错误示范**：误将 `collect-ui/docs/`（demo 构建产物）复制到 `his/frontend/collect-ui/`，导致页面变成组件文档中心。

---

## 2. collect-ui 框架的 form 与 store 同步问题

### 问题描述

`collect-ui` 的表单组件（`form.tsx`）在提交时 **不会** 将表单值写回 MobX store。这导致：

- `appendFields: "${searchForm}"` 从 store 读取的值是过时的（只有初始化时的值）
- `appendFormFields: "searchForm"` 从 Ant Design form 实例读取，也不包含分页组件管理的 `page`/`size`

### 影响范围

所有使用 `submitOnChange: true` + `reload-init-action` 的搜索表单，配合分页组件时都会遇到此问题。

### 已应用的修复

在 `collect-ui/src/components/form/form.tsx` 的 `onFinish` 中添加了 store 同步逻辑：

```typescript
// 将表单值同步回 store，确保 appendFields 能读到最新值
// 同时保留 store 中的独有字段（如分页组件管理的 page/size）
if (props.name && store && typeof store.setValue === 'function' && typeof store.getValue === 'function') {
  const storeValue = store.getValue(props.name)
  if (storeValue && typeof storeValue === 'object') {
    const merged = { ...storeValue }
    for (const key in values) {
      if (values[key] !== undefined) {
        merged[key] = values[key]
      }
    }
    store.setValue(props.name, merged)
  } else {
    store.setValue(props.name, values)
  }
}
```

### 正确的分页配置模式

修复后，`initAction` 只需 `appendFields`（不再需要 `appendFormFields`）：

```json
{
  "tag": "ajax",
  "group": "loadDataList",
  "api": "post:/template_data/data?service=xxx_query",
  "appendFields": "${searchForm}",
  "adapt": { "dataList": "${data}", "count": "${count}" }
}
```

分页组件自动更新 store 中的 `searchForm.page` / `searchForm.size`，`appendFields` 读取合并后的完整数据。

---

## 3. collect-ui 的 appendFields vs appendFormFields 执行顺序

在 `ajax.tsx` 中，字段合并的顺序是：

1. `appendFields`（从 MobX store 读取）
2. `appendFormFields`（从 Ant Design form 实例读取，**覆盖**同名 key）
3. `api.data`（静态数据）
4. `data`（动态表达式）

**陷阱**：当两者同时引用同一个 store key（如 `searchForm`）时，`appendFormFields` 会用 form 的过时值覆盖 store 中的正确值。

**解决方案**：二选一，不要同时使用。分页场景用 `appendFields`，纯表单提交用 `appendFormFields`。

---

## 4. Oracle 数据库字段名大小写

### 问题

Oracle 的 `SELECT *` 返回大写字段名（`MASTER_ID`, `AREA_CODE`），而前端表单字段名是小写（`master_id`, `area_code`）。

### 解决方案

SQL 查询中必须用双引号别名强制小写：

```sql
SELECT a.master_id as "master_id",
       a.area_code as "area_code",
       ...
FROM pix_outp_reg_master a
```

否则后端返回的 JSON key 是大写，与前端 form 字段名不匹配，导致编辑回显失败。

---

## 5. Oracle 绑定变量语法

go-ora 驱动在 `Conn.Query/Exec` 直接模式下不自动将 `?` 转换为 `:1, :2, ...`。

已在 `collect/src/collect/service_imp/module_sql_service.go` 中添加 `convertOracleBindVars()` 函数，在执行前自动转换。

---

## 6. collect 框架的 Oracle GORM 支持

`collect/src/collect/service_imp/base_handler.go` 的 `GetGormDb()` 原来只支持 `mysql` 和 `sqlite3`。已添加 `oracle` 分支，使用 `godoes/gorm-oracle` dialect。

---

## 7. 菜单数据来源

前端菜单 **不是** 来自数据库，而是来自 mock 文件：

```
collect/system/menu/mock/menu_tree.json
```

新增页面菜单需要同时修改此文件和 `collect/frontend/page_data/data/menu/menu.json`。

---

## 8. collect 框架的双重 JSON 响应

`template_data/data` 端点返回两个串联的 JSON 对象（第一个通常是错误信息，第二个是实际结果）。

解析时需要用 `json.JSONDecoder().raw_decode()` 循环解析，不能用 `response.json()`。

---

## 9. collect 框架 multipart/form-data 的 service 参数

`HandlerRequest` 要求 `service` 参数必须在 form body 中（multipart/form-data），不能只放在 URL query string。

正确做法：将 `service=xxx` 放在 POST body 中，不要只放在 URL 上。

---

## 10. Node.js 构建内存不足

sport-ui 项目较大，Vite 构建时默认内存不够。必须设置：

```bash
NODE_OPTIONS="--max-old-space-size=8192" npx vite build
```

---

## 11. collect-ui 组件名称约定

| 组件 | 正确名称 | 常见错误 |
|------|----------|----------|
| 表格列渲染 | `cellRenderer` | `cellRender` |
| 分页属性 | `pagination: true`（boolean） | `pagination: {}`（对象） |
| 分页组件 | 放在 `bottomToolBar` 中 | 不要嵌套在 table 内 |

---

## 12. layout-fit 顶层属性

`layout-fit` 的合法顶层属性包括：

- `searchToolBar` — 搜索栏
- `topRight` — 右上角按钮区
- `children` — 主内容区（table、dialog 等）
- `bottomToolBar` — 底部分页栏
- `dialog` — 弹窗

不要手动嵌套 flex div 来实现布局，框架已内置。

---

## 13. MobX observable 数组导致 select options 不刷新

### 问题描述

当通过 `adapt` 动态更新 store 中的数组（如 `specialClinicOptions`）时，Ant Design Select 的 `options` 绑定 `${specialClinicOptions}` **不会重新渲染**。下拉列表仍然显示旧数据。

### 根因

MobX observable 数组在 `store.setValue()` 后引用地址不变，React 的浅比较认为 props 没有变化，跳过 re-render。

### 解决方案

**必须**使用展开运算符创建新数组引用：

```json
// 错误 — MobX observable 数组引用不变，Select 不刷新
"options": "${specialClinicOptions}"

// 正确 — 展开运算符产生新数组，触发 Select 重新渲染
"options": "${[...specialClinicOptions]}"
```

### 影响范围

所有通过级联（cascade）动态更新的 select `options` 都需要此写法，包括但不限于：
- 门诊专科 `specialClinicOptions`
- 医生 `doctorOptions`

仅在 `initAction` 中加载一次且不随级联变化的选项（如 `areaOptions`、`durationOptions`）可以不加。

---

## 14. Radio 组件自定义 onChange 覆盖 Ant Design 表单上下文

### 问题描述

Radio 组件（`radio.tsx`）的自定义 `onChange` 处理函数通过 `onChange={onChange}` 传递给 `Radio.Group`，**覆盖**了 Ant Design 通过 React context 注入的默认 onChange。导致点击第二个选项后无法切换回第一个选项（radio 卡住）。

### 根因

1. Ant Design v5 的 `Form.Item` 通过 React context（而非 `cloneElement` props）向子组件注入 `value` 和 `onChange`
2. 因此 `props.onChange` 始终为 `undefined`，自定义 onChange 调用的 `props.onChange(event)` 无效
3. 但 `onChange={onChange}` 仍然覆盖了 `Radio.Group` 的 context onChange，导致表单收不到值变更通知
4. 首次点击有效是因为 `Radio.Group` 的内部视觉状态更新了，后续切换时表单值未被真正更新

### 解决方案

**不要**向 `Radio.Group` 传递自定义 `onChange`，让 Ant Design 的 context 机制处理：

```tsx
// 错误 — 自定义 onChange 覆盖 context onChange
<Radio.Group {...newProps} onChange={onChange}>
  {options.map(...)}
</Radio.Group>

// 正确 — 移除 onChange，让 Ant Design 接管
<Radio.Group {...newProps}>
  {options.map(...)}
</Radio.Group>
```

### 影响范围

表单内的所有 `Radio` / `Radio.Group` 组件。

---

## 15. dialog 的 action 时机

`dialog.action` 在对话框 **关闭时** 触发。保存操作应放在 `footer` 按钮的 `action` 中，不要放在 `dialog.action` 里。
