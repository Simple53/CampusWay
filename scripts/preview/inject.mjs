import {createPreviewBridge} from './mock.mjs';
const simulator=createPreviewBridge();
window.go={main:{AppServer:simulator.api}};
// Accept simulator controls from this origin's immediate parent only.
window.addEventListener('message',event=>{if(event.origin!==location.origin||event.source!==parent||event.data?.type!=='campusway-preview')return;simulator.reset(event.data.scenario);});
parent.postMessage({type:'campusway-preview-ready'},location.origin);
