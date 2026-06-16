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
const TARGET_URL =
  process.env.WEBSQL_TARGET_URL ||
  `${BASE_URL}/collect-ui#/collect-ui/framework/websql-pool`;
const API_URL = process.env.WEBSQL_API_URL || `${BASE_URL}/template_data/data`;
const OUT_DIR =
  process.env.WEBSQL_OUTPUT_DIR ||
  "/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation";

function assertCheck(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
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
  const out = JSON.parse(String(res.stdout || "{}"));
  if (!out || String(out.code || "") !== "0" || out.success === false) {
    throw new Error(`${service} failed: ${out?.msg || "unknown error"}`);
  }
  return out;
}

function websql(data) {
  return runCurl("webshell.websql_execute", {
    driver: "sqlite",
    sqlite_path: "./database/price.db",
    commit_mode: "direct",
    ...data,
  }).data;
}

function parseRequestData(req) {
  try {
    return JSON.parse(req.postData() || "{}");
  } catch (_error) {
    return {};
  }
}

async function setWebSQLEditorValue(page, sqlText, tableName) {
  const result = await page.evaluate(
    ({ sqlText: value, tableName: name }) => {
      const editors = window?.monaco?.editor?.getEditors?.() || [];
      const editor = editors.find((item) => {
        const node = item?.getContainerDomNode?.();
        return node && node.closest(".websql-editor-body");
      });
      if (!editor) {
        return { ok: false, reason: "websql editor not found" };
      }
      editor.setValue(value);
      const model = editor.getModel();
      const index = String(value).indexOf(name);
      const offset = index >= 0 ? index + Math.max(1, Math.floor(name.length / 2)) : 0;
      const position = model.getPositionAt(offset);
      editor.setPosition(position);
      editor.revealPositionInCenter(position);
      editor.focus();
      const visible = editor.getScrolledVisiblePosition(position);
      const rect = editor.getContainerDomNode().getBoundingClientRect();
      return {
        ok: true,
        x: rect.left + (visible?.left || 0) + 6,
        y: rect.top + (visible?.top || 0) + 8,
        position,
      };
    },
    { sqlText, tableName }
  );
  assertCheck(result.ok, result.reason || "set editor value failed");
  return result;
}

async function readWebSQLEditorValue(page) {
  return page.evaluate(() => {
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    const editor = editors.find((item) => {
      const node = item?.getContainerDomNode?.();
      return node && node.closest(".websql-editor-body");
    });
    return String(editor?.getValue?.() || "");
  });
}

async function readDdlEditorValue(page) {
  return page.evaluate(() => {
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    const editor = editors.find((item) => {
      const node = item?.getContainerDomNode?.();
      return node && node.closest(".websql-ddl-editor");
    });
    return String(editor?.getValue?.() || "");
  });
}

async function openContextMenuAt(page, point) {
  await page.mouse.click(point.x, point.y, { button: "right" });
  await page.waitForSelector(".monaco-menu .action-label", {
    state: "visible",
    timeout: 10000,
  });
}

async function readContextMenuItems(page) {
  const labels = page.locator(".monaco-menu .action-label");
  const count = await labels.count();
  const items = [];
  for (let index = 0; index < count; index += 1) {
    const item = labels.nth(index);
    const text = String(await item.innerText().catch(() => "")).trim();
    const box = await item.boundingBox().catch(() => null);
    if (!box) {
      items.push({ text, x: 0, y: 0 });
      continue;
    }
    items.push({
      text,
      x: box.x + box.width / 2,
      y: box.y + box.height / 2,
    });
  }
  return items;
}

function hasTableContextMenuItems(menuTexts) {
  return menuTexts.includes("查字段") || menuTexts.includes("查 DDL");
}

async function closeContextMenu(page) {
  await page.keyboard.press("Escape");
  await page
    .locator(".monaco-menu .action-label")
    .first()
    .waitFor({ state: "hidden", timeout: 5000 })
    .catch(() => undefined);
}

async function closeTopModal(page) {
  const closeButtons = page.locator(".ant-modal-close");
  const count = await closeButtons.count();
  if (count > 0) {
    await closeButtons.nth(count - 1).click();
  } else {
    await page.keyboard.press("Escape");
  }
  await page.waitForFunction(
    () =>
      !Array.from(document.querySelectorAll(".ant-modal-wrap")).some((node) => {
        const style = window.getComputedStyle(node);
        const rect = node.getBoundingClientRect();
        return (
          style.display !== "none" &&
          style.visibility !== "hidden" &&
          rect.width > 0 &&
          rect.height > 0
        );
      }),
    null,
    { timeout: 10000 }
  );
}

