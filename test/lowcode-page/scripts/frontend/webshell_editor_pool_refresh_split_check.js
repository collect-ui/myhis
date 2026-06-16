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

async function expandTreeNode(page, nodeName) {
  const title = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(nodeName) }).first();
  await title.waitFor({ state: 'visible', timeout: 20000 });
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
    await sleep(800);
  }
}

async function clickFileTreeNode(page, fileName) {
  const node = page.locator('.workspace-source-tree .ant-tree-title', { hasText: toTreeTitle(fileName) }).first();
  await node.waitFor({ state: 'visible', timeout: 20000 });
  await node.click();
}

async function splitActiveTabHorizontally(page, fileName) {
  const tab = page.locator('.ant-tabs-tab', { hasText: toTreeTitle(fileName) }).last();
  await tab.waitFor({ state: 'visible', timeout: 20000 });
  await tab.click({ button: 'right' });
  const menu = page.locator('div[role="menu"].contexify:visible, div[role="menu"]:visible').last();
  await menu.waitFor({ state: 'visible', timeout: 8000 });
  const menuItem = menu.locator('[role="menuitem"]', { hasText: toLooseTitle('水平分割') }).first();
  await menuItem.waitFor({ state: 'visible', timeout: 8000 });
  await menuItem.click();
}

async function waitEditorRootCount(page, expected, timeout = 20000) {
  await page.waitForFunction((expectedCount) => {
    const roots = Array.from(document.querySelectorAll('[data-subgroup-id][data-viewer-id]'))
      .filter((root) => {
        const rect = root.getBoundingClientRect();
        const style = window.getComputedStyle(root);
        return rect.width > 260 && rect.height > 180 && style.display !== 'none' && style.visibility !== 'hidden';
      });
    return roots.length >= expectedCount;
  }, expected, { timeout });
}

async function getEditorSnapshotBySide(page, side) {
  return page.evaluate((sideName) => {
    const roots = Array.from(document.querySelectorAll('[data-subgroup-id][data-viewer-id]'))
      .filter((root) => {
        const rect = root.getBoundingClientRect();
        const style = window.getComputedStyle(root);
        return rect.width > 260 && rect.height > 180 && style.display !== 'none' && style.visibility !== 'hidden';
      })
      .sort((left, right) => left.getBoundingClientRect().left - right.getBoundingClientRect().left);
    const root = sideName === 'right' ? roots[roots.length - 1] : roots[0];
    if (!root) {
      return null;
    }
    const viewerId = root.getAttribute('data-viewer-id') || '';
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    let best = null;
    let bestArea = 0;
    for (const editor of editors) {
      try {
        const model = editor?.getModel?.();
        const node = editor?.getContainerDomNode?.();
        const slot = node?.closest?.('[data-slot-id][data-viewer-id]');
        if (!model || !node || !slot || slot.getAttribute('data-viewer-id') !== viewerId) {
          continue;
        }
        const rect = node.getBoundingClientRect();
        const slotStyle = window.getComputedStyle(slot);
        const nodeStyle = window.getComputedStyle(node);
        if (rect.width < 180 || rect.height < 120 || slotStyle.visibility !== 'visible' || nodeStyle.display === 'none') {
          continue;
        }
        const area = rect.width * rect.height;
        if (area > bestArea) {
          bestArea = area;
          best = { editor, model, rect };
        }
      } catch (_error) {
        // ignore
      }
    }
    if (!best) {
      return {
        viewerId,
        rootRect: root.getBoundingClientRect().toJSON?.() || {},
        value: '',
        found: false,
      };
    }
    return {
      viewerId,
      activeToken: root.getAttribute('data-active-token') || '',
      path: root.getAttribute('data-store-current-file-path') || '',
      value: String(best.editor.getValue?.() || ''),
      uri: decodeURIComponent(String(best.model?.uri || '')),
      found: true,
    };
  }, side);
}

