#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
const { chromium } = require('playwright');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://192.168.232.130:8015/template_data/data';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)); }
function mark(step) { console.log(`[create-doc-check] ${step}`); }

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

function flattenDocNodes(tree) {
  const docs = [];
  const walk = (nodes) => {
    if (!Array.isArray(nodes)) return;
    nodes.forEach((node) => {
      if (!node) return;
      if (isDir(node)) {
        walk(Array.isArray(node.children) ? node.children : []);
        return;
      }
      const collectDocId = normalizeId(node.collect_doc_id || node.id || '');
      if (!collectDocId) return;
      docs.push({
        collectDocId,
        title: String(node.title || node.name || '').trim(),
        subTitle: String(node.sub_title || node.cn_title || '').trim(),
        parentDir: normalizeId(node.parent_id || node.doc_group_id || ''),
        orderIndex: Number(node.order_index || 0) || 0,
      });
    });
  };
  walk(Array.isArray(tree) ? tree : []);
  return docs;
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

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const caseName = `auto_console_create_${Date.now()}`;
  const beforeTree = runCurl('config.doc_group_http_service_tree_query', { to_tree: true });
  const beforeDocs = flattenDocNodes(beforeTree.data || []);
  const beforeIdSet = new Set(beforeDocs.map((item) => item.collectDocId));
  const test2Group = (() => {
    let hit = null;
    const walk = (nodes) => {
      if (hit || !Array.isArray(nodes)) return;
      nodes.forEach((node) => {
        if (hit || !node) return;
        if (isDir(node)) {
          const title = String(node.title || node.display_title || node.name || '').trim();
          if (title === 'test2') {
            hit = { id: normalizeId(node.id || node.doc_group_id || ''), title };
            return;
          }
          walk(Array.isArray(node.children) ? node.children : []);
        }
      });
    };
    walk(beforeTree.data || []);
    return hit;
  })();

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1680, height: 980 } });

  const consoleErrors = [];
  const pageErrors = [];
  const failedRequests = [];

  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      consoleErrors.push(msg.text());
    }
  });
  page.on('pageerror', (err) => pageErrors.push(String(err)));
  page.on('requestfailed', (req) => {
    failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || 'failed'}`);
  });
  page.on('dialog', async (dialog) => {
    mark(`browser dialog: ${dialog.type()} ${dialog.message()}`);
    if (dialog.type() === 'prompt') {
      await dialog.accept(caseName).catch(() => undefined);
      return;
    }
    await dialog.dismiss().catch(() => undefined);
  });

  const summary = {
    pageUrl: PAGE_URL,
    caseName,
    expectedGroupId: test2Group?.id || '',
    createdDocId: '',
    createdTitle: '',
    createdParentDir: '',
    saveNotice: '',
    saveToast: '',
    cleanupDeleted: false,
    consoleErrors,
    pageErrors,
    failedRequests,
    pass: false,
  };

  try {
    mark('goto');
    await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
    await sleep(2200);

    mark('open http tab');
    const httpTab = page.getByText('HTTP目录').first();
    if (await httpTab.isVisible().catch(() => false)) {
      await httpTab.click();
      await sleep(900);
    }

    mark('click add console');
    const addConsoleBtn = page.locator('button[title="新增HTTP控制台"]').first();
    await addConsoleBtn.click();
    await page.waitForSelector('.workspace-http-console-root', { timeout: 20000 });
    await sleep(700);

    mark('prepare request');
    await page.locator('.workspace-http-console-mode').first().selectOption('backend');
    await page.locator('.workspace-http-console-method').first().selectOption('get');
    await page.locator('.workspace-http-console-url').first().fill('https://jsonplaceholder.typicode.com/todos/1');
    await setMonacoValue(page, 'http-console-header-', '{}');
    await setMonacoValue(page, 'http-console-request-', '{}');
    await sleep(300);

    mark('trigger save');
    await page.evaluate(() => {
      const btn = document.querySelector("button[title='保存']");
      if (btn) {
        btn.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
      }
    });
    mark('wait modal');
    await page.waitForSelector('.ant-modal', { timeout: 20000 });
    await sleep(300);

    mark('fill modal title');
    const modal = page.locator('.ant-modal').last();
    await modal.getByPlaceholder('例如：查询用户列表').fill(caseName);
    mark('submit modal');
    await modal.locator('.ant-btn-primary').last().click();
    await sleep(1800);

    summary.saveNotice = String((await page.locator('.workspace-http-console-notice').first().innerText().catch(() => '')) || '').trim();
    summary.saveToast = String((await page.locator('.ant-message').innerText().catch(() => '')) || '').trim();

    const screenshot = path.join(OUT_DIR, 'webshell-editor-pool-console-create-doc-closure-check.png');
    await page.screenshot({ path: screenshot, fullPage: true });
    summary.screenshot = screenshot;

    await sleep(700);
    mark('query tree');
    const afterTree = runCurl('config.doc_group_http_service_tree_query', { to_tree: true });
    const afterDocs = flattenDocNodes(afterTree.data || []);
    const created = afterDocs.find((item) => !beforeIdSet.has(item.collectDocId) && item.title === caseName) || null;

    if (created) {
      summary.createdDocId = created.collectDocId;
      summary.createdTitle = created.title;
      summary.createdParentDir = created.parentDir;
      const del = runCurl('config.doc_delete', { collect_doc_id_list: [created.collectDocId] });
      summary.cleanupDeleted = String(del.code || '') === '0';
    }

    summary.pass =
      !!summary.createdDocId &&
      summary.createdTitle === caseName &&
      summary.createdParentDir === String(summary.expectedGroupId || '') &&
      summary.cleanupDeleted === true &&
      summary.pageErrors.length === 0;

    const outJson = path.join(OUT_DIR, 'webshell-editor-pool-console-create-doc-closure-check.json');
    fs.writeFileSync(outJson, JSON.stringify(summary, null, 2));
    console.log(JSON.stringify(summary, null, 2));

    if (!summary.pass) {
      process.exitCode = 2;
    }
  } finally {
    await browser.close();
  }
})();
