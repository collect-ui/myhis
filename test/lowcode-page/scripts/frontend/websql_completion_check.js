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

async function findWebSQLEditor(page) {
  return page.evaluateHandle(() => {
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    return (
      editors.find((editor) => {
        const node = editor?.getContainerDomNode?.();
        return node && node.closest(".websql-lowcode");
      }) || null
    );
  });
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

function hasTableLikeSuggestion(texts) {
  return texts.some((text) => /(?:Table|View) · \d+ fields/.test(String(text)));
}

function hasTextIncluding(texts, value) {
  return texts.some((text) => String(text).includes(value));
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
    const sqlText =
      markerOffset >= 0
        ? raw.slice(0, markerOffset) + raw.slice(markerOffset + marker.length)
        : raw;
    editor.setValue(sqlText);
    const model = editor.getModel();
    const offset =
      markerOffset >= 0 ? Math.max(0, markerOffset) : model.getValueLength();
    const position = model.getPositionAt(offset);
    editor.setPosition(position);
    editor.focus();
    editor.trigger("websql-completion-check", "editor.action.triggerSuggest", {});
    return { ok: true, offset, position, uri: String(model.uri || "") };
  }, sqlText);
  assertCheck(result.ok, result.reason || "trigger suggest failed");
  return result;
}

async function expectSuggestion(page, sqlText, expected, timeout = 20000) {
  const started = Date.now();
  let lastTexts = [];
  while (Date.now() - started < timeout) {
    await triggerSuggest(page, sqlText);
    await sleep(450);
    lastTexts = await visibleSuggestionTexts(page);
    if (lastTexts.some((text) => text.includes(expected))) {
      return lastTexts;
    }
    await sleep(450);
  }
  throw new Error(`suggestion not found: ${expected}; last=${JSON.stringify(lastTexts.slice(0, 20))}`);
}

async function readSuggestions(page, sqlText, timeout = 20000) {
  const started = Date.now();
  let lastTexts = [];
  while (Date.now() - started < timeout) {
    await triggerSuggest(page, sqlText);
    await sleep(450);
    lastTexts = await visibleSuggestionTexts(page);
    if (lastTexts.length > 0) {
      return lastTexts;
    }
    await sleep(300);
  }
  return lastTexts;
}

async function expectSuggestions(page, options) {
  const {
    sqlText,
    include = [],
    exclude = [],
    expectTableLike,
    firstIncludes,
    timeout,
  } = options;
  const texts = await readSuggestions(page, sqlText, timeout);
  if (firstIncludes) {
    assertCheck(
      String(texts[0] || "").includes(firstIncludes),
      `first suggestion mismatch: expected=${firstIncludes}; sql=${sqlText}; last=${JSON.stringify(texts.slice(0, 20))}`
    );
  }
  include.forEach((value) => {
    assertCheck(
      hasTextIncluding(texts, value),
      `suggestion not found: ${value}; sql=${sqlText}; last=${JSON.stringify(texts.slice(0, 20))}`
    );
  });
  exclude.forEach((value) => {
    assertCheck(
      !hasTextIncluding(texts, value),
      `unexpected suggestion found: ${value}; sql=${sqlText}; last=${JSON.stringify(texts.slice(0, 20))}`
    );
  });
  if (expectTableLike === true) {
    assertCheck(
      hasTableLikeSuggestion(texts),
      `expected table suggestions; sql=${sqlText}; last=${JSON.stringify(texts.slice(0, 20))}`
    );
  }
  if (expectTableLike === false) {
    assertCheck(
      !hasTableLikeSuggestion(texts),
      `unexpected table suggestions; sql=${sqlText}; last=${JSON.stringify(texts.slice(0, 20))}`
    );
  }
  return texts;
}

async function expectNoSuggestions(page, sqlText, timeout = 5000) {
  const started = Date.now();
  let lastTexts = [];
  while (Date.now() - started < timeout) {
    await triggerSuggest(page, sqlText);
    await sleep(350);
    lastTexts = await visibleSuggestionTexts(page);
    if (lastTexts.length === 0) {
      return [];
    }
    await page.keyboard.press("Escape").catch(() => undefined);
    await sleep(200);
  }
  throw new Error(`expected no suggestions; sql=${sqlText}; last=${JSON.stringify(lastTexts.slice(0, 20))}`);
}

