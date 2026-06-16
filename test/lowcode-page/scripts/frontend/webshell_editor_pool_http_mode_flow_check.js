#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
const { chromium } = require('playwright');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://192.168.232.130:8015/template_data/data';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || 'backend';

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

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

function normalizeId(raw) {
  const value = String(raw || '').trim();
  if (!value) return '';
  if (value.startsWith('g_') || value.startsWith('d_')) return value.slice(2);
  return value;
}

function isDir(node) {
  if (!node) return false;
  if (String(node.type || '').toLowerCase() === 'dir') return true;
  if (String(node.node_type || '').toLowerCase() === 'service') return true;
  return node.is_dir === '1' || node.is_dir === 1 || node.is_dir === true;
}

function findGroupIdByTitle(tree, targetTitle) {
  let hit = '';
  const walk = (nodes) => {
    if (hit || !Array.isArray(nodes)) return;
    for (const node of nodes) {
      if (!node) continue;
      if (!isDir(node)) continue;
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

function findDocIdByTitle(tree, targetTitle) {
  let hit = '';
  const walk = (nodes) => {
    if (hit || !Array.isArray(nodes)) return;
    for (const node of nodes) {
      if (!node) continue;
      if (isDir(node)) {
        walk(Array.isArray(node.children) ? node.children : []);
        continue;
      }
      const title = String(node.title || node.display_title || node.name || '').trim();
      if (title === targetTitle) {
        hit = normalizeId(node.collect_doc_id || node.id || '');
        return;
      }
    }
  };
  walk(Array.isArray(tree) ? tree : []);
  return hit;
}

async function getMonacoValue(page, marker) {
  return page.evaluate(({ marker }) => {
    const models = window?.monaco?.editor?.getModels?.() || [];
    for (const model of models) {
      const raw = String(model?.uri || '');
      const decoded = decodeURIComponent(raw);
      if (raw.includes(marker) || decoded.includes(marker)) {
        return String(model.getValue() || '');
      }
    }
    return '';
  }, { marker });
}

async function waitForBodyText(page, pattern, timeoutMs) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    const text = await page.evaluate(() => String(document.body?.innerText || ''));
    if (pattern.test(text)) {
      return { ok: true, text };
    }
    await sleep(300);
  }
  const finalText = await page.evaluate(() => String(document.body?.innerText || ''));
  return { ok: false, text: finalText };
}

async function selectHttpDoc(page, groupTitle, docTitle) {
  const tree = page.locator('.workspace-http-tree').first();
  await tree.waitFor({ state: 'visible', timeout: 20000 });

  const groupNode = tree.locator('.ant-tree-treenode', { has: page.locator('.ant-tree-title', { hasText: groupTitle }) }).first();
  const switcher = groupNode.locator('.ant-tree-switcher').first();
  const switcherClass = String((await switcher.getAttribute('class').catch(() => '')) || '');
  if (switcherClass.includes('ant-tree-switcher_close')) {
    await switcher.click().catch(() => undefined);
    await sleep(900);
  }

  const docNodeTitle = tree.locator('.ant-tree-title', { hasText: docTitle }).first();
  await docNodeTitle.waitFor({ state: 'visible', timeout: 20000 });
  await docNodeTitle.click();
  await sleep(1500);
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const now = Date.now();
  const frontendName = `auto_frontend_mode_${now}`;
  const backendName = `auto_backend_mode_${now}`;

  const summary = {
    pageUrl: PAGE_URL,
    apiUrl: API_URL,
    projectCode: PROJECT_CODE,
    created: {
      groupTitle: 'test2',
      groupId: '',
      frontendDocId: '',
      backendDocId: '',
      frontendName,
      backendName,
    },
    displayChecks: {
      frontend: { modeShown: false, methodShown: false, urlShown: false },
      backend: { modeShown: false, methodShown: false, urlShown: false },
    },
    requestChecks: {
      frontendSend: { ok: false, statusKeyword: '', responsePreview: '' },
      backendSend: { ok: false, statusKeyword: '', responsePreview: '' },
    },
    addPanelDefaults: {
      ok: false,
      requestUrl: '',
      modeText: '',
      methodText: '',
    },
    dbFieldChecks: {
      frontend: { ok: false, request_mode: '', request_method: '', request_url: '', code: '' },
      backend: { ok: false, request_mode: '', request_method: '', request_url: '', code: '' },
    },
    cleanup: {
      frontendDeleted: false,
      backendDeleted: false,
    },
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    screenshot: '',
    pass: false,
  };

  const beforeTree = runCurl('config.doc_group_http_service_tree_query', { project_code: PROJECT_CODE, to_tree: true });
  const groupId = findGroupIdByTitle(beforeTree.data || [], 'test2');
  if (!groupId) {
    throw new Error('cannot find HTTP group: test2');
  }
  summary.created.groupId = groupId;

  runCurl('config.doc_save', {
    doc: {
      title: frontendName,
      sub_title: 'frontend mode case',
      type: '2',
      project_code: PROJECT_CODE,
      parent_dir: groupId,
      order_index: 1,
      request_mode: 'frontend',
      request_method: 'post',
      request_url: '/template_data/data',
      request_headers: '{}',
      code: JSON.stringify({ service: 'config.doc_group_http_service_tree_query', to_tree: true }, null, 2),
      code_desc: 'frontend direct case',
      code_result: '',
    },
    important_list: [],
    params: [],
    result: [],
    demo: [],
  });

  runCurl('config.doc_save', {
    doc: {
      title: backendName,
      sub_title: 'backend mode case',
      type: '2',
      project_code: PROJECT_CODE,
      parent_dir: groupId,
      order_index: 1,
      request_mode: 'backend',
      request_method: 'get',
      request_url: 'https://postman-echo.com/get?source=webshell-mode-check',
      request_headers: '{}',
      code: '{}',
      code_desc: 'backend proxy case',
      code_result: '',
    },
    important_list: [],
    params: [],
    result: [],
    demo: [],
  });

  const treeAfterCreate = runCurl('config.doc_group_http_service_tree_query', { project_code: PROJECT_CODE, to_tree: true });
  summary.created.frontendDocId = findDocIdByTitle(treeAfterCreate.data || [], frontendName);
  summary.created.backendDocId = findDocIdByTitle(treeAfterCreate.data || [], backendName);

  if (!summary.created.frontendDocId || !summary.created.backendDocId) {
    throw new Error('failed to locate created doc ids in tree');
  }

  const frontendDetail = runCurl('config.doc_detail', { collect_doc_id: summary.created.frontendDocId, project_code: PROJECT_CODE });
  const backendDetail = runCurl('config.doc_detail', { collect_doc_id: summary.created.backendDocId, project_code: PROJECT_CODE });

  const frontendDoc = frontendDetail.data?.doc || {};
  const backendDoc = backendDetail.data?.doc || {};

  summary.dbFieldChecks.frontend.request_mode = String(frontendDoc.request_mode || '');
  summary.dbFieldChecks.frontend.request_method = String(frontendDoc.request_method || '');
  summary.dbFieldChecks.frontend.request_url = String(frontendDoc.request_url || '');
  summary.dbFieldChecks.frontend.code = String(frontendDoc.code || '');
  summary.dbFieldChecks.frontend.ok =
    String(frontendDoc.project_code || '') === PROJECT_CODE &&
    summary.dbFieldChecks.frontend.request_mode === 'frontend' &&
    summary.dbFieldChecks.frontend.request_method === 'post' &&
    summary.dbFieldChecks.frontend.request_url === '/template_data/data' &&
    !/"method"\s*:|"url"\s*:|"_request_mode"\s*:|"_request_headers"\s*:/.test(summary.dbFieldChecks.frontend.code);

  summary.dbFieldChecks.backend.request_mode = String(backendDoc.request_mode || '');
  summary.dbFieldChecks.backend.request_method = String(backendDoc.request_method || '');
  summary.dbFieldChecks.backend.request_url = String(backendDoc.request_url || '');
  summary.dbFieldChecks.backend.code = String(backendDoc.code || '');
  summary.dbFieldChecks.backend.ok =
    String(backendDoc.project_code || '') === PROJECT_CODE &&
    summary.dbFieldChecks.backend.request_mode === 'backend' &&
    summary.dbFieldChecks.backend.request_method === 'get' &&
    summary.dbFieldChecks.backend.request_url.includes('postman-echo.com/get') &&
    !/"method"\s*:|"url"\s*:|"_request_mode"\s*:|"_request_headers"\s*:/.test(summary.dbFieldChecks.backend.code);

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

    await selectHttpDoc(page, 'test2', frontendName);
    let bodyText = await page.evaluate(() => String(document.body?.innerText || ''));
    summary.displayChecks.frontend.modeShown = bodyText.includes('前台直发');
    summary.displayChecks.frontend.methodShown = bodyText.includes('POST');
    summary.displayChecks.frontend.urlShown = bodyText.includes('/template_data/data');

    await page.getByRole('button', { name: '发送接口' }).first().click();
    const frontendWait = await waitForBodyText(page, /直发完成|直发失败/, 15000);
    const frontendBody = frontendWait.text || '';
    summary.requestChecks.frontendSend.statusKeyword = frontendBody.includes('直发完成') ? '直发完成' : (frontendBody.includes('直发失败') ? '直发失败' : '');
    const frontendResp = await getMonacoValue(page, 'workspace-http-console-response-');
    summary.requestChecks.frontendSend.responsePreview = String(frontendResp || '').slice(0, 1000);
    if (!summary.requestChecks.frontendSend.statusKeyword && /"success"\s*:\s*true|"code"\s*:\s*"0"/.test(summary.requestChecks.frontendSend.responsePreview)) {
      summary.requestChecks.frontendSend.statusKeyword = '直发完成(按响应)';
    }
    summary.requestChecks.frontendSend.ok =
      (summary.requestChecks.frontendSend.statusKeyword === '直发完成' || /"success"\s*:\s*true|"code"\s*:\s*"0"/.test(summary.requestChecks.frontendSend.responsePreview)) &&
      !/"status_code"\s*:/.test(summary.requestChecks.frontendSend.responsePreview);

    await selectHttpDoc(page, 'test2', backendName);
    bodyText = await page.evaluate(() => String(document.body?.innerText || ''));
    summary.displayChecks.backend.modeShown = bodyText.includes('后端代发');
    summary.displayChecks.backend.methodShown = bodyText.includes('GET');
    summary.displayChecks.backend.urlShown = bodyText.includes('postman-echo.com/get');

    await page.getByRole('button', { name: '发送接口' }).first().click();
    const backendWait = await waitForBodyText(page, /代发完成|代发失败|直发失败|直发完成/, 18000);
    const backendBody = backendWait.text || '';
    summary.requestChecks.backendSend.statusKeyword =
      backendBody.includes('代发完成') ? '代发完成'
        : backendBody.includes('代发失败') ? '代发失败'
          : backendBody.includes('直发完成') ? '直发完成'
            : backendBody.includes('直发失败') ? '直发失败'
              : '';
    const backendResp = await getMonacoValue(page, 'workspace-http-console-response-');
    summary.requestChecks.backendSend.responsePreview = String(backendResp || '').slice(0, 1000);
    summary.requestChecks.backendSend.ok =
      summary.requestChecks.backendSend.statusKeyword === '代发完成' &&
      /"status_code"\s*:\s*\d+/.test(summary.requestChecks.backendSend.responsePreview);

    const tree = page.locator('.workspace-http-tree').first();
    await tree.locator('.ant-tree-title', { hasText: 'test2' }).first().click();
    await sleep(500);
    await page.locator('button[title="新增HTTP"]').first().click();
    await page.waitForSelector('.ant-modal', { timeout: 10000 });
    await sleep(800);

    summary.addPanelDefaults.requestUrl = await page.locator('#workspace-http-doc-form_request_url').inputValue().catch(() => '');
    const selectorTexts = await page.locator('.ant-modal .ant-select-selector').allInnerTexts().catch(() => []);
    summary.addPanelDefaults.modeText = String(selectorTexts[1] || '');
    summary.addPanelDefaults.methodText = String(selectorTexts[2] || '');
    summary.addPanelDefaults.ok =
      summary.addPanelDefaults.requestUrl === '/template_data/data' &&
      summary.addPanelDefaults.modeText.includes('前台直发') &&
      (summary.addPanelDefaults.methodText.includes('GET') || summary.addPanelDefaults.methodText.includes('POST'));

    await page.locator('.ant-modal .ant-btn').filter({ hasText: '取 消' }).first().click().catch(() => undefined);

    const shot = path.join(OUT_DIR, 'webshell-editor-pool-http-mode-flow-check.png');
    await page.screenshot({ path: shot, fullPage: true });
    summary.screenshot = shot;
  } finally {
    await browser.close();
  }

  try {
    runCurl('config.doc_delete', { project_code: PROJECT_CODE, collect_doc_id_list: [summary.created.frontendDocId] });
    summary.cleanup.frontendDeleted = true;
  } catch (_error) {
    summary.cleanup.frontendDeleted = false;
  }
  try {
    runCurl('config.doc_delete', { project_code: PROJECT_CODE, collect_doc_id_list: [summary.created.backendDocId] });
    summary.cleanup.backendDeleted = true;
  } catch (_error) {
    summary.cleanup.backendDeleted = false;
  }

  const nonCanceledPageErrors = summary.pageErrors.filter((item) => !/Canceled/i.test(String(item || '')));
  summary.nonCanceledPageErrors = nonCanceledPageErrors;

  summary.pass =
    summary.displayChecks.frontend.modeShown &&
    summary.displayChecks.frontend.methodShown &&
    summary.displayChecks.frontend.urlShown &&
    summary.displayChecks.backend.modeShown &&
    summary.displayChecks.backend.methodShown &&
    summary.displayChecks.backend.urlShown &&
    summary.requestChecks.frontendSend.ok &&
    summary.requestChecks.backendSend.ok &&
    summary.addPanelDefaults.ok &&
    summary.dbFieldChecks.frontend.ok &&
    summary.dbFieldChecks.backend.ok &&
    summary.cleanup.frontendDeleted &&
    summary.cleanup.backendDeleted &&
    nonCanceledPageErrors.length === 0;

  const outJson = path.join(OUT_DIR, 'webshell-editor-pool-http-mode-flow-check.json');
  fs.writeFileSync(outJson, JSON.stringify(summary, null, 2));
  console.log(JSON.stringify(summary, null, 2));

  if (!summary.pass) {
    process.exitCode = 2;
  }
})();
