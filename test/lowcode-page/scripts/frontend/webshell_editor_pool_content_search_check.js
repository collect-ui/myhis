#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
const { chromium } = require('playwright');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://127.0.0.1:8015/template_data/data';
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || 'backend';
const PROJECT_NAME = process.env.WEBSHELL_EDITOR_POOL_PROJECT_NAME || '月神后端';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/content-search-validation';

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function runCurl(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  const res = spawnSync('curl', [
    '--noproxy',
    '*',
    '-s',
    '-m',
    '30',
    `${API_URL}?service=${service}`,
    '-H',
    'Content-Type: application/json',
    '--data',
    payload,
  ], { encoding: 'utf8' });
  if (res.status !== 0) {
    throw new Error(res.stderr || `curl failed: ${service}`);
  }
  let out = {};
  try {
    out = JSON.parse(String(res.stdout || '{}'));
  } catch (error) {
    throw new Error(`parse response failed (${service}): ${error.message}`);
  }
  return out;
}

async function waitBodyPattern(page, pattern, timeoutMs) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    const text = await page.evaluate(() => String(document.body?.innerText || ''));
    if (pattern.test(text)) {
      return text;
    }
    await sleep(300);
  }
  return page.evaluate(() => String(document.body?.innerText || ''));
}

async function getMonacoUris(page) {
  return page.evaluate(() => {
    const models = window?.monaco?.editor?.getModels?.() || [];
    return models.map((m) => decodeURIComponent(String(m?.uri || '')));
  });
}

async function waitToast(page, pattern, timeoutMs) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    const text = await page.locator('.ant-message').innerText().catch(() => '');
    if (pattern.test(String(text || ''))) {
      return true;
    }
    await sleep(150);
  }
  return false;
}

