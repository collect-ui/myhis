#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawn, spawnSync } = require('child_process');
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
    return { ok: true, value: runCurl(service, data) };
  } catch (error) {
    return { ok: false, error: String(error && error.message ? error.message : error) };
  }
}

function runCurlAsync(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  return new Promise((resolve) => {
    const child = spawn('curl', [
      '--noproxy', '*',
      '-sS',
      `${API_URL}?service=${service}`,
      '-H',
      'Content-Type: application/json',
      '--data',
      payload,
    ], { stdio: ['ignore', 'pipe', 'pipe'] });
    let out = '';
    let err = '';
    child.stdout.on('data', (chunk) => { out += chunk.toString(); });
    child.stderr.on('data', (chunk) => { err += chunk.toString(); });
    child.on('close', (code) => {
      if (code !== 0) {
        resolve({ ok: false, error: err || `curl failed: ${service}` });
        return;
      }
      try {
        const parsed = JSON.parse(String(out || '{}'));
        const ok = parsed && String(parsed.code || '') === '0' && parsed.success !== false;
        resolve({ ok, parsed, error: ok ? '' : String(parsed?.msg || 'unknown error') });
      } catch (parseError) {
        resolve({ ok: false, error: `parse response failed (${service}): ${parseError.message}` });
      }
    });
  });
}

function toTreeTitle(name) {
  return new RegExp(`^${name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`);
}

function toLooseTitle(name) {
  return new RegExp(`^\\s*${name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*$`);
}

async function rightClickMenuAction(page, nodeName, menuName) {
  const node = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(nodeName) }).first();
  await node.waitFor({ state: 'visible', timeout: 15000 });
  await node.click({ button: 'right' });
  const menu = page.locator('div[role="menu"].contexify:visible, div[role="menu"]:visible').last();
  await menu.waitFor({ state: 'visible', timeout: 8000 });
  const menuItem = menu.locator('[role="menuitem"]', { hasText: toLooseTitle(menuName) }).first();
  await menuItem.waitFor({ state: 'visible', timeout: 8000 });
  await menuItem.click();
}

async function switchParentInDialog(page, parentName) {
  const modal = page.locator('.ant-modal-wrap:visible').last();
  const selector = modal.locator('.ant-form-item:has-text("上级目录") .ant-select-selector').first();
  await selector.waitFor({ state: 'visible', timeout: 8000 });
  await selector.click();
  const option = page.locator('.ant-select-tree-dropdown:visible .ant-select-tree-title', { hasText: toTreeTitle(parentName) }).first();
  await option.waitFor({ state: 'visible', timeout: 8000 });
  await option.click();
  await page.waitForTimeout(200);
  const selected = await modal.locator('.ant-form-item:has-text("上级目录") .ant-select-selection-item').first().textContent().catch(() => '');
  return String(selected || '').includes(parentName);
}

async function fillFileDialog(page, { name, fullPath }) {
  const modal = page.locator('.ant-modal-wrap:visible').last();
  await modal.waitFor({ state: 'visible', timeout: 10000 });

  const nameInput = modal.getByPlaceholder('请输入目录或文件名').first();
  const pathInput = modal.getByPlaceholder('例如: /frontend/src').first();
  await nameInput.fill(name);
  await pathInput.fill(fullPath);

  await modal.locator('.ant-btn-primary').last().click();
  await modal.waitFor({ state: 'hidden', timeout: 20000 }).catch(() => undefined);
}

async function waitTreeHasNode(page, nodeName, timeoutMs = 20000) {
  const node = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(nodeName) }).first();
  await node.waitFor({ state: 'visible', timeout: timeoutMs });
}

async function expandTreeNode(page, nodeName) {
  const title = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(nodeName) }).first();
  await title.waitFor({ state: 'visible', timeout: 15000 });
  const switcher = title.locator('xpath=ancestor::div[contains(@class,"ant-tree-treenode")][1]').locator('.ant-tree-switcher').first();
  const hasSwitcher = await switcher.count();
  if (!hasSwitcher) {
    return;
  }
  const cls = String((await switcher.getAttribute('class')) || '');
  if (cls.includes('ant-tree-switcher_close')) {
    await switcher.click();
    await sleep(600);
  }
}

async function openRootSource(page) {
  const tree = page.locator('.workspace-source-tree').first();
  await tree.waitFor({ state: 'visible', timeout: 20000 });
  await sleep(800);
}

async function triggerSync(page) {
  const btn = page.locator('button[title="同步"]').first();
  await btn.waitFor({ state: 'visible', timeout: 15000 });
  await btn.click();
  const confirm = page.locator('.ant-modal-wrap:visible .ant-btn-primary').last();
  await confirm.waitFor({ state: 'visible', timeout: 8000 });
  await confirm.click();
  await sleep(2500);
}

