"""Rebuild reviewed UN bundles from official Publications Office Formex archives.

Offline only. Run from the repository root after placing the named source files
under .work/regulations. Original archives/PDFs are preserved in sources/.
"""
from collections import Counter
import hashlib
import json
from pathlib import Path
import re
import shutil
import xml.etree.ElementTree as ET
import zipfile

HERE = Path(__file__).resolve().parent
WORK = HERE.parents[2] / '.work' / 'regulations'


def text(element):
    if element is None:
        return ''
    if element.tag in {'BIB.INSTANCE', 'TOC'}:
        return ''
    if element.tag in {'QUOT.START', 'QUOT.END'}:
        return chr(int(element.get('CODE'), 16))
    if element.tag == 'INCL.ELEMENT':
        if element.get('TYPE') == 'FORMEX.DOC':
            return ' [See the separately indexed appendix in this publication.] '
        return ' [Figure: see the preserved official PDF.] '
    if element.tag == 'NOTE' and element.get('NOTE.REF'):
        return ' [Footnote reference: ' + element.get('NOTE.REF') + '] '
    value = element.text or ''
    for child in element:
        value += text(child) + (child.tail or '')
    if element.tag == 'NOTE':
        value = ' [Footnote: ' + value + '] '
    if element.tag in {'P', 'NP', 'NO.P', 'ROW', 'CELL', 'TITLE', 'TI', 'STI', 'TXT', 'GR.SEQ', 'ITEM'}:
        value += '\n'
    return value


def clean(element):
    return re.sub(r'[ \t\r\f\v]+', ' ', text(element)).replace('\u00a0', ' ').strip()


def compact(element):
    return ' '.join(clean(element).split())


def parse_archive(archive, document_id):
    rows, seen, source_units, table_count = [], set(), Counter(), 0
    prefix = 'reg:' + document_id + ':'

    def add(number, title, body, parent, kind, source_file, **extra):
        if number in seen:
            raise ValueError('Duplicate source numbering: ' + number)
        seen.add(number)
        row = {'key': prefix + number, 'document_id': document_id, 'number': number,
               'title': title or number, 'body': body or title or number,
               'description': body or title or number,
               'parent': prefix + parent if parent else None, 'kind': kind,
               'normative': None, 'normative_note': '法规原文；规范性由所在条款和适用条件决定，未自动作合规判断。',
               'source_file': source_file, **extra}
        rows.append(row)
        return number

    def walk(node, scope, parent, file, ancestors=()):
        nonlocal table_count
        if node.tag in {'TITLE', 'BIB.INSTANCE', 'TOC'}:
            return
        if node.tag == 'TBL':
            table_count += 1
            heading = compact(node.find('TITLE'))
            match = re.search(r'Table\s+([A-Z]\d+)', heading)
            table_id = scope + '.Table' + (match.group(1) if match else str(table_count))
            columns = int(node.get('COLS', '1'))
            inherited, grid, headers = {}, [], []
            for row in node.findall('./CORPUS/ROW'):
                cells = [''] * columns
                for col, (value, remaining) in list(inherited.items()):
                    cells[col] = value
                    if remaining <= 1:
                        del inherited[col]
                    else:
                        inherited[col] = (value, remaining - 1)
                for cell in row.findall('CELL'):
                    start = int(cell.get('COL', '1')) - 1
                    value = compact(cell)
                    for col in range(start, min(columns, start + int(cell.get('COLSPAN', '1')))):
                        cells[col] = value
                        if int(cell.get('ROWSPAN', '1')) > 1:
                            inherited[col] = (value, int(cell.get('ROWSPAN')) - 1)
                if row.get('TYPE') == 'HEADER':
                    headers = cells
                else:
                    grid.append(cells)
            add(table_id, heading or 'Table', clean(node), parent, 'table', file,
                columns=headers, row_count=len(grid))
            for i, cells in enumerate(grid, 1):
                actual = cells[-2] if len(cells) >= 2 and re.fullmatch(r'(?:M\s*\d+|\d+(?:\.\d+)+)', cells[-2]) else ''
                label = (actual + ' · ' if actual else '') + next((c for c in reversed(cells) if c), '')
                body = '\n'.join((headers[j] + ': ' if j < len(headers) and headers[j] else '') + c
                                 for j, c in enumerate(cells) if c)
                add(table_id + '.row' + str(i), label, body, table_id, 'table_row', file,
                    cells=cells, source_number=actual or None, source_row=i)
            source_units['table_rows'] += len(grid)
            return
        if node.tag == 'GR.SEQ':
            title = node.find('TITLE')
            no = title.find('.//NO.P') if title is not None else None
            raw = compact(no).rstrip('.')
            if raw:
                number = (scope + '.' if scope else '') + raw
            else:
                first = compact(title.find('TI')) if title is not None else ''
                number = (scope + '.' if scope else '') + re.sub(r'[^A-Za-z0-9]+', '', first.title())
            if number and number not in seen:
                parent = add(number, compact(title), clean(node), parent, 'section', file)
            # Annex part subclauses restart numbering; retain their exact part scope.
            if not raw and number:
                scope = number
        elif node.tag == 'NP':
            raw = compact(node.find('NO.P')).rstrip('.')
            if raw:
                if raw.startswith('('):
                    number = parent + raw
                else:
                    number = (scope + '.' if scope else '') + raw
                parent_number = parent
                if not raw.startswith('(') and '.' in raw:
                    candidate = (scope + '.' if scope else '') + raw.rsplit('.', 1)[0]
                    if candidate in seen:
                        parent_number = candidate
                # Titles without explicit subsection wrappers are still real clause nodes.
                parent = add(number, compact(node.find('TXT'))[:180], clean(node), parent_number,
                             'clause', file, source_number=raw)
                source_units['numbered_paragraphs'] += 1
        for child in node:
            walk(child, scope, parent, file, ancestors + (node.tag,))

    with zipfile.ZipFile(archive) as bundle:
        documents = []
        for file in bundle.namelist():
            if not file.endswith('.xml'):
                continue
            root = ET.fromstring(bundle.read(file))
            if root.tag in {'GENERAL', 'ANNEX'}:
                documents.append((file, root))
        # GENERAL first, followed by source annex order, independent of ZIP member order.
        def sort_key(item):
            root, title = item[1], compact(item[1].find('TITLE'))
            if root.tag == 'GENERAL':
                return (0, 0)
            annex = re.search(r'Annex\s+(\d+)', title, re.I)
            appendix = re.search(r'Appendix\s+(\d+)', title, re.I)
            return (int(annex.group(1)) if annex else 99, int(appendix.group(1)) if appendix else 0)
        for file, root in sorted(documents, key=sort_key):
            title = compact(root.find('TITLE'))
            if root.tag == 'GENERAL':
                scope, parent = '', None
                annotations = clean(root.find('GR.ANNOTATION'))
                preface = '\n'.join(clean(c) for c in root.find('CONTENTS') if c.tag not in {'TOC', 'GR.SEQ'})
                add('preface', 'Publication and edition notice', annotations + '\n' + preface,
                    None, 'publication_note', file)
            else:
                annex = re.search(r'Annex\s+(\d+)', title, re.I)
                appendix = re.search(r'Appendix\s+(\d+)', title, re.I)
                scope = 'Annex' + annex.group(1)
                parent = scope if appendix else None
                if appendix:
                    scope += '.Appendix' + appendix.group(1)
                add(scope, title, clean(root.find('CONTENTS')), parent, 'annex', file)
                parent = scope
            walk(root.find('CONTENTS'), scope, parent, file)
    return rows, dict(source_units), len(documents)