async function setEditorValueBySide(page, side, value) {
  return page.evaluate(({ sideName, nextValue }) => {
    const roots = Array.from(document.querySelectorAll('[data-subgroup-id][data-viewer-id]'))
      .filter((root) => {
        const rect = root.getBoundingClientRect();
        const style = window.getComputedStyle(root);
        return rect.width > 260 && rect.height > 180 && style.display !== 'none' && style.visibility !== 'hidden';
      })
      .sort((left, right) => left.getBoundingClientRect().left - right.getBoundingClientRect().left);
    const root = sideName === 'right' ? roots[roots.length - 1] : roots[0];
    if (!root) {
      return false;
    }
    const viewerId = root.getAttribute('data-viewer-id') || '';
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    let best = null;
    let bestArea = 0;
    for (const editor of editors) {
      try {
        const model = editor?.getModel?.();
        const node = editor?.getContainerDomNode?.();
        const slot = node?.closest?.('[data-slot-id][data-viewer-id]');
        if (!model || !node || !slot || slot.getAttribute('data-viewer-id') !== viewerId) {
          continue;
        }
        const rect = node.getBoundingClientRect();
        const slotStyle = window.getComputedStyle(slot);
        if (rect.width < 180 || rect.height < 120 || slotStyle.visibility !== 'visible') {
          continue;
        }
        const area = rect.width * rect.height;
        if (area > bestArea) {
          bestArea = area;
          best = editor;
        }
      } catch (_error) {
        // ignore
      }
    }
    if (!best) {
      return false;
    }
    best.setValue(String(nextValue || ''));
    best.focus?.();
    return true;
  }, { sideName: side, nextValue: value });
}

async function clickToolbarButtonBySide(page, side, label) {
  return page.evaluate(({ sideName, buttonLabel }) => {
    const roots = Array.from(document.querySelectorAll('[data-subgroup-id][data-viewer-id]'))
      .filter((root) => {
        const rect = root.getBoundingClientRect();
        const style = window.getComputedStyle(root);
        return rect.width > 260 && rect.height > 180 && style.display !== 'none' && style.visibility !== 'hidden';
      })
      .sort((left, right) => left.getBoundingClientRect().left - right.getBoundingClientRect().left);
    const root = sideName === 'right' ? roots[roots.length - 1] : roots[0];
    if (!root) {
      return false;
    }
    const button = Array.from(root.querySelectorAll('button'))
      .find((item) => String(item.textContent || '').trim().includes(buttonLabel) && !item.disabled);
    if (!button) {
      return false;
    }
    button.click();
    return true;
  }, { sideName: side, buttonLabel: label });
}

