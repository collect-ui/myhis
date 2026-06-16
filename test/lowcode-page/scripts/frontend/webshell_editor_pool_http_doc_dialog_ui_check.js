#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)); }

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const summary = {
    pageUrl: PAGE_URL,
    modalOpen: false,
    title: '',
    rowAligned: false,
    headersBlockVisible: false,
    errors: [],
    screenshot: '',
    pass: false,
  };

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1680, height: 980 } });

  page.on('pageerror', (err) => summary.errors.push(String(err)));
  page.on('console', (msg) => {
    if (msg.type() === 'error') summary.errors.push(msg.text());
  });

  try {
    await page.goto(PAGE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await sleep(5500);

    const tree = page.locator('.workspace-http-tree').first();
    const groupNode = tree.locator('.ant-tree-title', { hasText: 'test2' }).first();
    if (await groupNode.isVisible().catch(() => false)) {
      await groupNode.click().catch(() => undefined);
      await page.waitForTimeout(280);
    }

    await page.locator('button[title="新增HTTP"]').first().click();
    await sleep(800);

    const modal = page.locator('.ant-modal:visible').first();
    summary.modalOpen = await modal.isVisible().catch(() => false);
    summary.title = String((await modal.locator('.ant-modal-title').first().innerText().catch(() => '')) || '').trim();

    // 切到“接口”态，验证接口编辑区的布局和视觉
    const typeSelect = page.locator('[id$=\"workspace-http-doc-form_type\"]').first();
    if (await typeSelect.isVisible().catch(() => false)) {
      await typeSelect.click().catch(() => undefined);
      await page.waitForTimeout(120);
      await page.locator('.ant-select-dropdown:visible .ant-select-item-option', { hasText: '接口' }).first().click().catch(() => undefined);
      await page.waitForTimeout(350);
    }

    const layout = await page.evaluate(() => {
      const getTop = (id) => {
        if (!id) return null;
        const el = document.querySelector(id);
        if (!el) return null;
        return Math.round(el.getBoundingClientRect().top);
      };
      const pick = (suffix) => {
        const node = document.querySelector(`[id$=\"${suffix}\"]`);
        if (!node) return null;
        return `#${node.id}`;
      };
      const modeTop = getTop(pick('workspace-http-doc-form_request_mode'));
      const methodTop = getTop(pick('workspace-http-doc-form_request_method'));
      const urlTop = getTop(pick('workspace-http-doc-form_request_url'));
      return { modeTop, methodTop, urlTop };
    });

    if ([layout.modeTop, layout.methodTop, layout.urlTop].every((n) => Number.isFinite(n))) {
      const maxDiff = Math.max(
        Math.abs(layout.modeTop - layout.methodTop),
        Math.abs(layout.modeTop - layout.urlTop),
        Math.abs(layout.methodTop - layout.urlTop)
      );
      summary.rowAligned = maxDiff <= 6;
    }

    summary.headersBlockVisible = await page.locator('.ant-modal:visible').getByText('请求头 Headers').first().isVisible().catch(() => false);

    const shot = path.join(OUT_DIR, 'webshell-editor-pool-http-doc-dialog-ui-check.png');
    await page.screenshot({ path: shot, fullPage: true });
    summary.screenshot = shot;
  } finally {
    await browser.close();
  }

  const nonCanceledErrors = summary.errors.filter((item) => !/Canceled/i.test(String(item || '')));
  summary.nonCanceledErrors = nonCanceledErrors;
  summary.pass = summary.modalOpen && summary.rowAligned && summary.headersBlockVisible && nonCanceledErrors.length === 0;

  const outJson = path.join(OUT_DIR, 'webshell-editor-pool-http-doc-dialog-ui-check.json');
  fs.writeFileSync(outJson, JSON.stringify(summary, null, 2));
  console.log(JSON.stringify(summary, null, 2));

  if (!summary.pass) process.exitCode = 2;
})();
