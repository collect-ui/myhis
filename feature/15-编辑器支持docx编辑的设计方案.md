# 要求
- 目前支持docx 的预览，我需要支持编辑
- http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool

- 结合本项目 网上考一个开源组件实现案例，
- 给出一个对比，和实现原理，我需要一个go 版本docx 编辑方案
- 能在http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool 支持
- 方案和对比写在本文
实现思路写在本文

# 只写设计，设计写在本文，追加
- 前端怎么实现
    - 前端如何先预览+编辑
    - 怎么实现解析
- 后端怎么实现
    - 后端怎么实现保存
    - 后端算法逻辑是什么

---

# Webshell Editor Pool 支持 DOCX 编辑设计方案

## 1. 目标与结论

在 `http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool` 中，让 `.docx` 文件从“只读预览”升级为“预览 + 编辑 + 保存回远端工作区文件”。

本设计只覆盖 DOCX，不扩展 XLSX/PPTX；保存目标仍是当前 webshell 工作区项目对应的远端文件路径。

推荐方案：

1. 前端使用 `SuperDoc` 作为 DOCX 编辑内核，保留现有 `docx-preview` 作为只读预览和降级渲染。
2. 后端使用当前 Go + SFTP 文件链路扩展二进制读取/保存能力，新增 `content_base64` 保存分支，不能复用当前文本 `content` 保存分支。
3. 服务端不做完整 WYSIWYG 排版编辑，只负责路径校验、DOCX 文件校验、冲突检测、备份、原子写入和元数据返回。

原因：

1. 当前项目已经在 `/data/project/sport-ui/src/components/workspace-editor-pool-core.tsx` 中把 `.docx` 识别为 `renderKind=docx`，并通过 `/data/project/sport-ui/src/components/docx-preview.tsx` 使用 `docx-preview` 渲染 base64 内容。
2. 当前后端 `plugins/module_workspace_file_access.go` 的 `workspace_file_content` 读取 DOCX 时返回 `content_base64`，但 `workspace_file_save` 只按字符串 `content` 写入，不能安全保存 DOCX 二进制。
3. Go 适合做 DOCX 文件存储、校验、模板化或局部结构修改；浏览器内所见即所得 DOCX 编辑更适合由成熟 JS 文档编辑内核承担。

## 2. 开源组件调研与对比

调研时间：2026-05-14。

| 方案 | 能力 | 集成形态 | 优点 | 主要风险 | 结论 |
| --- | --- | --- | --- | --- | --- |
| SuperDoc | 浏览器内渲染和编辑 DOCX，支持 React/Vue/原生 JS；基于 OOXML、ProseMirror、Yjs、JSZip | 前端组件嵌入，Go 后端只保存二进制 | 不需要单独 Office 服务；适合嵌入现有 `workspace-editor-pool`；和当前 React/sport-ui 改造边界匹配 | AGPLv3/商业双许可；复杂 Word 特性需要实测 | 推荐作为本项目 MVP |
| ONLYOFFICE Docs | 完整在线 Office，支持 DOCX 编辑、协作、评论、审阅 | 独立 DocumentServer + iframe/JS API 或 WOPI | 格式兼容和协作能力强 | 需要部署独立服务；回调保存、JWT、内网可访问 URL、许可证/并发限制要治理 | 作为企业级二期方案 |
| Collabora Online | LibreOffice 技术栈在线编辑，支持 WOPI | 独立服务 + WOPI Host | 开源、能力完整、适合私有化 | Go 侧要实现 WOPI：CheckFileInfo/GetFile/PutFile/Lock/Unlock 等；接入重 | 不作为 MVP |
| docx-preview | DOCX 转 DOM 只读预览 | 已在项目中使用 | 轻量，当前已跑通 | 不支持编辑和保存 DOCX | 保留为预览/降级 |
| Mammoth.js | DOCX 转 HTML/文本 | 前端或 Node 转换 | 适合内容抽取 | 不是 DOCX 编辑器；HTML 回写 DOCX 会丢版式；官方提醒需处理安全风险 | 不用于编辑 |
| Godocx | Go 纯库创建/修改 DOCX | Go 服务端结构化处理 | 可做模板、段落、表格、图片等服务端局部修改 | 不提供浏览器所见即所得编辑 UI | 可作为后端二期“模板/批量修改”工具 |
| UniOffice | Go 读写编辑 DOCX/XLSX/PPTX | Go 服务端库 | 能力强，支持已有文档编辑、表格、图片等 | 商业产品，需要 license key | 不作为开源首选 |

参考来源：

1. SuperDoc GitHub：`https://github.com/superdoc-dev/superdoc`
2. ONLYOFFICE DocumentServer GitHub：`https://github.com/ONLYOFFICE/DocumentServer`
3. ONLYOFFICE 保存流程：`https://api.onlyoffice.com/docs/docs-api/get-started/how-it-works/saving-file/`
4. ONLYOFFICE Callback handler：`https://api.onlyoffice.com/docs/docs-api/usage-api/callback-handler/`
5. Collabora Online：`https://www.collaboraonline.com/`
6. Collabora Online GitHub：`https://github.com/CollaboraOnline/online`
7. Microsoft WOPI CheckFileInfo：`https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/checkfileinfo/checkfileinfo-response`
8. WOPI REST API：`https://api.onlyoffice.com/docs/docs-api/using-wopi/wopi-rest-api/`
9. Mammoth.js：`https://github.com/mwilliamson/mammoth.js`
10. Godocx：`https://gomutex.github.io/godocx/`
11. UniOffice：`https://github.com/unidoc/unioffice`

## 3. 当前项目现状

### 3.1 前端现状

入口组件：

1. `collect/frontend/page_data/data/server/webshell_editor_pool_panel_fragment.json`
2. `/data/project/sport-ui/src/components/workspace-editor-pool-core.tsx`
3. `/data/project/sport-ui/src/components/docx-preview.tsx`

当前链路：

