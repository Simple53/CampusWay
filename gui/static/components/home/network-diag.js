import {t} from '../../core/i18n.js';
import {store} from '../../core/store.js';
import {button,esc} from '../../core/ui.js';

/**
 * networkDiag renders the connectivity and DNS/IP diagnostic module,
 * allowing one-click test of campus portals and internet targets similar to Clash Verge.
 */
export function networkDiag(){
	const r = store.diagReport;
	const testing = store.diagTesting;
	return `<section class="home-module diag-module" aria-labelledby="diag-title">
		<div class="module-heading">
			<h2 id="diag-title">${t('networkCheck')}</h2>
			${button(testing ? 'testing' : 'testNow', 'run-diag', 'quiet', testing ? 'disabled' : '')}
		</div>
		<div class="diag-summary">
			<div><span>${t('localIP')}</span><strong>${esc(r?.localIP || '—')}</strong></div>
			<div><span>${t('dnsServer')}</span><strong>${esc(r?.dns || '—')}</strong></div>
		</div>
		${r?.targets?.length ? `
		<div class="diag-target-list">
			${r.targets.map(target => `
				<div class="diag-target-row ${target.ok ? 'ok' : 'fail'}">
					<span class="target-name">${esc(target.name)}</span>
					<span class="target-status">${target.ok ? `${target.ms} ms` : (target.err ? esc(target.err.slice(0, 30)) : 'FAIL')}</span>
				</div>
			`).join('')}
		</div>
		` : `<p class="hint">${t('testNow')}</p>`}
	</section>`;
}
