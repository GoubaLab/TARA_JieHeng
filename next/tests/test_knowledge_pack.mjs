import test from 'node:test';
import assert from 'node:assert/strict';
import {validateKnowledgePackPreview} from '../client/src/knowledge-pack.js';
const pack=()=>({schema:'jieheng.knowledge-pack/1',pack_id:'synthetic',version:'1',entries:[{id:'case-1',name:'Synthetic case',edition:'1',source:'https://example.org/advisory',content:'Synthetic facts',evidence_urls:['https://example.org/advisory'],cve_ids:['CVE-2020-1234']}]});
test('knowledge preview rejects malformed renderable fields before React sees imported JSON',()=>{
 for(const field of ['name','id','content','edition','source','cve_ids']){
  const p=pack();p.entries[0][field]=field==='cve_ids'?'CVE-2020-1234':{unexpected:'object'};assert.throws(()=>validateKnowledgePackPreview(p));
 }
 for(const row of [null,[],42]){const p=pack();p.entries=[row];assert.throws(()=>validateKnowledgePackPreview(p));}
 const p=pack();p.version={};assert.throws(()=>validateKnowledgePackPreview(p));
});
test('knowledge preview supports ordinary cases and notice-only imports without mutating input',()=>{
 const p=pack(),before=JSON.stringify(p);assert.equal(validateKnowledgePackPreview(p),p);assert.equal(JSON.stringify(p),before);
 const notice={...p,entries:[],review_required:[{cve_id:'CVE-2020-1234',status:'missing'}]};assert.equal(validateKnowledgePackPreview(notice),notice);
 assert.throws(()=>validateKnowledgePackPreview({...p,entries:[]}));
});