async function ensureProjectButton(page) {
  const projectBtn = page.getByRole('button', { name: 'test' }).first();
  await projectBtn.waitFor({ state: 'visible', timeout: 20000 });
  await projectBtn.click();
  await sleep(1200);
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const ts = Date.now();
  const rootDirName = `test`;
  const level1Name = `lvl1_${ts}`;
  const level2Name = `lvl2_${ts}`;
  const fileName = `case_${ts}.txt`;
  const renamedFileName = `case_${ts}_renamed.txt`;

  const rootDirPath = path.posix.join(PROJECT_DIR, rootDirName);
  const level1Path = path.posix.join(rootDirPath, level1Name);
  const level2Path = path.posix.join(level1Path, level2Name);
  const filePath = path.posix.join(level2Path, fileName);
  const renamedFilePath = path.posix.join(level2Path, renamedFileName);

  const result = {
    pageUrl: PAGE_URL,
    projectCode: PROJECT_CODE,
    paths: {
      rootDirPath,
      level1Path,
      level2Path,
      filePath,
      renamedFilePath,
    },
    checks: {
      addRootDir: false,
      addNestedDir: false,
      addNestedFile: false,
      editDialogPrefill: false,
      editParentShown: false,
      parentDropdownSelectable: false,
      renameFile: false,
      deleteFile: false,
      treeHasMultiLevel: false,
      syncLocked: false,
      serverFileCreated: false,
      serverFileRenamed: false,
      serverFileDeleted: false,
    },
    api: {
      syncLockSecondCallMsg: '',
      calls: [],
    },
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    screenshot: '',
    pass: false,
  };

  // Clean previous /data/project/test/test so case is deterministic.
  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: rootDirPath,
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
  page.on('response', async (resp) => {
    const url = resp.url();
    if (!url.includes('/template_data/data?service=webshell.')) {
      return;
    }
    if (!/(workspace_file_|workspace_project_sync_files)/.test(url)) {
      return;
    }
    let body = '';
    try {
      body = await resp.text();
    } catch (error) {
      body = `read_body_failed: ${String(error && error.message ? error.message : error)}`;
    }
    result.api.calls.push({
      url,
      status: resp.status(),
      body: String(body || '').slice(0, 800),
    });
  });

  try {
    await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
    await sleep(2200);

    await ensureProjectButton(page);
    await openRootSource(page);

    // 1) Add root folder via left-top Add button.
    const addBtn = page.locator('button[title="新增"]').first();
    await addBtn.click();
    await fillFileDialog(page, { name: rootDirName, fullPath: rootDirPath });
    await sleep(2200);
    await waitTreeHasNode(page, rootDirName);
    result.checks.addRootDir = true;

    // 2) Right-click root folder -> Add level1 directory.
    await rightClickMenuAction(page, rootDirName, '新增');
    await fillFileDialog(page, { name: level1Name, fullPath: level1Path });
    await sleep(2200);
    await expandTreeNode(page, rootDirName);
    await waitTreeHasNode(page, level1Name);
    result.checks.addNestedDir = true;

    // 3) Right-click level1 -> Add level2 directory.
    await rightClickMenuAction(page, level1Name, '新增');
    await fillFileDialog(page, { name: level2Name, fullPath: level2Path });
    await sleep(2200);
    await expandTreeNode(page, rootDirName);
    await expandTreeNode(page, level1Name);
    await waitTreeHasNode(page, level2Name);

    // 4) Right-click level2 -> Add file.
    await rightClickMenuAction(page, level2Name, '新增');
    await fillFileDialog(page, { name: fileName, fullPath: filePath });
    await sleep(2600);
    await expandTreeNode(page, rootDirName);
    await expandTreeNode(page, level1Name);
    await expandTreeNode(page, level2Name);
    await waitTreeHasNode(page, fileName);
    result.checks.addNestedFile = true;

    // Server file created check.
    const fileContentRes = runCurl('webshell.workspace_file_content', {
      project_code: PROJECT_CODE,
      path: filePath,
    });
    result.checks.serverFileCreated = String(fileContentRes?.data?.path || '') === filePath;

    // 5) Right-click file -> Edit, verify prefill, rename.
    await rightClickMenuAction(page, fileName, '编辑');
    const modal = page.locator('.ant-modal-wrap:visible').last();
    const nameInput = modal.getByPlaceholder('请输入目录或文件名').first();
    const pathInput = modal.getByPlaceholder('例如: /frontend/src').first();
    const oldName = await nameInput.inputValue();
    const oldPath = await pathInput.inputValue();
    const parentText = await modal.locator('.ant-form-item:has-text("上级目录") .ant-select-selection-item').first().textContent().catch(() => '');
    result.checks.editDialogPrefill = oldName.includes(fileName) && oldPath.includes(filePath);
    result.checks.editParentShown = String(parentText || '').includes(level2Name);
    const parentSwitchL1 = await switchParentInDialog(page, level1Name);
    const parentSwitchL2 = await switchParentInDialog(page, level2Name);
    result.checks.parentDropdownSelectable = parentSwitchL1 && parentSwitchL2;
    await nameInput.fill(renamedFileName);
    await pathInput.fill(renamedFilePath);
    await modal.locator('.ant-btn-primary').last().click();
    await modal.waitFor({ state: 'hidden', timeout: 20000 }).catch(() => undefined);
    await sleep(2600);
    await waitTreeHasNode(page, renamedFileName);
    result.checks.renameFile = true;

    const renamedRes = runCurl('webshell.workspace_file_content', {
      project_code: PROJECT_CODE,
      path: renamedFilePath,
    });
    result.checks.serverFileRenamed = String(renamedRes?.data?.path || '') === renamedFilePath;

    const oldPathCheck = runCurlAllowFail('webshell.workspace_file_content', {
      project_code: PROJECT_CODE,
      path: filePath,
    });
    if (!oldPathCheck.ok) {
      result.checks.serverFileRenamed = result.checks.serverFileRenamed && true;
    }

    // 6) Right-click renamed file -> Delete.
    await rightClickMenuAction(page, renamedFileName, '删除');
    const confirm = page.locator('.ant-modal-wrap:visible .ant-btn-primary').last();
    await confirm.waitFor({ state: 'visible', timeout: 8000 });
    await confirm.click();
    await sleep(3000);
    result.checks.deleteFile = true;

    const deletedRes = runCurlAllowFail('webshell.workspace_file_content', {
      project_code: PROJECT_CODE,
      path: renamedFilePath,
    });
    result.checks.serverFileDeleted = !deletedRes.ok;

    // 7) Multi-level tree display check.
    await waitTreeHasNode(page, rootDirName);
    await waitTreeHasNode(page, level1Name);
    await waitTreeHasNode(page, level2Name);
    result.checks.treeHasMultiLevel = true;

    // 8) Sync lock check: issue 2 parallel sync calls, second should fail fast with lock message.
    const projectList = runCurl('webshell.workspace_project_query', { project_code: PROJECT_CODE, pagination: false });
    const project = Array.isArray(projectList?.data) ? projectList.data[0] : null;
    if (project && project.webshell_workspace_project_id) {
      const syncPayload = {
        project_code: PROJECT_CODE,
        webshell_workspace_project_id: project.webshell_workspace_project_id,
        exclude_dirs: project.exclude_dirs || 'node_modules,.git,dist,.next',
      };
      const [first, second] = await Promise.all([
        runCurlAsync('webshell.workspace_project_sync_files', syncPayload),
        runCurlAsync('webshell.workspace_project_sync_files', syncPayload),
      ]);
      const lockMsg = [first.error, second.error, first?.parsed?.msg, second?.parsed?.msg].filter(Boolean).join(' | ');
      result.api.syncLockSecondCallMsg = lockMsg;
      result.checks.syncLocked = (!first.ok || !second.ok) && /同步任务正在执行中|lock=/i.test(lockMsg);
      if (!first.ok && !second.ok) {
        throw new Error(`sync calls both failed: ${lockMsg}`);
      }
    }

    // 9) Trigger one UI sync to ensure button still works with lock logic.
    await triggerSync(page);

    const shot = path.join(OUT_DIR, 'webshell-editor-pool-workspace-file-crud-sync-check.png');
    await page.screenshot({ path: shot, fullPage: true });
    result.screenshot = shot;
  } catch (error) {
    result.error = String(error && error.stack ? error.stack : error);
  } finally {
    await browser.close();
  }

  // Cleanup generated test folder.
  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: rootDirPath,
  });

  result.pass =
    result.checks.addRootDir &&
    result.checks.addNestedDir &&
    result.checks.addNestedFile &&
    result.checks.editDialogPrefill &&
    result.checks.editParentShown &&
    result.checks.parentDropdownSelectable &&
    result.checks.renameFile &&
    result.checks.deleteFile &&
    result.checks.treeHasMultiLevel &&
    result.checks.syncLocked &&
    result.checks.serverFileCreated &&
    result.checks.serverFileRenamed &&
    result.checks.serverFileDeleted &&
    result.consoleErrors.length === 0 &&
    result.pageErrors.length === 0;

  const out = path.join(OUT_DIR, 'webshell-editor-pool-workspace-file-crud-sync-check.json');
  fs.writeFileSync(out, JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));

  if (!result.pass) {
    process.exit(2);
  }
})();
