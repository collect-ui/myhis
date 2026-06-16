const { chromium } = require('playwright');
const fs = require('fs');

const PAGE_URL = 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const OUT_DIR = '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)); }

async function getResponsePanelText(page) {
  const lines = page
    .locator('.workspace-http-console-panels .workspace-http-console-panel')
    .nth(1)
    .locator('.view-lines');
  if (await lines.count().catch(() => 0)) {
    return String((await lines.first().innerText().catch(() => '')) || '').trim();
  }
  return '';
}

async function runCase(page, name, { mode, method, url }) {
  await page.locator('.workspace-http-console-mode').first().selectOption(mode);
  await page.locator('.workspace-http-console-method').first().selectOption(method.toLowerCase());
  await page.locator('.workspace-http-console-url').first().fill(url);
  await sleep(250);

  const before = await getResponsePanelText(page);
  await page.locator('.workspace-http-console-send').first().click();
  let after = before;
  let stableCount = 0;
  let changed = false;
  for (let i = 0; i < 60; i += 1) {
    await sleep(250);
    const current = await getResponsePanelText(page);
    if (current !== before) {
      changed = true;
      if (current === after) {
        stableCount += 1;
      } else {
        stableCount = 0;
      }
      after = current;
      if (stableCount >= 2) {
        break;
      }
    }
  }

  let pageMessage = '';
  const notice = page.locator('.workspace-http-console-notice').first();
  if (await notice.count().catch(() => 0)) {
    pageMessage = String((await notice.innerText().catch(() => '')) || '').trim();
  }

  return {
    name,
    mode,
    method,
    url,
    changed,
    responsePreview: after.slice(0, 900),
    responseLength: after.length,
    isError: /请求失败|failed|error/i.test(after),
    pageMessage,
  };
}

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1680, height: 980 } });

  const consoleErrors = [];
  const pageErrors = [];
  const failedRequests = [];

  page.on('console', (msg) => {
    if (msg.type() === 'error') consoleErrors.push(msg.text());
  });
  page.on('pageerror', (err) => pageErrors.push(String(err)));
  page.on('requestfailed', (req) => {
    failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || 'failed'}`);
  });

  await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
  await sleep(2500);

  await page.getByText('HTTP目录').first().click();
  await sleep(700);

  const sshNode = page.locator('.ant-tree-treenode', { has: page.getByText('SSH', { exact: true }) }).first();
  await sshNode.locator('.ant-tree-switcher').first().click();
  await sleep(700);

  await page.getByText(/get_env_detail_service/).first().click();
  await sleep(1200);

  let openedConsole = false;
  const openConsoleBtn = page.getByRole('button', { name: '打开控制台' }).first();
  if (await openConsoleBtn.isVisible().catch(() => false)) {
    await openConsoleBtn.click();
    openedConsole = true;
  }
  if (!openedConsole) {
    const compactConsoleBtn = page.getByRole('button', { name: '控制台' }).first();
    if (await compactConsoleBtn.isVisible().catch(() => false)) {
      await compactConsoleBtn.click();
      openedConsole = true;
    }
  }
  if (!openedConsole) {
    const sendBtn = page.getByRole('button', { name: '发送请求' }).first();
    if (await sendBtn.isVisible().catch(() => false)) {
      await sendBtn.click();
      openedConsole = true;
    }
  }
  if (!openedConsole) {
    throw new Error('failed to open http console');
  }
  await page.waitForSelector('.workspace-http-console-root', { timeout: 20000 });
  await sleep(900);

  const results = [];
  results.push(await runCase(page, 'internal-frontend', {
    mode: 'frontend',
    method: 'POST',
    url: '/template_data/data',
  }));

  results.push(await runCase(page, 'internal-backend', {
    mode: 'backend',
    method: 'POST',
    url: '/template_data/data',
  }));

  results.push(await runCase(page, 'internal-backend-absolute', {
    mode: 'backend',
    method: 'POST',
    url: 'http://192.168.232.130:8015/template_data/data',
  }));

  results.push(await runCase(page, 'external-frontend', {
    mode: 'frontend',
    method: 'GET',
    url: 'https://postman-echo.com/get',
  }));

  results.push(await runCase(page, 'external-frontend-cors', {
    mode: 'frontend',
    method: 'GET',
    url: 'https://jsonplaceholder.typicode.com/todos/1',
  }));

  results.push(await runCase(page, 'external-frontend-post-todos1', {
    mode: 'frontend',
    method: 'POST',
    url: 'https://jsonplaceholder.typicode.com/todos/1',
  }));

  results.push(await runCase(page, 'external-backend', {
    mode: 'backend',
    method: 'GET',
    url: 'https://postman-echo.com/get',
  }));

  results.push(await runCase(page, 'external-backend-post-todos1', {
    mode: 'backend',
    method: 'POST',
    url: 'https://jsonplaceholder.typicode.com/todos/1',
  }));

  const shot = `${OUT_DIR}/webshell-editor-pool-console-request-matrix-v2.png`;
  await page.screenshot({ path: shot, fullPage: true });

  const output = {
    pageUrl: PAGE_URL,
    results,
    consoleErrors,
    pageErrors,
    failedRequests,
    screenshot: shot,
  };

  const outJson = `${OUT_DIR}/webshell-editor-pool-console-request-matrix-v2.json`;
  fs.writeFileSync(outJson, JSON.stringify(output, null, 2));
  console.log(JSON.stringify(output, null, 2));

  await browser.close();
})();
