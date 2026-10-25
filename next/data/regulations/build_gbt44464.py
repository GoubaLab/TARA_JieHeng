"""Rebuild GB/T 44464 from the fixed, publicly reposted 24-page PDF, offline.

Needs pdfplumber only when rebuilding. The application reads the resulting JSON.
The shared manifest is deliberately untouched; a standalone replacement document
record is written to .work/regulations/gbt44464-manifest-patch.json for review.
"""
from collections import Counter
import hashlib
import json
from pathlib import Path
import re
import shutil
import sys

HERE = Path(__file__).resolve().parent
WORK = HERE.parents[2] / '.work' / 'regulations'
DOCUMENT = 'gbt-44464-2024'
PREFIX = 'reg:' + DOCUMENT + ':'
EXPECTED_SHA256 = '6f2edc2abd2e42788c72991b753fc4a73eb320195426b41bfd83c448689d4602'
LANDING = 'https://www.olsrz.com/contents/28/597.html'
DOWNLOAD = 'https://www.olsrz.com/upload/files/2026/5/fdac1d1271afdfb8.PDF'
OFFICIAL = 'https://openstd.samr.gov.cn/bzgk/std/newGbInfo?hcno=D63AAC0203E9B169F74B10E547A3CBCE'
SOURCE_NOTE = '公开转载 PDF，非标准发布机构直供；版本及发布日期经官方目录核对，正文尚未与官方副本逐字比对。'
CLAUSE = re.compile(r'^((?:[1-7]|[A-D])(?:\.\d+)*)(?:\s+(.*)|$)')
ANNEX = re.compile(r'^附\s*录\s*([A-D])$')
FORMULAS = {
    'B.1': ('B.5.3.3', 18, 'R_df = N_af / (N_af + N_mf)', r'R_{df}=\frac{N_{af}}{N_{af}+N_{mf}}'),
    'B.2': ('B.5.3.6', 18, 'R_dv = N_av / (N_av + N_mv)', r'R_{dv}=\frac{N_{av}}{N_{av}+N_{mv}}'),
    'C.1': ('C.3', 20, 'R_Ff = N_Ff / N_df', r'R_{Ff}=\frac{N_{Ff}}{N_{df}}'),
    'C.2': ('C.6', 21, 'R_Fv = N_Fv / N_df', r'R_{Fv}=\frac{N_{Fv}}{N_{df}}'),
}
SYMBOLS = {'Rdf':'R_df','Naf':'N_af','Nmf':'N_mf','Rdv':'R_dv','Nav':'N_av','Nmv':'N_mv',
           'RFf':'R_Ff','NFf':'N_Ff','Ndf':'N_df','RFv':'R_Fv','NFv':'N_Fv'}
TERMS = {'vehicledatasecuritymanagementsystem':'vehicle data security management system',
         'cabindata':'cabin data','personalinformationsubject':'personal information subject',
         'faceobject':'face object','faceboundaryframe':'face boundary frame',
         'vehiclelicenseplateobject':'vehicle license plate object',
         'vehiclelicenseplateboundaryframe':'vehicle license plate boundary frame',
         'maskcoveringrate':'mask covering rate','detectionrate':'detection rate',
         'falsedetectionrate':'false detection rate'}
TABLES = {13: ('A.1','数据分级方法','A.3.2'), 14: ('A.2','个人信息分类分级示例','A.4'),
          22: ('D.1','数据收集设备参数','D.1')}
FIGURES = {'1': ('3.6',6,'人脸边界框示意图'), '2': ('3.9',6,'遮盖率示意图'),
           'B.1': ('B.5.3.2',17,'匿名化漏检数结果示例')}
EDITORIAL_NOTES = [
    '公开转载本公式(C.2)的分母及符号说明均印作 N_df，现按原文保留；未自行改为 N_dv，待官方正文核对。',
    '公开转载本 D.4.2 末尾引用 5.4.3、6.3.3，但本件正文没有这两个条号；现照录，不推断替换，待官方正文核对。',
]


