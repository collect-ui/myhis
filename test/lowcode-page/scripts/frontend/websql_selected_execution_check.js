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

const BASE_URL = process.env.WEBSQL_BASE_URL || "http://192.168.232.130:8015";
const TARGET_URL =
  process.env.WEBSQL_TARGET_URL ||
  `${BASE_URL}/collect-ui#/collect-ui/framework/webshell-editor-pool`;
const API_URL = process.env.WEBSQL_API_URL || `${BASE_URL}/template_data/data`;
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

function parseRequestData(request) {
  try {
    return JSON.parse(request.postData() || "{}");
  } catch (_error) {
    return {};
  }
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
  await sleep(800);
}

async function selectWebSQLEditorRange(page, markerStart, markerEnd) {
  const result = await page.evaluate(
    ({ markerStart: startText, markerEnd: endText }) => {
      const editors = window?.monaco?.editor?.getEditors?.() || [];
      const editor = editors.find((item) => {
        const node = item?.getContainerDomNode?.();
        return node && node.closest(".websql-lowcode");
      });
      if (!editor) {
        return { ok: false, reason: "websql monaco editor not found" };
      }
      const model = editor.getModel?.();
      const value = String(model?.getValue?.() || "");
      const start = value.indexOf(startText);
      const end = value.indexOf(endText);
      if (start < 0 || end < 0 || end < start) {
        return { ok: false, reason: `marker not found: ${startText} / ${endText}` };
      }
      const rangeEnd = end + endText.length;
      const startPos = model.getPositionAt(start);
      const endPos = model.getPositionAt(rangeEnd);
      const selection = new window.monaco.Selection(
        startPos.lineNumber,
        startPos.column,
        endPos.lineNumber,
        endPos.column
      );
      editor.focus();
      editor.setSelection(selection);
      editor.revealRangeInCenter?.(selection);
      return {
        ok: true,
        selectedText: String(model.getValueInRange(selection) || ""),
      };
    },
    { markerStart, markerEnd }
  );
  assertCheck(result.ok, result.reason || "failed to select editor range");
  await sleep(400);
  return String(result.selectedText || "");
}

async function clickToolbarExecute(page) {
  const button = page.locator(".websql-execute-btn").first();
  await button.waitFor({ state: "visible", timeout: 15000 });
  await button.click();
}

async function waitForExecutionResponse(page, trigger) {
  const responsePromise = page.waitForResponse((resp) => {
    if (!resp.url().includes("service=webshell.websql_execute")) {
      return false;
    }
    if (resp.request().method() !== "POST") {
      return false;
    }
    const payload = parseRequestData(resp.request());
    return payload.operation === "execute";
  }, { timeout: 30000 });
  await trigger();
  const response = await responsePromise;
  return {
    payload: parseRequestData(response.request()),
    json: await response.json(),
  };
}

async function triggerSuggest(page, sqlText) {
  await page.keyboard.press("Escape").catch(() => undefined);
  const result = await page.evaluate((value) => {
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    const editor = editors.find((item) => {
      const node = item?.getContainerDomNode?.();
      return node && node.closest(".websql-lowcode");
    });
    if (!editor) {
      return { ok: false, reason: "websql monaco editor not found" };
    }
    const marker = "|";
    const raw = String(value || "");
    const markerOffset = raw.indexOf(marker);
    const sqlValue =
      markerOffset >= 0 ? raw.slice(0, markerOffset) + raw.slice(markerOffset + 1) : raw;
    editor.setValue(sqlValue);
    const model = editor.getModel();
    const offset = markerOffset >= 0 ? Math.max(0, markerOffset) : model.getValueLength();
    const position = model.getPositionAt(offset);
    editor.setPosition(position);
    editor.focus();
    editor.trigger("websql-selected-execution-check", "editor.action.triggerSuggest", {});
    return { ok: true };
  }, sqlText);
  assertCheck(result.ok, result.reason || "trigger suggest failed");
  await sleep(500);
}

