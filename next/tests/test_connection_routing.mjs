import test from 'node:test';
import assert from 'node:assert/strict';
import {routeConnections} from '../client/src/connection-routing.js';
import {staticDiagramSvg} from '../client/src/architecture-view.js';
const node=(id,x,y)=>({id,name:id,kind:'entity',entityType:'device',x,y,width:210,height:118});
const sample=()=>({nodes:[node('a',70,140),node('b',390,70),node('c',710,180)],flows:[
{id:'wired',source:'a',target:'b',sourceSide:'right',targetSide:'left',name:'控制指令',protocol:'CAN FD'},
{id:'wifi',source:'a',target:'c',name:'状态遥测',protocol:'Wi-Fi'},
{id:'nfc',source:'b',target:'a',name:'配置授权',protocol:'NFC'}],equal_spacing:true});
const segments=r=>r.points.slice(1).map((b,i)=>{const a=r.points[i],h=a.y===b.y;assert.ok(h||a.x===b.x);return {id:r.id,h,pos:h?a.y:a.x,lo:Math.min(h?a.x:a.y,h?b.x:b.y),hi:Math.max(h?a.x:a.y,h?b.x:b.y)}});
const intersect=(a,b)=>a.h===b.h?a.pos===b.pos&&Math.max(a.lo,b.lo)<=Math.min(a.hi,b.hi):a.pos>=b.lo&&a.pos<=b.hi&&b.pos>=a.lo&&b.pos<=a.hi;
function verify(g,result){
 assert.equal(result.conflicts.length,0,JSON.stringify(result.conflicts));assert.equal(result.routes.length,g.flows.length);
 const lines=result.routes.flatMap(segments);
 for(const a of lines){for(const b of lines)if(a.id!==b.id)assert.equal(intersect(a,b),false,`${a.id} crosses ${b.id}`);
 for(const box of g.nodes){const enters=a.h?a.pos>box.y&&a.pos<box.y+box.height&&a.hi>box.x&&a.lo<box.x+box.width:a.pos>box.x&&a.pos<box.x+box.width&&a.hi>box.y&&a.lo<box.y+box.height;assert.equal(enters,false,`${a.id} enters ${box.id}`);}}
}
test('all media share entity ports; NFC takes a short path, no separate entities or crossing, stable under reordering',()=>{
 const g=sample(),before=structuredClone(g),result=routeConnections(g);verify(g,result);
 const nfc=result.routes.find(r=>r.id==='nfc');assert.ok(segments(nfc).reduce((n,s)=>n+s.hi-s.lo,0)<250);
 assert.deepEqual(routeConnections({...g,nodes:[...g.nodes].reverse(),flows:[...g.flows].reverse()}).routes,result.routes);assert.deepEqual(g,before);
 const svg=staticDiagramSvg(g,result.routes);assert.equal((svg.match(/class="dfd-node /g)||[]).length,3);assert.equal((svg.match(/communication-badge/g)||[]).length,2);assert.doesNotMatch(svg,/wireless-endpoint|W01|flow-pulse/);
});
test('moved entities re-route and export bounds include every flow, label and node',()=>{
 const g=sample();g.nodes[2].x+=120;g.nodes[2].y+=80;const before=structuredClone(g),result=routeConnections(g);verify(g,result);assert.deepEqual(g,before);
 const svg=staticDiagramSvg(g,result.routes),[x,y,w,h]=svg.match(/viewBox="([^"]+)"/)[1].split(' ').map(Number);
 for(const r of result.routes)for(const p of r.points)assert.ok(p.x>=x&&p.x<=x+w&&p.y>=y&&p.y<=y+h);
});
test('blocked routes report conflict instead of exporting missing flows',()=>{
 const g=sample();g.nodes[1].x=70;g.nodes[1].y=140;assert.ok(routeConnections(g).conflicts.length);assert.throws(()=>staticDiagramSvg(g,[]),/布线冲突/);
});
test('manual waypoint routing never loses its guide point or folds back onto its own path',()=>{
 const g={nodes:[node('a',0,0),node('b',500,0)],flows:[{id:'f',source:'a',target:'b',sourceSide:'right',targetSide:'left'}]};
 for(const p of [{x:350,y:210},{x:-100,y:-100},{x:800,y:200},{x:350,y:60}]){
 g.flows[0].route_layout={waypoint:p};const result=routeConnections(g);assert.equal(result.conflicts.length,0);
 const r=result.routes[0],parts=segments(r);assert.ok(parts.some(s=>(s.h?p.y:p.x)===s.pos&&(s.h?p.x:p.y)>=s.lo&&(s.h?p.x:p.y)<=s.hi));
 for(let i=0;i<parts.length;i++)for(let j=i+2;j<parts.length;j++)assert.equal(intersect(parts[i],parts[j]),false,'no self-intersection or backtracking');
 }
});
