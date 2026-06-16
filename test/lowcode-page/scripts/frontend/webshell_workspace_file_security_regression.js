#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://127.0.0.1:8015/template_data/data';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || 'test';
const PROJECT_DIR = process.env.WEBSHELL_EDITOR_POOL_PROJECT_DIR || '/data/project/test';

function parseJson(raw, service) {
  try {
    return JSON.parse(String(raw || '{}'));
  } catch (error) {
    throw new Error(`parse response failed (${service}): ${error.message}`);
  }
}

function runCurlRaw(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  const res = spawnSync('curl', [
    '--noproxy', '*',
    '-sS',
    `${API_URL}?service=${service}`,
    '-H', 'Content-Type: application/json',
    '--data', payload,
  ], { encoding: 'utf8' });
  if (res.status !== 0) {
    throw new Error(res.stderr || `curl failed: ${service}`);
  }
  return parseJson(res.stdout, service);
}

function runCurl(service, data) {
  const parsed = runCurlRaw(service, data);
  if (!parsed || String(parsed.code || '') !== '0' || parsed.success === false) {
    throw new Error(`${service} failed: ${parsed?.msg || 'unknown error'}`);
  }
  return parsed;
}

function runCurlAllowFail(service, data) {
  try {
    const parsed = runCurlRaw(service, data);
    const ok = parsed && String(parsed.code || '') === '0' && parsed.success !== false;
    return {
      ok,
      code: String(parsed?.code || ''),
      msg: String(parsed?.msg || ''),
      data: parsed?.data,
    };
  } catch (error) {
    return {
      ok: false,
      code: 'transport_error',
      msg: String(error && error.message ? error.message : error),
      data: null,
    };
  }
}

