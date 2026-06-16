---
name: lowcode-json
description: Edit and debug low-code page JSON and related backend service YAML in this repository. Use when changing collect/frontend/page_data/**/*.json, collect/**/*.yml, service handler_params/modules, collect-ui actions/stores/forms/components, fixing webshell workspace interactions, or validating tags/services against /data/project/collect-ui docs, /data/project/sport/tooltip-docs, and runtime implementation.
---

# lowcode-json

## Outcome
Make minimal, schema-preserving low-code JSON/YAML changes that work across the collect-ui runtime and backend low-code service engine. Prefer documented low-code configuration over custom Go/TS/React code unless the requested behavior cannot be expressed with existing components, actions, stores, forms, services, modules, or handlers.

## Project Mental Model
- Page services are mounted by `collect/frontend/service.yml` and implemented in `collect/frontend/page_data/index.yml`; call them as `frontend.<key>`.
- Frontend `data_file` paths in `page_data/index.yml` are relative to `collect/frontend/page_data/`, so `data_file: data/server/webshell.json` means `collect/frontend/page_data/data/server/webshell.json`.
- Page JSON is often rendered, not static: `file2datajson` loads the JSON, `service2field` injects child fragments, `{{to_json .field}}` consumes them, and `param2result` returns the final page data.
- Business services are mounted by `collect/<domain>/service.yml` and exposed as `<domain>.<key>`; resolve an `ajax` service by opening the mounted child `index.yml`, then its SQL/model/table references.
- Component/action tags come from collect-ui by default. Sport-specific tags/actions such as `localstore`, `editor`, `workspace-editor-pool`, `ssh`, `handlerPath`, and `batch-shell` must be checked against `/data/project/sport-ui/src/main.tsx` and the matching source.

## First Moves
1. Locate the exact page/fragment under `collect/frontend/page_data/**/*.json` from the URL/service key, then find nearby working examples with similar UI or action chains.
2. Identify every component `tag` and action `tag` you will touch.
3. Read the matching collect-ui doc before editing:
   - Components: `/data/project/collect-ui/docs/readme/components/<tag>.md`
   - Actions: `/data/project/collect-ui/docs/readme/action/<tag>.md`
4. Resolve composition before judging a page:
   - `collect/frontend/page_data/index.yml` maps service keys to `data_file`.
   - `service2field` loads child fragments into template fields.
   - `{{to_json .fragment_name}}` placeholders mean the visible runtime page is composed from multiple JSON files.
5. For any `ajax` or service call (`/template_data/data?service=...`), resolve the backend service in `collect/**/service.yml` and child `index.yml` files.
6. Read backend tooltip docs before changing service YAML:
   - Service fields: `/data/project/sport/tooltip-docs/key/<field>.md`
   - Handler params: `/data/project/sport/tooltip-docs/kv/key/<handler>.md`
   - Module types: `/data/project/sport/tooltip-docs/kv/module/<module>.md`
7. If docs and behavior disagree, inspect implementation in `/data/project/collect-ui/src/components/`, `/data/project/collect-ui/src/action/`, `/data/project/collect-ui/src/index.tsx`, `/data/project/collect/src/collect/service_imp/`, and sport-ui registration/source as applicable.
8. Apply one small behavioral change at a time; avoid mass JSON/YAML reformatting.

## Edit Workflow
1. Map the data chain: `initStore` / `initStoreType` -> component props -> form values -> action params -> `adapt` / `update-store` -> reload target.
2. Keep one key model per interaction chain. Do not mix values such as `row.value` and `row.project_code` unless an explicit adapter converts between them.
3. Prefer existing action groups, forms, store keys, and service conventions in the same page/module.
4. Keep `appendFormFields` as the default form submit/query path. Use `appendFields` for extra non-form values such as pagination, selected row id, or fixed flags.
5. After edits, validate JSON syntax and then verify the affected user chain in order.

## Composition Rules
- Do not assume the file named by the route is the whole page. For example, `frontend.webshell` composes `webshell.json`, `webshell_ssh_fragment.json`, `webshell_editor_fragment.json`, and editor-pool fragments through `index.yml`.
- A fragment may be an object, array, or top-level placeholder target. Top-level `tag` can be absent when the fragment is injected into `items` or `children`.
- To inspect composed runtime JSON, call the page service the same way the frontend does: `POST /template_data/data?service=frontend.<key>` with JSON body `{"service":"frontend.<key>"}`. Plain GET can return 404 and is not authoritative.
- When editing placeholders, verify both sides: the producer `save_field` name in `index.yml` and the consumer `{{to_json .save_field}}` expression in JSON.
- Backend services compose similarly: top-level `collect/<module>/service.yml` mounts child `path: ".../index.yml"` files; the externally called service name is usually `<service.yml key>.<child index.yml key>`.