1. 用户在左侧源码树点击文件。
2. `workspace-editor-pool` 调用 `webshell.workspace_file_content`。
3. Go 后端读取远端文件，`.docx` 返回：
   - `kind: "docx"`
   - `mime: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"`
   - `content_base64`
4. 前端 `inferRenderKind()` 判断为 `docx`。
5. UI 进入 `activeTab.renderKind === "docx"` 分支，用 `DocxPreview` 只读渲染。

当前缺口：

1. DOCX 没有“编辑/保存”按钮。
2. `saveTab()` 只允许 `isTextTab(tab)`，DOCX 不会进入保存。
3. 后端 `writeSingle()` 使用 `content := gocast.ToString(params["content"])` 后 `[]byte(content)` 写入，适合文本，不适合 DOCX 二进制。
4. `workspace_file_content.max_bytes` 默认 2MB，较复杂 DOCX 容易被截断。

### 3.2 后端现状

服务定义位置：

1. `collect/webshell/workspace_project/index.yml`
2. `webshell.workspace_file_content`
3. `webshell.workspace_file_save`

Go 实现位置：

1. `plugins/module_workspace_file_access.go`
2. `WorkspaceFileAccessService.Result`
3. `readSingle`
4. `writeSingle`

已有安全能力：

1. 根据 `project_code` 查工作区项目。
2. 解密服务器用户并建立 SSH/SFTP。
3. 使用 `normalizeWorkspaceTargetPath` 和 `ensureRemotePathChainSafe` 做路径约束。
4. 识别 DOCX 扩展和 MIME。

需要补齐：

1. 二进制读取大小上限单独配置。
2. 二进制保存参数和写入逻辑。
3. DOCX 文件结构校验。
4. 保存前冲突检测和可选备份。

## 4. 总体架构

```text
用户点击 .docx
  -> workspace-editor-pool 打开 tab
  -> webshell.workspace_file_content 读取 base64
  -> 预览模式：DocxPreview 渲染
  -> 编辑模式：DocxEditor/SuperDoc 解析 Blob 并进入 editing
  -> 用户保存
  -> 前端导出 DOCX Blob/ArrayBuffer
  -> base64 或 multipart 提交到 Go 服务
  -> Go 校验 project/path/token/sha256/docx zip
  -> SFTP 写入同目录临时文件
  -> rename 覆盖原文件
  -> 返回新 sha256/size/modify_time
  -> 前端刷新 tab 状态和预览
```

核心原则：

1. 前端负责 OOXML 解析、编辑 UI 和导出 DOCX。
2. 后端只处理文件级二进制，避免在 Go 中重建完整 Word 排版引擎。
3. 保存必须走二进制接口，不能把 DOCX 当 UTF-8 文本。
4. 预览和编辑分离：预览失败不影响下载/保存；编辑内核加载失败时回退只读预览。

## 5. 前端设计

### 5.1 新增组件

新增文件：

1. `/data/project/sport-ui/src/components/docx-editor.tsx`

职责：

1. 接收 `base64 | Blob | ArrayBuffer`。
2. 转为 DOCX Blob。
3. 懒加载 SuperDoc，初始化编辑器。
4. 暴露 `save()`、`isDirty()`、`dispose()` 方法给 `workspace-editor-pool-core.tsx`。
5. 监听编辑状态，更新当前 tab 的 `dirty/saveMessage`。

建议组件属性：

```ts
type DocxEditorProps = {
  token: string;
  path: string;
  title: string;
  base64: string;
  mime?: string;
  mode: "preview" | "edit";
  onDirtyChange?: (dirty: boolean) => void;
  onSaveBinary?: (payload: { contentBase64: string; size: number }) => Promise<void>;
};
```

### 5.2 注册与渲染入口

如需作为低代码组件注册：

1. `/data/project/sport-ui/src/main.tsx` 新增 `setRegister("docx-editor", DocxEditor)`。

本期更推荐先在 `workspace-editor-pool-core.tsx` 内部直接使用，原因是 DOCX tab 状态、保存快捷键、toolbar、split panel 都在该组件内部维护，直接改造边界更小。

改造位置：

1. `/data/project/sport-ui/src/components/workspace-editor-pool-core.tsx`
2. `activeTab.renderKind === "docx"` 分支
3. 顶部 toolbar 的保存、刷新、预览/编辑按钮区域
4. `saveTab()` / `saveActiveFile()` 增加 DOCX 分支

### 5.3 交互设计

DOCX tab 顶部按钮：

1. `预览`：使用现有 `DocxPreview`。
2. `编辑`：切换到 SuperDoc 编辑器。
3. `保存`：仅编辑模式且 dirty 时可用。
4. `刷新`：重新读取远端文件；dirty 时先确认。
5. `下载`：可选，直接下载当前编辑器导出的 Blob 或远端原文件。

默认行为：

1. 初次打开 `.docx` 默认进入预览模式，保证大文件打开更稳。
2. 用户点击 `编辑` 后才加载 SuperDoc，避免影响普通源码编辑首屏。
3. 如果文件超过 `DOCX_EDIT_MAX_BYTES`，只允许预览/下载，不允许在线编辑。
4. 编辑失败时保留 `DocxPreview` 和错误提示。

### 5.4 前端如何先预览再编辑

打开文件时仍走现有读取链路：

```text
webshell.workspace_file_content
  -> content_base64
  -> activeTab.contentBase64
  -> mode=preview
  -> DocxPreview(base64)
```

用户点击编辑：

```text
contentBase64
  -> atob
  -> Uint8Array
  -> Blob(application/vnd.openxmlformats-officedocument.wordprocessingml.document)
  -> SuperDoc(document=Blob, documentMode=editing)
```

切回预览：

1. 如果没有修改，继续使用原 `contentBase64`。
2. 如果有修改但未保存，优先使用编辑器导出的临时 Blob 生成预览。
3. 如果编辑器暂不支持实时导出预览，则提示“预览为上次保存版本”，保存后自动刷新预览。

