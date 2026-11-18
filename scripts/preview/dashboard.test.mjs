import assert from 'node:assert/strict';
import {store} from '../../gui/static/core/store.js';
import {routes,routeState,connections} from '../../gui/static/pages/connections.js';
import {networkSpeed,formatBytes} from '../../gui/static/components/home/network-speed.js';
store.snapshot={profiles:[{id:'a',name:'A',server:'a.demo.invalid',protocol:'atrust'},{id:'b',name:'B',server:'b.demo.invalid',protocol:'easyconnect'}],selected:'b',active:'a',state:'connected',preferences:{view:'grid'},logs:[]};
assert.equal(routeState(store.snapshot.profiles[0]),'connected');assert.equal(routeState(store.snapshot.profiles[1]),'disconnected');
assert.ok(!connections().includes('selection-bar'));assert.equal((routes().match(/data-action="edit-route"/g)||[]).length,2);
store.snapshot.active='';store.snapshot.state='error';store.failedID='a';assert.equal(routeState(store.snapshot.profiles[0]),'error');assert.equal(routeState(store.snapshot.profiles[1]),'disconnected');
assert.equal(formatBytes(1024),'1.0 KB');assert.equal(formatBytes(NaN),'—');
store.snapshot.state='connected';store.snapshot.traffic={sampledAt:Date.now(),downloadRate:2048,uploadRate:1024,uploaded:1024,downloaded:2048};assert.ok(networkSpeed().includes('2.0 KB/s'));
store.snapshot.traffic.sampledAt=Date.now()-6000;assert.ok(!networkSpeed().includes('2.0 KB/s'));
console.log('Route indicators, independent editing, removed footer and speed freshness passed.');

for (const view of ['grid','table']) {
 store.snapshot.preferences.view=view;
 store.snapshot.active='a';store.snapshot.state='awaiting_auth';
 const html=routes();
 assert.equal((html.match(/data-action="connect-route"/g)||[]).length,2);
 assert.equal((html.match(/<(?:article|tr) data-route-connect=/g)||[]).length,2);
 assert.ok(html.includes('aria-checked="true"'));
 store.snapshot.active='';store.snapshot.state='disconnected';
 assert.ok(!routes().includes('aria-checked="true"'));
}
const {home}=await import('../../gui/static/pages/home.js');
store.snapshot.preferences.widgets=['logs'];
store.snapshot.logs=[{time:new Date().toISOString(),level:'warn',message:'test'}];
assert.ok(!home().includes('widget-resize-handle'));
assert.ok(home().includes('WARN'));
assert.ok(home().includes('/icons.svg#copy'));
assert.ok(!home().includes('#settings/diagnostics'));
console.log('Per-route switches, verification cancellation state, SVG copy and log levels passed.');
