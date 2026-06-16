#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
const { chromium } = require('playwright');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://127.0.0.1:8015/template_data/data';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || 'test';
const PROJECT_DIR = process.env.WEBSHELL_EDITOR_POOL_PROJECT_DIR || '/data/project/test';

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function runCurl(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  const res = spawnSync('curl', [
    '--noproxy', '*',
    '-sS',
    `${API_URL}?service=${service}`,
    '-H',
    'Content-Type: application/json',
    '--data',
    payload,
  ], { encoding: 'utf8' });
  if (res.status !== 0) {
    throw new Error(res.stderr || `curl failed: ${service}`);
  }
  let parsed = {};
  try {
    parsed = JSON.parse(String(res.stdout || '{}'));
  } catch (error) {
    throw new Error(`parse response failed (${service}): ${error.message}`);
  }
  if (!parsed || String(parsed.code || '') !== '0' || parsed.success === false) {
    throw new Error(`${service} failed: ${parsed?.msg || 'unknown error'}`);
  }
  return parsed;
}

function runCurlAllowFail(service, data) {
  try {
    return { ok: true, value: runCurl(service, data), error: '' };
  } catch (error) {
    return { ok: false, value: null, error: String(error && error.message ? error.message : error) };
  }
}

function escapeRegExp(text) {
  return String(text || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function toTreeTitle(name) {
  return new RegExp(`^${escapeRegExp(name)}$`);
}

function toLooseTitle(name) {
  return new RegExp(`^\\s*${escapeRegExp(name)}\\s*$`);
}

async function ensureProjectSelected(page) {
  const projectBtn = page.getByRole('button', { name: PROJECT_CODE }).first();
  await projectBtn.waitFor({ state: 'visible', timeout: 20000 });
  await projectBtn.click();
  await sleep(1200);
}

async function expandTreeNode(page, nodeName) {
  const title = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(nodeName) }).first();
  await title.waitFor({ state: 'visible', timeout: 20000 });
  const switcher = title.locator('xpath=ancestor::div[contains(@class,"ant-tree-treenode")][1]').locator('.ant-tree-switcher').first();
  if (await switcher.count() <= 0) {
    return;
  }
  const cls = String((await switcher.getAttribute('class')) || '');
  if (cls.includes('ant-tree-switcher_close')) {
    await switcher.click();
    await sleep(700);
  }
}

async function rightClickMenuAction(page, nodeName, menuName) {
  const node = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(nodeName) }).first();
  await node.waitFor({ state: 'visible', timeout: 20000 });
  await node.click({ button: 'right' });
  const menu = page.locator('div[role="menu"].contexify:visible, div[role="menu"]:visible').last();
  await menu.waitFor({ state: 'visible', timeout: 8000 });
  const menuItem = menu.locator('[role="menuitem"]', { hasText: toLooseTitle(menuName) }).first();
  await menuItem.waitFor({ state: 'visible', timeout: 8000 });
  await menuItem.click();
}

async function addFileUnderNode(page, nodeName, fileName, filePath) {
  await expandTreeNode(page, nodeName);
  await rightClickMenuAction(page, nodeName, '新增');
  const modal = page.locator('.ant-modal-wrap:visible').last();
  await modal.waitFor({ state: 'visible', timeout: 10000 });
  const nameInput = modal.getByPlaceholder('请输入目录或文件名（有后缀自动识别为文件）').first();
  const pathInput = modal.getByPlaceholder('例如: /frontend/src').first();
  await nameInput.fill(fileName);
  await sleep(400);
  await pathInput.fill(filePath);
  await modal.locator('.ant-btn-primary').last().click();
  await modal.waitFor({ state: 'hidden', timeout: 20000 }).catch(() => undefined);
}

async function clickFileTreeNode(page, fileName) {
  const node = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(fileName) }).first();
  await node.waitFor({ state: 'visible', timeout: 20000 });
  await node.click();
}

async function waitForFileOpened(page, filePath, fileName, timeoutMs = 20000) {
  await page.waitForFunction(
    ({ targetPath, targetName }) => {
      const activeTabList = Array.from(document.querySelectorAll('.ant-tabs-tab-active'));
      const hasActiveTab = activeTabList.some((node) => String(node.textContent || '').includes(targetName));
      if (hasActiveTab) {
        return true;
      }
      const pathNodes = Array.from(document.querySelectorAll('span,div'));
      return pathNodes.some((node) => String(node.textContent || '').includes(targetPath));
    },
    { targetPath: filePath, targetName: fileName },
    { timeout: timeoutMs },
  );
}

