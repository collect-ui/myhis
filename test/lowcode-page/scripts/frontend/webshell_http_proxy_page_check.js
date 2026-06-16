#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

async function main() {
  let playwright;
  try {
    playwright = require('playwright');
  } catch (error) {
    console.error('playwright is required to run this script');
    process.exit(1);
  }

  const { chromium } = playwright;
  const baseUrl = process.env.WEBSHELL_HTTP_PROXY_BASE_URL || 'http://192.168.232.130:8015';
  const directPageUrl = process.env.WEBSHELL_HTTP_PROXY_PAGE_URL || `${baseUrl}/collect-ui#/collect-ui/framework/webshell_editor_pool`;
  const fallbackPageUrl = `${baseUrl}/collect-ui#/collect-ui/framework/webshell`;
  const outputDir = process.env.WEBSHELL_HTTP_PROXY_OUTPUT_DIR || path.join(process.cwd(), 'test/lowcode-page/results/latest/http-proxy-validation');
  const consoleErrors = [];
  const pageErrors = [];
  const failedRequests = [];

  fs.mkdirSync(outputDir, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ ignoreHTTPSErrors: true });

  try {
    const page = await context.newPage();
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    page.on('pageerror', (err) => {
      pageErrors.push(String(err));
    });
    page.on('requestfailed', (req) => {
      failedRequests.push(`${req.url()} :: ${req.failure()?.errorText || 'failed'}`);
    });

    await page.goto(directPageUrl, { waitUntil: 'networkidle', timeout: 30000 });

    const directLoaded = await page.locator('text=HTTP目录').first().isVisible().catch(() => false);
    if (!directLoaded) {
      await page.goto(fallbackPageUrl, { waitUntil: 'networkidle', timeout: 30000 });
      await page.getByRole('button', { name: '工作空间' }).click().catch(async () => {
        await page.getByTitle('工作空间').click();
      });
    }

    await page.waitForSelector('text=HTTP目录', { timeout: 20000 });
    await page.waitForSelector('text=工作空间', { timeout: 20000 });

    const treeNode = page.getByText('test(tset)').first();
    await treeNode.waitFor({ state: 'visible', timeout: 20000 });
    await treeNode.click({ button: 'right' });
    await page.getByRole('menuitem', { name: /编辑接口/ }).click();

    await page.waitForSelector('text=编辑接口', { timeout: 15000 });
    await page.waitForSelector('text=请求模式', { timeout: 15000 });
    await page.waitForSelector('text=请求方法', { timeout: 15000 });
    await page.waitForSelector('text=请求URL', { timeout: 15000 });
    await page.waitForSelector('text=请求Header', { timeout: 15000 });

    const screenshotPath = path.join(outputDir, 'webshell-http-proxy-page.png');
    const resultPath = path.join(outputDir, 'webshell-http-proxy-page-result.json');
    const summaryPath = path.join(outputDir, 'webshell-http-proxy-page-summary.md');

    await page.screenshot({ path: screenshotPath, fullPage: true });

    const detail = {
      baseUrl,
      directPageUrl,
      fallbackPageUrl,
      usedFallback: !directLoaded,
      matchedNode: 'test(tset)',
      visibleLabels: ['请求模式', '请求方法', '请求URL', '请求Header'],
      consoleErrors,
      pageErrors,
      failedRequests,
    };

    fs.writeFileSync(resultPath, JSON.stringify(detail, null, 2));
    fs.writeFileSync(summaryPath, [
      '# Webshell HTTP Proxy Page Check',
      '',
      `- Base URL: ${baseUrl}`,
      `- Direct Page URL: ${directPageUrl}`,
      `- Used Fallback: ${detail.usedFallback}`,
      `- Matched Node: ${detail.matchedNode}`,
      `- Console Errors: ${consoleErrors.length}`,
      `- Page Errors: ${pageErrors.length}`,
      `- Failed Requests: ${failedRequests.length}`,
      `- Screenshot: ${screenshotPath}`,
    ].join('\n'));

    console.log(JSON.stringify(detail, null, 2));
  } finally {
    await context.close();
    await browser.close();
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