async function runSearchRound(page, opts) {
  const modal = page.locator('.ant-modal').filter({ hasText: '内容搜索' }).first();
  const keywordInput = modal.locator('input[placeholder="输入关键字(至少2个字符)"]').first();
  const includeInput = modal.locator('input[placeholder="*.py,*.sql (可选)"]').first();
  const includeSelect = modal.locator('.ant-select').filter({ hasText: /后缀筛选|全部后缀|Python \(\*\.py\)|SQL \(\*\.sql\)|Go \(\*\.go\)|TypeScript \(\*\.ts\)|TSX \(\*\.tsx\)|JavaScript \(\*\.js\)|Vue \(\*\.vue\)|YAML/ }).first();
  const searchBtn = modal.getByRole('button', { name: '搜索' }).first();
  const clearBtn = modal.getByRole('button', { name: '清空' }).first();
  await modal.waitFor({ state: 'visible', timeout: 15000 });
  await keywordInput.waitFor({ state: 'visible', timeout: 15000 });
  if (await clearBtn.isVisible().catch(() => false)) {
    await clearBtn.click().catch(() => undefined);
    await sleep(200);
  }
  await keywordInput.fill('');
  await keywordInput.fill(String(opts.keyword || ''));
  if (await includeInput.isVisible().catch(() => false)) {
    await includeInput.fill('');
    if (opts.includeGlob) {
      await includeInput.fill(String(opts.includeGlob));
    }
  } else if (await includeSelect.isVisible().catch(() => false)) {
    if (opts.includeGlob) {
      await includeSelect.click().catch(() => undefined);
      await sleep(200);
      if (String(opts.includeGlob) === '*.py') {
        await page.locator('.ant-select-dropdown .ant-select-item-option').filter({ hasText: 'Python (*.py)' }).first().click().catch(() => undefined);
      } else if (String(opts.includeGlob) === '*.sql') {
        await page.locator('.ant-select-dropdown .ant-select-item-option').filter({ hasText: 'SQL (*.sql)' }).first().click().catch(() => undefined);
      } else {
        await page.locator('.ant-select-dropdown .ant-select-item-option').filter({ hasText: '全部后缀' }).first().click().catch(() => undefined);
      }
    }
  }
  await page.keyboard.press('Escape').catch(() => undefined);
  await keywordInput.press('Enter');

  const toastKeywordError = await waitToast(page, /关键字至少2个字符/, 2500);
  const text = await waitBodyPattern(page, /命中文件|关键字至少2个字符|项目\[/, opts.timeoutMs || 15000);
  const lineCount = await page.locator('text=/第\\d+行/').count().catch(() => 0);
  const hasSummary = /命中文件\s*\d+/.test(text) && /耗时\s*\d+ms/.test(text);
  const hasKeywordError = toastKeywordError || /关键字至少2个字符/.test(text);
  return {
    keyword: opts.keyword,
    include_glob: opts.includeGlob || '',
    lineCount,
    hasSummary,
    hasKeywordError,
    bodyHasResult: /第\d+行/.test(text),
  };
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const now = Date.now();
  const summary = {
    pageUrl: PAGE_URL,
    apiUrl: API_URL,
    projectCode: PROJECT_CODE,
    projectName: PROJECT_NAME,
    startedAt: new Date().toISOString(),
    apiChecks: {
      baseline: {},
      invalidKeyword: {},
      notFoundProject: {},
    },
    uiChecks: {
      contentSearchButtonVisible: false,
      contentSearchDialogVisible: false,
      panelInputVisible: false,
      roundA: {},
      roundB: {},
      roundInvalid: {},
      clickOpen: {
        expectedPath: '',
        monacoHasExpectedPath: false,
        openRequestHit: false,
        openRequestContainsPath: false,
      },
    },
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    screenshot: '',
    pass: false,
  };

  const base = runCurl('webshell.workspace_file_content_search', {
    project_code: PROJECT_CODE,
    keyword: 'import',
    max_results: 1,
  });
  summary.apiChecks.baseline = {
    code: String(base.code || ''),
    success: base.success === true,
    msg: String(base.msg || ''),
    itemCount: Array.isArray(base?.data?.items) ? base.data.items.length : 0,
    summary: base?.data?.summary || {},
  };
  summary.uiChecks.clickOpen.expectedPath = String((base?.data?.items?.[0]?.file_path) || '');

  const invalidKeyword = runCurl('webshell.workspace_file_content_search', {
    project_code: PROJECT_CODE,
    keyword: 'a',
  });
  summary.apiChecks.invalidKeyword = {
    code: String(invalidKeyword.code || ''),
    success: invalidKeyword.success === true,
    msg: String(invalidKeyword.msg || ''),
  };

  const notFoundProject = runCurl('webshell.workspace_file_content_search', {
    project_code: 'not_exists',
    keyword: 'import',
  });
  summary.apiChecks.notFoundProject = {
    code: String(notFoundProject.code || ''),
    success: notFoundProject.success === true,
    msg: String(notFoundProject.msg || ''),
  };

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1680, height: 960 } });
  page.setDefaultTimeout(15000);
  let openRequestPayload = '';

  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      summary.consoleErrors.push(msg.text());
    }
  });
  page.on('pageerror', (err) => summary.pageErrors.push(String(err)));
  page.on('request', (req) => {
    const url = req.url();
    if (!url.includes('service=webshell.workspace_file_content')) {
      return;
    }
    openRequestPayload = String(req.postData() || '');
  });
  page.on('requestfailed', (req) => {
    summary.failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || 'failed'}`);
  });

  try {
    await page.goto(PAGE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await sleep(5500);

    const projectBtn = page.getByRole('button', { name: PROJECT_NAME }).first();
    if (await projectBtn.isVisible().catch(() => false)) {
      await projectBtn.click();
      await sleep(800);
    }

    const searchToggle = page.locator('button[title="内容搜索"]').first();
    summary.uiChecks.contentSearchButtonVisible = await searchToggle.isVisible().catch(() => false);
    if (summary.uiChecks.contentSearchButtonVisible) {
      await searchToggle.click();
      await sleep(500);
    }
    summary.uiChecks.contentSearchDialogVisible = await page.locator('.ant-modal .ant-modal-title', { hasText: '内容搜索' }).first().isVisible().catch(() => false);

    const panelKeyword = page.locator('input[placeholder="输入关键字(至少2个字符)"]').first();
    summary.uiChecks.panelInputVisible = await panelKeyword.isVisible().catch(() => false);

    summary.uiChecks.roundA = await runSearchRound(page, { keyword: 'import', includeGlob: '' });
    summary.uiChecks.roundB = await runSearchRound(page, { keyword: 'import', includeGlob: '*.py' });
    summary.uiChecks.roundInvalid = await runSearchRound(page, { keyword: 'a', includeGlob: '' });

    const firstLine = page.locator('text=/第\\d+行/').first();
    if (await firstLine.isVisible().catch(() => false)) {
      await firstLine.click();
      await sleep(1400);
      const uris = await getMonacoUris(page);
      const expected = summary.uiChecks.clickOpen.expectedPath;
      summary.uiChecks.clickOpen.monacoHasExpectedPath = !!expected && uris.some((u) => u.includes(expected));
      summary.uiChecks.clickOpen.openRequestHit = openRequestPayload.includes('workspace_file_content');
      summary.uiChecks.clickOpen.openRequestContainsPath = !!expected && openRequestPayload.includes(expected);
    }

    const shot = path.join(OUT_DIR, `content-search-${now}.png`);
    await page.screenshot({ path: shot, fullPage: true });
    summary.screenshot = shot;
  } finally {
    await browser.close();
  }

  summary.pass = [
    summary.apiChecks.baseline.code === '0',
    summary.apiChecks.baseline.itemCount > 0,
    /关键字至少2个字符/.test(summary.apiChecks.invalidKeyword.msg || ''),
    /项目\[not_exists\]不存在/.test(summary.apiChecks.notFoundProject.msg || ''),
    summary.uiChecks.contentSearchButtonVisible,
    summary.uiChecks.contentSearchDialogVisible,
    summary.uiChecks.panelInputVisible,
    summary.uiChecks.roundA.hasSummary,
    summary.uiChecks.roundA.lineCount > 0,
    summary.uiChecks.roundB.hasSummary,
    summary.uiChecks.roundB.lineCount > 0,
    summary.uiChecks.roundInvalid.hasKeywordError,
    summary.uiChecks.clickOpen.openRequestHit,
    summary.uiChecks.clickOpen.openRequestContainsPath || summary.uiChecks.clickOpen.monacoHasExpectedPath,
    summary.consoleErrors.length === 0,
    summary.pageErrors.length === 0,
    summary.failedRequests.length === 0,
  ].every(Boolean);

  const outFile = path.join(OUT_DIR, `content-search-${now}.json`);
  fs.writeFileSync(outFile, `${JSON.stringify(summary, null, 2)}\n`, 'utf8');

  console.log(JSON.stringify({
    pass: summary.pass,
    outFile,
    screenshot: summary.screenshot,
    roundA: summary.uiChecks.roundA,
    roundB: summary.uiChecks.roundB,
    roundInvalid: summary.uiChecks.roundInvalid,
    clickOpen: summary.uiChecks.clickOpen,
    consoleErrors: summary.consoleErrors.length,
    pageErrors: summary.pageErrors.length,
    failedRequests: summary.failedRequests.length,
  }, null, 2));

  process.exit(summary.pass ? 0 : 1);
})();