### 5.5 前端怎么实现解析

DOCX 本质是 ZIP 包，核心内容在 `word/document.xml`、样式在 `word/styles.xml`，图片和关系文件在 `word/media/*`、`word/_rels/*`。

前端不手写 OOXML 解析器，交给 SuperDoc：

1. JSZip 解包 DOCX。
2. 解析 OOXML 为内部文档模型。
3. 使用 ProseMirror 承载编辑状态。
4. 可选使用 Yjs 做协作状态。
5. 保存时把编辑后的模型重新打包为 DOCX Blob。

项目侧只做轻量预检：

1. base64 是否为空。
2. Blob MIME 是否为 DOCX。
3. 文件大小是否超过在线编辑限制。
4. SuperDoc 加载失败时回退 `DocxPreview`。

### 5.6 保存状态模型

`OpenTab` 建议新增字段：

```ts
type OpenTab = {
  docxEditMode?: "preview" | "edit";
  docxDirty?: boolean;
  docxBaseSha256?: string;
  docxSaving?: boolean;
  docxSaveError?: string;
};
```

读取 DOCX 时后端返回 `sha256`，前端记录为 `docxBaseSha256`。保存时回传该值，用于后端判断远端文件是否已被其他人改过。

### 5.7 快捷键

`Ctrl+S` / `Command+S`：

1. 文本 tab：沿用现有 `saveActiveFile()`。
2. DOCX tab：调用 `saveDocxTab(activeToken)`。
3. HTTP tab：沿用现有 HTTP 文档保存。

### 5.8 前端伪代码

```ts
async function saveDocxTab(token: string) {
  const tab = getTabByToken(token);
  if (!tab || tab.renderKind !== "docx") return;

  const blob = await docxEditorRefs.current[token].exportBlob();
  const contentBase64 = await blobToBase64(blob);

  await postJson("webshell.workspace_file_binary_save", {
    project_code: tab.projectCode,
    path: tab.path,
    content_base64: contentBase64,
    base_sha256: tab.docxBaseSha256,
    mime: tab.mime,
  });

  setOpenTab(token, {
    contentBase64,
    docxDirty: false,
    saveMessage: "saved",
  });
}
```

实际实现时，`exportBlob()` 需要按 SuperDoc 当前版本 API 封装在 `docx-editor.tsx` 内，外层不直接依赖第三方实例细节。

## 6. 后端设计

### 6.1 服务定义

建议在 `collect/webshell/workspace_project/index.yml` 新增两个服务，继续复用 `workspace_file_access` module：

```yml
- key: workspace_file_binary_content
  module: workspace_file_access
  http: true
  params:
    operation:
      default: "binary_content"
    project_code:
      check:
        template: "{{must .project_code}}"
        err_msg: 项目编码不能为空
    path:
      check:
        template: "{{must .path}}"
        err_msg: 文件路径不能为空
    max_bytes:
      default: 20971520

- key: workspace_file_binary_save
  module: workspace_file_access
  http: true
  params:
    operation:
      default: "save_binary"
    project_code:
      check:
        template: "{{must .project_code}}"
        err_msg: 项目编码不能为空
    path:
      check:
        template: "{{must .path}}"
        err_msg: 文件路径不能为空
    content_base64:
      check:
        template: "{{must .content_base64}}"
        err_msg: 文件内容不能为空
    base_sha256:
      default: ""
    mime:
      default: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    max_write_bytes:
      default: 20971520
```

也可以不新增 `workspace_file_binary_content`，而是扩展现有 `workspace_file_content.max_bytes`。但单独新增更清晰，避免影响文本编辑器读取限制。

### 6.2 Go 实现位置

修改文件：

1. `plugins/module_workspace_file_access.go`

新增 operation：

1. `binary_content`
2. `save_binary`

`Result()` 分发：

```go
switch operation {
case "content":
    return s.readSingle(params, sftpClient, projectRoot)
case "binary_content":
    return s.readBinary(params, sftpClient, projectRoot)
case "content_batch":
    return s.readBatch(params, sftpClient, projectRoot)
case "save":
    return s.writeSingle(params, sftpClient, projectRoot)
case "save_binary":
    return s.writeBinary(params, sftpClient, projectRoot)
}
```

### 6.3 后端如何保存

保存接口输入：

```json
{
  "project_code": "backend",
  "path": "/data/project/example/docs/a.docx",
  "content_base64": "UEsDB...",
  "base_sha256": "打开文件时的sha256",
  "mime": "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
}
```

保存流程：

1. 校验 `project_code/path/content_base64`。
2. 根据 `project_code` 解析项目根目录和服务器用户。
3. 通过 `normalizeWorkspaceTargetPath` 保证目标路径仍在项目根目录下。
4. 只允许 `.docx` 扩展，拒绝 `.doc/.xls/.ppt` 等其他格式。
5. base64 解码为 `[]byte`，校验大小不超过 `max_write_bytes`。
6. 校验 ZIP magic：文件头应为 `PK`。
7. 使用 `archive/zip` 校验 DOCX 结构，至少存在：
   - `[Content_Types].xml`
   - `word/document.xml`
8. 计算新内容 `sha256`。
9. 如请求带 `base_sha256`，读取远端当前文件计算 sha256；不一致则返回冲突错误，提示用户刷新或另存。
10. 可选备份原文件到同目录或项目 `.webshell-backup/docx/YYYYMMDD/`。
11. 写入同目录临时文件：`.<filename>.tmp.<uuid>`。
12. 关闭临时文件后，使用 SFTP `Rename(temp, target)` 覆盖。
13. 返回保存后的 `path/size/sha256/modify_time/success`。

### 6.4 后端算法逻辑

核心算法是“二进制安全写入 + DOCX 结构校验 + 乐观锁冲突检测”。

伪代码：

