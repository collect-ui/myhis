#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
let playwright;
try {
  playwright = require('playwright');
} catch (_error) {
  playwright = require('/data/project/sport-ui/node_modules/playwright');
}
const { chromium } = playwright;

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://127.0.0.1:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://127.0.0.1:8015/template_data/data';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || 'test';
const FILE_PATH = process.env.WEBSHELL_EDITOR_POOL_HTML_FILE_PATH || '/data/project/test/test.html';
const FILE_NAME = path.posix.basename(FILE_PATH);
const HTML_TEXT = `<!doctype html>\n<html>\n  <body>\n    <h1 id=\"html-preview-pass\">HTML PREVIEW OK</h1>\n    <script>window.__html_preview_ok = 1;</script>\n  </body>\n</html>\n`;

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
  const parsed = JSON.parse(String(res.stdout || '{}'));
  if (!parsed || String(parsed.code || '') !== '0' || parsed.success === false) {
    throw new Error(`${service} failed: ${parsed?.msg || 'unknown error'}`);
  }
  return parsed;
}

function escapeRe(text) {
  return String(text).replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const result = {
    pass: false,
    pageUrl: PAGE_URL,
    projectCode: PROJECT_CODE,
    filePath: FILE_PATH,
    checks: {
      createFile: false,
      openFileInEditor: false,
      defaultSplitPreviewVisible: false,
      previewButtonVisible: false,
      previewRenderOk: false,
      splitPreviewRenderOk: false,
    },
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    screenshot: '',
    report: '',
    error: '',
  };

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1680, height: 980 } });
  page.on('console', (msg) => {
    if (msg.type() === 'error') result.consoleErrors.push(msg.text());
  });
  page.on('pageerror', (error) => result.pageErrors.push(String(error)));
  page.on('requestfailed', (req) => {
    result.failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || 'failed'}`);
  });

  try {
    // 0) 清理并通过 API 创建文件，确保用例可重复执行
    try {
      runCurl('webshell.workspace_file_delete_with_sync', {
        project_code: PROJECT_CODE,
        path: FILE_PATH,
      });
    } catch (_error) {
      // ignore when file does not exist
    }
    runCurl('webshell.workspace_file_add_with_sync', {
      project_code: PROJECT_CODE,
      name: FILE_NAME,
      path: FILE_PATH,
      is_dir: '0',
      parent_id: '',
    });
    result.checks.createFile = true;

    // 1) 打开页面并选项目
    await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
    const projectBtn = page.getByRole('button', { name: new RegExp(`^${escapeRe(PROJECT_CODE)}$`) }).first();
    await projectBtn.waitFor({ state: 'visible', timeout: 20000 });
    await projectBtn.click();
    await sleep(1500);

    // 2) 通过 API 写入 HTML 内容
    runCurl('webshell.workspace_file_save', {
      project_code: PROJECT_CODE,
      path: FILE_PATH,
      content: HTML_TEXT,
    });
    await page.locator('button[title="同步"]').first().click();
    const confirm = page.locator('.ant-modal-wrap:visible .ant-btn-primary').last();
    await confirm.waitFor({ state: 'visible', timeout: 10000 });
    await confirm.click();
    await sleep(2500);

    // 3) 在树里打开 test.html
    const fileNode = page.locator('.workspace-source-tree .ant-tree-title', { hasText: new RegExp(`^${escapeRe(FILE_NAME)}$`) }).first();
    await fileNode.waitFor({ state: 'visible', timeout: 20000 });
    await fileNode.click();
    await sleep(1800);
    result.checks.openFileInEditor = true;

    // 4) 默认打开即分屏预览，确保用户不需要先找按钮。
    const defaultSplitWrap = page.locator('[data-testid="workspace-route-html-split-preview"]').first();
    await defaultSplitWrap.waitFor({ state: 'visible', timeout: 10000 });
    const defaultSplitText = await defaultSplitWrap.frameLocator('iframe').locator('#html-preview-pass').textContent();
    result.checks.defaultSplitPreviewVisible = String(defaultSplitText || '').includes('HTML PREVIEW OK');

    // 5) 预览模式校验
    const previewBtn = page.getByRole('button', { name: '预览' }).first();
    await previewBtn.waitFor({ state: 'visible', timeout: 10000 });
    result.checks.previewButtonVisible = true;
    await previewBtn.click();
    const previewWrap = page.locator('[data-testid="workspace-route-html-preview"]').first();
    await previewWrap.waitFor({ state: 'visible', timeout: 10000 });
    const previewFrame = previewWrap.locator('iframe').first();
    await previewFrame.waitFor({ state: 'visible', timeout: 10000 });
    const previewText = await previewWrap.frameLocator('iframe').locator('#html-preview-pass').textContent();
    result.checks.previewRenderOk = String(previewText || '').includes('HTML PREVIEW OK');

    // 6) 分屏模式校验
    await page.getByRole('button', { name: '分屏' }).first().click();
    const splitWrap = page.locator('[data-testid="workspace-route-html-split-preview"]').first();
    await splitWrap.waitFor({ state: 'visible', timeout: 10000 });
    const splitFrame = splitWrap.locator('iframe').first();
    await splitFrame.waitFor({ state: 'visible', timeout: 10000 });
    const splitText = await splitWrap.frameLocator('iframe').locator('#html-preview-pass').textContent();
    result.checks.splitPreviewRenderOk = String(splitText || '').includes('HTML PREVIEW OK');

    const shot = path.join(OUT_DIR, 'webshell-editor-pool-html-preview-check.png');
    await page.screenshot({ path: shot, fullPage: true });
    result.screenshot = shot;
  } catch (error) {
    result.error = String(error && error.stack ? error.stack : error);
  } finally {
    await browser.close();
  }

  result.pass =
    result.checks.createFile &&
    result.checks.openFileInEditor &&
    result.checks.defaultSplitPreviewVisible &&
    result.checks.previewButtonVisible &&
    result.checks.previewRenderOk &&
    result.checks.splitPreviewRenderOk &&
    result.consoleErrors.length === 0 &&
    result.pageErrors.length === 0;

  const reportPath = path.join(OUT_DIR, 'webshell-editor-pool-html-preview-check.json');
  fs.writeFileSync(reportPath, JSON.stringify(result, null, 2));
  result.report = reportPath;
  console.log(JSON.stringify(result, null, 2));

  if (!result.pass) {
    process.exit(2);
  }
})();