async function waitEditorContains(page, side, marker, timeout = 20000) {
  await page.waitForFunction(({ sideName, expectedMarker }) => {
    const roots = Array.from(document.querySelectorAll('[data-subgroup-id][data-viewer-id]'))
      .filter((root) => {
        const rect = root.getBoundingClientRect();
        const style = window.getComputedStyle(root);
        return rect.width > 260 && rect.height > 180 && style.display !== 'none' && style.visibility !== 'hidden';
      })
      .sort((left, right) => left.getBoundingClientRect().left - right.getBoundingClientRect().left);
    const root = sideName === 'right' ? roots[roots.length - 1] : roots[0];
    if (!root) {
      return false;
    }
    const viewerId = root.getAttribute('data-viewer-id') || '';
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    for (const editor of editors) {
      const node = editor?.getContainerDomNode?.();
      const slot = node?.closest?.('[data-slot-id][data-viewer-id]');
      if (!node || !slot || slot.getAttribute('data-viewer-id') !== viewerId) {
        continue;
      }
      const rect = node.getBoundingClientRect();
      const slotStyle = window.getComputedStyle(slot);
      if (rect.width < 180 || rect.height < 120 || slotStyle.visibility !== 'visible') {
        continue;
      }
      if (String(editor.getValue?.() || '').includes(expectedMarker)) {
        return true;
      }
    }
    return false;
  }, { sideName: side, expectedMarker: marker }, { timeout });
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const ts = Date.now();
  const rootName = 'test';
  const rootPath = path.posix.join(PROJECT_DIR, rootName);
  const fileName = `refresh_case_${ts}.md`;
  const filePath = path.posix.join(rootPath, fileName);
  const initialContent = `# refresh split check\ninitial ${ts}\n`;
  const screenshot = path.join(OUT_DIR, 'webshell-editor-pool-refresh-split-check.png');
  const reportPath = path.join(OUT_DIR, 'webshell-editor-pool-refresh-split-check.json');

  const result = {
    pageUrl: PAGE_URL,
    projectCode: PROJECT_CODE,
    fileName,
    filePath,
    checks: {
      filePrepared: false,
      fileOpened: false,
      splitCreated: false,
      repeatedRefresh: false,
    },
    iterations: [],
    trace: {
      contentRequestPaths: [],
      saveRequestPaths: [],
      rootSnapshots: [],
    },
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    screenshot,
    pass: false,
  };

  runCurlAllowFail('webshell.workspace_file_add_with_sync', {
    project_code: PROJECT_CODE,
    name: rootName,
    path: rootPath,
    is_dir: '1',
    parent_id: '',
  });
  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: filePath,
  });
  runCurl('webshell.workspace_file_add_with_sync', {
    project_code: PROJECT_CODE,
    name: fileName,
    path: filePath,
    is_dir: '0',
    parent_id: '',
  });
  runCurl('webshell.workspace_file_save', {
    project_code: PROJECT_CODE,
    path: filePath,
    content: initialContent,
  });
  result.checks.filePrepared = true;

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1720, height: 980 } });
  page.setDefaultTimeout(20000);

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
    if (!url.includes('/template_data/data?service=webshell.workspace_file_')) {
      return;
    }
    const body = String(req.postData() || '');
    const match = body.match(/"path"\s*:\s*"([^"]+)"/);
    const requestPath = match && match[1] ? match[1] : '';
    if (url.includes('workspace_file_content')) {
      result.trace.contentRequestPaths.push(requestPath);
    }
    if (url.includes('workspace_file_save')) {
      result.trace.saveRequestPaths.push(requestPath);
    }
  });

  try {
    await page.goto(PAGE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await sleep(5000);

    const projectBtn = page.getByRole('button', { name: PROJECT_CODE }).first();
    await projectBtn.waitFor({ state: 'visible', timeout: 20000 });
    await projectBtn.click();
    await sleep(1200);

    await expandTreeNode(page, rootName);
    await clickFileTreeNode(page, fileName);
    await waitEditorRootCount(page, 1);
    await waitEditorContains(page, 'left', 'initial', 20000);
    result.checks.fileOpened = true;

    await splitActiveTabHorizontally(page, fileName);
    await sleep(1500);
    await waitEditorRootCount(page, 2);
    await waitEditorContains(page, 'right', 'initial', 20000);
    result.checks.splitCreated = true;

    for (let index = 1; index <= 3; index += 1) {
      const marker = `refresh-marker-${ts}-${index}`;
      const content = `# refresh split check\n${marker}\nline ${index}\n`;
      const setOk = await setEditorValueBySide(page, 'left', content);
      const saveOk = await clickToolbarButtonBySide(page, 'left', '保存');
      await sleep(1000);
      const serverContent = String(runCurl('webshell.workspace_file_content', {
        project_code: PROJECT_CODE,
        path: filePath,
      })?.data?.content_text || '');
      const rightBefore = await getEditorSnapshotBySide(page, 'right');
      const refreshOk = await clickToolbarButtonBySide(page, 'right', '刷新');
      await waitEditorContains(page, 'right', marker, 20000);
      const rightAfter = await getEditorSnapshotBySide(page, 'right');

      result.iterations.push({
        index,
        marker,
        setOk,
        saveOk,
        refreshOk,
        serverHasMarker: serverContent.includes(marker),
        rightBeforeHasMarker: String(rightBefore?.value || '').includes(marker),
        rightAfterHasMarker: String(rightAfter?.value || '').includes(marker),
        rightToken: rightAfter?.activeToken || '',
        rightPath: rightAfter?.path || '',
      });
    }

    result.trace.rootSnapshots = await page.evaluate(() => Array.from(document.querySelectorAll('[data-subgroup-id][data-viewer-id]')).map((root) => {
      const rect = root.getBoundingClientRect();
      return {
        viewerId: root.getAttribute('data-viewer-id') || '',
        subgroupId: root.getAttribute('data-subgroup-id') || '',
        activeToken: root.getAttribute('data-active-token') || '',
        currentFilePath: root.getAttribute('data-store-current-file-path') || '',
        rect: { left: rect.left, top: rect.top, width: rect.width, height: rect.height },
      };
    }));
    await page.screenshot({ path: screenshot, fullPage: true });
  } catch (error) {
    result.error = String(error && error.stack ? error.stack : error);
  } finally {
    await browser.close();
  }

  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: filePath,
  });

  result.checks.repeatedRefresh =
    result.iterations.length === 3 &&
    result.iterations.every((item) => item.setOk && item.saveOk && item.refreshOk && item.serverHasMarker && item.rightAfterHasMarker);

  result.pass =
    result.checks.filePrepared &&
    result.checks.fileOpened &&
    result.checks.splitCreated &&
    result.checks.repeatedRefresh &&
    result.consoleErrors.length === 0 &&
    result.pageErrors.length === 0;

  fs.writeFileSync(reportPath, `${JSON.stringify(result, null, 2)}\n`, 'utf8');
  console.log(JSON.stringify({
    pass: result.pass,
    reportPath,
    screenshot,
    checks: result.checks,
    iterations: result.iterations,
    consoleErrors: result.consoleErrors.length,
    pageErrors: result.pageErrors.length,
    failedRequests: result.failedRequests.length,
  }, null, 2));

  process.exit(result.pass ? 0 : 2);
})();