```go
func writeBinary(params, sftpClient, projectRoot) Result {
    targetPath := normalizeWorkspaceTargetPath(projectRoot, params["path"])
    ensureRemotePathChainSafe(sftpClient, projectRoot, targetPath, true)

    if ext(targetPath) != ".docx" {
        return NotOk("仅支持保存docx文件")
    }

    raw := normalizeBase64(params["content_base64"])
    data := base64.StdEncoding.DecodeString(raw)
    if len(data) == 0 || len(data) > maxWriteBytes {
        return NotOk("文件大小超过限制")
    }

    if !isValidDocx(data) {
        return NotOk("DOCX结构校验失败")
    }

    currentHash := sha256RemoteFile(sftpClient, targetPath)
    if baseSha256 != "" && currentHash != "" && baseSha256 != currentHash {
        return NotOk("远端文件已变更，请刷新后再保存")
    }

    backupRemoteFileIfNeeded(sftpClient, targetPath)

    tmpPath := path.Join(path.Dir(targetPath), "."+path.Base(targetPath)+".tmp."+uuid)
    writeAll(tmpPath, data)
    rename(tmpPath, targetPath)

    return Ok({path, size, sha256, success:true})
}
```

`isValidDocx(data)`：

1. `zip.NewReader(bytes.NewReader(data), int64(len(data)))`
2. 遍历 zip entry。
3. 记录是否存在 `[Content_Types].xml` 和 `word/document.xml`。
4. 累加未压缩大小，超过上限直接拒绝，防止 zip bomb。
5. 禁止绝对路径、`../` 路径 entry。

### 6.5 是否需要 Go DOCX 编辑库

MVP 不需要 Go 侧编辑 DOCX 内容。Go 侧只保存前端导出的完整 DOCX 文件。

后续如需要服务端能力，可分两类：

1. 模板/批量替换：优先评估 Godocx，保持开源纯 Go。
2. 高保真复杂文档结构编辑：评估 UniOffice，但它是商业库，需要 license key。

服务端结构化修改和浏览器交互编辑要分开设计，不能把 Go 库当成在线 Word 编辑器。

## 7. ONLYOFFICE/Collabora 备选 Go 方案

如果后续需要多人协作、审阅、批注、复杂 Word 特性兼容，推荐升级为 Office 服务集成。

### 7.1 ONLYOFFICE Go 集成思路

前端：

1. 在 DOCX tab 内嵌 ONLYOFFICE iframe/editor。
2. 初始化 `DocsAPI.DocEditor`，设置：
   - `document.fileType = "docx"`
   - `document.url = Go 后端可访问的下载 URL`
   - `document.key = 文件版本 key`
   - `editorConfig.callbackUrl = Go 保存回调 URL`
   - `token = JWT`

后端 Go：

1. 提供临时下载 URL，ONLYOFFICE DocumentServer 能访问到原始 DOCX。
2. 提供 callback handler。
3. 当 callback `status=2` 或 forcesave `status=6` 时，从回调 body 的 `url` 下载编辑后的 DOCX。
4. 走与本方案相同的 `writeBinary` 校验/备份/写入逻辑。
5. 返回 `{"error":0}`。

适用场景：

1. 要完整 Office 能力。
2. 要多人协作。
3. 能接受独立服务部署和网络回调复杂度。

### 7.2 Collabora Go/WOPI 集成思路

前端：

1. DOCX tab 打开 Collabora iframe。
2. iframe URL 指向 Collabora discovery 得到的 edit action，并带 `WOPISrc` 和 `access_token`。

后端 Go 作为 WOPI Host：

1. `GET /wopi/files/{id}`：CheckFileInfo，返回文件名、大小、版本、权限、`UserCanWrite=true` 等。
2. `GET /wopi/files/{id}/contents`：返回 DOCX 二进制。
3. `POST /wopi/files/{id}/contents` + `X-WOPI-Override: PUT`：保存完整 DOCX 二进制。
4. `POST /wopi/files/{id}`：实现 Lock/Unlock/RefreshLock/Rename 等。
5. access token 绑定用户、项目、路径、过期时间。

该方案工程量明显大于 SuperDoc，适合作为二期或企业协作版本。

## 8. 安全与边界

1. 路径必须继续使用项目根目录 guard，禁止任意路径写入。
2. 保存仅允许 `.docx`。
3. 前端预览/编辑不要信任 DOCX 中的外部链接和嵌入资源。
4. 后端校验 zip entry，拒绝 zip slip 和异常膨胀。
5. base64 JSON 保存建议限制 20MB；超过该限制时二期改为 multipart 直传接口。
6. 保存前做 `base_sha256` 乐观锁，降低多人覆盖风险。
7. 临时文件必须写在目标同目录，保证 rename 尽量原子。
8. 保存失败要清理临时文件。
9. 不要把 DOCX 内容写入日志。

## 9. 分阶段实施

### 9.1 MVP

1. 前端新增 `docx-editor.tsx`，接入 SuperDoc。
2. `workspace-editor-pool-core.tsx` 增加 DOCX `预览/编辑/保存` 分支。
3. 后端新增 `workspace_file_binary_save`。
4. 后端 `workspace_file_content` 或新增 `workspace_file_binary_content` 返回 `sha256`。
5. 支持单人编辑保存，冲突时提示刷新。

### 9.2 二期

1. 增加 DOCX 大文件 multipart 保存接口。
2. 增加保存历史/版本恢复。
3. 增加服务端 Godocx 模板替换接口。
4. 增加 DOCX 编辑自动化测试样例。

### 9.3 三期

1. 评估 ONLYOFFICE 或 Collabora。
2. 实现 Go callback/WOPI Host。
3. 支持协作、批注、审阅、锁。

## 10. 验证方案

### 10.1 后端

1. `go fmt ./...`
2. `go test ./...`
3. 空参数、错误路径、非 docx、非法 base64、非法 zip、超过大小上限。
4. 正常保存后重新读取，校验 `sha256` 和 `content_base64` 变化。
5. `base_sha256` 不一致时必须拒绝覆盖。

