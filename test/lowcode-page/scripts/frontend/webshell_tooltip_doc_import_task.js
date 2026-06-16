#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

const ROOT = '/data/project/sport';
const PAGE_URL = process.env.TOOLTIP_DOCS_PAGE_URL || 'http://192.168.232.130:8015/static/console/index.html';
const STATE_FILE = path.join(ROOT, 'tooltip-docs', 'import-state.json');
const PROGRESS_FILE = path.join(ROOT, 'tooltip-docs', 'import-progress.md');
const EVIDENCE_DIR = path.join(ROOT, 'tooltip-docs', 'evidence');
const TOOLTIP_ROOT = path.join(ROOT, 'tooltip-docs');

function nowIso() {
  return new Date().toISOString();
}

function nowDateTimeLocal() {
  const d = new Date();
  const p = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
}

function ensureDirForFile(filePath) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
}

function cleanText(input) {
  return String(input || '').replace(/\r/g, '').trim();
}

function isCodeLike(input) {
  const text = cleanText(input);
  if (!text) return false;
  if (text.startsWith('{') || text.startsWith('[')) return true;
  if (/(^|\n)\s*-\s+key\s*:/m.test(text)) return true;
  if (/(^|\n)\s*[a-zA-Z_][a-zA-Z0-9_]*\s*:/m.test(text)) return true;
  return false;
}

function stripNumberPrefix(input) {
  return String(input || '').replace(/^\s*\d+\s*[.)、]\s*/, '').trim();
}

function normalizeYamlSnippet(code) {
  const raw = String(code || '').replace(/\r/g, '');
  const lines = raw.split('\n');
  let firstIdx = -1;
  for (let i = 0; i < lines.length; i += 1) {
    if (cleanText(lines[i])) {
      firstIdx = i;
      break;
    }
  }
  if (firstIdx < 0) return raw;

  const firstLine = lines[firstIdx];
  const m = firstLine.match(/^(\s*)-\s+/);
  if (!m) return raw;
  const baseIndent = m[1].length;

  let minChildIndent = Number.POSITIVE_INFINITY;
  for (let i = firstIdx + 1; i < lines.length; i += 1) {
    const line = lines[i];
    if (!cleanText(line)) continue;
    const indent = (line.match(/^(\s*)/) || [''])[0].length;
    if (indent > baseIndent) {
      minChildIndent = Math.min(minChildIndent, indent - baseIndent);
    }
  }
  if (!Number.isFinite(minChildIndent)) return raw;

  const shift = minChildIndent - 2;
  if (shift <= 0) return raw;

  const fixed = lines.map((line, idx) => {
    if (idx <= firstIdx) return line;
    if (!cleanText(line)) return line;
    const indent = (line.match(/^(\s*)/) || [''])[0].length;
    if (indent <= baseIndent) return line;
    const nextIndent = Math.max(baseIndent + 1, indent - shift);
    return `${' '.repeat(nextIndent)}${line.trimStart()}`;
  });
  return fixed.join('\n');
}

function buildTableMarkdown(table) {
  const headers = (table.headers || []).map((h) => cleanText(h));
  const rows = (table.rows || []).map((r) => (r || []).map((c) => cleanText(c)));
  if (!headers.length) return '';
  const line1 = `| ${headers.join(' | ')} |`;
  const line2 = `| ${headers.map(() => '---').join(' | ')} |`;
  const body = rows.map((r) => `| ${headers.map((_, i) => r[i] || '').join(' | ')} |`);
  return [line1, line2, ...body].join('\n');
}

function pickTop(items, limit) {
  return (items || []).map((s) => cleanText(s)).filter(Boolean).slice(0, limit);
}