def main():
    manifest = json.loads((HERE / 'manifest.json').read_text())
    (HERE / 'sources').mkdir(exist_ok=True)
    for short, document_id in [('r155', 'un-r155-2025-5'), ('r156', 'un-r156-2021-388')]:
        doc = next(d for d in manifest['documents'] if d['id'] == document_id)
        rows, units, parts = parse_archive(WORK / (short + '.fmx4'), document_id)
        path = HERE / (document_id + '.json')
        path.write_text(json.dumps(rows, ensure_ascii=False, indent=2) + '\n')
        doc.update(clauses_file=path.name, clauses_sha256=hashlib.sha256(path.read_bytes()).hexdigest(),
                   clause_count=len(rows), coverage='complete_for_edition',
                   coverage_evidence={'format': 'Official Formex XML and matching official PDF',
                                      'xml_parts': parts, **units,
                                      'note': 'All numbered paragraphs, lettered subitems, sections, annexes and table rows retained. Figures remain in the preserved official PDF; table rows expand source merged cells.'})
        artifacts = []
        for ext, suffix in [('fmx4', 'fmx4.zip'), ('pdfa2a', 'pdf')]:
            target = HERE / 'sources' / (document_id + '.' + suffix)
            shutil.copyfile(WORK / (short + '.' + ext), target)
            artifacts.append({'file': str(target.relative_to(HERE)), 'sha256': hashlib.sha256(target.read_bytes()).hexdigest(),
                              'media_type': 'application/pdf' if suffix == 'pdf' else 'application/zip',
                              'source_url': doc['download_url'].replace('format=fmx4', 'format=' + ext)})
        doc['artifacts'] = artifacts
        print(document_id, len(rows), units)
    (HERE / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + '\n')


if __name__ == '__main__':
    main()
