import test from 'node:test';
import assert from 'node:assert/strict';
import {nodeMarkup,nodeAppearance,labelMarkup,isDiagramEndpoint,normalizedArchitecture,staticDiagramSvg} from '../client/src/architecture-view.js';
import {flowDashArray} from '../client/src/communication.js';
import {numberedGraph,buildCodeIndex} from '../client/src/numbering.js';
import {routeConnections} from '../client/src/connection-routing.js';
import {Diagram} from '../client/src/diagram.js';
const fixture=()=>({view_mode:'dfd',show_numbers:false,nodes:[{id:'e',name:'External',kind:'entity',entityType:'external',x:0,y:0,width:220,height:120},{id:'p',name:'Process',kind:'process_step',x:480,y:0,width:220,height:120},{id:'d',name:'Stored data',kind:'data_store',x:960,y:0,width:220,height:120}],flows:[{id:'input',name:'Input',source:'e',target:'p',emphasis:'critical'},{id:'write',name:'Internal write',source:'p',target:'d',emphasis:'internal',show_label:false},{id:'power',name:'Power',source:'e',target:'p',channel_kind:'energy'}]});
test('expanded DFD preserves node identity and exposes process/store ports only in its own mode',()=>{
 const g=fixture();assert.deepEqual(normalizedArchitecture(g).nodes.map(n=>n.kind),['entity','process_step','data_store']);
 for(const n of g.nodes){assert.ok(isDiagramEndpoint(n,g));assert.equal((nodeMarkup(n,{dfd:true}).match(/data-port=/g)||[]).length,4)}
 assert.equal(isDiagramEndpoint(g.nodes[2],{view_mode:'entities'}),false);
 assert.equal(isDiagramEndpoint({kind:'boundary'},g),false);
 assert.equal(numberedGraph(g,buildCodeIndex({architecture:g})).nodes[1].name,'Process');
});
test('expanded DFD exports all connectors with consistent colors, hidden internal labels and no animation',()=>{
 const g=fixture(),r=routeConnections(g);assert.equal(r.conflicts.length,0);assert.equal(r.routes.length,3);
 const svg=staticDiagramSvg(g,r.routes);
 assert.match(svg,/stroke="#b42318"/);assert.match(svg,/stroke="#ad6800"/);assert.match(svg,/fill="#f2f7ed"/);
 assert.doesNotMatch(svg,/data-port=|<animate|flow-pulse/);
 assert.equal((svg.match(/class="flow-label-handle/g)||[]).length,2);
});
test('expanded DFD uses the platform palette, dashed boundaries and communication conventions',()=>{
 const boundary={id:'b',name:'Trust domain',kind:'boundary',x:0,y:0,width:1000,height:600};
 for(const n of [...fixture().nodes,boundary]){
  const expected=nodeAppearance(n),markup=nodeMarkup(n,{dfd:true,interactive:false});
  assert.ok(markup.includes(`fill="${expected.fill}"`));assert.ok(markup.includes(`stroke="${expected.stroke}"`));
  if(n.kind==='boundary')assert.match(markup,/stroke-dasharray="7 5"/);
 }
 for(const [medium,dash] of [['wireless','7 5'],['logical','2 4'],['wired','']]){
  const flow={id:'f',name:'Link',emphasis:'internal',communication:{medium,technology:medium==='wireless'?'Wi-Fi':''}};
  assert.equal(flowDashArray(flow),dash);
  if(medium==='wireless')assert.match(labelMarkup(flow,{points:[{x:0,y:0},{x:200,y:0}]},{dfd:true}),/communication-badge/);
 }
});
test('switching to entity mode cannot silently coerce processing or storage nodes',async()=>{
 const d=Object.create(Diagram.prototype);Object.assign(d,{alive:true,graph:fixture(),message(text){this.notice=text},commit(){assert.fail('must not mutate incompatible diagram')}});
 const before=structuredClone(d.graph);await d.onClick({target:{closest:()=>({dataset:{dfd:'view-mode'}})}});
 assert.deepEqual(d.graph,before);assert.match(d.notice,/保留展开 DFD/);
});