function buildDocMarkdown(task, source) {
  const title = cleanText(source.title) || task.source_title;
  const textareas = (source.textarea_values || []).map((v) => cleanText(v)).filter(Boolean);
  const codeBlocks = textareas.filter((t) => isCodeLike(t));
  const descBlocks = textareas.filter((t) => !isCodeLike(t));
  const listItems = pickTop(source.list_items || [], 8);
  const subTitles = pickTop(source.sub_titles || [], 12);
  const table = Array.isArray(source.tables) && source.tables.length > 0 ? source.tables[0] : null;

  const targetName = path.basename(task.target_file, '.md');
  let head;
  if (task.id.startsWith('M-')) {
    head = `# module=${task.source_title}`;
  } else if (task.id.startsWith('H-')) {
    head = `# key=${task.source_title}`;
  } else {
    head = `# ${targetName}`;
  }

  const lines = [];
  lines.push(head);
  lines.push('');
  lines.push(`来源：\`服务文档 -> ${task.source_group} -> ${task.source_doc_text || task.source_title}\``);
  lines.push(`页面标题：\`${title}\``);
  lines.push('');

  const effectCandidates = [
    ...descBlocks,
    ...listItems,
    ...pickTop((source.body_text || '').split('\n'), 20),
  ].map((v) => cleanText(v)).filter(Boolean);

  const effect = effectCandidates.slice(0, 3);
  if (effect.length) {
    lines.push('## 作用');
    for (const item of effect) {
      lines.push(`- ${item}`);
    }
    lines.push('');
  }

  if (table && table.headers && table.headers.length) {
    const tableMd = buildTableMarkdown(table);
    if (tableMd) {
      lines.push('## 参数');
      lines.push(tableMd);
      lines.push('');
    }
  }

  if (subTitles.length) {
    lines.push('## 结构');
    for (const t of subTitles) {
      lines.push(`- ${t}`);
    }
    lines.push('');
  }

  if (codeBlocks.length) {
    lines.push('## 示例');
    const lang = task.id.startsWith('M-') || task.id.startsWith('H-') || /\.ya?ml/i.test(codeBlocks[0]) ? 'yml' : '';
    const block = lang === 'yml' ? normalizeYamlSnippet(codeBlocks[0]) : codeBlocks[0];
    lines.push(`\`\`\`${lang}`);
    lines.push(block);
    lines.push('```');
    lines.push('');
  }

  const notes = listItems.slice(0, 6);
  if (notes.length > 0) {
    lines.push('## 说明要点');
    for (const n of notes) {
      lines.push(`- ${n}`);
    }
    lines.push('');
  }

  return `${lines.join('\n').replace(/\n{3,}/g, '\n\n').trim()}\n`;
}

function buildEvidenceMarkdown(task, source, paths) {
  const excerpt = [];
  const ta = (source.textarea_values || []).map((v) => cleanText(v)).filter(Boolean);
  const firstCode = ta.find((v) => isCodeLike(v));
  const firstDesc = ta.find((v) => !isCodeLike(v));
  if (firstCode) {
    excerpt.push(`- 示例代码首行：\`${cleanText(firstCode).split('\n')[0]}\``);
  }
  if (firstDesc) {
    excerpt.push(`- 说明：${cleanText(firstDesc).slice(0, 180)}`);
  }
  const lis = pickTop(source.list_items || [], 3);
  for (const i of lis) {
    excerpt.push(`- 列表要点：${i}`);
  }

  const lines = [
    `# ${task.id}`,
    '',
    `- 时间：${nowDateTimeLocal()}`,
    '- 读取方式：无头浏览器点击左侧目录树',
    `- 源分组：${task.source_group}`,
    `- 源文档：\`${task.source_doc_text || task.source_title}\``,
    `- 页面标题：\`${cleanText(source.title)}\``,
    `- 目标文件：\`${task.target_file}\``,
    `- 截图：\`${paths.screenshotPath}\``,
    `- 原始读取：\`${paths.sourceJsonPath}\``,
    '',
    '## 关键摘录（页面展示内容）',
  ];
  if (excerpt.length) {
    lines.push(...excerpt);
  } else {
    lines.push('- 无可用摘录（仅保留原始 JSON）');
  }
  lines.push('');
  lines.push('## 改写策略');
  lines.push('- 保留原文核心语义。');
  lines.push('- 统一整理成 tooltip 结构：作用、参数、示例、说明要点。');
  lines.push('');
  return lines.join('\n');
}

function updateStateFile(taskId) {
  const raw = fs.readFileSync(STATE_FILE, 'utf8');
  const state = JSON.parse(raw);
  const idx = state.tasks.findIndex((t) => t.id === taskId);
  if (idx < 0) {
    throw new Error(`task not found in state: ${taskId}`);
  }
  state.tasks[idx].status = 'completed';
  state.tasks[idx].note = `${nowDateTimeLocal()} 已完成，来源为无头浏览器点击读取`;

  const next = state.tasks.find((t) => t.status === 'pending' && t.id !== taskId);
  state.current_task_id = next ? next.id : '';
  state.updated_at = new Date().toISOString().replace('Z', '+08:00');

  fs.writeFileSync(STATE_FILE, `${JSON.stringify(state, null, 2)}\n`, 'utf8');
}

