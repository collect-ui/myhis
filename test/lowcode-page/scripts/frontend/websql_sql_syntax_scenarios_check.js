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
const OUT_DIR =
  process.env.WEBSQL_OUTPUT_DIR ||
  "/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation";

const DOM_SQL = (tableName, marker) =>
  `SELECT COUNT(*) AS total FROM ${tableName} WHERE type = 'table' AND name LIKE '${marker}%'; -- ${marker}`;

const SQL_TOKEN_SCENARIOS = [
  {
    name: "select-cte-pagination-comment",
    sql: "WITH RECURSIVE cnt(x) AS (SELECT 1 UNION ALL SELECT x + 1 FROM cnt WHERE x < 10) SELECT COUNT(*) AS total FROM cnt WHERE x BETWEEN 1 AND 9 ORDER BY total DESC LIMIT 5 OFFSET 0; -- read",
    expect: [
      ["WITH", "keyword"],
      ["RECURSIVE", "keyword"],
      ["SELECT", "keyword"],
      ["COUNT", "predefined"],
      ["FROM", "keyword"],
      ["WHERE", "keyword"],
      ["BETWEEN", "keyword"],
      ["ORDER", "keyword"],
      ["BY", "keyword"],
      ["DESC", "keyword"],
      ["LIMIT", "keyword"],
      ["OFFSET", "keyword"],
      ["-- read", "comment"],
    ],
  },
  {
    name: "insert-update-delete-returning",
    sql: "INSERT INTO users(id,name) VALUES (1,'a'); UPDATE users SET name = 'b' WHERE id = 1 RETURNING id; DELETE FROM users WHERE id IN (1,2);",
    expect: [
      ["INSERT", "keyword"],
      ["INTO", "keyword"],
      ["VALUES", "keyword"],
      ["UPDATE", "keyword"],
      ["SET", "keyword"],
      ["WHERE", "keyword"],
      ["RETURNING", "keyword"],
      ["DELETE", "keyword"],
      ["FROM", "keyword"],
      ["IN", "keyword"],
      ["'a'", "string"],
    ],
  },
  {
    name: "ddl-create-alter-drop",
    sql: "CREATE TEMP TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT DEFAULT 'n', CONSTRAINT uk UNIQUE(name)); ALTER TABLE users ADD COLUMN meta JSON; DROP INDEX IF EXISTS idx_users_name;",
    expect: [
      ["CREATE", "keyword"],
      ["TEMP", "keyword"],
      ["TABLE", "keyword"],
      ["IF", "keyword"],
      ["EXISTS", "keyword"],
      ["INTEGER", "keyword"],
      ["PRIMARY", "keyword"],
      ["KEY", "keyword"],
      ["TEXT", "keyword"],
      ["DEFAULT", "keyword"],
      ["CONSTRAINT", "keyword"],
      ["UNIQUE", "keyword"],
      ["ALTER", "keyword"],
      ["ADD", "keyword"],
      ["COLUMN", "keyword"],
      ["JSON", "keyword"],
      ["DROP", "keyword"],
      ["INDEX", "keyword"],
    ],
  },
  {
    name: "transaction-control",
    sql: "BEGIN TRANSACTION; SAVEPOINT s1; RELEASE s1; COMMIT; ROLLBACK;",
    expect: [
      ["BEGIN", "keyword"],
      ["TRANSACTION", "keyword"],
      ["SAVEPOINT", "keyword"],
      ["RELEASE", "keyword"],
      ["COMMIT", "keyword"],
      ["ROLLBACK", "keyword"],
    ],
  },
  {
    name: "join-window-operators",
    sql: "SELECT u.id, ROW_NUMBER() OVER (PARTITION BY u.name ORDER BY u.id) AS rn FROM users u LEFT JOIN orders o ON o.user_id = u.id WHERE o.total >= 10;",
    expect: [
      ["SELECT", "keyword"],
      ["OVER", "keyword"],
      ["PARTITION", "keyword"],
      ["BY", "keyword"],
      ["ORDER", "keyword"],
      ["AS", "keyword"],
      ["FROM", "keyword"],
      ["LEFT", "keyword"],
      ["JOIN", "keyword"],
      ["ON", "keyword"],
      ["WHERE", "keyword"],
      [">=", "operator"],
      ["10", "number"],
    ],
  },
  {
    name: "quoted-identifiers-block-comment",
    sql: "SELECT `tick_name`, [odd name], 'literal' FROM \"quoted table\" WHERE note LIKE '%--x%' /* block comment */;",
    expect: [
      ["SELECT", "keyword"],
      ["`tick_name`", "identifier"],
      ["'literal'", "string"],
      ["FROM", "keyword"],
      ["WHERE", "keyword"],
      ["LIKE", "keyword"],
      ["/* block comment */", "comment"],
    ],
  },
];

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function assertCheck(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function normalizeText(value) {
  return String(value || "")
    .replace(/\u00a0/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function brightness(rgb) {
  const matched = String(rgb || "").match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
  if (!matched) {
    return -1;
  }
  return (Number(matched[1]) + Number(matched[2]) + Number(matched[3])) / 3;
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
  await sleep(1800);
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

async function waitForWebsqlEditor(page) {
  await page.locator(".websql-lowcode .monaco-editor").first().waitFor({
    state: "visible",
    timeout: 45000,
  });
  await page.waitForFunction(
    () => {
      const editors = window.monaco?.editor?.getEditors?.() || [];
      return editors.some((editor) => editor?.getContainerDomNode?.()?.closest(".websql-lowcode"));
    },
    null,
    { timeout: 45000 }
  );
}

async function setEditorValue(page, scope, value) {
  const result = await page.evaluate(
    ({ scope: targetScope, value: nextValue }) => {
      const visible = (node) => {
        if (!node) return false;
        const rect = node.getBoundingClientRect();
        return rect.width > 100 && rect.height > 80;
      };
      const editors = window.monaco?.editor?.getEditors?.() || [];
      const editor = editors.find((item) => {
        const node = item?.getContainerDomNode?.();
        if (!node || !visible(node)) return false;
        if (targetScope === "ddl") return !!node.closest(".websql-ddl-editor");
        return !!node.closest(".websql-editor-body") && !node.closest(".websql-ddl-editor");
      });
      if (!editor) {
        return { ok: false, reason: `editor not found: ${targetScope}` };
      }
      editor.setValue(String(nextValue || ""));
      editor.focus();
      return {
        ok: true,
        uri: String(editor.getModel?.()?.uri || ""),
        language: String(editor.getModel?.()?.getLanguageId?.() || ""),
      };
    },
    { scope, value }
  );
  assertCheck(result.ok, result.reason || `set editor value failed: ${scope}`);
  await sleep(700);
  return result;
}

async function readEditorSyntax(page, scope) {
  return page.evaluate((targetScope) => {
    const normalize = (value) =>
      String(value || "")
        .replace(/\u00a0/g, " ")
        .replace(/\s+/g, " ")
        .trim();
    const visible = (node) => {
      if (!node) return false;
      const rect = node.getBoundingClientRect();
      return rect.width > 100 && rect.height > 80;
    };
    const tokenize = (sql) => {
      const lines = String(sql || "").split(/\r\n|\n|\r/);
      const rawLines = window.monaco?.editor?.tokenize?.(String(sql || ""), "sql") || [];
      let baseOffset = 0;
      const out = [];
      rawLines.forEach((lineTokens, lineIndex) => {
        const lineText = lines[lineIndex] || "";
        lineTokens.forEach((token, index) => {
          const next = lineTokens[index + 1]?.offset ?? lineText.length;
          const text = lineText.slice(token.offset, next);
          if (!normalize(text)) return;
          out.push({
            line: lineIndex + 1,
            offset: baseOffset + token.offset,
            text,
            normalizedText: normalize(text),
            type: String(token.type || ""),
          });
        });
        baseOffset += lineText.length + 1;
      });
      return out;
    };
    const editors = window.monaco?.editor?.getEditors?.() || [];
    const editor = editors.find((item) => {
      const node = item?.getContainerDomNode?.();
      if (!node || !visible(node)) return false;
      if (targetScope === "ddl") return !!node.closest(".websql-ddl-editor");
      return !!node.closest(".websql-editor-body") && !node.closest(".websql-ddl-editor");
    });
    const node = editor?.getContainerDomNode?.();
    const value = String(editor?.getValue?.() || "");
    const backgroundNode = node?.querySelector(".monaco-editor-background") || node;
    const spans = Array.from(node?.querySelectorAll(".view-lines .view-line span") || [])
      .map((span) => ({
        text: normalize(span.textContent),
        className: String(span.className || ""),
        color: window.getComputedStyle(span).color || "",
      }))
      .filter((item) => item.text);
    const tokenColors = Array.from(new Set(spans.map((item) => item.color).filter(Boolean)));
    return {
      scope: targetScope,
      found: !!editor,
      language: String(editor?.getModel?.()?.getLanguageId?.() || ""),
      value,
      localTheme: String(node?.getAttribute("data-sport-ui-local-theme") || ""),
      editorBg: backgroundNode ? window.getComputedStyle(backgroundNode).backgroundColor : "",
      monacoClass: String(node?.className || ""),
      tokens: tokenize(value),
      domTokens: spans.slice(0, 80),
      tokenColors,
      distinctTokenColorCount: tokenColors.length,
    };
  }, scope);
}

async function tokenizeScenarios(page) {
  return page.evaluate((cases) => {
    const normalize = (value) =>
      String(value || "")
        .replace(/\u00a0/g, " ")
        .replace(/\s+/g, " ")
        .trim();
    return cases.map((item) => {
      const sql = String(item.sql || "");
      const raw = window.monaco?.editor?.tokenize?.(sql, "sql")?.[0] || [];
      const tokens = raw
        .map((token, index) => {
          const next = raw[index + 1]?.offset ?? sql.length;
          const text = sql.slice(token.offset, next);
          return {
            offset: token.offset,
            text,
            normalizedText: normalize(text),
            type: String(token.type || ""),
          };
        })
        .filter((token) => token.normalizedText);
      return { name: item.name, sql, tokens };
    });
  }, SQL_TOKEN_SCENARIOS.map(({ name, sql }) => ({ name, sql })));
}

function findToken(tokens, text) {
  const expected = normalizeText(text).toUpperCase();
  return tokens.find((token) => normalizeText(token.text).toUpperCase() === expected);
}

function assertTokenType(tokens, text, typePrefix, label) {
  const token = findToken(tokens, text);
  assertCheck(!!token, `${label}: token not found: ${text}`);
  assertCheck(
    String(token.type || "").startsWith(typePrefix),
    `${label}: ${text} expected ${typePrefix}, got ${token.type}`
  );
  return token;
}

function assertTokenizerMatrix(matrix, summary) {
  summary.tokenizer = {};
  SQL_TOKEN_SCENARIOS.forEach((scenario) => {
    const actual = matrix.find((item) => item.name === scenario.name);
    assertCheck(!!actual, `tokenizer scenario missing: ${scenario.name}`);
    scenario.expect.forEach(([text, type]) => {
      assertTokenType(actual.tokens, text, type, scenario.name);
    });
    summary.tokenizer[scenario.name] = {
      tokenCount: actual.tokens.length,
      checked: scenario.expect.length,
      samples: actual.tokens.slice(0, 24),
    };
  });
}

function assertDomSyntax(info, label, requiredTexts, options = {}) {
  const minColors = Number(options.minColors || 5);
  assertCheck(info.found, `${label}: editor missing`);
  assertCheck(info.language === "sql", `${label}: expected sql language, got ${info.language}`);
  assertCheck(info.localTheme === "light", `${label}: expected local light theme, got ${info.localTheme}`);
  assertCheck(brightness(info.editorBg) >= 245, `${label}: editor is not light: ${info.editorBg}`);
  assertCheck(info.distinctTokenColorCount >= minColors, `${label}: token colors too few: ${info.distinctTokenColorCount}`);
  requiredTexts.forEach(([text, type]) => assertTokenType(info.tokens, text, type, label));
  const keywordDom = info.domTokens.find((item) => normalizeText(item.text).toUpperCase() === "SELECT");
  if (keywordDom) {
    const identifierDom = info.domTokens.find((item) => normalizeText(item.text).includes("total"));
    assertCheck(
      !identifierDom || keywordDom.color !== identifierDom.color,
      `${label}: keyword color matches identifier color`
    );
  }
}

async function clickAddSqlTab(page) {
  await page.locator(".websql-add-sql-btn").first().click();
  await sleep(1000);
}

async function setEditorValueAndResolvePoint(page, sql, tableName) {
  const result = await page.evaluate(
    ({ sqlText, table }) => {
      const editors = window.monaco?.editor?.getEditors?.() || [];
      const editor = editors.find((item) => {
        const node = item?.getContainerDomNode?.();
        return node && node.closest(".websql-editor-body") && !node.closest(".websql-ddl-editor");
      });
      if (!editor) {
        return { ok: false, reason: "websql editor not found" };
      }
      editor.setValue(sqlText);
      const model = editor.getModel();
      const index = String(sqlText).indexOf(table);
      const offset = index >= 0 ? index + Math.max(1, Math.floor(table.length / 2)) : 0;
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
      };
    },
    { sqlText: sql, table: tableName }
  );
  assertCheck(result.ok, result.reason || "failed to resolve SQL table point");
  await sleep(700);
  return result;
}

async function readContextMenuItems(page) {
  const labels = page.locator(".monaco-menu .action-label");
  const count = await labels.count();
  const items = [];
  for (let index = 0; index < count; index += 1) {
    const item = labels.nth(index);
    const text = String(await item.innerText().catch(() => "")).trim();
    const box = await item.boundingBox().catch(() => null);
    items.push({
      text,
      x: box ? box.x + box.width / 2 : 0,
      y: box ? box.y + box.height / 2 : 0,
    });
  }
  return items;
}

async function clickContextMenuItem(page, items, label) {
  const index = items.findIndex((entry) => String(entry.text || "").includes(label));
  assertCheck(index >= 0, `context menu item not found: ${label}; got ${items.map((item) => item.text).join(",")}`);
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
  const labelLocator = page.locator(".monaco-menu .action-label").filter({ hasText: label }).first();
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
    // Fall through to keyboard navigation if Monaco keeps the row outside normal hit testing.
  }
  for (let i = 0; i < index; i += 1) {
    await page.keyboard.press("ArrowDown");
  }
  await page.keyboard.press("Enter");
}

async function openDdlDialog(page, tableName) {
  const sql = `SELECT * FROM ${tableName} t WHERE t.id = 1;`;
  const point = await setEditorValueAndResolvePoint(page, sql, tableName);
  await page.mouse.click(point.x, point.y, { button: "right" });
  await page.waitForSelector(".monaco-menu .action-label", {
    state: "visible",
    timeout: 10000,
  });
  const menuItems = await readContextMenuItems(page);
  const responsePromise = page.waitForResponse(
    (resp) => {
      if (!resp.url().includes("service=webshell.websql_execute") || resp.request().method() !== "POST") {
        return false;
      }
      const payload = parseRequestData(resp.request());
      return payload.operation === "object_detail" && String(payload.object_name || "") === tableName;
    },
    { timeout: 30000 }
  );
  await clickContextMenuItem(page, menuItems, "查 DDL");
  await responsePromise;
  await page.waitForSelector(".websql-ddl-editor .monaco-editor", {
    state: "visible",
    timeout: 20000,
  });
  await sleep(1000);
}

async function main() {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const stamp = Date.now();
  const tableName = `websql_syntax_${stamp}`;
  const marker = `syntax_${stamp}`;
  const reportPath = path.join(OUT_DIR, "websql-sql-syntax-scenarios-check.json");
  const screenshots = {
    standalone: path.join(OUT_DIR, "websql-sql-syntax-standalone.png"),
    ddl: path.join(OUT_DIR, "websql-sql-syntax-ddl.png"),
    webshell: path.join(OUT_DIR, "websql-sql-syntax-webshell.png"),
  };
  const summary = {
    standaloneUrl: STANDALONE_URL,
    webshellUrl: WEBSHELL_URL,
    tableName,
    checks: {},
    tokenizer: {},
    editors: {},
    screenshots,
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    pass: false,
  };

  websql({
    operation: "execute",
    sql: `CREATE TABLE IF NOT EXISTS ${tableName} (id INTEGER PRIMARY KEY, type TEXT DEFAULT 'table', name TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);`,
  });

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    ignoreHTTPSErrors: true,
    viewport: { width: 1680, height: 980 },
  });
  await context.addInitScript(() => {
    for (const key of Object.keys(window.localStorage || {})) {
      if (
        key.startsWith("workspace-websql-connections") ||
        key.startsWith("workspace-websql-recent") ||
        key.startsWith("workspace-websql-favorites") ||
        key.startsWith("websql-lowcode")
      ) {
        window.localStorage.removeItem(key);
      }
    }
  });
  const page = await context.newPage();
  page.on("console", (msg) => {
    if (msg.type() === "error") summary.consoleErrors.push(msg.text());
  });
  page.on("pageerror", (error) => summary.pageErrors.push(String(error)));
  page.on("requestfailed", (req) => {
    summary.failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || "failed"}`);
  });

  try {
    await gotoPage(page, STANDALONE_URL);
    await clearWebSQLLocalState(page);
    await gotoPage(page, STANDALONE_URL);
    await openWebSQLPanel(page);
    await waitForWebsqlEditor(page);
    assertTokenizerMatrix(await tokenizeScenarios(page), summary);
    summary.checks.tokenizerMatrix = true;

    await setEditorValue(page, "websql", DOM_SQL(tableName, marker));
    const standaloneMain = await readEditorSyntax(page, "websql");
    assertDomSyntax(standaloneMain, "standalone-main", [
      ["SELECT", "keyword"],
      ["COUNT", "predefined"],
      ["FROM", "keyword"],
      ["WHERE", "keyword"],
      ["'table'", "string"],
      [`-- ${marker}`, "comment"],
    ]);
    summary.editors.standaloneMain = standaloneMain;
    summary.checks.standaloneMainSyntax = true;

    await clickAddSqlTab(page);
    await setEditorValue(page, "websql", DOM_SQL(tableName, `${marker}_tab`));
    const standaloneTab = await readEditorSyntax(page, "websql");
    assertDomSyntax(standaloneTab, "standalone-new-tab", [
      ["SELECT", "keyword"],
      ["COUNT", "predefined"],
      ["FROM", "keyword"],
      ["LIKE", "keyword"],
      [`-- ${marker}_tab`, "comment"],
    ]);
    summary.editors.standaloneNewTab = standaloneTab;
    summary.checks.standaloneNewTabSyntax = true;
    await page.screenshot({ path: screenshots.standalone, fullPage: true });

    await openDdlDialog(page, tableName);
    const ddlInfo = await readEditorSyntax(page, "ddl");
    assertCheck(/CREATE\s+TABLE/i.test(ddlInfo.value), "ddl editor missing CREATE TABLE");
    assertCheck(ddlInfo.value.includes(tableName), "ddl editor missing syntax test table");
    assertDomSyntax(ddlInfo, "standalone-ddl", [
      ["CREATE", "keyword"],
      ["TABLE", "keyword"],
      ["INTEGER", "keyword"],
      ["PRIMARY", "keyword"],
      ["KEY", "keyword"],
      ["TEXT", "keyword"],
      ["DEFAULT", "keyword"],
    ], { minColors: 3 });
    summary.editors.standaloneDdl = ddlInfo;
    summary.checks.standaloneDdlSyntax = true;
    await page.screenshot({ path: screenshots.ddl, fullPage: true });
    await page.keyboard.press("Escape");
    await sleep(500);

    await gotoPage(page, WEBSHELL_URL);
    await clearWebSQLLocalState(page);
    await gotoPage(page, WEBSHELL_URL);
    await openWebSQLPanel(page);
    await waitForWebsqlEditor(page);
    await setEditorValue(page, "websql", DOM_SQL(tableName, `${marker}_webshell`));
    const webshellInfo = await readEditorSyntax(page, "websql");
    assertDomSyntax(webshellInfo, "webshell-embedded", [
      ["SELECT", "keyword"],
      ["COUNT", "predefined"],
      ["FROM", "keyword"],
      ["WHERE", "keyword"],
      [`-- ${marker}_webshell`, "comment"],
    ]);
    summary.editors.webshellEmbedded = webshellInfo;
    summary.checks.webshellEmbeddedSyntax = true;
    await page.screenshot({ path: screenshots.webshell, fullPage: true });

    const blockingConsoleErrors = summary.consoleErrors.filter((item) => !/ResizeObserver|favicon/i.test(item));
    const blockingRequestFailures = summary.failedRequests.filter((item) => !/favicon/i.test(item));
    assertCheck(blockingConsoleErrors.length === 0, `console errors: ${blockingConsoleErrors.join("\n")}`);
    assertCheck(summary.pageErrors.length === 0, `page errors: ${summary.pageErrors.join("\n")}`);
    assertCheck(blockingRequestFailures.length === 0, `requestfailed: ${blockingRequestFailures.join("\n")}`);
    assertCheck(Object.values(summary.checks).every(Boolean), "not all SQL syntax checks passed");
    summary.pass = true;
  } catch (error) {
    summary.error = error?.stack || String(error);
    process.exitCode = 1;
  } finally {
    await browser.close().catch(() => undefined);
    websql({
      operation: "execute",
      sql: `DROP TABLE IF EXISTS ${tableName};`,
    });
    fs.writeFileSync(reportPath, JSON.stringify(summary, null, 2));
    console.log(`report: ${reportPath}`);
    console.log(JSON.stringify({ pass: summary.pass, checks: summary.checks }, null, 2));
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