## Doc Lookup Map
Read only the docs needed for the touched tags. Common high-risk docs:

- Data requests: `action/ajax.md`, `action/reload-init-action.md`
- Store updates: `action/update-store.md`, `action/update-form.md`, `action/update-map.md`, `action/update-field.md`
- Flow guards and array mutation: `action/check.md`, `action/arr-push.md`, `action/method.md`
- Forms: `components/form.md`, `components/form-item.md`, `action/submit-form.md`, `action/reset-form.md`, `action/validate-form.md`
- Confirmation and destructive actions: `action/confirm.md`, `components/dialog.md`
- Data displays: `components/table.md`, `components/listview.md`, `components/pagination.md`
- Selection/editing widgets: `components/pull-down.md`, `components/input-table.md`, `components/select.md`, `components/input.md`
- Navigation/state containers: `components/tabs.md`, `components/router-tab.md`, `components/panel-group.md`, `components/panel.md`, `components/panel-resize.md`, `components/layout-fit.md`
- Project-specific runtime extensions: `/data/project/sport-ui/src/main.tsx`, then the registered component/action source.
- Backend service config: `/data/project/sport/tooltip-docs/key/`, `/data/project/sport/tooltip-docs/kv/key/`, `/data/project/sport/tooltip-docs/kv/module/`

Use the QA audit snapshots in those docs as an API coverage signal. If a field is marked undocumented or production-unseen, confirm in source before relying on it.

Useful demo/test fixtures for behavior comparison:
- Action coverage: `/data/project/collect-ui/demo/data/test/action-gap-demo.json`
- Tabs coverage: `/data/project/collect-ui/demo/data/test/tabs-api-gap-demo.json`
- Panel layout coverage: `/data/project/collect-ui/demo/data/test/panel-showcase.json`, `/data/project/collect-ui/demo/data/test/panel-demo.json`
- Input table coverage: `/data/project/collect-ui/demo/data/test/input-table-form-demo.json`, `/data/project/collect-ui/demo/data/test/input-table-standalone-demo.json`
- Pull-down coverage: `/data/project/collect-ui/demo/data/test/pull-down-api-gap-demo.json`

## Backend Service Rules
- Service lifecycle is `params` -> `handler_params` -> `module` -> `result_handler`. A failure in an earlier stage stops later stages.
- `params` is the shared parameter pool. HTTP request fields enter it first, then handlers/modules read and write it.
- `handler_params` runs before the module; `result_handler` runs after the module. Both use the same handler docs under `tooltip-docs/kv/key/`.
- `module` selects the backend executor. Check `tooltip-docs/kv/module/<module>.md` before changing fields like `data_file`, `count_file`, `table`, or model params.
- `module: empty` is commonly used for composed services where `handler_params` does most of the work, such as frontend page services that load JSON fragments.
- `module: sql` usually uses `data_file` and optionally `count_file`; verify SQL files beside the service `index.yml`.
- `file2datajson` reads and renders the page JSON file, then parses it; use it with `param2result` for frontend page services.
- `service2field` calls another service and writes its result into `save_field`; if `append_param` is true, existing params are passed through as well.
- `param2result` exposes a parameter field as the service response.
- For Go implementation details, use `/data/project/collect/src/collect/service_imp/` because this repo's `go.mod` replaces `github.com/collect-ui/collect => ../collect`.
- For backend behavior, verify both config and SQL/model/table references. Do not change local database files unless explicitly requested.

## Action Rules
- `ajax`
  - `api` should include HTTP method and URL, for example `post:/template_data/data?...`.
  - `adapt` writes response fields into store; check downstream names exactly.
  - `appendFormFields` has priority over `appendFields`; do not manually rebuild an entire form payload in `data`.
  - Use `group` for init/reload targets and keep group names unique per reload scope.
  - Use `start`/`end` for loading flags and `onSuccess`/`onError` for follow-up chains when needed.
- `reload-init-action`
  - The `group` must match an `initAction` group exactly.
  - Do not bind reload actions to high-frequency form/store changes.
  - Keep reload groups isolated when two lists/dialogs use different data scopes.
- `update-store`
  - Use `value` for store updates; dot paths are acceptable for nested fields when the surrounding page already uses that style.
  - Avoid update loops where a store write immediately retriggers the same reload or submit chain.