function appendProgress(task, source) {
  const lines = [];
  lines.push(`- 完成 \`${task.id}\`：`);
  lines.push(`  - 源读取：无头浏览器点击 \`服务文档 -> ${task.source_group} -> ${task.source_doc_text || task.source_title}\``);
  lines.push(`  - 页面标题：\`${cleanText(source.title)}\``);
  lines.push(`  - 目标写入：\`${task.target_file}\``);
  lines.push(`  - 证据：\`evidence/${task.id}.md\` + \`evidence/${task.id}-source.png\` + \`evidence/${task.id}-source.json\``);
  fs.appendFileSync(PROGRESS_FILE, `${lines.join('\n')}\n`, 'utf8');
}

async function expandGroupAndClickItem(page, group, sourceTitle) {
  const headers = page.locator('.aside .el-collapse-item__header');
  const headerCount = await headers.count();
  let header = null;
  for (let i = 0; i < headerCount; i += 1) {
    const h = headers.nth(i);
    const txt = cleanText(await h.innerText().catch(() => ''));
    if (txt === group) {
      header = h;
      break;
    }
  }
  if (!header) {
    const all = [];
    for (let i = 0; i < headerCount; i += 1) {
      all.push(cleanText(await headers.nth(i).innerText().catch(() => '')));
    }
    throw new Error(`group header not found: ${group}, available=${JSON.stringify(all)}`);
  }

  const groupItem = header.locator('xpath=ancestor::div[contains(@class,"el-collapse-item")][1]');
  await groupItem.waitFor({ state: 'visible', timeout: 20000 });

  const headerClass = await header.getAttribute('class');
  if (!String(headerClass || '').includes('is-active')) {
    await header.click();
    await page.waitForTimeout(500);
  }

  const candidates = groupItem.locator('.doc-item');
  const total = await candidates.count();
  if (total < 1) {
    throw new Error(`group has no doc items: ${group}`);
  }

  let target = null;
  for (let i = 0; i < total; i += 1) {
    const node = candidates.nth(i);
    const txt = cleanText(await node.innerText().catch(() => ''));
    if (txt.includes(sourceTitle)) {
      target = node;
      break;
    }
  }

  if (!target) {
    for (let i = 0; i < total; i += 1) {
      const node = candidates.nth(i);
      const txt = cleanText(await node.innerText().catch(() => ''));
      if (stripNumberPrefix(txt).includes(stripNumberPrefix(sourceTitle))) {
        target = node;
        break;
      }
    }
  }

  if (!target) {
    const allTexts = [];
    for (let i = 0; i < total; i += 1) {
      const node = candidates.nth(i);
      const txt = cleanText(await node.innerText().catch(() => ''));
      allTexts.push(txt);
    }
    throw new Error(`doc item not found in group=${group}, sourceTitle=${sourceTitle}, available=${JSON.stringify(allTexts)}`);
  }

  const docText = cleanText(await target.innerText().catch(() => sourceTitle));
  await target.click();
  await page.waitForTimeout(1200);
  return docText;
}