async function main() {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const tableName = `websql_completion_demo_${Date.now()}`;
  const otherTableName = `websql_completion_side_${Date.now()}`;
  const screenshotPath = path.join(OUT_DIR, "websql-completion-check.png");
  const reportPath = path.join(OUT_DIR, "websql-completion-check.json");
  const summary = {
    url: TARGET_URL,
    tableName,
    otherTableName,
    checks: {},
    suggestions: {},
    completionResponse: null,
    consoleErrors: [],
    pageErrors: [],
    requestFailed: [],
    screenshot: screenshotPath,
    report: reportPath,
  };

  websql({
    operation: "execute",
    sql: `CREATE TABLE IF NOT EXISTS ${tableName} (id INTEGER PRIMARY KEY, name TEXT, completion_note TEXT);`,
  });
  websql({
    operation: "execute",
    sql: `INSERT INTO ${tableName}(name, completion_note) VALUES ('alpha', 'seed');`,
  });
  websql({
    operation: "execute",
    sql: `CREATE TABLE IF NOT EXISTS ${otherTableName} (id INTEGER PRIMARY KEY, other_flag TEXT, other_remark TEXT);`,
  });
  websql({
    operation: "execute",
    sql: `INSERT INTO ${otherTableName}(other_flag, other_remark) VALUES ('probe', 'side');`,
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

  const completionResponsePromise = page
    .waitForResponse((resp) => {
      if (!resp.url().includes("service=webshell.websql_execute") || resp.request().method() !== "POST") {
        return false;
      }
      const payload = parseRequestData(resp.request());
      return payload.operation === "schema" && payload.schema_scope === "completion";
    }, { timeout: 30000 })
    .catch(() => null);

  try {
    await page.goto(TARGET_URL, { waitUntil: "domcontentloaded", timeout: 45000 });
    await page.waitForSelector(".websql-lowcode .monaco-editor", { state: "visible", timeout: 45000 });
    await page.waitForFunction(() => !!window?.monaco?.editor?.getEditors?.().length, null, { timeout: 30000 });
    const completionResponse = await completionResponsePromise;
    if (completionResponse) {
      const json = await completionResponse.json();
      const data = json.data || {};
      summary.completionResponse = {
        code: json.code,
        tableCount: Array.isArray(data.tables) ? data.tables.length : 0,
        viewCount: Array.isArray(data.views) ? data.views.length : 0,
      };
    }

    const hasEditor = await page.evaluate(() => {
      const editors = window?.monaco?.editor?.getEditors?.() || [];
      return editors.some((editor) => {
        const node = editor?.getContainerDomNode?.();
        return node && node.closest(".websql-lowcode");
      });
    });
    assertCheck(hasEditor, "websql editor handle missing");
    summary.checks.pageOpened = true;
    summary.checks.editorVisible = true;

    summary.suggestions.keyword = await expectSuggestions(page, {
      sqlText: "sel|",
      include: ["SELECT"],
      firstIncludes: "SELECT",
      expectTableLike: false,
    });
    summary.checks.keywordSuggest = true;

    summary.suggestions.fromTable = await expectSuggestions(page, {
      sqlText: "SELECT * FROM websql_completion_demo_|",
      include: [tableName],
      exclude: [`completion_note${tableName}`],
      expectTableLike: true,
    });
    summary.checks.fromTableSuggest = true;

    summary.suggestions.joinTable = await expectSuggestions(page, {
      sqlText: `SELECT * FROM ${tableName} d JOIN websql_completion_side_|`,
      include: [otherTableName],
      exclude: [`completion_note${tableName}`],
      expectTableLike: true,
    });
    summary.checks.joinTableSuggest = true;

    summary.suggestions.postTableKeyword = await expectSuggestions(page, {
      sqlText: `SELECT * FROM ${tableName} d |`,
      include: ["WHERE", "JOIN"],
      exclude: ["SELECT", "FROM"],
      expectTableLike: false,
    });
    summary.checks.postTableKeyword = true;

    summary.suggestions.joinOnKeyword = await expectSuggestions(page, {
      sqlText: `SELECT * FROM ${tableName} d JOIN ${otherTableName} s |`,
      include: ["ON"],
      exclude: ["SELECT", "FROM", "WHERE", "NULL"],
      expectTableLike: false,
      firstIncludes: "ON",
    });
    summary.checks.joinOnKeyword = true;

    summary.suggestions.whereField = await expectSuggestions(page, {
      sqlText: `SELECT * FROM ${tableName} d WHERE com|`,
      include: ["completion_note"],
      exclude: [otherTableName, "other_flag"],
      expectTableLike: false,
    });
    summary.checks.whereFieldSuggest = true;

    summary.suggestions.whereLikeKeywordPrefix = await expectSuggestions(page, {
      sqlText: `SELECT * FROM ${tableName} d WHERE name li|`,
      include: ["LIKE"],
      exclude: [otherTableName, "completion_note"],
      expectTableLike: false,
      firstIncludes: "LIKE",
    });
    summary.checks.whereLikeKeywordPrefix = true;

    summary.suggestions.qualifiedField = await expectSuggestions(page, {
      sqlText: `SELECT d.com| FROM ${tableName} d`,
      include: ["completion_note"],
      exclude: [otherTableName, "other_flag"],
      expectTableLike: false,
    });
    summary.checks.qualifiedFieldSuggest = true;

    summary.suggestions.insertField = await expectSuggestions(page, {
      sqlText: `INSERT INTO ${tableName}(|`,
      include: ["completion_note"],
      exclude: [otherTableName],
      expectTableLike: false,
    });
    summary.checks.insertFieldSuggest = true;

    summary.suggestions.valueLiteral = await expectSuggestions(page, {
      sqlText: `SELECT * FROM ${tableName} d WHERE d.id = |`,
      include: ["NULL", "TRUE", "FALSE"],
      exclude: ["completion_note"],
      expectTableLike: false,
      firstIncludes: "NULL",
    });
    summary.checks.valueLiteralSuggest = true;

    summary.suggestions.insertValuesLiteral = await expectSuggestions(page, {
      sqlText: `INSERT INTO ${tableName} VALUES (|`,
      include: ["NULL", "TRUE", "FALSE"],
      exclude: ["completion_note", otherTableName],
      expectTableLike: false,
      firstIncludes: "NULL",
    });
    summary.checks.insertValuesLiteralSuggest = true;

    summary.suggestions.updateSetKeyword = await expectSuggestions(page, {
      sqlText: `UPDATE ${tableName} SET completion_note = 'done' |`,
      include: ["WHERE", "ORDER BY"],
      exclude: ["SELECT", "FROM", "JOIN", "GROUP BY", "NULL"],
      expectTableLike: false,
      firstIncludes: "WHERE",
    });
    summary.checks.updateSetKeyword = true;

    summary.suggestions.whereConditionKeyword = await expectSuggestions(page, {
      sqlText: `SELECT * FROM ${tableName} d WHERE d.id = 1 |`,
      include: ["AND", "ORDER BY"],
      exclude: ["completion_note"],
      expectTableLike: false,
      firstIncludes: "AND",
    });
    summary.checks.whereConditionKeyword = true;

    summary.suggestions.groupByKeyword = await expectSuggestions(page, {
      sqlText: `SELECT * FROM ${tableName} d GROUP|`,
      include: ["GROUP BY"],
      exclude: ["SELECT", "FROM"],
      expectTableLike: false,
      firstIncludes: "GROUP BY",
    });
    summary.checks.groupByKeyword = true;

    summary.suggestions.insertIntoKeyword = await expectSuggestions(page, {
      sqlText: "INSERT |",
      include: ["INTO"],
      exclude: [tableName, "NULL"],
      expectTableLike: false,
      firstIncludes: "INTO",
    });
    summary.checks.insertIntoKeyword = true;

    summary.suggestions.deleteFromKeyword = await expectSuggestions(page, {
      sqlText: "DELETE |",
      include: ["FROM"],
      exclude: [tableName, "NULL"],
      expectTableLike: false,
      firstIncludes: "FROM",
    });
    summary.checks.deleteFromKeyword = true;

    summary.suggestions.deletePostTableKeyword = await expectSuggestions(page, {
      sqlText: `DELETE FROM ${tableName} |`,
      include: ["WHERE", "ORDER BY"],
      exclude: ["JOIN", "GROUP BY", "SELECT", "NULL"],
      expectTableLike: false,
      firstIncludes: "WHERE",
    });
    summary.checks.deletePostTableKeyword = true;

    summary.suggestions.multiStatementIsolation = await expectSuggestions(page, {
      sqlText: `SELECT * FROM ${otherTableName} x WHERE x.other_flag;\nSELECT * FROM ${tableName} d WHERE |`,
      include: ["completion_note"],
      exclude: ["other_flag"],
      expectTableLike: false,
    });
    summary.checks.multiStatementIsolation = true;

    summary.suggestions.commentSilent = await expectNoSuggestions(
      page,
      "-- completion comment |"
    );
    summary.checks.commentSilent = true;

    summary.suggestions.createTableSilent = await expectNoSuggestions(
      page,
      "CREATE TABLE |"
    );
    summary.checks.createTableSilent = true;

    await page.screenshot({ path: screenshotPath, fullPage: true });
    assertCheck(summary.consoleErrors.length === 0, `console errors: ${summary.consoleErrors.join("\n")}`);
    assertCheck(summary.pageErrors.length === 0, `page errors: ${summary.pageErrors.join("\n")}`);
    assertCheck(summary.requestFailed.length === 0, `requestfailed: ${JSON.stringify(summary.requestFailed)}`);
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
    websql({
      operation: "execute",
      sql: `DROP TABLE IF EXISTS ${otherTableName};`,
    });
    fs.writeFileSync(reportPath, JSON.stringify(summary, null, 2));
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
