# AGENTS.md

Guidance for coding agents working in this repository.

## Project Snapshot

- Module: `moon` (Go `1.23`)
- HTTP framework: Gin (`github.com/gin-gonic/gin`)
- ORM: GORM (`gorm.io/gorm`)
- Low-code framework: `github.com/collect-ui/collect` (local replace `=> ../collect`)
- Architecture: low-code backend driven by YAML/JSON (under `collect/`) plus Go plugins
- Key areas: `main.go`, `model/`, `plugins/`, `collect/`, `conf/`, `frontend/`
- Dependency override in `go.mod`: `replace github.com/collect-ui/collect => ../collect`

## Common Commands

### Run Locally

```bash
./linux-startup        # build + start on port from conf/application.properties
./linux-shutdown       # graceful stop (kill by PID/port)
./linux-start-dev.sh   # delegates to linux-startup
./shutdown.sh          # delegates to linux-shutdown
```

**重要：启动命令统一用 `linux-start-dev`，不要直接运行 `moon.exe`。**

After changing Go code or config, restart: `./linux-shutdown && ./linux-startup`

**修改 YAML/JSON 配置不需要重启服务**，框架支持热加载。只有改 Go 代码才需要重启。

Startup verification (port from conf, e.g. 8017):
```bash
ss -ltnp | rg ':8017' || true
curl --noproxy '*' -sS -m 5 -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8017/
```

### Build

```bash
./build.sh              # clean build, upx compress, copy assets to dist/
go build -o moon.exe main.go   # quick build
go build ./...           # compile check all packages
```

### Tests

```bash
go test ./...                              # all packages
go test -v ./...                           # verbose
go test ./plugins/...                      # single package
go test -v -run TestName ./path/to/pkg     # single test
go test -v -run 'TestName/Subcase' ./path  # single subtest (table-driven)
go test -coverprofile=coverage.out ./...   # coverage
go tool cover -html=coverage.out           # coverage report
go test -json -run "$pattern" -v ./...     # JSON output (used by regression script)
```

Regression test suite:
```bash
bash test/run_agent_runtime_regression.sh
```

### Formatting, Lint, Vet

```bash
go fmt ./...
go vet ./...
go mod tidy
staticcheck ./...       # if available
golangci-lint run       # if available
```

### Recommended Validation Flow

For Go code changes: `go fmt ./... && go test ./... && go vet ./...`
For config-only changes: `go test ./...`, then start app if runtime wiring is affected.

## Repository Structure

- `main.go` – application bootstrap, Gin routes, TLS, session middleware
- `model/` – GORM domain models + `register.go` (model registry via `init()`)
- `plugins/` – custom low-code/plugin handlers + `a_register.go` (plugin registry)
- `collect/` – YAML/JSON service definitions (low-code pages, services, stores, forms)
- `conf/` – `application.properties` (DB, LDAP, Jira, SSL, scheduling config)
- `frontend/` – static SPA frontend assets
- `database/` – local DB files (treat as environment data, not source)
- `docs/`, `feature/` – design docs
- `test/` – test artifacts and regression scripts
- `sql/` – SQL migration files

## Low‑Code 开发指南

低代码开发详细指引（CRUD 目录结构、base.sql 模式、分页、增删改配置、路由注册、前端页面开发、Store 作用域与 targetStore、表单实战约定）已抽取为独立 skill：

> **`.opencode/skills/lowcode-backend/SKILL.md`**

加载方式：`skill({ name: "lowcode-backend" })`

关键约束（已在 AI 记忆中固化，无需每次都加载 skill）：
- 本项目数据库是 **Oracle**，分页禁止使用 `LIMIT`，必须使用 `rownum` 三文件模式（详见 [`docs/oracle-pagination.md`](docs/oracle-pagination.md)）。**禁止**在 SQL 模板里手写 `OFFSET ... FETCH NEXT`，框架 GORM Oracle 驱动会自动追加，手写会报 `ORA-00933`
- **`delete_flag` 语义与直觉相反**：`'1'` = 正常（未删除），`'0'` = 已删除。写 SQL 时过滤正常记录必须用 `delete_flag = '1'`（或 `nvl(delete_flag, '1') = '1'`），绝不能写 `= '0'`
- 禁止写 Go 业务代码
- 禁止前端 schema 匿名函数
- 低代码优先原则

## Code Style Guidelines

### Imports

- Standard Go order: standard library, third-party, local (`moon/...`).
- One import block per file.
- Preserve existing aliases:
  - `common "github.com/collect-ui/collect/src/collect/common"`
  - `config "github.com/collect-ui/collect/src/collect/config"`
  - `templateService "github.com/collect-ui/collect/src/collect/service_imp"`
  - `utils "github.com/collect-ui/collect/src/collect/utils"`
  - `collect "github.com/collect-ui/collect/src/collect/utils"` (same as utils)
  - `mysqlDriver "github.com/go-sql-driver/mysql"`
  - `go_ora "github.com/sijms/go-ora/v2"`

### Formatting

