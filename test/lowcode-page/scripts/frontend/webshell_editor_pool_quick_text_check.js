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

const PAGE_URL =
  process.env.WEBSHELL_EDITOR_POOL_PAGE_URL ||
  "http://127.0.0.1:8015/collect-ui#/collect-ui/framework/webshell-editor-pool";
const API_URL =
  process.env.WEBSHELL_EDITOR_POOL_API_URL ||
  "http://127.0.0.1:8015/template_data/data";
const OUT_DIR =
  process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR ||
  "/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation";
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || "test";
const FILE_PATH = process.env.WEBSHELL_EDITOR_POOL_QUICK_TEXT_FILE || "/data/project/test/test/test.md";
const FILE_NAME = path.posix.basename(FILE_PATH);
const TARGET_TITLE = "无头浏览器回归要求";

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function runCurl(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  const res = spawnSync("curl", [
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
  ], { encoding: "utf8" });
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

function escapeRegExp(input) {
  return String(input || "").replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

async function selectProject(page) {
  const btn = page.getByRole("button", { name: new RegExp(`^${escapeRegExp(PROJECT_CODE)}$`) }).first();
  await btn.waitFor({ state: "visible", timeout: 22000 });
  await btn.click();
  await sleep(1200);
}

async function openFileFromTree(page) {
  const input = page.locator('input[placeholder="回车搜索(至少2个字符)"]:visible').first();
  await input.waitFor({ state: "visible", timeout: 22000 });
  await input.fill("");
  await input.fill(FILE_NAME);
  await input.press("Enter");
  await sleep(1200);
  const title = page.locator(".workspace-source-tree .ant-tree-title:visible").filter({ hasText: new RegExp(`^${escapeRegExp(FILE_NAME)}$`) }).first();
  await title.waitFor({ state: "visible", timeout: 22000 });
  await title.click();
  await sleep(1800);
}

async function waitEditor(page) {
  const ready = await page.waitForFunction(() => {
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    return editors.some((editor) => {
      try {
        const node = editor?.getContainerDomNode?.();
        const slot = node?.closest?.("[data-slot-id]");
        const style = slot ? window.getComputedStyle(slot) : null;
        return !!editor?.getModel?.() && !!style && style.visibility !== "hidden" && style.display !== "none";
      } catch (_error) {
        return false;
      }
    });
  }, null, { timeout: 22000 });
  return ready;
}

async function prepareEditor(page) {
  await waitEditor(page);
  return page.evaluate(() => {
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    for (const editor of editors) {
      const node = editor?.getContainerDomNode?.();
      const slot = node?.closest?.("[data-slot-id]");
      if (!slot) continue;
      const style = window.getComputedStyle(slot);
      if (style.visibility === "hidden" || style.display === "none") continue;
      const model = editor?.getModel?.();
      if (!model) continue;
      editor.setValue("");
      editor.setPosition({ lineNumber: 1, column: 1 });
      editor.focus();
      return { ok: true, uri: String(model.uri || "") };
    }
    return { ok: false, reason: "visible editor not found" };
  });
}

async function getEditorValue(page) {
  return page.evaluate(() => {
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    for (const editor of editors) {
      const node = editor?.getContainerDomNode?.();
      const slot = node?.closest?.("[data-slot-id]");
      if (!slot) continue;
      const style = window.getComputedStyle(slot);
      if (style.visibility === "hidden" || style.display === "none") continue;
      if (!editor?.getModel?.()) continue;
      return String(editor.getValue?.() || "");
    }
    return "";
  });
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const summary = {
    pageUrl: PAGE_URL,
    projectCode: PROJECT_CODE,
    filePath: FILE_PATH,
    targetTitle: TARGET_TITLE,
    checks: {
      apiSeedFound: false,
      apiSeedHasLineBreak: false,
      bulkReadmeSeedsPresent: false,
      categorySeedCounts: false,
      pageOpened: false,
      projectSelected: false,
      fileOpened: false,
      dropdownOpened: false,
      quickTextBeforeLocationNav: false,
      virtualScrollActive: false,
      reopenVirtualListRendersTop: false,
      sequenceVisible: false,
      tooltipPreservesLineBreak: false,
      roleCheckboxVisible: false,
      effectDefaultMatched: false,
      insertedAtCursor: false,
      insertedLineBreak: false,
      savedToFile: false,
      savedLineBreak: false,
      useCountIncreased: false,
    },
    totalQuickTextCount: 0,
    renderedQuickTextItemCount: 0,
    categoryCounts: {
      readme: 0,
      vibeCoding: 0,
      design: 0,
      test: 0,
    },
    beforeUseCount: 0,
    afterUseCount: 0,
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    screenshot: "",
    pass: false,
    error: "",
  };

  const seedRes = runCurl("webshell.quick_text_query", {
    effect_ext: "md",
    pagination: false,
  });
  const seedRows = Array.isArray(seedRes.data) ? seedRes.data : [];
  summary.totalQuickTextCount = seedRows.length;
  summary.categoryCounts = {
    readme: seedRows.filter((item) => String(item.title || "").startsWith("README ")).length,
    vibeCoding: seedRows.filter((item) => String(item.title || "").startsWith("Vibe Coding ")).length,
    design: seedRows.filter((item) => String(item.title || "").startsWith("设计 ")).length,
    test: seedRows.filter((item) => String(item.title || "").startsWith("测试 ")).length,
  };
  summary.checks.bulkReadmeSeedsPresent = summary.totalQuickTextCount >= 48;
  summary.checks.categorySeedCounts = Object.values(summary.categoryCounts).every((count) => count >= 10);
  const seed = seedRows.find((item) => String(item.title || "") === TARGET_TITLE);
  if (!seed) {
    throw new Error(`quick text seed not found: ${TARGET_TITLE}`);
  }
  summary.checks.apiSeedFound = true;
  summary.checks.apiSeedHasLineBreak = String(seed.content || "").includes("\n");
  summary.beforeUseCount = Number(seed.use_count || 0);

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1680, height: 980 } });
  page.on("console", (msg) => {
    if (msg.type() === "error") summary.consoleErrors.push(msg.text());
  });
  page.on("pageerror", (error) => summary.pageErrors.push(String(error)));
  page.on("requestfailed", (req) => {
    summary.failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || "failed"}`);
  });

  try {
    await page.goto(PAGE_URL, { waitUntil: "networkidle", timeout: 60000 });
    summary.checks.pageOpened = true;
    await sleep(1800);
    await selectProject(page);
    summary.checks.projectSelected = true;
    await openFileFromTree(page);
    summary.checks.fileOpened = true;
    const prepared = await prepareEditor(page);
    if (!prepared?.ok) {
      throw new Error(prepared?.reason || "editor prepare failed");
    }

    const trigger = page.locator('[data-testid="workspace-quick-text-trigger"]').first();
    await trigger.waitFor({ state: "visible", timeout: 12000 });
    const quickTriggerBox = await trigger.boundingBox();
    const locationNavBox = await page.locator("span", { hasText: /^位置导航$/ }).first().boundingBox();
    summary.checks.quickTextBeforeLocationNav = !!quickTriggerBox && !!locationNavBox && quickTriggerBox.x < locationNavBox.x;
    await trigger.click();
    const dropdown = page.locator('[data-testid="workspace-quick-text-dropdown"]').first();
    await dropdown.waitFor({ state: "visible", timeout: 12000 });
    summary.checks.dropdownOpened = true;
    summary.checks.roleCheckboxVisible = await dropdown.locator('label', { hasText: "测试" }).locator('input[type="checkbox"]').first().isVisible().catch(() => false);
    const effectValue = await dropdown.locator("select").first().inputValue();
    summary.checks.effectDefaultMatched = effectValue === "md";
    summary.renderedQuickTextItemCount = await dropdown.locator('[data-testid="workspace-quick-text-item"]').count();
    const firstSequenceText = await dropdown.locator('[data-testid="workspace-quick-text-index"]').first().textContent().catch(() => "");
    summary.checks.sequenceVisible = String(firstSequenceText || "").trim() === "#1";
    summary.checks.virtualScrollActive = summary.totalQuickTextCount >= 40 &&
      summary.renderedQuickTextItemCount > 0 &&
      summary.renderedQuickTextItemCount < summary.totalQuickTextCount;

    const virtualList = dropdown.locator('[data-testid="workspace-quick-text-virtual-list"]').first();
    await virtualList.evaluate((node) => {
      node.scrollTop = Math.max(0, node.scrollHeight - node.clientHeight);
      node.dispatchEvent(new Event("scroll", { bubbles: true }));
    });
    await sleep(250);
    await trigger.click();
    await dropdown.waitFor({ state: "hidden", timeout: 12000 });
    await trigger.click();
    await dropdown.waitFor({ state: "visible", timeout: 12000 });
    await sleep(250);
    const reopenedVirtualList = dropdown.locator('[data-testid="workspace-quick-text-virtual-list"]').first();
    const reopenedScrollTop = await reopenedVirtualList.evaluate((node) => node.scrollTop).catch(() => -1);
    const reopenedFirstSequenceText = await dropdown.locator('[data-testid="workspace-quick-text-index"]').first().textContent().catch(() => "");
    const reopenedRenderedCount = await dropdown.locator('[data-testid="workspace-quick-text-item"]').count();
    summary.checks.reopenVirtualListRendersTop = reopenedScrollTop === 0 &&
      String(reopenedFirstSequenceText || "").trim() === "#1" &&
      reopenedRenderedCount > 0;

    const target = dropdown.locator('[data-testid="workspace-quick-text-item"]', { hasText: TARGET_TITLE }).first();
    await target.waitFor({ state: "visible", timeout: 12000 });
    await target.locator('[data-testid="workspace-quick-text-preview"]').first().hover();
    const tooltipContent = page.locator('[data-testid="workspace-quick-text-full-content"]').last();
    await tooltipContent.waitFor({ state: "visible", timeout: 12000 });
    const tooltipText = String(await tooltipContent.textContent() || "");
    const tooltipWhiteSpace = await tooltipContent.evaluate((node) => window.getComputedStyle(node).whiteSpace).catch(() => "");
    summary.checks.tooltipPreservesLineBreak = tooltipText.includes("测试要求：\n\n- 使用无头浏览器打开目标页面") &&
      String(tooltipWhiteSpace || "").includes("pre-wrap");
    await target.locator("button").first().click();
    await dropdown.waitFor({ state: "hidden", timeout: 12000 });
    await sleep(500);

    const editorValue = await getEditorValue(page);
    summary.checks.insertedAtCursor = editorValue.includes("测试要求：") && editorValue.includes("- 使用无头浏览器打开目标页面");
    summary.checks.insertedLineBreak = editorValue.includes("测试要求：\n\n- 使用无头浏览器打开目标页面");

    await page.getByRole("button", { name: /保存/ }).first().click();
    await sleep(1200);
    const contentRes = runCurl("webshell.workspace_file_content", {
      project_code: PROJECT_CODE,
      path: FILE_PATH,
    });
    const savedText = String(contentRes?.data?.content_text || "");
    summary.checks.savedToFile = savedText.includes("测试要求：") && savedText.includes("- 使用无头浏览器打开目标页面");
    summary.checks.savedLineBreak = savedText.includes("测试要求：\n\n- 使用无头浏览器打开目标页面");

    const afterRes = runCurl("webshell.quick_text_query", {
      effect_ext: "md",
      pagination: false,
    });
    const after = (Array.isArray(afterRes.data) ? afterRes.data : []).find((item) => String(item.quick_text_id || "") === String(seed.quick_text_id || ""));
    summary.afterUseCount = Number(after?.use_count || 0);
    summary.checks.useCountIncreased = summary.afterUseCount > summary.beforeUseCount;

    summary.screenshot = path.join(OUT_DIR, "webshell-editor-pool-quick-text-check.png");
    await page.screenshot({ path: summary.screenshot, fullPage: true });
    summary.pass = Object.values(summary.checks).every(Boolean) &&
      summary.consoleErrors.length === 0 &&
      summary.pageErrors.length === 0 &&
      summary.failedRequests.length === 0;
  } catch (error) {
    summary.error = String(error && error.message ? error.message : error);
    summary.screenshot = path.join(OUT_DIR, "webshell-editor-pool-quick-text-check.fail.png");
    await page.screenshot({ path: summary.screenshot, fullPage: true }).catch(() => undefined);
  } finally {
    await browser.close();
    const out = path.join(OUT_DIR, "webshell-editor-pool-quick-text-check.json");
    fs.writeFileSync(out, JSON.stringify(summary, null, 2));
    console.log(JSON.stringify(summary, null, 2));
    if (!summary.pass) {
      process.exit(1);
    }
  }
})();
