#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

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
const OUT_DIR =
  process.env.WEBSQL_OUTPUT_DIR ||
  "/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation";

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function assertCheck(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

async function visibleCount(page, selector) {
  return page.locator(selector).evaluateAll((nodes) => {
    const visible = (node) => {
      const rect = node.getBoundingClientRect();
      const style = window.getComputedStyle(node);
      return rect.width > 0 && rect.height > 0 && style.visibility !== "hidden" && style.display !== "none";
    };
    return nodes.filter(visible).length;
  });
}

async function visibleButtonTextCount(page, label) {
  return page.locator("button").filter({ hasText: label }).evaluateAll((nodes, expected) => {
    const visible = (node) => {
      const rect = node.getBoundingClientRect();
      const style = window.getComputedStyle(node);
      return rect.width > 0 && rect.height > 0 && style.visibility !== "hidden" && style.display !== "none";
    };
    return nodes.filter((node) => visible(node) && String(node.textContent || "").trim() === expected).length;
  }, label);
}

async function rightClickSqlTab(page, paneSelector, label) {
  const tabText = page.locator(`${paneSelector} .workspace-websql-sql-tabs`).getByText(label, { exact: true }).first();
  await tabText.waitFor({ state: "visible", timeout: 15000 });
  await tabText.click({ button: "right" });
}

async function chooseMenu(page, label) {
  const item = page
    .locator('.contexify_item:visible, [role="menuitem"]:visible')
    .filter({ hasText: label })
    .last();
  await item.waitFor({ state: "visible", timeout: 10000 });
  await item.click();
  await sleep(1200);
}

async function setPaneSql(page, paneSelector, sql) {
  const result = await page.evaluate(
    ({ paneSelector, sql }) => {
      const root = document.querySelector(paneSelector);
      const editors = window?.monaco?.editor?.getEditors?.() || [];
      const editor = editors.find((item) => {
        const node = item?.getContainerDomNode?.();
        return node && root && root.contains(node);
      });
      if (!editor) {
        return { ok: false, reason: `editor not found in ${paneSelector}` };
      }
      editor.setValue(String(sql || ""));
      editor.focus();
      return { ok: true, uri: String(editor.getModel()?.uri || "") };
    },
    { paneSelector, sql }
  );
  assertCheck(result.ok, result.reason || "failed to set SQL editor");
  await sleep(900);
  return result;
}

function parseRequestData(request) {
  try {
    return JSON.parse(request.postData() || "{}");
  } catch (_error) {
    return {};
  }
}

async function waitExecute(page, expectedSqlPart) {
  const response = await page.waitForResponse(
    (resp) => {
      if (!resp.url().includes("service=webshell.websql_execute") || resp.request().method() !== "POST") {
        return false;
      }
      const data = parseRequestData(resp.request());
      return data.operation === "execute" && String(data.sql || "").includes(expectedSqlPart);
    },
    { timeout: 25000 }
  );
  const json = await response.json();
  json.__requestData = parseRequestData(response.request());
  return json;
}

async function run() {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const reportPath = path.join(OUT_DIR, "websql-tab-split-check.json");
  const horizontalShot = path.join(OUT_DIR, "websql-tab-split-horizontal.png");
  const verticalShot = path.join(OUT_DIR, "websql-tab-split-vertical.png");

  const summary = {
    targetUrl: TARGET_URL,
    checks: {},
    consoleErrors: [],
    pageErrors: [],
    requestFailed: [],
    responses: {},
    screenshots: {
      horizontal: horizontalShot,
      vertical: verticalShot,
    },
  };

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 920 } });
  await context.addInitScript(() => {
    for (const key of Object.keys(window.localStorage || {})) {
      if (key.startsWith("websql-lowcode-workbench-split")) {
        window.localStorage.removeItem(key);
      }
    }
  });
  const page = await context.newPage();

  page.on("console", (msg) => {
    if (["error"].includes(msg.type())) {
      summary.consoleErrors.push(msg.text());
    }
  });
  page.on("pageerror", (error) => {
    summary.pageErrors.push(error.message);
  });
  page.on("requestfailed", (request) => {
    summary.requestFailed.push({
      url: request.url(),
      method: request.method(),
      failure: request.failure()?.errorText || "",
    });
  });

  try {
    await page.goto(TARGET_URL, { waitUntil: "domcontentloaded", timeout: 45000 });
    await page.locator(".websql-lowcode").first().waitFor({ state: "visible", timeout: 30000 });
    await page.locator(".workspace-websql-sql-tabs").first().waitFor({ state: "visible", timeout: 30000 });
    await page.locator(".websql-standalone-main-split").first().waitFor({ state: "visible", timeout: 30000 });

    summary.checks.pageOpened = true;
    summary.checks.noPingOrDdlToolbarButtons =
      (await visibleButtonTextCount(page, "Ping")) === 0 &&
      (await visibleButtonTextCount(page, "DDL")) === 0;
    assertCheck(summary.checks.noPingOrDdlToolbarButtons, "operation toolbar should not show Ping or DDL buttons");
    summary.checks.defaultSinglePanel =
      (await visibleCount(page, ".websql-standalone-main-split")) === 1 &&
      (await visibleCount(page, ".websql-workbench-split")) === 0;
    assertCheck(summary.checks.defaultSinglePanel, "default view should render exactly one normal SQL panel");

    await rightClickSqlTab(page, ".websql-main", "SQL 1");
    await chooseMenu(page, "左右分割");
    await page.locator(".websql-workbench-split.direction-horizontal").first().waitFor({
      state: "visible",
      timeout: 15000,
    });
    summary.checks.horizontalSplit = true;
    summary.checks.perPaneToolbars =
      (await visibleCount(page, ".websql-workbench-split .websql-pane-toolbar")) >= 2 &&
      (await visibleCount(page, ".websql-workbench-split .workspace-websql-sql-tabs")) >= 2 &&
      (await visibleCount(page, ".websql-main > .websql-toolbar")) === 0;
    assertCheck(summary.checks.perPaneToolbars, "split view should use per-pane toolbars and SQL tabs");
    summary.checks.twoSqlEditorsAfterSplit =
      (await visibleCount(page, ".websql-workbench-split .websql-editor-body .monaco-editor")) >= 2;
    assertCheck(summary.checks.twoSqlEditorsAfterSplit, "split view should render two SQL editors");

    const secondarySql = "SELECT 20260518 AS split_marker;";
    await setPaneSql(page, ".websql-pane-2", secondarySql);
    const secondaryResponse = waitExecute(page, "split_marker");
    await page.locator(".websql-pane-2 .websql-execute-btn").first().click();
    summary.responses.secondary = await secondaryResponse;
    assertCheck(
      Number(summary.responses.secondary?.data?.row_count) === 1,
      "secondary split pane should execute SQL and return one row"
    );
    await page.getByText("split_marker").first().waitFor({ state: "visible", timeout: 15000 });
    summary.checks.secondaryExecution = true;

    const primarySql = "SELECT 18 AS primary_marker;";
    await setPaneSql(page, ".websql-primary-pane", primarySql);
    const primaryResponse = waitExecute(page, "primary_marker");
    await page.locator(".websql-primary-pane .websql-execute-btn").first().click();
    summary.responses.primary = await primaryResponse;
    assertCheck(
      Number(summary.responses.primary?.data?.row_count) === 1,
      "primary split pane should still execute SQL and return one row"
    );
    summary.checks.primaryExecution = true;

    await page.screenshot({ path: horizontalShot, fullPage: true });

    const pane2TabsBefore = await visibleCount(page, ".websql-pane-2 .workspace-websql-sql-tabs .ant-tabs-tab");
    await page.locator(".websql-pane-2 .websql-add-sql-btn").first().click();
    await page.waitForFunction(
      (before) => {
        const nodes = Array.from(document.querySelectorAll(".websql-pane-2 .workspace-websql-sql-tabs .ant-tabs-tab"));
        return nodes.filter((node) => {
          const rect = node.getBoundingClientRect();
          const style = window.getComputedStyle(node);
          return rect.width > 0 && rect.height > 0 && style.display !== "none" && style.visibility !== "hidden";
        }).length > before;
      },
      pane2TabsBefore,
      { timeout: 10000 }
    );
    summary.checks.addSqlTabInSplitPane = true;

    await rightClickSqlTab(page, ".websql-pane-2", "SQL 1");
    await chooseMenu(page, "左右分割");
    await page.locator(".websql-workbench-split.direction-horizontal .websql-pane-3").first().waitFor({
      state: "visible",
      timeout: 15000,
    });
    summary.checks.thirdPaneFromSecondPane = true;
    summary.checks.threePaneToolbarsAndTabs =
      (await visibleCount(page, ".websql-workbench-split .websql-pane-toolbar")) >= 3 &&
      (await visibleCount(page, ".websql-workbench-split .workspace-websql-sql-tabs")) >= 3 &&
      (await visibleCount(page, ".websql-workbench-split .websql-editor-body .monaco-editor")) >= 3;
    assertCheck(summary.checks.threePaneToolbarsAndTabs, "second split should add a third full SQL panel");

    const thirdSql = "SELECT 303 AS third_marker;";
    await setPaneSql(page, ".websql-pane-3", thirdSql);
    const thirdResponse = waitExecute(page, "third_marker");
    await page.locator(".websql-pane-3 .websql-execute-btn").first().click();
    summary.responses.third = await thirdResponse;
    assertCheck(
      Number(summary.responses.third?.data?.row_count) === 1,
      "third split pane should execute SQL and return one row"
    );
    summary.checks.thirdExecution = true;

    await rightClickSqlTab(page, ".websql-pane-3", "SQL 1");
    await chooseMenu(page, "上下分割");
    await page.locator(".websql-workbench-split.direction-vertical .websql-pane-4").first().waitFor({
      state: "visible",
      timeout: 15000,
    });
    summary.checks.verticalSplit = true;
    summary.checks.fourPaneAfterVerticalSplit =
      (await visibleCount(page, ".websql-workbench-split .websql-sql-pane")) >= 4 &&
      (await visibleCount(page, ".websql-workbench-split .workspace-websql-sql-tabs")) >= 4;
    assertCheck(summary.checks.fourPaneAfterVerticalSplit, "vertical split should add a fourth full SQL panel");
    await page.screenshot({ path: verticalShot, fullPage: true });

    assertCheck(summary.consoleErrors.length === 0, `console errors: ${summary.consoleErrors.join("\\n")}`);
    assertCheck(summary.pageErrors.length === 0, `page errors: ${summary.pageErrors.join("\\n")}`);
    assertCheck(summary.requestFailed.length === 0, "there are failed browser requests");

    summary.ok = true;
  } catch (error) {
    summary.ok = false;
    summary.error = error.message;
    await page.screenshot({ path: path.join(OUT_DIR, "websql-tab-split-fail.png"), fullPage: true }).catch(() => {});
    throw error;
  } finally {
    fs.writeFileSync(reportPath, `${JSON.stringify(summary, null, 2)}\n`);
    await browser.close();
  }
}

run().catch((error) => {
  console.error(error);
  process.exit(1);
});
