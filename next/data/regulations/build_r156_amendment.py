"""Build the separately versioned, adopted R156 01-series amendment offline.

The three official PDFs are the evidence chain: proposal text + the report's
adopted modifications + the depositary's entry-into-force confirmation.
Never overwrite the 2021 base edition or update the shared manifest here.
"""
import hashlib
import json
from pathlib import Path
import re
from pypdf import PdfReader

HERE = Path(__file__).resolve().parent
WORK = HERE.parents[2] / '.work' / 'regulations'
DOCUMENT = 'un-r156-01-series-2026'
PREFIX = 'reg:' + DOCUMENT + ':'
FILES = {
    'text': ('un-r156-01-series-2026-amendment.pdf', 3,
             'https://documents.un.org/api/symbol/access?l=en&s=ECE%2FTRANS%2FWP.29%2F2025%2F142&t=pdf'),
    'report': ('un-r156-01-series-2026-report.pdf', 43,
               'https://documents.un.org/api/symbol/access?l=en&s=ECE%2FTRANS%2FWP.29%2F1188&t=pdf'),
    'notice': ('un-r156-01-series-2026-notice.pdf', 1,
               'https://treaties.un.org/doc/Publication/CN/2026/CN.237.2026-Eng.pdf'),
}
NUMBER = re.compile(r'^"?(\d+(?:\.\d+)*)\.\s+(.+)')
SOURCE_DIGESTS = {
    'text': '67a1ce6d9ee791c207ef04ea6cd5b05ddc1206678297651547f687a0243bfdc2',
    'report': 'e0d44605f40b3bee9fcaf38e47fcb72184a005e1145bc8e01abf717cd321bbde',
    'notice': '3a79f64befd17f461284ea2dfce119345ed36b116727c14ffdc696eedd2bf5d6',
}


def clean(line):
    # RXSWIN is printed with a subscript X; pypdf inserts a layout-only space.
    return re.sub(r'\s+', ' ', line).strip().replace('R XSWIN', 'RXSWIN')


def lines_for(reader, page):
    lines = [clean(line) for line in reader.pages[page - 1].extract_text().splitlines() if line.strip()]
    lines = [line for line in lines if line not in {'ECE/TRANS/WP.29/2025/142', str(page)}]
    return [(line, page) for line in lines]


