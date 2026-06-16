#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawn, spawnSync } = require('child_process');
const { chromium } = require('playwright');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://127.0.0.1:8015/template_data/data';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || 'test';
const PROJECT_DIR = process.env.WEBSHELL_EDITOR_POOL_PROJECT_DIR || '/data/project/test';
const ROUNDS = Number(process.env.WEBSHELL_FILE_CRUD_ROUNDS || 3);

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function parseResponse(raw, service) {
  let parsed = {};
  try {
    parsed = JSON.parse(String(raw || '{}'));
  } catch (error) {
    throw new Error(`parse response failed (${service}): ${error.message}`);
  }
  if (!parsed || String(parsed.code || '') !== '0' || parsed.success === false) {
    throw new Error(`${service} failed: ${parsed?.msg || 'unknown error'}`);
  }
  return parsed;
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
  return parseResponse(res.stdout, service);
}

function runCurlAllowFail(service, data) {
  try {
    return { ok: true, value: runCurl(service, data), error: '' };
  } catch (error) {
    return { ok: false, value: null, error: String(error && error.message ? error.message : error) };
  }
}

function runCurlAsync(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  return new Promise((resolve) => {
    const child = spawn('curl', [
      '--noproxy', '*',
      '-sS',
      `${API_URL}?service=${service}`,
      '-H',
      'Content-Type: application/json',
      '--data',
      payload,
    ], { stdio: ['ignore', 'pipe', 'pipe'] });
    let out = '';
    let err = '';
    child.stdout.on('data', (chunk) => { out += chunk.toString(); });
    child.stderr.on('data', (chunk) => { err += chunk.toString(); });
    child.on('close', (code) => {
      if (code !== 0) {
        resolve({ ok: false, code, error: err || `curl failed: ${service}` });
        return;
      }
      try {
        const parsed = JSON.parse(String(out || '{}'));
        const ok = parsed && String(parsed.code || '') === '0' && parsed.success !== false;
        resolve({ ok, parsed, error: ok ? '' : String(parsed?.msg || 'unknown error') });
      } catch (parseError) {
        resolve({ ok: false, error: `parse response failed (${service}): ${parseError.message}` });
      }
    });
  });
}

function getFirstRecordByPath(projectCode, targetPath) {
  const resp = runCurl('webshell.workspace_file_query', {
    project_code: projectCode,
    path_exact: targetPath,
    pagination: false,
  });
  const list = Array.isArray(resp.data) ? resp.data : [];
  return list[0] || null;
}

