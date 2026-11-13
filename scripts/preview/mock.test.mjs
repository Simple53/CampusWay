import assert from 'node:assert/strict';
import {createPreviewBridge} from './mock.mjs';
const wait=()=>new Promise(resolve=>setTimeout(resolve,25));
const sim=createPreviewBridge({delay:5}),api=sim.api;
assert.equal((await api.GetSnapshot()).preferences.language,'zh-CN');
assert.equal((await api.GetSnapshot()).profiles[0].username,'');
await api.Connect('demo-atrust','secret-sentinel');await wait();assert.equal((await api.GetSnapshot()).state,'connected');
assert.ok(!JSON.stringify(await api.GetSnapshot()).includes('secret-sentinel'));
sim.reset('verification');await api.Connect('demo-atrust','');await wait();let s=await api.GetSnapshot();assert.equal(s.state,'awaiting_auth');await api.SubmitChallenge(s.challenge.id,'123456');await wait();assert.equal((await api.GetSnapshot()).state,'connected');
sim.reset('failure');await api.Connect('demo-atrust','');await wait();assert.equal((await api.GetSnapshot()).state,'error');
sim.reset('routes');await api.Connect('demo-atrust','');await api.Disconnect();await wait();assert.equal((await api.GetSnapshot()).state,'disconnected');
await api.Connect('demo-atrust','');sim.reset('empty');await wait();assert.equal((await api.GetSnapshot()).profiles.length,0);assert.equal((await api.GetSnapshot()).state,'disconnected');
const before=JSON.stringify(await api.GetSnapshot());await assert.rejects(api.ImportData('{"version":1,"profiles":[{"name":"broken"}]}'));assert.equal(JSON.stringify(await api.GetSnapshot()),before);
sim.reset('routes');const p=(await api.GetSnapshot()).profiles[0];p.username='private-user';await api.SaveProfile(p,'secret-sentinel');const data=await api.ExportData();assert.ok(!data.includes('private-user'));assert.ok(!data.includes('secret-sentinel'));
console.log('Preview scenarios, cancellation, import atomicity and secret isolation passed.');

sim.reset("empty");const created=await api.SaveProfile({id:"",name:"New route",server:"new.demo.invalid",port:443,protocol:"atrust",options:{socksPort:1080,httpPort:1081}},"discarded");assert.ok(created.id);assert.equal((await api.GetSnapshot()).profiles.length,1);

sim.reset('connected');let measured=await api.GetSnapshot();assert.ok(measured.traffic.downloadRate>0);assert.ok(measured.traffic.uploadRate>0);assert.ok(Number.isFinite(measured.traffic.sampledAt));await api.Disconnect();await wait();assert.equal((await api.GetSnapshot()).traffic,null);
