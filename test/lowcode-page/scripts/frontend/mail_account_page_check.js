#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

async function main() {
  let playwright;
  try {
    playwright = require('playwright');
  } catch (error) {
    console.error('playwright is required to run this script');
    console.error('install it in an available node environment, then rerun');
    process.exit(1);
  }

  const { chromium } = playwright;
  const baseUrl = process.env.MAIL_ACCOUNT_BASE_URL || 'http://192.168.232.130:8015';
  const apiUrl = process.env.MAIL_ACCOUNT_API_URL || `${baseUrl}/template_data/data`;
  const pageUrl = process.env.MAIL_ACCOUNT_PAGE_URL || `${baseUrl}/collect-ui#/collect-ui/framework/mail_account`;
  const username = process.env.MAIL_ACCOUNT_USERNAME || 'admin';
  const password = process.env.MAIL_ACCOUNT_PASSWORD || '123456';
  const outputDir = process.env.MAIL_ACCOUNT_OUTPUT_DIR || process.cwd();
  const timestamp = new Date().toISOString().replace(/[-:.TZ]/g, '').slice(0, 14);
  const prefix = `mail-page-${timestamp}`;
  const consoleErrors = [];
  const pageErrors = [];

  fs.mkdirSync(outputDir, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ ignoreHTTPSErrors: true });

  try {
    const loginResponse = await context.request.post(`${apiUrl}?service=system.login`, {
      data: {
        service: 'system.login',
        username,
        password,
      },
    });
    const loginJson = await loginResponse.json();
    if (!loginJson.success) {
      throw new Error(`login failed: ${loginJson.msg || 'unknown error'}`);
    }

    const page = await context.newPage();
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    page.on('pageerror', (err) => {
      pageErrors.push(String(err));
    });

    await page.goto(pageUrl, { waitUntil: 'networkidle' });
    await page.waitForSelector('text=邮箱登记', { timeout: 15000 });
    await page.getByRole('button', { name: '批量导入' }).click();

    const textarea = page.locator('textarea').first();
    const payload = [
      `${prefix}-1@example.com----pwd-${timestamp}-1----guid-${timestamp}-1----recovery-${timestamp}-1`,
      `${prefix}-1@example.com----pwd-${timestamp}-1----guid-${timestamp}-1----recovery-${timestamp}-1`,
      `broken-line-${timestamp}`,
      `${prefix}-2@example.com----pwd-${timestamp}-2----guid-${timestamp}-2----recovery-${timestamp}-2----tail`,
    ].join('\n');

    await textarea.fill(payload);
    await page.waitForFunction(() => {
      const button = Array.from(document.querySelectorAll('button')).find((item) => (item.innerText || '').includes('开始导入'));
      return !!button && !button.disabled;
    }, { timeout: 15000 });

    await page.getByRole('button', { name: '开始导入' }).click();
    await page.waitForSelector('text=导入完成', { timeout: 15000 });
    await page.waitForSelector('text=解析与导入摘要', { timeout: 15000 });
    await page.getByRole('button', { name: 'Close' }).last().click({ force: true });
    await page.waitForSelector(`text=${prefix}-1@example.com`, { timeout: 15000 });

    const screenshotPath = path.join(outputDir, 'page-after-import.png');
    await page.screenshot({ path: screenshotPath, fullPage: true });

    const deletedEmail = `${prefix}-1@example.com`;
    await page.getByPlaceholder('请输入邮箱关键字').fill(deletedEmail);
    await page.getByRole('button', { name: '搜索' }).click();

    const emailCell = page.locator('[col-id="email_name"]', { hasText: deletedEmail }).first();
    await emailCell.waitFor({ state: 'visible', timeout: 15000 });
    await page.getByRole('button', { name: /更\s*多\s*操\s*作/ }).last().click();
    await page.getByRole('menuitem', { name: /删\s*除\s*账\s*号/ }).click();
    await page.getByText(`确认删除邮箱【${deletedEmail}】吗？`).waitFor({ timeout: 15000 });
    await page.getByRole('button', { name: /确\s*定/ }).click();
    await page.getByRole('button', { name: '搜索' }).click();
    await page.waitForFunction((email) => {
      const actionButtons = Array.from(document.querySelectorAll('button')).filter((btn) => /更\s*多\s*操\s*作/.test(btn.innerText || ''));
      return actionButtons.length >= 1;
    }, deletedEmail, { timeout: 15000 });

    const deleteScreenshotPath = path.join(outputDir, 'page-after-delete.png');
    await page.screenshot({ path: deleteScreenshotPath, fullPage: true });

    const apiResponsePath = path.join(outputDir, 'api-response.json');
    fs.writeFileSync(apiResponsePath, JSON.stringify({
      login: loginJson,
      importedPrefix: prefix,
      deletedEmail,
    }, null, 2));

    const consoleLogPath = path.join(outputDir, 'console-errors.log');
    fs.writeFileSync(consoleLogPath, [...consoleErrors, ...pageErrors].join('\n'));

    const summaryPath = path.join(outputDir, 'summary.md');
    fs.writeFileSync(summaryPath, [
      '# Mail Account Page Check',
      '',
      `- Base URL: ${baseUrl}`,
      `- Page URL: ${pageUrl}`,
      `- Imported prefix: ${prefix}`,
      `- Deleted email: ${deletedEmail}`,
      `- Console errors: ${consoleErrors.length}`,
      `- Page errors: ${pageErrors.length}`,
      `- Screenshot: ${screenshotPath}`,
      `- Delete Screenshot: ${deleteScreenshotPath}`,
    ].join('\n'));
  } finally {
    await context.close();
    await browser.close();
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
