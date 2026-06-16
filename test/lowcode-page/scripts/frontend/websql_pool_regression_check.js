#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");

let playwright;
try {
  playwright = require("playwright");
} catch (_error) {
  playwright = require("/data/project/sport-ui/node_modules/playwright");
}

const { chromium } = playwright;

const BASE_URL = process.env.WEBSQL_BASE_URL || "http://127.0.0.1:8015";
const STANDALONE_URL =
  process.env.WEBSQL_STANDALONE_URL ||
  `${BASE_URL}/collect-ui#/collect-ui/framework/websql-pool`;
const WEBSHELL_URL =
  process.env.WEBSQL_WEBSHELL_URL ||
  `${BASE_URL}/collect-ui#/collect-ui/framework/webshell-editor-pool`;
const API_URL = process.env.WEBSQL_API_URL || `${BASE_URL}/template_data/data`;
const FAVORITE_PROJECT_CODE = process.env.WEBSQL_FAVORITE_PROJECT_CODE || "standalone";
const OUT_DIR =
  process.env.WEBSQL_OUTPUT_DIR ||
  "/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation";

const QUERY_SQL =
  "WITH RECURSIVE cnt(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM cnt WHERE x<60) SELECT x AS n FROM cnt ORDER BY x;";
const EMPTY_QUERY_SQL = "SELECT 1 AS n WHERE 1=0;";
const MANUAL_SQL = "CREATE TEMP TABLE websql_ui_tmp(id INTEGER);";

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function assertCheck(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function runCurl(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  const res = spawnSync(
    "curl",
    [
      "--noproxy",
      "*",
      "-sS",
      "-m",
      "30",
      `${API_URL}?service=${service}`,
      "-H",
      "Content-Type: application/json",
      "--data",
      payload,
    ],
    { encoding: "utf8" }
  );
  if (res.status !== 0) {
    throw new Error(res.stderr || `curl failed: ${service}`);
  }
  let out = {};
  try {
    out = JSON.parse(String(res.stdout || "{}"));
  } catch (error) {
    throw new Error(`parse response failed (${service}): ${error.message}`);
  }
  if (!out || String(out.code || "") !== "0" || out.success === false) {
    throw new Error(`${service} failed: ${out?.msg || "unknown error"}`);
  }
  return out;
}

function websql(data) {
  return runCurl("webshell.websql_execute", data).data || {};
}

function websqlFavoriteQuery(projectCode = FAVORITE_PROJECT_CODE) {
  return runCurl("webshell.websql_favorite_query", {
    project_code: projectCode,
    pagination: false,
  }).data || [];
}

function websqlRecentQuery(projectCode = FAVORITE_PROJECT_CODE) {
  return (
    runCurl("webshell.websql_recent_query", {
      project_code: projectCode,
      size: 20,
    }).data || []
  );
}

function cleanWebSQLFavoriteRows(predicate, projectCode = FAVORITE_PROJECT_CODE) {
  const rows = websqlFavoriteQuery(projectCode);
  const ids = rows
    .filter(predicate)
    .map((item) => String(item.websql_favorite_id || item.id || ""))
    .filter(Boolean);
  if (ids.length > 0) {
    runCurl("webshell.websql_favorite_delete", {
      project_code: projectCode,
      websql_favorite_id_list: ids,
    });
  }
  return ids.length;
}

function cleanupRegressionFavorites() {
  const projectCodes = Array.from(new Set([FAVORITE_PROJECT_CODE, "standalone", "test"]));
  return projectCodes.reduce((total, projectCode) => {
    return (
      total +
      cleanWebSQLFavoriteRows((item) => {
        const pathValue = String(item.path || "");
        const folderValue = String(item.folder || "");
        const nameValue = String(item.name || item.title || "");
        return (
          pathValue === "回归" ||
          pathValue.startsWith("回归/") ||
          folderValue === "回归" ||
          folderValue.startsWith("回归/") ||
          nameValue.startsWith("递归数字查询")
        );
      }, projectCode)
    );
  }, 0);
}

async function clearWebSQLLocalState(page) {
  await page.evaluate(() => {
    const prefixes = [
      "workspace-websql-connections",
      "workspace-websql-recent",
      "workspace-websql-favorites",
      "websql-lowcode",
      "workspace-websql-panel-state",
    ];
    for (const key of Object.keys(window.localStorage || {})) {
      if (prefixes.some((prefix) => key.startsWith(prefix))) {
        window.localStorage.removeItem(key);
      }
    }
  });
}

async function gotoPage(page, url) {
  if (page.url() === url) {
    await page.reload({ waitUntil: "domcontentloaded", timeout: 45000 });
  } else {
    await page.goto(url, { waitUntil: "domcontentloaded", timeout: 45000 });
  }
  await sleep(2500);
}

async function openWebSQLPanel(page) {
  if (await page.locator(".websql-lowcode").first().isVisible().catch(() => false)) {
    return;
  }
  const toggles = [
    page.locator('button[title*="WebSQL"]').last(),
    page.getByRole("button", { name: /^SQL$/ }).last(),
    page.getByText(/^SQL$/).last(),
  ];
  for (const locator of toggles) {
    if (await locator.isVisible().catch(() => false)) {
      await locator.click();
      await sleep(1600);
      if (await page.locator(".websql-lowcode").first().isVisible().catch(() => false)) {
        return;
      }
    }
  }
  await page.locator(".websql-lowcode").first().waitFor({ state: "visible", timeout: 25000 });
}

function brightness(rgb) {
  const matched = String(rgb || "").match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
  if (!matched) {
    return -1;
  }
  return (Number(matched[1]) + Number(matched[2]) + Number(matched[3])) / 3;
}

async function verifyLightTheme(page, label) {
  const info = await page.evaluate(() => {
    const visible = (node) => {
      if (!node) return false;
      const rect = node.getBoundingClientRect();
      return rect.width > 100 && rect.height > 80;
    };
    const root = document.querySelector(".websql-lowcode");
    const body = root?.querySelector(".websql-editor-body");
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    const editor = editors.find((item) => {
      const node = item?.getContainerDomNode?.();
      return node?.closest(".websql-lowcode") && !node.closest(".websql-ddl-editor") && visible(node);
    });
    const monaco = editor?.getContainerDomNode?.() || root?.querySelector(".monaco-editor");
    const background = monaco?.querySelector(".monaco-editor-background") || monaco;
    return {
      bodyBg: body ? window.getComputedStyle(body).backgroundColor : "",
      editorBg: background ? window.getComputedStyle(background).backgroundColor : "",
      monacoClass: monaco ? String(monaco.className || "") : "",
      localTheme: monaco ? String(monaco.getAttribute("data-sport-ui-local-theme") || "") : "",
    };
  });
  return {
    label,
    ...info,
    ok:
      brightness(info.bodyBg) >= 245 &&
      brightness(info.editorBg) >= 245 &&
      info.localTheme === "light",
  };
}

async function addSqlTabAndVerifyTheme(page, summary, label) {
  await page.locator(".websql-add-sql-btn").first().click();
  await sleep(1200);
  const theme = await verifyLightTheme(page, `${label}:after-add-tab`);
  summary.themeChecks.push(theme);
  assertCheck(theme.ok, `${label} editor is not light after adding SQL tab`);
}

async function setWebSQLEditorValue(page, value) {
  const result = await page.evaluate((nextValue) => {
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    for (const editor of editors) {
      const node = editor?.getContainerDomNode?.();
      if (!node || !node.closest(".websql-lowcode")) continue;
      const model = editor?.getModel?.();
      if (!model) continue;
      editor.setValue(String(nextValue || ""));
      editor.focus();
      return { ok: true, uri: String(model.uri || "") };
    }
    return { ok: false, reason: "websql monaco editor not found" };
  }, value);
  assertCheck(result.ok, result.reason || "failed to set WebSQL editor value");
  await sleep(900);
  return result;
}

async function clickToolbarButton(page, text) {
  const btn = page
    .locator(".websql-toolbar-left button")
    .filter({ hasText: new RegExp(`^${text}$`) })
    .first();
  await btn.waitFor({ state: "visible", timeout: 15000 });
  await btn.click();
}

async function clickLastVisibleButton(page, text) {
  const buttons = page.locator("button");
  const count = await buttons.count();
  for (let i = count - 1; i >= 0; i -= 1) {
    const button = buttons.nth(i);
    const normalized = await button
      .textContent()
      .then((value) => String(value || "").replace(/\s+/g, ""))
      .catch(() => "");
    if (normalized === text && (await button.isVisible().catch(() => false))) {
      await button.click();
      return;
    }
  }
  throw new Error(`visible button not found: ${text}`);
}

async function visibleModal(page, text, timeout = 10000) {
  const modal = page.locator(".ant-modal:visible").filter({ hasText: text }).last();
  await modal.waitFor({ state: "visible", timeout });
  return modal;
}

async function isModalVisible(page, text) {
  return page
    .locator(".ant-modal:visible")
    .filter({ hasText: text })
    .first()
    .isVisible()
    .catch(() => false);
}

async function assertModalActionable(modal, message) {
  const box = await modal.boundingBox();
  assertCheck(!!box && box.width > 100 && box.height > 80, message);
}

async function closeVisibleModal(page, text) {
  const modal = page.locator(".ant-modal:visible").filter({ hasText: text }).last();
  if (!(await modal.isVisible().catch(() => false))) {
    return;
  }
  const closeButton = modal.locator(".ant-modal-close").first();
  if (await closeButton.isVisible().catch(() => false)) {
    await closeButton.click();
    await sleep(500);
    return;
  }
  await page.keyboard.press("Escape");
  await sleep(500);
}

function parseRequestData(req) {
  const raw = req.postData() || "{}";
  try {
    return JSON.parse(raw);
  } catch (_error) {
    return {};
  }
}

async function waitWebSQLExecuteResponse(page, expectedOperation = "") {
  const response = await page.waitForResponse(
    (resp) => {
      if (!resp.url().includes("service=webshell.websql_execute") || resp.request().method() !== "POST") {
        return false;
      }
      if (!expectedOperation) {
        return true;
      }
      return String(parseRequestData(resp.request()).operation || "") === expectedOperation;
    },
    { timeout: 20000 }
  );
  const json = await response.json();
  json.__requestData = parseRequestData(response.request());
  return json;
}

async function bodyText(page) {
  return String((await page.locator("body").textContent().catch(() => "")) || "");
}

async function waitBodyText(page, matcher, timeout = 20000) {
  const started = Date.now();
  while (Date.now() - started < timeout) {
    const text = await bodyText(page);
    if (matcher.test(text)) {
      return text;
    }
    await sleep(400);
  }
  const finalText = await bodyText(page);
  throw new Error(`body text did not match ${matcher}; preview=${finalText.slice(0, 800)}`);
}

async function verifyMySQLConnectionOption(page, summary) {
  await page.locator('button[title="新增连接"]').first().click();
  await page.locator(".websql-conn-form.visible").first().waitFor({ state: "visible", timeout: 10000 });
  await page.locator(".websql-conn-form.visible .ant-select").first().click();
  const mysqlOption = page
    .locator(".ant-select-dropdown:visible .ant-select-item-option-content")
    .filter({ hasText: /^MySQL$/ })
    .first();
  await mysqlOption.waitFor({ state: "visible", timeout: 10000 });
  summary.checks.mysqlOptionInNewConnection = true;
  await mysqlOption.click();
  await sleep(600);
  summary.checks.mysqlFieldsVisible =
    (await page.locator(".websql-conn-form.visible").getByText(/^Host$/).isVisible().catch(() => false)) &&
    (await page.locator(".websql-conn-form.visible").getByText(/^Port$/).isVisible().catch(() => false));
  await page.locator(".websql-conn-form.visible .ant-select").first().click();
  await page
    .locator(".ant-select-dropdown:visible .ant-select-item-option-content")
    .filter({ hasText: /^SQLite$/ })
    .first()
    .click();
  await page.locator('button[title="关闭连接配置"]').first().click();
  await sleep(600);
  await page.evaluate(() => {
    for (const key of Object.keys(window.localStorage || {})) {
      if (key.startsWith("workspace-websql-connections")) {
        window.localStorage.removeItem(key);
      }
    }
  });
}

async function verifyBuiltinMySQLDefaults(page, summary) {
  const connections = await page.evaluate(() => {
    return Array.from(document.querySelectorAll(".websql-conn-btn")).map((node) =>
      String(node.textContent || "").replace(/\s+/g, " ").trim()
    );
  });
  summary.uiResponses.builtinConnections = connections;
  const mysqlRow =
    connections.find((text) => text.includes("MySQL")) || "";
  summary.uiResponses.builtinMySQLConnection = mysqlRow;
  assertCheck(mysqlRow.includes("202.140.140.117:3306"), `builtin mysql host mismatch: ${mysqlRow}`);
  summary.checks.builtinMySQLDefaultHost = true;
}

async function verifyQueryPaging(page, summary) {
  await setWebSQLEditorValue(page, QUERY_SQL);
  await clickToolbarButton(page, "执行");
  await waitBodyText(page, /游标\s*0\s*\/\s*本页\s*50\s*条|50\s*行/);
  summary.checks.queryFirstPage = true;

  const loadMoreResponse = waitWebSQLExecuteResponse(page, "execute");
  await page.locator(".websql-result-pagebar-actions button").filter({ hasText: "加载更多" }).first().click();
  const loadMoreJson = await loadMoreResponse;
  summary.uiResponses.loadMore = loadMoreJson.data || {};
  assertCheck(
    Number(summary.uiResponses.loadMore.cursor_offset) === 50 &&
      Number(summary.uiResponses.loadMore.row_count) === 10,
    "load more did not request the next cursor page"
  );
  summary.checks.queryLoadMore = true;

  await setWebSQLEditorValue(page, QUERY_SQL);
  await clickToolbarButton(page, "执行");
  await waitBodyText(page, /游标\s*0\s*\/\s*本页\s*50\s*条|50\s*行/);
  const nextResponse = waitWebSQLExecuteResponse(page, "execute");
  await page.locator(".websql-result-pagebar-actions button").filter({ hasText: "下一页" }).first().click();
  const nextJson = await nextResponse;
  summary.uiResponses.nextPage = nextJson.data || {};
  assertCheck(
    Number(summary.uiResponses.nextPage.cursor_offset) === 50 &&
      Number(summary.uiResponses.nextPage.row_count) === 10,
    "next page did not request cursor 50"
  );
  summary.checks.queryNextPage = true;
}

async function verifyEmptyQueryRendersGrid(page, summary) {
  await setWebSQLEditorValue(page, EMPTY_QUERY_SQL);
  const responsePromise = waitWebSQLExecuteResponse(page, "execute");
  await clickToolbarButton(page, "执行");
  const json = await responsePromise;
  const data = json.data || {};
  summary.uiResponses.emptyQuery = {
    row_count: data.row_count,
    column_count: Array.isArray(data.columns) ? data.columns.length : 0,
    statement_type: data.statement_type,
  };
  assertCheck(Number(data.row_count) === 0, "empty SELECT should return row_count 0");
  assertCheck(Array.isArray(data.columns) && data.columns.length > 0, "empty SELECT should keep result columns");

  await sleep(900);
  const render = await page.evaluate(() => {
    const root = document.querySelector(".websql-lowcode");
    const visible = (el) => !!el && !!(el.offsetWidth || el.offsetHeight || el.getClientRects().length);
    const gridVisible = Array.from(root?.querySelectorAll(".websql-result-grid") || []).some(visible);
    const emptyVisible = Array.from(root?.querySelectorAll(".websql-empty") || []).some(
      (el) => visible(el) && /查询结果为空/.test(String(el.textContent || ""))
    );
    const execInfoVisible = Array.from(root?.querySelectorAll(".websql-exec-info") || []).some(visible);
    return { gridVisible, emptyVisible, execInfoVisible, text: String(root?.innerText || "").slice(0, 1000) };
  });
  summary.uiResponses.emptyQueryRender = render;
  assertCheck(render.gridVisible || render.emptyVisible, "empty SELECT should render a query result area");
  assertCheck(!render.execInfoVisible, "empty SELECT should not render execution info card");
  summary.checks.queryEmptyGrid = true;
}

async function verifyFavoriteAndCopy(page, context, summary) {
  await context.grantPermissions(["clipboard-read", "clipboard-write"], { origin: BASE_URL });
  await setWebSQLEditorValue(page, QUERY_SQL);
  await clickToolbarButton(page, "执行");
  await waitBodyText(page, /查询完成|50\s*行/);
  await setWebSQLEditorValue(page, QUERY_SQL);
  await clickToolbarButton(page, "收藏");
  const favoriteModal = await visibleModal(page, "收藏 SQL");
  await favoriteModal.locator("input").nth(0).fill("递归数字查询");
  await favoriteModal.locator("input").nth(1).fill("回归/多层/一级/二级");
  await favoriteModal.locator("textarea").fill(QUERY_SQL);
  await clickLastVisibleButton(page, "确定");
  await waitBodyText(page, /已收藏 SQL：递归数字查询/);
  summary.checks.favoriteSaved = true;
  summary.checks.favoriteNamed = true;

  await clickToolbarButton(page, "快捷引入");
  await page.getByText("SQL 收藏夹").waitFor({ state: "visible", timeout: 10000 });
  summary.checks.quickImportDialog = true;
  await page.locator(".websql-favorite-tree").getByText("回归").waitFor({ state: "visible", timeout: 10000 });
  await page.locator(".websql-favorite-tree").getByText("多层").waitFor({ state: "visible", timeout: 10000 });
  await page.locator(".websql-favorite-tree").getByText("一级").waitFor({ state: "visible", timeout: 10000 });
  await page.locator(".websql-favorite-tree").getByText("二级").waitFor({ state: "visible", timeout: 10000 });
  await page.locator(".websql-favorite-tree").getByText("递归数字查询").waitFor({ state: "visible", timeout: 10000 });
  summary.checks.favoriteTree = true;
  summary.checks.favoriteMultilevelTree = true;

  await page.getByRole("button", { name: /新建目录/ }).click();
  const folderCreateEditor = page.locator(".websql-favorite-folder-inline-editor:visible").first();
  await folderCreateEditor.waitFor({ state: "visible", timeout: 10000 });
  assertCheck(!(await isModalVisible(page, "新建收藏目录")), "creating folder should not open a nested modal");
  await folderCreateEditor.locator("input").fill("回归/多层/空目录/子目录");
  await folderCreateEditor.getByRole("button", { name: "保存" }).click();
  await waitBodyText(page, /已创建收藏目录：回归\/多层\/空目录\/子目录/);
  await page.locator(".websql-favorite-tree").getByText("空目录").waitFor({ state: "visible", timeout: 10000 });
  await page.locator(".websql-favorite-tree").getByText("子目录").waitFor({ state: "visible", timeout: 10000 });
  summary.checks.favoriteFolderCreate = true;
  summary.checks.favoriteNoNestedModal = true;
  summary.checks.favoriteInlineEdit = true;

  await page.getByRole("button", { name: /新建目录/ }).click();
  const baseTestFolderEditor = page.locator(".websql-favorite-folder-inline-editor:visible").first();
  await baseTestFolderEditor.waitFor({ state: "visible", timeout: 10000 });
  await baseTestFolderEditor.locator("input").fill("test");
  await baseTestFolderEditor.getByRole("button", { name: "保存" }).click();
  await waitBodyText(page, /已创建收藏目录：test/);
  await page.locator(".websql-favorite-tree").getByText("test", { exact: true }).waitFor({
    state: "visible",
    timeout: 10000,
  });

  await page.locator(".websql-favorite-tree").getByText("test", { exact: true }).click();
  await page.getByRole("button", { name: /新建目录/ }).click();
  const nestedTestFolderEditor = page.locator(".websql-favorite-folder-inline-editor:visible").first();
  await nestedTestFolderEditor.waitFor({ state: "visible", timeout: 10000 });
  const testPrefill = await nestedTestFolderEditor.locator("input").inputValue();
  assertCheck(testPrefill === "test/", `selected test folder should prefill child path, got ${testPrefill}`);
  await nestedTestFolderEditor.locator("input").fill("test/test2/test3");
  await nestedTestFolderEditor.getByRole("button", { name: "保存" }).click();
  await waitBodyText(page, /已创建收藏目录：test\/test2\/test3/);
  await page.locator(".websql-favorite-tree").getByText("test2", { exact: true }).waitFor({
    state: "visible",
    timeout: 10000,
  });
  await page.locator(".websql-favorite-tree").getByText("test3", { exact: true }).waitFor({
    state: "visible",
    timeout: 10000,
  });
  summary.checks.favoriteExistingFolderNestedCreate = true;
  summary.checks.favoriteExistingFolderNestedRetained = true;
  const persistedTestFolders = websqlFavoriteQuery().filter(
    (item) => String(item.type || item.item_type || "") === "folder" && String(item.path || "") === "test/test2/test3"
  );
  assertCheck(persistedTestFolders.length > 0, "test/test2/test3 should be persisted in database");
  summary.checks.favoriteDatabasePersisted = true;

  await page.locator(".websql-favorite-tree").getByText("子目录").click();
  await page.locator(".websql-favorite-detail").getByRole("button", { name: "编辑" }).click();
  const folderEditEditor = page.locator(".websql-favorite-folder-inline-editor:visible").first();
  await folderEditEditor.waitFor({ state: "visible", timeout: 10000 });
  assertCheck(!(await isModalVisible(page, "编辑收藏目录")), "editing folder should not open a nested modal");
  await folderEditEditor.locator("input").fill("回归/多层/空目录改/子目录改");
  await folderEditEditor.getByRole("button", { name: "保存" }).click();
  await waitBodyText(page, /已更新收藏目录：回归\/多层\/空目录改\/子目录改/);
  await page.locator(".websql-favorite-tree").getByText("空目录改").waitFor({ state: "visible", timeout: 10000 });
  await page.locator(".websql-favorite-tree").getByText("子目录改").waitFor({ state: "visible", timeout: 10000 });
  assertCheck(
    !(await page.locator(".websql-favorite-tree").getByText("子目录", { exact: true }).isVisible().catch(() => false)),
    "old favorite folder name should be gone after rename"
  );
  summary.checks.favoriteFolderUpdate = true;

  await page.locator(".websql-favorite-tree").getByText("子目录改").click();
  await page.locator(".websql-favorite-detail").getByRole("button", { name: "删除" }).click();
  await waitBodyText(page, /已删除收藏项/);
  assertCheck(
    !(await page.locator(".websql-favorite-tree").getByText("子目录改", { exact: true }).isVisible().catch(() => false)),
    "favorite folder should be gone after delete"
  );
  summary.checks.favoriteFolderDelete = true;

  await page.locator(".websql-favorite-tree").getByText("二级").click();
  await page.locator(".websql-favorite-detail").getByRole("button", { name: "编辑" }).click();
  const folderCascadeEditor = page.locator(".websql-favorite-folder-inline-editor:visible").first();
  await folderCascadeEditor.waitFor({ state: "visible", timeout: 10000 });
  await folderCascadeEditor.locator("input").fill("回归/多层/一级改/二级改");
  await folderCascadeEditor.getByRole("button", { name: "保存" }).click();
  await waitBodyText(page, /已更新收藏目录：回归\/多层\/一级改\/二级改/);
  await page.locator(".websql-favorite-tree").getByText("一级改").waitFor({ state: "visible", timeout: 10000 });
  await page.locator(".websql-favorite-tree").getByText("二级改").waitFor({ state: "visible", timeout: 10000 });
  await page.locator(".websql-favorite-tree").getByText("递归数字查询").click();
  await page.locator(".websql-favorite-detail").getByText("目录：回归/多层/一级改/二级改").waitFor({
    state: "visible",
    timeout: 10000,
  });
  summary.checks.favoriteFolderCascadeUpdate = true;

  await page.locator(".websql-favorite-detail").getByText("递归数字查询").waitFor({ state: "visible", timeout: 10000 });
  await page.locator(".websql-favorite-detail").getByRole("button", { name: "复制" }).click();
  await sleep(500);
  const favoriteClipboard = await page.evaluate(() => navigator.clipboard.readText().catch(() => ""));
  summary.clipboard.favorite = favoriteClipboard;
  summary.checks.favoriteCopy = favoriteClipboard.includes("WITH RECURSIVE");

  const recursiveRecent = page
    .locator(".websql-quick-list .websql-quick-item")
    .filter({ hasText: "WITH RECURSIVE" })
    .first();
  await recursiveRecent.waitFor({ state: "visible", timeout: 10000 });
  await recursiveRecent.locator(".websql-quick-actions button").filter({ hasText: "复制" }).first().click();
  await sleep(500);
  const recentClipboard = await page.evaluate(() => navigator.clipboard.readText().catch(() => ""));
  summary.clipboard.recent = recentClipboard;
  summary.checks.recentCopy = recentClipboard.includes("WITH RECURSIVE");

  await page.locator(".websql-favorite-detail").getByRole("button", { name: "引入" }).click();
  await waitBodyText(page, /已引入收藏 SQL/);

  await clickToolbarButton(page, "快捷引入");
  await visibleModal(page, "快捷引入");
  await page.locator(".websql-favorite-tree").getByText("递归数字查询").click();
  await page.locator(".websql-favorite-detail").getByRole("button", { name: "编辑" }).click();
  const favoriteEditEditor = page.locator(".websql-favorite-sql-inline-editor:visible").first();
  await favoriteEditEditor.waitFor({ state: "visible", timeout: 10000 });
  assertCheck(!(await isModalVisible(page, "编辑收藏 SQL")), "editing SQL favorite should not open a nested modal");
  await favoriteEditEditor.locator("input").nth(0).fill("递归数字查询改");
  await favoriteEditEditor.locator("input").nth(1).fill("回归/多层/收藏改/叶子");
  await favoriteEditEditor.locator("textarea").fill(QUERY_SQL.replace("ORDER BY x", "ORDER BY n DESC"));
  await favoriteEditEditor.getByRole("button", { name: "保存" }).click();
  await waitBodyText(page, /已更新收藏 SQL：递归数字查询改/);
  await page.locator(".websql-favorite-tree").getByText("收藏改").waitFor({ state: "visible", timeout: 10000 });
  await page.locator(".websql-favorite-tree").getByText("叶子").waitFor({ state: "visible", timeout: 10000 });
  await page.locator(".websql-favorite-tree").getByText("递归数字查询改").waitFor({ state: "visible", timeout: 10000 });
  summary.checks.favoriteEdit = true;

  await page.locator(".websql-favorite-tree").getByText("递归数字查询改").click();
  await page.locator(".websql-favorite-detail").getByRole("button", { name: "删除" }).click();
  await waitBodyText(page, /已删除收藏项/);
  assertCheck(
    !(await page.locator(".websql-favorite-tree").getByText("递归数字查询改", { exact: true }).isVisible().catch(() => false)),
    "SQL favorite should be gone after delete"
  );
  summary.checks.favoriteDelete = true;

  await closeVisibleModal(page, "快捷引入");
  await sleep(500);
}

async function verifyManualRollback(page, summary) {
  await setWebSQLEditorValue(page, MANUAL_SQL);
  await clickToolbarButton(page, "执行");
  await waitBodyText(page, /待提交|等待手动提交/);
  summary.checks.manualPendingUi = true;

  const rollback = page
    .locator(".websql-exec-actions button")
    .filter({ hasText: "回滚" })
    .first();
  await rollback.waitFor({ state: "visible", timeout: 15000 });
  const rollbackResponse = waitWebSQLExecuteResponse(page, "rollback");
  await rollback.click();
  const confirmRollback = page.locator(".ant-popconfirm-buttons button").last();
  if (await confirmRollback.isVisible({ timeout: 3000 }).catch(() => false)) {
    await confirmRollback.click();
  }
  const rollbackJson = await rollbackResponse;
  summary.uiResponses.rollback = rollbackJson.data || { msg: rollbackJson.msg, code: rollbackJson.code };
  summary.uiResponses.rollbackRequest = rollbackJson.__requestData || {};
  assertCheck(!!String(summary.uiResponses.rollbackRequest.event_id || ""), "rollback request event_id is empty");
  assertCheck(
    summary.uiResponses.rollback.commit_status === "rolled_back",
    `rollback button did not roll back: ${JSON.stringify(summary.uiResponses.rollback)}`
  );
  summary.checks.manualRollbackUi = true;
}

async function verifyRecentDatabasePersistence(page, summary) {
  const items = websqlRecentQuery();
  summary.api.recent = {
    count: items.length,
    topIds: items.slice(0, 5).map((item) => item.id || item.recent_sql_id || ""),
  };
  assertCheck(
    items.some((item) => String(item.sql || "").includes("WITH RECURSIVE cnt")),
    "recent SQL should be persisted in database"
  );

  await clearWebSQLLocalState(page);
  const recentReloadResponse = page.waitForResponse(
    (resp) =>
      resp.url().includes("service=webshell.websql_recent_query") && resp.request().method() === "POST",
    { timeout: 15000 }
  );
  await gotoPage(page, STANDALONE_URL);
  await openWebSQLPanel(page);
  const reloadJson = await recentReloadResponse.then((resp) => resp.json());
  const recentTitles = Array.isArray(reloadJson?.data)
    ? reloadJson.data.map((item) => String(item?.sql || ""))
    : [];
  summary.uiResponses.recentReloadQuery = {
    code: reloadJson?.code,
    count: recentTitles.length,
    sample: recentTitles.slice(0, 5),
  };
  assertCheck(
    recentTitles.some((text) => text.includes("WITH RECURSIVE cnt")),
    "recent SQL query should reload from database after clearing local state"
  );
  summary.checks.recentDatabasePersisted = true;
}

function verifyApi(summary) {
  const first = websql({
    operation: "execute",
    driver: "sqlite",
    sqlite_path: "./database/price.db",
    sql: QUERY_SQL,
    page_size: 50,
    cursor_offset: 0,
    max_rows: 500,
    commit_mode: "manual",
  });
  summary.api.cursorFirst = {
    rowCount: first.row_count,
    hasNext: first.has_next,
    nextCursor: first.next_cursor,
  };
  assertCheck(first.row_count === 50 && first.has_next === true && first.next_cursor === 50, "API first cursor page failed");

  const second = websql({
    operation: "execute",
    driver: "sqlite",
    sqlite_path: "./database/price.db",
    sql: QUERY_SQL,
    page_size: 50,
    cursor_offset: 50,
    max_rows: 500,
    commit_mode: "manual",
  });
  summary.api.cursorSecond = {
    rowCount: second.row_count,
    hasPrev: second.has_prev,
    hasNext: second.has_next,
    prevCursor: second.prev_cursor,
  };
  assertCheck(second.row_count === 10 && second.has_prev === true && second.has_next === false, "API second cursor page failed");

  const manual = websql({
    operation: "execute",
    driver: "sqlite",
    sqlite_path: "./database/price.db",
    sql: "CREATE TEMP TABLE websql_api_tmp(id INTEGER);",
    commit_mode: "manual",
    transaction_ttl_seconds: 120,
    connection_id: "autocheck",
    connection_name: "api-regression",
  });
  summary.api.manual = {
    eventId: manual.event_id,
    pendingCommit: manual.pending_commit,
    commitStatus: manual.commit_status,
  };
  assertCheck(manual.pending_commit === true && !!manual.event_id, "API manual pending event failed");

  const rolledBack = websql({
    operation: "rollback",
    event_id: manual.event_id,
  });
  summary.api.rollback = rolledBack;
  assertCheck(rolledBack.commit_status === "rolled_back", "API rollback failed");

  const direct = websql({
    operation: "execute",
    driver: "sqlite",
    sqlite_path: "./database/price.db",
    sql: "CREATE TEMP TABLE websql_api_direct_tmp(id INTEGER);",
    commit_mode: "direct",
    connection_id: "autocheck",
    connection_name: "api-regression",
  });
  summary.api.direct = {
    eventId: direct.event_id,
    commitStatus: direct.commit_status,
  };
  assertCheck(direct.commit_status === "direct_committed" && !!direct.event_id, "API direct commit event failed");

  const events = websql({
    operation: "list_events",
    size: 5,
  });
  summary.api.events = {
    count: Array.isArray(events.items) ? events.items.length : 0,
    pendingCount: events.pending_count,
    latestStatuses: (Array.isArray(events.items) ? events.items : []).map((item) => item.status),
  };
  assertCheck(Array.isArray(events.items) && events.items.length > 0, "API event list failed");

  summary.checks.apiCursor = true;
  summary.checks.apiManualRollback = true;
  summary.checks.apiDirectEvent = true;
  summary.checks.apiEventList = true;
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const resultPath = path.join(OUT_DIR, "websql-pool-regression-check.json");
  const screenshots = {
    standalone: path.join(OUT_DIR, "websql-pool-regression-standalone.png"),
    webshell: path.join(OUT_DIR, "websql-pool-regression-webshell.png"),
    final: path.join(OUT_DIR, "websql-pool-regression-final.png"),
  };

  const summary = {
    baseUrl: BASE_URL,
    standaloneUrl: STANDALONE_URL,
    webshellUrl: WEBSHELL_URL,
    apiUrl: API_URL,
    checks: {
      apiCursor: false,
      apiManualRollback: false,
      apiDirectEvent: false,
      apiEventList: false,
      standaloneOpened: false,
      standaloneDark: false,
      webshellOpenedNoLogin: false,
      webshellDark: false,
      builtinMySQLDefaultHost: false,
      mysqlOptionInNewConnection: false,
      mysqlFieldsVisible: false,
      queryFirstPage: false,
      queryNextPage: false,
      queryLoadMore: false,
      queryEmptyGrid: false,
      favoriteSaved: false,
      favoriteNamed: false,
      favoriteTree: false,
      quickImportDialog: false,
      favoriteMultilevelTree: false,
      favoriteNoNestedModal: false,
      favoriteInlineEdit: false,
      favoriteFolderCreate: false,
      favoriteExistingFolderNestedCreate: false,
      favoriteExistingFolderNestedRetained: false,
      favoriteDatabasePersisted: false,
      favoriteFolderUpdate: false,
      favoriteFolderDelete: false,
      favoriteFolderCascadeUpdate: false,
      favoriteEdit: false,
      favoriteDelete: false,
      favoriteCopy: false,
      recentCopy: false,
      recentDatabasePersisted: false,
      manualPendingUi: false,
      manualRollbackUi: false,
    },
    themeChecks: [],
    api: {},
    uiResponses: {},
    clipboard: {},
    screenshots,
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    pass: false,
    error: "",
  };

  let browser;
  try {
    verifyApi(summary);
    summary.favoriteCleanupCount = cleanupRegressionFavorites();

    browser = await chromium.launch({ headless: true });
    const context = await browser.newContext({
      ignoreHTTPSErrors: true,
      viewport: { width: 1680, height: 980 },
    });
    const page = await context.newPage();
    page.on("console", (msg) => {
      if (msg.type() === "error") summary.consoleErrors.push(msg.text());
    });
    page.on("pageerror", (error) => summary.pageErrors.push(String(error)));
    page.on("requestfailed", (req) => {
      summary.failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || "failed"}`);
    });

    await gotoPage(page, STANDALONE_URL);
    await clearWebSQLLocalState(page);
    await gotoPage(page, STANDALONE_URL);
    await openWebSQLPanel(page);
    summary.checks.standaloneOpened = true;
    await verifyBuiltinMySQLDefaults(page, summary);
    const standaloneTheme = await verifyLightTheme(page, "standalone");
    summary.themeChecks.push(standaloneTheme);
    summary.checks.standaloneDark = standaloneTheme.ok;
    assertCheck(standaloneTheme.ok, "standalone WebSQL editor is not light");
    await addSqlTabAndVerifyTheme(page, summary, "standalone");
    await page.screenshot({ path: screenshots.standalone, fullPage: true });

    await gotoPage(page, WEBSHELL_URL);
    await clearWebSQLLocalState(page);
    await gotoPage(page, WEBSHELL_URL);
    await openWebSQLPanel(page);
    summary.checks.webshellOpenedNoLogin = true;
    const webshellTheme = await verifyLightTheme(page, "webshell");
    summary.themeChecks.push(webshellTheme);
    summary.checks.webshellDark = webshellTheme.ok;
    assertCheck(webshellTheme.ok, "webshell WebSQL editor is not light");
    await addSqlTabAndVerifyTheme(page, summary, "webshell");
    await verifyMySQLConnectionOption(page, summary);
    await page.screenshot({ path: screenshots.webshell, fullPage: true });

    await gotoPage(page, WEBSHELL_URL);
    await clearWebSQLLocalState(page);
    await gotoPage(page, WEBSHELL_URL);
    await openWebSQLPanel(page);
    await verifyQueryPaging(page, summary);
    await verifyEmptyQueryRendersGrid(page, summary);

    await gotoPage(page, STANDALONE_URL);
    await clearWebSQLLocalState(page);
    await gotoPage(page, STANDALONE_URL);
    await openWebSQLPanel(page);
    await verifyFavoriteAndCopy(page, context, summary);
    await verifyRecentDatabasePersistence(page, summary);
    await verifyManualRollback(page, summary);
    await page.screenshot({ path: screenshots.final, fullPage: true });

    const blockingConsoleErrors = summary.consoleErrors.filter((item) => !/ResizeObserver|favicon/i.test(item));
    const blockingRequestFailures = summary.failedRequests.filter((item) => !/favicon/i.test(item));
    assertCheck(blockingConsoleErrors.length === 0, `console errors found: ${blockingConsoleErrors.slice(0, 3).join(" | ")}`);
    assertCheck(summary.pageErrors.length === 0, `page errors found: ${summary.pageErrors.slice(0, 3).join(" | ")}`);
    assertCheck(blockingRequestFailures.length === 0, `request failures found: ${blockingRequestFailures.slice(0, 3).join(" | ")}`);
    assertCheck(Object.values(summary.checks).every(Boolean), "not all checks passed");

    summary.pass = true;
    await context.close();
  } catch (error) {
    summary.error = error?.stack || String(error);
    process.exitCode = 1;
  } finally {
    if (browser) {
      await browser.close().catch(() => undefined);
    }
    fs.writeFileSync(resultPath, JSON.stringify(summary, null, 2));
    console.log(JSON.stringify(summary, null, 2));
  }
})();