async function clickContextMenuItem(page, items, label) {
  const index = items.findIndex((entry) =>
    String(entry.text || "").includes(label)
  );
  assertCheck(index >= 0, `context menu item not found: ${label}`);
  const item = items[index];
  if (item.x && item.y) {
    await page.mouse.move(item.x, item.y);
    await page.mouse.down();
    await sleep(50);
    await page.mouse.up();
    const closed = await page
      .locator(".monaco-menu .action-label")
      .first()
      .waitFor({ state: "hidden", timeout: 1000 })
      .then(() => true)
      .catch(() => false);
    if (closed) {
      return;
    }
  }
  const labelLocator = page
    .locator(".monaco-menu .action-label")
    .filter({ hasText: label })
    .first();
  const itemLocator = labelLocator.locator(
    "xpath=ancestor::*[contains(concat(' ', normalize-space(@class), ' '), ' action-item ')]"
  );
  try {
    await itemLocator.click({ timeout: 5000, force: true });
    const closed = await page
      .locator(".monaco-menu .action-label")
      .first()
      .waitFor({ state: "hidden", timeout: 1000 })
      .then(() => true)
      .catch(() => false);
    if (closed) {
      return;
    }
  } catch (_error) {
    // Fall back to keyboard navigation if Monaco keeps the context item outside normal hit testing.
  }
  for (let i = 0; i < index; i += 1) {
    await page.keyboard.press("ArrowDown");
  }
  await page.keyboard.press("Enter");
}

async function waitObjectDetailResponse(page, tableName) {
  const response = await page.waitForResponse(
    (resp) => {
      if (
        !resp.url().includes("service=webshell.websql_execute") ||
        resp.request().method() !== "POST"
      ) {
        return false;
      }
      const payload = parseRequestData(resp.request());
      return (
        payload.operation === "object_detail" &&
        String(payload.object_name || "") === tableName
      );
    },
    { timeout: 30000 }
  );
  return response.json();
}