- Use `gofmt`; do not hand-format.
- Files may contain Chinese text/business labels; keep ASCII elsewhere.
- Small focused changes over broad rewrites.

### Naming

- Packages: lowercase, single word (exception: `work_task` uses underscore).
- Exported: PascalCase (`GetRegisterList`, `WebSQLService`, `Result`).
- Unexported: camelCase (`openWebSQLDB`, `webSQLConnectionConfig`).
- File names: snake_case for plugin files (`handler_params_*.go`, `module_websql.go`).
- Constants: PascalCase for exported, camelCase for package-private.
- Models: `TableNameXxx` const + `TableName() string` + `PrimaryKey() []string` methods.
- Model ID fields: if DB column is `*_id`, Go struct field must use `ID` (all caps), never `Id`. Example: `DoctorID`, `AreaCode` (if no `_id` suffix, keep original).

### Types and Structs

- Embed shared base structs: `templateService.BaseHandler`, `templateService.DatabaseModel`.
- Use `map[string]interface{}` pervasively for dynamic JSON/API data.
- Use `[]map[string]interface{}` for lists of objects.
- Use `gocast.ToString()`, `gocast.ToInt64()`, `gocast.ToInt()` for safe casting.
- Keep struct field order stable; follow existing registration patterns:
  - Models: register in `model/register.go` via domain package `GetTable()`.
  - Plugins: register in `plugins/a_register.go` via `GetRegisterList()`.

### Error Handling

- Check errors immediately with `if err != nil { return ... }`.
- Return contextual errors with `fmt.Errorf("...: %w", err)`.
- Plugin handlers: use `common.Ok(data, "操作成功")` / `common.NotOk(err.Error())`.
- Custom error types implement `Error()` and `Unwrap()` (for `errors.As`/`errors.Is`).
- Avoid panics in request-path code (only in startup/`init()`).

### Control Flow

- Early returns for validation failures.
- Plugin `Result` methods should be linear and easy to scan.
- Deferred cleanup: `defer db.Close()`, `defer rows.Close()`.
- Switch on strings for operation dispatch (common pattern).

### Comments

- Chinese comments for business logic; English for technical explanations.
- Document exported functions/types when adding new public API.
- Short and focused on intent, not line-by-line narration.

### Testing

- Standard `testing` package only (no third-party test framework).
- Use `t.Fatalf(...)` for failures (not `t.Error`/`t.Errorf`).
- Table-driven tests with `[]struct{...}` + `t.Run(name, ...)` for multiple cases.
- Use SQLite `:memory:` for DB integration tests where practical.
- Test files live alongside source: `plugins/module_websql_test.go`, etc.

### Configuration and Low-Code Files

- Prefer changing YAML/JSON under `collect/` over hard-coding Go.
- Preserve key names, indentation, and schema shape in config files.
- Check references across `collect/`, `conf/`, and plugin code before renaming service keys.
- Service router: `collect/service_router.yml` maps keys to YAML paths.
- Each module has `service.yml` + `index.yml`; leaf files define actual services.

### Database and Generated Artifacts

- Do not commit `database/`, `windows/`, `bin/`, `test/bin/`, IDE files, or archives.
- Generated GORM models have `.gen.go` suffix with `DO NOT EDIT` header.
- Do not query or inspect `mail_account` table schema/data.
- Verify generated files are intended source artifacts, not environment output.

## Working Conventions for Agents

- Read neighboring files before changing patterns in a subsystem.
- Preserve compatibility with `../collect` replace target.
- Adding a plugin: update implementation + `plugins/a_register.go`.
- Adding a model: update domain package + `model/register.go`.
- For behavior already driven by low-code config, prefer config changes over Go code.
- Write Go/JS/React only when config alone is insufficient; state why.
- `origin/master` is the only remote branch visible in this checkout.

## Practical Notes

- No `.cursor/rules/`, `.cursorrules`, or `.github/copilot-instructions.md` exist.
- Frontend assets exist but no top-level Node build manifest.
- If a build fails, first verify sibling `../collect` module exists and is in sync.
- The project uses Chinese UI labels, error messages, and comments extensively.
- Cache hints do NOT tell you to use `go run` directly; always use `./linux-startup`.
- 老 HIS 参考目录在 `/data/his`；前后端关系说明见 `/data/his/前后端关系图.md`。OPDM 号源/排班老前端在 `/data/his/frontend/opdm`，号源管理入口对应 `src/pages/numberSourceSchedule/`（旧 URL 形如 `/opdm-ui/numberSourceSchedule?_funCode=hygl&_sys=OPDM...`），新增临时排班弹框对应 `src/pages/numberSourceSchedule/components/addTemporarySchedule.vue`；迁移当前低代码号源新增页时优先对照该弹框的三列基础表单、时段号源/分时号源分区和下拉联动。

## 开发踩坑记录

前端构建体系、form/store 同步、Oracle 字段名、collect-ui 组件约定等非显而易见的配置陷阱，详见：

> **[`docs/dev-pitfalls.md`](docs/dev-pitfalls.md)**
