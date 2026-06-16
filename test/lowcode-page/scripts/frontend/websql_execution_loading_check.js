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

const BASE_URL = process.env.WEBSQL_BASE_URL || "http://192.168.232.130:8015";
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

function parsePostData(request) {
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

async function gotoPage(page) {
  await page.goto(TARGET_URL, { waitUntil: "domcontentloaded", timeout: 45000 });
  await sleep(2500);
}

async function setWebSQLEditorValue(page, value) {
  const result = await page.evaluate((nextValue) => {
    const editor = (window?.monaco?.editor?.getEditors?.() || []).find((item) => {
      const node = item?.getContainerDomNode?.();
      return node && node.closest(".websql-lowcode");
    });
    const model = editor?.getModel?.();
    if (!editor || !model) {
      return { ok: false, reason: "websql monaco editor not found" };
    }
    editor.setValue(String(nextValue || ""));
    const line = model.getLineCount?.() || 1;
    const column = model.getLineMaxColumn?.(line) || 1;
    if (window.monaco?.Selection) {
      editor.setSelection(new window.monaco.Selection(line, column, line, column));
    }
    editor.focus();
    return { ok: true };
  }, value);
  assertCheck(result.ok, result.reason || "failed to set WebSQL editor value");
  await sleep(600);
}

async function readLoadingState(page) {
  return page.evaluate(() => {
    const visible = (selector) => {
      const el = document.querySelector(selector);
      if (!el) return false;
      const rect = el.getBoundingClientRect();
      const style = window.getComputedStyle(el);
      return rect.width > 0 && rect.height > 0 && style.display !== "none";
    };
    const button = document.querySelector(".websql-execute-btn");
    return {
      buttonLoading: !!button?.classList?.contains("ant-btn-loading"),
      buttonDisabled: !!button?.hasAttribute?.("disabled"),
      maskVisible: visible(".websql-executing-mask"),
      maskText: String(document.querySelector(".websql-executing-mask")?.textContent || "").trim(),
      statusText: String(document.querySelector(".websql-status")?.textContent || "").trim(),
    };
  });
}

async function main() {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const reportPath = path.join(OUT_DIR, "websql-execution-loading-check.json");
  const screenshotPath = path.join(OUT_DIR, "websql-execution-loading-check.png");
  const loadingScreenshotPath = path.join(
    OUT_DIR,
    "websql-execution-loading-check-during.png"
  );
  const summary = {
    url: TARGET_URL,
    delayedExecuteRequests: 0,
    during: null,
    after: null,
    response: null,
    consoleErrors: [],
    pageErrors: [],
    requestFailed: [],
    loadingScreenshot: loadingScreenshotPath,
    screenshot: screenshotPath,
    report: reportPath,
  };

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1600, height: 960 } });

  page.on("console", (msg) => {
    if (msg.type() === "error") summary.consoleErrors.push(msg.text());
  });
  page.on("pageerror", (error) => summary.pageErrors.push(String(error)));
  page.on("requestfailed", (request) => {
    summary.requestFailed.push({
      url: request.url(),
      method: request.method(),
      errorText: request.failure()?.errorText || "",
    });
  });

  await page.route("**/template_data/data?**", async (route) => {
    const req = route.request();
    const payload = parsePostData(req);
    if (
      req.method() === "POST" &&
      req.url().includes("service=webshell.websql_execute") &&
      payload.operation === "execute" &&
      String(payload.sql || "").includes("sys_projects")
    ) {
      summary.delayedExecuteRequests += 1;
      await sleep(1800);
    }
    await route.continue();
  });

  try {
    await gotoPage(page);
    await clearWebSQLLocalState(page);
    await gotoPage(page);
    await page.locator(".websql-lowcode .monaco-editor").first().waitFor({
      state: "visible",
      timeout: 45000,
    });
    await setWebSQLEditorValue(
      page,
      "SELECT a.project_code, a.project_name FROM sys_projects a WHERE IFNULL(a.flag_del, '0') = '0';"
    );

    const responsePromise = page.waitForResponse((resp) => {
      if (!resp.url().includes("service=webshell.websql_execute")) return false;
      const payload = parsePostData(resp.request());
      return payload.operation === "execute" && String(payload.sql || "").includes("sys_projects");
    }, { timeout: 30000 });

    await page.locator(".websql-execute-btn").first().click();
    await page.waitForFunction(() => {
      const button = document.querySelector(".websql-execute-btn");
      const mask = document.querySelector(".websql-executing-mask");
      const maskRect = mask?.getBoundingClientRect?.();
      const maskStyle = mask ? window.getComputedStyle(mask) : null;
      return (
        !!button?.classList?.contains("ant-btn-loading") &&
        !!maskRect &&
        maskRect.width > 0 &&
        maskRect.height > 0 &&
        maskStyle?.display !== "none"
      );
    }, { timeout: 5000 });
    summary.during = await readLoadingState(page);
    await page.screenshot({ path: loadingScreenshotPath, fullPage: true });

    const response = await responsePromise;
    summary.response = await response.json();
    await page.waitForFunction(() => {
      const button = document.querySelector(".websql-execute-btn");
      const mask = document.querySelector(".websql-executing-mask");
      const maskRect = mask?.getBoundingClientRect?.();
      return !button?.classList?.contains("ant-btn-loading") && (!maskRect || maskRect.width === 0 || maskRect.height === 0);
    }, { timeout: 5000 });
    summary.after = await readLoadingState(page);
    await page.screenshot({ path: screenshotPath, fullPage: true });

    assertCheck(summary.delayedExecuteRequests === 1, "execute request was not delayed");
    assertCheck(summary.during.buttonLoading, `button did not enter loading: ${JSON.stringify(summary.during)}`);
    assertCheck(summary.during.buttonDisabled, `button was not disabled: ${JSON.stringify(summary.during)}`);
    assertCheck(summary.during.maskVisible, `loading mask missing: ${JSON.stringify(summary.during)}`);
    assertCheck(summary.during.maskText.includes("SQL 执行中"), `loading mask text missing: ${JSON.stringify(summary.during)}`);
    assertCheck(!summary.after.buttonLoading, `button loading did not clear: ${JSON.stringify(summary.after)}`);
    assertCheck(!summary.after.maskVisible, `loading mask did not clear: ${JSON.stringify(summary.after)}`);
    assertCheck(summary.response?.success === true, `execute response failed: ${JSON.stringify(summary.response)}`);
    assertCheck(summary.consoleErrors.length === 0, `console errors: ${summary.consoleErrors.join(" | ")}`);
    assertCheck(summary.pageErrors.length === 0, `page errors: ${summary.pageErrors.join(" | ")}`);
    assertCheck(summary.requestFailed.length === 0, `request failed: ${JSON.stringify(summary.requestFailed)}`);
  } finally {
    fs.writeFileSync(reportPath, JSON.stringify(summary, null, 2));
    await page.close().catch(() => undefined);
    await browser.close().catch(() => undefined);
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