async function pageSmokeCheck() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1600, height: 900 } });
  const consoleErrors = [];
  const pageErrors = [];
  const failedRequests = [];

  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      consoleErrors.push(msg.text());
    }
  });
  page.on('pageerror', (error) => pageErrors.push(String(error)));
  page.on('requestfailed', (req) => failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || 'failed'}`));

  await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 60000 });
  await sleep(2500);
  const shot = path.join(OUT_DIR, 'webshell-workspace-file-crud-api-regression-smoke.png');
  await page.screenshot({ path: shot, fullPage: true });
  await browser.close();

  return { consoleErrors, pageErrors, failedRequests, screenshot: shot };
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const rootDirPath = path.posix.join(PROJECT_DIR, 'test');
  const result = {
    pageUrl: PAGE_URL,
    apiUrl: API_URL,
    projectCode: PROJECT_CODE,
    rounds: ROUNDS,
    rootDirPath,
    roundResults: [],
    lockCheck: {
      ok: false,
      details: {},
    },
    smoke: {
      screenshot: '',
      consoleErrors: [],
      pageErrors: [],
      failedRequests: [],
    },
    pass: false,
  };

  // Ensure deterministic start
  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: rootDirPath,
  });

  try {
    const projectResp = runCurl('webshell.workspace_project_query', {
      project_code: PROJECT_CODE,
      pagination: false,
    });
    const project = Array.isArray(projectResp.data) ? projectResp.data[0] : null;
    if (!project || !project.webshell_workspace_project_id) {
      throw new Error(`project not found: ${PROJECT_CODE}`);
    }

    for (let i = 0; i < ROUNDS; i += 1) {
      const ts = Date.now() + i;
      const level1Name = `lvl1_${ts}`;
      const level2Name = `lvl2_${ts}`;
      const fileName = `case_${ts}.txt`;
      const renamedFileName = `case_${ts}_renamed.txt`;

      const level1Path = path.posix.join(rootDirPath, level1Name);
      const level2Path = path.posix.join(level1Path, level2Name);
      const filePath = path.posix.join(level2Path, fileName);
      const renamedFilePath = path.posix.join(level2Path, renamedFileName);

      const item = {
        round: i + 1,
        paths: { level1Path, level2Path, filePath, renamedFilePath },
        checks: {
          addRoot: false,
          addL1: false,
          addL2: false,
          addFile: false,
          serverFileCreated: false,
          rename: false,
          serverFileRenamed: false,
          deleteFile: false,
          serverFileDeleted: false,
          multiLevelVisibleInDb: false,
          cleanupRoot: false,
        },
        errors: [],
      };

      try {
        runCurl('webshell.workspace_file_add_with_sync', {
          project_code: PROJECT_CODE,
          name: 'test',
          path: rootDirPath,
          is_dir: '1',
          parent_id: '',
        });
        item.checks.addRoot = true;

        runCurl('webshell.workspace_file_add_with_sync', {
          project_code: PROJECT_CODE,
          name: level1Name,
          path: level1Path,
          is_dir: '1',
          parent_id: '',
        });
        item.checks.addL1 = true;

        runCurl('webshell.workspace_file_add_with_sync', {
          project_code: PROJECT_CODE,
          name: level2Name,
          path: level2Path,
          is_dir: '1',
          parent_id: '',
        });
        item.checks.addL2 = true;

        runCurl('webshell.workspace_file_add_with_sync', {
          project_code: PROJECT_CODE,
          name: fileName,
          path: filePath,
          is_dir: '0',
          parent_id: '',
        });
        item.checks.addFile = true;

        const fileContent = runCurl('webshell.workspace_file_content', {
          project_code: PROJECT_CODE,
          path: filePath,
        });
        item.checks.serverFileCreated = String(fileContent?.data?.path || '') === filePath;

        const fileRow = getFirstRecordByPath(PROJECT_CODE, filePath);
        if (!fileRow || !fileRow.file_id) {
          throw new Error(`cannot find file row by path: ${filePath}`);
        }

        runCurl('webshell.workspace_file_update_with_sync', {
          project_code: PROJECT_CODE,
          file_id: fileRow.file_id,
          name: renamedFileName,
          path: renamedFilePath,
          is_dir: '0',
          parent_id: fileRow.parent_id || '',
        });
        item.checks.rename = true;

        const renamedContent = runCurl('webshell.workspace_file_content', {
          project_code: PROJECT_CODE,
          path: renamedFilePath,
        });
        item.checks.serverFileRenamed = String(renamedContent?.data?.path || '') === renamedFilePath;

        const oldFileRead = runCurlAllowFail('webshell.workspace_file_content', {
          project_code: PROJECT_CODE,
          path: filePath,
        });
        item.checks.serverFileRenamed = item.checks.serverFileRenamed && !oldFileRead.ok;

        runCurl('webshell.workspace_file_delete_with_sync', {
          project_code: PROJECT_CODE,
          path: renamedFilePath,
        });
        item.checks.deleteFile = true;

        const deletedRead = runCurlAllowFail('webshell.workspace_file_content', {
          project_code: PROJECT_CODE,
          path: renamedFilePath,
        });
        item.checks.serverFileDeleted = !deletedRead.ok;

        const level1Row = getFirstRecordByPath(PROJECT_CODE, level1Path);
        const level2Row = getFirstRecordByPath(PROJECT_CODE, level2Path);
        item.checks.multiLevelVisibleInDb = !!level1Row && !!level2Row;

        runCurl('webshell.workspace_file_delete_with_sync', {
          project_code: PROJECT_CODE,
          path: rootDirPath,
        });
        const rootAfter = getFirstRecordByPath(PROJECT_CODE, rootDirPath);
        item.checks.cleanupRoot = !rootAfter;
      } catch (error) {
        item.errors.push(String(error && error.message ? error.message : error));
      }

      result.roundResults.push(item);
    }

    // lock check (parallel sync)
    const syncPayload = {
      project_code: PROJECT_CODE,
      webshell_workspace_project_id: project.webshell_workspace_project_id,
      exclude_dirs: project.exclude_dirs || 'node_modules,.git,dist,.next',
    };
    const [s1, s2] = await Promise.all([
      runCurlAsync('webshell.workspace_project_sync_files', syncPayload),
      runCurlAsync('webshell.workspace_project_sync_files', syncPayload),
    ]);

    const secondMsg = String(s1.ok ? (s2.error || s2.parsed?.msg || '') : (s1.error || s1.parsed?.msg || ''));
    const oneFailedWithLock = (!s1.ok || !s2.ok) && /同步任务正在执行中|lock=/i.test(secondMsg);
    result.lockCheck = {
      ok: oneFailedWithLock,
      details: { first: s1, second: s2, secondMsg },
    };

    result.smoke = await pageSmokeCheck();
  } catch (error) {
    result.error = String(error && error.stack ? error.stack : error);
  }

  // Final cleanup
  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: rootDirPath,
  });

  const roundsPass = result.roundResults.every((item) => {
    const c = item.checks || {};
    return c.addRoot && c.addL1 && c.addL2 && c.addFile && c.serverFileCreated && c.rename && c.serverFileRenamed && c.deleteFile && c.serverFileDeleted && c.multiLevelVisibleInDb && c.cleanupRoot && (!item.errors || item.errors.length === 0);
  });

  result.pass =
    roundsPass &&
    result.lockCheck.ok &&
    result.smoke.consoleErrors.length === 0 &&
    result.smoke.pageErrors.length === 0;

  const out = path.join(OUT_DIR, 'webshell-workspace-file-crud-api-regression.json');
  fs.writeFileSync(out, JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));

  if (!result.pass) {
    process.exit(2);
  }
})();
