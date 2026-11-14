import {t} from '../core/i18n.js';
import {store} from '../core/store.js';
import {connectionStatus} from '../components/home/connection-status.js';
import {networkSpeed} from '../components/home/network-speed.js';
import {networkDiag} from '../components/home/network-diag.js';
import {connectionLogs} from '../components/home/connection-logs.js';

let renderedModules={};

/**
 * modules generates HTML for the available home dashboard components.
 */
function modules(){
	return {
		'home-status': connectionStatus(),
		'home-speed': networkSpeed(),
		'home-diag': networkDiag(),
		'home-logs': connectionLogs()
	};
}

/**
 * home renders the dashboard based on user selected widget preferences.
 */
export function home(){
	renderedModules = modules();
	const activeWidgets = store.snapshot.preferences.widgets || ['status', 'speed', 'diag', 'logs'];
	const widgetMap = {
		'status': 'home-status',
		'speed': 'home-speed',
		'diag': 'home-diag',
		'logs': 'home-logs'
	};

	const renderedList = activeWidgets
		.map(w => widgetMap[w])
		.filter(id => id && renderedModules[id])
		.map(id => `<div id="${id}">${renderedModules[id]}</div>`)
		.join('');

	return `<section class="home-panel">
		<div class="page-heading">
			<h1>${t('home')}</h1>
		</div>
		<div class="home-grid">
			${renderedList}
		</div>
		${store.snapshot.preview?`<p class="simulation-note">${t('previewHint')}</p>`:''}
	</section>`;
}

// Keep elapsed timer and focused state intact when only traffic/diagnostic samples update.
export function updateHome(){
	for(const [id,html] of Object.entries(modules())){
		const e=document.getElementById(id);
		if(e&&renderedModules[id]!==html){
			e.innerHTML=html;
			renderedModules[id]=html;
		}
	}
}