def baseline_lines(chars):
    """Use glyph matrices: this PDF misreports punctuation's top by seven points."""
    groups = []
    for char in sorted(chars, key=lambda c: (-c['matrix'][5], c['x0'])):
        y = char['matrix'][5]
        if not groups or abs(groups[-1][0] - y) > 3:
            groups.append((y, [char]))
        else:
            groups[-1][1].append(char)
    return [(y, ''.join(c['text'] for c in sorted(cs, key=lambda c: c['x0'])).strip())
            for y, cs in groups]


def normalize(text):
    for old, new in sorted(TERMS.items(), key=lambda pair: -len(pair[0])):
        text = text.replace(old, new)
    if re.match(r'^[RN][A-Za-z]', text):
        for old, new in SYMBOLS.items():
            text = text.replace(old, new)
    for number, (_, _, plain, _) in FORMULAS.items():
        if '………' in text and text.endswith('(' + number + ')'):
            text = plain + '  (' + number + ')'
    return text


def extract(source):
    import pdfplumber
    pages, tables = {}, {}
    with pdfplumber.open(source) as pdf:
        if len(pdf.pages) != 24:
            raise ValueError('Expected the reviewed 24-page reproduction')
        for physical, page in enumerate(pdf.pages, 1):
            raw = baseline_lines(page.chars)
            if physical < 4:
                continue
            table = None
            if physical in TABLES:
                found = page.find_tables()
                if len(found) != 1:
                    raise ValueError('Expected exactly one table on PDF page ' + str(physical))
                table = found[0]
                grid = []
                for row in table.rows:
                    cells = []
                    for bbox in row.cells:
                        if bbox is None:
                            cells.append(None)
                            continue
                        x0,y0,x1,y1 = bbox
                        chars = [c for c in page.chars if x0 <= c['x0'] < x1 and
                                 y0 <= page.height-c['matrix'][5] < y1]
                        cells.append(''.join(s for _, s in baseline_lines(chars)))
                    grid.append(cells)
                number, title, parent = TABLES[physical]
                if number == 'A.1':
                    columns = [grid[0][0], *grid[1][1:]]
                    rows, notes = grid[2:6], [grid[6][0]]
                elif number == 'A.2':
                    columns, rows = grid[0], grid[1:12]
                    notes = re.split(r'(?=注[123]:)', grid[-1][0])
                    notes = [s for s in notes if s]
                else:
                    columns, rows, notes = grid[0], grid[1:], []
                # Keep merged-cell provenance; repeat inherited context for row lookup.
                merged = []
                for r, cells in enumerate(rows):
                    for c, value in enumerate(cells):
                        if value is None:
                            prior = r - 1
                            while prior >= 0 and rows[prior][c] is None:
                                prior -= 1
                            if prior < 0:
                                raise ValueError('Missing merged-cell origin')
                            cells[c] = rows[prior][c]
                            merged.append({'row':r+1,'column':c+1,'inherited_from_row':prior+1})
                tables[number] = {'number':number,'title':title,'parent':parent,'page':physical,
                                  'columns':columns,'rows':rows,'notes':notes,'merged_cells':merged,
                                  'raw_grid':grid}
            clean, inserted = [], False
            for y, text in raw:
                if not text or text == 'GB/T44464—2024':
                    continue
                if table and table.bbox[1] <= page.height-y < table.bbox[3]:
                    if not inserted:
                        t = tables[TABLES[physical][0]]
                        clean.append((' | '.join(t['columns']),physical))
                        for row in t['rows']:
                            # Prefix row labels so the template's serial number 2 cannot become clause 2.
                            clean.append(('表行: ' + ' | '.join(row),physical))
                        clean.extend((s,physical) for s in t['notes'])
                        inserted = True
                    continue
                clean.append((normalize(text),physical))
            footer = 'Ⅲ' if physical == 4 else str(physical-4)
            if clean[-1][0] != footer:
                raise ValueError('Unexpected printed footer on PDF page ' + str(physical))
            clean.pop()
            pages[physical] = clean
    return pages, tables


