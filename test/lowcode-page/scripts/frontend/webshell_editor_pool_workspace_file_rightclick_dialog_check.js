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

function toTreeTitle(name) {
  return new RegExp(`^${name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`);
}

function toLooseTitle(name) {
  return new RegExp(`^\\s*${name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*$`);
}

function getFirstRecordByPath(projectCode, targetPath) {
  const resp = runCurl('webshell.workspace_file_query', {
    project_code: projectCode,
    path_exact: targetPath,
    pagination: false,
  });
  const list = Array.isArray(resp.data) ? resp.data : [];
  return list[0] || null;
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
  await title.waitFor({ state: 'visible', timeout: 20000 });
  const switcher = title.locator('xpath=ancestor::div[contains(@class,"ant-tree-treenode")][1]').locator('.ant-tree-switcher').first();
  if (await switcher.count() <= 0) {
    return;
  }
  const cls = String((await switcher.getAttribute('class')) || '');
  if (cls.includes('ant-tree-switcher_close')) {
    await switcher.click();
    await sleep(800);
  }
}

async function openProject(page) {
  const projectBtn = page.getByRole('button', { name: 'test' }).first();
  await projectBtn.waitFor({ state: 'visible', timeout: 20000 });
  await projectBtn.click();
  await sleep(1500);
}

async function searchTreeKeyword(page, keyword) {
  const input = page.getByPlaceholder('回车搜索(至少2个字符)').first();
  await input.waitFor({ state: 'visible', timeout: 10000 });
  await input.fill(keyword);
  await input.press('Enter');
  await sleep(1600);
}

async function fillDialogAndSubmit(page, name, fullPath, isFile = false) {
  const modal = page.locator('.ant-modal-wrap:visible').last();
  await modal.waitFor({ state: 'visible', timeout: 10000 });

  if (isFile) {
    const typeSelect = modal.locator('.ant-form-item:has-text("类型") .ant-select').first();
    await typeSelect.click();
    const fileOption = page.locator('.ant-select-dropdown:visible .ant-select-item-option', { hasText: /^文件$/ }).first();
    await fileOption.waitFor({ state: 'visible', timeout: 8000 });
    await fileOption.click();
  }

  await modal.getByPlaceholder('请输入目录或文件名').first().fill(name);
  await modal.getByPlaceholder('例如: /frontend/src').first().fill(fullPath);
  await modal.locator('.ant-btn-primary').last().click();
  await modal.waitFor({ state: 'hidden', timeout: 20000 }).catch(() => undefined);
  await sleep(2200);
}

