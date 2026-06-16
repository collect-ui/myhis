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

function addWorkspacePath(targetPath, parentId, isDir = '1') {
  runCurl('webshell.workspace_file_add_with_sync', {
    project_code: PROJECT_CODE,
    name: path.posix.basename(targetPath),
    path: targetPath,
    parent_id: parentId || '',
    is_dir: isDir,
  });
  return getRecordByPath(targetPath);
}

function getRecordByPath(targetPath) {
  const resp = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    path_exact: targetPath,
    pagination: false,
  });
  const list = Array.isArray(resp.data) ? resp.data : [];
  if (!list[0]) {
    throw new Error(`record not found: ${targetPath}`);
  }
  return list[0];
}

function listChildren(parentId) {
  const resp = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    parent_id: parentId || '',
    root_only: !parentId,
    pagination: false,
  });
  return Array.isArray(resp.data) ? resp.data : [];
}

function namesOf(rows) {
  return rows.map((item) => String(item.name || item.title || ''));
}

function assertOrdered(children, expectedNames, label) {
  const actual = namesOf(children);
  let cursor = -1;
  for (const name of expectedNames) {
    const index = actual.indexOf(name);
    if (index < 0) {
      throw new Error(`${label}: missing ${name}; actual=${actual.join(',')}`);
    }
    if (index <= cursor) {
      throw new Error(`${label}: ${name} not after previous item; actual=${actual.join(',')}`);
    }
    cursor = index;
  }
}

function toTreeTitle(name) {
  return new RegExp(`^${name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`);
}

function xpathLiteral(value) {
  if (!String(value).includes("'")) {
    return `'${value}'`;
  }
  if (!String(value).includes('"')) {
    return `"${value}"`;
  }
  return `concat('${String(value).split("'").join("',\"'\",'")}')`;
}

function treeTitleLocator(page, nodeName) {
  return page.locator(`xpath=//div[contains(@class,"workspace-source-tree")]//span[contains(@class,"ant-tree-title") and normalize-space(.)=${xpathLiteral(nodeName)}]`).first();
}

function treeRowLocator(page, nodeName) {
  return page.locator(`xpath=//div[contains(@class,"workspace-source-tree")]//span[contains(@class,"ant-tree-title") and normalize-space(.)=${xpathLiteral(nodeName)}]/ancestor::div[contains(@class,"ant-tree-treenode")][1]`).first();
}

async function ensureProjectButton(page) {
  const projectBtn = page.getByRole('button', { name: PROJECT_CODE }).first();
  await projectBtn.waitFor({ state: 'visible', timeout: 25000 });
  await projectBtn.click();
  await sleep(1200);
}

async function waitTreeHasNode(page, nodeName, timeoutMs = 25000) {
  const node = treeTitleLocator(page, nodeName);
  await node.waitFor({ state: 'visible', timeout: timeoutMs });
}

async function expandTreeNode(page, nodeName) {
  const title = treeTitleLocator(page, nodeName);
  await title.waitFor({ state: 'visible', timeout: 25000 });
  const switcher = title.locator('xpath=ancestor::div[contains(@class,"ant-tree-treenode")][1]').locator('.ant-tree-switcher').first();
  if (!(await switcher.count())) {
    return;
  }
  const cls = String((await switcher.getAttribute('class')) || '');
  if (cls.includes('ant-tree-switcher_close')) {
    await switcher.click();
    await sleep(900);
  }
}