### 10.2 前端

新增脚本：

1. `test/lowcode-page/scripts/frontend/webshell_editor_pool_docx_edit_check.js`

验证步骤：

1. 打开 `http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool`。
2. 选择一个包含 `.docx` 的工作区项目。
3. 打开 DOCX，确认默认预览可见。
4. 点击 `编辑`，确认 SuperDoc 编辑器加载。
5. 修改一段文字。
6. 点击保存。
7. 刷新文件，确认修改仍存在。
8. 检查 `consoleErrors/pageErrors/requestfailed` 为 0。

### 10.3 回归点

1. 普通文本文件保存不回归。
2. Markdown/HTML 预览不回归。
3. 图片预览不回归。
4. HTTP 控制台 tab 不回归。
5. 分屏、关闭 tab、最近文件记录不回归。

## 11. 推荐文件改动清单

前端：

1. `/data/project/sport-ui/package.json`：新增 SuperDoc 依赖。
2. `/data/project/sport-ui/src/components/docx-editor.tsx`：新增 DOCX 编辑组件。
3. `/data/project/sport-ui/src/components/workspace-editor-pool-core.tsx`：接入 DOCX 编辑和保存分支。
4. `/data/project/sport-ui/src/main.tsx`：如低代码注册需要，注册 `docx-editor`。

后端：

1. `plugins/module_workspace_file_access.go`：新增 `readBinary/writeBinary/isValidDocx/sha256` 等逻辑。
2. `collect/webshell/workspace_project/index.yml`：新增或扩展二进制读取/保存服务。
3. `plugins/a_register.go`：如果继续复用 `workspace_file_access`，无需新增注册；如果拆新模块，则注册新 module。

测试：

1. `test/lowcode-page/scripts/frontend/webshell_editor_pool_docx_edit_check.js`
2. `test/lowcode-page/results/latest/http-proxy-validation/` 输出报告和截图。

## 12. 最小可交付定义

本功能完成的判定标准：

1. 在 `webshell-editor-pool` 打开 `.docx` 默认展示当前只读预览。
2. 点击 `编辑` 后可进入 DOCX 编辑器。
3. 修改内容后保存，后端以二进制方式写回远端同一路径。
4. 刷新或重新打开后能看到修改后的 DOCX。
5. 保存非法 DOCX、冲突版本、超大文件时有明确错误提示。
6. `go test ./...` 通过，前端 DOCX 编辑冒烟脚本通过。

## 13. 免费与自研边界补充

### 13.1 是否可以读取开源项目代码后自己实现

可以阅读开源项目的公开文档、协议说明和标准，但不建议读取其源码后“照着写一个自己的闭源版本”。

原因：

1. 如果直接复制、翻译、改写或按源码结构重写，可能被认定为派生作品，需要遵守原项目 license。
2. SuperDoc 和 ONLYOFFICE DocumentServer 都有 AGPLv3 路线；如果把 AGPL 代码或派生实现嵌入闭源产品，会有合规风险。
3. DOCX 在线编辑器本身复杂度很高，源码级仿写也不等于能快速得到稳定实现。

可接受的方式：

1. 阅读 DOCX/OOXML 标准、WOPI 协议、ONLYOFFICE/Collabora/SuperDoc 的公开 API 文档。
2. 按公开协议自己实现 Go 后端接口，例如 WOPI Host、文件读取、保存回调、锁、版本号。
3. 使用 clean-room 方式自研：一个人只整理功能行为、接口和测试用例，不带源码细节；另一个人按这些规格重新实现。
4. 使用 license 兼容的免费开源组件，并在产品发布形态上满足对应 license。

不建议的方式：

1. 复制 SuperDoc/ONLYOFFICE/Collabora 源码片段。
2. 把 AGPL 项目的核心代码改名后放入本项目。
3. 对照源码逐行翻译为 Go/TS/React。
4. 只因为“代码在 GitHub 上”就当作无授权成本使用。

### 13.2 免费方案选择

这里的“免费”需要拆成两类：

1. **零授权费，但需要遵守强 copyleft license**：例如 SuperDoc AGPLv3、ONLYOFFICE DocumentServer AGPLv3。
2. **零授权费，且集成边界相对更干净**：例如 Collabora Online CODE + Go WOPI Host；但实现成本更高。

推荐免费路线：

| 场景 | 推荐方案 | 说明 |
| --- | --- | --- |
| 只要求不付授权费，且可以接受 AGPL 合规 | SuperDoc + Go 二进制保存 | 集成最快，适合 MVP，但需要确认 AGPL 合规要求 |
| 需要免费，并尽量降低 AGPL 嵌入风险 | Collabora Online CODE + Go WOPI Host | 需要部署独立服务，并实现 WOPI；开发量更大 |
| 需要完整 Office 能力且可接受 AGPL 服务部署 | ONLYOFFICE Community + Go callback | 免费可用，但需要独立 DocumentServer、JWT、回调和版本治理 |
| 想完全自己写 DOCX 网页编辑器 | 不推荐 | OOXML、排版、选择区、表格、图片、导出都很重，短期不可控 |

### 13.3 本项目建议结论

如果必须“免费”，本设计调整为：

1. **MVP 仍可选 SuperDoc**，前提是确认项目发布形态能满足 AGPLv3；不能复制其代码，只作为依赖使用。
2. **如果不能接受 AGPL**，改用 Collabora Online CODE，并由 Go 实现 WOPI Host；这是更合规但更重的免费方案。
3. **不要读取商业/AGPL 项目源码后自研仿写**。如果要自研，只能基于 OOXML 标准、WOPI 协议和 clean-room 规格实现。
4. **Go 后端仍按本方案负责文件级能力**：路径校验、权限、二进制读取、DOCX 结构校验、sha256 冲突检测、SFTP 原子写入。

### 13.4 参考来源

