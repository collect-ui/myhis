#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
const { chromium } = require('playwright');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://127.0.0.1:8015/template_data/data';
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || 'autodesk';
const PROJECT_NAME = process.env.WEBSHELL_EDITOR_POOL_PROJECT_NAME || '后端客户端';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';

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

function escapeRegExp(input) {
  return String(input || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function normalizePath(input) {
  return String(input || '').replace(/\\/g, '/').replace(/\/+$/g, '');
}

function toSnakeFileToken(input) {
  return String(input || '')
    .trim()
    .replace(/-/g, '_')
    .replace(/([a-z0-9])([A-Z])/g, '$1_$2')
    .toLowerCase();
}

function findLineNoByPattern(text, pattern) {
  const lines = String(text || '').split(/\r?\n/);
  for (let i = 0; i < lines.length; i += 1) {
    if (pattern.test(String(lines[i] || ''))) {
      return i + 1;
    }
  }
  return 0;
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

async function getVisibleEditorState(page) {
  return page.evaluate(() => {
    const rootEl = document.querySelector('[data-viewer-id][data-store-current-file-path]');
    const currentFilePath = String(rootEl?.getAttribute?.('data-store-current-file-path') || '');
    const getVisibleEditor = () => {
      const monacoNs = window?.monaco;
      const editors = monacoNs?.editor?.getEditors?.() || [];
      for (const editor of editors) {
        try {
          const host = editor?.getContainerDomNode?.();
          const slotEl = host?.closest?.('[data-slot-id]');
          const viewerId = slotEl?.getAttribute?.('data-viewer-id');
          if (!viewerId) {
            continue;
          }
          const style = slotEl ? window.getComputedStyle(slotEl) : null;
          if (!style || style.display === 'none' || style.visibility === 'hidden') {
            continue;
          }
          const model = editor?.getModel?.();
          if (!model) {
            continue;
          }
          return { editor, model };
        } catch (_error) {
          // ignore
        }
      }
      return null;
    };

    const pair = getVisibleEditor();
    if (!pair) {
      return null;
    }
    const { editor, model } = pair;
    const position = editor.getPosition?.() || { lineNumber: 1, column: 1 };
    return {
      currentFilePath,
      uri: decodeURIComponent(String(model?.uri || '')),
      lineNumber: Number(position.lineNumber || 1),
      column: Number(position.column || 1),
      lineCount: Number(model.getLineCount?.() || 0),
    };
  });
}

async function waitForEditorPath(page, expectedPath, timeoutMs = 25000) {
  const normalizedExpected = normalizePath(expectedPath);
  return waitUntil(async () => {
    const info = await getVisibleEditorState(page);
    if (!info) {
      return null;
    }
    const normalizedCurrentPath = normalizePath(String(info.currentFilePath || ''));
    if (normalizedCurrentPath && normalizedCurrentPath.includes(normalizedExpected)) {
      return info;
    }
    return null;
  }, timeoutMs, 250);
}

async function waitForEditorLine(page, expectedLine, timeoutMs = 25000) {
  const lineNo = Number(expectedLine || 0);
  if (!Number.isFinite(lineNo) || lineNo < 1) {
    return null;
  }
  return waitUntil(async () => {
    const info = await getVisibleEditorState(page);
    if (!info) {
      return null;
    }
    if (Number(info.lineNumber || 0) === lineNo) {
      return info;
    }
    return null;
  }, timeoutMs, 200);
}

async function openFileFromTree(page, keyword, fileName) {
  const input = page.locator('input[placeholder="回车搜索(至少2个字符)"]:visible').first();
  await input.waitFor({ state: 'visible', timeout: 20000 });
  await input.fill('');
  await input.fill(String(keyword || fileName || ''));
  await input.press('Enter');
  await sleep(700);
  const exact = new RegExp(`^${escapeRegExp(fileName)}$`);
  const title = page.locator('.workspace-source-tree .ant-tree-title').filter({ hasText: exact }).first();
  await title.waitFor({ state: 'visible', timeout: 18000 });
  await title.click();
  await sleep(1500);
}

async function clickFieldValueInEditor(page, fieldName, expectedValue) {
  const result = await page.evaluate(({ fieldName, expectedValue }) => {
    const getVisibleEditor = () => {
      const monacoNs = window?.monaco;
      const editors = monacoNs?.editor?.getEditors?.() || [];
      for (const editor of editors) {
        try {
          const host = editor?.getContainerDomNode?.();
          const slotEl = host?.closest?.('[data-slot-id]');
          const viewerId = slotEl?.getAttribute?.('data-viewer-id');
          if (!viewerId) {
            continue;
          }
          const style = slotEl ? window.getComputedStyle(slotEl) : null;
          if (!style || style.display === 'none' || style.visibility === 'hidden') {
            continue;
          }
          const model = editor?.getModel?.();
          if (!model) {
            continue;
          }
          return { editor, model };
        } catch (_error) {
          // ignore
        }
      }
      return null;
    };

    const pair = getVisibleEditor();
    if (!pair) {
      return { ok: false, reason: 'visible editor not found' };
    }
    const { editor, model } = pair;
    const maxLine = Number(model.getLineCount?.() || 0);
    const escapedField = String(fieldName || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const escapedValue = String(expectedValue || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const re = new RegExp(`\\b${escapedField}\\s*:\\s*(?:'([^']*)'|\\"([^\\\"]*)\\"|([^#\\s]+))`);

    let hitLine = 0;
    let hitColumn = 0;
    let lineText = '';
    for (let i = 1; i <= maxLine; i += 1) {
      const text = String(model.getLineContent?.(i) || '');
      const m = text.match(re);
      if (!m) {
        continue;
      }
      const value = String(m[1] || m[2] || m[3] || '').trim();
      if (!value) {
        continue;
      }
      if (escapedValue && !new RegExp(`^${escapedValue}$`).test(value)) {
        continue;
      }
      const idx = text.indexOf(value);
      if (idx < 0) {
        continue;
      }
      hitLine = i;
      hitColumn = idx + 2;
      lineText = text;
      break;
    }

    if (!hitLine || !hitColumn) {
      return { ok: false, reason: `value not found for ${fieldName}: ${expectedValue}` };
    }

    editor.revealLineInCenter?.(hitLine);
    editor.setPosition?.({ lineNumber: hitLine, column: hitColumn });
    editor.focus?.();

    const visible = editor.getScrolledVisiblePosition?.({ lineNumber: hitLine, column: hitColumn });
    const dom = editor.getContainerDomNode?.();
    if (!visible || !dom) {
      return { ok: false, reason: 'visible position unavailable' };
    }
    const rect = dom.getBoundingClientRect();
    const x = rect.left + visible.left + 4;
    const y = rect.top + visible.top + Math.max(6, Math.floor(visible.height / 2));

    return {
      ok: true,
      x,
      y,
      lineNumber: hitLine,
      column: hitColumn,
      lineText,
      uri: decodeURIComponent(String(model?.uri || '')),
    };
  }, { fieldName, expectedValue });

  if (!result || !result.ok) {
    throw new Error(result?.reason || `click target not found: ${fieldName}=${expectedValue}`);
  }

  let ctrlHoverHint = false;
  await page.keyboard.down('Control');
  try {
    await page.mouse.move(Number(result.x), Number(result.y));
    await sleep(120);
    ctrlHoverHint = await page.evaluate(() => !!document.querySelector('.workspace-config-jump-link'));
    await page.mouse.click(Number(result.x), Number(result.y));
  } finally {
    await page.keyboard.up('Control');
  }
  await sleep(1400);
  return { ...result, ctrlHoverHint };
}

async function clickHandlerKeyInEditor(page, keyValue) {
  const result = await page.evaluate(({ keyValue }) => {
    const getVisibleEditor = () => {
      const monacoNs = window?.monaco;
      const editors = monacoNs?.editor?.getEditors?.() || [];
      for (const editor of editors) {
        try {
          const host = editor?.getContainerDomNode?.();
          const slotEl = host?.closest?.('[data-slot-id]');
          const viewerId = slotEl?.getAttribute?.('data-viewer-id');
          if (!viewerId) {
            continue;
          }
          const style = slotEl ? window.getComputedStyle(slotEl) : null;
          if (!style || style.display === 'none' || style.visibility === 'hidden') {
            continue;
          }
          const model = editor?.getModel?.();
          if (!model) {
            continue;
          }
          return { editor, model };
        } catch (_error) {
          // ignore
        }
      }
      return null;
    };

    const pair = getVisibleEditor();
    if (!pair) {
      return { ok: false, reason: 'visible editor not found' };
    }
    const { editor, model } = pair;
    const maxLine = Number(model.getLineCount?.() || 0);
    const escapedValue = String(keyValue || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const re = new RegExp(`^\\s*-\\s*key\\s*:\\s*(?:'${escapedValue}'|\\"${escapedValue}\\"|${escapedValue})\\s*(?:#.*)?$`);

    let hitLine = 0;
    let hitColumn = 0;
    let lineText = '';
    for (let i = 1; i <= maxLine; i += 1) {
      const text = String(model.getLineContent?.(i) || '');
      if (!re.test(text)) {
        continue;
      }
      const idx = text.indexOf(String(keyValue || ''));
      if (idx < 0) {
        continue;
      }
      hitLine = i;
      hitColumn = idx + 2;
      lineText = text;
      break;
    }

    if (!hitLine || !hitColumn) {
      return { ok: false, reason: `handler key not found: ${keyValue}` };
    }

    editor.revealLineInCenter?.(hitLine);
    editor.setPosition?.({ lineNumber: hitLine, column: hitColumn });
    editor.focus?.();

    const visible = editor.getScrolledVisiblePosition?.({ lineNumber: hitLine, column: hitColumn });
    const dom = editor.getContainerDomNode?.();
    if (!visible || !dom) {
      return { ok: false, reason: 'visible position unavailable' };
    }
    const rect = dom.getBoundingClientRect();
    const x = rect.left + visible.left + 4;
    const y = rect.top + visible.top + Math.max(6, Math.floor(visible.height / 2));

    return {
      ok: true,
      x,
      y,
      lineNumber: hitLine,
      column: hitColumn,
      lineText,
      uri: decodeURIComponent(String(model?.uri || '')),
    };
  }, { keyValue });

  if (!result || !result.ok) {
    throw new Error(result?.reason || `click key target failed: ${keyValue}`);
  }

  let ctrlHoverHint = false;
  await page.keyboard.down('Control');
  try {
    await page.mouse.move(Number(result.x), Number(result.y));
    await sleep(120);
    ctrlHoverHint = await page.evaluate(() => !!document.querySelector('.workspace-config-jump-link'));
    await page.mouse.click(Number(result.x), Number(result.y));
  } finally {
    await page.keyboard.up('Control');
  }
  await sleep(1400);
  return { ...result, ctrlHoverHint };
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const summary = {
    pageUrl: PAGE_URL,
    apiUrl: API_URL,
    projectCode: PROJECT_CODE,
    projectName: PROJECT_NAME,
    startedAt: new Date().toISOString(),
    expected: {
      syncIndexPath: '',
      tableFilePath: '',
      tableLine: 0,
      modifyConfigPath: '',
      moduleImplPath: '',
      moduleImplLine: 0,
      handlerImplPaths: {},
      docIndexPath: '',
      docQueryLine: 0,
    },
    steps: {
      openProject: false,
      openSyncIndex: false,
      jumpTable: false,
      jumpModifyConfig: false,
      jumpModuleImpl: false,
      jumpHandlerImpl: false,
      jumpServiceRef: false,
    },
    details: {
      tableJump: {},
      modifyConfigJump: {},
      moduleJump: {},
      handlerJump: {},
      serviceJump: {},
    },
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    screenshot: '',
    error: '',
    pass: false,
  };

  const syncIndexQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'collect/config/sync/index.yml',
    pagination: false,
  });
  const tableFileQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'model/base/collect_doc_important.gen.go',
    pagination: false,
  });
  const modifyConfigQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'collect/config/sync/doc_modify.json',
    pagination: false,
  });
  const docIndexQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'collect/config/doc/index.yml',
    pagination: false,
  });
  const projectQuery = runCurl('webshell.workspace_project_query', {
    project_code: PROJECT_CODE,
    pagination: false,
  });

  summary.expected.syncIndexPath = String((syncIndexQuery.data || [])[0]?.path || '');
  summary.expected.tableFilePath = String((tableFileQuery.data || [])[0]?.path || '');
  summary.expected.modifyConfigPath = String((modifyConfigQuery.data || [])[0]?.path || '');
  summary.expected.docIndexPath = String((docIndexQuery.data || [])[0]?.path || '');
  const projectDir = String((projectQuery.data || [])[0]?.project_dir || '');

  const collectDependencyRoot = normalizePath(path.resolve(projectDir, '../collect'));
  summary.expected.moduleImplPath = `${collectDependencyRoot}/src/collect/service_imp/module_empty.go`;
  const handlerKeys = ['service2field', 'get_modify_data', 'filter_arr', 'params2result'];
  summary.expected.handlerImplPaths = Object.fromEntries(
    handlerKeys.map((key) => [key, `${collectDependencyRoot}/src/collect/service_imp/handler_params_${toSnakeFileToken(key)}.go`]),
  );

  if (
    !summary.expected.syncIndexPath ||
    !summary.expected.tableFilePath ||
    !summary.expected.modifyConfigPath ||
    !summary.expected.docIndexPath ||
    !projectDir
  ) {
    throw new Error('预期文件路径查询失败，无法继续执行用例');
  }

  const tableTextRes = runCurl('webshell.workspace_file_content', {
    project_code: PROJECT_CODE,
    path: summary.expected.tableFilePath,
  });
  const tableText = String(tableTextRes?.data?.content_text || '');
  summary.expected.tableLine = findLineNoByPattern(tableText, /collect_doc_important/);

  const moduleTextRes = runCurl('webshell.workspace_file_content', {
    project_code: PROJECT_CODE,
    path: summary.expected.moduleImplPath,
  });
  const moduleText = String(moduleTextRes?.data?.content_text || '');
  summary.expected.moduleImplLine = findLineNoByPattern(moduleText, /^\s*type\s+EmptyService\s+struct\s*\{/);

  const docIndexTextRes = runCurl('webshell.workspace_file_content', {
    project_code: PROJECT_CODE,
    path: summary.expected.docIndexPath,
  });
  const docIndexText = String(docIndexTextRes?.data?.content_text || '');
  summary.expected.docQueryLine = findLineNoByPattern(docIndexText, /^\s*-\s*key\s*:\s*doc_query\s*$/);

  let browser = null;
  let page = null;
  let fatalError = null;
  try {
    browser = await chromium.launch({ headless: true });
    page = await browser.newPage({ viewport: { width: 1700, height: 980 } });
    page.setDefaultTimeout(22000);

    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        summary.consoleErrors.push(msg.text());
      }
    });
    page.on('pageerror', (err) => summary.pageErrors.push(String(err)));
    page.on('requestfailed', (req) => {
      summary.failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || 'failed'}`);
    });

    await page.goto(PAGE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await sleep(6500);

    const projectBtn = page.getByRole('button', { name: PROJECT_NAME }).first();
    if (await projectBtn.isVisible().catch(() => false)) {
      await projectBtn.click();
      await sleep(1000);
    }
    summary.steps.openProject = true;

    await openFileFromTree(page, 'sync/index.yml', 'index.yml');
    const syncState = await waitForEditorPath(page, summary.expected.syncIndexPath, 22000);
    summary.steps.openSyncIndex = !!syncState;
    if (!syncState) {
      throw new Error('未成功打开 collect/config/sync/index.yml');
    }

    const tableClick = await clickFieldValueInEditor(page, 'table', 'collect_doc_important');
    const tableState = await waitForEditorPath(page, summary.expected.tableFilePath, 22000);
    const tableLineState = summary.expected.tableLine > 0 ? await waitForEditorLine(page, summary.expected.tableLine, 12000) : null;
    summary.steps.jumpTable = !!tableState;
    summary.details.tableJump = {
      clickLine: tableClick.lineNumber,
      clickColumn: tableClick.column,
      clickText: tableClick.lineText,
      ctrlHoverHint: !!tableClick.ctrlHoverHint,
      openedPath: tableState?.currentFilePath || '',
      expectedLine: summary.expected.tableLine,
      reachedLine: tableLineState?.lineNumber || 0,
    };
    if (!tableState) {
      throw new Error('table 跳转未打开 collect_doc_important.gen.go');
    }

    await openFileFromTree(page, 'sync/index.yml', 'index.yml');
    const modifyClick = await clickFieldValueInEditor(page, 'modify_config', 'doc_modify.json');
    const modifyState = await waitForEditorPath(page, summary.expected.modifyConfigPath, 22000);
    summary.steps.jumpModifyConfig = !!modifyState;
    summary.details.modifyConfigJump = {
      clickLine: modifyClick.lineNumber,
      clickColumn: modifyClick.column,
      clickText: modifyClick.lineText,
      ctrlHoverHint: !!modifyClick.ctrlHoverHint,
      openedPath: modifyState?.currentFilePath || '',
    };
    if (!modifyState) {
      throw new Error('modify_config 跳转未打开 doc_modify.json');
    }

    await openFileFromTree(page, 'sync/index.yml', 'index.yml');
    const moduleClick = await clickFieldValueInEditor(page, 'module', 'empty');
    const moduleState = await waitForEditorPath(page, summary.expected.moduleImplPath, 22000);
    const moduleLineState = summary.expected.moduleImplLine > 0 ? await waitForEditorLine(page, summary.expected.moduleImplLine, 12000) : null;
    summary.steps.jumpModuleImpl = !!moduleState;
    summary.details.moduleJump = {
      clickLine: moduleClick.lineNumber,
      clickColumn: moduleClick.column,
      clickText: moduleClick.lineText,
      ctrlHoverHint: !!moduleClick.ctrlHoverHint,
      openedPath: moduleState?.currentFilePath || '',
      expectedLine: summary.expected.moduleImplLine,
      reachedLine: moduleLineState?.lineNumber || 0,
    };
    if (!moduleState) {
      throw new Error('module 跳转未打开 module_empty.go');
    }

    const handlerResults = [];
    for (const key of handlerKeys) {
      const expectedPath = String(summary.expected.handlerImplPaths?.[key] || '');
      if (!expectedPath) {
        throw new Error(`handler 期望路径为空: ${key}`);
      }
      await openFileFromTree(page, 'sync/index.yml', 'index.yml');
      const handlerClick = await clickHandlerKeyInEditor(page, key);
      const handlerState = await waitForEditorPath(page, expectedPath, 22000);
      handlerResults.push({
        key,
        clickLine: handlerClick.lineNumber,
        clickColumn: handlerClick.column,
        clickText: handlerClick.lineText,
        ctrlHoverHint: !!handlerClick.ctrlHoverHint,
        expectedPath,
        openedPath: handlerState?.currentFilePath || '',
        pass: !!handlerState,
      });
      if (!handlerState) {
        throw new Error(`handler key 跳转失败: ${key}`);
      }
    }
    summary.steps.jumpHandlerImpl = handlerResults.every((item) => !!item.pass);
    summary.details.handlerJump = {
      checkedCount: handlerResults.length,
      items: handlerResults,
    };

    await openFileFromTree(page, 'sync/index.yml', 'index.yml');
    const serviceClick = await clickFieldValueInEditor(page, 'service', 'config.doc_query');
    const serviceState = await waitForEditorPath(page, summary.expected.docIndexPath, 22000);
    const serviceLineState = summary.expected.docQueryLine > 0 ? await waitForEditorLine(page, summary.expected.docQueryLine, 12000) : null;
    summary.steps.jumpServiceRef = !!serviceState;
    summary.details.serviceJump = {
      clickLine: serviceClick.lineNumber,
      clickColumn: serviceClick.column,
      clickText: serviceClick.lineText,
      ctrlHoverHint: !!serviceClick.ctrlHoverHint,
      openedPath: serviceState?.currentFilePath || '',
      expectedLine: summary.expected.docQueryLine,
      reachedLine: serviceLineState?.lineNumber || 0,
    };
    if (!serviceState) {
      throw new Error('service 跳转未打开 config/doc/index.yml');
    }

    summary.pass = Object.values(summary.steps).every(Boolean)
      && summary.consoleErrors.length === 0
      && summary.pageErrors.length === 0
      && summary.failedRequests.length === 0;

    summary.screenshot = path.join(OUT_DIR, 'webshell-editor-pool-config-link-jump-go-check.png');
    await page.screenshot({ path: summary.screenshot, fullPage: true });
  } catch (error) {
    fatalError = error;
    summary.error = error?.stack || String(error);
    summary.pass = false;
    if (page) {
      const failPath = path.join(OUT_DIR, 'webshell-editor-pool-config-link-jump-go-check.fail.png');
      summary.screenshot = failPath;
      await page.screenshot({ path: failPath, fullPage: true }).catch(() => undefined);
    }
  } finally {
    if (browser) {
      await browser.close().catch(() => undefined);
    }
    summary.finishedAt = new Date().toISOString();
    const reportPath = path.join(OUT_DIR, 'webshell-editor-pool-config-link-jump-go-check.json');
    fs.writeFileSync(reportPath, JSON.stringify(summary, null, 2));
    console.log(JSON.stringify({ pass: summary.pass, reportPath, screenshot: summary.screenshot, error: summary.error || '' }, null, 2));
    if (fatalError) {
      process.exit(1);
    }
  }
})();