- `check`
  - Use as a fail-fast guard before side effects.
  - `check` must evaluate to truthy; `title` is the error message when it fails.
- `arr-push`
  - `from` must resolve to an existing array unless using `method`.
  - With `method`, confirm the method exists in the current target/store context and returns or mutates the intended array.
  - For panel/list mutations, verify the view receives a new array reference or an observable method update.
- `update-form`
  - Use `formName` plus `value` to fill or patch a form.
  - Prefer this over store-only writes when the visible form instance must update.
- `submit-form` / `validate-form` / `reset-form`
  - Always match `formName` to a real `form.name`.
  - Use `validate-form` for custom cross-field checks, then run `submit-form` or `ajax` in `onSuccess`.
  - `reset-form` clears values and validation; if fields must be preserved, reset first and then `update-form`.
- `confirm`
  - Destructive or sync/network side effects belong in `onOk`.
  - Cancel paths must not execute the side-effect action.
- `method`
  - Treat as a last resort for behavior that documented actions cannot express. Keep expressions small and check variable scope.
- `localstore` (sport-ui)
  - Registered in `/data/project/sport-ui/src/main.tsx`.
  - `read` requires `key` and `storeField`; `defaultValue` is useful for first load.
  - `write` requires `key` plus either `storeField` or `value`.
  - Scope keys by project/user/page when state must not leak across workspaces.
  - `clear` with `all: true` is broad and should be treated as destructive.
- `handlerPath`, `update-current-path`, `update-history`, `batch-shell` (sport-ui)
  - These are webshell-specific actions; inspect `/data/project/sport-ui/src/action/` before changing their call sites.
  - They depend on form refs and panel/shell runtime objects, so validate with browser interaction, not static JSON only.

## Component Rules
- `form`
  - `name` and `initialValues` must align with `initStore`.
  - For same-form display/disable/validation linkage, prefer current form values over store mirroring.
  - `submitOnChange` and broad `changeAction` can be noisy; scope with `changeFields` where possible.
- `form-item`
  - Prefer `itemVisible` and `itemDisabled`; `visible` is compatibility-only for new work.
  - Expressions read current form fields directly, for example `${userType === 'enterprise'}`. Do not wrap with `getFormValue`.
  - Hidden or disabled fields do not participate in validation; verify this matches the workflow.
- `dialog`
  - `open` must bind to store state, and close/cancel actions must write it back to `false`.
  - Form dialogs need stable `form.name`; submit should validate, save, close, then reload in that order unless the page has a different established pattern.
- `table`
  - `rowData` must be an array; use defensive expressions when backend data can be empty or differently shaped.
  - Set a stable height through `style` unless the surrounding layout already guarantees one.
  - `keyField` is required for editing, drag, and reliable row identity.
  - Row and cell action context uses `row`, `fields`, `oldValue`, `newValue`, `api`, etc.; match names from `components/table.md`.
- `input-table`
  - When binding `value` to a store array, define matching `initStoreType` on the owning `layout-fit`; otherwise MST may create frozen rows and edits can fail.
  - When nested inside `form-item.name`, it can bind through the form field; compare with `input-table-form-demo.json` before adding redundant `value` wiring.
  - `value` auto-syncs to the bound store variable; do not add redundant action chains unless the doc/source requires it.
- `listview`
  - `itemData` must be an array and `keyField` must identify each row.
  - Inside `itemAttr`, use `${row.field}`. Put row-level switching in `rowClickAction`; child button `action` should handle only that button.
  - If a list does not refresh after a mutation, ensure the store update creates a new array reference.
- `tabs`
  - Bind `activeKey` to store and keep it aligned with item keys.
  - Put real panel content inside `items[].children` or `itemData/itemAttr` tab children.
  - Avoid fake tab switching with external `visible`; it can unmount editor/form/tree state.
  - For panes rendered from arrays, use a fresh array expression such as `${[...list]}` when needed to force rerender.
- `panel-group` / `panel` / `panel-resize`
  - The outer container must provide height, usually through `layout-fit` and `style: {"height":"100%"}`.
  - `children` must be an array; interleave `panel` and `panel-resize` deliberately.
  - `panel-resize` often needs explicit width/height/cursor/style to be visible and draggable.
  - Keep panel `id`, `order`, `defaultSize`, `minSize`, and `autoSaveId` stable; scope saved layout ids by project/workspace where needed.