1. SuperDoc：`https://github.com/superdoc-dev/superdoc`
2. ONLYOFFICE DocumentServer：`https://github.com/ONLYOFFICE/DocumentServer`
3. Collabora Online MPLv2 说明：`https://www.collaboraonline.com/terms/collabora-online-mplv2/`
4. GNU AGPLv3 Section 13：`https://www.gnu.org/licenses/agpl-3.0.en.html#section13`

## 14. 当前最终推荐与上下文恢复记录

### 14.1 当前最终推荐

在“免费、可长期维护、避免读取 AGPL/商业源码后仿写”的约束下，当前最终推荐调整为：

**首选：Collabora Online CODE + Go WOPI Host。**

推荐原因：

1. Collabora 作为独立服务部署，`sport-ui` 不需要直接嵌入 AGPL 前端编辑器代码。
2. Go 后端只实现 WOPI 标准协议和当前 webshell 工作区文件读写，不需要自研完整 DOCX 排版/编辑引擎。
3. DOCX 编辑能力来自 LibreOffice/Collabora 技术栈，表格、图片、样式、页眉页脚等能力比自研轻量编辑器稳。
4. 和当前项目已有 SFTP 工作区文件能力匹配：Go 负责鉴权、路径约束、读取、保存、锁、版本号。

推荐顺序：

1. **首选**：Collabora Online CODE + Go WOPI Host。
2. **快速但需 AGPL 合规确认**：SuperDoc 依赖方式集成。
3. **企业增强备选**：ONLYOFFICE Community + Go callback。
4. **不推荐**：读取 SuperDoc/ONLYOFFICE/Collabora 源码后仿写 DOCX 编辑器。

### 14.2 下一次快速恢复上下文

如果下次继续开发，先看这一节即可恢复上下文：

1. 目标页面：`http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool`
2. 当前 DOCX 预览组件：`/data/project/sport-ui/src/components/docx-preview.tsx`
3. 当前 editor pool 核心：`/data/project/sport-ui/src/components/workspace-editor-pool-core.tsx`
4. 当前后端入口：`main.go`
5. 当前 webshell 文件访问模块：`plugins/module_workspace_file_access.go`
6. 当前工作区路径 guard：`plugins/workspace_path_guard.go`
7. 当前项目/服务器/SFTP helper：`plugins/module_workspace_content_search.go`
8. 本次选择：先实现 Go WOPI Host 后端，再接 Collabora iframe 前端。

### 14.3 WOPI 实现目标

Go 后端新增路由：

1. `POST /wopi/token`：根据 `project_code/path` 生成 `file_id/access_token/wopi_src`。
2. `GET /wopi/files/:file_id`：CheckFileInfo。
3. `GET /wopi/files/:file_id/contents`：读取 DOCX 二进制。
4. `POST /wopi/files/:file_id/contents` + `X-WOPI-Override: PUT`：保存 DOCX 二进制。
5. `POST /wopi/files/:file_id` + `X-WOPI-Override: LOCK/GET_LOCK/REFRESH_LOCK/UNLOCK`：基础锁。

token 设计：

1. `access_token` 是 HMAC 签名的 base64url JSON。
2. payload 包含 `file_id/project_code/path/exp/user_id`。
3. WOPI 路由用 `access_token` 解出真实工作区文件。
4. `file_id` 必须和 token 内 `file_id` 一致。

文件能力设计：

1. 只允许 `.docx`。
2. 读取时走当前项目配置的 SSH/SFTP。
3. 保存时校验 DOCX ZIP 结构。
4. 保存时写同目录临时文件，再 rename 覆盖。
5. 锁先用内存锁实现，满足单进程开发环境；多实例部署时再替换为数据库/Redis 锁。

### 14.4 实施日志

每次开发都必须在这里记录“做了什么、改了哪些文件、验证结果、未完成项”。

#### 2026-05-15 记录 1

已完成：

1. 明确最终免费首选方案为 `Collabora Online CODE + Go WOPI Host`。
2. 明确不走“读取 AGPL/商业源码后仿写”的路线。
3. 明确先做 Go 后端 WOPI Host，再接前端 Collabora iframe。

计划立即实施：

1. 新增 `plugins/wopi_docx.go`，实现 WOPI token、CheckFileInfo、GetFile、PutFile、Lock/Unlock。
2. 修改 `main.go` 注册 `/wopi/files/:file_id` 路由。
3. 使用现有工作区项目和 SFTP helper，不新增数据库表。
4. 先用内存锁，后续按需要改持久化锁。

未完成：

1. 前端尚未接 Collabora iframe。
2. 尚未部署 Collabora Online CODE。
3. 尚未提供生成 WOPI 编辑 URL 的前端按钮/API。
4. 尚未做浏览器联调。

#### 2026-05-15 记录 2

已完成：

1. 新增 `plugins/wopi_docx.go`。
2. 在 `main.go` 注册 `plugins.RegisterWOPIRoutes(r)`。
3. 扩展 CORS 允许头：
   - `X-WOPI-Override`
   - `X-WOPI-Lock`
   - `X-WOPI-OldLock`
   - `X-WOPI-RequestedName`
4. 实现 `POST /wopi/token`：
   - 登录态存在时读取 session 用户。
   - 校验 `project_code/path`。
   - 校验路径在工作区项目根目录下。
   - 仅允许 `.docx`。
   - 返回 `file_id/access_token/wopi_src/collabora_base_url`。
5. 实现 WOPI `CheckFileInfo`：
   - 路由：`GET /wopi/files/:file_id`
   - 返回 `BaseFileName/OwnerId/Size/UserId/Version/UserCanWrite/SupportsLocks/SupportsUpdate` 等。
6. 实现 WOPI 文件读取：
   - 路由：`GET /wopi/files/:file_id/contents`
   - 通过现有项目、服务器用户、SSH/SFTP 配置读取远端 DOCX 二进制。