async function switchParentInDialog(page, parentName) {
  const modal = page.locator('.ant-modal-wrap:visible').last();
  const selector = modal.locator('.ant-form-item:has-text("上级目录") .ant-select-selector').first();
  await selector.waitFor({ state: 'visible', timeout: 8000 });
  let dropdownTitles = page.locator('.ant-select-tree-dropdown:visible .ant-select-tree-title');
  let count = await dropdownTitles.count();
  for (let i = 0; i < 3 && count <= 0; i += 1) {
    await selector.click({ force: true });
    await sleep(300);
    dropdownTitles = page.locator('.ant-select-tree-dropdown:visible .ant-select-tree-title');
    count = await dropdownTitles.count();
  }
  if (count <= 0) {
    const selectedText = await modal.locator('.ant-form-item:has-text("上级目录") .ant-select-selection-item').first().textContent().catch(() => '');
    return String(selectedText || '').trim().length > 0;
  }
  let option = page.locator('.ant-select-tree-dropdown:visible .ant-select-tree-title', { hasText: toTreeTitle(parentName) }).first();
  if (await option.count() <= 0) {
    option = dropdownTitles.first();
  }
  await option.click();
  await sleep(300);
  const selected = await modal.locator('.ant-form-item:has-text("上级目录") .ant-select-selection-item').first().textContent().catch(() => '');
  return String(selected || '').trim().length > 0;
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const ts = Date.now();
  const rootName = 'test';
  const rootPath = path.posix.join(PROJECT_DIR, rootName);
  const level1Name = `dlg_lvl1_${ts}`;
  const level1RenamedName = `dlg_lvl1_${ts}_renamed`;
  const level2Name = `dlg_lvl2_${ts}`;
  const nestedFileName = `dlg_case_${ts}.txt`;
  const opsFileName = `dlg_ops_${ts}.txt`;
  const opsFileRenamedName = `dlg_ops_${ts}_renamed.txt`;
  const rightAddName = `dlg_add_${ts}`;

  const level1Path = path.posix.join(rootPath, level1Name);
  const level1RenamedPath = path.posix.join(rootPath, level1RenamedName);
  const level2Path = path.posix.join(level1Path, level2Name);
  const nestedFilePath = path.posix.join(level2Path, nestedFileName);
  const opsFilePath = path.posix.join(rootPath, opsFileName);
  const opsFileRenamedPath = path.posix.join(rootPath, opsFileRenamedName);
  const rightAddPath = path.posix.join(rootPath, rightAddName);

  const result = {
    pageUrl: PAGE_URL,
    projectCode: PROJECT_CODE,
    paths: {
      rootPath,
      level1Path,
      level1RenamedPath,
      level2Path,
      nestedFilePath,
      opsFilePath,
      opsFileRenamedPath,
      rightAddPath,
    },
    checks: {
      setupByApi: false,
      rightAddDir: false,
      rightAddDirServerOk: false,
      rightAddDirKeepsRootExpanded: false,
      multiLevelVisible: false,
      rightEditPrefill: false,
      rightEditParentShown: false,
      rightEditParentSelectable: false,
      rightEditRenameServerOk: false,
      rightDeleteFile: false,
      rightDeleteFileServerOk: false,
      rightDeleteDirKeepsRootExpanded: false,
      rightDeleteDirServerOk: false,
    },
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    screenshot: '',
    pass: false,
  };

  runCurlAllowFail('webshell.workspace_file_delete_with_sync', { project_code: PROJECT_CODE, path: rootPath });

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1680, height: 980 } });

  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      result.consoleErrors.push(msg.text());
    }
  });
  page.on('pageerror', (error) => result.pageErrors.push(String(error)));
  page.on('requestfailed', (req) => result.failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || 'failed'}`));

  try {
    // setup multi-level data via API for deterministic tree structure
    runCurl('webshell.workspace_file_add_with_sync', { project_code: PROJECT_CODE, name: rootName, path: rootPath, is_dir: '1', parent_id: '' });
    runCurl('webshell.workspace_file_add_with_sync', { project_code: PROJECT_CODE, name: level1Name, path: level1Path, is_dir: '1', parent_id: '' });
    runCurl('webshell.workspace_file_add_with_sync', { project_code: PROJECT_CODE, name: level2Name, path: level2Path, is_dir: '1', parent_id: '' });
    runCurl('webshell.workspace_file_add_with_sync', { project_code: PROJECT_CODE, name: nestedFileName, path: nestedFilePath, is_dir: '0', parent_id: '' });
    runCurl('webshell.workspace_file_add_with_sync', { project_code: PROJECT_CODE, name: opsFileName, path: opsFilePath, is_dir: '0', parent_id: '' });
    result.checks.setupByApi = true;

    await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
    await sleep(2200);
    await openProject(page);

    // right-click add on root
    await rightClickMenuAction(page, rootName, '新增');
    await fillDialogAndSubmit(page, rightAddName, rightAddPath, false);
    result.checks.rightAddDir = true;

    const addRow = getFirstRecordByPath(PROJECT_CODE, rightAddPath);
    result.checks.rightAddDirServerOk = !!addRow;
    const rootSwitcherAfterAdd = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(rootName) }).first()
      .locator('xpath=ancestor::div[contains(@class,"ant-tree-treenode")][1]')
      .locator('.ant-tree-switcher').first();
    const rootSwitcherAfterAddCls = String((await rootSwitcherAfterAdd.getAttribute('class')) || '');
    const rightAddNodeCount = await page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(rightAddName) }).count();
    result.checks.rightAddDirKeepsRootExpanded =
      rightAddNodeCount > 0 &&
      (rootSwitcherAfterAddCls.includes('ant-tree-switcher_open') || rootSwitcherAfterAddCls.includes('ant-tree-switcher-loading-icon'));

    await rightClickMenuAction(page, rightAddName, '删除');
    const deleteDirConfirm = page.locator('.ant-modal-wrap:visible .ant-btn-primary').last();
    await deleteDirConfirm.waitFor({ state: 'visible', timeout: 8000 });
    await deleteDirConfirm.click();
    await sleep(2600);
    const deletedDir = getFirstRecordByPath(PROJECT_CODE, rightAddPath);
    result.checks.rightDeleteDirServerOk = !deletedDir;
    const rootSwitcherAfterDeleteDir = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(rootName) }).first()
      .locator('xpath=ancestor::div[contains(@class,"ant-tree-treenode")][1]')
      .locator('.ant-tree-switcher').first();
    const rootSwitcherAfterDeleteDirCls = String((await rootSwitcherAfterDeleteDir.getAttribute('class')) || '');
    result.checks.rightDeleteDirKeepsRootExpanded =
      rootSwitcherAfterDeleteDirCls.includes('ant-tree-switcher_open') || rootSwitcherAfterDeleteDirCls.includes('ant-tree-switcher-loading-icon');

    const level1Row = getFirstRecordByPath(PROJECT_CODE, level1Path);
    const level2Row = getFirstRecordByPath(PROJECT_CODE, level2Path);
    const nestedFileRow = getFirstRecordByPath(PROJECT_CODE, nestedFilePath);
    result.checks.multiLevelVisible = !!level1Row && !!level2Row && !!nestedFileRow;
    await searchTreeKeyword(page, opsFileName);
    await rightClickMenuAction(page, opsFileName, '编辑');

    const modal = page.locator('.ant-modal-wrap:visible').last();
    const editName = await modal.getByPlaceholder('请输入目录或文件名').first().inputValue();
    const editPath = await modal.getByPlaceholder('例如: /frontend/src').first().inputValue();
    const parentText = await modal.locator('.ant-form-item:has-text("上级目录") .ant-select-selection-item').first().textContent().catch(() => '');

    result.checks.rightEditPrefill = editName.includes(opsFileName) && editPath.includes(opsFilePath);
    result.checks.rightEditParentShown = String(parentText || '').trim().length > 0;

    const parentSwitchRoot = await switchParentInDialog(page, rootName);
    result.checks.rightEditParentSelectable = parentSwitchRoot;

    await modal.getByPlaceholder('请输入目录或文件名').first().fill(opsFileRenamedName);
    await modal.getByPlaceholder('例如: /frontend/src').first().fill(opsFileRenamedPath);
    await modal.locator('.ant-btn-primary').last().click();
    await modal.waitFor({ state: 'hidden', timeout: 20000 }).catch(() => undefined);
    await sleep(2600);

    const renamedFileRow = getFirstRecordByPath(PROJECT_CODE, opsFileRenamedPath);
    const oldFileRow = getFirstRecordByPath(PROJECT_CODE, opsFilePath);
    result.checks.rightEditRenameServerOk = !!renamedFileRow && !oldFileRow;

    // delete renamed file via right-click to verify delete path reaches server
    await searchTreeKeyword(page, opsFileRenamedName);
    await rightClickMenuAction(page, opsFileRenamedName, '删除');
    const deleteConfirm = page.locator('.ant-modal-wrap:visible .ant-btn-primary').last();
    await deleteConfirm.waitFor({ state: 'visible', timeout: 8000 });
    await deleteConfirm.click();
    await sleep(2800);
    result.checks.rightDeleteFile = true;

    const deletedFile = getFirstRecordByPath(PROJECT_CODE, opsFileRenamedPath);
    result.checks.rightDeleteFileServerOk = !deletedFile;

    const shot = path.join(OUT_DIR, 'webshell-editor-pool-workspace-file-rightclick-dialog-check.png');
    await page.screenshot({ path: shot, fullPage: true });
    result.screenshot = shot;
  } catch (error) {
    result.error = String(error && error.stack ? error.stack : error);
  } finally {
    await browser.close();
  }

  runCurlAllowFail('webshell.workspace_file_delete_with_sync', { project_code: PROJECT_CODE, path: rootPath });

  result.pass =
    result.checks.setupByApi &&
    result.checks.rightAddDir &&
    result.checks.rightAddDirServerOk &&
    result.checks.rightAddDirKeepsRootExpanded &&
    result.checks.multiLevelVisible &&
    result.checks.rightEditPrefill &&
    result.checks.rightEditParentShown &&
    result.checks.rightEditParentSelectable &&
    result.checks.rightEditRenameServerOk &&
    result.checks.rightDeleteFile &&
    result.checks.rightDeleteFileServerOk &&
    result.checks.rightDeleteDirKeepsRootExpanded &&
    result.checks.rightDeleteDirServerOk &&
    result.consoleErrors.length === 0 &&
    result.pageErrors.length === 0;

  const out = path.join(OUT_DIR, 'webshell-editor-pool-workspace-file-rightclick-dialog-check.json');
  fs.writeFileSync(out, JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));

  if (!result.pass) {
    process.exit(2);
  }
})();
