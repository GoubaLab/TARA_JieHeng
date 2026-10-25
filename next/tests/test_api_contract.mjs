import test from 'node:test';
import assert from 'node:assert/strict';
import {command, request, allWorkspaces, workspaceEvents} from '../client/src/api.js';

const originalFetch=globalThis.fetch;
function capture(reply={workspace_id:'ws_1',revision:5}) {
  const calls=[];
  globalThis.fetch=async (url,options={})=>{
    calls.push({url,...options});
    return new Response(JSON.stringify(typeof reply==='function'?reply(url):reply),{status:200,headers:{'Content-Type':'application/json'}});
  };
  return calls;
}
test.afterEach(()=>{globalThis.fetch=originalFetch;});
test('resource mutations carry version and idempotency headers, without a command envelope',async()=>{
  const calls=capture();
  const cases=[
    ['define-item',{},'PUT','item'],['replace-architecture',{},'PUT','architecture'],
    ['record-asset',{name:'资产'},'POST','assets'],['record-asset',{id:'asset_1',name:'资产'},'PUT','assets/asset_1'],
    ['record-scenario',{title:'场景'},'POST','scenarios'],['record-scenario',{id:'scenario_1'},'PUT','scenarios/scenario_1'],
    ['decide-treatment',{scenario_id:'scenario_1',treatment:'reduce'},'PUT','scenarios/scenario_1/treatment'],
    ['specify-requirement',{},'POST','requirements'],['specify-requirement',{id:'req_1'},'PUT','requirements/req_1'],
    ['record-evidence',{id:null},'POST','evidence'],['adopt-method',{},'PUT','assessment-method'],
    ['seal-baseline',{},'POST','baselines'],['apply-reference-selection',{},'POST','reference-imports'],
    ['select-regulations',{references:['reg:gb-44496-2024:4.1.1']},'PUT','regulation-references'],
  ];
  for (const [action,payload,method,path] of cases) {
    const snapshot=structuredClone(payload);
    await command({id:'ws_1',revision:4},action,payload,'retry-id-123');
    const call=calls.at(-1);
    assert.equal(call.url,'/api/v1/workspaces/ws_1/'+path);
    assert.equal(call.method,method);
    assert.equal(call.headers['If-Match'],'"4"');
    assert.equal(call.headers['Idempotency-Key'],'retry-id-123');
    assert.equal(call.headers['X-Workspace-Client'],'jieheng');
    const data=JSON.parse(call.body);
    assert.equal(data.request_id,undefined);assert.equal(data.expected_revision,undefined);
    assert.equal(data.id,undefined);
    if(action==='decide-treatment')assert.equal(data.scenario_id,undefined);
    assert.deepEqual(payload,snapshot);
  }
});
test('unknown operation is rejected before any request',()=>{
  const calls=capture();
  assert.throws(()=>command({id:'x',revision:0},'unknown',{}));
  assert.equal(calls.length,0);
});
test('problem response keeps HTTP status and request id for diagnosis',async()=>{
  globalThis.fetch=async()=>new Response(JSON.stringify({title:'保存版本冲突',detail:'请刷新后重试',request_id:'req-trace'}),{status:412});
  await assert.rejects(()=>request('/workspaces/ws_1'),e=>e.status===412&&e.requestId==='req-trace'&&e.message==='请刷新后重试');
});
test('project picker and history do not lose records beyond a page',async()=>{
  const calls=capture(url=>{
    const offset=Number(new URL(url,'http://localhost').searchParams.get('offset'));
    const items=Array.from({length:offset===0?100:3},(_,i)=>({id:offset+i}));
    return {items,entries:items,total:103,offset,limit:100};
  });
  assert.equal((await allWorkspaces()).items.length,103);
  assert.equal((await workspaceEvents('ws_1')).entries.length,103);
  assert.equal(calls.length,4);
  assert.ok(calls.every(c=>c.url.startsWith('/api/v1/')));
});