async function main() {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const tableName = `websql_context_menu_${Date.now()}`;
  const screenshotPath = path.join(OUT_DIR, "websql-table-context-menu-check.png");
  const reportPath = path.join(OUT_DIR, "websql-table-context-menu-check.json");
  const summary = {
    url: TARGET_URL,
    tableName,
    checks: {},
    responses: {},
    consoleErrors: [],
    pageErrors: [],
    requestFailed: [],
    screenshot: screenshotPath,
    report: reportPath,
  };

  websql({
    operation: "execute",
    sql: `CREATE TABLE IF NOT EXISTS ${tableName} (id INTEGER PRIMARY KEY, context_name TEXT NOT NULL, context_note TEXT DEFAULT 'seed');`,
  });

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    ignoreHTTPSErrors: true,
  });
  await context.addInitScript(() => {
    for (const key of Object.keys(window.localStorage || {})) {
      if (key.startsWith("workspace-websql-connections")) {
        window.localStorage.removeItem(key);
      }
    }
  });
  const page = await context.newPage();
  page.on("console", (msg) => {
    if (msg.type() === "error") {
      summary.consoleErrors.push(msg.text());
    }
  });
  page.on("pageerror", (error) => summary.pageErrors.push(error.message));
  page.on("requestfailed", (request) => {
    summary.requestFailed.push({
      url: request.url(),
      method: request.method(),
      failure: request.failure()?.errorText || "",
    });
  });

  try {
    await page.goto(TARGET_URL, { waitUntil: "domcontentloaded", timeout: 45000 });
    await page.waitForSelector(".websql-lowcode .monaco-editor", {
      state: "visible",
      timeout: 45000,
    });
    await page.waitForFunction(
      () => !!window?.monaco?.editor?.getEditors?.().length,
      null,
      { timeout: 30000 }
    );
    summary.checks.pageOpened = true;

    const sql = `SELECT * FROM ${tableName} t WHERE t.id = 1;`;
    const selectPoint = await setWebSQLEditorValue(page, sql, "SELECT");
    await openContextMenuAt(page, selectPoint);
    const selectMenuTexts = (await readContextMenuItems(page)).map(
      (item) => item.text
    );
    summary.selectMenuTexts = selectMenuTexts;
    assertCheck(
      !hasTableContextMenuItems(selectMenuTexts),
      `table context menu should not show on SELECT: ${selectMenuTexts.join(",")}`
    );
    await closeContextMenu(page);

    const fromPoint = await setWebSQLEditorValue(page, sql, "FROM");
    await openContextMenuAt(page, fromPoint);
    const fromMenuTexts = (await readContextMenuItems(page)).map(
      (item) => item.text
    );
    summary.fromMenuTexts = fromMenuTexts;
    assertCheck(
      !hasTableContextMenuItems(fromMenuTexts),
      `table context menu should not show on FROM: ${fromMenuTexts.join(",")}`
    );
    await closeContextMenu(page);

    const point = await setWebSQLEditorValue(page, sql, tableName);
    await openContextMenuAt(page, point);
    summary.checks.contextMenuVisible = true;
    const menuItems = await readContextMenuItems(page);
    const menuTexts = menuItems.map((item) => item.text);
    summary.menuTexts = menuTexts;
    assertCheck(menuTexts.includes("查字段"), `missing 查字段: ${menuTexts.join(",")}`);
    assertCheck(menuTexts.includes("查 DDL"), `missing 查 DDL: ${menuTexts.join(",")}`);

    const fieldsResponsePromise = waitObjectDetailResponse(page, tableName).catch(
      (error) => ({ __error: error.message || String(error) })
    );
    await clickContextMenuItem(page, menuItems, "查字段");
    const fieldsJson = await fieldsResponsePromise;
    assertCheck(!fieldsJson.__error, fieldsJson.__error);
    summary.responses.fields = {
      code: fieldsJson.code,
      columnCount: Array.isArray(fieldsJson.data?.columns)
        ? fieldsJson.data.columns.length
        : 0,
    };
    await page.waitForSelector(".websql-columns-table", {
      state: "visible",
      timeout: 20000,
    });
    await page.waitForFunction(
      (name) => document.body.innerText.includes("字段 · " + name),
      tableName,
      { timeout: 10000 }
    );
    const columnsText = await page.locator(".ant-modal").last().innerText();
    assertCheck(columnsText.includes("context_name"), "field table missing context_name");
    assertCheck(columnsText.includes("context_note"), "field table missing context_note");
    summary.checks.fieldsDialog = true;
    await closeTopModal(page);

    const ddlPoint = await setWebSQLEditorValue(page, sql, tableName);
    await openContextMenuAt(page, ddlPoint);
    const ddlMenuItems = await readContextMenuItems(page);
    const ddlResponsePromise = waitObjectDetailResponse(page, tableName).catch(
      (error) => ({ __error: error.message || String(error) })
    );
    await clickContextMenuItem(page, ddlMenuItems, "查 DDL");
    const ddlJson = await ddlResponsePromise;
    assertCheck(!ddlJson.__error, ddlJson.__error);
    summary.responses.ddl = {
      code: ddlJson.code,
      ddlLength: String(ddlJson.data?.ddl || "").length,
    };
    await page.waitForSelector(".websql-ddl-editor", {
      state: "visible",
      timeout: 20000,
    });
    await page.waitForFunction(
      (name) => document.body.innerText.includes(name),
      tableName,
      { timeout: 10000 }
    );
    const ddlText = await readDdlEditorValue(page);
    assertCheck(/CREATE\s+TABLE/i.test(ddlText), "DDL editor missing CREATE TABLE");
    assertCheck(ddlText.includes(tableName), "DDL editor missing table name");
    assertCheck(ddlText.includes("context_note"), "DDL editor missing field");
    summary.checks.ddlDialog = true;
    await closeTopModal(page);

    await setWebSQLEditorValue(page, "", tableName);
    const mysqlClicked = await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll(".websql-conn-btn"));
      const button = buttons.find((item) => String(item.textContent || "").includes("MySQL"));
      if (!button) {
        return false;
      }
      button.click();
      return true;
    });
    assertCheck(mysqlClicked, "MySQL connection button not found");
    await sleep(1200);
    const mysqlSql = await readWebSQLEditorValue(page);
    summary.mysqlSqlAfterClick = mysqlSql;
    assertCheck(
      !/SHOW\s+TABLES/i.test(mysqlSql),
      `MySQL click should not create SHOW TABLES query: ${mysqlSql}`
    );
    summary.checks.mysqlEmptyQuery = true;

    await page.screenshot({ path: screenshotPath, fullPage: true });
    assertCheck(
      summary.consoleErrors.length === 0,
      `console errors: ${summary.consoleErrors.join("\n")}`
    );
    assertCheck(
      summary.pageErrors.length === 0,
      `page errors: ${summary.pageErrors.join("\n")}`
    );
    assertCheck(
      summary.requestFailed.length === 0,
      `requestfailed: ${JSON.stringify(summary.requestFailed)}`
    );
    summary.ok = true;
  } catch (error) {
    summary.ok = false;
    summary.error = error.message;
    await page.screenshot({ path: screenshotPath, fullPage: true }).catch(() => undefined);
    throw error;
  } finally {
    await browser.close().catch(() => undefined);
    websql({
      operation: "execute",
      sql: `DROP TABLE IF EXISTS ${tableName};`,
    });
    fs.writeFileSync(reportPath, JSON.stringify(summary, null, 2));
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
