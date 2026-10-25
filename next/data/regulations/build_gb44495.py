"""Offline import of the user-provided GB 44495 PDF, including its amendment.

The original 2024 text and the separately printed 2026 amendment retain distinct
document IDs. This script deliberately does not update the shared manifest.
"""
from collections import Counter
import hashlib
import json
from pathlib import Path
import re
import shutil
import sys

from pypdf import PdfReader

HERE = Path(__file__).resolve().parent
WORK = HERE.parents[2] / '.work' / 'regulations'
SOURCE = Path('provide-your-authorized-input.pdf')
EXPECTED_SHA256 = '36429113332b3a942f92ad117b35a6fe3e45fece311c6ff032421dd246bfaf3b'
SOURCE_URL = 'https://openstd.samr.gov.cn/bzgk/std/newGbInfo?hcno=2DB552CAA58F589705C3DC7AD47AC2AB'
NUMBER = re.compile(r'^(\d+(?:\.\d+)*)(?:\s+(.*)|$)')
LETTER = re.compile(r'^([a-z])\)\s*(.*)')
CHINESE = '\u3400-\u9fff'


def normalize(line):
    line = line.strip().replace('\u00a0', ' ')
    # Remove layout justification gaps, retaining actual Latin word separators.
    line = re.sub(r'(?<=[' + CHINESE + r'])\s+(?=[' + CHINESE + r'])', '', line)
    line = re.sub(r'(?<=[,，、;；:：。(（])\s+(?=[' + CHINESE + r'])', '', line)
    return line


def source_lines(reader, physical_page):
    lines = [normalize(s) for s in reader.pages[physical_page - 1].extract_text().splitlines() if s.strip()]
    lines = [s for s in lines if not re.fullmatch(r'GB\s*44495[—-]2024', s)]
    if lines and re.fullmatch(r'[\dⅠⅡⅢⅣⅤ]+', lines[-1]):
        lines.pop()  # Printed page numbers are not clause numbers.
    return [(s, physical_page) for s in lines]


def body_text(lines):
    # Preserve paragraph/list boundaries while joining typesetting line wraps.
    paragraphs, current = [], ''
    break_after, term_title_next = False, False
    for line, _ in lines:
        new = (NUMBER.match(line) or LETTER.match(line) or line.startswith(('———', '●', '注', '示例', '[来源'))
               or re.match(r'^(?:GB|[A-Z0-9]+:)', line))
        if current and (new or break_after or current.endswith(('。', ';', '；', ':', '：', ']'))):
            paragraphs.append(current)
            current = ''
        separator = ' ' if current and current[-1].isascii() and line[0].isascii() else ''
        current += separator + line
        match = NUMBER.match(line)
        heading = match.group(2) if match else None
        break_after = bool(match and (not heading or (len(heading) < 50 and not re.search(r'[。,:;，；：]', heading))))
        if term_title_next:
            break_after, term_title_next = True, False
        if match and re.fullmatch(r'3\.\d+', match.group(1)) and not heading:
            term_title_next = True
    if current:
        paragraphs.append(current)
    return '\n'.join(paragraphs)


