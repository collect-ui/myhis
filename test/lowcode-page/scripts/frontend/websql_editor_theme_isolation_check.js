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

const BASE_URL = process.env.THEME_BASE_URL || "http://192.168.232.130:8015";
const API_URL = process.env.THEME_API_URL || "http://127.0.0.1:8015/template_data/data";
const WEBSQL_URL =
  process.env.THEME_WEBSQL_URL ||
  `${BASE_URL}/collect-ui#/collect-ui/framework/websql-pool`;
const EDITOR_URL =
  process.env.THEME_EDITOR_URL ||
  `${BASE_URL}/collect-ui#/collect-ui/framework/webshell-editor-pool`;
const OUT_DIR =
  process.env.THEME_OUTPUT_DIR ||
  "/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation";
const PROJECT_CODE = process.env.THEME_PROJECT_CODE || "test";
const PROJECT_DIR = process.env.THEME_PROJECT_DIR || "/data/project/test";

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
  let parsed = {};
  try {
    parsed = JSON.parse(String(res.stdout || "{}"));
  } catch (error) {
    throw new Error(`parse response failed (${service}): ${error.message}`);
  }
  if (!parsed || String(parsed.code || "") !== "0" || parsed.success === false) {
    throw new Error(`${service} failed: ${parsed?.msg || "unknown error"}`);
  }
  return parsed;
}

function runCurlAllowFail(service, data) {
  try {
    return { ok: true, value: runCurl(service, data), error: "" };
  } catch (error) {
    return { ok: false, value: null, error: String(error?.message || error) };
  }
}