async function dragTreeNode(page, sourceName, targetName, mode) {
  const sourceTitle = treeTitleLocator(page, sourceName);
  const targetTitle = treeTitleLocator(page, targetName);
  await sourceTitle.waitFor({ state: 'visible', timeout: 25000 });
  await targetTitle.waitFor({ state: 'visible', timeout: 25000 });
  const source = treeRowLocator(page, sourceName);
  const target = treeRowLocator(page, targetName);
  const sourceBox = await source.boundingBox();
  const box = await target.boundingBox();
  const targetTitleBox = await targetTitle.boundingBox();
  const resolvedTargetName = String(await target.locator('.ant-tree-title').first().textContent() || '').trim();
  if (resolvedTargetName !== targetName) {
    throw new Error(`resolved target mismatch: expected=${targetName}, actual=${resolvedTargetName}`);
  }
  if (!sourceBox) {
    throw new Error(`source box missing: ${sourceName}`);
  }
  if (!box) {
    throw new Error(`target box missing: ${targetName}`);
  }
  const y = mode === 'before' ? 1 : mode === 'after' ? Math.max(1, box.height - 1) : Math.max(2, box.height / 2);
  const responsePromise = page.waitForResponse((resp) => resp.url().includes('workspace_file_move_with_sync'), { timeout: 45000 }).catch(() => null);
  const reloadPromise = page.waitForResponse((resp) => resp.url().includes('workspace_file_query'), { timeout: 45000 }).catch(() => null);
  const sourceX = sourceBox.x + Math.max(2, Math.min(sourceBox.width - 2, sourceBox.width / 2));
  const sourceY = sourceBox.y + Math.max(2, sourceBox.height / 2);
  await sourceTitle.dragTo(target, {
    force: true,
    sourcePosition: {
      x: Math.max(2, Math.min(sourceBox.width - 2, sourceX - sourceBox.x)),
      y: Math.max(2, sourceY - sourceBox.y),
    },
    targetPosition: {
      x: Math.max(2, Math.min(box.width - 2, targetTitleBox ? (targetTitleBox.x + targetTitleBox.width / 2 - box.x) : 96)),
      y,
    },
  });
  const response = await responsePromise;
  if (!response) {
    throw new Error(`move response missing after ${sourceName} -> ${targetName} (${mode})`);
  }
  const body = await response.json().catch(() => null);
  if (!body || String(body.code || '') !== '0' || body.success === false) {
    throw new Error(`move failed after ${sourceName} -> ${targetName} (${mode}): ${body?.msg || response.status()}`);
  }
  await reloadPromise;
  await sleep(1200);
}

