#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://192.168.232.130:8015/template_data/data';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';
const PROJECT_A = process.env.WEBSHELL_EDITOR_POOL_PROJECT_A || 'backend';
const PROJECT_B = process.env.WEBSHELL_EDITOR_POOL_PROJECT_B || 'collect-ui';

function runCurl(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  const res = spawnSync('curl', [
    '-s',
    `${API_URL}?service=${service}`,
    '-H',
    'Content-Type: application/json',
    '--data',
    payload,
  ], { encoding: 'utf8' });
  if (res.status !== 0) {
    throw new Error(res.stderr || `curl failed: ${service}`);
  }
  const out = JSON.parse(String(res.stdout || '{}'));
  if (!out || String(out.code || '') !== '0' || out.success === false) {
    throw new Error(`${service} failed: ${out?.msg || 'unknown error'}`);
  }
  return out;
}

function normalizeId(raw) {
  const value = String(raw || '').trim();
  if (!value) return '';
  if (value.startsWith('g_') || value.startsWith('d_')) return value.slice(2);
  return value;
}

function isDir(node) {
  if (!node) return false;
  if (String(node.node_type || '').toLowerCase() === 'service') return true;
  return node.is_dir === '1' || node.is_dir === 1 || node.is_dir === true;
}

function findGroupIdByTitle(tree, title) {
  let hit = '';
  const walk = (nodes) => {
    if (hit || !Array.isArray(nodes)) return;
    for (const node of nodes) {
      if (!node || !isDir(node)) continue;
      const t = String(node.title || node.display_title || node.name || '').trim();
      if (t === title) {
        hit = normalizeId(node.doc_group_id || node.id || '');
        return;
      }
      walk(Array.isArray(node.children) ? node.children : []);
    }
  };
  walk(Array.isArray(tree) ? tree : []);
  return hit;
}

function findDocIdByTitle(tree, title) {
  let hit = '';
  const walk = (nodes) => {
    if (hit || !Array.isArray(nodes)) return;
    for (const node of nodes) {
      if (!node) continue;
      if (isDir(node)) {
        walk(Array.isArray(node.children) ? node.children : []);
      } else {
        const t = String(node.title || node.display_title || node.name || '').trim();
        if (t === title) {
          hit = normalizeId(node.collect_doc_id || node.id || '');
          return;
        }
      }
    }
  };
  walk(Array.isArray(tree) ? tree : []);
  return hit;
}