def build_rows(reader, source_sha):
    rows, seen = [], set()

    def add(document_id, locator, number, title, lines, parent=None, kind='clause', source_number=None, **extra):
        key = 'reg:' + document_id + ':' + locator
        if key in seen:
            raise ValueError('Duplicate imported key: ' + key)
        seen.add(key)
        pages = sorted({page for _, page in lines})
        text = body_text(lines)
        rows.append({'key': key, 'document_id': document_id, 'number': number, 'source_number': source_number,
                     'title': title, 'body': text, 'description': text,
                     'parent': ('reg:' + document_id + ':' + parent) if parent else None,
                     'kind': kind, 'normative': False if kind in {'publication_note', 'bibliography'} else None,
                     'normative_note': '强制性标准原文。注、示例及适用条件随条款保留，不自动转换为独立合规义务。',
                     'language': 'zh-CN', 'source_url': SOURCE_URL,
                     'source_file': 'sources/gb-44495-2024.pdf', 'source_sha256': source_sha,
                     'page': pages[0], 'page_end': pages[-1],
                     'print_page': str(pages[0] - 6) if pages[0] >= 7 else 'Ⅲ',
                     'source_pages': pages, 'printed_pages': [str(p - 6) if p >= 7 else 'Ⅲ' for p in pages],
                     **extra})

    document_id = 'gb-44495-2024'
    front = source_lines(reader, 5)
    add(document_id, 'foreword', '前言', '前言', front, kind='publication_note')
    lines = [line for page in range(7, 21) for line in source_lines(reader, page)]
    if lines[0][0] != '汽车整车信息安全技术要求':
        raise ValueError('Unexpected opening page')
    lines = lines[1:]
    headings = [(index, NUMBER.match(line).group(1), NUMBER.match(line).group(2) or '')
                for index, (line, _) in enumerate(lines) if NUMBER.match(line)]
    numbers = {number for _, number, _ in headings}
    if len(numbers) != len(headings):
        raise ValueError('Duplicate numeric clause labels')
    for position, (start, number, heading) in enumerate(headings):
        own_end = headings[position + 1][0] if position + 1 < len(headings) else len(lines)
        end = own_end
        for next_start, next_number, _ in headings[position + 1:]:
            if not next_number.startswith(number + '.'):
                end = next_start
                break
        else:
            end = len(lines)
        own = lines[start:own_end]
        parent = number.rsplit('.', 1)[0] if '.' in number else None
        if parent and parent not in numbers:
            raise ValueError('Missing numbered parent: ' + number)
        section = any(n.startswith(number + '.') for n in numbers)
        title = heading or own[1][0]
        add(document_id, number, number, title[:120], lines[start:end], parent,
            'section' if section else 'clause', source_number=number)
        # Each lettered or visually bulleted item is independently referencable.
        markers = [(i, 'letter', LETTER.match(line).group(1)) if LETTER.match(line)
                   else (i, 'dash', None) if line.startswith('———')
                   else (i, 'dot', None)
                   for i, (line, _) in enumerate(own) if LETTER.match(line) or line.startswith(('———', '●'))]
        dash_count, dot_count, dash_parent = 0, 0, number
        for index, (at, style, letter) in enumerate(markers):
            stop = markers[index + 1][0] if index + 1 < len(markers) else len(own)
            item_parent = number
            if style == 'letter':
                locator = number + '(' + letter + ')'
                displayed, original = number + '(' + letter + ')', letter + ')'
            elif style == 'dash':
                dash_count += 1
                dot_count = 0
                locator = number + '.list' + str(dash_count)
                dash_parent = locator
                displayed, original = number + ' · 第' + str(dash_count) + '列项（原文无编号）', None
                # Include all subordinate bullet lines when citing their enclosing dash item.
                stop = next((j for j, kind, _ in markers[index + 1:] if kind != 'dot'), len(own))
            else:
                dot_count += 1
                item_parent = dash_parent
                locator = dash_parent + '.item' + str(dot_count)
                displayed, original = number + ' · 子列项' + str(dot_count) + '（原文无编号）', None
            item_lines = own[at:stop]
            add(document_id, locator, displayed, item_lines[0][0][:120], item_lines, item_parent,
                'list_item', source_number=original, source_marker={'letter': original, 'dash': '———', 'dot': '●'}[style])
        # Unnumbered normative references and abbreviations retain their printed labels.
        if number in {'2', '4'}:
            for i, (line, page) in enumerate(own):
                match = re.match(r'^(GB(?:/T)?\d+(?:—\d+)?)\s+(.+)', line) if number == '2' else re.match(r'^([A-Z0-9]+):(.+)', line)
                if match:
                    label = match.group(1)
                    locator = number + '.entry' + str(i)
                    add(document_id, locator, label, line, [(line, page)], number,
                        'reference' if number == '2' else 'definition', source_number=None)
    add(document_id, 'bibliography', '参考文献', '参考文献', source_lines(reader, 21), kind='bibliography')

    # The formal amendment is a separate document, never silently applied to the original.
    document_id = 'gb-44495-2024-amd1'
    amendment = source_lines(reader, 22)
    starts = [(i, re.match(r'^([一二三四五])、', line).group(1)) for i, (line, _) in enumerate(amendment)
              if re.match(r'^([一二三四五])、', line)]
    if [s[1] for s in starts] != list('一二三四五'):
        raise ValueError('Unexpected amendment structure')
    add(document_id, 'publication', '批准与实施', '批准与实施日期', amendment[:starts[0][0]], kind='publication_note',
        publication_date='2026-01-28', effective_date='2026-01-28')
    targets = [['1'], ['5', '6.1', '5.1', '5.2', '8.1', '3.2'], ['8', '8.2.1.1', '8.2.1.2', '8.2.1.3', '8.2.1.4'],
               ['9.1', '9.2'], ['10']]
    for i, (at, number) in enumerate(starts):
        stop = starts[i + 1][0] if i + 1 < len(starts) else len(amendment)
        add(document_id, number, number, amendment[at][0][:120], amendment[at:stop], kind='amendment_item',
            source_number=number + '、', amends_keys=['reg:gb-44495-2024:' + n for n in targets[i]])
    return rows, numbers


