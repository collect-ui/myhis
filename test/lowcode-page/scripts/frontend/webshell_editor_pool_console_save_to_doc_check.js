#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
const { chromium } = require('playwright');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://192.168.232.130:8015/template_data/data';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || 'backend';

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)); }

function runCurl(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  const res = spawnSync('curl', [
    '-s',
    `${API_URL}?service=${service}`,
    '-H',
    'Content-Type: application/json',
    '--data',
    payload,
  ], { encoding: 'utf8' });
  if (res.status !== 0) throw new Error(res.stderr || `curl failed: ${service}`);
  const out = JSON.parse(String(res.stdout || '{}'));
  if (!out || String(out.code || '') !== '0' || out.success === false) {
    throw new Error(`${service} failed: ${out?.msg || 'unknown error'}`);
  }
  return out;
}

function normalizeId(raw) {
  const value = String(raw || '').trim();
  if (!value) return '';
  if (value.startsWith('g_') || value.startsWith('d_')) return value.slice(2);
  return value;
}

function isDir(node) {
  if (!node) return false;
  if (String(node.node_type || '').toLowerCase() === 'service') return true;
  return node.is_dir === '1' || node.is_dir === 1 || node.is_dir === true;
}

function findGroupId(tree, targetTitle) {
  let hit = '';
  const walk = (nodes) => {
    if (hit || !Array.isArray(nodes)) return;
    for (const node of nodes) {
      if (!node || !isDir(node)) continue;
      const title = String(node.title || node.display_title || node.name || '').trim();
      if (title === targetTitle) {
        hit = normalizeId(node.id || node.doc_group_id || '');
        return;
      }
      walk(Array.isArray(node.children) ? node.children : []);
    }
  };
  walk(Array.isArray(tree) ? tree : []);
  return hit;
}

function findDocId(tree, targetTitle) {
  let hit = '';
  const walk = (nodes) => {
    if (hit || !Array.isArray(nodes)) return;
    for (const node of nodes) {
      if (!node) continue;
      if (isDir(node)) {
        walk(Array.isArray(node.children) ? node.children : []);
      } else {
        const title = String(node.title || node.display_title || node.name || '').trim();
        if (title === targetTitle) {
          hit = normalizeId(node.collect_doc_id || node.id || '');
          return;
        }
      }
    }
  };
  walk(Array.isArray(tree) ? tree : []);
  return hit;
}

async function setMonacoValue(page, marker, value) {
  return page.evaluate(({ marker, value }) => {
    const models = window?.monaco?.editor?.getModels?.() || [];
    for (const model of models) {
      const raw = String(model?.uri || '');
      const decoded = decodeURIComponent(raw);
      if (raw.includes(marker) || decoded.includes(marker)) {
        model.setValue(String(value || ''));
        return true;
      }
    }
    return false;
  }, { marker, value });
}

