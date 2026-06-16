#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://127.0.0.1:8015/template_data/data';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || 'backend';
const PROXY_URL = process.env.WEBSHELL_EDITOR_POOL_PROXY_URL || 'https://postman-echo.com/get?source=http-full-flow-check';

function runCurl(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  const res = spawnSync(
    'curl',
    [
      '-s',
      `${API_URL}?service=${service}`,
      '-H',
      'Content-Type: application/json',
      '--data',
      payload,
    ],
    { encoding: 'utf8' }
  );

  if (res.status !== 0) {
    throw new Error(res.stderr || `curl failed: ${service}`);
  }

  let out = {};
  try {
    out = JSON.parse(String(res.stdout || '{}'));
  } catch (error) {
    throw new Error(`${service} invalid json response: ${String(res.stdout || '').slice(0, 300)}`);
  }

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

(function main() {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const now = Date.now();
  const groupName = `full_flow_group_${now}`;
  const groupNameEdited = `full_flow_group_edited_${now}`;
  const docName = `full_flow_doc_${now}`;
  const docNameEdited = `full_flow_doc_edited_${now}`;

  const summary = {
    apiUrl: API_URL,
    projectCode: PROJECT_CODE,
    proxyUrl: PROXY_URL,
    created: {
      groupName,
      groupNameEdited,
      docName,
      docNameEdited,
      groupId: '',
      docId: '',
    },
    steps: {
      groupCreate: false,
      groupQuery: false,
      groupUpdate: false,
      groupQueryAfterUpdate: false,
      docCreate: false,
      docQuery: false,
      docDetail: false,
      docEdit: false,
      docDetailAfterEdit: false,
      directSend: false,
      proxySend: false,
      docDelete: false,
      groupDelete: false,
    },
    checks: {
      docProjectCodeOk: false,
      docModeUpdated: false,
      docMethodUpdated: false,
      docUrlUpdated: false,
      directSendHasData: false,
      proxySendHasStatusCode: false,
    },
    cleanup: {
      docDeleted: false,
      groupDeleted: false,
    },
    error: '',
    pass: false,
  };

  try {
    runCurl('config.doc_group_save', {
      doc_group_list: [
        {
          name: groupName,
          desc: 'full-flow-group',
          type: 'service',
          project_code: PROJECT_CODE,
          order_index: 1,
        },
      ],
    });
    summary.steps.groupCreate = true;

    const treeAfterGroupCreate = runCurl('config.doc_group_http_service_tree_query', {
      project_code: PROJECT_CODE,
      to_tree: true,
    });
    summary.created.groupId = findGroupIdByTitle(treeAfterGroupCreate.data || [], groupName);
    summary.steps.groupQuery = !!summary.created.groupId;
    if (!summary.created.groupId) {
      throw new Error('group create success but cannot find group in tree');
    }

    runCurl('config.doc_group_update', {
      doc_group_list: [
        {
          doc_group_id: summary.created.groupId,
          name: groupNameEdited,
          desc: 'full-flow-group-edited',
          type: 'service',
          project_code: PROJECT_CODE,
          order_index: 2,
        },
      ],
    });
    summary.steps.groupUpdate = true;

    const treeAfterGroupUpdate = runCurl('config.doc_group_http_service_tree_query', {
      project_code: PROJECT_CODE,
      to_tree: true,
    });
    summary.steps.groupQueryAfterUpdate =
      treeContainsTitle(treeAfterGroupUpdate.data || [], groupNameEdited) &&
      !treeContainsTitle(treeAfterGroupUpdate.data || [], groupName);
    if (!summary.steps.groupQueryAfterUpdate) {
      throw new Error('group update check failed');
    }

    runCurl('config.doc_save', {
      doc: {
        title: docName,
        sub_title: 'full-flow-doc',
        type: '2',
        project_code: PROJECT_CODE,
        parent_dir: summary.created.groupId,
        order_index: 1,
        request_mode: 'frontend',
        request_method: 'post',
        request_url: '/template_data/data',
        request_headers: '{}',
        code: JSON.stringify({ service: 'config.doc_group_http_service_tree_query', project_code: PROJECT_CODE, to_tree: true }, null, 2),
        code_desc: 'full flow create',
        code_result: '',
      },
      important_list: [],
      params: [],
      result: [],
      demo: [],
    });
    summary.steps.docCreate = true;

    const treeAfterDocCreate = runCurl('config.doc_group_http_service_tree_query', {
      project_code: PROJECT_CODE,
      to_tree: true,
    });
    summary.created.docId = findDocIdByTitle(treeAfterDocCreate.data || [], docName);
    summary.steps.docQuery = !!summary.created.docId;
    if (!summary.created.docId) {
      throw new Error('doc create success but cannot find doc in tree');
    }

    const docDetailCreate = runCurl('config.doc_detail', {
      collect_doc_id: summary.created.docId,
      project_code: PROJECT_CODE,
    });
    const createdDoc = docDetailCreate?.data?.doc || {};
    summary.checks.docProjectCodeOk = String(createdDoc.project_code || '') === PROJECT_CODE;
    summary.steps.docDetail = String(createdDoc.collect_doc_id || '') === summary.created.docId;
    if (!summary.steps.docDetail) {
      throw new Error('doc detail query failed after create');
    }

    runCurl('config.doc_edit', {
      doc: {
        collect_doc_id: summary.created.docId,
        title: docNameEdited,
        sub_title: 'full-flow-doc-edited',
        type: '2',
        project_code: PROJECT_CODE,
        parent_dir: summary.created.groupId,
        order_index: 3,
        request_mode: 'backend',
        request_method: 'get',
        request_url: PROXY_URL,
        request_headers: '{}',
        code: '{}',
        code_desc: 'full flow edited',
        code_result: '',
      },
      important_list: Array.isArray(docDetailCreate?.data?.important_list) ? docDetailCreate.data.important_list : [],
      params: Array.isArray(docDetailCreate?.data?.params) ? docDetailCreate.data.params : [],
      result: Array.isArray(docDetailCreate?.data?.result) ? docDetailCreate.data.result : [],
      demo: Array.isArray(docDetailCreate?.data?.demo) ? docDetailCreate.data.demo : [],
    });
    summary.steps.docEdit = true;

    const docDetailEdited = runCurl('config.doc_detail', {
      collect_doc_id: summary.created.docId,
      project_code: PROJECT_CODE,
    });
    const editedDoc = docDetailEdited?.data?.doc || {};
    summary.checks.docModeUpdated = String(editedDoc.request_mode || '') === 'backend';
    summary.checks.docMethodUpdated = String(editedDoc.request_method || '').toLowerCase() === 'get';
    summary.checks.docUrlUpdated = String(editedDoc.request_url || '') === PROXY_URL;
    summary.steps.docDetailAfterEdit =
      String(editedDoc.collect_doc_id || '') === summary.created.docId &&
      String(editedDoc.title || '') === docNameEdited;
    if (!summary.steps.docDetailAfterEdit) {
      throw new Error('doc detail check failed after edit');
    }

    const directSend = runCurl('config.doc_group_http_service_tree_query', {
      project_code: PROJECT_CODE,
      to_tree: true,
    });
    summary.checks.directSendHasData = Array.isArray(directSend.data);
    summary.steps.directSend = summary.checks.directSendHasData;

    const proxySend = runCurl('webshell.http_proxy_request', {
      request_method: 'get',
      request_url: PROXY_URL,
      request_header: {},
      request_data: {},
    });
    const statusCode = Number(proxySend?.data?.status_code || 0);
    summary.checks.proxySendHasStatusCode = Number.isFinite(statusCode) && statusCode > 0;
    summary.steps.proxySend = summary.checks.proxySendHasStatusCode;

    runCurl('config.doc_delete', {
      project_code: PROJECT_CODE,
      collect_doc_id_list: [summary.created.docId],
    });
    summary.steps.docDelete = true;
    summary.cleanup.docDeleted = true;

    const treeAfterDocDelete = runCurl('config.doc_group_http_service_tree_query', {
      project_code: PROJECT_CODE,
      to_tree: true,
    });
    if (treeContainsTitle(treeAfterDocDelete.data || [], docNameEdited)) {
      throw new Error('doc delete check failed');
    }

    runCurl('config.doc_group_delete', {
      project_code: PROJECT_CODE,
      doc_group_list: [{ doc_group_id: summary.created.groupId }],
    });
    summary.steps.groupDelete = true;
    summary.cleanup.groupDeleted = true;

    const treeAfterGroupDelete = runCurl('config.doc_group_http_service_tree_query', {
      project_code: PROJECT_CODE,
      to_tree: true,
    });
    if (treeContainsTitle(treeAfterGroupDelete.data || [], groupNameEdited)) {
      throw new Error('group delete check failed');
    }
  } catch (error) {
    summary.error = String(error && error.message ? error.message : error);
  } finally {
    try {
      if (!summary.cleanup.docDeleted && summary.created.docId) {
        runCurl('config.doc_delete', {
          project_code: PROJECT_CODE,
          collect_doc_id_list: [summary.created.docId],
        });
        summary.cleanup.docDeleted = true;
      }
    } catch (_err) {
      summary.cleanup.docDeleted = false;
    }

    try {
      if (!summary.cleanup.groupDeleted && summary.created.groupId) {
        runCurl('config.doc_group_delete', {
          project_code: PROJECT_CODE,
          doc_group_list: [{ doc_group_id: summary.created.groupId }],
        });
        summary.cleanup.groupDeleted = true;
      }
    } catch (_err) {
      summary.cleanup.groupDeleted = false;
    }
  }

  summary.pass =
    summary.steps.groupCreate &&
    summary.steps.groupQuery &&
    summary.steps.groupUpdate &&
    summary.steps.groupQueryAfterUpdate &&
    summary.steps.docCreate &&
    summary.steps.docQuery &&
    summary.steps.docDetail &&
    summary.steps.docEdit &&
    summary.steps.docDetailAfterEdit &&
    summary.steps.directSend &&
    summary.steps.proxySend &&
    summary.steps.docDelete &&
    summary.steps.groupDelete &&
    summary.checks.docProjectCodeOk &&
    summary.checks.docModeUpdated &&
    summary.checks.docMethodUpdated &&
    summary.checks.docUrlUpdated &&
    summary.checks.directSendHasData &&
    summary.checks.proxySendHasStatusCode &&
    summary.cleanup.docDeleted &&
    summary.cleanup.groupDeleted &&
    !summary.error;

  const outJson = path.join(OUT_DIR, 'webshell-editor-pool-http-full-flow-check.json');
  fs.writeFileSync(outJson, JSON.stringify(summary, null, 2));
  console.log(JSON.stringify(summary, null, 2));

  if (!summary.pass) {
    process.exitCode = 2;
  }
})();