async function reloadWorkspace(page, rootName) {
  await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
  await ensureProjectButton(page);
  await page.locator('.workspace-source-tree').first().waitFor({ state: 'visible', timeout: 25000 });
  await waitTreeHasNode(page, rootName);
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const ts = Date.now();
  const rootName = `drag_order_${ts}`;
  const aDirName = `a_dir_${ts}`;
  const bDirName = `b_dir_${ts}`;
  const alphaName = `alpha_${ts}.txt`;
  const betaName = `beta_${ts}.txt`;
  const gammaName = `gamma_${ts}.txt`;
  const rootMoveName = `root_move_${ts}.txt`;

  const rootPath = path.posix.join(PROJECT_DIR, rootName);
  const aDirPath = path.posix.join(rootPath, aDirName);
  const bDirPath = path.posix.join(rootPath, bDirName);
  const alphaPath = path.posix.join(rootPath, alphaName);
  const betaPath = path.posix.join(rootPath, betaName);
  const gammaPath = path.posix.join(rootPath, gammaName);
  const rootMoveInsidePath = path.posix.join(rootPath, rootMoveName);
  const rootMoveRootPath = path.posix.join(PROJECT_DIR, rootMoveName);

  const result = {
    pageUrl: PAGE_URL,
    projectCode: PROJECT_CODE,
    fixture: { rootPath, aDirPath, bDirPath, alphaPath, betaPath, gammaPath, rootMoveInsidePath, rootMoveRootPath },
    checks: {},
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    moveRequests: [],
    screenshots: {},
    pass: false,
  };

  runCurlAllowFail('webshell.workspace_file_delete_with_sync', { project_code: PROJECT_CODE, path: rootPath });
  runCurlAllowFail('webshell.workspace_file_delete_with_sync', { project_code: PROJECT_CODE, path: rootMoveRootPath });

  const root = addWorkspacePath(rootPath, '', '1');
  addWorkspacePath(aDirPath, root.file_id, '1');
  const bDir = addWorkspacePath(bDirPath, root.file_id, '1');
  addWorkspacePath(alphaPath, root.file_id, '0');
  addWorkspacePath(betaPath, root.file_id, '0');
  addWorkspacePath(gammaPath, root.file_id, '0');
  addWorkspacePath(rootMoveInsidePath, root.file_id, '0');

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
    if (!req.url().includes('workspace_file_move_with_sync')) {
      return;
    }
    let data = null;
    try {
      data = JSON.parse(req.postData() || '{}');
    } catch (error) {
      data = req.postData();
    }
    result.moveRequests.push(data);
  });

  try {
    await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
    await ensureProjectButton(page);
    await page.locator('.workspace-source-tree').first().waitFor({ state: 'visible', timeout: 25000 });
    await waitTreeHasNode(page, rootName);
    await expandTreeNode(page, rootName);
    await waitTreeHasNode(page, alphaName);
    await waitTreeHasNode(page, betaName);
    result.checks.initialTreeVisible = true;

    await dragTreeNode(page, betaName, aDirName, 'before');
    let rootChildren = listChildren(root.file_id);
    assertOrdered(rootChildren, [betaName, aDirName], 'before-first order');
    result.checks.dragBeforeFirst = true;

    await reloadWorkspace(page, rootName);
    await expandTreeNode(page, rootName);
    await dragTreeNode(page, alphaName, rootMoveName, 'after');
    rootChildren = listChildren(root.file_id);
    assertOrdered(rootChildren, [rootMoveName, alphaName], 'after-last order');
    result.checks.dragAfterLast = true;

    await reloadWorkspace(page, rootName);
    await expandTreeNode(page, rootName);
    await dragTreeNode(page, gammaName, bDirName, 'inside');
    const gammaInDir = getRecordByPath(path.posix.join(bDirPath, gammaName));
    if (String(gammaInDir.parent_id || '') !== String(bDir.file_id)) {
      throw new Error(`gamma parent mismatch after inside move: ${gammaInDir.parent_id}`);
    }
    result.checks.dragInsideFolder = true;

    await reloadWorkspace(page, rootName);
    await expandTreeNode(page, rootName);
    await expandTreeNode(page, bDirName);
    await dragTreeNode(page, gammaName, alphaName, 'before');
    const gammaBack = getRecordByPath(gammaPath);
    if (String(gammaBack.parent_id || '') !== String(root.file_id)) {
      throw new Error(`gamma parent mismatch after moving outside: ${gammaBack.parent_id}`);
    }
    rootChildren = listChildren(root.file_id);
    assertOrdered(rootChildren, [gammaName, alphaName], 'outside order');
    result.checks.dragOutsideFolder = true;

    await reloadWorkspace(page, rootName);
    await expandTreeNode(page, rootName);
    await dragTreeNode(page, rootMoveName, rootName, 'before');
    const rootMoved = getRecordByPath(rootMoveRootPath);
    if (String(rootMoved.parent_id || '') !== '') {
      throw new Error(`root move parent mismatch: ${rootMoved.parent_id}`);
    }
    result.checks.dragToWorkspaceRoot = true;

    await reloadWorkspace(page, rootName);
    await dragTreeNode(page, rootMoveName, rootName, 'inside');
    const rootMovedBack = getRecordByPath(rootMoveInsidePath);
    if (String(rootMovedBack.parent_id || '') !== String(root.file_id)) {
      throw new Error(`root move back parent mismatch: ${rootMovedBack.parent_id}`);
    }
    result.checks.dragRootBackInside = true;

    result.screenshots.final = path.join(OUT_DIR, 'webshell-editor-pool-workspace-file-drag-order-check.png');
    await page.screenshot({ path: result.screenshots.final, fullPage: true });
    result.pass = Object.values(result.checks).every(Boolean) &&
      result.consoleErrors.length === 0 &&
      result.pageErrors.length === 0 &&
      result.failedRequests.length === 0;
    if (!result.pass) {
      throw new Error(`drag check failed: ${JSON.stringify({ checks: result.checks, consoleErrors: result.consoleErrors, pageErrors: result.pageErrors, failedRequests: result.failedRequests }, null, 2)}`);
    }
  } catch (error) {
    result.error = String(error && error.stack ? error.stack : error);
    result.screenshots.failed = path.join(OUT_DIR, 'webshell-editor-pool-workspace-file-drag-order-check.fail.png');
    await page.screenshot({ path: result.screenshots.failed, fullPage: true }).catch(() => undefined);
    throw error;
  } finally {
    await browser.close();
    runCurlAllowFail('webshell.workspace_file_delete_with_sync', { project_code: PROJECT_CODE, path: rootPath });
    runCurlAllowFail('webshell.workspace_file_delete_with_sync', { project_code: PROJECT_CODE, path: rootMoveRootPath });
    const out = path.join(OUT_DIR, 'webshell-editor-pool-workspace-file-drag-order-check.json');
    fs.writeFileSync(out, JSON.stringify(result, null, 2));
    console.log(JSON.stringify(result, null, 2));
  }
})().catch((error) => {
  console.error(error && error.stack ? error.stack : error);
  process.exit(1);
});
