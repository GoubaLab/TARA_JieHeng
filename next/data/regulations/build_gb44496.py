"""Import the reviewed user-supplied GB 44496 PDF entirely offline.

Keeps the 2024 standard and the separately printed 2026 amendment immutable.
Writes merge patches, never the shared manifest. No OCR or inferred provisions.
"""
from collections import Counter
import hashlib
import json
from pathlib import Path
import re
import shutil
import sys

from pypdf import PdfReader
from build_gb44495 import body_text

HERE = Path(__file__).resolve().parent
WORK = HERE.parents[2] / '.work' / 'regulations'
SOURCE = Path('provide-your-authorized-input.pdf')
SOURCE_SHA = '0f521729624ef87ae55689b7cefc50e6a5a7ae879ba3f30bf9af42dda7145a81'
URL = 'https://openstd.samr.gov.cn/bzgk/std/newGbInfo?hcno=8BC0D8B44DD4E71F9557BADE5175565A'
NUMBER = re.compile(r'^(\d+(?:\.\d+)*)(?:\s+(.*)|$)')
LETTER = re.compile(r'^([a-z])）\s*(.*)')
SUBITEM = re.compile(r'^(\d+)）\s*(.*)')


def normalize(raw):
    line = raw.replace('\u3000', ' ').replace('\u00a0', ' ').strip()
    line = re.sub(r'(?<=\d)\s*\.\s*(?=\d)', '.', line)
    line = re.sub(r'\s*([，。、；：（）“”~—⁃])\s*', r'\1', line)
    line = re.sub(r'(?<=[\u3400-\u9fff])\s+(?=[\u3400-\u9fff])', '', line)
    return re.sub(r'[ \t]+', ' ', line)


def lines_on(reader, page):
    lines = [normalize(s) for s in reader.pages[page - 1].extract_text().splitlines() if s.strip()]
    lines = [s for s in lines if not re.fullmatch(r'GB\s*44496—2024', s) and not re.fullmatch(r'[— ]+', s)]
    if lines and re.fullmatch(r'[\dⅠⅡⅢ]+', lines[-1]):
        lines.pop()
    return [(line, page) for line in lines]


def build(reader):
    rows, seen = [], set()

    def add(doc, locator, number, title, lines, parent=None, kind='clause', source_number=None, **extra):
        key = 'reg:' + doc + ':' + locator
        if key in seen:
            raise ValueError('Duplicate reference: ' + key)
        seen.add(key)
        pages = sorted({page for _, page in lines})
        body = body_text(lines)
        rows.append({'key': key, 'document_id': doc, 'number': number, 'source_number': source_number,
                     'title': title[:150], 'body': body, 'description': body,
                     'parent': 'reg:' + doc + ':' + parent if parent else None, 'kind': kind,
                     'normative': False if kind == 'publication_note' else None,
                     'normative_note': '强制性标准原文；注、示例和适用条件随条款保留，不自动转换为独立合规义务。',
                     'language': 'zh-CN', 'source_url': URL, 'source_file': 'sources/gb-44496-2024.pdf',
                     'source_sha256': SOURCE_SHA, 'page': pages[0], 'page_end': pages[-1],
                     'print_page': str(pages[0] - 6) if pages[0] >= 7 else 'Ⅲ',
                     'source_pages': pages, 'printed_pages': [str(p - 6) if p >= 7 else 'Ⅲ' for p in pages], **extra})

    doc = 'gb-44496-2024'
    add(doc, 'foreword', '前言', '前言', lines_on(reader, 5), kind='publication_note')
    lines = [line for p in range(7, 14) for line in lines_on(reader, p)]
    if lines[0][0] != '汽车软件升级通用技术要求':
        raise ValueError('Unexpected opening page')
    lines = lines[1:]
    headings = []
    for at, (line, page) in enumerate(lines):
        match = NUMBER.match(line)
        if not match:
            continue
        # A wrapped cross-reference to GB 44495 starts a line on physical page 11.
        # Its normal text font and the printed sentence confirm it is not a heading.
        if page == 11 and match.group(1) == '8.3.4.2.2' and line.startswith('8.3.4.2.2 和 8.3.4.3'):
            continue
        headings.append((at, match.group(1), match.group(2) or ''))
    numbers = {n for _, n, _ in headings}
    if len(numbers) != len(headings):
        raise ValueError('Duplicate numeric heading')
    for i, (start, number, title) in enumerate(headings):
        own_end = headings[i + 1][0] if i + 1 < len(headings) else len(lines)
        end = next((j for j, n, _ in headings[i + 1:] if not n.startswith(number + '.')), len(lines))
        own = lines[start:own_end]
        parent = number.rsplit('.', 1)[0] if '.' in number else None
        while parent and parent not in numbers:
            parent = parent.rsplit('.', 1)[0] if '.' in parent else None
        # The source goes directly from chapter 7 to 7.1.1 and 7.1.2. Do not invent 7.1.
        section = any(n.startswith(number + '.') for n in numbers)
        add(doc, number, number, title or own[1][0], lines[start:end], parent,
            'section' if section else 'clause', source_number=number)
        markers = [(j, 'letter', LETTER.match(line).group(1)) if LETTER.match(line)
                   else (j, 'subitem', SUBITEM.match(line).group(1))
                   for j, (line, _) in enumerate(own) if LETTER.match(line) or SUBITEM.match(line)]
        letter_parent = number
        for k, (at, kind, label) in enumerate(markers):
            stop = markers[k + 1][0] if k + 1 < len(markers) else len(own)
            parent = number if kind == 'letter' else letter_parent
            locator = parent + '(' + label + ')'
            if kind == 'letter':
                letter_parent = locator
                stop = next((j for j, t, _ in markers[k + 1:] if t == 'letter'), len(own))
            add(doc, locator, locator, own[at][0], own[at:stop], parent, 'list_item', source_number=label + '）')
        if number == '2':
            ref = [(line, p) for line, p in own if line.startswith('GB 44495—2024')]
            add(doc, '2.reference1', 'GB 44495—2024', ref[0][0], ref, '2', 'reference')

    doc = 'gb-44496-2024-amd1'
    amendment = lines_on(reader, 14)
    starts = [(i, line[0]) for i, (line, _) in enumerate(amendment) if re.match(r'^[一二三四五]、', line)]
    if [n for _, n in starts] != list('一二三四五'):
        raise ValueError('Unexpected amendment structure')
    add(doc, 'publication', '批准与实施', '批准与实施日期', amendment[:starts[0][0]], kind='publication_note',
        publication_date='2026-01-28', effective_date='2026-01-28')
    targets = [['1'], ['4', '4.1.1', '3.5'], ['6.5', '6.6', '6.7', '6.8', '6.9', '6.10', '6.12', '6.13'],
               ['7.1.1(b)', '7.1.2(b)'], ['9']]
    for i, (at, n) in enumerate(starts):
        stop = starts[i + 1][0] if i + 1 < len(starts) else len(amendment)
        add(doc, n, n, amendment[at][0], amendment[at:stop], kind='amendment_item', source_number=n + '、',
            amends_keys=['reg:gb-44496-2024:' + target for target in targets[i]])
    return rows, numbers


