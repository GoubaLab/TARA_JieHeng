import test from 'node:test';
import assert from 'node:assert/strict';
import {parseAttackSteps,formatAttackSteps} from '../client/src/attack-steps.js';
test('numbered pasted actions retain multiline prerequisites and are renumbered',()=>{
 const actions=parseAttackSteps('1、访问接口\n前提：维护模式\n2) 转发报文\n3. 控制输出');
 assert.deepEqual(actions,['访问接口\n前提：维护模式','转发报文','控制输出']);
 assert.equal(formatAttackSteps([actions[2],actions[0]]),'1、控制输出\n2、访问接口\n前提：维护模式');
});
test('unstructured legacy descriptions and empty new steps remain editable',()=>{
 assert.deepEqual(parseAttackSteps('入口说明\n必要前提'),['入口说明\n必要前提']);
 assert.deepEqual(parseAttackSteps(formatAttackSteps(['访问接口',''])),['访问接口','']);
});