async function visibleSuggestionTexts(page) {
  return page.evaluate(() => {
    return Array.from(document.querySelectorAll(".suggest-widget .monaco-list-row"))
      .filter((el) => {
        const rect = el.getBoundingClientRect();
        const style = window.getComputedStyle(el);
        return rect.width > 0 && rect.height > 0 && style.visibility !== "hidden" && style.display !== "none";
      })
      .map((el) => String(el.textContent || "").replace(/\s+/g, " ").trim())
      .filter(Boolean);
  });
}

async function expectSuggestion(page, sqlText, predicate, errorMessage) {
  const started = Date.now();
  let lastTexts = [];
  while (Date.now() - started < 20000) {
    await triggerSuggest(page, sqlText);
    lastTexts = await visibleSuggestionTexts(page);
    if (lastTexts.some(predicate)) {
      return lastTexts;
    }
    await sleep(300);
  }
  throw new Error(`${errorMessage}; last=${JSON.stringify(lastTexts.slice(0, 20))}`);
}

async function readVisibleResultValues(page) {
  return page.evaluate(() => {
    const cells = Array.from(
      document.querySelectorAll(".websql-result-grid .ag-center-cols-container .ag-cell-value")
    );
    return cells
      .map((cell) => String(cell.textContent || "").trim())
      .filter(Boolean)
      .slice(0, 20);
  });
}