def main(source=SOURCE):
    source = Path(source)
    if hashlib.sha256(source.read_bytes()).hexdigest() != SOURCE_SHA:
        raise ValueError('Source differs from reviewed 15-page PDF')
    reader = PdfReader(source)
    if len(reader.pages) != 15:
        raise ValueError('Unexpected page count')
    rows, numbers = build(reader)
    shutil.copyfile(source, HERE / 'sources' / 'gb-44496-2024.pdf')
    patches = []
    for doc in ('gb-44496-2024', 'gb-44496-2024-amd1'):
        selected = [r for r in rows if r['document_id'] == doc]
        path = HERE / (doc + '.json')
        path.write_text(json.dumps(selected, ensure_ascii=False, indent=2) + '\n')
        amendment = doc.endswith('amd1')
        patch = {'id': doc, 'coverage': 'complete_for_edition', 'clause_count': len(selected),
                 'status': '正式第 1 号修改单（用户提供原件）' if amendment else '已收录原始版；第 1 号修改单单独收录',
                 'clauses_file': path.name, 'clauses_sha256': hashlib.sha256(path.read_bytes()).hexdigest(),
                 'source_note': '正文来自用户提供的 GB+44496-2024.pdf；官方链接仅用于核对标准目录，并非本次 PDF 下载来源。',
                 'coverage_note': '用户提供正式 PDF 的第 14 页，5 项修改分别入库。' if amendment else
                                  '原始版 9 章、全部编号段落、字母分项及两项嵌套数字分项已入库；修改单未覆盖原文。',
                 'coverage_evidence': {'source_kind': 'user_provided_pdf', 'pdf_total_pages': 15,
                     'source_pages': [14] if amendment else [5] + list(range(7, 14)), 'blank_pages': [2],
                     'non_content_pages': [4, 6, 15],
                     'numbered_nodes': 5 if amendment else len(numbers), 'chapter_count': 0 if amendment else 9,
                     'table_count': 0, 'annex_count': 0, 'node_kinds': dict(Counter(r['kind'] for r in selected)),
                     'note': '直接提取原文，核对跨页内容、目录和原始编号；正文无表格及附录。原文第7章直接列7.1.1、7.1.2，不补造7.1。条号和汉字排版间隙规范化，原件保持不变。第4、6、15页无正文，仅有底部浅色防伪标识。'},
                 'artifacts': [{'file': 'sources/gb-44496-2024.pdf', 'sha256': SOURCE_SHA,
                                'media_type': 'application/pdf', 'source_kind': 'user_provided',
                                'source_name': source.name, 'source_url': URL}], 'required_source': None}
        if amendment:
            patch.update(publication_date='2026-01-28', effective_date='2026-01-28', amends='gb-44496-2024')
        patches.append(patch)
        print(doc, len(selected), patch['coverage_evidence']['node_kinds'])
    (WORK / 'gb44496-manifest-patch.json').write_text(json.dumps(patches, ensure_ascii=False, indent=2) + '\n')


if __name__ == '__main__':
    main(sys.argv[1] if len(sys.argv) > 1 else SOURCE)