def main(source=SOURCE):
    source = Path(source)
    raw = source.read_bytes()
    source_sha = hashlib.sha256(raw).hexdigest()
    if EXPECTED_SHA256 and source_sha != EXPECTED_SHA256:
        raise ValueError('Source PDF differs from the reviewed import')
    reader = PdfReader(source)
    if len(reader.pages) != 23:
        raise ValueError('Expected the reviewed 23-page PDF')
    rows, numbers = build_rows(reader, source_sha)
    target = HERE / 'sources' / 'gb-44495-2024.pdf'
    target.parent.mkdir(exist_ok=True)
    shutil.copyfile(source, target)
    patches = []
    for document_id in ('gb-44495-2024', 'gb-44495-2024-amd1'):
        selected = [row for row in rows if row['document_id'] == document_id]
        out = HERE / (document_id + '.json')
        out.write_text(json.dumps(selected, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
        amendment = document_id.endswith('amd1')
        patch = {'id': document_id, 'coverage': 'complete_for_edition',
                 'status': '正式第 1 号修改单（用户提供原件）' if amendment else '已收录原始版；第 1 号修改单单独收录',
                 'clause_count': len(selected), 'clauses_file': out.name,
                 'source_note': '正文来自用户提供的 GB+44495-2024.pdf；官方链接仅用于核对标准目录，并非本次 PDF 下载来源。',
                 'clauses_sha256': hashlib.sha256(out.read_bytes()).hexdigest(),
                 'coverage_note': '用户提供正式 PDF 的第 22 页，5 项修改分别入库。' if amendment else
                                  '原始版 10 章、全部编号段落、字母分项、无编号列项、缩略语、前言和参考文献已入库；修改单未覆盖原文。',
                 'coverage_evidence': {'source_kind': 'user_provided_pdf', 'pdf_total_pages': 23,
                     'source_pages': [22] if amendment else [5] + list(range(7, 22)),
                     'blank_pages': [2, 4, 6, 23], 'numbered_nodes': 5 if amendment else len(numbers),
                     'chapter_count': 0 if amendment else 10, 'table_count': 0, 'annex_count': 0,
                     'node_kinds': dict(Counter(row['kind'] for row in selected)),
                     'note': '正文及修改单均可直接提取文本；核对全部章节及跨页分项。原文无表格和附录。排版字间空隙合并；原文无编号列项使用明确标注的内部定位符。'},
                 'artifacts': [{'file': 'sources/gb-44495-2024.pdf', 'sha256': source_sha,
                                'media_type': 'application/pdf', 'source_kind': 'user_provided',
                                'source_name': source.name, 'source_url': SOURCE_URL}], 'required_source': None}
        if amendment:
            patch.update(publication_date='2026-01-28', effective_date='2026-01-28', amends='gb-44495-2024')
        patches.append(patch)
        print(document_id, len(selected), patch['coverage_evidence']['node_kinds'])
    WORK.mkdir(exist_ok=True)
    (WORK / 'gb44495-manifest-patch.json').write_text(json.dumps(patches, ensure_ascii=False, indent=2) + '\n')
    print('source_sha256', source_sha)


if __name__ == '__main__':
    main(sys.argv[1] if len(sys.argv) > 1 else SOURCE)