- `editor` and `workspace-editor-pool` (sport-ui)
  - Registered in `/data/project/sport-ui/src/main.tsx`, implemented under `/data/project/sport-ui/src/components/`.
  - `editor` value flows through `value`, `path`, form fields, and optional action props such as `saveAction` / `onChangeAction`; confirm Monaco height and model path.
  - `workspace-editor-pool` depends on `content`, `activeKey`, `panelList`, project code, and per-pane cache state; verify tab switching, unsaved content, and active token alignment in the browser.
- `pull-down`
  - Page JSON should use `config`; runtime converts it to `pullDownConfig`.
  - Align `labelField`, `valueField`, `value`, `multiple`, and downstream row/value reads.
  - Use `action` for value changes and `searchAction` for keyword changes; `onChange` is legacy.

## Webshell Workspace Guardrails
- Start from `collect/frontend/page_data/index.yml` and resolve the `frontend.webshell` service tree before editing. The live route is a composed page, not just `webshell.json`.
- Then resolve backend calls such as `webshell.workspace_project_query`, `webshell.workspace_project_sync_files`, and `webshell.workspace_file_query` through `collect/webshell/service.yml` and the mounted child `index.yml`.
- Current webshell frontend uses both collect-ui docs and sport-ui extensions: `localstore`, `editor`, `workspace-editor-pool`, `ssh`, `handlerPath`, `batch-shell`, `update-current-path`, and `update-history`.
- For workspace projects, keep the chain in one mode:
  - Option mode: rows shaped as `{label, value, path}` and downstream reads `row.value`.
  - Raw row mode: rows shaped as `{project_code, project_name}` and downstream reads `row.project_code`.
- Keep `workspaceCurrent`, tab/list `keyField`, `activeKey`, and loaded-map keys aligned.
- Isolate reload groups such as project switcher reloads and project-management dialog reloads.
- Sync buttons must open `confirm` first; the real sync action runs only from `onOk`.
- For HTTP console/doc editing, distinguish:
  - document state: `workspaceHttpSelectedDoc`, `workspaceHttpDocCacheMap`
  - form state: `workspace-http-console-form`, `workspaceHttpConsoleForm`
  - tab state: `workspaceHttpWorkbenchTabs`, `workspaceHttpWorkbenchActiveKey`, `workspaceHttpWorkbenchDocActiveKey`
  - tree state: `workspaceHttpTree`, `workspaceHttpTreeSelected`

## Validation
Run the smallest checks that prove the change:
1. JSON syntax for changed files, for example `jq empty <file>` when available.
2. YAML/service syntax for changed service files. Prefer the repo's existing parser/build/test path when available; otherwise use a lightweight YAML parse check.
3. Search for unresolved or renamed keys with `rg` across `collect/frontend/page_data/`, `collect/**/index.yml`, `collect/**/service.yml`, SQL files, and related models/plugins.
4. For broad webshell or framework changes, compare all used tags against docs and sport-ui registration:
   - collect tags recursively with `jq -r '.. | objects | .tag? // empty'`
   - confirm each tag has a collect-ui doc or a `setRegister` / `setAction` entry in `/data/project/sport-ui/src/main.tsx`
5. For backend service changes, POST the affected service with representative JSON payload and verify `success/code/msg/data/count` shape. Use the frontend's content type: `Content-Type: application/json`.
6. If runtime behavior changed, restart with the repo scripts and verify on port `8015`:
   - `./linux-shutdown`
   - `ss -ltnp | rg ':8015' || true`
   - `./linux-startup`
   - `ss -ltnp | rg ':8015'`
   - `curl --noproxy '*' -sS -m 5 -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8015/`
7. Browser-smoke the changed workflow when the issue is visual or interaction-driven. For webshell, load `http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell`, capture console/page errors, and check key `/template_data/data` responses.

Verify interactions in dependency order: open container/dialog -> initial load -> search/reset -> paging/selection -> edit/save/delete/sync -> close -> reload target.

## Troubleshooting Priority
1. JSON validity and expression syntax
2. Component/action doc availability for every touched tag
3. Backend service YAML docs in `tooltip-docs` for changed `module`, `handler_params`, `params`, `result_handler`, `data_file`, and `save_field`
4. Runtime registration in collect-ui/sport-ui source
5. Store/form key alignment and `initStoreType`
6. Action context variables (`row`, `value`, `activeKey`, `fields`, `data`, `msg`)
7. Backend service response shape (`success`, `code`, `msg`, `data`, `count`) and SQL/model/table references
8. Reload group isolation and loop prevention
9. Runtime asset mount path and browser console/page errors

## References
- Source map and lookup paths: `references/source-index.md`
- Webshell project management and common regressions: `references/webshell-workspace-project.md`