def make_rows(directory=HERE):
    readers, artifacts = {}, []
    for kind, (filename, count, url) in FILES.items():
        path = directory / 'sources' / filename
        raw = path.read_bytes()
        if hashlib.sha256(raw).hexdigest() != SOURCE_DIGESTS[kind]:
            raise ValueError('Source differs from the reviewed official PDF: ' + filename)
        reader = PdfReader(path)
        if len(reader.pages) != count:
            raise ValueError('Unexpected source page count: ' + filename)
        readers[kind] = reader
        artifacts.append({'file': 'sources/' + filename, 'sha256': hashlib.sha256(raw).hexdigest(),
                          'media_type': 'application/pdf', 'source_url': url})
    rows = []

    def add(locator, title, lines, parent=None, kind='amendment_item', source='text', number=None, **extra):
        source_file, _, url = FILES[source]
        body = '\n'.join(line for line, _ in lines)
        rows.append({'key': PREFIX + locator, 'document_id': DOCUMENT, 'number': number or locator,
                     'title': title, 'body': body, 'description': body,
                     'parent': PREFIX + parent if parent else None, 'kind': kind,
                     'normative': False if kind == 'publication_note' else None,
                     'normative_note': '已通过的修订原文，与原始系列分开保存；不自动生成合订文本或合规结论。',
                     'language': 'en', 'source_url': url, 'source_file': 'sources/' + source_file,
                     'source_sha256': next(a['sha256'] for a in artifacts if a['file'] == 'sources/' + source_file),
                     'source_pages': sorted({page for _, page in lines}), **extra})

    add('publication', 'Text publication and document identification', lines_for(readers['text'], 1),
        kind='publication_note', number='出版说明')
    lines = lines_for(readers['text'], 2) + lines_for(readers['text'], 3)
    add('amendment', 'Adopted amendment text — read together with report 1188, paragraph 78(c)', lines,
        kind='section', number='01 系列修订正文')

    # Every numbered item in the two-page amendment body is separately selectable.
    # The entire source above also retains all insertion/replacement directions.
    chunks, current = [], None
    annex = None
    renumber_index = 0
    for line, page in lines:
        annex_match = re.match(r'^Annex ([12])\., amend to read:', line)
        if annex_match:
            current = None
            annex = 'Annex' + annex_match.group(1)
            continue
        if line.startswith(('Insert new ', 'Paragraph ', 'Paragraphs ')):
            current = None
            if line.startswith('Paragraphs '):
                renumber_index += 1
                add('renumber' + str(renumber_index), 'Renumbering instruction', [(line, page)],
                    parent='amendment', kind='renumbering', number='重新编号 ' + str(renumber_index))
            continue
        match = NUMBER.match(line)
        if match:
            original_number, title = match.groups()
            # The official amendment displays old and new Annex 2 numbers together
            # (e.g. 9.11.). Keep that exact source label, and expose the new number.
            final_number = original_number
            if annex == 'Annex2' and re.fullmatch(r'(?:9|1[0-6])\.(?:1[1-8])', original_number):
                old, new = map(int, original_number.split('.'))
                if new != old + 2:
                    raise ValueError('Unexpected Annex 2 renumbering')
                final_number = str(new)
            locator = (annex + '.' if annex else '') + final_number
            current = {'locator': locator, 'number': final_number, 'original': original_number,
                       'title': title.rstrip('"'), 'annex': annex, 'lines': [(line, page)]}
            chunks.append(current)
        elif current and line not in {'"…', '…'}:
            current['lines'].append((line, page))
    locators = {chunk['locator'] for chunk in chunks}
    if len(locators) != len(chunks):
        raise ValueError('Duplicate numbered amendment item')
    for annex_id in ('Annex1', 'Annex2'):
        annex_lines = [item for c in chunks if c['annex'] == annex_id for item in c['lines']]
        add(annex_id, annex_id.replace('Annex', 'Annex ') + ' — amended entries', annex_lines,
            parent='amendment', kind='section', number=annex_id.replace('Annex', 'Annex '))
    for chunk in chunks:
        locator, annex_id, number = chunk['locator'], chunk['annex'], chunk['number']
        parent_number = number.rsplit('.', 1)[0] if '.' in number else None
        candidate = (annex_id + '.' if annex_id else '') + parent_number if parent_number else None
        parent = candidate if candidate in locators else (annex_id or 'amendment')
        children = [c for c in chunks if c['locator'].startswith(locator + '.')]
        body_lines = chunk['lines'] + [item for c in children for item in c['lines']]
        add(locator, chunk['title'][:160], body_lines, parent=parent,
            kind='section' if children else 'amendment_item',
            number=(annex_id.replace('Annex', 'Annex ') + ' · ' if annex_id else '') + number,
            source_number=chunk['original'], target_number=number,
            target_annex=annex_id, renumbered_from=(chunk['original'].split('.')[0]
                if annex_id == 'Annex2' and chunk['original'] != number else None))

    # These are binding modifications to the earlier proposal, not optional notes.
    report = clean(readers['report'].pages[17].extract_text())
    at = report.index('(c) In ECE/TRANS/WP.29/2025/142, Appendix 1 to Annex 1,')
    end = report.index(' K. Consideration of draft corrigenda', at)
    modification = report[at:end]
    if 'UN Regulation No. 156 series of amendments number:' not in modification:
        raise ValueError('Missing adopted Appendix 1 modification')
    if 'provisions of the 01 series of amendments to Regulation No. 156.' not in modification:
        raise ValueError('Missing adopted Annex 4 modification')
    add('adopted-78c', 'Adopted modifications — report 1188, paragraph 78(c)', [(modification, 18)],
        number='通过时修改 78(c)', kind='section', source='report')
    first, second = modification.split(' and in Annex 4, ', 1)
    add('Annex1.Appendix1.series', 'Appendix 1 to Annex 1 — series number field', [(first, 18)],
        parent='adopted-78c', source='report', number='Annex 1 · Appendix 1 · 系列号字段',
        source_number='78(c)', target_annex='Annex1.Appendix1')
    add('Annex4.series', 'Annex 4 — explicit reference to the 01 series', [('In Annex 4, ' + second, 18)],
        parent='adopted-78c', source='report', number='Annex 4 · 01 系列引用',
        source_number='78(c)', target_annex='Annex4')
    notice = clean(readers['notice'].pages[0].extract_text())
    if 'as of 4 June 2026' not in notice or 'as amended by paragraph 78(c)' not in notice:
        raise ValueError('Missing official adoption / entry-into-force evidence')
    add('entry-into-force', 'Depositary confirmation — entry into force on 4 June 2026', [(notice, 1)],
        kind='publication_note', source='notice', number='生效通知', effective_date='2026-06-04')
    expected = {'7.1.1.3', '7.1.4', '7.2.1.2', '13', *('13.' + str(i) for i in range(1, 7)),
                *('Annex1.' + n for n in ['9', '9.1', '9.2', '9.3', '9.3.1', '9.3.2',
                    '9.4', '9.4.1', '9.4.2', '9.5', '9.5.1', '9.5.2']),
                *('Annex2.' + str(i) for i in range(8, 19))}
    if locators != expected or renumber_index != 2:
        raise ValueError('Unexpected numbered amendment coverage: ' + repr(locators ^ expected))
    if len({row['key'] for row in rows}) != len(rows):
        raise ValueError('Duplicate final key')
    return rows, artifacts


