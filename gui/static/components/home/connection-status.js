import {t,message} from '../../core/i18n.js';
import {store,selected,active,primary} from '../../core/store.js';
import {esc,icon,status,button,toggleSwitch} from '../../core/ui.js';

/**
 * connectionStatus renders the primary profile status and slide-toggle switch.
 */
export function connectionStatus(){
	const s = store.snapshot;
	const p = active() || selected();
	const isConnected = s.state === 'connected';
	const isBusy = s.state === 'connecting' || s.state === 'disconnecting' || s.state === 'awaiting_auth';

	return `<section class="home-module connection-module" aria-labelledby="connection-title">
		<div class="module-heading">
			<h2 id="connection-title">${t('currentStatus')}</h2>
			<div style="display:flex;align-items:center;gap:10px;">
				${status(s.state)}
				${p ? toggleSwitch(isConnected, 'toggle-connect', isBusy ? 'disabled' : '') : ''}
			</div>
		</div>
		${!p ? `
			<div class="welcome">
				${icon('route')}
				<h3>${t('emptyTitle')}</h3>
				<p>${t('emptyBody')}</p>
				${button('add','add','primary')}
			</div>
		` : `
			<h3>${esc(p.name)}</h3>
			<p class="gateway">${esc(p.server)}<span>${p.protocol==='atrust'?'aTrust':'EasyConnect'}</span></p>
			${s.state==='error'?`<p class="error" role="alert">${esc(message(s.message))}</p>`:''}
			<div class="main-actions">
				${button(primary(p),'connect','primary')}
				${button('manage','goto-connections','quiet')}
			</div>
			${s.state==='connected'?`
				<dl class="connection-details">
					<div>
						<dt>${t('localProxy')}</dt>
						<dd>SOCKS5 127.0.0.1:${p.options?.socksPort||1080}<br>HTTP 127.0.0.1:${p.options?.httpPort||1081}</dd>
					</div>
					<div>
						<dt>${t('duration')}</dt>
						<dd data-elapsed="${s.connectedAt}">—</dd>
					</div>
				</dl>
			`:''}
		`}
	</section>`;
}
