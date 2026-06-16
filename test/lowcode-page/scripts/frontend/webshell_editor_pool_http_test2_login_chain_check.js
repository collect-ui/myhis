#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://127.0.0.1:8015/template_data/data';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || 'backend';
const GROUP_TITLE = process.env.WEBSHELL_EDITOR_POOL_GROUP_TITLE || 'test2';
const LOGIN_TITLE = process.env.WEBSHELL_EDITOR_POOL_LOGIN_TITLE || '登录';
const USER_TITLE = process.env.WEBSHELL_EDITOR_POOL_USER_TITLE || '获取用户信息';

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

  let out = {};
  try {
    out = JSON.parse(String(res.stdout || '{}'));
  } catch (error) {
    throw new Error(`${service} invalid response: ${String(res.stdout || '').slice(0, 500)}`);
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

function findGroup(tree, title) {
  let hit = null;
  const walk = (nodes) => {
    if (hit || !Array.isArray(nodes)) return;
    for (const node of nodes) {
      if (!node || !isDir(node)) continue;
      const t = String(node.title || node.display_title || node.name || '').trim();
      if (t === title) {
        hit = node;
        return;
      }
      walk(Array.isArray(node.children) ? node.children : []);
    }
  };
  walk(Array.isArray(tree) ? tree : []);
  return hit;
}

function findDocByTitle(groupNode, title) {
  if (!groupNode || !Array.isArray(groupNode.children)) return null;
  for (const node of groupNode.children) {
    if (!node || isDir(node)) continue;
    const t = String(node.title || node.display_title || node.name || '').trim();
    if (t === title) {
      return node;
    }
  }
  return null;
}

function parseJson(raw, fallback) {
  try {
    const text = String(raw === undefined || raw === null ? '' : raw).trim();
    if (!text) return fallback;
    return JSON.parse(text);
  } catch (_error) {
    return fallback;
  }
}

function buildProxyPayload(docDetail) {
  const doc = docDetail?.data?.doc || {};
  return {
    request_method: String(doc.request_method || 'post').toLowerCase(),
    request_url: String(doc.request_url || ''),
    request_header: parseJson(doc.request_headers, {}),
    request_data: parseJson(doc.code, {}),
  };
}

(function main() {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const summary = {
    apiUrl: API_URL,
    projectCode: PROJECT_CODE,
    groupTitle: GROUP_TITLE,
    loginTitle: LOGIN_TITLE,
    userTitle: USER_TITLE,
    docs: {
      groupId: '',
      loginDocId: '',
      userDocId: '',
    },
    requests: {
      login: {},
      userInfo: {},
    },
    checks: {
      loginResponseIsObject: false,
      userResponseIsObject: false,
      loginSetCookie: false,
      userSentCookie: false,
      userHasIdentityFields: false,
    },
    pass: false,
    error: '',
  };

  try {
    const tree = runCurl('config.doc_group_http_service_tree_query', {
      project_code: PROJECT_CODE,
      to_tree: true,
    });

    const groupNode = findGroup(tree.data || [], GROUP_TITLE);
    if (!groupNode) {
      throw new Error(`group not found: ${GROUP_TITLE}`);
    }
    summary.docs.groupId = normalizeId(groupNode.doc_group_id || groupNode.id || '');

    const loginNode = findDocByTitle(groupNode, LOGIN_TITLE);
    const userNode = findDocByTitle(groupNode, USER_TITLE);
    if (!loginNode || !userNode) {
      throw new Error(`doc not found in group ${GROUP_TITLE}: login=${!!loginNode}, user=${!!userNode}`);
    }

    summary.docs.loginDocId = normalizeId(loginNode.collect_doc_id || loginNode.id || '');
    summary.docs.userDocId = normalizeId(userNode.collect_doc_id || userNode.id || '');

    const loginDetail = runCurl('config.doc_detail', {
      collect_doc_id: summary.docs.loginDocId,
      project_code: PROJECT_CODE,
    });
    const userDetail = runCurl('config.doc_detail', {
      collect_doc_id: summary.docs.userDocId,
      project_code: PROJECT_CODE,
    });

    const loginReq = buildProxyPayload(loginDetail);
    const userReq = buildProxyPayload(userDetail);
    summary.requests.login = loginReq;
    summary.requests.userInfo = userReq;

    const loginResp = runCurl('webshell.http_proxy_request', Object.assign({}, loginReq, {
      project_code: PROJECT_CODE,
      clear_cookie: true,
    }));

    const userResp = runCurl('webshell.http_proxy_request', Object.assign({}, userReq, {
      project_code: PROJECT_CODE,
    }));

    const loginData = loginResp?.data || {};
    const userData = userResp?.data || {};
    const loginJson = loginData.response_json;
    const userJson = userData.response_json;

    summary.checks.loginResponseIsObject = !!(loginJson && typeof loginJson === 'object' && !Array.isArray(loginJson));
    summary.checks.userResponseIsObject = !!(userJson && typeof userJson === 'object' && !Array.isArray(userJson));
    summary.checks.loginSetCookie = Number(loginData?.cookie?.set_cookie_count || 0) > 0;
    summary.checks.userSentCookie = Number(userData?.cookie?.sent_count || 0) > 0;

    const userDataObj = userJson && typeof userJson === 'object' ? userJson.data : null;
    summary.checks.userHasIdentityFields = !!(
      userDataObj &&
      typeof userDataObj === 'object' &&
      String(userDataObj.username || '').trim() &&
      String(userDataObj.userid || '').trim()
    );

    summary.loginResult = {
      status_code: loginData.status_code,
      status_text: loginData.status_text,
      cookie: loginData.cookie || {},
      response_json: loginJson,
    };

    summary.userResult = {
      status_code: userData.status_code,
      status_text: userData.status_text,
      cookie: userData.cookie || {},
      response_json: userJson,
    };

    summary.pass =
      summary.checks.loginResponseIsObject &&
      summary.checks.userResponseIsObject &&
      summary.checks.loginSetCookie &&
      summary.checks.userSentCookie &&
      summary.checks.userHasIdentityFields;
  } catch (error) {
    summary.error = String(error && error.message ? error.message : error);
    summary.pass = false;
  }

  const outJson = path.join(OUT_DIR, 'webshell-editor-pool-http-test2-login-chain-check.json');
  fs.writeFileSync(outJson, JSON.stringify(summary, null, 2));
  console.log(JSON.stringify(summary, null, 2));

  if (!summary.pass) {
    process.exitCode = 2;
  }
})();