function escapeRegExp(value) {
  return String(value || "").replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function rgbBrightness(rgb) {
  const matched = String(rgb || "").match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
  if (!matched) {
    return -1;
  }
  return (Number(matched[1]) + Number(matched[2]) + Number(matched[3])) / 3;
}

async function waitForWebsqlEditor(page) {
  await page.locator(".websql-lowcode").first().waitFor({ state: "visible", timeout: 45000 });
  await page.waitForFunction(() => {
    const editors = window.monaco?.editor?.getEditors?.() || [];
    return editors.some((editor) => editor?.getContainerDomNode?.()?.closest(".websql-lowcode"));
  }, null, { timeout: 45000 });
}

async function waitForWorkspaceEditor(page) {
  await page.locator('[data-testid="workspace-route-editor-host"]').first().waitFor({ state: "visible", timeout: 45000 });
  await page.waitForFunction(() => {
    const host = document.querySelector('[data-testid="workspace-route-editor-host"]');
    const editors = window.monaco?.editor?.getEditors?.() || [];
    return !!host && editors.some((editor) => {
      const node = editor?.getContainerDomNode?.();
      if (!node || !host.contains(node)) return false;
      const rect = node.getBoundingClientRect();
      return rect.width > 100 && rect.height > 80;
    });
  }, null, { timeout: 45000 });
}

async function focusMonacoIn(page, selector) {
  const locator = page.locator(selector).first();
  await locator.waitFor({ state: "visible", timeout: 20000 });
  const box = await locator.boundingBox();
  assertCheck(!!box, `monaco box not found: ${selector}`);
  await page.mouse.click(box.x + Math.min(120, box.width / 2), box.y + Math.min(70, box.height / 2));
  await sleep(300);
}

async function replaceFocusedText(page, value) {
  await page.keyboard.press("Control+A");
  await page.keyboard.type(value);
  await sleep(500);
}

async function readThemeInfo(page, scope) {
  return page.evaluate((targetScope) => {
    const visible = (node) => {
      if (!node) return false;
      const rect = node.getBoundingClientRect();
      return rect.width > 100 && rect.height > 80;
    };
    const editors = window.monaco?.editor?.getEditors?.() || [];
    let editor = null;
    if (targetScope === "websql") {
      editor = editors.find((item) => {
        const node = item?.getContainerDomNode?.();
        return node?.closest(".websql-lowcode") && !node.closest(".websql-ddl-editor") && visible(node);
      });
    } else {
      const host = document.querySelector('[data-testid="workspace-route-editor-host"]');
      editor = editors.find((item) => {
        const node = item?.getContainerDomNode?.();
        return host?.contains(node) && visible(node);
      });
    }
    const node = editor?.getContainerDomNode?.();
    const body =
      targetScope === "websql"
        ? document.querySelector(".websql-lowcode .websql-editor-body")
        : document.querySelector('[data-testid="workspace-route-editor-host"]');
    const backgroundNode = node?.querySelector(".monaco-editor-background") || node;
    const line = node?.querySelector(".view-line span");
    const tokenSamples = Array.from(node?.querySelectorAll(".view-lines .view-line span") || [])
      .map((span) => ({
        text: String(span.textContent || ""),
        className: String(span.className || ""),
        color: window.getComputedStyle(span).color || "",
      }))
      .filter((item) => item.text.trim());
    const tokenColors = Array.from(new Set(tokenSamples.map((item) => item.color).filter(Boolean)));
    const margin = node?.querySelector(".margin");
    const bodyStyle = body ? window.getComputedStyle(body) : null;
    const editorStyle = backgroundNode ? window.getComputedStyle(backgroundNode) : null;
    const lineStyle = line ? window.getComputedStyle(line) : null;
    const marginStyle = margin ? window.getComputedStyle(margin) : null;
    const bodyBg = bodyStyle?.backgroundColor || "";
    const editorBg = editorStyle?.backgroundColor || "";
    const lineColor = lineStyle?.color || "";
    const marginBg = marginStyle?.backgroundColor || "";
    const bodyBrightness = window.__themeBrightness
      ? window.__themeBrightness(bodyBg)
      : -1;
    const editorBrightness = window.__themeBrightness
      ? window.__themeBrightness(editorBg)
      : -1;
    return {
      scope: targetScope,
      found: !!editor,
      bodyBg,
      editorBg,
      marginBg,
      lineColor,
      tokenSamples: tokenSamples.slice(0, 16),
      tokenColors,
      distinctTokenColorCount: tokenColors.length,
      bodyBrightness,
      editorBrightness,
      monacoClass: String(node?.className || ""),
      localTheme: String(node?.getAttribute("data-sport-ui-local-theme") || ""),
      value: String(editor?.getValue?.() || ""),
    };
  }, scope);
}

async function openProjectFile(page, fileName) {
  const projectButton = page.getByRole("button", { name: new RegExp(`^${escapeRegExp(PROJECT_CODE)}$`) }).first();
  await projectButton.waitFor({ state: "visible", timeout: 30000 });
  await projectButton.click();
  await page.locator(".workspace-source-tree").first().waitFor({ state: "visible", timeout: 30000 });
  const fileTitle = page
    .locator(".workspace-source-tree .ant-tree-title")
    .filter({ hasText: new RegExp(`^\\s*${escapeRegExp(fileName)}\\s*$`) })
    .first();
  await fileTitle.waitFor({ state: "visible", timeout: 30000 });
  await fileTitle.click();
  await sleep(1200);
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const stamp = Date.now();
  const fileName = `theme_isolation_${stamp}.txt`;
  const filePath = path.posix.join(PROJECT_DIR, fileName);
  const websqlMarker = `theme_websql_${stamp}`;
  const editorMarker = `theme_editor_${stamp}`;
  const screenshots = {
    websqlInitial: path.join(OUT_DIR, "websql-editor-theme-isolation-websql-initial.png"),
    editorDark: path.join(OUT_DIR, "websql-editor-theme-isolation-editor-dark.png"),
    websqlAfterEditor: path.join(OUT_DIR, "websql-editor-theme-isolation-websql-after-editor.png"),
    editorAfterWebsql: path.join(OUT_DIR, "websql-editor-theme-isolation-editor-after-websql.png"),
  };
  const reportPath = path.join(OUT_DIR, "websql-editor-theme-isolation-check.json");

  runCurlAllowFail("webshell.workspace_file_delete_with_sync", {
    project_code: PROJECT_CODE,
    path: filePath,
  });
  runCurl("webshell.workspace_file_add_with_sync", {
    project_code: PROJECT_CODE,
    name: fileName,
    path: filePath,
    is_dir: "0",
    parent_id: "",
  });

  const result = {
    websqlUrl: WEBSQL_URL,
    editorUrl: EDITOR_URL,
    projectCode: PROJECT_CODE,
    fileName,
    filePath,
    checks: {},
    theme: {},
    screenshots,
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    saveResponse: null,
    readback: "",
    pass: false,
  };

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: { width: 1680, height: 980 } });
  await context.addInitScript(() => {
    window.__themeBrightness = (rgb) => {
      const matched = String(rgb || "").match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
      if (!matched) return -1;
      return (Number(matched[1]) + Number(matched[2]) + Number(matched[3])) / 3;
    };
    for (const key of Object.keys(window.localStorage || {})) {
      if (key.startsWith("workspace-websql-recent") || key.startsWith("websql-lowcode")) {
        window.localStorage.removeItem(key);
      }
    }
  });
  const page = await context.newPage();
  page.on("console", (msg) => {
    if (msg.type() === "error") {
      result.consoleErrors.push(msg.text());
    }
  });
  page.on("pageerror", (error) => result.pageErrors.push(String(error)));
  page.on("requestfailed", (req) => {
    result.failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || "failed"}`);
  });

  try {
    await page.goto(WEBSQL_URL, { waitUntil: "domcontentloaded", timeout: 60000 });
    await waitForWebsqlEditor(page);
    result.theme.websqlInitial = await readThemeInfo(page, "websql");
    result.checks.websqlInitialLight =
      result.theme.websqlInitial.found &&
      result.theme.websqlInitial.localTheme === "light" &&
      rgbBrightness(result.theme.websqlInitial.editorBg) >= 245;
    await page.screenshot({ path: screenshots.websqlInitial, fullPage: true });

    await page.locator(".websql-add-sql-btn").first().click();
    await sleep(700);
    await focusMonacoIn(page, ".websql-lowcode .websql-editor-body .monaco-editor");
    await replaceFocusedText(
      page,
      `SELECT COUNT(*) AS total FROM sqlite_master WHERE type = 'table' AND name LIKE '${websqlMarker}%'; -- ${websqlMarker}`
    );
    await page.waitForFunction((marker) => {
      const editors = window.monaco?.editor?.getEditors?.() || [];
      return editors.some((editor) => {
        const node = editor?.getContainerDomNode?.();
        return node?.closest(".websql-lowcode") && String(editor.getValue?.() || "").includes(marker);
      });
    }, websqlMarker, { timeout: 10000 });
    result.theme.websqlAfterTyping = await readThemeInfo(page, "websql");
    result.checks.websqlTyped = result.theme.websqlAfterTyping.value.includes(websqlMarker);
    result.checks.websqlSyntaxHighlighted =
      result.theme.websqlAfterTyping.distinctTokenColorCount >= 2 &&
      (result.theme.websqlAfterTyping.tokenSamples || []).some((item) =>
        String(item.className || "").includes("mtk")
      );

    await page.goto(EDITOR_URL, { waitUntil: "domcontentloaded", timeout: 60000 });
    await sleep(1800);
    await openProjectFile(page, fileName);
    await waitForWorkspaceEditor(page);
    result.theme.editorBeforeTyping = await readThemeInfo(page, "editor");
    result.checks.editorOpened = result.theme.editorBeforeTyping.found;
    result.checks.editorDark =
      result.theme.editorBeforeTyping.found &&
      rgbBrightness(result.theme.editorBeforeTyping.editorBg) <= 80;
    await page.screenshot({ path: screenshots.editorDark, fullPage: true });

    await focusMonacoIn(page, '[data-testid="workspace-route-editor-host"] .monaco-editor');
    await replaceFocusedText(page, `editor text ${editorMarker}\n`);
    const savePromise = page.waitForResponse(
      (resp) =>
        resp.url().includes("service=webshell.workspace_file_save") &&
        resp.request().method() === "POST",
      { timeout: 20000 }
    );
    await page.keyboard.press("Control+S");
    const saveResp = await savePromise;
    result.saveResponse = await saveResp.json().catch(() => ({}));
    const readback = runCurl("webshell.workspace_file_content", {
      project_code: PROJECT_CODE,
      path: filePath,
      max_bytes: 4096,
    });
    result.readback = String(readback?.data?.content_text || "");
    result.checks.editorSavedReadback = result.readback.includes(editorMarker);

    await page.goto(WEBSQL_URL, { waitUntil: "domcontentloaded", timeout: 60000 });
    await waitForWebsqlEditor(page);
    result.theme.websqlAfterEditor = await readThemeInfo(page, "websql");
    result.checks.websqlAfterEditorLight =
      result.theme.websqlAfterEditor.found &&
      result.theme.websqlAfterEditor.localTheme === "light" &&
      rgbBrightness(result.theme.websqlAfterEditor.editorBg) >= 245;
    result.checks.websqlTextUnaffected = result.theme.websqlAfterEditor.value.includes(websqlMarker);
    await page.screenshot({ path: screenshots.websqlAfterEditor, fullPage: true });

    await page.goto(EDITOR_URL, { waitUntil: "domcontentloaded", timeout: 60000 });
    await sleep(1000);
    await waitForWorkspaceEditor(page);
    result.theme.editorAfterWebsql = await readThemeInfo(page, "editor");
    result.checks.editorAfterWebsqlDark =
      result.theme.editorAfterWebsql.found &&
      rgbBrightness(result.theme.editorAfterWebsql.editorBg) <= 80;
    result.checks.editorTextUnaffected = result.theme.editorAfterWebsql.value.includes(editorMarker);
    await page.screenshot({ path: screenshots.editorAfterWebsql, fullPage: true });

    result.checks.noConsoleErrors = result.consoleErrors.length === 0;
    result.checks.noPageErrors = result.pageErrors.length === 0;
    result.checks.noFailedRequests = result.failedRequests.length === 0;
    assertCheck(Object.values(result.checks).every(Boolean), `checks failed: ${JSON.stringify(result.checks, null, 2)}`);
    result.pass = true;
  } catch (error) {
    result.error = String(error?.stack || error?.message || error);
    await page.screenshot({
      path: path.join(OUT_DIR, "websql-editor-theme-isolation-fail.png"),
      fullPage: true,
    }).catch(() => undefined);
    process.exitCode = 1;
  } finally {
    result.cleanup = runCurlAllowFail("webshell.workspace_file_delete_with_sync", {
      project_code: PROJECT_CODE,
      path: filePath,
    });
    fs.writeFileSync(reportPath, JSON.stringify(result, null, 2));
    await browser.close();
    console.log(`report: ${reportPath}`);
    console.log(JSON.stringify({ pass: result.pass, checks: result.checks }, null, 2));
  }
})();
