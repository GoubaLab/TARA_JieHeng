import test from 'node:test';
import assert from 'node:assert/strict';
import { buildCodeIndex, effectiveNumbering, formatNumber, pathNumberKey, numberedGraph } from '../client/src/numbering.js';
import { staticDiagramSvg } from '../client/src/architecture-view.js';

test('server assignments survive reorder, retired gaps and custom format without changing IDs', () => {
 const ws = { numbering: { prefixes: { asset: '资产' }, separator: '/', padding: 4 }, numbering_state: { asset: { removed: 1, retained: 2 } }, assets: [{ id: 'new' }, { id: 'retained' }] };
 const before = structuredClone(ws), code = buildCodeIndex(ws);
 assert.equal(code('asset','retained'), '资产/0002'); assert.equal(code('asset','new'), '资产/0003'); assert.deepEqual(ws,before);
 assert.equal(formatNumber(effectiveNumbering(), 'asset', 12345), 'AS-12345');
});
test('legacy numbering and scoped Unicode paths match server contract, even special object keys', () => {
 const ws = { scenarios: [{ id:'场景', paths:[{id:'same'}] }, {id:'other', paths:[{id:'same'}]}], assets: [{id:'__proto__'}, {id:'constructor'}] };
 const code = buildCodeIndex(ws);
 assert.equal(pathNumberKey('场景','same'), '6:场景same'); assert.equal(code('path',pathNumberKey('场景','same')), 'AP-001'); assert.equal(code('path',pathNumberKey('other','same')), 'AP-002');
 assert.equal(code('asset','__proto__'), 'AS-001'); assert.equal(code('asset','constructor'), 'AS-002'); assert.equal(ws.numbering_state, undefined);
});
test('numbered diagram and static export share display codes without rewriting business labels', () => {
 const graph = {nodes:[{id:'one',name:'控制器',kind:'entity',entityType:'device',x:0,y:0,width:300,height:180,children:[{id:'app',name:'控制软件',kind:'software'}]}, {id:'two',name:'网关',kind:'entity',entityType:'device',x:500,y:0,width:300,height:180}],flows:[{id:'flow',name:'状态',source:'one',target:'two',communication:{medium:'wireless',technology:'Wi-Fi'}}]};
 const code = buildCodeIndex({architecture:graph}), view = numberedGraph(graph,code);
 assert.equal(graph.nodes[0].name,'控制器'); assert.match(view.nodes[0].children[0].name,/SW-001/);
 const svg = staticDiagramSvg(view,[],r=>'M'+r.points.map(p=>`${p.x},${p.y}`).join('L'));
 for (const label of ['EN-001','SW-001','DF-001','Wi-Fi']) assert.ok(svg.includes(label),label);
 assert.doesNotMatch(svg, /<animate|data-port=|data-flow-label=/);
});

test('unsaved legacy edits reserve old graph numbers before assigning new objects', () => {
 const ws = {architecture:{nodes:[{id:'old',kind:'entity'}, {id:'kept',kind:'entity'}], flows:[]}};
 const draft = {nodes:[{id:'new',kind:'entity'}, {id:'kept',kind:'entity'}], flows:[]};
 const code=buildCodeIndex(ws,draft);
 assert.equal(code('entity','kept'),'EN-002'); assert.equal(code('entity','new'),'EN-003');
});
