import test from 'node:test';
import assert from 'node:assert/strict';
import {Diagram} from '../client/src/diagram.js';
import {staticDiagramSvg,escapeXml} from '../client/src/architecture-view.js';
import {routeConnections,editableSegments,flowWaypoint} from '../client/src/connection-routing.js';
const sample=(medium='wired')=>({equal_spacing:true,nodes:[{id:'a',name:'A',kind:'entity',x:0,y:0,width:200,height:100},{id:'b',name:'B',kind:'entity',x:500,y:0,width:200,height:100}],flows:[{id:'flow',name:'状态消息',source:'a',target:'b',sourceSide:'right',targetSide:'left',communication:{medium,technology:medium==='wireless'?'Wi-Fi':'CAN',security_properties:['integrity']}}]});
function harness(medium){
 const d=Object.create(Diagram.prototype),element={setAttribute(){},classList:{toggle(){}}},captures=new Set();
 Object.assign(d,{graph:sample(medium),saved:JSON.stringify(sample(medium)),esc:escapeXml,alive:true,view:{x:-60,y:-60,w:900,h:600},routes:[],undo:[],redo:[],flowMotion:true,selected:null,root:{querySelector:()=>element},svg:{setAttribute(){},focus(){},setPointerCapture:id=>captures.add(id),hasPointerCapture:id=>captures.has(id),releasePointerCapture:id=>captures.delete(id)},inspect(){},point:e=>({x:e.clientX,y:e.clientY}),message(text){this.lastMessage=text},draft(){this.drafted=structuredClone(this.graph)}});
 d.layout();d.render();
 const target=(kind,role)=>({closest:selector=>selector==='[data-flow]'?{dataset:{flow:'flow'}}:selector===`[data-flow-${kind}]`?{dataset:{flowEnd:role,flowSegment:String(role)}}:null});
 const pointer=(x,y,kind='end',role='target',id=1)=>({button:0,pointerId:id,clientX:x,clientY:y,target:target(kind,role),preventDefault(){}});
 return {d,down:(x,y,kind,role)=>d.pointerDown(pointer(x,y,kind,role)),move:(x,y,id=1)=>d.pointerMove(pointer(x,y,'end','target',id)),up:()=>d.pointerUp({pointerId:1}),action:name=>d.onClick({target:{closest:()=>({dataset:{dfd:name}})}})};
}
const semantics=({sourceSide,targetSide,route_layout,wireless_layout,labelOffset,...f})=>f;
for(const medium of ['wired','wireless','logical']) test(`${medium}: drag arrow to another side on the same entity, undo and redo`,()=>{
 const h=harness(medium),before=structuredClone(h.d.graph);
 assert.doesNotMatch(h.d.svg.innerHTML,/data-flow-end=/,'controls only appear on selection');
 h.down(500,50);h.move(600,100);h.up();
 assert.equal(h.d.graph.flows[0].targetSide,'bottom');assert.equal(h.d.routes[0].target.side,'bottom');
 assert.deepEqual(semantics(h.d.graph.flows[0]),semantics(before.flows[0]));assert.deepEqual(h.d.graph.nodes,before.nodes);
 assert.equal(h.d.undo.length,1);assert.equal((h.d.svg.innerHTML.match(/data-flow-end=/g)||[]).length,2);
 assert.doesNotMatch(h.d.svg.innerHTML,/wireless-endpoint|data-wireless-/);
 h.d.history('undo');assert.deepEqual(h.d.graph,before);h.d.history('redo');assert.equal(h.d.graph.flows[0].targetSide,'bottom');
});
test('endpoint cannot attach to a different entity or empty canvas; cancelled drag restores exact data',()=>{
 const h=harness('wireless'),before=structuredClone(h.d.graph);
 h.down(500,50);h.move(600,100,2);assert.deepEqual(h.d.graph,before);
 h.move(100,0);h.up();assert.deepEqual(h.d.graph,before);assert.equal(h.d.undo.length,0);
 h.down(500,50);h.move(600,100);h.d.cancelInteraction();h.up();assert.deepEqual(h.d.graph,before);
 h.down(500,50);h.move(600,100);h.move(950,300);h.up();assert.deepEqual(h.d.graph,before);
 h.down(500,50);h.move(600,100);h.move(500,50);h.up();assert.deepEqual(h.d.graph,before);assert.equal(h.d.undo.length,0);
});
for(const medium of ['wired','wireless','logical']) test(`${medium}: segment drag snaps, preserves endpoints, tidies bends and exports same static path`,async()=>{
 const h=harness(medium),before=structuredClone(h.d.graph),part=editableSegments(h.d.routes[0].points)[0];
 h.down(part.point.x,part.point.y,'segment',part.index);h.move(part.point.x,part.point.y+87);h.up();
 assert.deepEqual(flowWaypoint(h.d.graph.flows[0]),{x:280,y:140});assert.equal(h.d.undo.length,1);
 assert.deepEqual(semantics(h.d.graph.flows[0]),semantics(before.flows[0]));assert.deepEqual(h.d.graph.nodes,before.nodes);
 const points=h.d.routes[0].points;
 for(let i=1;i<points.length;i++){let a=points[i-1],b=points[i];assert.ok((a.x===b.x)!==(a.y===b.y));if(i>1){let c=points[i-2];assert.ok(!(c.x===a.x&&a.x===b.x)&&!(c.y===a.y&&a.y===b.y),'no redundant collinear bend');}}
 const svg=staticDiagramSvg(h.d.graph,h.d.routes),path=points.map((p,i)=>`${i?'L':'M'} ${p.x} ${p.y}`).join(' ');
 assert.ok(svg.includes(`d="${path}"`));assert.doesNotMatch(svg,/connection-end|connection-segment|wireless-endpoint|data-flow|flow-pulse|<animate/);
 await h.action('tidy-route');assert.equal(flowWaypoint(h.d.graph.flows[0]),undefined);assert.equal(h.d.routes[0].points.length,2);assert.equal(h.d.graph.flows[0].targetSide,'left');
 h.d.history('undo');assert.deepEqual(flowWaypoint(h.d.graph.flows[0]),{x:280,y:140});
});
test('side selectors, restore automatic, legacy layout precedence and cancelled segment edits',async()=>{
 const h=harness();h.d.selected='flow';h.d.inspector={querySelector:s=>({value:s.includes('source')?'bottom':'top'})};
 await h.action('apply-flow-sides');assert.equal(h.d.routes[0].source.side,'bottom');assert.equal(h.d.routes[0].target.side,'top');
 await h.action('reset-route');assert.equal(h.d.graph.flows[0].sourceSide,null);assert.equal(h.d.graph.flows[0].targetSide,null);
 h.d.graph.flows[0].wireless_layout={waypoint:{x:320,y:300},source_offset:{x:30,y:140}};
 assert.equal(flowWaypoint(h.d.graph.flows[0]),undefined,'explicit empty new layout overrides legacy');
 const before=structuredClone(h.d.graph),part=editableSegments(h.d.routes[0].points)[0];
 h.down(part.point.x,part.point.y,'segment',part.index);h.move(part.point.x,part.point.y+80);h.d.cancelInteraction();h.up();assert.deepEqual(h.d.graph,before);
});