async function waitForHistoryCount(page, expectedCount, timeoutMs = 15000) {
  await page.waitForFunction(
    ({ expected }) => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const text = buttons.map((node) => String(node.textContent || '')).find((item) => /历史\(\d+\)/.test(item)) || '';
      const matched = text.match(/历史\((\d+)\)/);
      const count = matched ? Number(matched[1]) : 0;
      return count === expected;
    },
    { expected: expectedCount },
    { timeout: timeoutMs },
  );
}

async function waitForHistoryCountAtLeast(page, expectedMin, timeoutMs = 15000) {
  await page.waitForFunction(
    ({ expected }) => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const text = buttons.map((node) => String(node.textContent || '')).find((item) => /历史\(\d+\)/.test(item)) || '';
      const matched = text.match(/历史\((\d+)\)/);
      const count = matched ? Number(matched[1]) : 0;
      return count >= expected;
    },
    { expected: expectedMin },
    { timeout: timeoutMs },
  );
}

async function getHistoryButton(page) {
  return page.getByRole('button', { name: /历史\(\d+\)/ }).first();
}

async function getHistoryCount(page) {
  const button = await getHistoryButton(page);
  const text = String((await button.textContent().catch(() => '')) || '');
  const matched = text.match(/历史\((\d+)\)/);
  return matched ? Number(matched[1]) : 0;
}

async function openHistoryDialog(page) {
  const button = await getHistoryButton(page);
  await button.waitFor({ state: 'visible', timeout: 20000 });
  await button.click();
  const modal = page.locator('.ant-modal-wrap:visible').last();
  await modal.waitFor({ state: 'visible', timeout: 10000 });
  return modal;
}

async function clearHistoryIfNeeded(page) {
  const count = await getHistoryCount(page);
  if (count <= 0) {
    return false;
  }
  const modal = await openHistoryDialog(page);
  const clearBtn = modal.getByRole('button', { name: '清空' }).first();
  await clearBtn.click();
  await sleep(800);
  await modal.press('Escape').catch(() => undefined);
  await sleep(400);
  return true;
}