function treeContainsTitle(tree, title) {
  let found = false;
  const walk = (nodes) => {
    if (found || !Array.isArray(nodes)) return;
    for (const node of nodes) {
      if (!node) continue;
      const t = String(node.title || node.display_title || node.name || '').trim();
      if (t === title) {
        found = true;
        return;
      }
      walk(Array.isArray(node.children) ? node.children : []);
    }
  };
  walk(Array.isArray(tree) ? tree : []);
  return found;
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const now = Date.now();
  const groupAName = `iso_group_${PROJECT_A}_${now}`;
  const groupBName = `iso_group_${PROJECT_B}_${now}`;
  const docAName = `iso_doc_${PROJECT_A}_${now}`;
  const docBName = `iso_doc_${PROJECT_B}_${now}`;

  const summary = {
    apiUrl: API_URL,
    projectA: PROJECT_A,
    projectB: PROJECT_B,
    created: {
      groupAName,
      groupBName,
      docAName,
      docBName,
      groupAId: '',
      groupBId: '',
      docAId: '',
      docBId: '',
    },
    checks: {
      treeAHasA: false,
      treeAHasB: false,
      treeBHasA: false,
      treeBHasB: false,
      detailAOwnProject: false,
      detailAOtherProjectBlocked: false,
      detailBOwnProject: false,
      detailBOtherProjectBlocked: false,
    },
    cleanup: {
      docADeleted: false,
      docBDeleted: false,
      groupADeleted: false,
      groupBDeleted: false,
    },
    pass: false,
  };

  try {
    runCurl('config.doc_group_save', {
      doc_group_list: [
        { name: groupAName, desc: 'isolation-a', type: 'service', project_code: PROJECT_A, order_index: 1 },
      ],
    });
    runCurl('config.doc_group_save', {
      doc_group_list: [
        { name: groupBName, desc: 'isolation-b', type: 'service', project_code: PROJECT_B, order_index: 1 },
      ],
    });

    const treeA0 = runCurl('config.doc_group_http_service_tree_query', { project_code: PROJECT_A, to_tree: true });
    const treeB0 = runCurl('config.doc_group_http_service_tree_query', { project_code: PROJECT_B, to_tree: true });
    summary.created.groupAId = findGroupIdByTitle(treeA0.data || [], groupAName);
    summary.created.groupBId = findGroupIdByTitle(treeB0.data || [], groupBName);
    if (!summary.created.groupAId || !summary.created.groupBId) {
      throw new Error('failed to create or locate isolation groups');
    }

    runCurl('config.doc_save', {
      doc: {
        title: docAName,
        sub_title: 'isolation-a',
        type: '2',
        project_code: PROJECT_A,
        parent_dir: summary.created.groupAId,
        order_index: 1,
        request_mode: 'frontend',
        request_method: 'post',
        request_url: '/template_data/data',
        request_headers: '{}',
        code: '{}',
        code_desc: '',
        code_result: '',
      },
      important_list: [],
      params: [],
      result: [],
      demo: [],
    });

    runCurl('config.doc_save', {
      doc: {
        title: docBName,
        sub_title: 'isolation-b',
        type: '2',
        project_code: PROJECT_B,
        parent_dir: summary.created.groupBId,
        order_index: 1,
        request_mode: 'frontend',
        request_method: 'post',
        request_url: '/template_data/data',
        request_headers: '{}',
        code: '{}',
        code_desc: '',
        code_result: '',
      },
      important_list: [],
      params: [],
      result: [],
      demo: [],
    });

    const treeA = runCurl('config.doc_group_http_service_tree_query', { project_code: PROJECT_A, to_tree: true });
    const treeB = runCurl('config.doc_group_http_service_tree_query', { project_code: PROJECT_B, to_tree: true });

    summary.created.docAId = findDocIdByTitle(treeA.data || [], docAName);
    summary.created.docBId = findDocIdByTitle(treeB.data || [], docBName);
    if (!summary.created.docAId || !summary.created.docBId) {
      throw new Error('failed to create or locate isolation docs');
    }

    summary.checks.treeAHasA = treeContainsTitle(treeA.data || [], docAName);
    summary.checks.treeAHasB = treeContainsTitle(treeA.data || [], docBName);
    summary.checks.treeBHasA = treeContainsTitle(treeB.data || [], docAName);
    summary.checks.treeBHasB = treeContainsTitle(treeB.data || [], docBName);

    const detailAOwn = runCurl('config.doc_detail', { collect_doc_id: summary.created.docAId, project_code: PROJECT_A });
    const detailAOther = runCurl('config.doc_detail', { collect_doc_id: summary.created.docAId, project_code: PROJECT_B });
    const detailBOwn = runCurl('config.doc_detail', { collect_doc_id: summary.created.docBId, project_code: PROJECT_B });
    const detailBOther = runCurl('config.doc_detail', { collect_doc_id: summary.created.docBId, project_code: PROJECT_A });

    summary.checks.detailAOwnProject = String(detailAOwn?.data?.doc?.collect_doc_id || '') === summary.created.docAId;
    summary.checks.detailAOtherProjectBlocked = !String(detailAOther?.data?.doc?.collect_doc_id || '');
    summary.checks.detailBOwnProject = String(detailBOwn?.data?.doc?.collect_doc_id || '') === summary.created.docBId;
    summary.checks.detailBOtherProjectBlocked = !String(detailBOther?.data?.doc?.collect_doc_id || '');
  } finally {
    try {
      if (summary.created.docAId) {
        runCurl('config.doc_delete', { project_code: PROJECT_A, collect_doc_id_list: [summary.created.docAId] });
        summary.cleanup.docADeleted = true;
      }
    } catch (_error) {
      summary.cleanup.docADeleted = false;
    }
    try {
      if (summary.created.docBId) {
        runCurl('config.doc_delete', { project_code: PROJECT_B, collect_doc_id_list: [summary.created.docBId] });
        summary.cleanup.docBDeleted = true;
      }
    } catch (_error) {
      summary.cleanup.docBDeleted = false;
    }
    try {
      if (summary.created.groupAId) {
        runCurl('config.doc_group_delete', {
          project_code: PROJECT_A,
          doc_group_list: [{ doc_group_id: summary.created.groupAId }],
        });
        summary.cleanup.groupADeleted = true;
      }
    } catch (_error) {
      summary.cleanup.groupADeleted = false;
    }
    try {
      if (summary.created.groupBId) {
        runCurl('config.doc_group_delete', {
          project_code: PROJECT_B,
          doc_group_list: [{ doc_group_id: summary.created.groupBId }],
        });
        summary.cleanup.groupBDeleted = true;
      }
    } catch (_error) {
      summary.cleanup.groupBDeleted = false;
    }
  }

  summary.pass =
    summary.checks.treeAHasA &&
    !summary.checks.treeAHasB &&
    !summary.checks.treeBHasA &&
    summary.checks.treeBHasB &&
    summary.checks.detailAOwnProject &&
    summary.checks.detailAOtherProjectBlocked &&
    summary.checks.detailBOwnProject &&
    summary.checks.detailBOtherProjectBlocked &&
    summary.cleanup.docADeleted &&
    summary.cleanup.docBDeleted &&
    summary.cleanup.groupADeleted &&
    summary.cleanup.groupBDeleted;

  const outJson = path.join(OUT_DIR, 'webshell-editor-pool-http-project-isolation-check.json');
  fs.writeFileSync(outJson, JSON.stringify(summary, null, 2));
  console.log(JSON.stringify(summary, null, 2));
  if (!summary.pass) process.exitCode = 2;
})();
