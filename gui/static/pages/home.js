import {t} from '../core/i18n.js';
import {store} from '../core/store.js';
import {connectionStatus} from '../components/home/connection-status.js';
import {networkSpeed} from '../components/home/network-speed.js';
import {connectionLogs} from '../components/home/connection-logs.js';
let renderedModules={};
function modules(){return {'home-status':connectionStatus(),'home-speed':networkSpeed(),'home-logs':connectionLogs()};}
export function home(){renderedModules=modules();return `<section class="home-panel"><div class="page-heading"><h1>${t('home')}</h1></div><div class="home-grid">${Object.entries(renderedModules).map(([id,html])=>`<div id="${id}">${html}</div>`).join('')}</div>${store.snapshot.preview?`<p class="simulation-note">${t('previewHint')}</p>`:''}</section>`;}
// Keep the elapsed timer and focused action intact when only traffic samples change.
export function updateHome(){for(const [id,html] of Object.entries(modules())){const e=document.getElementById(id);if(e&&renderedModules[id]!==html){e.innerHTML=html;renderedModules[id]=html;}}}
