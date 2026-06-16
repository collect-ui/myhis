#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)); }

async function openHttpService(page) {
  await page.getByText('HTTP目录').first().click();
  await sleep(900);

  const groupNames = ['SSH', 'test2', '文档管理', 'ldap', 'hrm'];
  for (const name of groupNames) {
    const groupNode = page.locator('.ant-tree-treenode', { has: page.getByText(new RegExp(`^${name}$`)) }).first();
    if (await groupNode.isVisible().catch(() => false)) {
      const switcher = groupNode.locator('.ant-tree-switcher').first();
      if (await switcher.isVisible().catch(() => false)) {
        await switcher.click();
        await sleep(700);
      } else {
        await groupNode.click();
        await sleep(700);
      }
      break;
    }
  }

  const serviceMatchers = [
    /get_env_detail_service/,
    /sync_doc/,
    /ldap_search/,
    /my_env_tree/,
    /test\(tset\)/,
  ];

  let clickedService = '';
  for (const matcher of serviceMatchers) {
    const node = page.getByText(matcher).first();
    if (await node.isVisible().catch(() => false)) {
      clickedService = String((await node.textContent()) || '').trim();
      await node.click();
      await sleep(1100);
      break;
    }
  }

  if (!clickedService) {
    const fallback = page.locator('.ant-tree-title').filter({ hasText: '(' }).first();
    if (await fallback.isVisible().catch(() => false)) {
      clickedService = String((await fallback.textContent()) || '').trim();
      await fallback.click();
      await sleep(1100);
    }
  }

  if (!clickedService) {
    throw new Error('failed to open http service node');
  }

  return clickedService;
}

async function setFirstMonacoModelValue(page, matchText, value) {
  return page.evaluate(({ matchText, value }) => {
    const monaco = window.monaco;
    const models = monaco?.editor?.getModels?.() || [];
    for (const model of models) {
      const raw = String(model?.uri || '');
      const decoded = decodeURIComponent(raw);
      if (raw.includes(matchText) || decoded.includes(matchText)) {
        model.setValue(String(value || ''));
        return true;
      }
    }
    return false;
  }, { matchText, value });
}

async function waitForNoticeChange(page, before) {
  for (let i = 0; i < 60; i += 1) {
    await sleep(250);
    const now = String((await page.locator('.workspace-http-console-notice').first().innerText().catch(() => '')) || '').trim();
    if (now && now !== before) {
      return now;
    }
  }
  return String((await page.locator('.workspace-http-console-notice').first().innerText().catch(() => '')) || '').trim();
}

(async () => {
  const { chromium } = require('playwright');
  fs.mkdirSync(OUT_DIR, { recursive: true });

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

  const summary = {
    pageUrl: PAGE_URL,
    serviceName: '',
    expected: {
      method: 'GET',
      mode: 'backend',
      url: 'https://jsonplaceholder.typicode.com/todos/1',
    },
    step: {},
    consoleErrors,
    pageErrors,
    failedRequests,
  };

  try {
    await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
    await sleep(2200);

    summary.serviceName = await openHttpService(page);

    await page.getByRole('button', { name: '发送请求' }).first().click();
    await page.waitForSelector('.workspace-http-console-root', { timeout: 20000 });
    await sleep(900);

    await page.locator('.workspace-http-console-mode').first().selectOption('backend');
    await page.locator('.workspace-http-console-method').first().selectOption('get');
    await page.locator('.workspace-http-console-url').first().fill(summary.expected.url);

    await setFirstMonacoModelValue(page, 'http-console-request-', '{}');
    await setFirstMonacoModelValue(page, 'http-console-header-', '{}');
    await sleep(300);

    const beforeNotice = String((await page.locator('.workspace-http-console-notice').first().innerText().catch(() => '')) || '').trim();
    await page.locator('.workspace-http-console-send').first().click();
    const noticeAfterSend = await waitForNoticeChange(page, beforeNotice);

    summary.step.afterSendNotice = noticeAfterSend;
    summary.step.sendSuccess = /请求成功/.test(noticeAfterSend) && /状态\s*200/.test(noticeAfterSend);

    await page.getByRole('button', { name: '保存' }).first().click();
    await sleep(900);

    const toastText = await page.locator('.ant-message').innerText().catch(() => '');
    summary.step.saveToast = String(toastText || '').trim();

    const saveShot = path.join(OUT_DIR, 'webshell-editor-pool-console-save-after-save.png');
    await page.screenshot({ path: saveShot, fullPage: true });
    summary.step.afterSaveScreenshot = saveShot;

    await page.reload({ waitUntil: 'networkidle', timeout: 60000 });
    await sleep(2200);

    await openHttpService(page);
    await sleep(900);

    const heroMethod = String((await page.locator('.workspace-http-method').first().innerText().catch(() => '')) || '').trim();
    const heroUrl = String((await page.locator('.workspace-http-path').first().innerText().catch(() => '')) || '').trim();
    summary.step.reloadHero = { method: heroMethod, url: heroUrl };

    await page.getByRole('button', { name: '发送请求' }).first().click();
    await page.waitForSelector('.workspace-http-console-root', { timeout: 20000 });
    await sleep(900);

    const modeAfterReload = await page.locator('.workspace-http-console-mode').first().inputValue();
    const methodAfterReload = await page.locator('.workspace-http-console-method').first().inputValue();
    const urlAfterReload = await page.locator('.workspace-http-console-url').first().inputValue();

    summary.step.reloadConsole = {
      mode: modeAfterReload,
      method: methodAfterReload,
      url: urlAfterReload,
    };

    const noticeAfterReload = String((await page.locator('.workspace-http-console-notice').first().innerText().catch(() => '')) || '').trim();
    summary.step.reloadNotice = noticeAfterReload;

    summary.pass =
      summary.step.sendSuccess === true &&
      String(heroMethod || '').toUpperCase() === 'GET' &&
      heroUrl === summary.expected.url &&
      modeAfterReload === 'backend' &&
      methodAfterReload === 'get' &&
      urlAfterReload === summary.expected.url;

    const reloadShot = path.join(OUT_DIR, 'webshell-editor-pool-console-save-after-reload.png');
    await page.screenshot({ path: reloadShot, fullPage: true });
    summary.step.afterReloadScreenshot = reloadShot;

    const outJson = path.join(OUT_DIR, 'webshell-editor-pool-console-save-check.json');
    fs.writeFileSync(outJson, JSON.stringify(summary, null, 2));
    console.log(JSON.stringify(summary, null, 2));

    if (!summary.pass) {
      process.exitCode = 2;
    }
  } finally {
    await browser.close();
  }
})();
