package devops

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const TableNameWebshellQuickText = "webshell_quick_text"

type WebshellQuickText struct {
	QuickTextID string  `gorm:"column:quick_text_id;primaryKey" json:"quick_text_id"`
	Title       *string `gorm:"column:title" json:"title"`
	Description *string `gorm:"column:description;type:text" json:"description"`
	Content     *string `gorm:"column:content;type:text" json:"content"`
	RoleType    *string `gorm:"column:role_type" json:"role_type"`
	EffectExts  *string `gorm:"column:effect_exts" json:"effect_exts"`
	UseCount    *int    `gorm:"column:use_count" json:"use_count"`
	IsFavorite  *string `gorm:"column:is_favorite" json:"is_favorite"`
	IsDelete    *string `gorm:"column:is_delete" json:"is_delete"`
	CreateTime  *string `gorm:"column:create_time" json:"create_time"`
	CreateUser  *string `gorm:"column:create_user" json:"create_user"`
	ModifyTime  *string `gorm:"column:modify_time" json:"modify_time"`
	ModifyUser  *string `gorm:"column:modify_user" json:"modify_user"`
}

func (*WebshellQuickText) TableName() string {
	return TableNameWebshellQuickText
}

func (*WebshellQuickText) PrimaryKey() []string {
	return []string{"quick_text_id"}
}

type quickTextSeed struct {
	Title         string
	Description   string
	Content       string
	LegacyContent string
	RoleType      string
	EffectExts    string
	IsFavorite    string
}

func EnsureDefaultWebshellQuickTexts(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	var count int64
	if err := db.Model(&WebshellQuickText{}).Where("is_delete = ?", "0").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		if err := repairDefaultWebshellQuickTexts(db); err != nil {
			return err
		}
		return ensureMissingDefaultWebshellQuickTexts(db)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	rows := make([]WebshellQuickText, 0, len(defaultQuickTextSeeds))
	for _, seed := range defaultQuickTextSeeds {
		title := seed.Title
		description := seed.Description
		content := seed.Content
		roleType := seed.RoleType
		effectExts := seed.EffectExts
		useCount := 0
		isFavorite := seed.IsFavorite
		isDelete := "0"
		createUser := "system"
		modifyUser := "system"
		createTime := now
		modifyTime := now
		rows = append(rows, WebshellQuickText{
			QuickTextID: "quick_text_" + uuid.NewString(),
			Title:       &title,
			Description: &description,
			Content:     &content,
			RoleType:    &roleType,
			EffectExts:  &effectExts,
			UseCount:    &useCount,
			IsFavorite:  &isFavorite,
			IsDelete:    &isDelete,
			CreateTime:  &createTime,
			CreateUser:  &createUser,
			ModifyTime:  &modifyTime,
			ModifyUser:  &modifyUser,
		})
	}
	return db.Create(&rows).Error
}