async function selectConsoleOption(page, index, label) {
  const root = page.locator('[id$="workspace-http-console-form"]').first();
  const select = root.locator('.ant-select').nth(index);
  await select.click();
  await page.waitForTimeout(120);
  const option = page.locator('.ant-select-dropdown:visible .ant-select-item-option', { hasText: label }).first();
  await option.click();
  await page.waitForTimeout(120);
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const now = Date.now();
  const caseName = `auto_console_save_${now}`;
  const saveHeadersObj = { 'X-Debug-Token': 'token-123', Authorization: 'Bearer demo' };
  const saveBodyObj = { service: 'config.doc_group_http_service_tree_query', to_tree: true };
  const saveMethod = 'get';
  const saveMode = 'backend';
  const saveUrl = `https://postman-echo.com/get?source=console-save-all-${now}`;

  const summary = {
    pageUrl: PAGE_URL,
    projectCode: PROJECT_CODE,
    caseName,
    docId: '',
    groupId: '',
    ui: {
      saveToast: '',
      detailHeaderCountShown: false,
      detailHeaderContentShown: false,
      detailHeadersCollapseWorks: false,
      metaNoHeaderCard: false,
      metaSingleRow: false,
    },
    db: {
      project_code: '',
      request_headers: '',
      code: '',
      request_mode: '',
      request_method: '',
      request_url: '',
      headersSaved: false,
      bodySaved: false,
    },
    cleanupDeleted: false,
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    screenshot: '',
    pass: false,
  };

  const treeBefore = runCurl('config.doc_group_http_service_tree_query', { project_code: PROJECT_CODE, to_tree: true });
  const groupId = findGroupId(treeBefore.data || [], 'test2');
  if (!groupId) throw new Error('cannot find group test2');
  summary.groupId = groupId;

  runCurl('config.doc_save', {
    doc: {
      title: caseName,
      sub_title: 'console save case',
      type: '2',
      project_code: PROJECT_CODE,
      parent_dir: groupId,
      order_index: 1,
      request_mode: 'frontend',
      request_method: 'post',
      request_url: '/template_data/data',
      request_headers: '{}',
      code: '{}',
      code_desc: '',
      code_result: '',
    },
    important_list: [],
    params: [],
    result: [],
    demo: [],
  });

  const treeAfterCreate = runCurl('config.doc_group_http_service_tree_query', { project_code: PROJECT_CODE, to_tree: true });
  summary.docId = findDocId(treeAfterCreate.data || [], caseName);
  if (!summary.docId) throw new Error('created doc not found in tree');

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1680, height: 980 } });

  page.on('console', (msg) => {
    if (msg.type() === 'error') summary.consoleErrors.push(msg.text());
  });
  page.on('pageerror', (err) => summary.pageErrors.push(String(err)));
  page.on('requestfailed', (req) => {
    summary.failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || 'failed'}`);
  });

  try {
    await page.goto(PAGE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await sleep(6000);

    const tree = page.locator('.workspace-http-tree').first();
    const groupNode = tree.locator('.ant-tree-treenode', { has: page.locator('.ant-tree-title', { hasText: 'test2' }) }).first();
    const switcher = groupNode.locator('.ant-tree-switcher').first();
    const switcherClass = String((await switcher.getAttribute('class').catch(() => '')) || '');
    if (switcherClass.includes('ant-tree-switcher_close')) {
      await switcher.click().catch(() => undefined);
      await sleep(900);
    }

    await tree.locator('.ant-tree-title', { hasText: caseName }).first().click();
    await sleep(1500);

    await page.getByRole('button', { name: '新开控制台' }).first().click();
    await sleep(1000);

    const headerToggles = page.locator('.workspace-http-console-collapse .ant-collapse-header');
    const headerToggleCount = await headerToggles.count().catch(() => 0);
    for (let i = 0; i < headerToggleCount; i += 1) {
      const toggle = headerToggles.nth(i);
      const text = String((await toggle.innerText().catch(() => '')) || '');
      if (!/请求头\s*Headers/.test(text)) continue;
      if (!(await toggle.isVisible().catch(() => false))) continue;
      await toggle.click().catch(() => undefined);
      await sleep(300);
      break;
    }

    let headerSetOk = await setMonacoValue(page, 'workspace-http-console-headers-', JSON.stringify(saveHeadersObj, null, 2));
    if (!headerSetOk) {
      for (let i = 0; i < headerToggleCount; i += 1) {
        const toggle = headerToggles.nth(i);
        if (!(await toggle.isVisible().catch(() => false))) continue;
        await toggle.click().catch(() => undefined);
        await sleep(280);
      }
      headerSetOk = await setMonacoValue(page, 'workspace-http-console-headers-', JSON.stringify(saveHeadersObj, null, 2));
    }
    const bodySetOk = await setMonacoValue(page, 'workspace-http-console-body-', JSON.stringify(saveBodyObj, null, 2));
    summary.ui.headerEditorSet = headerSetOk;
    summary.ui.bodyEditorSet = bodySetOk;

    await selectConsoleOption(page, 0, 'GET');
    await selectConsoleOption(page, 1, '后端代发');
    await page.locator('[id$="workspace-http-console-form"] input[placeholder*="输入请求URL"]').first().fill(saveUrl);

    await sleep(300);

    await page.getByRole('button', { name: /保\s*存/ }).first().click();
    await sleep(1500);

    summary.ui.saveToast = String((await page.locator('.ant-message').innerText().catch(() => '')) || '').trim();

    await tree.locator('.ant-tree-title', { hasText: caseName }).first().click();
    await sleep(1500);
    const bodyText = await page.evaluate(() => String(document.body?.innerText || ''));
    summary.ui.detailHeaderCountShown = /请求头\s*Headers/.test(bodyText);

    const detailCollapse = page.locator('.workspace-http-doc .workspace-http-console-collapse').first();
    if (await detailCollapse.isVisible().catch(() => false)) {
      const beforeOpen = await detailCollapse.locator('.ant-collapse-content-active').count().catch(() => 0);
      await detailCollapse.locator('.ant-collapse-header').first().click().catch(() => undefined);
      await sleep(300);
      const afterOpen = await detailCollapse.locator('.ant-collapse-content-active').count().catch(() => 0);
      summary.ui.detailHeadersCollapseWorks = beforeOpen !== afterOpen;
      if (afterOpen < 1) {
        await detailCollapse.locator('.ant-collapse-header').first().click().catch(() => undefined);
        await sleep(260);
      }
    }

    summary.ui.detailHeaderContentShown = await page.evaluate(() => {
      const models = window?.monaco?.editor?.getModels?.() || [];
      for (const model of models) {
        const raw = String(model?.uri || '');
        const decoded = decodeURIComponent(raw);
        if (raw.includes('workspace-http-headers-') || decoded.includes('workspace-http-headers-')) {
          const value = String(model.getValue() || '');
          return /X-Debug-Token/.test(value) && /Authorization/.test(value);
        }
      }
      return false;
    });

    const metaLayout = await page.evaluate(() => {
      const cards = Array.from(document.querySelectorAll('.workspace-http-doc .workspace-http-meta-card'));
      const labels = cards.map((card) => String(card.querySelector('.workspace-http-meta-label')?.textContent || '').trim());
      const tops = cards.map((card) => Math.round(card.getBoundingClientRect().top));
      const rowCount = Array.from(new Set(tops)).length;
      return {
        count: cards.length,
        labels,
        rowCount,
      };
    });
    summary.ui.metaNoHeaderCard = !metaLayout.labels.includes('请求头');
    summary.ui.metaSingleRow = metaLayout.count <= 4 && metaLayout.rowCount <= 1;

    const detailResp = runCurl('config.doc_detail', { collect_doc_id: summary.docId, project_code: PROJECT_CODE });
    const doc = detailResp.data?.doc || {};
    summary.db.request_headers = String(doc.request_headers || '');
    summary.db.code = String(doc.code || '');
    summary.db.project_code = String(doc.project_code || '');
    summary.db.request_mode = String(doc.request_mode || '');
    summary.db.request_method = String(doc.request_method || '');
    summary.db.request_url = String(doc.request_url || '');

    const headersObj = (() => {
      try { return JSON.parse(summary.db.request_headers || '{}'); } catch (_error) { return {}; }
    })();
    const codeObj = (() => {
      try { return JSON.parse(summary.db.code || '{}'); } catch (_error) { return {}; }
    })();

    summary.db.headersSaved =
      headersObj['X-Debug-Token'] === saveHeadersObj['X-Debug-Token'] &&
      headersObj.Authorization === saveHeadersObj.Authorization;

    summary.db.bodySaved =
      codeObj.service === saveBodyObj.service &&
      codeObj.to_tree === saveBodyObj.to_tree &&
      !Object.prototype.hasOwnProperty.call(codeObj, 'method') &&
      !Object.prototype.hasOwnProperty.call(codeObj, 'url') &&
      !Object.prototype.hasOwnProperty.call(codeObj, '_request_mode') &&
      !Object.prototype.hasOwnProperty.call(codeObj, '_request_headers');

    const shot = path.join(OUT_DIR, 'webshell-editor-pool-console-save-to-doc-check.png');
    await page.screenshot({ path: shot, fullPage: true });
    summary.screenshot = shot;
  } finally {
    await browser.close();
  }

  try {
    runCurl('config.doc_delete', { project_code: PROJECT_CODE, collect_doc_id_list: [summary.docId] });
    summary.cleanupDeleted = true;
  } catch (_error) {
    summary.cleanupDeleted = false;
  }

  const nonCanceledErrors = summary.pageErrors.filter((item) => !/Canceled/i.test(String(item || '')));
  summary.nonCanceledPageErrors = nonCanceledErrors;

  summary.pass =
    summary.db.headersSaved &&
    summary.db.bodySaved &&
    summary.ui.detailHeaderCountShown &&
    summary.ui.detailHeaderContentShown &&
    summary.ui.detailHeadersCollapseWorks &&
    summary.ui.metaNoHeaderCard &&
    summary.ui.metaSingleRow &&
    summary.db.request_mode === saveMode &&
    summary.db.request_method === saveMethod &&
    summary.db.request_url === saveUrl &&
    summary.db.project_code === PROJECT_CODE &&
    summary.cleanupDeleted &&
    nonCanceledErrors.length === 0;

  const outJson = path.join(OUT_DIR, 'webshell-editor-pool-console-save-to-doc-check.json');
  fs.writeFileSync(outJson, JSON.stringify(summary, null, 2));
  console.log(JSON.stringify(summary, null, 2));

  if (!summary.pass) process.exitCode = 2;
})();
