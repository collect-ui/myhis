#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';
const GROUP_TITLE = process.env.WEBSHELL_EDITOR_POOL_GROUP_TITLE || 'test2';
const DOC_TITLE = process.env.WEBSHELL_EDITOR_POOL_DOC_TITLE || '登录';

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function escapeRegex(text) {
  return String(text || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

async function clickRightTreeTitle(page, title, options) {
  const strict = !!(options && options.strict);
  const minX = Number((options && options.minX) || 900);
  return page.evaluate(({ title, strict, minX }) => {
    const normalize = (v) => String(v || '').trim();
    const target = normalize(title);
    const list = Array.from(document.querySelectorAll('.ant-tree-title'));
    const filtered = list.filter((el) => {
      const t = normalize(el.textContent);
      if (strict) return t === target;
      return t.includes(target);
    });
    if (!filtered.length) return { ok: false, reason: 'not_found' };
    const visible = filtered.filter((el) => {
      const rect = el.getBoundingClientRect();
      return rect.width > 0 && rect.height > 0 && rect.x >= minX && rect.y >= 0 && rect.y <= window.innerHeight;
    });
    const picked = (visible.length ? visible : filtered)[0];
    if (!picked) return { ok: false, reason: 'no_pick' };
    picked.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
    return { ok: true };
  }, { title, strict, minX });
}

async function expandRightTreeGroup(page, groupTitle) {
  return page.evaluate(({ groupTitle }) => {
    const normalize = (v) => String(v || '').trim();
    const target = normalize(groupTitle);
    const list = Array.from(document.querySelectorAll('.ant-tree-title')).filter((el) => normalize(el.textContent) === target);
    const picked = list.find((el) => {
      const rect = el.getBoundingClientRect();
      return rect.width > 0 && rect.height > 0 && rect.x >= 900 && rect.y >= 0 && rect.y <= window.innerHeight;
    }) || list[0];
    if (!picked) return { ok: false, reason: 'group_not_found' };
    const node = picked.closest('.ant-tree-treenode');
    const switcher = node ? node.querySelector('.ant-tree-switcher') : null;
    if (switcher && String(switcher.className || '').includes('ant-tree-switcher_close')) {
      switcher.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
      return { ok: true, expanded: true };
    }
    return { ok: true, expanded: false };
  }, { groupTitle });
}

async function openHttpDoc(page) {
  await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
  await sleep(1800);

  const httpDirTab = page.getByText('HTTP目录').first();
  if (await httpDirTab.isVisible().catch(() => false)) {
    await httpDirTab.click();
    await sleep(500);
  }

  const groupTitleRegex = new RegExp(`^\\s*${escapeRegex(GROUP_TITLE)}\\s*$`);
  await page.locator('.ant-tree-title', { hasText: groupTitleRegex }).first().waitFor({ state: 'visible', timeout: 20000 });

  const expanded = await expandRightTreeGroup(page, GROUP_TITLE);
  if (!expanded || !expanded.ok) {
    throw new Error(`cannot expand group: ${GROUP_TITLE}`);
  }
  await sleep(800);

  const clickedDoc = await clickRightTreeTitle(page, DOC_TITLE, { strict: false, minX: 900 });
  if (!clickedDoc || !clickedDoc.ok) {
    throw new Error(`cannot click doc: ${DOC_TITLE}`);
  }
  await sleep(1200);
}

async function openLogDialogFromInterface(page, summary) {
  const btn = page.locator('button', { hasText: '查看日志' }).first();
  await btn.waitFor({ state: 'visible', timeout: 20000 });
  await btn.click();
  await sleep(600);

  const modal = page.locator('.ant-modal-content', { has: page.getByText('HTTP 代发日志') }).first();
  await modal.waitFor({ state: 'visible', timeout: 20000 });

  await page.waitForFunction(() => {
    const body = String(document.body?.innerText || '');
    return !body.includes('日志加载中...');
  }, {}, { timeout: 20000 }).catch(() => undefined);
  await sleep(300);

  const rows = await page.locator('.workspace-http-log-card').count().catch(() => 0);
  const hasEmpty = await page.getByText('暂无日志').first().isVisible().catch(() => false);
  const hasError = await page.locator('.ant-modal-content').first().getByText('日志查询失败').isVisible().catch(() => false);
  summary.interfacePanel = {
    clicked: true,
    dialogVisible: true,
    rowCount: rows,
    emptyStateVisible: hasEmpty,
    errorVisible: hasError,
  };

  const closeBtn = page.locator('.ant-modal-content button', { hasText: '关闭' }).first();
  if (await closeBtn.isVisible().catch(() => false)) {
    await closeBtn.click();
  } else {
    const iconClose = page.locator('.ant-modal-close').first();
    if (await iconClose.isVisible().catch(() => false)) {
      await iconClose.click();
    } else {
      await page.keyboard.press('Escape');
    }
  }
  await page.locator('.ant-modal-content').first().waitFor({ state: 'hidden', timeout: 20000 }).catch(() => undefined);
  await sleep(500);
}

async function openConsoleForCurrentDoc(page) {
  const tryOpen = async (name) => {
    const btn = page.locator('button', { hasText: name }).first();
    if (await btn.isVisible().catch(() => false)) {
      await btn.click();
      return true;
    }
    return false;
  };

  let opened = await tryOpen('新开控制台');
  if (!opened) opened = await tryOpen('打开控制台');
  if (!opened) opened = await tryOpen('控制台');
  if (!opened) {
    throw new Error('cannot find console open button');
  }

  await page.waitForFunction(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    return buttons.some((btn) => String(btn.textContent || '').includes('发送请求'));
  }, {}, { timeout: 20000 });
  await sleep(900);
}

async function openLogDialogFromConsole(page, summary) {
  const logBtns = page.locator('button', { hasText: '查看日志' });
  const count = await logBtns.count();
  const btn = logBtns.nth(Math.max(0, count - 1));
  await btn.waitFor({ state: 'visible', timeout: 20000 });
  await btn.click();
  await sleep(600);

  const modal = page.locator('.ant-modal-content', { has: page.getByText('HTTP 代发日志') }).first();
  await modal.waitFor({ state: 'visible', timeout: 20000 });

  await page.waitForFunction(() => {
    const body = String(document.body?.innerText || '');
    return !body.includes('日志加载中...');
  }, {}, { timeout: 20000 }).catch(() => undefined);
  await sleep(300);

  const rows = await page.locator('.workspace-http-log-card').count().catch(() => 0);
  const hasEmpty = await page.getByText('暂无日志').first().isVisible().catch(() => false);
  const hasError = await page.locator('.ant-modal-content').first().getByText('日志查询失败').isVisible().catch(() => false);
  summary.consolePanel = {
    clicked: true,
    dialogVisible: true,
    rowCount: rows,
    emptyStateVisible: hasEmpty,
    errorVisible: hasError,
  };

  const closeBtn = page.locator('.ant-modal-content button', { hasText: '关闭' }).first();
  if (await closeBtn.isVisible().catch(() => false)) {
    await closeBtn.click();
  } else {
    const iconClose = page.locator('.ant-modal-close').first();
    if (await iconClose.isVisible().catch(() => false)) {
      await iconClose.click();
    } else {
      await page.keyboard.press('Escape');
    }
  }
  await page.locator('.ant-modal-content').first().waitFor({ state: 'hidden', timeout: 20000 }).catch(() => undefined);
  await sleep(500);
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1680, height: 980 } });

  const summary = {
    pageUrl: PAGE_URL,
    groupTitle: GROUP_TITLE,
    docTitle: DOC_TITLE,
    interfacePanel: {
      clicked: false,
      dialogVisible: false,
      rowCount: 0,
      emptyStateVisible: false,
      errorVisible: false,
    },
    consolePanel: {
      clicked: false,
      dialogVisible: false,
      rowCount: 0,
      emptyStateVisible: false,
      errorVisible: false,
    },
    screenshot: '',
    pass: false,
    error: '',
  };

  try {
    await openHttpDoc(page);
    await openLogDialogFromInterface(page, summary);
    await openConsoleForCurrentDoc(page);
    await openLogDialogFromConsole(page, summary);

    summary.pass =
      summary.interfacePanel.dialogVisible &&
      summary.consolePanel.dialogVisible &&
      !summary.interfacePanel.errorVisible &&
      !summary.consolePanel.errorVisible;
  } catch (error) {
    summary.error = String(error && error.message ? error.message : error);
    summary.pass = false;
  }

  const shot = path.join(OUT_DIR, 'webshell-editor-pool-http-log-dialog-check.png');
  await page.screenshot({ path: shot, fullPage: true }).catch(() => undefined);
  summary.screenshot = shot;

  const outJson = path.join(OUT_DIR, 'webshell-editor-pool-http-log-dialog-check.json');
  fs.writeFileSync(outJson, JSON.stringify(summary, null, 2));
  console.log(JSON.stringify(summary, null, 2));

  await browser.close();
  if (!summary.pass) {
    process.exitCode = 2;
  }
})();