function expectFailure(result, pattern, label) {
  if (result.ok) {
    throw new Error(`${label} unexpectedly succeeded`);
  }
  if (pattern && !pattern.test(String(result.msg || ''))) {
    throw new Error(`${label} failed with unexpected message: ${result.msg}`);
  }
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const ts = Date.now();
  const sandboxDir = path.posix.join(PROJECT_DIR, 'test');
  const safeFilePath = path.posix.join(sandboxDir, `security_${ts}.txt`);
  const renameSourcePath = path.posix.join(sandboxDir, `rename_${ts}.txt`);
  const renameTargetPath = `/tmp/workspace_security_${ts}.txt`;
  const traversalPath = `${sandboxDir}/../../etc/passwd`;
  const outsideFilePath = '/etc/passwd';
  const outsideSavePath = `/tmp/workspace_security_write_${ts}.txt`;
  const outsideDirPath = `/tmp/workspace_security_dir_${ts}`;

  const result = {
    apiUrl: API_URL,
    projectCode: PROJECT_CODE,
    projectDir: PROJECT_DIR,
    sandboxDir,
    safeFilePath,
    renameSourcePath,
    checks: {
      setupSandboxDir: false,
      saveInsideProject: false,
      readInsideProject: false,
      contentTraversalBlocked: false,
      contentOutsideBlocked: false,
      batchOutsideBlocked: false,
      saveTraversalBlocked: false,
      saveOutsideBlocked: false,
      addTraversalBlocked: false,
      addOutsideBlocked: false,
      renameOutsideBlocked: false,
      deleteTraversalBlocked: false,
      deleteOutsideBlocked: false,
      safeFileStillReadable: false,
    },
    details: {},
    pass: false,
  };

  runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
    project_code: PROJECT_CODE,
    path: sandboxDir,
  });

  try {
    runCurl('webshell.workspace_file_add_with_sync', {
      project_code: PROJECT_CODE,
      name: 'test',
      path: sandboxDir,
      is_dir: '1',
      parent_id: '',
    });
    result.checks.setupSandboxDir = true;

    const saveInside = runCurl('webshell.workspace_file_save', {
      project_code: PROJECT_CODE,
      path: safeFilePath,
      content: `security-${ts}`,
    });
    result.details.saveInside = saveInside.data || {};
    result.checks.saveInsideProject = String(saveInside?.data?.path || '') === safeFilePath;

    const readInside = runCurl('webshell.workspace_file_content', {
      project_code: PROJECT_CODE,
      path: safeFilePath,
      max_bytes: 1024,
    });
    result.details.readInside = readInside.data || {};
    result.checks.readInsideProject = String(readInside?.data?.content_text || '') === `security-${ts}`;

    const contentTraversal = runCurlAllowFail('webshell.workspace_file_content', {
      project_code: PROJECT_CODE,
      path: traversalPath,
      max_bytes: 1024,
    });
    result.details.contentTraversal = contentTraversal;
    expectFailure(contentTraversal, /非法路径穿越|超出项目目录/, 'content traversal');
    result.checks.contentTraversalBlocked = true;

    const contentOutside = runCurlAllowFail('webshell.workspace_file_content', {
      project_code: PROJECT_CODE,
      path: outsideFilePath,
      max_bytes: 1024,
    });
    result.details.contentOutside = contentOutside;
    expectFailure(contentOutside, /超出项目目录/, 'content outside');
    result.checks.contentOutsideBlocked = true;

    const batchOutside = runCurlAllowFail('webshell.workspace_file_content_batch', {
      project_code: PROJECT_CODE,
      paths: [safeFilePath, outsideFilePath],
      max_bytes: 1024,
    });
    result.details.batchOutside = batchOutside;
    expectFailure(batchOutside, /超出项目目录/, 'content batch outside');
    result.checks.batchOutsideBlocked = true;

    const saveTraversal = runCurlAllowFail('webshell.workspace_file_save', {
      project_code: PROJECT_CODE,
      path: traversalPath,
      content: 'blocked',
    });
    result.details.saveTraversal = saveTraversal;
    expectFailure(saveTraversal, /非法路径穿越|超出项目目录/, 'save traversal');
    result.checks.saveTraversalBlocked = true;

    const saveOutside = runCurlAllowFail('webshell.workspace_file_save', {
      project_code: PROJECT_CODE,
      path: outsideSavePath,
      content: 'blocked',
    });
    result.details.saveOutside = saveOutside;
    expectFailure(saveOutside, /超出项目目录/, 'save outside');
    result.checks.saveOutsideBlocked = true;

    const addTraversal = runCurlAllowFail('webshell.workspace_file_add_with_sync', {
      project_code: PROJECT_CODE,
      name: 'blocked_dir',
      path: traversalPath,
      is_dir: '1',
      parent_id: '',
    });
    result.details.addTraversal = addTraversal;
    expectFailure(addTraversal, /非法路径穿越|超出项目目录/, 'add traversal');
    result.checks.addTraversalBlocked = true;

    const addOutside = runCurlAllowFail('webshell.workspace_file_add_with_sync', {
      project_code: PROJECT_CODE,
      name: path.posix.basename(outsideDirPath),
      path: outsideDirPath,
      is_dir: '1',
      parent_id: '',
    });
    result.details.addOutside = addOutside;
    expectFailure(addOutside, /超出项目目录/, 'add outside');
    result.checks.addOutsideBlocked = true;

    runCurl('webshell.workspace_file_add_with_sync', {
      project_code: PROJECT_CODE,
      name: path.posix.basename(renameSourcePath),
      path: renameSourcePath,
      is_dir: '0',
      parent_id: '',
    });
    const queryRes = runCurl('webshell.workspace_file_query', {
      project_code: PROJECT_CODE,
      path_exact: renameSourcePath,
      pagination: false,
    });
    const record = Array.isArray(queryRes.data) ? queryRes.data[0] : null;
    if (!record || !record.file_id) {
      throw new Error('rename source file record not found');
    }

    const renameOutside = runCurlAllowFail('webshell.workspace_file_update_with_sync', {
      project_code: PROJECT_CODE,
      file_id: record.file_id,
      name: path.posix.basename(renameTargetPath),
      path: renameTargetPath,
      is_dir: '0',
      parent_id: '',
    });
    result.details.renameOutside = renameOutside;
    expectFailure(renameOutside, /超出项目目录/, 'rename outside');
    result.checks.renameOutsideBlocked = true;

    const deleteTraversal = runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
      project_code: PROJECT_CODE,
      path: traversalPath,
    });
    result.details.deleteTraversal = deleteTraversal;
    expectFailure(deleteTraversal, /非法路径穿越|超出项目目录/, 'delete traversal');
    result.checks.deleteTraversalBlocked = true;

    const deleteOutside = runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
      project_code: PROJECT_CODE,
      path: outsideFilePath,
    });
    result.details.deleteOutside = deleteOutside;
    expectFailure(deleteOutside, /超出项目目录/, 'delete outside');
    result.checks.deleteOutsideBlocked = true;

    const safeReadAgain = runCurl('webshell.workspace_file_content', {
      project_code: PROJECT_CODE,
      path: safeFilePath,
      max_bytes: 1024,
    });
    result.details.safeReadAgain = safeReadAgain.data || {};
    result.checks.safeFileStillReadable = String(safeReadAgain?.data?.content_text || '') === `security-${ts}`;

    result.pass = Object.values(result.checks).every(Boolean);
  } catch (error) {
    result.error = String(error && error.message ? error.message : error);
  } finally {
    runCurlAllowFail('webshell.workspace_file_delete_with_sync', {
      project_code: PROJECT_CODE,
      path: sandboxDir,
    });
  }

  const outFile = path.join(OUT_DIR, 'webshell-workspace-file-security-regression.json');
  fs.writeFileSync(outFile, JSON.stringify(result, null, 2));
  console.log(JSON.stringify({ pass: result.pass, outFile, error: result.error || '' }, null, 2));
  process.exit(result.pass ? 0 : 1);
})();