async function clickHistoryEntry(page, fileName) {
  const modal = await openHistoryDialog(page);
  const entry = modal.getByRole('button').filter({ hasText: fileName }).first();
  await entry.waitFor({ state: 'visible', timeout: 10000 });
  await entry.click();
  await sleep(400);
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const ts = Date.now();
  const rootDirName = 'test';
  const switchFileName = 'test.md';
  const switchFilePath = path.posix.join(PROJECT_DIR, rootDirName, switchFileName);
  const fileName = `history_case_${ts}.md`;
  const filePath = path.posix.join(PROJECT_DIR, rootDirName, fileName);
  const screenshot = path.join(OUT_DIR, 'webshell-editor-pool-location-history-reopen-check.png');
  const reportPath = path.join(OUT_DIR, 'webshell-editor-pool-location-history-reopen-check.json');

  const result = {
    pageUrl: PAGE_URL,
    projectCode: PROJECT_CODE,
    fileName,
    filePath,
    switchFileName,
    switchFilePath,
    checks: {
      fileAddedByUi: false,
      fileOpenedAfterAdd: false,
      historyRecordedBeforeRefresh: false,
      historyReopenedAfterRefresh: false,
      historyCleared: false,
      historyRecordedAgainBeforeDelete: false,
      deletedFileGracefulHandled: false,
      deletedHistoryRemoved: false,
    },
    trace: {
      contentRequestPaths: [],
      notices: [],
      historyCountBeforeRefresh: 0,
      historyCountAfterRefresh: 0,
      historyCountAfterClear: 0,
      historyCountBeforeDeleteClick: 0,
      historyCountAfterDeleteClick: 0,
    },
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    screenshot,
    pass: false,
  };

  runCurlAllowFail('webshell.workspace_file_add_with_sync', {
    project_code: PROJECT_CODE,
    name: rootDirName,
    path: path.posix.join(PROJECT_DIR, rootDirName),
    is_dir: '1',
    parent_id: '',
  });
  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: filePath,
  });

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1680, height: 980 } });

  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      result.consoleErrors.push(msg.text());
    }
  });
  page.on('pageerror', (error) => result.pageErrors.push(String(error)));
  page.on('requestfailed', (req) => {
    result.failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || 'failed'}`);
  });
  page.on('request', (req) => {
    const url = req.url();
    if (!url.includes('/template_data/data?service=webshell.workspace_file_content')) {
      return;
    }
    const body = String(req.postData() || '');
    const matched = body.match(/"path"\s*:\s*"([^"]+)"/);
    if (matched && matched[1]) {
      result.trace.contentRequestPaths.push(matched[1]);
    }
  });
  page.on('response', async (resp) => {
    const url = resp.url();
    if (!url.includes('/template_data/data?service=')) {
      return;
    }
    if (!/workspace_file_content|workspace_file_add_with_sync|workspace_file_delete_with_sync/.test(url)) {
      return;
    }
    try {
      const body = await resp.text();
      if (/已从历史移除|不可读|不存在|删除/.test(body)) {
        result.trace.notices.push(String(body).slice(0, 300));
      }
    } catch (_error) {
      // ignore
    }
  });

  try {
    await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
    await sleep(1800);
    await ensureProjectSelected(page);
    await clearHistoryIfNeeded(page);
    await sleep(800);

    await addFileUnderNode(page, rootDirName, fileName, filePath);
    result.checks.fileAddedByUi = true;
    await sleep(2200);
    await clickFileTreeNode(page, fileName).catch(() => undefined);
    await waitForFileOpened(page, filePath, fileName, 30000);
    result.checks.fileOpenedAfterAdd = true;

    await waitForHistoryCount(page, 1, 15000);
    result.trace.historyCountBeforeRefresh = await getHistoryCount(page);
    result.checks.historyRecordedBeforeRefresh = result.trace.historyCountBeforeRefresh >= 1;

    await page.reload({ waitUntil: 'networkidle', timeout: 60000 });
    await sleep(1800);
    await ensureProjectSelected(page);
    result.trace.historyCountAfterRefresh = await getHistoryCount(page);

    await clickHistoryEntry(page, fileName);
    await waitForFileOpened(page, filePath, fileName, 30000);
    result.checks.historyReopenedAfterRefresh = true;

    await openHistoryDialog(page);
    const clearBtn = page.locator('.ant-modal-wrap:visible').last().getByRole('button', { name: '清空' }).first();
    await clearBtn.click();
    await sleep(1000);
    await page.locator('.ant-modal-wrap:visible').last().press('Escape').catch(() => undefined);
    await sleep(500);
    await waitForHistoryCount(page, 0, 15000);
    result.trace.historyCountAfterClear = await getHistoryCount(page);
    result.checks.historyCleared = result.trace.historyCountAfterClear === 0;

    await expandTreeNode(page, rootDirName);
    await clickFileTreeNode(page, switchFileName);
    await waitForFileOpened(page, switchFilePath, switchFileName, 30000);
    await clickFileTreeNode(page, fileName);
    await waitForFileOpened(page, filePath, fileName, 30000);
    await sleep(600);
    await waitForHistoryCountAtLeast(page, 1, 15000);
    result.trace.historyCountBeforeDeleteClick = await getHistoryCount(page);
    result.checks.historyRecordedAgainBeforeDelete = result.trace.historyCountBeforeDeleteClick >= 1;

    runCurl('webshell.workspace_file_delete_with_sync', {
      project_code: PROJECT_CODE,
      path: filePath,
    });
    await sleep(2200);
    await page.reload({ waitUntil: 'networkidle', timeout: 60000 });
    await sleep(1800);
    await ensureProjectSelected(page);
    await clickHistoryEntry(page, fileName);

    const notice = page.locator('.ant-message-notice').filter({ hasText: /已从历史移除|不可读|不存在/ }).last();
    await notice.waitFor({ state: 'visible', timeout: 12000 });
    result.checks.deletedFileGracefulHandled = true;
    await sleep(1200);
    result.trace.historyCountAfterDeleteClick = await getHistoryCount(page);
    result.checks.deletedHistoryRemoved = result.trace.historyCountAfterDeleteClick < result.trace.historyCountBeforeDeleteClick;

    result.pass =
      result.checks.fileAddedByUi &&
      result.checks.fileOpenedAfterAdd &&
      result.checks.historyRecordedBeforeRefresh &&
      result.checks.historyReopenedAfterRefresh &&
      result.checks.historyCleared &&
      result.checks.historyRecordedAgainBeforeDelete &&
      result.checks.deletedFileGracefulHandled &&
      result.checks.deletedHistoryRemoved &&
      result.consoleErrors.length === 0 &&
      result.pageErrors.length === 0 &&
      result.failedRequests.length === 0;
  } catch (error) {
    result.error = String(error && error.stack ? error.stack : error);
  } finally {
    await page.screenshot({ path: screenshot, fullPage: true }).catch(() => undefined);
    await browser.close().catch(() => undefined);
    runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
      project_code: PROJECT_CODE,
      path: filePath,
    });
    fs.writeFileSync(reportPath, JSON.stringify(result, null, 2));
    console.log(JSON.stringify(result, null, 2));
    if (!result.pass) {
      process.exit(1);
    }
  }
})();
