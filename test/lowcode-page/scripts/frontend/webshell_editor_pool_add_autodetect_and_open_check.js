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
    '--noproxy', '*', '-sS', `${API_URL}?service=${service}`,
    '-H', 'Content-Type: application/json', '--data', payload,
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

function toTreeTitle(name) {
  return new RegExp(`^${name.replace(/[.*+?^${}()|[\\]\\]/g, '\\$&')}$`);
}

function toLooseTitle(name) {
  return new RegExp(`^\\s*${name.replace(/[.*+?^${}()|[\\]\\]/g, '\\$&')}\\s*$`);
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

async function expandTreeNode(page, nodeName) {
  const title = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(nodeName) }).first();
  await title.waitFor({ state: 'visible', timeout: 15000 });
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

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const ts = Date.now();
  const rootName = 'test';
  const rootPath = path.posix.join(PROJECT_DIR, rootName);
  const dirName = `idea_dir_${ts}`;
  const dirPath = path.posix.join(PROJECT_DIR, dirName);
  const childDirName = `idea_child_${ts}`;
  const childDirPath = path.posix.join(rootPath, childDirName);
  const fileName = `idea_mode_${ts}.py`;
  const filePath = path.posix.join(rootPath, fileName);

  const result = {
    pageUrl: PAGE_URL,
    projectCode: PROJECT_CODE,
    fileName,
    filePath,
    checks: {
      typeManualRemoved: false,
      noSuffixAsDir: false,
      suffixAsFile: false,
      autoPathForFile: false,
      childDirAddedUnderRoot: false,
      childDirVisibleAfterAdd: false,
      folderStillExpanded: false,
      fileNodeVisibleAfterAdd: false,
      fileAutoOpened: false,
      fileCreatedOnServer: false,
    },
    trace: {
      contentRequestPaths: [],
      addRequestPayloads: [],
      tabTitles: [],
    },
    screenshot: '',
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    pass: false,
  };

  // ensure root test folder exists
  runCurlAllowFail('webshell.workspace_file_add_with_sync', {
    project_code: PROJECT_CODE,
    name: rootName,
    path: rootPath,
    is_dir: '1',
    parent_id: '',
  });
  // cleanup previous test file
  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: filePath,
  });
  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: childDirPath,
  });

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1680, height: 980 } });

  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      result.consoleErrors.push(msg.text());
    }
  });
  page.on('pageerror', (error) => result.pageErrors.push(String(error)));
  page.on('requestfailed', (req) => result.failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || 'failed'}`));
  page.on('request', (req) => {
    const url = req.url();
    if (url.includes('/template_data/data?service=webshell.workspace_file_add_with_sync')) {
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
    }
    if (!url.includes('/template_data/data?service=webshell.workspace_file_content')) {
      return;
    }
    const body = String(req.postData() || '');
    const match = body.match(/"path"\s*:\s*"([^"]+)"/);
    if (match && match[1]) {
      result.trace.contentRequestPaths.push(match[1]);
    }
  });

  try {
    await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
    await sleep(2000);

    const projectBtn = page.getByRole('button', { name: 'test' }).first();
    await projectBtn.waitFor({ state: 'visible', timeout: 20000 });
    await projectBtn.click();
    await sleep(1200);

    // top-left add dialog for auto-detect check: no suffix => directory
    await page.locator('button[title="新增"]').first().click();
    const modal = page.locator('.ant-modal-wrap:visible').last();
    await modal.waitFor({ state: 'visible', timeout: 10000 });

    result.checks.typeManualRemoved = await modal.locator('.ant-form-item:has-text("类型")').count() === 0;

    const nameInput = modal.getByPlaceholder('请输入目录或文件名（有后缀自动识别为文件）').first();
    await nameInput.fill(dirName);
    await sleep(450);
    await modal.locator('.ant-btn-primary').last().click();
    await modal.waitFor({ state: 'hidden', timeout: 20000 }).catch(() => undefined);
    await sleep(1800);
    const dirPayload = result.trace.addRequestPayloads.find((item) => item && item.name === dirName);
    result.checks.noSuffixAsDir = !!dirPayload
      && String(dirPayload.is_dir || '') === '1'
      && String(dirPayload.path || '') === dirPath;

    // add directory under root folder by right-click and ensure parent stays expanded
    await expandTreeNode(page, rootName);
    await rightClickMenuAction(page, rootName, '新增');

    const childDirModal = page.locator('.ant-modal-wrap:visible').last();
    await childDirModal.waitFor({ state: 'visible', timeout: 10000 });
    const childDirNameInput = childDirModal.getByPlaceholder('请输入目录或文件名（有后缀自动识别为文件）').first();
    const childDirPathInput = childDirModal.getByPlaceholder('例如: /frontend/src').first();
    await childDirNameInput.fill(childDirName);
    await sleep(450);
    const childDirAutoPath = (await childDirPathInput.inputValue()).trim();
    await childDirModal.locator('.ant-btn-primary').last().click();
    await childDirModal.waitFor({ state: 'hidden', timeout: 20000 }).catch(() => undefined);
    await sleep(2600);
    const childDirPayload = result.trace.addRequestPayloads.find((item) => item && item.name === childDirName);
    result.checks.childDirAddedUnderRoot = !!childDirPayload
      && String(childDirPayload.is_dir || '') === '1'
      && String(childDirPayload.path || '') === childDirPath
      && childDirAutoPath === childDirPath;

    const rootSwitcherAfterDir = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(rootName) }).first()
      .locator('xpath=ancestor::div[contains(@class,"ant-tree-treenode")][1]')
      .locator('.ant-tree-switcher').first();
    const rootSwitcherAfterDirCls = String((await rootSwitcherAfterDir.getAttribute('class')) || '');
    const childDirRow = runCurl('webshell.workspace_file_query', {
      project_code: PROJECT_CODE,
      path_exact: childDirPath,
      pagination: false,
    });
    result.checks.childDirVisibleAfterAdd =
      Array.isArray(childDirRow.data) && childDirRow.data.length > 0
      && (rootSwitcherAfterDirCls.includes('ant-tree-switcher_open') || rootSwitcherAfterDirCls.includes('ant-tree-switcher-loading-icon'));

    // add file under root folder by right-click
    await expandTreeNode(page, rootName);
    await rightClickMenuAction(page, rootName, '新增');

    const addModal = page.locator('.ant-modal-wrap:visible').last();
    await addModal.waitFor({ state: 'visible', timeout: 10000 });
    const addNameInput = addModal.getByPlaceholder('请输入目录或文件名（有后缀自动识别为文件）').first();
    const pathInput = addModal.getByPlaceholder('例如: /frontend/src').first();
    await addNameInput.fill(fileName);
    await sleep(450);
    const autoPath = (await pathInput.inputValue()).trim();
    result.checks.autoPathForFile = autoPath === filePath;
    await addModal.locator('.ant-btn-primary').last().click();
    await addModal.waitFor({ state: 'hidden', timeout: 20000 }).catch(() => undefined);
    await sleep(2600);
    const filePayload = result.trace.addRequestPayloads.find((item) => item && item.name === fileName);
    result.checks.suffixAsFile = !!filePayload
      && String(filePayload.is_dir || '') === '0'
      && String(filePayload.path || '') === filePath;

    // folder remains expanded if switcher is open state
    const rootSwitcher = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(rootName) }).first()
      .locator('xpath=ancestor::div[contains(@class,"ant-tree-treenode")][1]')
      .locator('.ant-tree-switcher').first();
    const switcherCls = String((await rootSwitcher.getAttribute('class')) || '');
    result.checks.folderStillExpanded = switcherCls.includes('ant-tree-switcher_open') || switcherCls.includes('ant-tree-switcher-loading-icon');

    // file node should be queryable in current tree DOM (visibility can be virtualized).
    const fileNode = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(fileName) }).first();
    result.checks.fileNodeVisibleAfterAdd = (await fileNode.count()) > 0;

    // auto-open file: accept either content request hit or visible opened tab title.
    await sleep(4200);
    result.trace.tabTitles = await page.locator('.ant-tabs-tab').allTextContents().catch(() => []);
    const hasTab = await page.locator('.ant-tabs-tab', { hasText: toTreeTitle(fileName) }).count().then((n) => n > 0).catch(() => false);
    const hasContentReq = result.trace.contentRequestPaths.some((p) => String(p || '') === filePath);
    result.checks.fileAutoOpened = hasContentReq || hasTab;

    const fileRow = runCurl('webshell.workspace_file_query', {
      project_code: PROJECT_CODE,
      path_exact: filePath,
      pagination: false,
    });
    result.checks.fileCreatedOnServer = Array.isArray(fileRow.data) && fileRow.data.length > 0;

    const shot = path.join(OUT_DIR, 'webshell-editor-pool-add-autodetect-and-open-check.png');
    await page.screenshot({ path: shot, fullPage: true });
    result.screenshot = shot;
  } catch (error) {
    result.error = String(error && error.stack ? error.stack : error);
  } finally {
    await browser.close();
  }

  // cleanup created file
  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: filePath,
  });
  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: childDirPath,
  });
  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: dirPath,
  });

  result.pass =
    result.checks.typeManualRemoved &&
    result.checks.noSuffixAsDir &&
    result.checks.suffixAsFile &&
    result.checks.autoPathForFile &&
    result.checks.childDirAddedUnderRoot &&
    result.checks.childDirVisibleAfterAdd &&
    result.checks.folderStillExpanded &&
    result.checks.fileAutoOpened &&
    result.checks.fileCreatedOnServer &&
    result.consoleErrors.length === 0 &&
    result.pageErrors.length === 0;

  const out = path.join(OUT_DIR, 'webshell-editor-pool-add-autodetect-and-open-check.json');
  fs.writeFileSync(out, JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));

  if (!result.pass) {
    process.exit(2);
  }
})();