def body_text(lines):
    paragraphs, current = [], ''
    heading_previous = False
    for text, _ in lines:
        marker = CLAUSE.match(text)
        new = bool(marker or ANNEX.match(text) or re.match(r'^(?:[a-z]\)|———|●|注|示例|\[来源|图|表|式中|[RN]_|\[\d+\])',text))
        if current and (new or heading_previous or ' | ' in text or ' | ' in current
                        or current.endswith(('。',';','；',':','：',']'))):
            paragraphs.append(current)
            current = ''
        current += (' ' if current and current[-1].isascii() and text[0].isascii() else '') + text
        heading_previous = bool(marker or ANNEX.match(text) or text.startswith(('(资料性)','(规范性)')))
        if re.match(r'^[\u3400-\u9fff]+ [a-z]',text):
            heading_previous = True
    if current:
        paragraphs.append(current)
    return '\n'.join(paragraphs)


def build_rows(pages, tables):
    rows, by_key = [], {}

    def add(locator, number, title, lines, parent=None, kind='clause', **extra):
        key = PREFIX + locator
        if key in by_key:
            raise ValueError('Duplicate key ' + key)
        physical = sorted({p for _,p in lines})
        body = body_text(lines)
        if not body or not physical:
            raise ValueError('Empty clause ' + key)
        row = {'key':key,'document_id':DOCUMENT,'number':number,'title':title,'body':body,'description':body,
               'parent':PREFIX+parent if parent else None,'kind':kind,'language':'zh-CN',
               'normative':False if locator.startswith(('A','C','foreword','bibliography')) else None,
               'normative_note':'推荐性国家标准。附录 A、C 为资料性，B、D 为规范性；引用不等于符合性确认。',
               'source_url':LANDING,'source_file':'sources/gbt-44464-2024.pdf','source_sha256':EXPECTED_SHA256,
               'source_note':SOURCE_NOTE,'page':physical[0],'page_end':physical[-1],
               'print_page':'Ⅲ' if physical[0]==4 else str(physical[0]-4),
               'source_pages':physical,'printed_pages':['Ⅲ' if p==4 else str(p-4) for p in physical],
               'provenance':{'source_kind':'public_reproduction','publisher_page':LANDING,'download_url':DOWNLOAD,
                             'official_metadata_url':OFFICIAL,'source_sha256':EXPECTED_SHA256},**extra}
        rows.append(row)
        by_key[key] = row
        return row

    add('foreword','前言','前言',pages[4],kind='publication_note')
    lines = [line for p in range(5,24) for line in pages[p]]
    if lines[0][0] != '汽车数据通用要求':
        raise ValueError('Unexpected opening title')
    lines = lines[1:]
    headings = []
    for i,(text,_) in enumerate(lines):
        match, annex = CLAUSE.match(text), ANNEX.match(text)
        if annex:
            headings.append((i,annex.group(1),lines[i+2][0]))
        elif match:
            headings.append((i,match.group(1),match.group(2) or ''))
    numbers = {n for _,n,_ in headings}
    if len(numbers) != len(headings):
        raise ValueError('Duplicate numbered heading')
    direct = {}
    for h,(start,number,title) in enumerate(headings):
        own_end = headings[h+1][0] if h+1 < len(headings) else len(lines)
        end = next((i for i,n,_ in headings[h+1:] if not n.startswith(number+'.')),len(lines))
        title = title or lines[start+1][0]
        parent = number.rsplit('.',1)[0] if '.' in number else None
        kind = 'annex' if number in 'ABCD' else ('section' if '.' not in number else 'clause')
        add(number,number,title[:180],lines[start:end],parent,kind,source_number=number)
        direct[number] = lines[start:own_end]
    for number,segment in direct.items():
        level1, level2, parent_item = 0,0,None
        for i,(text,page) in enumerate(segment):
            letter = re.match(r'^([a-z])\)\s*(.*)',text)
            level = 1 if letter or text.startswith('———') else (2 if text.startswith('●') else 0)
            if not level:
                continue
            if level == 1:
                level1 += 1
                level2 = 0
                locator = number+'('+letter.group(1)+')' if letter else number+'.item'+str(level1)
                label = number+'('+letter.group(1)+')' if letter else number+' / 分项 '+str(level1)
                parent, parent_item = number, locator
            else:
                if parent_item is None:
                    raise ValueError('Nested list without parent in ' + number)
                level2 += 1
                locator, parent = parent_item+'.subitem'+str(level2), parent_item
                label = number+' / 分项 '+str(level1)+'.'+str(level2)
            end = len(segment)
            for j in range(i+1,len(segment)):
                s = segment[j][0]
                is_level1 = bool(re.match(r'^[a-z]\)',s) or s.startswith('———'))
                is_level2 = s.startswith('●')
                if is_level1 or (level==2 and is_level2) or s.startswith(('注','示例','图','表')):
                    end=j;break
                if not is_level2 and segment[j-1][0].endswith(('。',';','；')):
                    end=j;break
            add(locator,label,re.sub(r'^(?:[a-z]\)|———|●)\s*','',text)[:180],segment[i:end],parent,'list_item',
                source_number=letter.group(1)+')' if letter else None,
                locator_note='原文有字母标号。' if letter else '原文无编号分项；分项序号是本库定位标识，不是标准原条号。')
    for number,t in tables.items():
        locator = 'Table'+number
        text = '表'+number+' '+t['title']+'\n'+' | '.join(t['columns'])+'\n'
        text += '\n'.join(' | '.join(row) for row in t['rows'])
        text += '\n'+'\n'.join(t['notes'])
        add(locator,'表'+number,t['title'],[(text,t['page'])],t['parent'],'table',
            columns=t['columns'],rows=t['rows'],notes=t['notes'],row_count=len(t['rows']),merged_cells=t['merged_cells'])
        for i,cells in enumerate(t['rows'],1):
            text = '表'+number+' / 第'+str(i)+'行\n'+'\n'.join(column+': '+cell for column,cell in zip(t['columns'],cells))
            if t['notes']:
                text += '\n'+'\n'.join(t['notes'])
            add(locator+'.row'+str(i),'表'+number+' / 第'+str(i)+'行',cells[0] or '空白模板行',
                [(text,t['page'])],locator,'table_row',columns=t['columns'],cells=cells,
                is_example=number=='D.1',source_number=None,
                locator_note='表格行号为本库定位标识；合并单元格内容向下展开，空白保留。')
    for number,(parent,page,plain,latex) in FORMULAS.items():
        own=direct[parent]
        start=next(i for i,(s,_) in enumerate(own) if s.startswith(plain))
        row=add('Formula'+number,'公式('+number+')',plain,own[start:],parent,'formula',formula=plain,latex=latex)
        if number=='C.2':
            row['source_note'] += ' '+EDITORIAL_NOTES[0]
            by_key[PREFIX+parent]['source_note'] += ' '+EDITORIAL_NOTES[0]
    for number,(parent,page,title) in FIGURES.items():
        add('Figure'+number,'图'+number,title,[(f'图{number} {title}\n原图完整保存在源 PDF 物理第 {page} 页；此条提供定位，不以文字重绘图形。',page)],
            parent,'figure',figure_in_source_pdf=True)
    by_key[PREFIX+'D.4.2']['source_note'] += ' '+EDITORIAL_NOTES[1]
    add('bibliography','参考文献','参考文献',pages[24],kind='bibliography')
    for text,page in pages[24]:
        m=re.match(r'^\[(\d+)\]',text)
        if m:
            add('bibliography.'+m.group(1),'参考文献['+m.group(1)+']',text,[(text,page)],'bibliography','bibliography_entry')
    for row in rows:
        if row['parent'] is not None and row['parent'] not in by_key:
            raise ValueError('Missing parent '+row['parent'])
        # Derived table/formula locators do not begin with the annex letter.
        # Preserve informative status through the complete parent chain.
        seen, parent = {row['key']}, row['parent']
        while parent:
            if parent in seen:
                raise ValueError('Cyclic parent chain '+parent)
            seen.add(parent)
            ancestor = by_key[parent]
            if ancestor['normative'] is False:
                row['normative'] = False
            parent = ancestor['parent']
    return rows


