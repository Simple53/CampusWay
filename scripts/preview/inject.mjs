import {createPreviewBridge} from './mock.mjs';
const simulator=createPreviewBridge();
const initialScene=new URLSearchParams(location.search).get('scene');
if(initialScene)simulator.reset(initialScene);
window.go={main:{AppServer:simulator.api}};
// Accept simulator controls from this origin's immediate parent only.
window.addEventListener('message',event=>{if(event.origin!==location.origin||event.source!==parent||event.data?.type!=='routedesk-preview')return;simulator.reset(event.data.scenario);});
parent.postMessage({type:'routedesk-preview-ready'},location.origin);
