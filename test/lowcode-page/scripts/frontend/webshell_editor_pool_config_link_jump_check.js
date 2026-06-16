#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
const { chromium } = require('playwright');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://127.0.0.1:8015/template_data/data';
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || 'backend';
const PROJECT_NAME = process.env.WEBSHELL_EDITOR_POOL_PROJECT_NAME || '月神后端';
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

async function waitForEditorPath(page, expectedPath, timeoutMs = 20000) {
  const normalizedExpected = normalizePath(expectedPath);
  const state = await waitUntil(async () => {
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
  return state;
}

async function waitForEditorLine(page, expectedLine, timeoutMs = 20000) {
  const lineNo = Number(expectedLine || 0);
  if (!Number.isFinite(lineNo) || lineNo < 1) {
    return null;
  }
  const state = await waitUntil(async () => {
    const info = await getVisibleEditorState(page);
    if (!info) {
      return null;
    }
    if (Number(info.lineNumber || 0) === lineNo) {
      return info;
    }
    return null;
  }, timeoutMs, 200);
  return state;
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
  await title.waitFor({ state: 'visible', timeout: 15000 });
  await title.click();
  await sleep(1400);
}

async function clickConfigValueInEditor(page, fieldName, expectedValue) {
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
  await sleep(1300);
  return { ...result, ctrlHoverHint };
}

async function clickRequireCallInEditor(page, expectedValue) {
  const result = await page.evaluate(({ expectedValue }) => {
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
    const requireRe = /\brequire\s*\(\s*(?:'([^']+)'|"([^"]+)"|([^)#\s]+))\s*\)/;

    let hitLine = 0;
    let hitColumn = 0;
    let lineText = '';
    for (let i = 1; i <= maxLine; i += 1) {
      const text = String(model.getLineContent?.(i) || '');
      const m = text.match(requireRe);
      if (!m) {
        continue;
      }
      const value = String(m[1] || m[2] || m[3] || '').trim();
      if (!value) {
        continue;
      }
      if (expectedValue && String(expectedValue).trim() && value !== String(expectedValue).trim()) {
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
      return { ok: false, reason: `require target not found: ${expectedValue || ''}` };
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
      requireValue: String(expectedValue || ''),
      uri: decodeURIComponent(String(model?.uri || '')),
    };
  }, { expectedValue });

  if (!result || !result.ok) {
    throw new Error(result?.reason || `click require target failed: ${expectedValue || ''}`);
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
  await sleep(1300);
  return { ...result, ctrlHoverHint };
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

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const now = Date.now();

  const summary = {
    pageUrl: PAGE_URL,
    apiUrl: API_URL,
    projectCode: PROJECT_CODE,
    projectName: PROJECT_NAME,
    startedAt: new Date().toISOString(),
    expected: {
      serviceRouterPath: '',
      checkArrayHandlerPath: '',
      checkArrayClassLine: 0,
      jiraServicePath: '',
      commentsIndexPath: '',
      sqlPath: '',
      issueCommitCountLine: 0,
      blackIndexPath: '',
      blackDataJsonPath: '',
      msgSendLogIndexPath: '',
      msgSendLogSqlPath: '',
      msgSendLogRequirePath: '',
      msgSendRuleIndexPath: '',
      modelFilePath: '',
      modelClassLine: 0,
    },
    steps: {
      openProject: false,
      openServiceRouter: false,
      jumpCheckArrayHandler: false,
      jumpPathToServiceYml: false,
      openCommentsIndex: false,
      jumpSqlFile: false,
      jumpServiceRef: false,
      openBlackIndex: false,
      jumpDataJson: false,
      openMsgSendLogIndex: false,
      jumpMsgSendLogSqlFile: false,
      jumpRequireFile: false,
      openMsgSendRuleIndex: false,
      jumpModelFile: false,
    },
    details: {
      checkArrayJump: {},
      pathJump: {},
      sqlJump: {},
      serviceJump: {},
      dataJsonJump: {},
      requireJump: {},
      modelJump: {},
    },
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    screenshot: '',
    error: '',
    pass: false,
  };

  const serviceRouterQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'service_router.yml',
    pagination: false,
  });
  const jiraServiceQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'jira/service.yml',
    pagination: false,
  });
  const commentsIndexQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'jira/comments/index.yml',
    pagination: false,
  });
  const sqlQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'issue_commit_count.sql',
    pagination: false,
  });
  const blackIndexQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'black_list/black/index.yml',
    pagination: false,
  });
  const blackDataJsonQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'black_list/black/query_black_list.json',
    pagination: false,
  });
  const msgSendLogIndexQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'message/message_send_log/index.yml',
    pagination: false,
  });
  const msgSendLogSqlQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'message/message_send_log/message_send_log_query.sql',
    pagination: false,
  });
  const msgSendLogRequireQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'message/message_send_log/base.sql',
    pagination: false,
  });
  const msgSendRuleIndexQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'message/msg_send_rule/index.yml',
    pagination: false,
  });
  const modelFileQuery = runCurl('webshell.workspace_file_query', {
    project_code: PROJECT_CODE,
    keyword: 'backend/models/models.py',
    pagination: false,
  });
  const workspaceProjectMetaQuery = runCurl('webshell.workspace_project_query', {
    project_code: PROJECT_CODE,
    pagination: false,
  });

  summary.expected.serviceRouterPath = String((serviceRouterQuery.data || [])[0]?.path || '');
  summary.expected.jiraServicePath = String((jiraServiceQuery.data || [])[0]?.path || '');
  summary.expected.commentsIndexPath = String((commentsIndexQuery.data || [])[0]?.path || '');
  summary.expected.sqlPath = String((sqlQuery.data || [])[0]?.path || '');
  summary.expected.blackIndexPath = String((blackIndexQuery.data || [])[0]?.path || '');
  summary.expected.blackDataJsonPath = String((blackDataJsonQuery.data || [])[0]?.path || '');
  summary.expected.msgSendLogIndexPath = String((msgSendLogIndexQuery.data || [])[0]?.path || '');
  summary.expected.msgSendLogSqlPath = String((msgSendLogSqlQuery.data || [])[0]?.path || '');
  summary.expected.msgSendLogRequirePath = String((msgSendLogRequireQuery.data || [])[0]?.path || '');
  summary.expected.msgSendRuleIndexPath = String((msgSendRuleIndexQuery.data || [])[0]?.path || '');
  summary.expected.modelFilePath = String(
    (modelFileQuery.data || []).find((item) => String(item?.path || '').endsWith('/backend/models/models.py'))?.path
      || (modelFileQuery.data || [])[0]?.path
      || '',
  );

  const projectMeta = (workspaceProjectMetaQuery.data || [])[0] || {};
  const fromRouterRoot = String(summary.expected.serviceRouterPath || '').replace(/\/backend_data_service\/service_router\.yml$/i, '');
  const projectRoot = String(fromRouterRoot || projectMeta.project_dir || '').replace(/\/+$/g, '');
  const checkArrayRelativePath = 'collect/service_imp/request_handlers/handlers/check_array.py';
  const checkArrayCandidateList = [];
  const pushCandidate = (p) => {
    const normalized = String(p || '').replace(/\\/g, '/').replace(/\/+/g, '/').trim();
    if (!normalized) {
      return;
    }
    if (!checkArrayCandidateList.includes(normalized)) {
      checkArrayCandidateList.push(normalized);
    }
  };

  const configuredPythonPkgPath = String(projectMeta.python_pkg_path || '').trim();
  if (configuredPythonPkgPath) {
    pushCandidate(`${configuredPythonPkgPath}/${checkArrayRelativePath}`);
    if (!/(?:site-packages|dist-packages)(?:\/|$)/i.test(configuredPythonPkgPath)) {
      pushCandidate(`${configuredPythonPkgPath}/lib/python2.7/site-packages/${checkArrayRelativePath}`);
      pushCandidate(`${configuredPythonPkgPath}/lib/python3/site-packages/${checkArrayRelativePath}`);
    }
  }
  if (projectRoot) {
    pushCandidate(`${projectRoot}/${checkArrayRelativePath}`);
  }

  const startupScriptCandidates = [];
  const pushStartupScript = (p) => {
    const normalized = String(p || '').replace(/\\/g, '/').replace(/\/+/g, '/').trim();
    if (!normalized) {
      return;
    }
    if (!startupScriptCandidates.includes(normalized)) {
      startupScriptCandidates.push(normalized);
    }
  };
  const scriptFieldList = ['startup_script', 'startup_script_path', 'dev_start_script', 'linux_start_script', 'run_script'];
  scriptFieldList.forEach((field) => {
    const v = String(projectMeta[field] || '').trim();
    if (!v) {
      return;
    }
    if (v.startsWith('/')) {
      pushStartupScript(v);
      return;
    }
    if (projectRoot) {
      pushStartupScript(`${projectRoot}/${v}`);
    }
  });
  if (projectRoot) {
    pushStartupScript(`${projectRoot}/linux-start-dev`);
    pushStartupScript(`${projectRoot}/linux-startup`);
    pushStartupScript(`${projectRoot}/linux-start-dev.sh`);
    pushStartupScript(`${projectRoot}/startup.sh`);
  }

  for (const startupScriptPath of startupScriptCandidates) {
    try {
      const startupScriptRes = runCurl('webshell.workspace_file_content', {
        project_code: PROJECT_CODE,
        path: startupScriptPath,
      });
      const startupScriptText = String(startupScriptRes?.data?.content_text || '');
      const pyHomeMatch = startupScriptText.match(/^\s*PY_HOME\s*=\s*(?:"([^"]+)"|'([^']+)'|([^\s#]+))/m);
      const pyHome = String(pyHomeMatch?.[1] || pyHomeMatch?.[2] || pyHomeMatch?.[3] || '').trim();
      if (!pyHome) {
        continue;
      }
      pushCandidate(`${pyHome}/lib/python2.7/site-packages/${checkArrayRelativePath}`);
      pushCandidate(`${pyHome}/lib/python3/site-packages/${checkArrayRelativePath}`);
      pushCandidate(`${pyHome}/${checkArrayRelativePath}`);
    } catch (_error) {
      // ignore and continue
    }
  }

  let checkArrayResolvedText = '';
  for (const candidatePath of checkArrayCandidateList) {
    try {
      const checkArrayRes = runCurl('webshell.workspace_file_content', {
        project_code: PROJECT_CODE,
        path: candidatePath,
      });
      const contentText = String(checkArrayRes?.data?.content_text || '');
      if (!contentText) {
        continue;
      }
      summary.expected.checkArrayHandlerPath = candidatePath;
      checkArrayResolvedText = contentText;
      break;
    } catch (_error) {
      // ignore and try next path
    }
  }

  if (
    !summary.expected.serviceRouterPath ||
    !summary.expected.checkArrayHandlerPath ||
    !summary.expected.jiraServicePath ||
    !summary.expected.commentsIndexPath ||
    !summary.expected.sqlPath ||
    !summary.expected.blackIndexPath ||
    !summary.expected.blackDataJsonPath ||
    !summary.expected.msgSendLogIndexPath ||
    !summary.expected.msgSendLogSqlPath ||
    !summary.expected.msgSendLogRequirePath ||
    !summary.expected.msgSendRuleIndexPath ||
    !summary.expected.modelFilePath
  ) {
    throw new Error('预期文件路径查询失败，无法继续执行用例');
  }

  const commentsTextRes = runCurl('webshell.workspace_file_content', {
    project_code: PROJECT_CODE,
    path: summary.expected.commentsIndexPath,
  });
  const commentsText = String(commentsTextRes?.data?.content_text || '');
  summary.expected.issueCommitCountLine = findLineNoByPattern(commentsText, /^\s*-\s*key\s*:\s*issue_commit_count\s*$/);
  const modelTextRes = runCurl('webshell.workspace_file_content', {
    project_code: PROJECT_CODE,
    path: summary.expected.modelFilePath,
  });
  const modelText = String(modelTextRes?.data?.content_text || '');
  summary.expected.modelClassLine = findLineNoByPattern(modelText, /^\s*class\s+MsgSendRule\s*(?:\(|:)/);
  summary.expected.checkArrayClassLine = findLineNoByPattern(checkArrayResolvedText, /^\s*class\s+CheckArray\s*(?:\(|:)/);

  let browser = null;
  let page = null;
  let fatalError = null;
  try {
    browser = await chromium.launch({ headless: true });
    page = await browser.newPage({ viewport: { width: 1700, height: 980 } });
    page.setDefaultTimeout(20000);

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
    await sleep(6000);

    const projectBtn = page.getByRole('button', { name: PROJECT_NAME }).first();
    if (await projectBtn.isVisible().catch(() => false)) {
      await projectBtn.click();
      await sleep(1000);
    }
    summary.steps.openProject = true;

    await openFileFromTree(page, 'service_router.yml', 'service_router.yml');
    const routerState = await waitForEditorPath(page, summary.expected.serviceRouterPath, 20000);
    summary.steps.openServiceRouter = !!routerState;
    if (!routerState) {
      throw new Error('未成功打开 service_router.yml');
    }

    const checkArrayClick = await clickConfigValueInEditor(page, 'key', 'check_array');
    const checkArrayState = await waitForEditorPath(page, summary.expected.checkArrayHandlerPath, 22000);
    const checkArrayLineState = summary.expected.checkArrayClassLine > 0
      ? await waitForEditorLine(page, summary.expected.checkArrayClassLine, 12000)
      : null;
    const checkArrayFinalState = checkArrayLineState || await getVisibleEditorState(page);
    const checkArrayLineMatched = summary.expected.checkArrayClassLine > 0
      ? Number(checkArrayFinalState?.lineNumber || 0) === Number(summary.expected.checkArrayClassLine || 0)
      : true;
    summary.steps.jumpCheckArrayHandler = !!checkArrayState && checkArrayLineMatched;
    summary.details.checkArrayJump = {
      clickLine: checkArrayClick.lineNumber,
      clickColumn: checkArrayClick.column,
      clickText: checkArrayClick.lineText,
      ctrlHoverHint: !!checkArrayClick.ctrlHoverHint,
      openedPath: checkArrayState?.currentFilePath || '',
      expectedLine: summary.expected.checkArrayClassLine,
      finalLine: Number(checkArrayFinalState?.lineNumber || 0),
    };
    if (!summary.steps.jumpCheckArrayHandler) {
      throw new Error(`check_array 跳转失败: expected ${summary.expected.checkArrayHandlerPath} line ${summary.expected.checkArrayClassLine}, actual ${Number(checkArrayFinalState?.lineNumber || 0)}`);
    }

    await openFileFromTree(page, 'service_router.yml', 'service_router.yml');
    const routerState2 = await waitForEditorPath(page, summary.expected.serviceRouterPath, 20000);
    if (!routerState2) {
      throw new Error('check_array 跳转后重新打开 service_router.yml 失败');
    }

    const pathClick = await clickConfigValueInEditor(page, 'path', 'jira/service.yml');
    const serviceState = await waitForEditorPath(page, summary.expected.jiraServicePath, 20000);
    summary.steps.jumpPathToServiceYml = !!serviceState;
    summary.details.pathJump = {
      clickLine: pathClick.lineNumber,
      clickColumn: pathClick.column,
      clickText: pathClick.lineText,
      ctrlHoverHint: !!pathClick.ctrlHoverHint,
      openedPath: serviceState?.currentFilePath || '',
    };
    if (!serviceState) {
      throw new Error('path 跳转未打开 jira/service.yml');
    }

    await openFileFromTree(page, 'comments/index.yml', 'index.yml');
    const commentsState = await waitForEditorPath(page, summary.expected.commentsIndexPath, 20000);
    summary.steps.openCommentsIndex = !!commentsState;
    if (!commentsState) {
      throw new Error('未成功打开 jira/comments/index.yml');
    }

    const sqlClick = await clickConfigValueInEditor(page, 'sql_file', 'issue_commit_count.sql');
    const sqlState = await waitForEditorPath(page, summary.expected.sqlPath, 20000);
    summary.steps.jumpSqlFile = !!sqlState;
    summary.details.sqlJump = {
      clickLine: sqlClick.lineNumber,
      clickColumn: sqlClick.column,
      clickText: sqlClick.lineText,
      ctrlHoverHint: !!sqlClick.ctrlHoverHint,
      openedPath: sqlState?.currentFilePath || '',
    };
    if (!sqlState) {
      throw new Error('sql_file 跳转未打开 issue_commit_count.sql');
    }

    await openFileFromTree(page, 'comments/index.yml', 'index.yml');
    const commentsState2 = await waitForEditorPath(page, summary.expected.commentsIndexPath, 20000);
    if (!commentsState2) {
      throw new Error('重新打开 comments/index.yml 失败');
    }

    const serviceClick = await clickConfigValueInEditor(page, 'service', 'jira.issue_commit_count');
    const serviceRefState = await waitForEditorPath(page, summary.expected.commentsIndexPath, 20000);
    const lineState = await waitForEditorLine(page, summary.expected.issueCommitCountLine, 20000);
    const finalState = lineState || await getVisibleEditorState(page);
    const lineMatched = Number(finalState?.lineNumber || 0) === Number(summary.expected.issueCommitCountLine || 0);
    summary.steps.jumpServiceRef = !!serviceRefState && lineMatched;
    summary.details.serviceJump = {
      clickLine: serviceClick.lineNumber,
      clickColumn: serviceClick.column,
      clickText: serviceClick.lineText,
      ctrlHoverHint: !!serviceClick.ctrlHoverHint,
      expectedLine: summary.expected.issueCommitCountLine,
      finalLine: Number(finalState?.lineNumber || 0),
      openedPath: serviceRefState?.currentFilePath || '',
    };
    if (!summary.steps.jumpServiceRef) {
      throw new Error(`service 跳转校验失败: expected line ${summary.expected.issueCommitCountLine}, actual ${Number(finalState?.lineNumber || 0)}`);
    }

    await openFileFromTree(page, 'black_list/black/index.yml', 'index.yml');
    const blackState = await waitForEditorPath(page, summary.expected.blackIndexPath, 20000);
    summary.steps.openBlackIndex = !!blackState;
    if (!blackState) {
      throw new Error('未成功打开 black_list/black/index.yml');
    }

    const dataJsonClick = await clickConfigValueInEditor(page, 'data_json', 'query_black_list.json');
    const dataJsonState = await waitForEditorPath(page, summary.expected.blackDataJsonPath, 20000);
    summary.steps.jumpDataJson = !!dataJsonState;
    summary.details.dataJsonJump = {
      clickLine: dataJsonClick.lineNumber,
      clickColumn: dataJsonClick.column,
      clickText: dataJsonClick.lineText,
      ctrlHoverHint: !!dataJsonClick.ctrlHoverHint,
      openedPath: dataJsonState?.currentFilePath || '',
    };
    if (!dataJsonState) {
      throw new Error('data_json 跳转未打开 query_black_list.json');
    }

    await openFileFromTree(page, 'message/message_send_log/index.yml', 'index.yml');
    const msgSendLogIndexState = await waitForEditorPath(page, summary.expected.msgSendLogIndexPath, 20000);
    summary.steps.openMsgSendLogIndex = !!msgSendLogIndexState;
    if (!msgSendLogIndexState) {
      throw new Error('未成功打开 message/message_send_log/index.yml');
    }

    const msgSendLogSqlClick = await clickConfigValueInEditor(page, 'sql_file', 'message_send_log_query.sql');
    const msgSendLogSqlState = await waitForEditorPath(page, summary.expected.msgSendLogSqlPath, 20000);
    summary.steps.jumpMsgSendLogSqlFile = !!msgSendLogSqlState;
    summary.details.requireJump = {
      sqlClickLine: msgSendLogSqlClick.lineNumber,
      sqlClickColumn: msgSendLogSqlClick.column,
      sqlClickText: msgSendLogSqlClick.lineText,
      sqlCtrlHoverHint: !!msgSendLogSqlClick.ctrlHoverHint,
      openedSqlPath: msgSendLogSqlState?.currentFilePath || '',
    };
    if (!msgSendLogSqlState) {
      throw new Error('message_send_log_query.sql 跳转失败');
    }

    const requireClick = await clickRequireCallInEditor(page, './base.sql');
    const requireState = await waitForEditorPath(page, summary.expected.msgSendLogRequirePath, 20000);
    summary.steps.jumpRequireFile = !!requireState;
    summary.details.requireJump = {
      ...summary.details.requireJump,
      requireClickLine: requireClick.lineNumber,
      requireClickColumn: requireClick.column,
      requireClickText: requireClick.lineText,
      requireCtrlHoverHint: !!requireClick.ctrlHoverHint,
      openedRequirePath: requireState?.currentFilePath || '',
    };
    if (!requireState) {
      throw new Error('require("./base.sql") 跳转失败');
    }

    await openFileFromTree(page, 'message/msg_send_rule/index.yml', 'index.yml');
    const msgSendRuleIndexState = await waitForEditorPath(page, summary.expected.msgSendRuleIndexPath, 20000);
    summary.steps.openMsgSendRuleIndex = !!msgSendRuleIndexState;
    if (!msgSendRuleIndexState) {
      throw new Error('未成功打开 message/msg_send_rule/index.yml');
    }

    const modelClick = await clickConfigValueInEditor(page, 'model', 'MsgSendRule');
    const modelPathState = await waitForEditorPath(page, summary.expected.modelFilePath, 20000);
    const modelLineState = await waitForEditorLine(page, summary.expected.modelClassLine, 20000);
    const modelFinalState = modelLineState || await getVisibleEditorState(page);
    const modelLineMatched = Number(modelFinalState?.lineNumber || 0) === Number(summary.expected.modelClassLine || 0);
    summary.steps.jumpModelFile = !!modelPathState && modelLineMatched;
    summary.details.modelJump = {
      clickLine: modelClick.lineNumber,
      clickColumn: modelClick.column,
      clickText: modelClick.lineText,
      ctrlHoverHint: !!modelClick.ctrlHoverHint,
      expectedLine: summary.expected.modelClassLine,
      finalLine: Number(modelFinalState?.lineNumber || 0),
      openedPath: modelPathState?.currentFilePath || '',
    };
    if (!summary.steps.jumpModelFile) {
      throw new Error(`model 跳转校验失败: expected line ${summary.expected.modelClassLine}, actual ${Number(modelFinalState?.lineNumber || 0)}`);
    }

    const shot = path.join(OUT_DIR, 'webshell-editor-pool-config-link-jump-check.png');
    await page.screenshot({ path: shot, fullPage: true });
    summary.screenshot = shot;
  } catch (error) {
    fatalError = error;
    summary.error = String(error?.message || error || '');
    if (page && !summary.screenshot) {
      const failShot = path.join(OUT_DIR, 'webshell-editor-pool-config-link-jump-check.fail.png');
      try {
        await page.screenshot({ path: failShot, fullPage: true });
        summary.screenshot = failShot;
      } catch (_error) {
        // ignore
      }
    }
  } finally {
    summary.finishedAt = new Date().toISOString();
    if (browser) {
      await browser.close().catch(() => undefined);
    }
    if (fatalError) {
      console.error('[config-link-jump-check] failed:', fatalError?.stack || fatalError?.message || String(fatalError));
    }
  }

  summary.pass = [
    summary.steps.openProject,
    summary.steps.openServiceRouter,
    summary.steps.jumpCheckArrayHandler,
    summary.steps.jumpPathToServiceYml,
    summary.steps.openCommentsIndex,
    summary.steps.jumpSqlFile,
    summary.steps.jumpServiceRef,
    summary.steps.openBlackIndex,
    summary.steps.jumpDataJson,
    summary.steps.openMsgSendLogIndex,
    summary.steps.jumpMsgSendLogSqlFile,
    summary.steps.jumpRequireFile,
    summary.steps.openMsgSendRuleIndex,
    summary.steps.jumpModelFile,
    summary.consoleErrors.length === 0,
    summary.pageErrors.length === 0,
    summary.failedRequests.length === 0,
  ].every(Boolean);

  const outFile = path.join(OUT_DIR, 'webshell-editor-pool-config-link-jump-check.json');
  fs.writeFileSync(outFile, `${JSON.stringify(summary, null, 2)}\n`, 'utf8');

  console.log(JSON.stringify({
    pass: summary.pass,
    outFile,
    screenshot: summary.screenshot,
    steps: summary.steps,
    details: summary.details,
    consoleErrors: summary.consoleErrors.length,
    pageErrors: summary.pageErrors.length,
    failedRequests: summary.failedRequests.length,
  }, null, 2));

  process.exit(summary.pass ? 0 : 1);
})().catch((error) => {
  console.error('[config-link-jump-check] failed:', error?.stack || error?.message || String(error));
  process.exit(1);
});