7. 实现 WOPI 文件保存：
   - 路由：`POST /wopi/files/:file_id/contents`
   - 要求 `X-WOPI-Override: PUT`。
   - 校验 DOCX ZIP 结构，要求存在 `[Content_Types].xml` 和 `word/document.xml`。
   - 使用同目录临时文件写入，然后 `PosixRename`，失败时回退 `Rename`。
8. 实现基础 WOPI 锁：
   - `LOCK`
   - `GET_LOCK`
   - `REFRESH_LOCK`
   - `UNLOCK`
   - 当前为进程内内存锁，过期时间 30 分钟。
9. 实现 HMAC `access_token`：
   - payload 包含 `file_id/project_code/path/user_id/exp`。
   - 默认有效期 2 小时，最大 24 小时。
   - `wopi_secret` 可从配置读取；未配置时回退 `company_key`，再回退开发默认值。

验证结果：

1. 已执行 `gofmt -w main.go plugins/wopi_docx.go`。
2. 已执行 `go test ./... -run '^$'`，通过，用于验证所有包编译。
3. 已执行 `go test ./plugins`，通过。
4. 已执行完整 `go test ./...`，未通过，失败点是已有真实 OpenAI 集成测试触发 `429 usage limit`：
   - `TestAgentRuntimeSessionRunFlowRealOpenAI`
   - `TestAgentRuntimeSessionMultiTurnRealChat`
   - 该失败与本次 WOPI 代码无关。

本次改动文件：

1. `plugins/wopi_docx.go`
2. `main.go`
3. `feature/15-编辑器支持docx编辑的设计方案.md`

重要实现细节：

1. WOPI 路由没有复用 `/template_data/data`，因为 Collabora 需要标准的 `/wopi/files/...` REST 接口。
2. WOPI 文件真实路径不放在 URL 中，URL 只暴露 `file_id`；真实 `project_code/path` 放在签名 token 中。
3. Collabora 服务端访问 WOPI 接口时不依赖浏览器 session，只依赖 `access_token`。
4. `/wopi/token` 仍依赖浏览器登录态，供 webshell 前端生成编辑链接。

下一步：

1. 在 `workspace-editor-pool-core.tsx` 的 DOCX 分支增加“在线编辑”按钮。
2. 前端调用 `POST /wopi/token` 获取 `wopi_src/access_token`。
3. 拼接 Collabora iframe URL。
4. 增加 `collabora_url` 配置说明。
5. 启动服务后，用真实 `.docx` 文件验证 CheckFileInfo/GetFile/Lock/PutFile。

未完成：

1. 尚未接入前端 Collabora iframe。
2. 尚未部署或配置 Collabora Online CODE 地址。
3. 尚未实现数据库/Redis 分布式锁。
4. 尚未做真实 Collabora 保存回调联调。
5. 尚未增加 Playwright 浏览器回归脚本。

#### 2026-05-15 记录 3

已完成：

1. 前端最小接入已完成，改动文件：
   - `/data/project/sport-ui/src/components/workspace-editor-pool-core.tsx`
2. `OpenTab` 新增 DOCX WOPI 状态字段：
   - `docxWopiVisible`
   - `docxWopiLoading`
   - `docxWopiError`
   - `docxWopiUrl`
3. 新增前端请求函数：
   - `postWOPIToken()`
   - 调用后端 `POST /wopi/token`
4. 新增前端 Collabora URL 拼接：
   - 使用后端返回的 `collabora_base_url`
   - 拼接 `/browser/dist/cool.html?WOPISrc=...&access_token=...&lang=zh-CN`
5. DOCX tab 工具栏新增：
   - `预览`
   - `在线编辑`
6. 点击 `在线编辑` 的流程：
   - 校验当前 tab 是 DOCX。
   - 请求 `/wopi/token`。
   - 生成 Collabora iframe URL。
   - 在当前 DOCX 面板内加载 iframe。
7. 未配置 `collabora_url` 时：
   - 前端提示 `未配置collabora_url`。
   - 不影响现有 `docx-preview` 只读预览。
8. 刷新 DOCX 文件时，会清空旧的 WOPI iframe 状态，避免复用过期 token。
9. 后端配置文件新增：
   - `collabora_url=`
   - `wopi_secret=`
   - `wopi_docx_max_bytes=52428800`

验证结果：

1. 已执行 `npx tsc --noEmit --skipLibCheck`，未通过。
2. TypeScript 失败点来自既有 `../collect-ui` 和 `src/dashboard-training.tsx` 类型问题，不是本次 DOCX WOPI 接入新增错误。主要错误包括：
   - `../collect-ui/src/components/render/render-child.tsx` 中 `hasFormRef/getFormRef` 类型不存在。
   - `../collect-ui/src/index.tsx` 中 `setRegister` 参数数量不匹配。
   - `src/dashboard-training.tsx` 中 `setRegister` 参数数量不匹配。
3. 本次前端未执行完整构建，因为类型检查已被既有问题阻断。

本次新增/修改文件：

1. `/data/project/sport-ui/src/components/workspace-editor-pool-core.tsx`
2. `conf/application.properties`
3. `feature/15-编辑器支持docx编辑的设计方案.md`

下一步：

1. 配置并启动 Collabora Online CODE，例如将 `collabora_url` 指向 `http://127.0.0.1:9980` 或实际部署地址。
2. 启动 sport 后端，登录 webshell 页面。
3. 打开真实 `.docx` 文件，点击 `在线编辑`。
4. 检查 `/wopi/token`、`/wopi/files/:file_id`、`/wopi/files/:file_id/contents` 请求是否按预期触发。
5. 如果 iframe 无法加载，优先检查：
   - Collabora 是否允许当前 WOPI Host 域名。
   - `wopi_src` 是否能被 Collabora 服务访问。
   - `access_token` 是否过期。
   - `collabora_url` 是否是浏览器可访问地址。

仍未完成：

1. 尚未部署 Collabora Online CODE。
2. 尚未做真实 Collabora iframe 保存联调。
3. 尚未实现数据库/Redis 分布式锁。
4. 尚未增加 Playwright 浏览器回归脚本。
5. 尚未处理 `npx tsc` 的既有 collect-ui 类型错误。

