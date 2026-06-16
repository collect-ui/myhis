# Webshell Workspace Project

## Backend Constraints
- Table: `webshell_workspace_project`
- Soft delete field: `is_delete` (`0` normal, `1` deleted)
- Global unique field: `project_code`
- Service mount path: `collect/webshell/service.yml` -> `workspace_project/index.yml`
- Query/count SQL pair:
  - `collect/webshell/workspace_project/workspace_project_list.sql`
  - `collect/webshell/workspace_project/workspace_project_count.sql`

## Frontend Key Model Rules
- Use one mode only in a chain:
  - Mode A: options are `{label, value, path}`; downstream reads `row.value`
  - Mode B: options are raw rows `{project_code, project_name}`; downstream reads `row.project_code`
- Never mix `row.value` and `row.project_code` in one chain.
- Keep alignment:
  - `workspaceCurrent = row.project_code` (or one consistent equivalent)
  - `workspaceLoadedMap` keys match tabs/list `keyField`
  - `activeKey` matches current workspace key

## Tabs Panel State Retention
- Keep tab panel content inside tabs (`items[].children` or `itemData/itemAttr` children).
- Do not place tab panels outside tabs and use external `visible` to mimic switching.
- Reason: external `visible` switching can destroy/recreate panel nodes and drop internal runtime state.

### Bad Pattern (state loss risk)
- `tabs` only renders tab headers.
- Real panel content is outside tabs.
- Source/HTTP panel toggles through outer `visible`.

### Good Pattern (state retained)
- `tabs.items` contains both Source and HTTP panel content.
- `activeKey` is bound to store field and changed by tabs internal handler.
- Panel-local state stays with tab content lifecycle.

## Tabs Review Checklist
1. Panel content is inside tabs children, not outside.
2. No external `visible` is used to simulate tab switching.
3. After switching tabs, form values/editor content/tree expand state still exist as expected.

## Reload Group Isolation
- `reload-workspace-project`: top drawer workspace switch
- `reload-workspace-project-manage`: project management dialog list
- Do not call one group from the other's action chain unless there is an explicit bridge step and no overwrite risk.

## Loop Prevention Checklist
1. Ensure `reload-init-action` is not bound to high-frequency form/store updates.
2. Ensure one reload chain does not update stores that immediately retrigger itself.
3. Ensure list/tabs keys are aligned (`keyField`, `activeKey`, loaded map keys).
4. Ensure adapter output shape matches all downstream expressions.

## Sync Button Regression Guard
- Requirement: click "sync" should only execute sync action **after** confirmation.
- Common mistake: action chain starts network/action call on button click event, and confirm dialog only wraps a later step.
- Safe pattern:
1. Button click opens confirm dialog only.
2. Confirm "OK" callback triggers the actual sync action group.
3. Cancel path performs no side effects.
- Validation checklist:
1. Click sync and then cancel: no request, no state change.
2. Click sync and confirm: exactly one sync action chain executes.
3. No preload/reload action is bound directly to sync button click event.

## Save/Close/Reload Ordering
- If UX requires instant close on success, use order:
1. submit/ajax success
2. close dialog (`update-store`)
3. reload list

## Form Submit and Query Parameters
- Prefer `appendFormFields` for save/query forms.
- Use `appendFields` for non-form fields (for example pagination).
- Avoid syncing form -> store -> ajax when direct append is possible.

## Minimal Regression Checklist
1. JSON syntax valid.
2. Workspace switch changes to target `project_code`.
3. `tabs.activeKey` equals current workspace key.
4. Loaded map key and tab key field are aligned.
5. Open project management triggers one query per explicit user action.
6. Add/edit/delete refreshes list as expected.
7. Delete has confirmation.
8. Sync action only runs after confirmation.