def make_document(rows, artifacts, clauses_sha256):
    doc = {'id': DOCUMENT, 'code': 'UN R156', 'title': '车辆软件更新与软件更新管理体系（01 系列修订）',
           'edition': '01 系列修订（2026-06-04 生效）',
           'status': '已通过并生效；修订文本，未与原始系列合并', 'coverage': 'complete_for_edition',
           'clause_count': len(rows), 'language': 'en', 'publication_date': '2025-08-28',
           'effective_date': '2026-06-04', 'adoption_confirmation_date': '2026-06-12',
           'publisher': 'United Nations Economic Commission for Europe / United Nations Treaty Section',
           'verified_at': '2026-10-25', 'source_url': FILES['text'][2],
           'status_source_url': FILES['notice'][2], 'document_type': 'amendment',
           'amends': 'un-r156-2021-388',
           'coverage_note': '独立收录 01 系列全部修订条文、两组重新编号指示及会议报告 78(c) 的两项通过时修改；附生效通知。不是整部 R156 的合订全文，原始系列保留独立版本。',
           'legal_notice': '修订正文文件封面保留 Proposal 原题；其通过及生效由会议报告 ECE/TRANS/WP.29/1188 第 78(c) 段和联合国保存人通知 C.N.237.2026 共同确认。',
           'clauses_file': DOCUMENT + '.json', 'clauses_sha256': clauses_sha256,
           'coverage_evidence': {'format': 'Official UN PDFs with paragraph-by-paragraph extraction',
               'amendment_text_pages': 3, 'amendment_numbered_entries': 33,
               'annex1_numbered_entries': 12, 'annex2_numbered_entries': 11,
               'renumbering_instructions': 2, 'adopted_modifications': 2,
               'adopted_modifications_locator': 'ECE/TRANS/WP.29/1188, page 18, paragraph 78(c)',
               'entry_into_force_evidence': 'C.N.237.2026, page 1: binding as of 4 June 2026',
               'note': 'All numbered entries and insertion/replacement directions on amendment pages 2–3 retained. Printed old/new Annex 2 labels preserved in source_number; target_number exposes final label. Read together with the two adopted modifications. No automatic consolidation.'},
           'artifacts': artifacts}
    return doc


def main():
    rows, artifacts = make_rows()
    output = HERE / (DOCUMENT + '.json')
    output.write_text(json.dumps(rows, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    doc = make_document(rows, artifacts, hashlib.sha256(output.read_bytes()).hexdigest())
    WORK.mkdir(parents=True, exist_ok=True)
    (WORK / 'r156-amendment-manifest-patch.json').write_text(json.dumps(doc, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(f'Wrote {len(rows)} rows, source PDFs verified; manifest unchanged.')


if __name__ == '__main__':
    main()