#### 2026-05-15 记录 4

已完成：

1. 按仓库约定重启本地服务：
   - `./linux-shutdown`
   - `ss -ltnp | rg ':8015' || true`
   - `./linux-startup`
   - `ss -ltnp | rg ':8015'`
   - `curl --noproxy '*' -sS -m 5 -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8015/`
2. 服务已启动，监听进程：
   - `run-dev-main`
   - pid：`594283`
   - 端口：`8015`
3. 健康检查结果：
   - `GET http://127.0.0.1:8015/` 返回 `301`，符合当前根路由重定向到 `/collect-ui` 的行为。
4. `run-dev.log` 已确认 WOPI 路由注册成功：
   - `POST /wopi/token`
   - `GET /wopi/files/:file_id`
   - `POST /wopi/files/:file_id`
   - `GET /wopi/files/:file_id/contents`
   - `POST /wopi/files/:file_id/contents`

当前可访问入口：

1. Webshell 页面：`http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool`
2. 本地根地址：`http://127.0.0.1:8015/`

仍未完成：

1. 尚未配置 `collabora_url`。
2. 尚未启动 Collabora Online CODE。
3. 尚未用真实 DOCX 进行 iframe 编辑和保存联调。

#### 2026-05-15 记录 5

已完成：

1. 新增后端单元测试文件：
   - `plugins/wopi_docx_test.go`
2. 覆盖 WOPI/DOCX 后端关键纯逻辑：
   - 最小合法 DOCX ZIP 结构可以通过 `validateDocxBytes`
   - 缺少 `word/document.xml` 时必须拒绝
   - ZIP entry 包含 `../` 时必须拒绝，防止 zip slip
   - 非 ZIP 内容必须拒绝
   - `access_token` 签名后可以正常解析
   - 已过期 `access_token` 必须拒绝
   - `makeWOPIFileID` 对同一 `project_code/path` 保持稳定，且不同项目编码隔离
3. 已核对当前前端接入状态：
   - `workspace-editor-pool-core.tsx` 已有 `postWOPIToken`
   - 已有 `buildCollaboraWOPIUrl`
   - DOCX toolbar 已有 `预览/在线编辑`
   - DOCX 面板已能在 `docxWopiVisible && docxWopiUrl` 时加载 iframe
4. 已核对当前后端接入状态：
   - `plugins/wopi_docx.go` 已注册 `/wopi/token`
   - 已实现 WOPI `CheckFileInfo/GetFile/PutFile/LOCK/GET_LOCK/REFRESH_LOCK/UNLOCK`
   - 保存时已做 DOCX ZIP 结构校验和同目录临时文件 rename
5. 已核对 Collabora 官方社区文档：
   - CODE 适合快速试用和自托管测试
   - 官方建议在 CODE 前面放反向代理
   - 集成侧需要把 Collabora URL 配给 WOPI 应用
   - 参考：`https://collaboraonline.github.io/docs/`

验证结果：

1. 已执行 `gofmt -w plugins/wopi_docx_test.go`。
2. 已执行 `go test ./plugins`，通过：
   - `ok moon/plugins 0.019s`
3. 已执行 `go test ./... -run '^$'`，通过：
   - `moon` 编译通过
   - `moon/model/*` 编译通过
   - `moon/plugins` 编译通过
   - 该命令只做全包编译/空跑，不触发已有真实 OpenAI 集成测试。
4. 已用签名 token 对本地服务做 WOPI 只读联通验证：
   - `GET /wopi/files/:file_id` 返回 `200`
   - 样例文件：`/data/project/test/张治-1-优化.docx`
   - 返回 `BaseFileName=张治-1-优化.docx`
   - 返回 `Size=43482`
   - 返回 `UserCanWrite=true`
5. 已验证 WOPI 文件内容读取：
   - `GET /wopi/files/:file_id/contents` 返回 `200`
   - `Content-Type=application/vnd.openxmlformats-officedocument.wordprocessingml.document`
   - 文件头为 `PK`
6. 已验证 WOPI 锁接口只读流程：
   - `LOCK` 返回 `200`
   - `GET_LOCK` 返回刚写入的 lock 值
   - `REFRESH_LOCK` 返回 `200`
   - `UNLOCK` 返回 `200`

本次新增/修改文件：

1. `plugins/wopi_docx_test.go`
2. `feature/15-编辑器支持docx编辑的设计方案.md`

下一步建议按这个顺序继续：

1. 配置 `conf/application.properties`：
   - `collabora_url=http://127.0.0.1:9980` 或实际 Collabora 访问地址
   - `wopi_secret=<生产随机长字符串>`
2. 启动 Collabora Online CODE，并允许当前 WOPI Host：
   - 浏览器访问的 sport 地址：`http://192.168.232.130:8015`
   - Collabora 容器/服务端必须能访问 `wopi_src` 返回的地址
3. 做真实联调：
   - 打开 `.docx`
   - 点击 `在线编辑`
   - 确认 iframe 加载
   - 修改并保存
   - 重新打开文件确认内容变化
4. 联调通过后再补 Playwright 冒烟脚本：
   - 无 `collabora_url`：点击 `在线编辑` 时应展示明确错误，不影响预览
   - 有 `collabora_url`：iframe URL 应包含 `WOPISrc` 和 `access_token`
   - 完整保存联调只能在 Collabora 服务可用时执行

仍未完成：

1. 尚未部署或启动 Collabora Online CODE。
2. 尚未配置非空 `collabora_url`。
3. 尚未做真实 iframe 编辑保存联调。
4. 尚未实现数据库/Redis 分布式锁。
5. 尚未补前端 Playwright 回归脚本。
6. 尚未对 `PUT /wopi/files/:file_id/contents` 做真实写回验证；为避免修改现有样例 DOCX，本次只验证读取和锁。


