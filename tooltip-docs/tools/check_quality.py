#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import json
import re
from pathlib import Path
from datetime import datetime

import yaml

ROOT = Path('/data/project/sport/tooltip-docs')
KV_ROOT = ROOT / 'kv'
KEY_DIR = KV_ROOT / 'key'
MODULE_DIR = KV_ROOT / 'module'

REQUIRED_SECTIONS = ['## 作用', '## 参数', '## 示例']

# 语义规则（基于源码与现网配置抽检结论）
MUST_HAVE_PARAMS_WRAPPER = {
    'add_param', 'combine_service', 'val2param', 'hook', 'new_col', 'file_response',
}
MUST_NOT_HAVE_PARAMS_WRAPPER = {
    'update_data', 'check_array', 'render_template',
}


def load_text(path: Path) -> str:
    return path.read_text(encoding='utf-8')


def extract_yaml_blocks(text: str):
    return re.findall(r"```(?:yml|yaml)\n(.*?)\n```", text, flags=re.S)


def find_key_example_block(text: str):
    # 只取第一个yaml示例做语义规则验证
    m = re.search(r"```(?:yml|yaml)\n(.*?)\n```", text, flags=re.S)
    return m.group(1) if m else ''


def path_exists_in_code_ref(text: str):
    # 提取反引号中的绝对路径并检查存在性
    refs = re.findall(r"`(/[^`]+)`", text)
    missing = []
    for r in refs:
        p = Path(r)
        if not p.exists():
            missing.append(r)
    return missing


def check_file(path: Path):
    issues = []
    text = load_text(path)
    stem = path.stem

    h1 = re.search(r"^#\s+([^\n]+)", text, flags=re.M)
    if not h1:
        issues.append({'type': 'missing_h1', 'msg': '缺少H1标题'})
        return issues, 0

    h1_text = h1.group(1).strip()
    if 'key=' in h1_text:
        key = h1_text.split('key=', 1)[1].strip()
        if key != stem:
            issues.append({'type': 'header_mismatch', 'msg': f'H1 key={key} 与文件名 {stem} 不一致'})
    if 'module=' in h1_text:
        module = h1_text.split('module=', 1)[1].strip()
        if module != stem:
            issues.append({'type': 'header_mismatch', 'msg': f'H1 module={module} 与文件名 {stem} 不一致'})

    for s in REQUIRED_SECTIONS:
        if s not in text:
            issues.append({'type': 'missing_section', 'msg': f'缺少 {s}'})

    yaml_blocks = extract_yaml_blocks(text)
    for idx, block in enumerate(yaml_blocks, start=1):
        try:
            yaml.safe_load(block)
        except Exception as e:
            issues.append({'type': 'yaml_parse_error', 'msg': f'第{idx}个YAML示例解析失败: {str(e).splitlines()[0]}'})

    # 语义规则：示例里 params 层级
    if path.parent.name == 'key':
        example = find_key_example_block(text)
        if stem in MUST_HAVE_PARAMS_WRAPPER:
            if re.search(r"-\s*key\s*:\s*['\"]?%s['\"]?\s*\n\s+params\s*:" % re.escape(stem), example) is None:
                issues.append({'type': 'semantic_warning', 'msg': '示例可能缺少 params: 包裹'})
        if stem in MUST_NOT_HAVE_PARAMS_WRAPPER:
            if re.search(r"-\s*key\s*:\s*['\"]?%s['\"]?\s*\n\s+params\s*:" % re.escape(stem), example):
                issues.append({'type': 'semantic_warning', 'msg': '示例不应使用 params: 包裹'})

    # 源码路径存在性（仅告警）
    missing_paths = path_exists_in_code_ref(text)
    for p in missing_paths:
        issues.append({'type': 'path_warning', 'msg': f'源码路径不存在: {p}'})

    return issues, len(yaml_blocks)


def main():
    md_files = sorted(KV_ROOT.rglob('*.md'))
    all_issues = []
    yaml_count = 0

    for f in md_files:
        issues, cnt = check_file(f)
        yaml_count += cnt
        for it in issues:
            all_issues.append({'file': str(f), **it})

    summary = {
        'generated_at': datetime.now().astimezone().strftime('%Y-%m-%d %H:%M:%S %z'),
        'files': len(md_files),
        'key_docs': len(list(KEY_DIR.glob('*.md'))),
        'module_docs': len(list(MODULE_DIR.glob('*.md'))),
        'yaml_blocks': yaml_count,
        'issues_total': len(all_issues),
        'issues_by_type': {},
    }
    for it in all_issues:
        summary['issues_by_type'][it['type']] = summary['issues_by_type'].get(it['type'], 0) + 1

    out_json = ROOT / 'evidence' / 'doc_quality_report_latest.json'
    out_md = ROOT / 'evidence' / 'doc_quality_report_latest.md'

    out_json.write_text(json.dumps({'summary': summary, 'issues': all_issues}, ensure_ascii=False, indent=2), encoding='utf-8')

    lines = []
    lines.append('# 文档质量报告（latest）')
    lines.append('')
    lines.append(f"- 生成时间：{summary['generated_at']}")
    lines.append(f"- 扫描文件：{summary['files']}（key={summary['key_docs']}，module={summary['module_docs']}）")
    lines.append(f"- YAML示例：{summary['yaml_blocks']}")
    lines.append(f"- 问题总数：{summary['issues_total']}")
    lines.append('')
    lines.append('## 问题分类')
    lines.append('')
    if summary['issues_by_type']:
        for t, c in sorted(summary['issues_by_type'].items()):
            lines.append(f"- {t}: {c}")
    else:
        lines.append('- 无')

    lines.append('')
    lines.append('## 详细问题')
    lines.append('')
    if all_issues:
        for it in all_issues:
            lines.append(f"- {it['file']}: [{it['type']}] {it['msg']}")
    else:
        lines.append('- 无')

    out_md.write_text('\n'.join(lines) + '\n', encoding='utf-8')

    print(str(out_json))
    print(str(out_md))
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == '__main__':
    main()
