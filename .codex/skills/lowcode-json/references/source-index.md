# Source Index

Use this index when editing low-code JSON and validating dependencies.

## Repository
- `AGENTS.md`
- `collect/frontend/page_data/`
- `collect/frontend/page_data/index.yml`
- `collect/frontend/service.yml`
- `collect/<domain>/service.yml`
- `collect/<domain>/<child>/index.yml`
- `frontend/data/vs/editor/editor.main.js`

## Project Service Map
- Frontend pages: `collect/frontend/service.yml` mounts `page_data/index.yml`; public service names are `frontend.<key>`.
- Page JSON files: `collect/frontend/page_data/index.yml` maps each `key` to a relative `data_file`, usually under `collect/frontend/page_data/data/`.
- Backend domains: `collect/webshell/service.yml`, `collect/sport/service.yml`, `collect/system/service.yml`, `collect/hrm/service.yml`, and similar files mount child `index.yml` files; public service names are `<domain>.<child key>`.
- SQL-backed services usually reference SQL beside the child `index.yml`; model-backed services require checking `table`, `filter`, and model registration.
- Runtime config changes under `collect/` generally need app restart before browser verification.

## Sibling Repos
- `/data/project/collect-ui/docs/readme/components/`
- `/data/project/collect-ui/docs/readme/action/`
- `/data/project/collect-ui/src/components/`
- `/data/project/collect-ui/src/action/`
- `/data/project/collect-ui/src/index.tsx`
- `/data/project/collect-ui/src/utils/getIcon.tsx`
- `/data/project/sport-ui/src/main.tsx`
- `/data/project/sport-ui/src/components/`
- `/data/project/sport-ui/src/action/`

## Lookup Strategy
1. Find a nearby page JSON with similar behavior and copy its schema style.
2. Resolve page composition in `collect/frontend/page_data/index.yml` before treating a fragment as standalone.
3. Resolve every `/template_data/data?service=...` call through the mounted backend service tree.
4. Verify component/action docs under `/data/project/collect-ui/docs/readme/`.
5. Verify collect-ui registration in `collect-ui/src/index.tsx`.
6. Verify collect-ui implementation in `collect-ui/src/components/` or `collect-ui/src/action/`.
7. For sport-specific tags/actions, verify registration in `/data/project/sport-ui/src/main.tsx` and implementation under `/data/project/sport-ui/src/`.
8. Verify icon/tag resolution in `collect-ui/src/utils/getIcon.tsx`.
9. Verify editor/runtime behavior in `frontend/data/vs/editor/editor.main.js` or `/data/project/sport-ui/src/components/editor` when action parsing looks wrong.

## Notes
- Keep key names and expression shapes consistent with existing JSON in the same page/module.
- Avoid mass reformatting for unrelated blocks.