func repairDefaultWebshellQuickTexts(db *gorm.DB) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, seed := range defaultQuickTextSeeds {
		if seed.LegacyContent == "" || seed.LegacyContent == seed.Content {
			continue
		}
		if err := db.Model(&WebshellQuickText{}).
			Where("is_delete = ? AND title = ? AND content = ?", "0", seed.Title, seed.LegacyContent).
			Updates(map[string]interface{}{
				"content":     seed.Content,
				"modify_time": now,
				"modify_user": "system",
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureMissingDefaultWebshellQuickTexts(db *gorm.DB) error {
	titles := make([]string, 0, len(defaultQuickTextSeeds))
	for _, seed := range defaultQuickTextSeeds {
		titles = append(titles, seed.Title)
	}
	existingTitles := make([]string, 0, len(titles))
	if err := db.Model(&WebshellQuickText{}).
		Where("title IN ?", titles).
		Pluck("title", &existingTitles).Error; err != nil {
		return err
	}
	existing := make(map[string]bool, len(existingTitles))
	for _, title := range existingTitles {
		existing[title] = true
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	rows := make([]WebshellQuickText, 0)
	for _, seed := range defaultQuickTextSeeds {
		if existing[seed.Title] {
			continue
		}
		title := seed.Title
		description := seed.Description
		content := seed.Content
		roleType := seed.RoleType
		effectExts := seed.EffectExts
		useCount := 0
		isFavorite := seed.IsFavorite
		isDelete := "0"
		createUser := "system"
		modifyUser := "system"
		createTime := now
		modifyTime := now
		rows = append(rows, WebshellQuickText{
			QuickTextID: "quick_text_" + uuid.NewString(),
			Title:       &title,
			Description: &description,
			Content:     &content,
			RoleType:    &roleType,
			EffectExts:  &effectExts,
			UseCount:    &useCount,
			IsFavorite:  &isFavorite,
			IsDelete:    &isDelete,
			CreateTime:  &createTime,
			CreateUser:  &createUser,
			ModifyTime:  &modifyTime,
			ModifyUser:  &modifyUser,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}

func buildDefaultQuickTextSeeds() []quickTextSeed {
	seeds := make([]quickTextSeed, 0, len(baseQuickTextSeeds)+len(readmeQuickTextSeeds))
	seeds = append(seeds, baseQuickTextSeeds...)
	seeds = append(seeds, readmeQuickTextSeeds...)
	return seeds
}

func newReadmeQuickTextSeed(title, description, roleType, content string) quickTextSeed {
	return quickTextSeed{
		Title:       title,
		Description: description,
		RoleType:    roleType,
		EffectExts:  "md,markdown,txt,readme",
		IsFavorite:  "0",
		Content:     content,
	}
}

var defaultQuickTextSeeds = buildDefaultQuickTextSeeds()

var baseQuickTextSeeds = []quickTextSeed{
	{
		Title:         "研发实现闭环",
		Description:   "从需求文档到实现、验证、日志沉淀的研发执行提示词。",
		RoleType:      "研发",
		EffectExts:    "md,markdown,txt,readme",
		IsFavorite:    "1",
		LegacyContent: `请以资深工程师视角完成本文需求：先阅读需求和相邻实现，定位真实入口；优先复用现有架构和配置；实现后运行必要的格式化、编译、测试和页面回归；最后把设计取舍、修改文件、验证结果和未解决风险记录到本文。`,
		Content: `请以资深工程师视角完成本文需求：

1. 先阅读需求文档和相邻实现，定位真实入口。
2. 优先复用现有架构、配置和组件能力。
3. 实现后运行必要的格式化、编译、测试和页面回归。
4. 最后把设计取舍、修改文件、验证结果和未解决风险记录到本文。`,
	},
	{
		Title:         "无头浏览器回归要求",
		Description:   "页面改动后使用 Playwright 验证真实工作流。",
		RoleType:      "测试",
		EffectExts:    "md,markdown,txt,readme",
		IsFavorite:    "1",
		LegacyContent: `测试要求：使用无头浏览器打开目标页面，按用户真实路径完成操作；记录 console error、pageerror、requestfailed；保存 JSON 报告和关键截图；失败时先根据截图和 DOM 证据修复，再重复验证直到通过。`,
		Content: `测试要求：

- 使用无头浏览器打开目标页面，按用户真实路径完成操作。
- 记录 console error、pageerror、requestfailed。
- 保存 JSON 报告和关键截图。
- 失败时先根据截图和 DOM 证据修复，再重复验证直到通过。`,
	},
	{
		Title:         "研发角色定义",
		Description:   "约束实现风格，避免大范围重构和无依据改动。",
		RoleType:      "研发",
		EffectExts:    "md,markdown,txt,readme",
		IsFavorite:    "0",
		LegacyContent: `你是一个务实的资深研发。先让现有系统结构教你怎么改；保持改动小而完整；不回滚用户已有变更；能用配置表达的行为优先用配置；必须说明每个关键实现选择和验证证据。`,
		Content: `你是一个务实的资深研发。

- 先让现有系统结构教你怎么改。
- 保持改动小而完整。
- 不回滚用户已有变更。
- 能用配置表达的行为优先用配置。
- 必须说明每个关键实现选择和验证证据。`,
	},
	{
		Title:         "测试角色定义",
		Description:   "用于补充测试人员职责和验收边界。",
		RoleType:      "测试",
		EffectExts:    "md,markdown,txt,readme",
		IsFavorite:    "0",
		LegacyContent: `你是严格的回归测试负责人。先列核心路径、边界路径和历史易回归点；每个测试必须有明确前置条件、操作步骤、预期结果和证据文件；发现问题时记录最小复现路径和修复后复验结果。`,
		Content: `你是严格的回归测试负责人。

- 先列核心路径、边界路径和历史易回归点。
- 每个测试必须有明确前置条件、操作步骤、预期结果和证据文件。
- 发现问题时记录最小复现路径和修复后复验结果。`,
	},
	{
		Title:         "低代码页面改造检查",
		Description:   "修改 collect/page_data JSON 时的检查清单。",
		RoleType:      "低代码",
		EffectExts:    "json,md,markdown,txt,readme",
		IsFavorite:    "1",
		LegacyContent: `低代码改造检查：先定位 initStore、表单、action group、list/tabs 绑定；保持 store key 一致；不要把 reload 绑到高频字段更新；优先复用现有 action 链；修改后用 jq 校验 JSON，并用页面真实操作验证打开、查询、保存、删除、刷新。`,
		Content: `低代码改造检查：

- 先定位 initStore、表单、action group、list/tabs 绑定。
- 保持 store key 一致。
- 不要把 reload 绑到高频字段更新。
- 优先复用现有 action 链。
- 修改后用 jq 校验 JSON，并用页面真实操作验证打开、查询、保存、删除、刷新。`,
	},
	{
		Title:         "Harness 执行约束",
		Description:   "适合放到任务 README 中约束自动化执行过程。",
		RoleType:      "Harness",
		EffectExts:    "md,markdown,txt,readme",
		IsFavorite:    "0",
		LegacyContent: `Harness 约束：执行前记录仓库状态；只改本需求相关文件；不要删除数据库、构建产物或用户改动；命令失败要保留关键输出并继续定位原因；长流程要产出可复跑脚本、报告路径和最终状态。`,
		Content: `Harness 约束：

- 执行前记录仓库状态。
- 只改本需求相关文件。
- 不要删除数据库、构建产物或用户改动。
- 命令失败要保留关键输出并继续定位原因。
- 长流程要产出可复跑脚本、报告路径和最终状态。`,
	},
	{
		Title:       "问题复现记录模板",
		Description: "快速插入 bug/回归问题记录结构。",
		RoleType:    "通用",
		EffectExts:  "md,markdown,txt,readme",
		IsFavorite:  "0",
		Content: `## 问题复现
- 页面/入口：
- 前置数据：
- 操作步骤：
- 实际结果：
- 预期结果：
- 证据文件：
- 初步判断：
- 修复后复验：`,
	},
	{
		Title:       "设计与验证日志模板",
		Description: "用于 feature 文档末尾沉淀实现过程。",
		RoleType:    "通用",
		EffectExts:  "md,markdown,txt,readme",
		IsFavorite:  "1",
		Content: `## 设计思路

## 修改范围

## 验证过程

## 执行日志

## 后续风险`,
	},
}

var readmeQuickTextSeeds = []quickTextSeed{
	newReadmeQuickTextSeed(
		"README 项目总览骨架",
		"用于 README 开头，快速说明项目价值、边界和读者入口。",
		"README",
		`## 项目总览

本项目用于解决：

- 核心用户：
- 主要场景：
- 不解决的问题：
- 当前稳定程度：

建议阅读顺序：

1. 先看快速开始，确认能在本地跑通。
2. 再看架构地图，理解主要模块和数据流。
3. 最后看开发、测试、部署章节，按团队约定交付变更。`,
	),
	newReadmeQuickTextSeed(
		"README 快速开始闭环",
		"把环境、启动、验证和常见失败点写成可执行路径。",
		"README",
		`## 快速开始

前置条件：

- 运行环境：
- 必要依赖：
- 本地配置：

启动步骤：

1. 安装依赖：
2. 初始化数据：
3. 启动服务：
4. 打开页面：

启动后验证：

- 健康检查：
- 核心接口：
- 核心页面：

常见失败点：

- 端口被占用：
- 配置缺失：
- 数据库未初始化：`,
	),
	newReadmeQuickTextSeed(
		"README 架构地图",
		"给维护者一个模块、边界、调用方向清晰的架构说明模板。",
		"README",
		`## 架构地图

核心模块：

- 入口层：
- 服务层：
- 数据层：
- 配置层：
- 前端层：

调用方向：

1. 用户操作进入页面或接口。
2. 接口读取配置、校验参数、调用服务。
3. 服务访问数据库或外部系统。
4. 结果回写页面状态并产生日志。

维护约束：

- 不跨层直接访问内部细节。
- 新能力优先复用现有配置和服务。
- 共享逻辑进入明确的公共模块。`,
	),
	newReadmeQuickTextSeed(
		"README 配置说明",
		"说明配置文件位置、关键字段、默认值和安全注意事项。",
		"README",
		`## 配置说明

配置文件：

- 主配置：
- 页面配置：
- 服务配置：
- 环境变量：

关键字段：

| 字段 | 默认值 | 说明 | 是否必填 |
| --- | --- | --- | --- |
|  |  |  |  |

配置修改后需要：

1. 校验格式。
2. 重启服务或刷新页面缓存。
3. 执行最小回归。

安全注意：

- 密钥不要写入 README 示例。
- 示例账号使用脱敏值。
- 生产配置单独管理。`,
	),
	newReadmeQuickTextSeed(
		"README 开发工作流",
		"描述从领任务到验证完成的日常研发流程。",
		"README",
		`## 开发工作流

推荐流程：

1. 阅读需求、历史实现和相邻模块。
2. 明确改动范围和风险点。
3. 小步实现，保持每次修改可解释。
4. 运行格式化、编译、单测和关键页面验证。
5. 在文档或变更记录里沉淀结论。

提交前检查：

- 是否只改了本需求相关文件。
- 是否保留用户已有改动。
- 是否有可复验的命令和报告。
- 是否说明未覆盖的风险。`,
	),
	newReadmeQuickTextSeed(
		"README 部署发布",
		"把构建、部署、回滚和发布后巡检写成标准步骤。",
		"README",
		`## 部署发布

构建步骤：

1. 拉取目标分支。
2. 安装或确认依赖。
3. 执行构建命令。
4. 校验产物完整性。

部署步骤：

1. 备份当前版本。
2. 上传新产物。
3. 应用配置变更。
4. 重启服务。
5. 执行健康检查。

回滚方案：

- 回滚产物：
- 回滚配置：
- 回滚数据：

发布后巡检：

- 日志错误：
- 核心接口：
- 核心页面：
- 性能指标：`,
	),
	newReadmeQuickTextSeed(
		"README 故障排查",
		"为线上或本地问题提供有序排查模板。",
		"README",
		`## 故障排查

先确认影响面：

- 发生时间：
- 影响用户：
- 影响功能：
- 是否可稳定复现：

排查顺序：

1. 看健康检查和进程状态。
2. 看最近发布、配置和数据变更。
3. 看服务日志和浏览器控制台。
4. 用最小输入复现问题。
5. 记录修复动作和复验结果。

输出结论：

- 根因：
- 修复：
- 预防：
- 待观察：`,
	),
	newReadmeQuickTextSeed(
		"README 变更记录",
		"用于 README 内维护用户能看懂的版本变化。",
		"README",
		`## 变更记录

### 版本：

发布日期：

新增：

- 

调整：

- 

修复：

- 

兼容性：

- 

验证：

- 构建：
- 单测：
- 页面回归：
- 接口回归：`,
	),
	newReadmeQuickTextSeed(
		"README API 使用说明",
		"为接口型项目补充请求、响应、错误和示例说明。",
		"README",
		`## API 使用说明

基础信息：

- Base URL：
- 鉴权方式：
- 请求格式：
- 响应格式：

接口列表：

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
|  |  |  |  |

错误响应：

| code | 含义 | 处理建议 |
| --- | --- | --- |
|  |  |  |

调用示例：

- 查询：
- 新增：
- 修改：
- 删除：`,
	),
	newReadmeQuickTextSeed(
		"README 贡献与约定",
		"写清楚团队协作、代码风格、分支和评审要求。",
		"README",
		`## 贡献与约定

开发约定：

- 遵循现有目录结构和命名风格。
- 公共能力先查找已有实现。
- 避免无关重构和格式化噪音。

分支约定：

- 功能分支：
- 修复分支：
- 发布分支：

评审要求：

1. 描述用户问题和解决方案。
2. 列出关键文件和风险点。
3. 附上可复跑的验证命令。
4. 对未覆盖范围做明确说明。`,
	),
	newReadmeQuickTextSeed(
		"Vibe Coding 需求澄清",
		"在 README 或任务文档中快速建立目标、约束和验收边界。",
		"Vibe Coding",
		`## Vibe Coding 需求澄清

先把问题说清楚：

- 用户是谁：
- 当前痛点：
- 希望达成的行为：
- 不接受的结果：

约束条件：

- 时间：
- 技术栈：
- 兼容性：
- 数据安全：

验收方式：

1. 用户路径能完整跑通。
2. 关键状态有可见反馈。
3. 失败场景有明确提示。
4. 有截图、日志或测试报告作为证据。`,
	),
	newReadmeQuickTextSeed(
		"Vibe Coding 探索式实现",
		"适合记录边做边验证的实现策略，避免盲目大改。",
		"Vibe Coding",
		`## Vibe Coding 探索式实现

执行策略：

1. 先跑通最小闭环，不急着抽象。
2. 每发现一个真实约束，就更新实现假设。
3. 保持改动能被快速撤回或替换。
4. 用真实页面、真实接口、真实数据验证体验。

记录内容：

- 初始假设：
- 发现的系统约束：
- 改动后的方案：
- 放弃的方案和原因：
- 下一步风险：`,
	),
	newReadmeQuickTextSeed(
		"Vibe Coding 小步提交",
		"约束 AI 辅助编码时的小步迭代和验证节奏。",
		"Vibe Coding",
		`## Vibe Coding 小步提交

每一步只解决一个问题：

- 数据结构：
- 接口服务：
- 页面交互：
- 样式细节：
- 自动化验证：

每一步完成后：

1. 查看 diff，确认没有无关改动。
2. 运行最便宜的验证。
3. 记录当前证据。
4. 再进入下一步。

完成标准：

- 功能可用。
- 代码可读。
- 验证可复跑。
- 风险说清楚。`,
	),
	newReadmeQuickTextSeed(
		"Vibe Coding 代码阅读",
		"指导先理解系统再修改，适合放在任务 README 里。",
		"Vibe Coding",
		`## Vibe Coding 代码阅读

阅读顺序：

1. 找入口：路由、页面、命令或任务脚本。
2. 找数据：模型、接口、配置、缓存。
3. 找状态：前端 store、后端事务、异步队列。
4. 找边界：权限、文件系统、网络、外部服务。

阅读输出：

- 真实入口：
- 关键调用链：
- 必须保持的兼容行为：
- 可以安全修改的位置：
- 需要验证的用户路径：`,
	),
	newReadmeQuickTextSeed(
		"Vibe Coding 重构护栏",
		"在 README 中说明何时能重构、如何控制重构范围。",
		"Vibe Coding",
		`## Vibe Coding 重构护栏

允许重构的条件：

- 重复逻辑已经影响本次需求。
- 现有结构阻碍测试或修复。
- 抽象后能减少真实复杂度。

不做的事情：

- 不为了风格统一改全局。
- 不移动无关文件。
- 不改变未验证的外部契约。

重构后验证：

1. 原有路径不变。
2. 新路径可用。
3. 失败路径仍有提示。
4. diff 能解释每个文件为什么被改。`,
	),
	newReadmeQuickTextSeed(
		"Vibe Coding AI 协作提示",
		"给 AI 编码助手的上下文、约束和交付格式模板。",
		"Vibe Coding",
		`## Vibe Coding AI 协作提示

请按以下方式执行：

1. 先阅读需求、相邻代码和项目约定。
2. 明确你的实现假设和风险。
3. 优先做最小可用闭环。
4. 修改前说明要改哪些文件。
5. 修改后运行验证并报告结果。

输出格式：

- 问题原因：
- 修改内容：
- 验证命令：
- 验证结果：
- 剩余风险：`,
	),
	newReadmeQuickTextSeed(
		"Vibe Coding 快速原型",
		"用于 README 记录快速验证想法的范围和退出条件。",
		"Vibe Coding",
		`## Vibe Coding 快速原型

原型目标：

- 需要验证的假设：
- 最小用户路径：
- 不纳入原型的范围：

实现原则：

1. 数据可以简化，但交互路径要真实。
2. 视觉可以克制，但状态反馈要完整。
3. 先证明价值，再决定是否产品化。

退出条件：

- 假设成立：
- 假设不成立：
- 需要补充数据：
- 需要重做方案：`,
	),
	newReadmeQuickTextSeed(
		"Vibe Coding 上下文压缩",
		"长任务中保留关键上下文，避免后续实现跑偏。",
		"Vibe Coding",
		`## Vibe Coding 上下文压缩

保留这些信息：

- 用户真实诉求：
- 已经确认的入口：
- 已经修改的文件：
- 已经通过的验证：
- 已知失败和原因：

丢弃这些噪音：

- 无关日志全文。
- 重复的命令输出。
- 已排除的猜测。

恢复工作时：

1. 先核对最新用户消息。
2. 查看当前 diff。
3. 只继续未完成的任务。`,
	),
	newReadmeQuickTextSeed(
		"Vibe Coding 反馈循环",
		"把用户反馈转成可验证改动的流程模板。",
		"Vibe Coding",
		`## Vibe Coding 反馈循环

处理反馈：

1. 复述现象，不急着给方案。
2. 用数据、截图或日志确认问题位置。
3. 区分数据问题、逻辑问题和体验问题。
4. 做最小修复。
5. 用同一路径复验。

记录结果：

- 反馈原文：
- 复现方式：
- 根因判断：
- 修复点：
- 复验截图或报告：`,
	),
	newReadmeQuickTextSeed(
		"Vibe Coding 交付复盘",
		"任务结束时沉淀设计取舍、验证和后续风险。",
		"Vibe Coding",
		`## Vibe Coding 交付复盘

本次完成：

- 用户价值：
- 关键改动：
- 保持不变的行为：

设计取舍：

- 选择的方案：
- 没选的方案：
- 主要原因：

验证证据：

- 编译：
- 单测：
- 页面：
- 接口：

后续风险：

- 技术债：
- 数据风险：
- 体验风险：`,
	),
	newReadmeQuickTextSeed(
		"设计 用户目标",
		"把设计讨论从功能清单拉回用户目标和成功标准。",
		"设计",
		`## 设计：用户目标

目标用户：

- 主要用户：
- 次要用户：
- 维护者：

用户要完成的任务：

1. 
2. 
3. 

成功标准：

- 更快：
- 更准：
- 更安全：
- 更容易复用：

非目标：

- 

设计风险：

- 用户理解成本：
- 数据质量：
- 权限边界：`,
	),
	newReadmeQuickTextSeed(
		"设计 信息架构",
		"规划 README、页面或工具的信息层级。",
		"设计",
		`## 设计：信息架构

一级信息：

- 用户首先要看到：
- 用户最常操作：
- 用户最需要确认：

二级信息：

- 详情：
- 历史：
- 设置：
- 帮助：

隐藏或折叠：

- 低频配置：
- 危险操作：
- 调试信息：

检查标准：

1. 首屏能回答这是什么。
2. 用户能找到下一步。
3. 错误状态不会被藏起来。`,
	),
	newReadmeQuickTextSeed(
		"设计 状态模型",
		"描述页面或服务的状态、转换和异常状态。",
		"设计",
		`## 设计：状态模型

核心状态：

- 初始：
- 加载中：
- 可编辑：
- 保存中：
- 已保存：
- 失败：

状态转换：

| 当前状态 | 事件 | 下一个状态 | 用户反馈 |
| --- | --- | --- | --- |
|  |  |  |  |

异常状态：

- 空数据：
- 无权限：
- 网络失败：
- 数据冲突：

实现要求：

- 状态来源单一。
- 异步结果可回收。
- 保存和撤销路径清晰。`,
	),
	newReadmeQuickTextSeed(
		"设计 交互流程",
		"把关键用户路径拆成触发、反馈、完成和失败处理。",
		"设计",
		`## 设计：交互流程

主路径：

1. 用户进入：
2. 用户选择：
3. 用户编辑：
4. 用户确认：
5. 系统反馈：

关键反馈：

- 操作可用：
- 操作进行中：
- 操作成功：
- 操作失败：

失败处理：

- 用户能看到原因。
- 用户能重试。
- 用户不会丢失已输入内容。
- 危险操作需要确认。`,
	),
	newReadmeQuickTextSeed(
		"设计 数据模型",
		"为 README 或设计文档补充实体、字段和生命周期。",
		"设计",
		`## 设计：数据模型

核心实体：

| 实体 | 说明 | 生命周期 |
| --- | --- | --- |
|  |  |  |

关键字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
|  |  |  |  |

数据关系：

- 一对一：
- 一对多：
- 多对多：

数据约束：

- 唯一性：
- 默认值：
- 软删除：
- 审计字段：`,
	),
	newReadmeQuickTextSeed(
		"设计 API 契约",
		"写清楚前后端协作所需的请求、响应和兼容规则。",
		"设计",
		`## 设计：API 契约

接口目标：

- 调用方：
- 被调用方：
- 用户动作：

请求参数：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
|  |  |  |  |  |

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
|  |  |  |

兼容要求：

- 新字段向后兼容。
- 错误码稳定。
- 分页和排序规则明确。
- 幂等操作说明清楚。`,
	),
	newReadmeQuickTextSeed(
		"设计 权限边界",
		"定义谁能看、谁能改、危险操作如何保护。",
		"设计",
		`## 设计：权限边界

角色：

- 管理员：
- 普通用户：
- 只读用户：
- 系统任务：

权限矩阵：

| 操作 | 管理员 | 普通用户 | 只读用户 |
| --- | --- | --- | --- |
| 查看 |  |  |  |
| 新增 |  |  |  |
| 修改 |  |  |  |
| 删除 |  |  |  |

边界规则：

- 后端必须校验权限。
- 前端隐藏不等于授权。
- 危险操作记录审计日志。
- 跨项目数据必须隔离。`,
	),
	newReadmeQuickTextSeed(
		"设计 错误体验",
		"统一错误提示、恢复路径和证据收集方式。",
		"设计",
		`## 设计：错误体验

错误分类：

- 用户输入错误：
- 网络或服务错误：
- 权限错误：
- 数据冲突：
- 未知错误：

提示原则：

1. 用用户能理解的话说明问题。
2. 给出下一步动作。
3. 保留用户已输入内容。
4. 技术细节进入日志或展开详情。

恢复路径：

- 重试：
- 返回：
- 保存草稿：
- 联系维护者：`,
	),
	newReadmeQuickTextSeed(
		"设计 性能策略",
		"规划列表、搜索、缓存和大数据量下的体验。",
		"设计",
		`## 设计：性能策略

性能风险：

- 首屏数据过大：
- 列表渲染过多：
- 搜索触发过频：
- 接口响应不稳定：

策略：

1. 分页或虚拟滚动。
2. 搜索防抖。
3. 结果缓存。
4. 增量加载。
5. 后台任务异步处理。

验收指标：

- 首屏时间：
- 操作响应：
- 大数据量条数：
- 浏览器内存：
- 接口耗时：`,
	),
	newReadmeQuickTextSeed(
		"设计 可观测性",
		"为功能设计补充日志、指标和排障证据。",
		"设计",
		`## 设计：可观测性

需要记录：

- 用户操作：
- 请求参数摘要：
- 关键状态变化：
- 外部依赖耗时：
- 错误堆栈：

指标：

- 成功率：
- 响应时间：
- 使用次数：
- 失败原因分布：

排障证据：

- trace id：
- 用户 id：
- 项目或资源 id：
- 时间范围：
- 前端截图或控制台日志：`,
	),
	newReadmeQuickTextSeed(
		"测试 验收矩阵",
		"把功能验收拆成路径、数据、预期和证据。",
		"测试",
		`## 测试：验收矩阵

| 场景 | 前置条件 | 操作 | 预期结果 | 证据 |
| --- | --- | --- | --- | --- |
| 主路径 |  |  |  |  |
| 空数据 |  |  |  |  |
| 异常输入 |  |  |  |  |
| 权限不足 |  |  |  |  |
| 数据量大 |  |  |  |  |

通过标准：

- 主路径可稳定复现。
- 失败路径提示清楚。
- 数据写入和回读一致。
- 验证证据可复查。`,
	),
	newReadmeQuickTextSeed(
		"测试 冒烟路径",
		"快速判断服务和核心页面是否可用。",
		"测试",
		`## 测试：冒烟路径

冒烟前置：

- 服务已启动：
- 数据库可访问：
- 页面资源可加载：

检查项：

1. 打开入口页面。
2. 完成登录或基础鉴权。
3. 查询核心列表。
4. 打开详情或编辑页。
5. 执行一次保存或提交。
6. 刷新后确认数据仍存在。

失败处理：

- 保存截图。
- 保存接口响应。
- 保存服务日志。
- 记录最小复现步骤。`,
	),
	newReadmeQuickTextSeed(
		"测试 回归清单",
		"功能改动后确认历史能力没有被破坏。",
		"测试",
		`## 测试：回归清单

回归范围：

- 本次直接修改的功能。
- 共享组件影响的页面。
- 相同接口的其他调用方。
- 历史高频问题路径。

检查顺序：

1. 主路径。
2. 历史 bug 路径。
3. 边界数据。
4. 权限差异。
5. 保存后刷新。

输出：

- 通过项：
- 失败项：
- 未覆盖项：
- 风险说明：`,
	),
	newReadmeQuickTextSeed(
		"测试 边界条件",
		"系统性列出边界输入和异常场景。",
		"测试",
		`## 测试：边界条件

输入边界：

- 空值：
- 超长文本：
- 特殊字符：
- 重复值：
- 非法格式：

状态边界：

- 未登录：
- 无权限：
- 数据已删除：
- 并发修改：
- 网络超时：

数据量边界：

- 0 条：
- 1 条：
- 100 条：
- 1000 条：

预期：

- 不崩溃。
- 有提示。
- 数据不丢。
- 日志可追踪。`,
	),
	newReadmeQuickTextSeed(
		"测试 数据准备",
		"规范测试数据来源、隔离和清理方式。",
		"测试",
		`## 测试：数据准备

测试数据：

- 项目：
- 用户：
- 文件：
- 配置：
- 外部依赖：

准备方式：

1. 优先使用脚本创建。
2. 标记测试数据来源。
3. 避免污染生产或共享样例。
4. 测试结束后清理或复原。

数据校验：

- 创建成功：
- 查询可见：
- 权限正确：
- 清理完成：`,
	),
	newReadmeQuickTextSeed(
		"测试 无头浏览器",
		"用真实浏览器路径验证页面行为和控制台错误。",
		"测试",
		`## 测试：无头浏览器

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
- 无前端错误和失败请求。`,
	),
	newReadmeQuickTextSeed(
		"测试 接口契约",
		"验证接口参数、响应结构、错误码和数据副作用。",
		"测试",
		`## 测试：接口契约

接口：

- 服务名：
- 方法：
- 路径：

请求用例：

- 最小参数：
- 完整参数：
- 缺少必填：
- 非法类型：
- 越权访问：

响应断言：

- code：
- data：
- msg：
- 分页字段：
- 审计字段：

副作用断言：

- 数据库写入：
- 使用次数：
- 日志记录：
- 缓存刷新：`,
	),
	newReadmeQuickTextSeed(
		"测试 性能基线",
		"为列表、搜索和大数据量操作建立可比较指标。",
		"测试",
		`## 测试：性能基线

测试对象：

- 页面：
- 接口：
- 数据量：
- 浏览器：

指标：

| 指标 | 目标 | 实际 | 说明 |
| --- | --- | --- | --- |
| 首屏加载 |  |  |  |
| 搜索响应 |  |  |  |
| 滚动流畅度 |  |  |  |
| 保存耗时 |  |  |  |

结论：

- 是否达标：
- 瓶颈位置：
- 优化建议：`,
	),
	newReadmeQuickTextSeed(
		"测试 失败复现",
		"统一 bug 记录，让修复和复验都有证据。",
		"测试",
		`## 测试：失败复现

问题摘要：

- 页面或接口：
- 发生时间：
- 影响范围：

复现步骤：

1. 
2. 
3. 

实际结果：

- 

预期结果：

- 

证据：

- 截图：
- 请求：
- 日志：
- 数据库记录：

修复后复验：

- 复验人：
- 复验时间：
- 复验结果：`,
	),
	newReadmeQuickTextSeed(
		"测试 发布验证",
		"发布后确认服务、页面、数据和回滚预案可用。",
		"测试",
		`## 测试：发布验证

发布前：

- 构建产物已生成。
- 配置差异已确认。
- 数据变更有备份或回滚方案。
- 冒烟脚本已准备。

发布后：

1. 健康检查通过。
2. 核心页面可打开。
3. 核心接口返回正常。
4. 关键数据可读写。
5. 日志无新增错误。

回滚确认：

- 回滚入口：
- 回滚耗时：
- 回滚验证：`,
	),
}
