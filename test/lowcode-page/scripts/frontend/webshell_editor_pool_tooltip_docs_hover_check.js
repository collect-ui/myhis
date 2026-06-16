#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
const { chromium } = require('playwright');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://127.0.0.1:8015/template_data/data';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';

const PY_PROJECT_CODE = 'backend';
const PY_PROJECT_NAME = '月神后端';
const GO_PROJECT_CODE = 'autodesk';
const GO_PROJECT_NAME = '后端客户端';

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function runCurl(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  const res = spawnSync('curl', [
    '--noproxy',
    '*',
    '-sS',
    '-m',
    '40',
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
  if (!out || String(out.code || '') !== '0' || out.success === false) {
    throw new Error(`${service} failed: ${out?.msg || 'unknown error'}`);
  }
  return out;
}

function normalizePath(input) {
  return String(input || '').replace(/\\/g, '/').replace(/\/+$/g, '');
}

function escapeRegExp(input) {
  return String(input || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

async function waitUntil(fn, timeoutMs, intervalMs = 250) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    const result = await fn();
    if (result) {
      return result;
    }
    await sleep(intervalMs);
  }
  return null;
}

async function switchProject(page, projectName) {
  const btn = page.getByRole('button', { name: new RegExp(`^${escapeRegExp(projectName)}$`) }).first();
  await btn.waitFor({ state: 'visible', timeout: 22000 });
  await btn.click();
  await sleep(1100);
}

async function openFileFromTree(page, keyword, fileName) {
  const input = page.locator('input[placeholder="回车搜索(至少2个字符)"]:visible').first();
  await input.waitFor({ state: 'visible', timeout: 22000 });
  await input.fill('');
  await input.fill(String(keyword || fileName || ''));
  await input.press('Enter');
  await sleep(800);
  const exact = new RegExp(`^${escapeRegExp(fileName)}$`);
  const title = page.locator('.workspace-source-tree .ant-tree-title:visible').filter({ hasText: exact }).first();
  await title.waitFor({ state: 'visible', timeout: 18000 });
  await title.click();
  await sleep(1300);
}

async function getVisibleEditorPath(page) {
  return page.evaluate(() => {
    const root = document.querySelector('[data-viewer-id][data-store-current-file-path]');
    return String(root?.getAttribute?.('data-store-current-file-path') || '');
  });
}

async function waitForEditorPath(page, expectedPath, timeoutMs = 22000) {
  const normalizedExpected = normalizePath(expectedPath);
  return waitUntil(async () => {
    const currentPath = normalizePath(await getVisibleEditorPath(page));
    if (currentPath && currentPath.includes(normalizedExpected)) {
      return currentPath;
    }
    return null;
  }, timeoutMs, 260);
}

async function hoverTokenAndRead(page, opts) {
  const target = await page.evaluate(({ lineRegexText, tokenText }) => {
    const monacoNs = window?.monaco;
    const editors = monacoNs?.editor?.getEditors?.() || [];
    const getVisible = () => {
      for (const editor of editors) {
        try {
          const host = editor?.getContainerDomNode?.();
          const slotEl = host?.closest?.('[data-slot-id]');
          if (!slotEl) continue;
          const style = window.getComputedStyle(slotEl);
          if (!style || style.display === 'none' || style.visibility === 'hidden') continue;
          const model = editor?.getModel?.();
          if (!model) continue;
          return { editor, model };
        } catch (_error) {
          // ignore
        }
      }
      return null;
    };

    const pair = getVisible();
    if (!pair) {
      return { ok: false, reason: 'visible editor not found' };
    }
    const { editor, model } = pair;
    const maxLine = Number(model.getLineCount?.() || 0);
    const re = new RegExp(lineRegexText);
    for (let i = 1; i <= maxLine; i += 1) {
      const text = String(model.getLineContent?.(i) || '');
      if (!re.test(text)) {
        continue;
      }
      const idx = text.indexOf(String(tokenText || ''));
      if (idx < 0) {
        continue;
      }
      const col = idx + 2;
      editor.revealLineInCenter?.(i);
      editor.setPosition?.({ lineNumber: i, column: col });
      editor.focus?.();
      const visible = editor.getScrolledVisiblePosition?.({ lineNumber: i, column: col });
      const dom = editor.getContainerDomNode?.();
      if (!visible || !dom) {
        return { ok: false, reason: 'visible position unavailable' };
      }
      const rect = dom.getBoundingClientRect();
      const x = rect.left + visible.left + 4;
      const y = rect.top + visible.top + Math.max(6, Math.floor(visible.height / 2));
      return { ok: true, x, y, lineNumber: i, column: col, lineText: text };
    }
    return { ok: false, reason: `target token not found: ${tokenText}` };
  }, { lineRegexText: opts.lineRegexText, tokenText: opts.tokenText });

  if (!target?.ok) {
    throw new Error(target?.reason || `hover target resolve failed: ${opts.tokenText}`);
  }

  await page.mouse.move(2, 2);
  await sleep(120);
  await page.mouse.move(Number(target.x), Number(target.y));

  const hoverStartedAt = Date.now();
  const hoverText = await waitUntil(async () => {
    const text = await page.locator('.monaco-hover:visible').first().innerText().catch(() => '');
    const clean = String(text || '').trim();
    if (!clean) {
      return null;
    }
    if (/^loading\.{0,3}$/i.test(clean)) {
      return null;
    }
    return clean ? clean : null;
  }, 12000, 150);

  if (!hoverText) {
    throw new Error(`hover text empty: ${opts.tokenText}`);
  }

  return {
    token: opts.tokenText,
    lineRegexText: opts.lineRegexText,
    lineNumber: target.lineNumber,
    column: target.column,
    lineText: target.lineText,
    hoverWaitMs: Date.now() - hoverStartedAt,
    hoverText,
  };
}

function assertContains(text, expected, label) {
  if (String(text || '').indexOf(String(expected || '')) < 0) {
    throw new Error(`${label} missing expected text: ${expected}`);
  }
}

function assertDelay(waitMs, minMs, label) {
  const value = Number(waitMs || 0);
  if (!(value >= minMs)) {
    throw new Error(`${label} hover delay too short: ${value}ms < ${minMs}ms`);
  }
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const summary = {
    pageUrl: PAGE_URL,
    apiUrl: API_URL,
    startedAt: new Date().toISOString(),
    pass: false,
    error: '',
    screenshot: '',
    python: {},
    go: {},
  };

  let browser;
  try {
    const pyQuery = runCurl('webshell.workspace_file_query', {
      project_code: PY_PROJECT_CODE,
      keyword: 'jira/comments/index.yml',
      pagination: false,
    });
    const pyPath = String((pyQuery.data || []).find((row) => String(row?.path || '').endsWith('/jira/comments/index.yml'))?.path || (pyQuery.data || [])[0]?.path || '');
    if (!pyPath) {
      throw new Error('python target file not found: jira/comments/index.yml');
    }

    const goQuery = runCurl('webshell.workspace_file_query', {
      project_code: GO_PROJECT_CODE,
      keyword: 'config/sync/index.yml',
      pagination: false,
    });
    const goPath = String((goQuery.data || []).find((row) => String(row?.path || '').endsWith('/collect/config/sync/index.yml'))?.path || (goQuery.data || [])[0]?.path || '');
    if (!goPath) {
      throw new Error('go target file not found: collect/config/sync/index.yml');
    }

    browser = await chromium.launch({ headless: true });

    const pyPage = await browser.newPage({ viewport: { width: 1680, height: 980 } });
    await pyPage.goto(PAGE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await pyPage.waitForTimeout(1800);
    await switchProject(pyPage, PY_PROJECT_NAME);
    await openFileFromTree(pyPage, 'jira/comments/index.yml', 'index.yml');
    if (!(await waitForEditorPath(pyPage, pyPath, 22000))) {
      throw new Error('python file not opened');
    }

    const pyModuleKey = await hoverTokenAndRead(pyPage, {
      lineRegexText: '^\\s*module:\\s*mysql\\s*$',
      tokenText: 'module',
    });
    assertContains(pyModuleKey.hoverText, 'module', 'python module key hover');
    assertContains(pyModuleKey.hoverText, '文档路径', 'python module key hover');
    assertContains(pyModuleKey.hoverText, '文档内容预览', 'python module key hover');
    assertDelay(pyModuleKey.hoverWaitMs, 1700, 'python module key');

    const pyModuleValue = await hoverTokenAndRead(pyPage, {
      lineRegexText: '^\\s*module:\\s*mysql\\s*$',
      tokenText: 'mysql',
    });
    assertContains(pyModuleValue.hoverText, 'module=mysql', 'python module=mysql hover');

    const pyTemplate = await hoverTokenAndRead(pyPage, {
      lineRegexText: '^\\s*template:\\s*"\\{% if search',
      tokenText: 'template',
    });
    assertContains(pyTemplate.hoverText, 'template', 'python template hover');

    summary.python = {
      filePath: pyPath,
      moduleKey: pyModuleKey,
      moduleValue: pyModuleValue,
      template: pyTemplate,
    };

    const goPage = await browser.newPage({ viewport: { width: 1680, height: 980 } });
    await goPage.goto(PAGE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await goPage.waitForTimeout(1800);
    await switchProject(goPage, GO_PROJECT_NAME);
    await openFileFromTree(goPage, 'config/sync/index.yml', 'index.yml');
    if (!(await waitForEditorPath(goPage, goPath, 22000))) {
      throw new Error('go file not opened');
    }

    const goModuleKey = await hoverTokenAndRead(goPage, {
      lineRegexText: '^\\s*module:\\s*empty\\s*$',
      tokenText: 'module',
    });
    assertContains(goModuleKey.hoverText, 'module', 'go module key hover');
    assertContains(goModuleKey.hoverText, '文档内容预览', 'go module key hover');
    assertDelay(goModuleKey.hoverWaitMs, 1700, 'go module key');

    const goModuleValue = await hoverTokenAndRead(goPage, {
      lineRegexText: '^\\s*module:\\s*empty\\s*$',
      tokenText: 'empty',
    });
    assertContains(goModuleValue.hoverText, 'module=empty', 'go module=empty hover');

    const goTableKey = await hoverTokenAndRead(goPage, {
      lineRegexText: '^\\s*table:\\s*collect_doc_important\\s*$',
      tokenText: 'table',
    });
    assertContains(goTableKey.hoverText, 'table', 'go table hover');

    summary.go = {
      filePath: goPath,
      moduleKey: goModuleKey,
      moduleValue: goModuleValue,
      table: goTableKey,
    };

    const shot = path.join(OUT_DIR, 'webshell-editor-pool-tooltip-docs-hover-check.png');
    await goPage.screenshot({ path: shot, fullPage: true });
    summary.screenshot = shot;

    await goPage.close().catch(() => {});
    await pyPage.close().catch(() => {});
    summary.pass = true;
  } catch (error) {
    summary.error = String(error?.message || error);
  } finally {
    if (browser) {
      await browser.close().catch(() => {});
    }
    summary.endedAt = new Date().toISOString();
    const outFile = path.join(OUT_DIR, 'webshell-editor-pool-tooltip-docs-hover-check.json');
    fs.writeFileSync(outFile, JSON.stringify(summary, null, 2));
    console.log(JSON.stringify({
      pass: summary.pass,
      outFile,
      screenshot: summary.screenshot,
      error: summary.error,
    }, null, 2));
    process.exit(summary.pass ? 0 : 1);
  }
})();
