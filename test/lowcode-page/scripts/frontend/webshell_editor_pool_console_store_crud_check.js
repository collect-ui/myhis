#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

const PAGE_URL = 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const OUT_DIR = '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';
const STORE_FILE = '/data/project/sport/test2/http_console_store.json';

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)); }

function readStoreSafe() {
  try {
    const raw = fs.readFileSync(STORE_FILE, 'utf8');
    const parsed = JSON.parse(raw || '{}');
    const items = Array.isArray(parsed?.items) ? parsed.items : [];
    return { raw, items };
  } catch (_error) {
    return { raw: '', items: [] };
  }
}

async function submitStoreNameModal(page, name) {
  const modalWrap = page.locator('.ant-modal-wrap:visible').last();
  await modalWrap.waitFor({ state: 'visible', timeout: 10000 });
  const input = modalWrap.locator('input.ant-input').first();
  await input.waitFor({ state: 'visible', timeout: 10000 });
  await input.fill(String(name || ''));
  await modalWrap.locator('.ant-btn-primary').last().dispatchEvent('click');
  await modalWrap.waitFor({ state: 'hidden', timeout: 10000 }).catch(() => undefined);
}

async function submitConfirmModal(page) {
  const modalWrap = page.locator('.ant-modal-wrap:visible').last();
  await modalWrap.waitFor({ state: 'visible', timeout: 10000 });
  await modalWrap.locator('.ant-btn-primary').last().dispatchEvent('click');
  await modalWrap.waitFor({ state: 'hidden', timeout: 10000 }).catch(() => undefined);
}

async function openHttpService(page) {
  await page.getByText('HTTP目录').first().click();
  await sleep(800);

  for (let round = 0; round < 3; round += 1) {
    const nodes = page.locator('.ant-tree-treenode');
    const count = await nodes.count();
    for (let i = 0; i < count; i += 1) {
      const node = nodes.nth(i);
      const switcher = node.locator('.ant-tree-switcher').first();
      const cls = await switcher.getAttribute('class').catch(() => '');
      if (cls && cls.includes('ant-tree-switcher_close')) {
        await switcher.click().catch(() => undefined);
      }
    }
    await sleep(260);
  }

  const target = page.getByText(/get_env_detail_service\(获取服务\)/).first();
  await target.click();
  await sleep(1000);
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const before = readStoreSafe();
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
  page.on('requestfailed', (req) => failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || 'failed'}`));
  const result = {
    pageUrl: PAGE_URL,
    storeFile: STORE_FILE,
    beforeCount: before.items.length,
    afterCount: 0,
    createdName: 'crud-case-create',
    updatedName: 'crud-case-updated',
    createdVisible: false,
    updatedVisible: false,
    deletedSuccess: false,
    consoleErrors,
    pageErrors,
    failedRequests,
    pass: false,
  };

  try {
    await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
    await sleep(2200);

    await openHttpService(page);
    await page.getByRole('button', { name: '发送请求' }).first().click();
    await page.waitForSelector('.workspace-http-console-root', { timeout: 20000 });
    await sleep(900);

    await page.locator('.workspace-http-console-mode').first().selectOption('backend');
    await page.locator('.workspace-http-console-method').first().selectOption('get');
    await page.locator('.workspace-http-console-url').first().fill('https://jsonplaceholder.typicode.com/todos/1');

    const reloadBtn = page.locator('.workspace-http-console-store-btn[title="查询测试数据"]');
    const addBtn = page.locator('.workspace-http-console-store-btn[title="新增测试数据"]');
    const updateBtn = page.locator('.workspace-http-console-store-btn[title="更新当前测试数据"]');
    const deleteBtn = page.locator('.workspace-http-console-store-btn[title="删除当前测试数据"]');
    const select = page.locator('.workspace-http-console-store-select').first();

    await reloadBtn.first().click();
    await sleep(600);

    await addBtn.first().click();
    await submitStoreNameModal(page, result.createdName);
    await sleep(400);

    const optionsAfterCreate = await select.locator('option').allTextContents();
    result.createdVisible = optionsAfterCreate.some((t) => String(t).includes(result.createdName));
    if (result.createdVisible) {
      const createdOption = select.locator('option').filter({ hasText: result.createdName }).first();
      const createdValue = await createdOption.getAttribute('value').catch(() => "");
      if (createdValue) {
        await select.selectOption(createdValue);
        await sleep(300);
      }
    }

    await page.locator('.workspace-http-console-url').first().fill('https://jsonplaceholder.typicode.com/todos/2');
    await updateBtn.first().click();
    await submitStoreNameModal(page, result.updatedName);
    await sleep(400);

    const optionsAfterUpdate = await select.locator('option').allTextContents();
    result.updatedVisible = optionsAfterUpdate.some((t) => String(t).includes(result.updatedName));

    await deleteBtn.first().click();
    await submitConfirmModal(page);
    await sleep(400);

    const optionsAfterDelete = await select.locator('option').allTextContents();
    result.deletedSuccess = !optionsAfterDelete.some((t) => String(t).includes(result.updatedName));

    await reloadBtn.first().click();
    await sleep(600);

    const shot = path.join(OUT_DIR, 'webshell-editor-pool-console-store-crud-check.png');
    await page.screenshot({ path: shot, fullPage: true });
    result.screenshot = shot;
  } finally {
    await browser.close();
  }

  const after = readStoreSafe();
  result.afterCount = after.items.length;
  result.pass =
    result.createdVisible &&
    result.updatedVisible &&
    result.deletedSuccess &&
    result.afterCount === result.beforeCount &&
    result.consoleErrors.length === 0 &&
    result.pageErrors.length === 0;

  const out = path.join(OUT_DIR, 'webshell-editor-pool-console-store-crud-check.json');
  fs.writeFileSync(out, JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));

  if (!result.pass) {
    process.exit(2);
  }
})();