async function extractActiveDoc(page) {
  await page.waitForSelector('.content .el-tabs__header .el-tabs__item.is-active', { timeout: 20000 });

  const data = await page.evaluate(() => {
    const clean = (v) => String(v || '').replace(/\r/g, '').trim();
    const panes = Array.from(document.querySelectorAll('.content .el-tabs__content .el-tab-pane'));
    const pane = panes.find((el) => {
      const style = window.getComputedStyle(el);
      return style && style.display !== 'none' && el.getAttribute('aria-hidden') !== 'true';
    }) || panes[panes.length - 1] || null;

    if (!pane) {
      return { ok: false, error: 'active pane not found' };
    }

    const activeTab = document.querySelector('.content .el-tabs__header .el-tabs__item.is-active');
    const title = clean((pane.querySelector('.doc-title') || {}).innerText || '');

    const subTitles = Array.from(pane.querySelectorAll('.doc-sub-title')).map((el) => clean(el.innerText)).filter(Boolean);
    const listItems = Array.from(pane.querySelectorAll('li')).map((el) => clean(el.innerText)).filter(Boolean);

    const textareas = Array.from(pane.querySelectorAll('textarea')).map((el) => clean(el.value || el.textContent || '')).filter(Boolean);

    const tables = Array.from(pane.querySelectorAll('.el-table')).map((tb) => {
      const headerCells = Array.from(tb.querySelectorAll('.el-table__header th .cell')).map((el) => clean(el.innerText)).filter(Boolean);
      const bodyRows = Array.from(tb.querySelectorAll('.el-table__body tbody tr')).map((tr) => {
        return Array.from(tr.querySelectorAll('td .cell')).map((td) => clean(td.innerText));
      });
      return {
        headers: headerCells,
        rows: bodyRows,
      };
    }).filter((t) => (t.headers || []).length > 0 || (t.rows || []).length > 0);

    return {
      ok: true,
      active_tab_text: clean(activeTab ? activeTab.innerText : ''),
      title,
      sub_titles: subTitles,
      list_items: listItems,
      textarea_count: textareas.length,
      textarea_values: textareas,
      tables,
      body_text: clean(pane.innerText || ''),
    };
  });

  if (!data || !data.ok) {
    throw new Error(data?.error || 'extract active doc failed');
  }
  return data;
}

async function run() {
  const args = process.argv.slice(2);
  const taskId = (args[0] || '').trim();
  if (!taskId) {
    throw new Error('usage: node webshell_tooltip_doc_import_task.js <TASK_ID>');
  }

  const state = JSON.parse(fs.readFileSync(STATE_FILE, 'utf8'));
  const task = state.tasks.find((t) => t.id === taskId);
  if (!task) {
    throw new Error(`task id not found: ${taskId}`);
  }

  if (task.status === 'completed') {
    console.log(JSON.stringify({
      pass: true,
      skipped: true,
      reason: `task already completed: ${taskId}`,
    }, null, 2));
    return;
  }

  fs.mkdirSync(EVIDENCE_DIR, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1680, height: 1000 } });

  let sourceDocText = '';
  let source = null;
  try {
    await page.goto(PAGE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForTimeout(1800);

    await page.waitForSelector('.aside .el-collapse-item', { timeout: 20000 });
    sourceDocText = await expandGroupAndClickItem(page, task.source_group, task.source_title);

    source = await extractActiveDoc(page);
    source.task_id = task.id;
    source.source_group = task.source_group;
    source.source_title = task.source_title;
    source.source_doc_text = sourceDocText;
    source.read_at = nowIso();

    const screenshotPath = path.join(EVIDENCE_DIR, `${task.id}-source.png`);
    const sourceJsonPath = path.join(EVIDENCE_DIR, `${task.id}-source.json`);
    const evidenceMdPath = path.join(EVIDENCE_DIR, `${task.id}.md`);
    const targetFilePath = path.join(TOOLTIP_ROOT, task.target_file);

    await page.screenshot({ path: screenshotPath, fullPage: true });

    source.screenshot = screenshotPath;
    ensureDirForFile(sourceJsonPath);
    fs.writeFileSync(sourceJsonPath, `${JSON.stringify(source, null, 2)}\n`, 'utf8');

    const md = buildDocMarkdown(task, source);
    ensureDirForFile(targetFilePath);
    fs.writeFileSync(targetFilePath, md, 'utf8');

    const evidenceMd = buildEvidenceMarkdown(task, source, { screenshotPath, sourceJsonPath });
    ensureDirForFile(evidenceMdPath);
    fs.writeFileSync(evidenceMdPath, `${evidenceMd}\n`, 'utf8');

    updateStateFile(task.id);
    appendProgress(task, source);

    console.log(JSON.stringify({
      pass: true,
      task_id: task.id,
      source_group: task.source_group,
      source_title: task.source_title,
      source_doc_text: sourceDocText,
      page_title: source.title,
      target_file: targetFilePath,
      evidence: {
        screenshotPath,
        sourceJsonPath,
        evidenceMdPath,
      },
    }, null, 2));
  } finally {
    await page.close().catch(() => {});
    await browser.close().catch(() => {});
  }
}

run().catch((error) => {
  console.error(JSON.stringify({ pass: false, error: String(error?.message || error) }, null, 2));
  process.exit(1);
});