async function main() {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const tableName = `websql_selected_exec_${Date.now()}`;
  const screenshotPath = path.join(OUT_DIR, "websql-selected-execution-check.png");
  const reportPath = path.join(OUT_DIR, "websql-selected-execution-check.json");
  const summary = {
    url: TARGET_URL,
    tableName,
    buttonExecution: null,
    shortcutExecution: null,
    suggestions: {},
    consoleErrors: [],
    pageErrors: [],
    requestFailed: [],
    screenshot: screenshotPath,
    report: reportPath,
  };

  websql({
    operation: "execute",
    driver: "sqlite",
    sqlite_path: "./database/price.db",
    commit_mode: "direct",
    sql: `CREATE TABLE IF NOT EXISTS ${tableName} (id INTEGER PRIMARY KEY, name TEXT, selected_flag TEXT);`,
  });
  websql({
    operation: "execute",
    driver: "sqlite",
    sqlite_path: "./database/price.db",
    commit_mode: "direct",
    sql: `INSERT INTO ${tableName}(name, selected_flag) VALUES ('alpha', 'button'), ('beta', 'shortcut');`,
  });

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1600, height: 960 } });

  page.on("console", (msg) => {
    if (msg.type() === "error") {
      summary.consoleErrors.push(msg.text());
    }
  });
  page.on("pageerror", (error) => {
    summary.pageErrors.push(String(error));
  });
  page.on("requestfailed", (request) => {
    summary.requestFailed.push({
      url: request.url(),
      method: request.method(),
      errorText: request.failure()?.errorText || "",
    });
  });

  try {
    await gotoPage(page, TARGET_URL);
    await clearWebSQLLocalState(page);
    await gotoPage(page, TARGET_URL);
    await openWebSQLPanel(page);
    await page.waitForSelector(".websql-lowcode .monaco-editor", {
      state: "visible",
      timeout: 45000,
    });

    const keywordSuggestions = await expectSuggestion(
      page,
      "SEL|",
      (text) => text.includes("SELECT"),
      "keyword suggestion SELECT not found"
    );
    summary.suggestions.keyword = keywordSuggestions.slice(0, 12);

    const tableSuggestions = await expectSuggestion(
      page,
      `SELECT * FROM ${tableName.slice(0, Math.max(1, tableName.length - 3))}|`,
      (text) => text.includes(tableName),
      `table suggestion not found: ${tableName}`
    );
    summary.suggestions.table = tableSuggestions.slice(0, 12);

    const columnSuggestions = await expectSuggestion(
      page,
      `SELECT t.| FROM ${tableName} t`,
      (text) => text.includes("selected_flag"),
      "column suggestion selected_flag not found"
    );
    summary.suggestions.column = columnSuggestions.slice(0, 12);

    const buttonSql = [
      "SELECT 'skip-button' AS ignored_marker;",
      `SELECT name, selected_flag FROM ${tableName} WHERE selected_flag = 'button' ORDER BY id;`,
    ].join("\n");
    await setWebSQLEditorValue(page, buttonSql);
    const selectedButtonSql = await selectWebSQLEditorRange(
      page,
      `SELECT name, selected_flag FROM ${tableName}`,
      "ORDER BY id;"
    );
    const buttonExecution = await waitForExecutionResponse(page, async () => {
      await clickToolbarExecute(page);
    });
    summary.buttonExecution = {
      selectedText: selectedButtonSql,
      requestSql: String(buttonExecution.payload.sql || ""),
      responseCode: String(buttonExecution.json.code || ""),
      rowCount: Number(buttonExecution.json?.data?.row_count || 0),
      statementType: String(buttonExecution.json?.data?.statement_type || ""),
    };
    assertCheck(
      String(buttonExecution.payload.sql || "").trim() === selectedButtonSql.trim(),
      "button execution did not send selected SQL"
    );
    assertCheck(
      Number(buttonExecution.json?.data?.row_count || 0) === 1,
      "button execution row count mismatch"
    );

    const shortcutSql = [
      "SELECT 'skip-shortcut' AS ignored_marker;",
      `SELECT name, selected_flag FROM ${tableName} WHERE selected_flag = 'shortcut' ORDER BY id;`,
    ].join("\n");
    await setWebSQLEditorValue(page, shortcutSql);
    const selectedShortcutSql = await selectWebSQLEditorRange(
      page,
      `SELECT name, selected_flag FROM ${tableName}`,
      "ORDER BY id;"
    );
    const shortcutExecution = await waitForExecutionResponse(page, async () => {
      await page.keyboard.press("Control+Alt+Enter");
    });
    summary.shortcutExecution = {
      selectedText: selectedShortcutSql,
      requestSql: String(shortcutExecution.payload.sql || ""),
      responseCode: String(shortcutExecution.json.code || ""),
      rowCount: Number(shortcutExecution.json?.data?.row_count || 0),
      statementType: String(shortcutExecution.json?.data?.statement_type || ""),
    };
    assertCheck(
      String(shortcutExecution.payload.sql || "").trim() === selectedShortcutSql.trim(),
      "shortcut execution did not send selected SQL"
    );
    assertCheck(
      Number(shortcutExecution.json?.data?.row_count || 0) === 1,
      "shortcut execution row count mismatch"
    );

    await sleep(1200);
    const visibleValues = await readVisibleResultValues(page);
    summary.visibleValues = visibleValues;
    assertCheck(
      visibleValues.includes("beta") || visibleValues.includes("shortcut"),
      `result grid did not show shortcut result: ${JSON.stringify(visibleValues)}`
    );

    await page.screenshot({ path: screenshotPath, fullPage: true });
    assertCheck(summary.consoleErrors.length === 0, `console errors: ${summary.consoleErrors.join(" | ")}`);
    assertCheck(summary.pageErrors.length === 0, `page errors: ${summary.pageErrors.join(" | ")}`);
    assertCheck(summary.requestFailed.length === 0, `request failed: ${JSON.stringify(summary.requestFailed)}`);
  } finally {
    fs.writeFileSync(reportPath, JSON.stringify(summary, null, 2));
    await page.close().catch(() => undefined);
    await browser.close().catch(() => undefined);
    try {
      websql({
        operation: "execute",
        driver: "sqlite",
        sqlite_path: "./database/price.db",
        commit_mode: "direct",
        sql: `DROP TABLE IF EXISTS ${tableName};`,
      });
    } catch (_error) {
      // ignore cleanup failure
    }
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