def manifest_record(rows, bundle_sha):
    counts = Counter(row['kind'] for row in rows)
    return {'id':DOCUMENT,'code':'GB/T 44464-2024','title':'汽车数据通用要求','edition':'2024',
            'status':'官方目录现行；公开转载正文待与官方副本逐字核对','coverage':'partial',
            'clause_count':len(rows),'language':'zh-CN','verified_at':'2026-10-25',
            'publication_date':'2024-08-23','effective_date':'2024-08-23',
            'publisher':'国家市场监督管理总局、国家标准化管理委员会','document_type':'recommended_standard',
            'source_url':LANDING,'metadata_source_url':OFFICIAL,'download_url':DOWNLOAD,'source_note':SOURCE_NOTE,
            'coverage_note':'已逐项录入所取得的 24 页公开转载本：第 1–7 章、附录 A–D、三张表、四条公式及注示例；三幅图保存在源 PDF。正文与官方副本逐字一致性待核对，因此标记为部分核验，不声称官方全文核验完成。',
            'required_source':'取得官方正式正文后，逐字核对公开转载本及已标注的原文引用、公式异常。',
            'editorial_notes':EDITORIAL_NOTES,'clauses_file':DOCUMENT+'.json','clauses_sha256':bundle_sha,
            'coverage_evidence':{'source_pdf_pages':24,'printed_body_pages':20,'visually_reviewed_pages':list(range(1,25)),
                                 'sections':7,'annexes':['A','B','C','D'],'tables':['A.1','A.2','D.1'],
                                 'formulas':['B.1','B.2','C.1','C.2'],'figures_in_pdf':['1','2','B.1'],
                                 'counts_by_kind':dict(counts),'official_text_comparison':'pending',
                                 'extraction':'Glyph-baseline grouping; table cells reconstructed from ruling geometry; formulas and all pages visually checked against this reproduction.'},
            'artifacts':[{'file':'sources/'+DOCUMENT+'.pdf','sha256':EXPECTED_SHA256,'media_type':'application/pdf',
                          'source_url':DOWNLOAD,'source_kind':'public_reproduction','source_note':SOURCE_NOTE}]}


