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

function escapeRegExp(value) {
  return String(value).replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function exactText(value) {
  return new RegExp(`^${escapeRegExp(value)}$`);
}

function looseText(value) {
  return new RegExp(`^\\s*${escapeRegExp(value)}\\s*$`);
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

async function selectProject(page) {
  const projectBtn = page.getByRole('button', { name: PROJECT_CODE }).first();
  await projectBtn.waitFor({ state: 'visible', timeout: 20000 });
  await projectBtn.click();
  await sleep(1200);
}

async function waitTreeNode(page, nodeName, timeout = 20000) {
  const node = page.locator('.workspace-source-tree .ant-tree-title', { hasText: exactText(nodeName) }).first();
  await node.waitFor({ state: 'visible', timeout });
  return node;
}

async function expandTreeNode(page, nodeName) {
  const title = await waitTreeNode(page, nodeName);
  const switcher = title
    .locator('xpath=ancestor::div[contains(@class,"ant-tree-treenode")][1]')
    .locator('.ant-tree-switcher')
    .first();
  if (await switcher.count() <= 0) {
    return;
  }
  const cls = String((await switcher.getAttribute('class')) || '');
  if (cls.includes('ant-tree-switcher_close')) {
    await switcher.click();
    await sleep(900);
  }
}

async function rightClickMenuAction(page, nodeName, menuName) {
  const node = await waitTreeNode(page, nodeName);
  await node.click({ button: 'right' });
  const menu = page.locator('div[role="menu"].contexify:visible, div[role="menu"]:visible').last();
  await menu.waitFor({ state: 'visible', timeout: 8000 });
  const menuItem = menu.locator('[role="menuitem"]', { hasText: looseText(menuName) }).first();
  await menuItem.waitFor({ state: 'visible', timeout: 8000 });
  await menuItem.click();
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const ts = Date.now();
  const parentName = 'test';
  const parentPath = path.posix.join(PROJECT_DIR, parentName);
  const fileName = `enter_save_${ts}.md`;
  const filePath = path.posix.join(parentPath, fileName);

  const result = {
    pageUrl: PAGE_URL,
    projectCode: PROJECT_CODE,
    parentPath,
    fileName,
    filePath,
    checks: {
      parentReady: false,
      dialogOpened: false,
      enterTriggeredAddRequest: false,
      modalClosedAfterEnter: false,
      serverFileCreated: false,
      fileVisibleInTree: false,
      fileSelectedInTree: false,
      fileOpened: false,
    },
    trace: {
      addRequestPayloads: [],
      contentRequestPaths: [],
      tabTitles: [],
    },
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    screenshot: '',
    pass: false,
  };

  runCurlAllowFail('webshell.workspace_file_add_with_sync', {
    project_code: PROJECT_CODE,
    name: parentName,
    path: parentPath,
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
    if (!url.includes('/template_data/data?service=webshell.workspace_file_add_with_sync')) {
      if (!url.includes('/template_data/data?service=webshell.workspace_file_content')) {
        return;
      }
      const body = String(req.postData() || '');
      const match = body.match(/"path"\s*:\s*"([^"]+)"/);
      if (match && match[1]) {
        result.trace.contentRequestPaths.push(match[1]);
      }
      return;
    }
    const raw = String(req.postData() || '');
    try {
      const payload = JSON.parse(raw || '{}');
      result.trace.addRequestPayloads.push({
        name: String(payload.name || ''),
        path: String(payload.path || ''),
        is_dir: String(payload.is_dir || ''),
        parent_id: String(payload.parent_id || ''),
      });
    } catch (error) {
      result.trace.addRequestPayloads.push({
        parse_error: String(error && error.message ? error.message : error),
        raw: raw.slice(0, 400),
      });
    }
  });

  try {
    await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
    await sleep(2200);
    await selectProject(page);

    await waitTreeNode(page, parentName);
    result.checks.parentReady = true;

    await expandTreeNode(page, parentName);
    await rightClickMenuAction(page, parentName, '新增');

    const modal = page.locator('.ant-modal-wrap:visible').last();
    await modal.waitFor({ state: 'visible', timeout: 10000 });
    result.checks.dialogOpened = true;

    const nameInput = modal.getByPlaceholder('请输入目录或文件名（有后缀自动识别为文件）').first();
    const pathInput = modal.getByPlaceholder('例如: /frontend/src').first();
    await nameInput.fill(fileName);
    await sleep(500);

    const autoPath = String(await pathInput.inputValue()).trim();
    await nameInput.press('Enter');
    await modal.waitFor({ state: 'hidden', timeout: 20000 });
    result.checks.modalClosedAfterEnter = true;

    await sleep(2600);
    const addPayload = result.trace.addRequestPayloads.find((item) => item && item.name === fileName);
    result.checks.enterTriggeredAddRequest = !!addPayload
      && String(addPayload.path || '') === filePath
      && String(addPayload.is_dir || '') === '0'
      && autoPath === filePath;

    const fileRow = runCurl('webshell.workspace_file_query', {
      project_code: PROJECT_CODE,
      path_exact: filePath,
      pagination: false,
    });
    result.checks.serverFileCreated = Array.isArray(fileRow.data) && fileRow.data.length > 0;

    await expandTreeNode(page, parentName);
    const fileNode = page.locator('.workspace-source-tree .ant-tree-title', { hasText: exactText(fileName) }).first();
    await fileNode.waitFor({ state: 'visible', timeout: 15000 });
    result.checks.fileVisibleInTree = true;

    const fileWrapper = fileNode.locator('xpath=ancestor::*[contains(@class,"ant-tree-node-content-wrapper")][1]');
    const selectedClass = String((await fileWrapper.getAttribute('class')) || '');
    result.checks.fileSelectedInTree = selectedClass.includes('ant-tree-node-selected');

    await sleep(4200);
    result.trace.tabTitles = await page.locator('.ant-tabs-tab').allTextContents().catch(() => []);
    const hasTab = await page.locator('.ant-tabs-tab', { hasText: exactText(fileName) }).count().then((n) => n > 0).catch(() => false);
    const hasContentReq = result.trace.contentRequestPaths.some((p) => String(p || '') === filePath);
    result.checks.fileOpened = hasTab || hasContentReq;

    const shot = path.join(OUT_DIR, 'webshell-editor-pool-file-dialog-enter-save-check.png');
    await page.screenshot({ path: shot, fullPage: true });
    result.screenshot = shot;
  } catch (error) {
    result.error = String(error && error.stack ? error.stack : error);
  } finally {
    await browser.close();
  }

  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: filePath,
  });

  result.pass =
    result.checks.parentReady &&
    result.checks.dialogOpened &&
    result.checks.enterTriggeredAddRequest &&
    result.checks.modalClosedAfterEnter &&
    result.checks.serverFileCreated &&
    result.checks.fileVisibleInTree &&
    result.checks.fileSelectedInTree &&
    result.checks.fileOpened &&
    result.consoleErrors.length === 0 &&
    result.pageErrors.length === 0;

  const out = path.join(OUT_DIR, 'webshell-editor-pool-file-dialog-enter-save-check.json');
  fs.writeFileSync(out, JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));

  if (!result.pass) {
    process.exit(2);
  }
})();
