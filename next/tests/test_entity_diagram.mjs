import test from 'node:test';
import assert from 'node:assert/strict';
import {isEntity,normalizedArchitecture,assignLegacyStore,nodeMarkup,staticDiagramSvg} from '../client/src/architecture-view.js';
import { Diagram } from '../client/src/diagram.js';
const node=(id,kind,x=0)=>({id,kind,name:id,x,y:0,width:240,height:130,description:'',reference:null});
const sample=()=>({nodes:[node('ecu','process'),node('outside','external',400),node('memory','store',800)],flows:[{id:'read',name:'读取数据',source:'outside',target:'memory',sourceSide:'left',targetSide:'right',protocol:'CAN',description:'read'}],equal_spacing:true});

test('old process and external records normalize without mutating or inventing store ownership',()=>{
 const graph=sample(),before=structuredClone(graph),result=normalizedArchitecture(graph);
 assert.deepEqual(graph,before);assert.equal(result.nodes[0].kind,'entity');assert.equal(result.nodes[0].entityType,'ecu');assert.equal(result.nodes[1].entityType,'external');assert.equal(result.nodes[2].kind,'store');
 assert.equal(isEntity(result.nodes[2]),false);assert.equal(isEntity(result.nodes[0]),true);
});
test('explicit storage assignment preserves identity and references while redirecting external flows',()=>{
 const graph=sample();graph.nodes[2].description='密钥';graph.nodes[2].reference='systems:9';const before=structuredClone(graph);
 const result=assignLegacyStore(graph,'memory','ecu');assert.deepEqual(graph,before);
 const parent=result.nodes.find(n=>n.id==='ecu');assert.deepEqual(parent.children,[{id:'memory',name:'memory',kind:'store',description:'密钥',reference:'systems:9'}]);
 assert.equal(result.nodes.length,2);assert.equal(result.flows[0].id,'read');assert.equal(result.flows[0].target,'ecu');assert.equal(result.flows[0].targetSide,null);assert.equal(result.flows[0].source,'outside');
});
test('assignment refuses internal self-loop rather than deleting a data flow',()=>{
 const graph=sample();graph.flows[0].source='ecu';const before=structuredClone(graph);
 assert.throws(()=>assignLegacyStore(graph,'memory','ecu'),/同一实体内/);assert.deepEqual(graph,before);
 assert.throws(()=>assignLegacyStore(graph,'memory','memory'),/目标实体/);
});
test('only entities expose connection ports, and children remain inside the entity card',()=>{
 const entity={...node('ecu','entity'),entityType:'ecu',children:[{id:'sw',kind:'software',name:'诊断应用',description:''}],attributes:[]};
 const markup=nodeMarkup(entity);assert.match(markup,/诊断应用/);assert.equal((markup.match(/data-port=/g)||[]).length,4);
 assert.doesNotMatch(nodeMarkup(node('memory','store')),/data-port=/);assert.doesNotMatch(nodeMarkup(node('boundary','boundary')),/data-port=/);
});
test('static export has directional arrows, children, white background and offset labels but no motion or interactive payload',()=>{
 const graph=normalizedArchitecture(sample());graph.nodes=graph.nodes.slice(0,2);graph.nodes[0].children=[{id:'sw',kind:'software',name:'<软件&模块>',description:'说明'}];graph.flows[0].target='ecu';graph.flows[0].labelOffset={x:25,y:-30};
 const routes=[{id:'read',points:[{x:400,y:65},{x:240,y:65}]}];const svg=staticDiagramSvg(graph,routes);
 assert.match(svg,/xmlns="http:\/\/www.w3.org\/2000\/svg"/);assert.match(svg,/marker-end="url\(#static-arrow\)"/);assert.match(svg,/M 400 65 L 240 65/);assert.match(svg,/translate\(345 35\)/);
 assert.match(svg,/&lt;软件&amp;模块&gt;/);assert.match(svg,/fill="#ffffff"/);
 assert.doesNotMatch(svg,/<animate|<script|@keyframes|animation:|flow-pulse|data-port=|data-dfd=|data-node=|onload=/i);
});
test('child and attribute buttons update the selected entity rather than creating top-level nodes',async()=>{
 const d=Object.create(Diagram.prototype);d.alive=true;d.selected='ecu';d.graph={nodes:[{...node('ecu','entity'),entityType:'ecu',children:[],attributes:[]}],flows:[],equal_spacing:true};d.undo=[];d.redo=[];
 d.readNodeEditor=n=>structuredClone(n);d.layout=()=>true;d.draft=()=>{};d.render=()=>{};d.inspect=()=>{};d.message=message=>assert.fail(message);
 const action=name=>d.onClick({target:{closest:()=>({dataset:{dfd:name}})}});
 await action('add-software');await action('add-store');await action('add-attribute');await action('add-attribute');
 assert.equal(d.graph.nodes.length,1);assert.deepEqual(d.graph.nodes[0].children.map(c=>c.kind),['software','store']);assert.deepEqual(d.graph.nodes[0].attributes.map(a=>a.name),['新属性','新属性 2']);assert.equal(d.undo.length,4);
});
test('motion toggle is a view preference and does not touch graph or undo history',async()=>{
 const d=Object.create(Diagram.prototype);d.alive=true;d.flowMotion=true;d.graph=sample();d.undo=[];d.render=()=>{};const before=structuredClone(d.graph);
 await d.onClick({target:{closest:()=>({dataset:{dfd:'motion'}})}});assert.equal(d.flowMotion,false);assert.deepEqual(d.graph,before);assert.deepEqual(d.undo,[]);
});


test('static exports include every internal object and attribute even when the card is too small',()=>{
 const entity={...node('ecu','entity'),entityType:'ecu',height:80,children:Array.from({length:12},(_,i)=>({id:'child'+i,kind:'software',name:'Internal object '+i,description:'Complete description '+i})),attributes:[{name:'Hardware',value:'Revision B'}]};
 const svg=staticDiagramSvg({nodes:[entity],flows:[]},[]);
 for(let i=0;i<12;i++){assert.ok(svg.includes('Internal object '+i));assert.ok(svg.includes('Complete description '+i));}
 assert.match(svg,/Hardware：Revision B/);assert.match(svg,/实体属性与内部组成明细/);assert.doesNotMatch(svg,/选中实体查看/);
 assert.ok(Number(svg.match(/<svg[^>]+height="([0-9]+)"/)[1])>600);
});