def main(source=None):
    source = Path(source) if source else HERE/'sources'/ (DOCUMENT+'.pdf')
    if not source.exists():
        source = WORK/'chinese'/'gbt44464-public-copy.pdf'
    if hashlib.sha256(source.read_bytes()).hexdigest() != EXPECTED_SHA256:
        raise ValueError('Input differs from the reviewed public reproduction')
    pages,tables = extract(source)
    rows=build_rows(pages,tables)
    output=HERE/(DOCUMENT+'.json')
    output.write_text(json.dumps(rows,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    destination=HERE/'sources'/(DOCUMENT+'.pdf')
    destination.parent.mkdir(parents=True,exist_ok=True)
    if source.resolve()!=destination.resolve():
        shutil.copyfile(source,destination)
    WORK.mkdir(parents=True,exist_ok=True)
    record=manifest_record(rows,hashlib.sha256(output.read_bytes()).hexdigest())
    (WORK/'gbt44464-manifest-patch.json').write_text(json.dumps(record,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    print(json.dumps({'rows':len(rows),'counts_by_kind':record['coverage_evidence']['counts_by_kind'],
                      'coverage':record['coverage'],'manifest_patch':str(WORK/'gbt44464-manifest-patch.json')},ensure_ascii=False))


if __name__=='__main__':
    main(sys.argv[1] if len(sys.argv)>1 else None)
